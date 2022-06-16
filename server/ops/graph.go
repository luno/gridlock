package ops

import (
	"fmt"
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

func stats(m api.Metrics) RateStats {
	return RateStats{
		Good:     m.CountGood,
		Warning:  m.CountWarning,
		Bad:      m.CountBad,
		Duration: m.Duration,
	}
}

type Graph struct {
	LatestBucket time.Time
	Regions      map[string]*Region
	Incoming     map[string]RateStats
}

func NewGraph() Graph {
	return Graph{
		Regions:  make(map[string]*Region),
		Incoming: make(map[string]RateStats),
	}
}

func (g Graph) EnsureRegion(region string) *Region {
	r, ok := g.Regions[region]
	if !ok {
		r = &Region{
			Nodes: make(map[RegionalNode]*Traffic),
			Cross: make(map[string]RateStats),
		}
		g.Regions[region] = r
	}
	return r
}

type Region struct {
	Nodes map[RegionalNode]*Traffic
	Cross map[string]RateStats
}

func (r *Region) EnsureNode(node RegionalNode) *Traffic {
	_, ok := r.Nodes[node]
	if !ok {
		r.Nodes[node] = &Traffic{
			Outgoing: make(map[RegionalNode]RateStats),
		}
	}
	return r.Nodes[node]
}

type RegionalNode struct {
	Name string
	Type api.NodeType
}

func (n RegionalNode) NodeName() string {
	return fmt.Sprintf("%s.%s", n.Name, n.Type)
}

type Traffic struct {
	Outgoing map[RegionalNode]RateStats
}

func (g *Graph) AddMetric(m api.Metrics, addStats bool) {
	srcRegion := g.EnsureRegion(m.SourceRegion)
	src := RegionalNode{Name: m.Source, Type: m.SourceType}
	srcTraffic := srcRegion.EnsureNode(src)

	tgtRegion := g.EnsureRegion(m.TargetRegion)
	tgt := RegionalNode{Name: m.Target, Type: m.TargetType}
	tgtRegion.EnsureNode(tgt)

	if !addStats {
		return
	}

	mStats := stats(m)

	if isFromInternet(m) {
		g.Incoming[m.TargetRegion] = g.Incoming[m.TargetRegion].Add(mStats)
	} else if m.SourceRegion != m.TargetRegion {
		srcRegion.Cross[m.TargetRegion] = srcRegion.Cross[m.TargetRegion].Add(mStats)
		// Don't add cross region traffic to the node
		return
	}
	srcTraffic.Outgoing[tgt] = srcTraffic.Outgoing[tgt].Add(mStats)
}
