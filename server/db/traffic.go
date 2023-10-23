package db

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
)

type Level string

const (
	Good    = "good"
	Warning = "warning"
	Bad     = "bad"
)

// TrafficKey is stored as a '.' separated key with an integer count value
type TrafficKey struct {
	FromID    string
	ToID      string
	Transport string
	Bucket    Bucket
	Level     Level
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

func StoreTrafficStat(ctx context.Context, conn redis.Conn,
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

func GetTrafficStat(ctx context.Context, conn redis.Conn, key TrafficKey) (int64, error) {
	i, err := redis.Int64(redis.DoContext(conn, ctx,
		"GET", trafficKeyToRedis(key),
	))
	return i, errors.Wrap(err, "")
}
