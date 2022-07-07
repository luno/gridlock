package graph

import (
	"github.com/luno/gridlock/api"
	"time"
)

const InternetLabel = "internet"

type Internet struct {
	Leaf
}

func (i Internet) Name() string {
	return InternetLabel
}

func (i Internet) DisplayName() string {
	return InternetLabel
}

func (i Internet) Type() NodeType {
	return NodeUser
}

type Global struct {
	nodes   map[string]Node
	traffic NodeTraffic
}

func NewGlobal() Global {
	return Global{
		nodes:   make(map[string]Node),
		traffic: NewTraffic(),
	}
}

func (g Global) Name() string {
	return "edge"
}

func (g Global) DisplayName() string {
	return "edge"
}

func (g Global) Type() NodeType {
	return NodeGlobal
}

func (g Global) IsLeaf() bool {
	return false
}

func (g Global) GetNodes() map[string]Node {
	return g.nodes
}

func (g Global) getRegion(region string) Node {
	r, ok := g.nodes[region]
	if !ok {
		r = NewRegion(region)
		g.nodes[region] = r
	}
	return r
}

func (g Global) EnsureNode(b Builder, region string, name string, typ api.NodeType) {
	if typ == api.NodeInternet {
		if _, ok := g.nodes[InternetLabel]; !ok {
			g.nodes[InternetLabel] = Internet{}
		}
		return
	}
	n := g.getRegion(region)
	n.EnsureNode(b, region, name, typ)
}

func (g Global) AddTraffic(b Builder,
	t time.Time, s RateStats,
	srcRegion, srcName string, srcType api.NodeType,
	tgtRegion, tgtName string, tgtType api.NodeType,
) {
	if srcType == api.NodeInternet {
		g.traffic.Add(InternetLabel, tgtRegion, t, s)
		return
	} else if tgtType == api.NodeInternet {
		g.traffic.Add(srcRegion, InternetLabel, t, s)
		return
	} else if srcRegion != tgtRegion {
		g.traffic.Add(srcRegion, tgtRegion, t, s)
		return
	}
	r := g.getRegion(srcRegion)
	r.AddTraffic(b, t, s,
		srcRegion, srcName, srcType,
		tgtRegion, tgtName, tgtType,
	)
}

func (g Global) GetTraffic() []Arc {
	return g.traffic.Flatten()
}

var _ Node = (*Global)(nil)
