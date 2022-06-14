package ops

import (
	"github.com/luno/gridlock/api"
	"strings"
	"time"
)

type NodeType int

const (
	NodeUnknown = iota
	NodeDatabase
	NodeUser
)

func isFromInternet(m api.Metrics) bool {
	return strings.ToLower(m.Source) == "internet"
}

func stats(m api.Metrics) Stats {
	return Stats{
		Good:    m.CountGood,
		Warning: m.CountWarning,
		Bad:     m.CountBad,
	}
}

type Graph struct {
	LatestBucket time.Time
	Regions      map[string]*Region
	Incoming     map[string]Stats
}

func NewGraph() Graph {
	return Graph{
		Regions:  make(map[string]*Region),
		Incoming: make(map[string]Stats),
	}
}

func (g Graph) EnsureRegion(region string) *Region {
	r, ok := g.Regions[region]
	if !ok {
		r = &Region{
			Nodes: make(map[string]*Node),
			Cross: make(map[string]Stats),
		}
		g.Regions[region] = r
	}
	return r
}

type Region struct {
	Nodes map[string]*Node
	Cross map[string]Stats
}

func (r *Region) EnsureNode(name string) *Node {
	_, ok := r.Nodes[name]
	if !ok {
		r.Nodes[name] = &Node{
			Outgoing: make(map[string]Stats),
		}
	}
	return r.Nodes[name]
}

type Node struct {
	Type     NodeType
	Outgoing map[string]Stats
}

func (g *Graph) AddMetric(m api.Metrics, addStats bool) {
	src := g.EnsureRegion(m.SourceRegion)
	s := src.EnsureNode(m.Source)
	tgt := g.EnsureRegion(m.TargetRegion)
	t := tgt.EnsureNode(m.Target)

	ext := isFromInternet(m)

	if m.Transport == api.TransportSQL {
		t.Type = NodeDatabase
	} else if ext {
		s.Type = NodeUser
	}
	if !addStats {
		return
	}

	mStats := stats(m)

	if ext {
		g.Incoming[m.TargetRegion] = g.Incoming[m.TargetRegion].Add(mStats)
	} else if m.SourceRegion != m.TargetRegion {
		src.Cross[m.TargetRegion] = src.Cross[m.TargetRegion].Add(mStats)
		// Don't add cross region traffic to the node
		return
	}

	s.Outgoing[m.Target] = s.Outgoing[m.Target].Add(mStats)
}
