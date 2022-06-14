package gridlock

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"io"
	"net/http"
	"strings"
	"sync"
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

	defaultMethod Method

	flushChan   chan chan error
	flushPeriod time.Duration
	reqTimeout  time.Duration

	q      chan incCall
	nodeMu sync.RWMutex
	nodes  map[string]api.NodeInfo
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
		flushChan:   make(chan chan error, 1),
		flushPeriod: 20 * time.Second,
		reqTimeout:  30 * time.Second,
		q:           make(chan incCall, 1000),
		nodes:       make(map[string]api.NodeInfo),
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

func (c *Client) RegisterNode(info api.NodeInfo) {
	c.nodeMu.Lock()
	defer c.nodeMu.Unlock()
	c.nodes[info.Name] = info
}

func (c *Client) GetNodes() []api.NodeInfo {
	c.nodeMu.RLock()
	defer c.nodeMu.RUnlock()

	ret := make([]api.NodeInfo, 0, len(c.nodes))
	for _, v := range c.nodes {
		ret = append(ret, v)
	}
	return ret
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
			if err := c.send(ctx, &agg, c.GetNodes()); err != nil {
				return err
			}
		case ch := <-c.flushChan:
			ch <- c.send(ctx, &agg, c.GetNodes())
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

func (c *Client) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}
	s := strings.TrimSpace(string(b))
	return nil, errors.New("failed to submit", j.MKV{"response": s})
}

func (c *Client) send(ctx context.Context, a *aggregate, nodes []api.NodeInfo) error {
	if len(*a) == 0 && len(nodes) == 0 {
		return nil
	}
	defer a.Reset()

	start := time.Now()
	ts := start.Unix()

	sub := api.SubmitMetrics{NodeInfo: nodes}
	var total int64
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
		total += calls[0] + calls[1] + calls[2]
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}

	_, err = c.do(ctx, http.MethodPost, c.baseURL+"/api/submit", bytes.NewReader(b))
	if err != nil {
		c.metrics.SubmissionErrors.Inc()
		return err
	}
	c.metrics.SubmissionLatency.Observe(time.Since(start).Seconds())
	c.metrics.SubmittedCalls.Add(float64(total))
	return nil
}

func (c *Client) GetTraffic(ctx context.Context) ([]api.Traffic, error) {
	httpResp, err := c.do(ctx, http.MethodGet, c.baseURL+"/api/traffic", nil)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp api.GetTrafficResponse
	err = json.Unmarshal(b, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Traffic, nil
}
