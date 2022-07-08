package graph

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
	"time"
)

type Region struct {
	Group
}

func NewRegion(name string) Region {
	return Region{Group: NewGroup(name)}
}

func (r Region) Name() string {
	return r.name
}

func (r Region) Type() NodeType {
	return NodeRegion
}

func (r Region) getGroup(name string) Node {
	s := formatGroup(name)
	grp, ok := r.nodes[s]
	if !ok {
		grp = NewGroup(name)
		r.nodes[s] = grp
	}
	return grp
}

func (r Region) getNode(name string, typ api.NodeType, groups []config.Group) Node {
	for _, g := range groups {
		if g.MatchNode(name, typ) {
			return r.getGroup(g.Name)
		}
	}
	return r.getGroup(name)
}

func (r Region) EnsureNode(b Builder, region, name string, typ api.NodeType) {
	if region != r.name {
		return
	}
	n := r.getNode(name, typ, b.Config.Groups)
	n.EnsureNode(b, region, name, typ)
}

func (r Region) AddTraffic(b Builder,
	t time.Time, s RateStats,
	srcRegion, srcName string, srcType api.NodeType,
	tgtRegion, tgtName string, tgtType api.NodeType,
) {
	if srcType == api.NodeInternet || tgtType == api.NodeInternet {
		panic("wtf")
	}
	src := r.getNode(srcName, srcType, b.Config.Groups)
	tgt := r.getNode(tgtName, tgtType, b.Config.Groups)

	src.AddTraffic(b, t, s, srcRegion, srcName, srcType, tgtRegion, tgtName, tgtType)
	if src.Name() == tgt.Name() {
		return
	}
	tgt.AddTraffic(b, t, s, srcRegion, srcName, srcType, tgtRegion, tgtName, tgtType)

	r.traffic.Add(src.Name(), tgt.Name(), t, s)
}
