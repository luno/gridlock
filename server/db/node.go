package db

import (
	"context"
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"github.com/luno/gridlock/api"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"strings"
	"time"
)

const nodeTTL = time.Hour
const nodeKeySeparator = "."

var ErrNodeNotFound = errors.New("node not found", j.C("ERR_b747d53800a4219d"))

type NodeKey struct {
	Region, Name string
}

func (k NodeKey) toRedis() string {
	return strings.Join([]string{k.Region, k.Name}, nodeKeySeparator)
}

func nodeKeyFromRedis(s string) (NodeKey, error) {
	parts := strings.Split(s, nodeKeySeparator)
	if len(parts) < 2 {
		return NodeKey{}, errors.New("invalid key", j.KV("key", s))
	}
	return NodeKey{
		Region: parts[0],
		Name:   parts[1],
	}, nil
}

func nodeKey(n api.NodeInfo) NodeKey {
	return NodeKey{
		Region: n.Region,
		Name:   n.Name,
	}
}

func StoreNode(ctx context.Context, conn redis.Conn, node api.NodeInfo) error {
	b, err := json.Marshal(node)
	if err != nil {
		return err
	}
	_, err = redis.DoContext(conn, ctx, "SET", nodeKey(node).toRedis(), b, "EX", int(nodeTTL.Seconds()))
	return errors.Wrap(err, "store node")
}

func GetSomeNodeKeys(ctx context.Context, conn redis.Conn, cursor int64) ([]NodeKey, int64, error) {
	next, keys, err := scanSomeKeys(ctx, conn, cursor)
	if err != nil {
		return nil, 0, err
	}
	ret := make([]NodeKey, 0, len(keys))
	for _, k := range keys {
		nk, err := nodeKeyFromRedis(k)
		if err != nil {
			log.Error(ctx, errors.Wrap(err, "unknown key"))
			continue
		}
		ret = append(ret, nk)
	}
	return ret, next, nil
}

func GetNode(ctx context.Context, conn redis.Conn, key NodeKey) (api.NodeInfo, error) {
	v, err := redis.Bytes(redis.DoContext(conn, ctx, "GET", key.toRedis()))
	if err != nil {
		return api.NodeInfo{}, errors.Wrap(err, "")
	}
	var ni api.NodeInfo
	err = json.Unmarshal(v, &ni)
	return ni, err
}
