package db

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"strconv"
	"strings"
	"time"
)

type Level string

const (
	Good    = "good"
	Warning = "warning"
	Bad     = "bad"
)

const (
	BucketDuration = time.Minute
	DefaultNodeTTL = time.Hour
)

type TrafficKey struct {
	FromID    string
	ToID      string
	Transport string
	Bucket    Bucket
	Level     Level
}

type Bucket struct {
	time.Time
}

func (b Bucket) Previous() Bucket {
	return Bucket{b.Add(-BucketDuration)}
}

func (b Bucket) Next() Bucket {
	return Bucket{b.Add(BucketDuration)}
}

func trafficKeyFromRedis(s string) (TrafficKey, error) {
	p := strings.Split(s, ".")
	if len(p) != 5 {
		return TrafficKey{}, errors.New("invalid key", j.KV("key", s))
	}
	from, to, trans, bucket, level := p[0], p[1], p[2], p[3], p[4]

	unix, err := strconv.ParseInt(bucket, 10, 64)
	if err != nil {
		return TrafficKey{}, errors.Wrap(err, "invalid timestamp", j.KV("value", bucket))
	}
	b := Bucket{time.Unix(unix, 0)}
	l := Level(level)
	switch level {
	case Good:
	case Warning:
	case Bad:
	default:
		return TrafficKey{}, errors.New("invalid level", j.KV("value", level))
	}
	return TrafficKey{
		FromID: from, ToID: to,
		Transport: trans,
		Bucket:    b, Level: l,
	}, nil
}

func trafficKeyToRedis(k TrafficKey) string {
	parts := []string{
		k.FromID,
		k.ToID,
		k.Transport,
		strconv.FormatInt(k.Bucket.Unix(), 10),
		string(k.Level),
	}
	return strings.Join(parts, ".")
}

func GetBucket(t time.Time) Bucket {
	return Bucket{t.Truncate(BucketDuration)}
}

func GetBucketsBetween(from, to time.Time) []Bucket {
	var ret []Bucket
	for b := GetBucket(from); b.Before(to); b = b.Next() {
		ret = append(ret, b)
	}
	return ret
}

func StoreNodeStat(ctx context.Context, conn redis.Conn,
	k TrafficKey, ttl time.Duration,
	count int64,
) error {
	key := trafficKeyToRedis(k)
	_, err := redis.DoContext(conn, ctx,
		"INCRBY", key, count,
	)
	if err != nil {
		return errors.Wrap(err, "")
	}
	expire := k.Bucket.Add(ttl)
	_, err = redis.DoContext(conn, ctx,
		"EXPIREAT", key, expire.Unix(),
	)
	return errors.Wrap(err, "")
}

type HandleNodeStatFunc func(context.Context, TrafficKey) error

func LoadSomeKeys(ctx context.Context, conn redis.Conn, cursor int64) ([]TrafficKey, int64, error) {
	keys, next, err := scanSomeKeys(ctx, conn, cursor)
	if err != nil {
		return nil, 0, err
	}
	ret := make([]TrafficKey, 0, len(keys))
	for _, k := range keys {
		key, err := trafficKeyFromRedis(k)
		if err != nil {
			log.Info(ctx, "failed to load key", j.KV("key", k), log.WithError(err))
			continue
		}
		ret = append(ret, key)
	}
	return ret, next, nil
}

func GetNodeStatCount(ctx context.Context, conn redis.Conn, key TrafficKey) (int64, error) {
	i, err := redis.Int64(redis.DoContext(conn, ctx,
		"GET", trafficKeyToRedis(key),
	))
	return i, errors.Wrap(err, "")
}
