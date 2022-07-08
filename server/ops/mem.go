package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"github.com/luno/jettison/errors"
	"sort"
	"sync"
)

type MemDB struct {
	mu      sync.RWMutex
	Nodes   map[db.TrafficKey]int64
	Buckets map[db.Bucket]map[db.TrafficKey]bool

	niMu     sync.RWMutex
	nodeInfo map[string]api.NodeInfo
	c        chan struct{}
}

func NewMemDB() *MemDB {
	return &MemDB{
		Nodes:    make(map[db.TrafficKey]int64),
		Buckets:  make(map[db.Bucket]map[db.TrafficKey]bool),
		c:        make(chan struct{}, 1),
		nodeInfo: make(map[string]api.NodeInfo),
	}
}

func (m *MemDB) GetTrafficStat(_ context.Context, key db.TrafficKey) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Nodes[key], nil
}

func (m *MemDB) StoreTrafficStat(_ context.Context, k db.TrafficKey, count int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Nodes[k] = count

	select {
	case m.c <- struct{}{}:
	default:
	}
	return nil
}

func (m *MemDB) GetBucket(_ context.Context, bucket db.Bucket) ([]db.TrafficKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b := m.Buckets[bucket]
	ret := make([]db.TrafficKey, 0, len(b))
	for k := range b {
		ret = append(ret, k)
	}
	return ret, nil
}

func (m *MemDB) StoreBucket(_ context.Context, bucket db.Bucket, keys []db.TrafficKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.Buckets[bucket]
	if !ok {
		b = make(map[db.TrafficKey]bool)
		m.Buckets[bucket] = b
	}
	for _, k := range keys {
		b[k] = true
	}
	return nil
}

func (m *MemDB) WaitForChanges() chan struct{} {
	return m.c
}

func (m *MemDB) RegisterNode(_ context.Context, key string, info api.NodeInfo) error {
	m.niMu.Lock()
	defer m.niMu.Unlock()
	m.nodeInfo[key] = info
	return nil
}

func (m *MemDB) GetNode(_ context.Context, key string) (api.NodeInfo, error) {
	m.niMu.RLock()
	defer m.niMu.RUnlock()
	ni, ok := m.nodeInfo[key]
	if !ok {
		return api.NodeInfo{}, errors.Wrap(db.ErrNodeNotFound, "")
	}
	return ni, nil
}

func (m *MemDB) GetNodes(context.Context) ([]api.NodeInfo, error) {
	m.niMu.RLock()
	defer m.niMu.RUnlock()

	ret := make([]api.NodeInfo, 0, len(m.nodeInfo))
	for _, v := range m.nodeInfo {
		ret = append(ret, v)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})
	return ret, nil
}

var _ TrafficDB = (*MemDB)(nil)
var _ NodeDB = (*MemDB)(nil)
