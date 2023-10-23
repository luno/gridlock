package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
)

const nodeTTL = time.Hour

var ErrNodeNotFound = errors.New("node not found", j.C("ERR_b747d53800a4219d"))

func StoreNode(ctx context.Context, conn redis.Conn, key string, info api.NodeInfo) error {
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	_, err = redis.DoContext(conn, ctx,
		"SET", key, b, "EX", int(nodeTTL.Seconds()),
	)
	return errors.Wrap(err, "store node")
}

func GetSomeNodeKeys(ctx context.Context, conn redis.Conn, cursor int64) ([]string, int64, error) {
	return scanSomeKeys(ctx, conn, cursor)
}

func GetNode(ctx context.Context, conn redis.Conn, key string) (api.NodeInfo, error) {
	v, err := redis.Bytes(redis.DoContext(conn, ctx,
		"GETEX", key, "EX", int(nodeTTL.Seconds()),
	))
	if errors.Is(err, redis.ErrNil) {
		return api.NodeInfo{}, errors.Wrap(ErrNodeNotFound, "")
	} else if err != nil {
		return api.NodeInfo{}, errors.Wrap(err, "")
	}
	var ni api.NodeInfo
	err = json.Unmarshal(v, &ni)
	return ni, err
}
