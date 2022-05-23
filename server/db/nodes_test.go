package db

import (
	"context"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/jtest"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestAgainstLocal(t *testing.T) {
	ctx := context.Background()
	conn, err := redis.DialURLContext(ctx, "redis://127.0.0.1:6379")
	jtest.RequireNil(t, err)

	now := time.Now()
	ttl := 5 * time.Minute

	for i := 1; i <= 10; i++ {
		src := "client-" + strconv.Itoa(i)
		tgt := "server-" + strconv.Itoa(i)

		key := NodeStatKey{
			SourceRegion: "us-west-1", Source: src,
			TargetRegion: "us-west-1", Target: tgt,
			Bucket: GetBucket(now), Level: Good,
		}

		err = StoreNodeStat(ctx, conn, key, ttl, rand.Int63n(100_000))
		jtest.RequireNil(t, err)

		key.Level = Warning
		err = StoreNodeStat(ctx, conn, key, ttl, rand.Int63n(1_000))
		jtest.RequireNil(t, err)

		key.Level = Bad
		err = StoreNodeStat(ctx, conn, key, ttl, rand.Int63n(100))
		jtest.RequireNil(t, err)
	}

	err = ScanAllNodeStatKeys(ctx, conn, func(ctx context.Context, key NodeStatKey) error {
		val, err := GetNodeStatCount(ctx, conn, key)
		if err != nil {
			return err
		}
		fmt.Println(key.Bucket, key.Source, "->", key.Target, key.Level, "=", val)
		return nil
	})
	jtest.RequireNil(t, err)
}
