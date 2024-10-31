package gridlock

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/luno/jettison/jtest"
	"github.com/stretchr/testify/assert"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/handlers"
	"github.com/luno/gridlock/server/ops"
)

type state struct {
	Log ops.TrafficStats
}

func (s state) TrafficStats() ops.TrafficStats {
	return s.Log
}

func TestClientSubmitsMetrics(t *testing.T) {
	t.Skip(`skip until we fix "Post /gridlock: stopped after 10 redirects`)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db := ops.NewMemDB()
	s := state{Log: ops.NewLoader(ctx, db, db)}

	srv := httptest.NewServer(handlers.CreateRouter(ctx, s))
	t.Cleanup(srv.Close)

	c := NewClient(
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	go func() {
		err := c.Deliver(ctx)
		jtest.Assert(t, context.Canceled, err)
	}()

	c.Record(Method{
		Source: "server1", SourceRegion: "region-a",
		Target: "server2", TargetRegion: "region-a",
	}, CallGood)

	c.Record(Method{
		Source: "server1", SourceRegion: "region-a",
		Target: "server2", TargetRegion: "region-a",
	}, CallGood)

	c.Record(Method{
		Source: "server1", SourceRegion: "region-a",
		Target: "server2", TargetRegion: "region-a",
	}, CallBad)

	<-c.Record(Method{
		Source: "server2", SourceRegion: "region-a",
		Target: "server1", TargetRegion: "region-a",
	}, CallWarning)

	jtest.RequireNil(t, c.Flush(ctx))

	traffic, err := c.GetTraffic(ctx)
	jtest.RequireNil(t, err)

	assert.Equal(t, []api.Traffic{
		{From: "server1", To: "server2", CountGood: 2, CountBad: 1},
		{From: "server2", To: "server1", CountWarning: 1},
	}, traffic)
}
