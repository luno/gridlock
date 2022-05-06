package ops

import (
	"context"
	"github.com/adamhicks/gridlock/server/db"
	"github.com/gomodule/redigo/redis"
	"time"
)

type NodeDB interface {
	WaitForChanges() chan struct{}

	GetNodeStatCount(ctx context.Context, key db.NodeStatKey) (int64, error)
	ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error
	StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error
}

type RedisDB struct {
	RedisConn redis.Conn
}

func (r RedisDB) WaitForChanges() chan struct{} {
	return make(chan struct{})
}

func (r RedisDB) GetNodeStatCount(ctx context.Context, key db.NodeStatKey) (int64, error) {
	return db.GetNodeStatCount(ctx, r.RedisConn, key)
}

func (r RedisDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	return db.ScanAllNodeStatKeys(ctx, r.RedisConn, f)
}

func (r RedisDB) StoreNodeStat(ctx context.Context, k db.NodeStatKey, ttl time.Duration, count int64) error {
	return db.StoreNodeStat(ctx, r.RedisConn, k, ttl, count)
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
