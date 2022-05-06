package gridlock

import (
	"context"
	"github.com/adamhicks/gridlock/api"
	"github.com/adamhicks/gridlock/server/handlers"
	"github.com/adamhicks/gridlock/server/ops"
	"github.com/luno/jettison/jtest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

type state struct {
	Log ops.TrafficStats
}

func (s state) TrafficStats() ops.TrafficStats {
	return s.Log
}

func TestClientSubmitsMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	s := state{Log: ops.NewLoader(ctx, ops.NewMemDB())}

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
