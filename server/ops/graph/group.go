package graph

import (
	"fmt"
	"time"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
)

func formatNode(name string, typ api.NodeType) string {
	return fmt.Sprintf("%s.%s", name, typ)
}

func formatGroup(name string) string {
	return fmt.Sprintf("%s.group", name)
}

type Grouper struct {
	Name    string   `json:"name"`
	Matches []string `json:"matches"`
}

func (g Grouper) Match(s string) bool {
	for _, m := range g.Matches {
		if m == s {
			return true
		}
	}
	return false
}

type Group struct {
	config config.Group

	nodes   map[string]Node
	traffic NodeTraffic
}

func NewGroup(c config.Group) Group {
	return Group{
		config:  c,
		nodes:   make(map[string]Node),
		traffic: NewTraffic(),
	}
}

func (g Group) Name() string {
	return formatGroup(g.config.Name)
}

func (g Group) DisplayName() string {
	return g.config.Name
}

func (g Group) Type() NodeType {
	return NodeGroup
}

func (g Group) IsLeaf() bool {
	return false
}

func (g Group) GetNodes() map[string]Node {
	return g.nodes
}

func (g Group) getNode(name string, typ api.NodeType) Node {
	if typ == api.NodeInternet {
		return getInternetNode(g.nodes)
	}
	s := formatNode(name, typ)
	n, ok := g.nodes[s]
	if !ok {
		match := g.config.MatchNode(name, typ)
		n = NewLeaf(name, typ, !match)
		g.nodes[s] = n
	}
	return n
}

func (g Group) EnsureNode(b Builder, region, name string, typ api.NodeType) {
	g.getNode(name, typ).EnsureNode(b, region, name, typ)
}

func (g Group) AddTraffic(_ Builder,
	t time.Time, s RateStats,
	_, srcName string, srcType api.NodeType,
	_, tgtName string, tgtType api.NodeType,
) {
	// Get the nodes here, any new nodes created here are from outside this group
	src := g.getNode(srcName, srcType)
	tgt := g.getNode(tgtName, tgtType)
	g.traffic.Add(src.Name(), tgt.Name(), t, s)
}

func (g Group) GetTraffic() []Arc {
	return g.traffic.Flatten()
}

func (g Group) IsAuxiliary() bool {
	return false
}

var _ Node = (*Group)(nil)
