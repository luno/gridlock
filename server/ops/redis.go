package ops

import (
	"context"
	"flag"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"time"
)

var redisAddr = flag.String("redis", "redis://127.0.0.1:6379", "Address to connect to the redis server")
var redisUser = flag.String("redis_user", "", "User for authentication to the redis server, requires password")
var redisPassword = flag.String("redis_password", "", "Password for authentication to the redis server")

func NewRedisPool(ctx context.Context) (*redis.Pool, error) {
	if *redisAddr == "" {
		return nil, errors.New("redis not configured")
	}

	log.Info(ctx, "redis database configured", j.KV("address", *redisAddr))

	do := []redis.DialOption{
		redis.DialReadTimeout(5 * time.Second),
		redis.DialWriteTimeout(5 * time.Second),
	}
	if *redisUser != "" || *redisPassword != "" {
		if *redisUser == "" || *redisPassword == "" {
			return nil, errors.New("redis username/password misconfiguration")
		}
		do = append(do,
			redis.DialUsername(*redisUser),
			redis.DialPassword(*redisPassword),
		)
	}

	return &redis.Pool{
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
	}, nil
}
