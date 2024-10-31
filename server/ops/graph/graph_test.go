package graph

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/ops/config"
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
		"eu-west-1":            {"console.group", "exchange.group", "internet"},
		"exchange.group":       {"console.service", "exchange-api.service", "exchange.database", "internet"},
		"console.group":        {"console.service", "exchange-api.service"},
		"console.service":      nil,
		"exchange-api.service": nil,
		"exchange.database":    nil,
		"internet":             nil,
	}

	require.Equal(t, expGraph, flatten(root))

	ts := time.Unix(100, 0).UTC()

	expTraffic := map[string][]Arc{
		"console.group":        {Arc{From: "console.service", To: "exchange-api.service", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 100, Warning: 10, Bad: 1, Duration: 60000000000}}}}},
		"console.service":      []Arc(nil),
		"edge":                 {Arc{From: "internet", To: "eu-west-1", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 1, Warning: 2, Bad: 0, Duration: 60000000000}}}}},
		"eu-west-1":            {Arc{From: "internet", To: "exchange.group", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 1, Warning: 2, Bad: 0, Duration: 60000000000}}}}, Arc{From: "console.group", To: "exchange.group", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 100, Warning: 10, Bad: 1, Duration: 60000000000}}}}},
		"exchange-api.service": []Arc(nil),
		"exchange.database":    []Arc(nil),
		"exchange.group":       {{From: "console.service", To: "exchange-api.service", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 100, Warning: 10, Bad: 1, Duration: 60000000000}}}}, {From: "exchange-api.service", To: "exchange.database", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 12, Warning: 0, Bad: 0, Duration: 60000000000}}}}, {From: "internet", To: "exchange-api.service", Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{time.Date(1970, time.January, 1, 0, 1, 40, 0, time.UTC): {Good: 1, Warning: 2, Bad: 0, Duration: 60000000000}}}}},
		"internet":             []Arc(nil),
	}

	expTraffic = map[string][]Arc{
		"edge": {{
			From: "internet", To: "eu-west-1",
			Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
				ts: {
					Good: 1, Warning: 2, Bad: 0,
					Duration: time.Minute,
				},
			}},
		}},
		"eu-west-1": {
			{
				From: "console.group", To: "exchange.group",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 100, Warning: 10, Bad: 1,
						Duration: time.Minute,
					},
				}},
			},
			{
				From: "internet", To: "exchange.group",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 1, Warning: 2, Bad: 0,
						Duration: time.Minute,
					},
				}},
			},
		},
		"console.group": {
			{
				From: "console.service", To: "exchange-api.service",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 100, Warning: 10, Bad: 1,
						Duration: time.Minute,
					},
				}},
			},
		},
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
			{
				From: "internet", To: "exchange-api.service",
				Traffic: TrafficLogs{Buckets: map[time.Time]RateStats{
					ts: {
						Good: 1, Warning: 2, Bad: 0,
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

		traffic := nxt.GetTraffic()
		sort.Slice(traffic, func(i, j int) bool {
			return traffic[i].From < traffic[j].From
		})
		ret[nxt.Name()] = traffic

		for _, node := range nxt.GetNodes() {
			nodes = append(nodes, node)
		}
	}
	return ret
}
