package ops

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
)

type TrafficDB interface {
	WaitForChanges() chan struct{}

	GetTrafficStat(ctx context.Context, key db.TrafficKey) (int64, error)
	StoreTrafficStat(ctx context.Context, k db.TrafficKey, count int64) error

	GetBucket(ctx context.Context, bucket db.Bucket) ([]db.TrafficKey, error)
	StoreBucket(ctx context.Context, bucket db.Bucket, keys []db.TrafficKey) error
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

func (r RedisTrafficDB) GetTrafficStat(ctx context.Context, key db.TrafficKey) (int64, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return 0, err
	}
	defer r.closeConnection(ctx, c)
	return db.GetTrafficStat(ctx, c, key)
}

func (r RedisTrafficDB) StoreTrafficStat(ctx context.Context, k db.TrafficKey, count int64) error {
	c, err := r.getConnection(ctx)
	if err != nil {
		return err
	}
	defer r.closeConnection(ctx, c)
	return db.StoreTrafficStat(ctx, c, k, db.DefaultNodeTTL, count)
}

func (r RedisTrafficDB) GetBucket(ctx context.Context, bucket db.Bucket) ([]db.TrafficKey, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer r.closeConnection(ctx, c)
	return db.GetBucket(ctx, c, bucket)
}

func (r RedisTrafficDB) StoreBucket(ctx context.Context, bucket db.Bucket, keys []db.TrafficKey) error {
	c, err := r.getConnection(ctx)
	if err != nil {
		return err
	}
	defer r.closeConnection(ctx, c)
	return db.StoreBucket(ctx, c, bucket, keys, db.DefaultNodeTTL)
}

var _ TrafficDB = (*RedisTrafficDB)(nil)
