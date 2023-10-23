package ops

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
)

type NodeDB interface {
	RegisterNode(context.Context, string, api.NodeInfo) error
	GetNode(context.Context, string) (api.NodeInfo, error)
	GetNodes(context.Context) ([]api.NodeInfo, error)
}

type RedisNodeDB struct {
	pool *redis.Pool
}

func NewRedisNodeDB(p *redis.Pool) RedisNodeDB {
	return RedisNodeDB{pool: p}
}

func (r RedisNodeDB) getConnection(ctx context.Context) (redis.Conn, error) {
	c, err := r.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := db.SelectNodeDatabase(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r RedisNodeDB) closeConnection(ctx context.Context, conn redis.Conn) {
	err := conn.Close()
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "closing redis connection"))
	}
}

func (r RedisNodeDB) RegisterNode(ctx context.Context, key string, n api.NodeInfo) error {
	c, err := r.getConnection(ctx)
	if err != nil {
		return err
	}
	defer r.closeConnection(ctx, c)

	return db.StoreNode(ctx, c, key, n)
}

func (r RedisNodeDB) GetNode(ctx context.Context, key string) (api.NodeInfo, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return api.NodeInfo{}, err
	}
	defer r.closeConnection(ctx, c)

	return db.GetNode(ctx, c, key)
}

func (r RedisNodeDB) GetNodes(ctx context.Context) ([]api.NodeInfo, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer r.closeConnection(ctx, c)

	var ret []api.NodeInfo
	var cursor int64
	for {
		keys, next, err := db.GetSomeNodeKeys(ctx, c, cursor)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			ni, err := db.GetNode(ctx, c, k)
			if errors.Is(err, db.ErrNodeNotFound) {
				continue
			} else if err != nil {
				return nil, err
			}
			ret = append(ret, ni)
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return ret, nil
}
