package graph

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
	"time"
)

type NodeType int

const (
	NodeUnknown = iota
	NodeDatabase
	NodeUser
	NodeMicroService
	NodeGroup
	NodeRegion
	NodeGlobal
)

type Arc struct {
	From    string
	To      string
	Traffic TrafficLogs
}

type Node interface {
	Name() string
	DisplayName() string
	Type() NodeType
	// IsAuxiliary indicates that this node has been included
	// because of traffic data rather than by definition
	// It used in groups to indicate nodes which are part of the group.
	IsAuxiliary() bool

	IsLeaf() bool
	GetNodes() map[string]Node
	EnsureNode(b Builder, region string, name string, typ api.NodeType)
	AddTraffic(b Builder,
		t time.Time, s RateStats,
		srcRegion, srcName string, srcType api.NodeType,
		tgtRegion, tgtName string, tgtType api.NodeType,
	)
	GetTraffic() []Arc
}

type Builder struct {
	Config config.Config
}

type TimeInclusionFunc func(ts time.Time, dur time.Duration) float64

type Range struct {
	From, To time.Time
}

func (r Range) Include(start time.Time, dur time.Duration) float64 {
	end := start.Add(dur)
	if start.After(r.To) || end.Before(r.From) {
		return 0
	}
	return 1
}

func stats(m api.Metrics) RateStats {
	return RateStats{
		Good:     m.CountGood,
		Warning:  m.CountWarning,
		Bad:      m.CountBad,
		Duration: m.Duration,
	}
}

func ConstructGraph(b Builder, ml []api.Metrics) Node {
	root := NewGlobal()
	for _, m := range ml {
		root.EnsureNode(b, m.SourceRegion, m.Source, m.SourceType)
		root.EnsureNode(b, m.TargetRegion, m.Target, m.TargetType)

		root.AddTraffic(b,
			time.Unix(m.Timestamp, 0).UTC(), stats(m),
			m.SourceRegion, m.Source, m.SourceType,
			m.TargetRegion, m.Target, m.TargetType,
		)
	}
	return root
}
