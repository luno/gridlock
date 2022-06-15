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
	RegisterNodes(context.Context, ...api.NodeInfo) error
	GetNodes(context.Context) ([]api.NodeInfo, error)
}

type RedisNodeDB struct {
	pool *redis.Pool
}

func NewRedisNodeRegistry(p *redis.Pool) RedisNodeDB {
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

func (r RedisNodeDB) RegisterNodes(ctx context.Context, nodes ...api.NodeInfo) error {
	c, err := r.getConnection(ctx)
	if err != nil {
		return err
	}
	defer r.closeConnection(ctx, c)

	for _, n := range nodes {
		err := db.StoreNode(ctx, c, n)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r RedisNodeDB) GetNodes(ctx context.Context) ([]api.NodeInfo, error) {
	c, err := r.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer r.closeConnection(ctx, c)

	allKeys := make(map[db.NodeKey]bool)
	var cursor int64
	for {
		nkl, next, err := db.GetSomeNodeKeys(ctx, c, cursor)
		if err != nil {
			return nil, err
		}
		for _, nk := range nkl {
			allKeys[nk] = true
		}
		if next == 0 {
			break
		}
		cursor = next
	}

	ret := make([]api.NodeInfo, 0, len(allKeys))
	for k := range allKeys {
		ni, err := db.GetNode(ctx, c, k)
		// The node could've expired
		if errors.Is(err, db.ErrNodeNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}
		ret = append(ret, ni)
	}
	return ret, nil
}
