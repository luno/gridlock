package ops

import (
	"context"
	"fmt"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/jtest"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestAgainstLocal(t *testing.T) {
	t.Skip("for testing against a running redis server")
	ctx := context.Background()

	r, err := NewRedis(ctx)
	jtest.RequireNil(t, err)

	now := time.Now()
	ttl := 5 * time.Minute

	for i := 1; i <= 10; i++ {
		src := "client-" + strconv.Itoa(i)
		tgt := "server-" + strconv.Itoa(i)

		key := db.TrafficKey{
			SourceRegion: "us-west-1", Source: src,
			TargetRegion: "us-west-1", Target: tgt,
			Bucket: db.GetBucket(now), Level: db.Good,
		}

		err = r.StoreNodeStat(ctx, key, ttl, rand.Int63n(100_000))
		jtest.RequireNil(t, err)

		key.Level = db.Warning
		err = r.StoreNodeStat(ctx, key, ttl, rand.Int63n(1_000))
		jtest.RequireNil(t, err)

		key.Level = db.Bad
		err = r.StoreNodeStat(ctx, key, ttl, rand.Int63n(100))
		jtest.RequireNil(t, err)
	}

	err = r.ScanAllNodeStatKeys(ctx, func(ctx context.Context, key db.TrafficKey) error {
		val, err := r.GetNodeStatCount(ctx, key)
		if err != nil {
			return err
		}
		fmt.Println(key.Bucket, key.Source, "->", key.Target, key.Level, "=", val)
		return nil
	})
	jtest.RequireNil(t, err)
}
