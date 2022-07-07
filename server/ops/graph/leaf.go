package graph

import (
	"github.com/luno/gridlock/api"
	"time"
)

type Leaf struct {
	name string
	typ  api.NodeType
}

func NewLeaf(name string, typ api.NodeType) Leaf {
	return Leaf{
		name: name,
		typ:  typ,
	}
}

func (l Leaf) Name() string {
	return formatNode(l.name, l.typ)
}

func (l Leaf) DisplayName() string {
	return l.name
}

func (l Leaf) Type() NodeType {
	switch l.typ {
	case api.NodeDatabase:
		return NodeDatabase
	case api.NodeInternet:
		return NodeUser
	default:
		return NodeMicroService
	}
}

func (l Leaf) IsLeaf() bool {
	return true
}

func (l Leaf) GetNodes() map[string]Node {
	return nil
}

func (l Leaf) EnsureNode(_ Builder, _, name string, typ api.NodeType) {
	if l.name != name || l.typ != typ {
		panic("hello?")
	}
}

func (l Leaf) AddTraffic(Builder,
	time.Time, RateStats,
	string, string, api.NodeType,
	string, string, api.NodeType,
) {
}

func (l Leaf) GetTraffic() []Arc {
	return nil
}

var _ Node = (*Leaf)(nil)
