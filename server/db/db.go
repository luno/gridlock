package db

import (
	"context"
	"github.com/gomodule/redigo/redis"
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

func scanSomeKeys(ctx context.Context, conn redis.Conn, cursor int64) (int64, []string, error) {
	resp, err := redis.Values(redis.DoContext(conn, ctx, "SCAN", cursor))
	if err != nil {
		return 0, nil, errors.Wrap(err, "")
	}
	next, err := redis.Int64(resp[0], nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "")
	}
	keys, err := redis.Strings(resp[1], nil)
	return next, keys, errors.Wrap(err, "")
}
