package db

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
)

const (
	trafficDatabase = 0
	nodeDatabase    = 1
)

func SelectTrafficDatabase(r redis.Conn) error {
	return selectDB(r, trafficDatabase)
}

func SelectNodeDatabase(r redis.Conn) error {
	return selectDB(r, nodeDatabase)
}

func selectDB(r redis.Conn, db int) error {
	_, err := r.Do("SELECT", db)
	return err
}

func scanSomeKeys(ctx context.Context, conn redis.Conn, cursor int64) ([]string, int64, error) {
	resp, err := redis.Values(redis.DoContext(conn, ctx, "SCAN", cursor))
	if err != nil {
		return nil, 0, errors.Wrap(err, "")
	}
	next, err := redis.Int64(resp[0], nil)
	if err != nil {
		return nil, 0, errors.Wrap(err, "")
	}
	keys, err := redis.Strings(resp[1], nil)
	return keys, next, errors.Wrap(err, "")
}

type NodeKey struct {
	Region string
	Name   string
	Type   api.NodeType
}

func Key(info api.NodeInfo) NodeKey {
	return NodeKey{
		Region: info.Region,
		Name:   info.Name,
		Type:   info.Type,
	}
}

func (k NodeKey) ID() string {
	h := sha1.New()
	_, _ = fmt.Fprintln(h, k.Region, k.Name, k.Type)
	return fmt.Sprintf("%x", h.Sum(nil))
}
