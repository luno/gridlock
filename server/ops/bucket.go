package ops

import (
	"context"

	"github.com/luno/gridlock/server/db"
	"github.com/luno/gridlock/server/ops/graph"
)

type BucketTraffic map[db.TrafficKey]graph.RateStats

func loadBucket(ctx context.Context, trafficDB TrafficDB, bucket db.Bucket) (BucketTraffic, error) {
	keys, err := trafficDB.GetBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}
	agg := make(map[db.TrafficKey]graph.RateStats)
	for _, k := range keys {
		count, err := trafficDB.GetTrafficStat(ctx, k)
		if err != nil {
			return nil, err
		}
		l := k.Level
		// Zero the key to aggregate stats
		k.Level = ""
		s := agg[k]
		switch l {
		case db.Good:
			s.Good += count
		case db.Warning:
			s.Warning += count
		case db.Bad:
			s.Bad += count
		}
		s.Duration = db.BucketDuration
		agg[k] = s
	}
	return agg, nil
}
