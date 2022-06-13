package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/jettison/log"
	"sort"
	"sync"
	"time"
)

type TrafficStats interface {
	Record(ctx context.Context, m ...api.Metrics) error
	GetMetricLog() []api.Metrics
}

type Stats struct {
	Good    int64
	Warning int64
	Bad     int64
}

func (s Stats) Add(o Stats) Stats {
	s.Good += o.Good
	s.Warning += o.Warning
	s.Bad += o.Bad
	return s
}

type Loader struct {
	db  NodeDB
	now func() time.Time

	mMu     sync.RWMutex
	metrics []api.Metrics
}

func NewLoader(ctx context.Context, db NodeDB) *Loader {
	l := &Loader{db: db, now: time.Now}
	go l.WatchKeysForever(ctx)
	return l
}

func (l *Loader) Record(ctx context.Context, m ...api.Metrics) error {
	for _, metric := range m {
		b := db.GetBucket(time.Unix(metric.Timestamp, 0))
		k := db.NodeStatKey{
			Transport:    string(metric.Transport),
			SourceRegion: metric.SourceRegion,
			Source:       metric.Source,
			TargetRegion: metric.TargetRegion,
			Target:       metric.Target,
			Bucket:       b,
		}

		k.Level = db.Good
		if err := l.db.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountGood); err != nil {
			return err
		}
		k.Level = db.Warning
		if err := l.db.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountWarning); err != nil {
			return err
		}
		k.Level = db.Bad
		if err := l.db.StoreNodeStat(ctx, k, db.DefaultNodeTTL, metric.CountBad); err != nil {
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
		if err := l.refreshTraffic(ctx); err != nil {
			return err
		}
		select {
		case <-fullScan.C:
		case <-l.db.WaitForChanges():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *Loader) refreshTraffic(ctx context.Context) error {
	metrics, err := loadAllMetrics(ctx, l.db)
	if err != nil {
		return err
	}
	log.Info(ctx, "loaded metrics from database", j.KV("count", len(metrics)))

	lastFull := db.GetBucket(l.now()).Previous().Unix()

	var cut int
	for i, m := range metrics {
		if m.Timestamp > lastFull {
			cut = i
			break
		}
	}
	if cut > 0 {
		metrics = metrics[:cut]
	}

	l.mMu.Lock()
	defer l.mMu.Unlock()
	l.metrics = metrics
	return nil
}

func loadAllMetrics(ctx context.Context, ndb NodeDB) ([]api.Metrics, error) {
	stats := make(map[db.NodeStatKey]Stats)
	err := ndb.ScanAllNodeStatKeys(ctx, func(ctx context.Context, key db.NodeStatKey) error {
		val, err := ndb.GetNodeStatCount(ctx, key)
		if err != nil {
			return err
		}
		aggrKey := key
		// Zero the level so we can use it as a key to aggregate Stats
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
		stats[aggrKey] = s
		return nil
	})
	if err != nil {
		return nil, err
	}

	var metrics []api.Metrics
	for k, s := range stats {
		metrics = append(metrics, api.Metrics{
			Source:       k.Source,
			SourceRegion: k.SourceRegion,
			Transport:    api.Transport(k.Transport),
			Target:       k.Target,
			TargetRegion: k.TargetRegion,
			Timestamp:    k.Bucket.Time.Unix(),
			CountGood:    s.Good,
			CountWarning: s.Warning,
			CountBad:     s.Bad,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Timestamp != metrics[j].Timestamp {
			return metrics[i].Timestamp < metrics[j].Timestamp
		}
		if metrics[i].Source != metrics[j].Source {
			return metrics[i].Source < metrics[j].Source
		}
		return metrics[i].Target < metrics[j].Target
	})
	return metrics, nil
}
