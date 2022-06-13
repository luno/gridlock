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

type Graph struct {
	LatestBucket time.Time
	Regions      map[string]*Region
}

func NewGraph() Graph {
	return Graph{Regions: make(map[string]*Region)}
}

func (g Graph) EnsureRegion(region string) *Region {
	r, ok := g.Regions[region]
	if !ok {
		r = &Region{Nodes: make(map[string]*Node)}
		g.Regions[region] = r
	}
	return r
}

type Region struct {
	Nodes map[string]*Node
}

func (r *Region) EnsureNode(name string) *Node {
	_, ok := r.Nodes[name]
	if !ok {
		r.Nodes[name] = &Node{}
	}
	return r.Nodes[name]
}

type Node struct {
	Type NodeType
}

func (g *Graph) AddMetric(m api.Metrics) {
	src := g.EnsureRegion(m.SourceRegion)
	s := src.EnsureNode(m.Source)
	tgt := g.EnsureRegion(m.TargetRegion)
	t := tgt.EnsureNode(m.Target)

	if m.Transport == api.TransportSQL {
		t.Type = NodeDatabase
	}
	if strings.ToLower(m.Source) == "internet" {
		s.Type = NodeUser
	}
}
