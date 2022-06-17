package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"time"
)

func storeMetrics(ctx context.Context, trafficDB TrafficDB, nodeDB NodeDB, metrics []api.Metrics) error {
	buckets := make(map[db.Bucket][]db.TrafficKey)
	for _, m := range metrics {
		keys, err := storeMetric(ctx, trafficDB, nodeDB, m)
		if err != nil {
			return err
		}
		for _, k := range keys {
			buckets[k.Bucket] = append(buckets[k.Bucket], k)
		}
	}
	for buck, keys := range buckets {
		err := trafficDB.StoreBucket(ctx, buck, keys)
		if err != nil {
			return err
		}
	}
	return nil
}

func storeMetric(ctx context.Context, trafficDB TrafficDB, nodeDB NodeDB, metric api.Metrics) ([]db.TrafficKey, error) {
	from := api.NodeInfo{
		Region: metric.SourceRegion,
		Type:   metric.SourceType,
		Name:   metric.Source,
	}
	if err := maybeStoreNode(ctx, nodeDB, from); err != nil {
		return nil, err
	}
	to := api.NodeInfo{
		Region: metric.TargetRegion,
		Type:   metric.TargetType,
		Name:   metric.Target,
	}
	if err := maybeStoreNode(ctx, nodeDB, to); err != nil {
		return nil, err
	}

	var ret []db.TrafficKey
	// TODO(adam): Split metric across buckets based on ts + duration
	b := db.BucketFromTime(time.Unix(metric.Timestamp, 0))
	k := db.TrafficKey{
		FromID: db.Key(from).ID(), ToID: db.Key(to).ID(),
		Transport: string(metric.Transport),
		Bucket:    b,
	}
	k.Level = db.Good
	err := trafficDB.StoreTrafficStat(ctx, k, metric.CountGood)
	if err != nil {
		return nil, err
	}
	ret = append(ret, k)

	k.Level = db.Warning
	err = trafficDB.StoreTrafficStat(ctx, k, metric.CountWarning)
	if err != nil {
		return nil, err
	}
	ret = append(ret, k)

	k.Level = db.Bad
	err = trafficDB.StoreTrafficStat(ctx, k, metric.CountBad)
	if err != nil {
		return nil, err
	}
	ret = append(ret, k)
	return ret, nil
}

func maybeStoreNode(ctx context.Context, nodeDB NodeDB, node api.NodeInfo) error {
	id := db.Key(node).ID()
	_, err := nodeDB.GetNode(ctx, id)
	if errors.Is(err, db.ErrNodeNotFound) {
		return nodeDB.RegisterNode(ctx, id, node)
	} else if err != nil {
		return err
	}
	return nil
}
