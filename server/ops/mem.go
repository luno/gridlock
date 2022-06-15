package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"sort"
	"sync"
	"time"
)

type MemDB struct {
	mu    sync.RWMutex
	Nodes map[db.TrafficKey]int64

	niMu     sync.RWMutex
	nodeInfo map[string]api.NodeInfo
	c        chan struct{}
}

func NewMemDB() *MemDB {
	return &MemDB{
		Nodes:    make(map[db.TrafficKey]int64),
		c:        make(chan struct{}, 1),
		nodeInfo: make(map[string]api.NodeInfo),
	}
}

func (m *MemDB) GetNodeStatCount(_ context.Context, key db.TrafficKey) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Nodes[key], nil
}

func (m *MemDB) ScanAllNodeStatKeys(ctx context.Context, f db.HandleNodeStatFunc) error {
	m.mu.RLock()
	nodesCopy := make(map[db.TrafficKey]int64)
	for key, val := range m.Nodes {
		nodesCopy[key] = val
	}
	m.mu.RUnlock()
	for key := range nodesCopy {
		if err := f(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemDB) StoreNodeStat(_ context.Context, k db.TrafficKey, _ time.Duration, count int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Nodes[k] = count

	select {
	case m.c <- struct{}{}:
	default:
	}
	return nil
}

func (m *MemDB) WaitForChanges() chan struct{} {
	return m.c
}

func (m *MemDB) RegisterNodes(_ context.Context, info ...api.NodeInfo) error {
	m.niMu.Lock()
	defer m.niMu.Unlock()
	for _, i := range info {
		m.nodeInfo[i.Name] = i
	}
	return nil
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
