package db

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"time"
)

const (
	BucketDuration = time.Minute
	DefaultNodeTTL = time.Hour
)

type Bucket struct {
	time.Time
}

func (b Bucket) Previous() Bucket {
	return Bucket{b.Add(-BucketDuration)}
}

func (b Bucket) Next() Bucket {
	return Bucket{b.Add(BucketDuration)}
}

func BucketFromTime(t time.Time) Bucket {
	return Bucket{t.Truncate(BucketDuration)}
}

func GetBucketsBetween(from, to time.Time) []Bucket {
	var ret []Bucket
	for b := BucketFromTime(from); b.Before(to); b = b.Next() {
		ret = append(ret, b)
	}
	return ret
}

func StoreBucket(ctx context.Context, conn redis.Conn,
	bucket Bucket, keys []TrafficKey, ttl time.Duration,
) error {
	args := make([]interface{}, 0, len(keys)+1)
	args = append(args, bucket.Unix())
	for _, k := range keys {
		args = append(args, trafficKeyToRedis(k))
	}
	_, err := redis.DoContext(conn, ctx,
		"SADD", args...,
	)
	if err != nil {
		return errors.Wrap(err, "")
	}
	expire := bucket.Add(ttl)
	_, err = redis.DoContext(conn, ctx,
		"EXPIREAT", bucket.Unix(), expire.Unix(),
	)
	return errors.Wrap(err, "")
}

func GetBucket(ctx context.Context, conn redis.Conn, bucket Bucket) ([]TrafficKey, error) {
	keys, err := redis.Strings(redis.DoContext(conn, ctx, "SMEMBERS", bucket.Unix()))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	ret := make([]TrafficKey, 0, len(keys))
	for _, k := range keys {
		tk, err := trafficKeyFromRedis(k)
		if err != nil {
			log.Error(ctx, err)
			continue
		}
		ret = append(ret, tk)
	}
	return ret, nil
}
