package gridlock

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CallAggregate [3]int64

type Counter interface {
	Inc()
	Add(v float64)
}

type Measure interface {
	Observe(secs float64)
}

type noopMetric struct{}

func (noopMetric) Inc()            {}
func (noopMetric) Add(float64)     {}
func (noopMetric) Observe(float64) {}

type Client struct {
	baseURL string
	cli     *http.Client
	metrics Metrics
	now     func() time.Time

	defaultMethod Method

	flushChan   chan chan error
	flushPeriod time.Duration
	reqTimeout  time.Duration

	q chan incCall
}

var errRetryable = errors.New("", j.C("ERR_43d3926acd268ae8"))

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(client *Client) {
		client.baseURL = url
	}
}

func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.cli = c
	}
}

func WithDefaultMethod(m Method) ClientOption {
	return func(client *Client) {
		client.defaultMethod = m
	}
}

func WithFlushPeriod(t time.Duration) ClientOption {
	return func(client *Client) {
		client.flushPeriod = t
	}
}

type Metrics struct {
	SuccessfulCalls   Counter
	DroppedCalls      Counter
	SubmittedCalls    Counter
	SubmissionLatency Measure
	SubmissionErrors  Counter
}

func (m *Metrics) defaultUnused() {
	if m.SuccessfulCalls == nil {
		m.SuccessfulCalls = noopMetric{}
	}
	if m.DroppedCalls == nil {
		m.DroppedCalls = noopMetric{}
	}
	if m.SubmittedCalls == nil {
		m.SubmittedCalls = noopMetric{}
	}
	if m.SubmissionLatency == nil {
		m.SubmissionLatency = noopMetric{}
	}
	if m.SubmissionErrors == nil {
		m.SubmissionErrors = noopMetric{}
	}
}

func WithMetrics(m Metrics) ClientOption {
	return func(client *Client) {
		client.metrics = m
	}
}

func NewClient(opts ...ClientOption) *Client {
	ret := &Client{
		cli:         http.DefaultClient,
		now:         time.Now,
		flushChan:   make(chan chan error, 1),
		flushPeriod: 20 * time.Second,
		reqTimeout:  30 * time.Second,
		q:           make(chan incCall, 1000),
	}
	for _, opt := range opts {
		opt(ret)
	}
	ret.metrics.defaultUnused()
	if ret.cli == nil {
		panic("no http client specified")
	}
	return ret
}

type incCall struct {
	M       Method
	Success CallSuccess
	Done    chan struct{}
}

type aggregate struct {
	Calls   map[Method]CallAggregate
	Started time.Time
	Ended   time.Time
}

func newAggregate(ts time.Time) aggregate {
	return aggregate{
		Calls:   make(map[Method]CallAggregate),
		Started: ts,
	}
}

func (a aggregate) Record(m Method, s CallSuccess) {
	l := a.Calls[m]
	switch s {
	case CallGood:
		l[0]++
	case CallWarning:
		l[1]++
	case CallBad:
		l[2]++
	}
	a.Calls[m] = l
}

func (a *aggregate) Close(ts time.Time) {
	a.Ended = ts
}

func (c *Client) Record(m Method, s CallSuccess) chan struct{} {
	done := make(chan struct{})
	select {
	case c.q <- incCall{M: m.Merge(c.defaultMethod), Success: s, Done: done}:
		c.metrics.SuccessfulCalls.Inc()
	default:
		c.metrics.DroppedCalls.Inc()
		close(done)
	}
	return done
}

func (c *Client) Deliver(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	t := time.NewTicker(c.flushPeriod)
	defer t.Stop()

	flush := func(ctx context.Context, agg aggregate, ch chan<- error) aggregate {
		ts := c.now()
		agg.Close(ts)
		go func() {
			ch <- c.sendBatch(ctx, agg)
		}()
		return newAggregate(ts)
	}
	agg := newAggregate(c.now())

	sendErrors := make(chan error)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case call := <-c.q:
			agg.Record(call.M, call.Success)
			close(call.Done)
		case <-t.C:
			agg = flush(ctx, agg, sendErrors)
		case err := <-sendErrors:
			if err != nil {
				return err
			}
		case ch := <-c.flushChan:
			agg = flush(ctx, agg, ch)
		}
	}
}

func (c *Client) Flush(ctx context.Context) error {
	rep := make(chan error, 1)
	select {
	case c.flushChan <- rep:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-rep:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func wrapHTTPError(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*url.Error); ok {
		if e.Timeout() || e.Temporary() {
			return errors.Wrap(errRetryable, err.Error())
		}
	}
	return err
}

func (c *Client) doRetry(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	retries := 4
	wait := time.Second
	for {
		resp, err := c.do(ctx, method, path, body)
		if err == nil {
			return resp, nil
		}
		if !errors.IsAny(err, context.DeadlineExceeded, errRetryable) || retries <= 0 {
			return nil, err
		}
		select {
		case <-time.After(wait):
			wait *= 2
			retries--
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		log.Info(ctx, "retrying request", j.MKV{"path": path})
	}
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, wrapHTTPError(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}
	if resp.StatusCode == http.StatusOK {
		return b, nil
	}
	s := strings.TrimSpace(string(b))
	return nil, errors.New("failed to submit", j.MKV{"response": s})
}

func (c *Client) sendBatch(ctx context.Context, a aggregate) error {
	if len(a.Calls) == 0 {
		return nil
	}

	t0 := time.Now()
	dur := a.Ended.Sub(a.Started)

	var sub api.SubmitMetrics
	var total int64
	for method, calls := range a.Calls {
		sub.Metrics = append(sub.Metrics, api.Metrics{
			Source:       method.Source,
			SourceRegion: method.SourceRegion,
			SourceType:   method.SourceType,
			Target:       method.Target,
			TargetRegion: method.TargetRegion,
			TargetType:   method.TargetType,
			Transport:    method.Transport,
			Timestamp:    a.Started.Unix(),
			Duration:     dur,
			CountGood:    calls[0],
			CountWarning: calls[1],
			CountBad:     calls[2],
		})
		total += calls[0] + calls[1] + calls[2]
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}

	_, err = c.doRetry(ctx, http.MethodPost, "/api/submit", b)
	if err != nil {
		c.metrics.SubmissionErrors.Inc()
		return err
	}
	c.metrics.SubmissionLatency.Observe(time.Since(t0).Seconds())
	c.metrics.SubmittedCalls.Add(float64(total))
	return nil
}

func (c *Client) GetTraffic(ctx context.Context) ([]api.Traffic, error) {
	r, err := c.do(ctx, http.MethodGet, "/api/traffic", nil)
	if err != nil {
		return nil, err
	}

	var resp api.GetTrafficResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Traffic, nil
}
