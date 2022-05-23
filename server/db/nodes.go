package db

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"strconv"
	"strings"
	"time"
)

type NodeStatKey struct {
	SourceRegion string
	Source       string
	TargetRegion string
	Target       string
	Bucket       Bucket
	Level        Level
}

type Level string

const (
	Good    = "good"
	Warning = "warning"
	Bad     = "bad"
)

type Bucket struct {
	time.Time
}

func keyFromRedis(s string) (NodeStatKey, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 6 {
		return NodeStatKey{}, errors.New("invalid number of parts", j.KV("parts", len(parts)))
	}
	unix, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		return NodeStatKey{}, errors.Wrap(err, "invalid timestamp", j.KV("value", parts[5]))
	}
	b := Bucket{time.Unix(unix, 0)}
	level := Level(parts[4])
	switch level {
	case Good:
	case Warning:
	case Bad:
	default:
		return NodeStatKey{}, errors.New("invalid level", j.KV("value", level))
	}
	return NodeStatKey{
		SourceRegion: parts[0], Source: parts[1],
		TargetRegion: parts[2], Target: parts[3],
		Bucket: b, Level: level,
	}, nil
}

func keyToRedis(k NodeStatKey) string {
	return strings.Join([]string{
		k.SourceRegion, k.Source,
		k.TargetRegion, k.Target,
		string(k.Level),
		strconv.FormatInt(k.Bucket.Unix(), 10),
	}, ".")
}

const (
	BucketDuration = time.Minute
	DefaultNodeTTL = time.Hour
)

func GetBucket(t time.Time) Bucket {
	return Bucket{t.Truncate(BucketDuration)}
}

func NextBucket(b Bucket) Bucket {
	return Bucket{b.Add(BucketDuration)}
}

func GetBucketsBetween(from, to time.Time) []Bucket {
	var ret []Bucket
	for b := GetBucket(from); b.Before(to); b = NextBucket(b) {
		ret = append(ret, b)
	}
	return ret
}

func StoreNodeStat(ctx context.Context, conn redis.Conn,
	k NodeStatKey, ttl time.Duration,
	count int64,
) (err error) {
	defer func() {
		if err != nil {
			_, _ = redis.DoContext(conn, ctx, "DISCARD")
		}
	}()

	expire := k.Bucket.Add(ttl)
	key := keyToRedis(k)

	if err := conn.Send("MULTI"); err != nil {
		return err
	}
	if err := conn.Send("INCRBY", key, count); err != nil {
		return err
	}
	if err := conn.Send("EXPIREAT", key, expire.Unix()); err != nil {
		return err
	}
	_, err = redis.DoContext(conn, ctx, "EXEC")
	return err
}

type HandleNodeStatFunc func(context.Context, NodeStatKey) error

func ScanAllNodeStatKeys(ctx context.Context, conn redis.Conn, f HandleNodeStatFunc) error {
	var cursor int64
	for {
		next, keys, err := scanSomeKeys(ctx, conn, cursor)
		if err != nil {
			return err
		}
		for _, k := range keys {
			key, err := keyFromRedis(k)
			if err != nil {
				continue
			}
			err = f(ctx, key)
			if err != nil {
				return err
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return nil
}

func scanSomeKeys(ctx context.Context, conn redis.Conn, cursor int64) (int64, []string, error) {
	resp, err := redis.Values(redis.DoContext(conn, ctx, "SCAN", cursor))
	if err != nil {
		return 0, nil, err
	}
	next, err := redis.Int64(resp[0], nil)
	if err != nil {
		return 0, nil, err
	}
	keys, err := redis.Strings(resp[1], nil)
	return next, keys, err
}

func GetNodeStatCount(ctx context.Context, conn redis.Conn, key NodeStatKey) (int64, error) {
	return redis.Int64(redis.DoContext(conn, ctx, "GET", keyToRedis(key)))
}