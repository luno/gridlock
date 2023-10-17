package graph

import (
	"sort"
	"testing"
	"time"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildGraph(t *testing.T) {
	ml := []api.Metrics{
		{
			SourceRegion: "eu-west-1", Source: "internet", SourceType: api.NodeInternet,
			Transport:    api.TransportHTTP,
			TargetRegion: "eu-west-1", Target: "exchange-api", TargetType: api.NodeService,
			Timestamp: 100,
			Duration:  time.Minute,
			CountGood: 1, CountWarning: 2, CountBad: 0,
		},
		{
			SourceRegion: "eu-west-1", Source: "console", SourceType: api.NodeService,
			Transport:    api.TransportGRPC,
			TargetRegion: "eu-west-1", Target: "exchange-api", TargetType: api.NodeService,
			Timestamp: 100,
			Duration:  time.Minute,
			CountGood: 100, CountWarning: 10, CountBad: 1,
		},
		{
			SourceRegion: "eu-west-1", Source: "exchange-api", SourceType: api.NodeService,
			Transport:    api.TransportSQL,
			TargetRegion: "eu-west-1", Target: "exchange", TargetType: api.NodeDatabase,
			Timestamp: 100,
			Duration:  time.Minute,
			CountGood: 12, CountWarning: 0, CountBad: 0,
		},
	}
	b := Builder{
		Config: config.Config{
			Groups: []config.Group{
				{
					Name: "exchange",
					Selectors: []config.Selector{
						{Name: "exchange-api"},
						{Name: "exchange"},
					},
				},
			},
		},
	}
	root := ConstructGraph(b, ml)

	expGraph := map[string][]string{
		"edge":                 {"eu-west-1", "internet"},
		"eu-west-1":            {"console.service", "exchange.group"},
		"exchange.group":       {"exchange-api.service", "exchange.database"},
		"console.service":      nil,
		"exchange-api.service": nil,
		"exchange.database":    nil,
		"internet":             nil,
	}

	assert.Equal(t, expGraph, flatten(root))

	ts := time.Unix(100, 0).UTC()
	expTraffic := map[string][]Arc{
		"edge": {{
			From: "internet", To: "eu-west-1",
			Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
				ts: {
					Good: 1, Warning: 2, Bad: 0,
					Duration: time.Minute,
				},
			}},
		}},
		"eu-west-1": {{
			From: "console.service", To: "exchange.group",
			Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
				ts: {
					Good: 100, Warning: 10, Bad: 1,
					Duration: time.Minute,
				},
			}},
		}},
		"console.service":      nil,
		"exchange-api.service": nil,
		"exchange.database":    nil,
		"exchange.group": {
			{
				From: "console.service", To: "exchange-api.service",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 100, Warning: 10, Bad: 1,
						Duration: time.Minute,
					},
				}},
			},
			{
				From: "exchange-api.service", To: "exchange.database",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 12, Warning: 0, Bad: 0,
						Duration: time.Minute,
					},
				}},
			},
		},
		"internet": nil,
	}
	assert.Equal(t, expTraffic, flatTraffic(root))
}

func flatten(n Node) map[string][]string {
	nodes := []Node{n}
	ret := make(map[string][]string)
	for len(nodes) > 0 {
		nxt := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]

		var names []string
		for name, node := range nxt.GetNodes() {
			names = append(names, name)
			nodes = append(nodes, node)
		}
		sort.Strings(names)
		ret[nxt.Name()] = names
	}
	return ret
}

func flatTraffic(n Node) map[string][]Arc {
	ret := make(map[string][]Arc)
	nodes := []Node{n}
	for len(nodes) > 0 {
		nxt := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]

		ret[nxt.Name()] = nxt.GetTraffic()

		for _, node := range nxt.GetNodes() {
			nodes = append(nodes, node)
		}
	}
	return ret
}
