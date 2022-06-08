package ops

import (
	"context"
	"flag"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"sync"
	"time"
)

type NodeDB interface {
	WaitForChanges() chan struct{}

	GetNodeStatCount(ctx context.Context, key db.NodeStatKey) (int64, error)
	ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error
	StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error
}

var redisAddr = flag.String("redis", "redis://127.0.0.1:6379", "Address to connect to the redis server")
var redisUser = flag.String("redis_user", "", "User for authentication to the redis server, requires password")
var redisPassword = flag.String("redis_password", "", "Password for authentication to the redis server")

type RedisDB struct {
	Pool *redis.Pool
}

func NewRedis(ctx context.Context) (RedisDB, error) {
	if *redisAddr == "" {
		return RedisDB{}, errors.New("redis not configured")
	}

	log.Info(ctx, "redis database configured", j.KV("address", *redisAddr))

	var do []redis.DialOption
	if *redisUser != "" || *redisPassword != "" {
		if *redisUser == "" || *redisPassword == "" {
			return RedisDB{}, errors.New("redis username/password misconfiguration")
		}
		do = []redis.DialOption{
			redis.DialUsername(*redisUser),
			redis.DialPassword(*redisPassword),
		}
	}

	return RedisDB{Pool: &redis.Pool{
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialURLContext(ctx, *redisAddr, do...)
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
	c, err := r.Pool.GetContext(ctx)
	if err != nil {
		return 0, err
	}
	defer func(c redis.Conn) {
		_ = c.Close()
	}(c)
	return db.GetNodeStatCount(ctx, c, key)
}

func (r RedisDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	c, err := r.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer func(c redis.Conn) {
		_ = c.Close()
	}(c)
	return db.ScanAllNodeStatKeys(ctx, c, f)
}

func (r RedisDB) StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error {
	c, err := r.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer func(c redis.Conn) {
		_ = c.Close()
	}(c)
	return db.StoreNodeStat(ctx, c, k, ttl, count)
}

type MemDB struct {
	mu    sync.RWMutex
	Nodes map[db.NodeStatKey]int64
	c     chan struct{}
}

func NewMemDB() *MemDB {
	return &MemDB{
		Nodes: make(map[db.NodeStatKey]int64),
		c:     make(chan struct{}, 1),
	}
}

func (m *MemDB) GetNodeStatCount(_ context.Context, key db.NodeStatKey) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Nodes[key], nil
}

func (m *MemDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	m.mu.RLock()
	nodesCopy := make(map[db.NodeStatKey]int64)
	for key, val := range m.Nodes {
		nodesCopy[key] = val
	}
	m.mu.RUnlock()
	for key := range nodesCopy {
		if err := f(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemDB) StoreNodeStat(_ context.Context, k db.NodeStatKey, _ time.Duration, count int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Nodes[k] = count

	select {
	case m.c <- struct{}{}:
	default:
	}
	return nil
}

func (m *MemDB) WaitForChanges() chan struct{} {
	return m.c
}

var _ NodeDB = (*RedisDB)(nil)
var _ NodeDB = (*MemDB)(nil)
