package ops

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"time"
)

type TrafficDB interface {
	WaitForChanges() chan struct{}

	GetNodeStatCount(ctx context.Context, key db.TrafficKey) (int64, error)
	ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error
	StoreNodeStat(ctx context.Context, k db.TrafficKey, ttl time.Duration, count int64) error
}

type RedisTrafficDB struct {
	Pool *redis.Pool
}

func NewRedisTrafficDB(p *redis.Pool) RedisTrafficDB {
	return RedisTrafficDB{Pool: p}
}

func (r RedisTrafficDB) getConnection(ctx context.Context) (redis.Conn, error) {
	c, err := r.Pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := db.SelectTrafficDatabase(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r RedisTrafficDB) closeConnection(ctx context.Context, conn redis.Conn) {
	err := conn.Close()
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "closing redis connection"))
	}
}

func (r RedisTrafficDB) WaitForChanges() chan struct{} {
	return make(chan struct{})
}

func (r RedisTrafficDB) GetNodeStatCount(ctx context.Context, key db.TrafficKey) (int64, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return 0, err
	}
	defer r.closeConnection(ctx, c)
	return db.GetNodeStatCount(ctx, c, key)
}

func (r RedisTrafficDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	cur := redisCursor{rn: r}
	for cur.More() {
		keys, err := cur.LoadSomeKeys(ctx)
		if err != nil {
			return err
		}
		for _, key := range keys {
			err := f(ctx, key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r RedisTrafficDB) StoreNodeStat(ctx context.Context, k db.TrafficKey, ttl time.Duration, count int64) error {
	c, err := r.getConnection(ctx)
	if err != nil {
		return err
	}
	defer r.closeConnection(ctx, c)
	return db.StoreNodeStat(ctx, c, k, ttl, count)
}

type redisCursor struct {
	rn      RedisTrafficDB
	Cursor  int64
	Started bool
}

func (c *redisCursor) LoadSomeKeys(ctx context.Context) ([]db.TrafficKey, error) {
	conn, err := c.rn.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer c.rn.closeConnection(ctx, conn)

	keys, next, err := db.LoadSomeKeys(ctx, conn, c.Cursor)
	if err != nil {
		return nil, err
	}
	c.Cursor = next
	c.Started = true
	return keys, nil
}

func (c redisCursor) More() bool {
	return !c.Started || c.Cursor != 0
}

var _ TrafficDB = (*RedisTrafficDB)(nil)
