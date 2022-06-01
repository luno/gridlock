package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/log"
	"sync"
	"time"
)

type Loader struct {
	db  NodeDB
	now func() time.Time

	tMu     sync.RWMutex
	traffic Traffic
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

func (l *Loader) GetTraffic() Traffic {
	l.tMu.RLock()
	defer l.tMu.RUnlock()
	return l.traffic
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
	earliest := l.now().Add(-6 * time.Minute)
	nodes, err := loadNodes(ctx, l.db, earliest)
	if err != nil {
		return err
	}

	l.tMu.Lock()
	defer l.tMu.Unlock()
	l.traffic = compileTraffic(nodes)
	return nil
}

func loadNodes(ctx context.Context, ndb NodeDB, earliest time.Time) (map[db.NodeStatKey]int64, error) {
	nodes := make(map[db.NodeStatKey]int64)
	err := ndb.ScanAllNodeStatKeys(ctx, func(ctx context.Context, key db.NodeStatKey) error {
		if key.Bucket.Before(earliest) {
			return nil
		}
		val, err := ndb.GetNodeStatCount(ctx, key)
		if err != nil {
			return err
		}
		nodes[key] = val
		return nil
	})
	return nodes, err
}
