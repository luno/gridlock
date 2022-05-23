package ops

import (
	"context"
	"flag"
	"github.com/adamhicks/gridlock/server/db"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"time"
)

type NodeDB interface {
	WaitForChanges() chan struct{}

	GetNodeStatCount(ctx context.Context, key db.NodeStatKey) (int64, error)
	ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error
	StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error
}

var redisAddr = flag.String("redis", "redis://127.0.0.1:6379", "Address to connect to the redis server")

type RedisDB struct {
	Pool *redis.Pool
}

func NewRedis() (RedisDB, error) {
	if *redisAddr == "" {
		return RedisDB{}, errors.New("redis not configured")
	}
	return RedisDB{Pool: &redis.Pool{
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialURLContext(ctx, *redisAddr)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     3,
		MaxActive:   10,
		IdleTimeout: time.Minute,
		Wait:        true,
	}}, nil
}

func (r RedisDB) WaitForChanges() chan struct{} {
	return make(chan struct{})
}

func (r RedisDB) GetNodeStatCount(ctx context.Context, key db.NodeStatKey) (int64, error) {
	c := r.Pool.Get()
	defer c.Close()
	return db.GetNodeStatCount(ctx, c, key)
}

func (r RedisDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	c := r.Pool.Get()
	defer c.Close()
	return db.ScanAllNodeStatKeys(ctx, c, f)
}

func (r RedisDB) StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error {
	c := r.Pool.Get()
	defer c.Close()
	return db.StoreNodeStat(ctx, c, k, ttl, count)
}

type MemDB struct {
	Nodes map[db.NodeStatKey]int64
	c     chan struct{}
}

func NewMemDB() MemDB {
	return MemDB{
		Nodes: make(map[db.NodeStatKey]int64),
		c:     make(chan struct{}),
	}
}

func (m MemDB) GetNodeStatCount(_ context.Context, key db.NodeStatKey) (int64, error) {
	return m.Nodes[key], nil
}

func (m MemDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	for key := range m.Nodes {
		if err := f(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (m MemDB) StoreNodeStat(_ context.Context, k db.NodeStatKey, _ time.Duration, count int64) error {
	m.Nodes[k] = count
	m.c <- struct{}{}
	return nil
}

func (m MemDB) WaitForChanges() chan struct{} {
	return m.c
}

var _ NodeDB = (*RedisDB)(nil)
var _ NodeDB = (*MemDB)(nil)
