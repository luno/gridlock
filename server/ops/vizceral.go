package ops

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/api/vizceral"
	"github.com/luno/gridlock/server/ops/config"
	"github.com/luno/gridlock/server/ops/graph"
	"time"
)

func compileNode(node graph.Node, tInc graph.TimeInclusionFunc) vizceral.Node {
	ret := vizceral.Node{
		Name:        node.Name(),
		DisplayName: node.DisplayName(),
	}
	if node.IsAuxiliary() {
		ret.Class = vizceral.ClassAuxiliary
	}

	switch node.Type() {
	case graph.NodeGroup:
		ret.Renderer = vizceral.RendererRegion
	case graph.NodeRegion:
		ret.Renderer = vizceral.RendererRegion
	case graph.NodeGlobal:
		ret.Renderer = vizceral.RendererGlobal
	default:
		ret.Renderer = vizceral.RendererFocusedChild
	}

	switch node.Type() {
	case graph.NodeGroup:
		ret.NodeType = vizceral.NodeService
	case graph.NodeDatabase:
		ret.NodeType = vizceral.NodeStorage
	case graph.NodeUser:
		ret.NodeType = vizceral.NodeUsers
	}

	for _, n := range node.GetNodes() {
		ret.Nodes = append(ret.Nodes, compileNode(n, tInc))
	}

	var lastUpdate time.Time

	for _, t := range node.GetTraffic() {
		max := t.Traffic.Max()
		if !max.IsZero() {
			vol := max.GoodRate() + max.WarningRate() + max.BadRate()
			if vol > ret.MaxVolume {
				ret.MaxVolume = vol
			}
		}
		last := t.Traffic.LastTimestamp()
		if last.After(lastUpdate) {
			lastUpdate = last
		}

		stats := t.Traffic.Summary(tInc)
		if stats.IsZero() {
			continue
		}
		m := vizceral.Metrics{
			Normal:  stats.GoodRate(),
			Warning: stats.WarningRate(),
			Danger:  stats.BadRate(),
		}
		ret.MaxVolume += m.Normal + m.Warning + m.Danger
		ret.Connections = append(ret.Connections,
			vizceral.Connection{
				Source:  t.From,
				Target:  t.To,
				Metrics: m,
			},
		)
	}
	ret.ServerUpdateTime = lastUpdate.Unix()

	return ret
}

func CompileVizceralGraph(ml []api.Metrics, from, to time.Time) vizceral.Node {
	g := graph.ConstructGraph(
		graph.Builder{Config: config.GetConfig()},
		ml,
	)
	r := graph.Range{From: from, To: to}
	return compileNode(g, r.Include)
}
