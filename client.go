package gridlock

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/adamhicks/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"io"
	"net/http"
	"time"
)

type CallAggregate [3]int64

type Counter interface {
	Inc()
}

type Measure interface {
	Observe(secs float64)
}

type noopMetric struct{}

func (noopMetric) Inc()            {}
func (noopMetric) Observe(float64) {}

type Client struct {
	baseURL string
	cli     *http.Client
	metrics Metrics

	defaultMethod Method

	flushChan   chan chan error
	flushPeriod time.Duration

	q chan incCall
}

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

type Metrics struct {
	SuccessfulCalls   Counter
	DroppedCalls      Counter
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
		flushChan:   make(chan chan error, 1),
		flushPeriod: 20 * time.Second,
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

type aggregate map[Method]CallAggregate

func (a *aggregate) Reset() {
	*a = make(aggregate)
}

func (a aggregate) Record(m Method, s CallSuccess) {
	l := a[m]
	switch s {
	case CallGood:
		l[0]++
	case CallWarning:
		l[1]++
	case CallBad:
		l[2]++
	}
	a[m] = l
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
	t := time.NewTicker(c.flushPeriod)
	defer t.Stop()

	agg := make(aggregate)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case call := <-c.q:
			agg.Record(call.M, call.Success)
			close(call.Done)
		case <-t.C:
			if err := c.send(ctx, &agg); err != nil {
				return err
			}
		case ch := <-c.flushChan:
			ch <- c.send(ctx, &agg)
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

func (c *Client) send(ctx context.Context, a *aggregate) error {
	if len(*a) == 0 {
		return nil
	}
	defer a.Reset()

	start := time.Now()
	ts := start.Unix()

	var sub api.SubmitMetrics

	for method, calls := range *a {
		sub.Metrics = append(sub.Metrics, api.Metrics{
			Source:       method.Source,
			SourceRegion: method.SourceRegion,
			Target:       method.Target,
			TargetRegion: method.TargetRegion,
			Transport:    method.Transport,
			Timestamp:    ts,
			CountGood:    calls[0],
			CountWarning: calls[1],
			CountBad:     calls[2],
		})
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/submit", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.cli.Do(req)
	if err != nil {
		c.metrics.SubmissionErrors.Inc()
		return err
	} else if resp.StatusCode != http.StatusOK {
		c.metrics.SubmissionErrors.Inc()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response")
		}
		return errors.New("failed to submit", j.MKV{"resp": string(b)})
	}
	c.metrics.SubmissionLatency.Observe(time.Since(start).Seconds())
	return nil
}

func (c *Client) GetTraffic(ctx context.Context) ([]api.Traffic, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/traffic", nil)
	if err != nil {
		return nil, err
	}
	httpResp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New(string(b))
	}

	var resp api.GetTraffic
	err = json.Unmarshal(b, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Traffic, nil
}
