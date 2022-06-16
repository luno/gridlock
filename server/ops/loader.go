package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"sync"
	"time"
)

type TrafficStats interface {
	Record(ctx context.Context, m ...api.Metrics) error
	GetMetricLog() []api.Metrics
	GetNodes() []api.NodeInfo
}

type RateStats struct {
	Good     int64
	Warning  int64
	Bad      int64
	Duration time.Duration
}

func (s RateStats) Add(o RateStats) RateStats {
	s.Good += o.Good
	s.Warning += o.Warning
	s.Bad += o.Bad
	s.Duration += o.Duration
	return s
}

func (s RateStats) GoodRate() float64 {
	return float64(s.Good) / s.Duration.Seconds()
}

func (s RateStats) WarningRate() float64 {
	return float64(s.Warning) / s.Duration.Seconds()
}

func (s RateStats) BadRate() float64 {
	return float64(s.Bad) / s.Duration.Seconds()
}

type Loader struct {
	trafficDB TrafficDB
	nodeDB    NodeDB

	now func() time.Time

	mMu     sync.RWMutex
	metrics []api.Metrics
	nodes   []api.NodeInfo
}

func (l *Loader) GetNodes() []api.NodeInfo {
	l.mMu.RLock()
	defer l.mMu.RUnlock()

	cNodes := make([]api.NodeInfo, len(l.nodes))
	copy(cNodes, l.nodes)
	return cNodes
}

func NewLoader(ctx context.Context, trafficDB TrafficDB, nodeDB NodeDB) *Loader {
	l := &Loader{trafficDB: trafficDB, nodeDB: nodeDB, now: time.Now}
	go l.WatchKeysForever(ctx)
	return l
}

func (l *Loader) maybeStoreNode(ctx context.Context, node api.NodeInfo) error {
	id := db.Key(node).ID()
	_, err := l.nodeDB.GetNode(ctx, id)
	if errors.Is(err, db.ErrNodeNotFound) {
		return l.nodeDB.RegisterNode(ctx, id, node)
	} else if err != nil {
		return err
	}
	return nil
}

func (l *Loader) Record(ctx context.Context, m ...api.Metrics) error {
	for _, metric := range m {
		b := db.GetBucket(time.Unix(metric.Timestamp, 0))

		from := api.NodeInfo{
			Region: metric.SourceRegion,
			Type:   metric.SourceType,
			Name:   metric.Source,
		}
		if err := l.maybeStoreNode(ctx, from); err != nil {
			return err
		}
		to := api.NodeInfo{
			Region: metric.TargetRegion,
			Type:   metric.TargetType,
			Name:   metric.Target,
		}
		if err := l.maybeStoreNode(ctx, to); err != nil {
			return err
		}

		k := db.TrafficKey{
			FromID: db.Key(from).ID(), ToID: db.Key(to).ID(),
			Transport: string(metric.Transport),
			Bucket:    b,
		}

		k.Level = db.Good
		if err := l.trafficDB.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountGood); err != nil {
			return err
		}
		k.Level = db.Warning
		if err := l.trafficDB.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountWarning); err != nil {
			return err
		}
		k.Level = db.Bad
		if err := l.trafficDB.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountBad); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loader) GetMetricLog() []api.Metrics {
	l.mMu.RLock()
	defer l.mMu.RUnlock()

	ret := make([]api.Metrics, len(l.metrics))
	copy(ret, l.metrics)

	return ret
}

func (l *Loader) WatchKeysForever(ctx context.Context) {
	for {
		err := l.WatchKeys(ctx)
		if errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			log.Error(ctx, err)
			time.Sleep(time.Minute)
		}
	}
}

func (l *Loader) WatchKeys(ctx context.Context) error {
	fullScan := time.NewTicker(time.Minute)
	defer fullScan.Stop()

	for {
		traffic, err := loadTraffic(ctx, l.trafficDB, l.now())
		if err != nil {
			return err
		}
		mLog, nodes, err := l.compileState(ctx, traffic)
		if err != nil {
			return err
		}
		l.setState(mLog, nodes)

		select {
		case <-fullScan.C:
		case <-l.trafficDB.WaitForChanges():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func loadTraffic(ctx context.Context, trafficDB TrafficDB, now time.Time) (map[db.TrafficKey]RateStats, error) {
	metrics, err := loadAllMetrics(ctx, trafficDB)
	if err != nil {
		return nil, err
	}
	lastFull := db.GetBucket(now).Previous()
	for k := range metrics {
		if k.Bucket.After(lastFull.Time) {
			delete(metrics, k)
		}
	}
	return metrics, nil
}

func (l *Loader) loadNode(ctx context.Context, key string, cache map[string]api.NodeInfo) (api.NodeInfo, error) {
	if ni, in := cache[key]; in {
		return ni, nil
	}
	ni, err := l.nodeDB.GetNode(ctx, key)
	if err != nil {
		return api.NodeInfo{}, err
	}
	cache[key] = ni
	return ni, nil
}

func (l *Loader) compileState(ctx context.Context, traffic map[db.TrafficKey]RateStats) ([]api.Metrics, []api.NodeInfo, error) {
	cache := make(map[string]api.NodeInfo)
	mLog := make([]api.Metrics, 0, len(traffic))

	for k, stats := range traffic {
		from, err := l.loadNode(ctx, k.FromID, cache)
		if errors.Is(err, db.ErrNodeNotFound) {
			log.Info(ctx, "skipped node", j.KV("node", k.FromID))
			continue
		} else if err != nil {
			return nil, nil, err
		}
		to, err := l.loadNode(ctx, k.ToID, cache)
		if errors.Is(err, db.ErrNodeNotFound) {
			log.Info(ctx, "skipped node", j.KV("node", k.ToID))
			continue
		} else if err != nil {
			return nil, nil, err
		}
		mLog = append(mLog, api.Metrics{
			Source:       from.Name,
			SourceRegion: from.Region,
			SourceType:   from.Type,
			Transport:    api.Transport(k.Transport),
			Target:       to.Name,
			TargetRegion: to.Region,
			TargetType:   to.Type,
			Timestamp:    k.Bucket.Unix(),
			Duration:     stats.Duration,
			CountGood:    stats.Good,
			CountWarning: stats.Warning,
			CountBad:     stats.Bad,
		})
	}

	nodes := make([]api.NodeInfo, 0, len(cache))
	for _, k := range cache {
		nodes = append(nodes, k)
	}
	return mLog, nodes, nil
}

func (l *Loader) setState(log []api.Metrics, nodes []api.NodeInfo) {
	l.mMu.Lock()
	defer l.mMu.Unlock()
	l.metrics = log
	l.nodes = nodes
}

func loadAllMetrics(ctx context.Context, tdb TrafficDB) (map[db.TrafficKey]RateStats, error) {
	t0 := time.Now()
	stats := make(map[db.TrafficKey]RateStats)
	err := tdb.ScanAllNodeStatKeys(ctx, func(ctx context.Context, key db.TrafficKey) error {
		val, err := tdb.GetNodeStatCount(ctx, key)
		if err != nil {
			return err
		}
		aggrKey := key
		// Zero the level, so we can use it as a key to aggregate RateStats across levels
		aggrKey.Level = ""
		s := stats[aggrKey]
		switch key.Level {
		case db.Good:
			s.Good += val
		case db.Warning:
			s.Warning += val
		case db.Bad:
			s.Bad += val
		}
		s.Duration = db.BucketDuration
		stats[aggrKey] = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Info(ctx, "loaded traffic from database", j.MKV{
		"count":      len(stats),
		"time_taken": time.Since(t0),
	})
	return stats, err
}
