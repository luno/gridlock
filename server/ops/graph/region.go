package graph

import (
	"time"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
)

type Region struct {
	Group
}

func NewRegion(name string) Region {
	return Region{Group: NewGroup(config.Group{Name: name})}
}

func (r Region) Name() string {
	return r.Group.config.Name
}

func (r Region) Type() NodeType {
	return NodeRegion
}

func (r Region) getGroup(group config.Group) Node {
	s := formatGroup(group.Name)
	grp, ok := r.nodes[s]
	if !ok {
		grp = NewGroup(group)
		r.nodes[s] = grp
	}
	return grp
}

func (r Region) getNode(name string, typ api.NodeType, groups []config.Group) Node {
	if typ == api.NodeInternet {
		return getInternetNode(r.nodes)
	}
	for _, g := range groups {
		if g.MatchNode(name, typ) {
			return r.getGroup(g)
		}
	}
	return r.getGroup(config.NodeMatcher(name, typ))
}

func (r Region) EnsureNode(b Builder, region, name string, typ api.NodeType) {
	if region != r.Name() {
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
	src := r.getNode(srcName, srcType, b.Config.Groups)
	tgt := r.getNode(tgtName, tgtType, b.Config.Groups)

	src.AddTraffic(b, t, s, srcRegion, srcName, srcType, tgtRegion, tgtName, tgtType)
	if src.Name() == tgt.Name() {
		return
	}
	tgt.AddTraffic(b, t, s, srcRegion, srcName, srcType, tgtRegion, tgtName, tgtType)

	r.traffic.Add(src.Name(), tgt.Name(), t, s)
}
