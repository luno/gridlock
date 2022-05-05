package ops

import (
	"context"
	"github.com/adamhicks/gridlock/api"
	"sort"
	"sync"
	"time"
)

type TrafficStats interface {
	Record(ctx context.Context, m ...api.Metrics) error
	GetTraffic() Traffic
}

type Flow struct {
	From string
	To   string
}

type Stats struct {
	Good    int64
	Warning int64
	Bad     int64
}

type Traffic struct {
	From, To time.Time
	Regions  map[string]*Region
}

type Region struct {
	Nodes map[string]*Node
}

type Node struct {
	Name     string
	Outgoing map[string]Stats
}

func (t Traffic) IsZero() bool {
	return t.From.IsZero() && t.To.IsZero() && len(t.Regions) == 0
}

type awaitDone struct {
	Metrics []api.Metrics
	done    chan struct{}
}

type TrafficLog struct {
	mu      sync.RWMutex
	current Traffic
	now     func() time.Time

	incMetrics chan awaitDone
	log        []api.Metrics

	refreshPeriod time.Duration
	windowPeriod  time.Duration
}

func NewTrafficLog() *TrafficLog {
	return &TrafficLog{
		now:           time.Now,
		incMetrics:    make(chan awaitDone, 100),
		refreshPeriod: 5 * time.Second,
		windowPeriod:  5 * time.Minute,
	}
}

func (l *TrafficLog) Record(ctx context.Context, metrics ...api.Metrics) error {
	ch := make(chan struct{})
	select {
	case l.incMetrics <- awaitDone{Metrics: metrics, done: ch}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *TrafficLog) setTraffic(t Traffic) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.current = t
}

func (l *TrafficLog) GetTraffic() Traffic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.current
}

func (l *TrafficLog) ProcessMetrics(ctx context.Context) error {
	refresh := time.NewTicker(l.refreshPeriod)
	defer refresh.Stop()

	for {
		select {
		case aw := <-l.incMetrics:
			l.log = append(l.log, aw.Metrics...)
			l.refreshTraffic(ctx)
			close(aw.done)
		case <-refresh.C:
			l.refreshTraffic(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func trimLog(l []api.Metrics, earliest time.Time) []api.Metrics {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Timestamp < l[j].Timestamp
	})
	fromTimestamp := earliest.Unix()
	var idx int
	for ; idx < len(l); idx++ {
		if l[idx].Timestamp >= fromTimestamp {
			break
		}
	}
	return l[idx:]
}

func (l *TrafficLog) refreshTraffic(_ context.Context) {
	to := l.now()
	from := to.Add(-l.windowPeriod)
	l.log = trimLog(l.log, from)
	l.setTraffic(compileTraffic(from, to, l.log))
}

func (r Region) ensureNode(name string) *Node {
	n, ok := r.Nodes[name]
	if ok {
		return n
	}
	n = &Node{
		Name:     name,
		Outgoing: make(map[string]Stats),
	}
	r.Nodes[name] = n
	return n
}

func compileTraffic(from, to time.Time, log []api.Metrics) Traffic {
	t := Traffic{
		From:    from,
		To:      to,
		Regions: make(map[string]*Region),
	}
	for _, m := range log {
		// TODO(adam): Handle cross region traffic at a regional node level
		if m.SourceRegion != m.TargetRegion {
			continue
		}

		r, ok := t.Regions[m.SourceRegion]
		if !ok {
			r = &Region{Nodes: make(map[string]*Node)}
			t.Regions[m.SourceRegion] = r
		}
		src := r.ensureNode(m.Source)
		r.ensureNode(m.Target)

		stats := src.Outgoing[m.Target]
		stats.Good += m.CountGood
		stats.Warning += m.CountWarning
		stats.Bad += m.CountBad
		src.Outgoing[m.Target] = stats
	}
	return t
}
