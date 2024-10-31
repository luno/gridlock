package ops

import (
	"context"
	"testing"
	"time"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/jtest"
	"github.com/stretchr/testify/assert"
)

func TestLoaderMemoryCleanup(t *testing.T) {
	ctx := context.Background()
	mdb := NewMemDB()
	buckets := make(map[db.Bucket]BucketTraffic)

	ts := time.Now()

	b := db.BucketFromTime(ts)

	from := api.NodeInfo{
		Region: "region1",
		Name:   "app1",
		Type:   api.NodeService,
	}
	to := api.NodeInfo{
		Region: "region1",
		Name:   "app2",
		Type:   api.NodeService,
	}

	err := mdb.StoreBucket(ctx, b, []db.TrafficKey{
		{
			FromID:    db.Key(from).ID(),
			ToID:      db.Key(to).ID(),
			Transport: string(api.TransportGRPC),
			Bucket:    b,
			Level:     db.Good,
		},
	})
	jtest.RequireNil(t, err)

	err = loadTraffic(ctx, mdb, buckets, ts)
	jtest.RequireNil(t, err)
	assert.Len(t, buckets, 61)

	ts = ts.Add(4 * time.Hour)
	err = loadTraffic(ctx, mdb, buckets, ts)
	jtest.RequireNil(t, err)
	assert.Len(t, buckets, 61)
}
