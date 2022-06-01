package ops

import (
	"context"
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
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

func NewTraffic() Traffic {
	return Traffic{Regions: make(map[string]*Region)}
}

func (t Traffic) EnsureRegion(region string) *Region {
	r, ok := t.Regions[region]
	if !ok {
		r = &Region{Nodes: make(map[string]*Node)}
		t.Regions[region] = r
	}
	return r
}

type Region struct {
	Nodes map[string]*Node
}

func (r *Region) EnsureNode(name string) *Node {
	n, ok := r.Nodes[name]
	if !ok {
		n = &Node{
			Name:     name,
			Outgoing: make(map[string]Stats),
		}
		r.Nodes[name] = n
	}
	return n
}

type Node struct {
	Name     string
	Outgoing map[string]Stats
}

func compileTraffic(nodes map[db.NodeStatKey]int64) Traffic {
	t := NewTraffic()
	var min, max time.Time

	for n, count := range nodes {
		if t.From.IsZero() || n.Bucket.Before(min) {
			t.From = n.Bucket.Time
		}
		if t.To.IsZero() || n.Bucket.After(max) {
			t.To = n.Bucket.Time
		}

		// TODO(adam): Handle cross region traffic at a regional node level
		if n.SourceRegion != n.TargetRegion {
			continue
		}
		r := t.EnsureRegion(n.SourceRegion)
		r.EnsureNode(n.Target)
		src := r.EnsureNode(n.Source)
		stats := src.Outgoing[n.Target]
		switch n.Level {
		case db.Good:
			stats.Good += count
		case db.Warning:
			stats.Warning += count
		case db.Bad:
			stats.Bad += count
		}
		src.Outgoing[n.Target] = stats
	}
	return t
}
