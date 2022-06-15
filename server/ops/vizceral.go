package ops

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/api/vizceral"
	"time"
)

func createNode(name string, rend vizceral.NodeRenderer, ts int64) vizceral.Node {
	return vizceral.Node{
		Class:    vizceral.ClassNormal,
		Name:     name,
		Renderer: rend,
		Updated:  ts,
	}
}

func CompileVizceralGraph(ml []api.Metrics, from, to time.Time) vizceral.Node {
	g := NewGraph()
	fromTs := from.Unix()
	toTs := to.Unix()
	var last int64
	for _, m := range ml {
		if m.Timestamp > last {
			last = m.Timestamp
		}
		add := m.Timestamp >= fromTs && m.Timestamp <= toTs
		g.AddMetric(m, add)
	}

	internet := createNode("INTERNET", vizceral.RendererRegion, last)
	root := vizceral.Node{
		Renderer:         vizceral.RendererGlobal,
		Name:             "edge",
		ServerUpdateTime: last,
		Nodes:            []vizceral.Node{internet},
		Connections:      []vizceral.Connection{},
	}

	for regionName, region := range g.Regions {
		rn := createNode(regionName, vizceral.RendererRegion, last)

		for nodeName, node := range region.Nodes {
			n := createNode(nodeName, vizceral.RendererFocusedChild, last)
			switch node.Type {
			case NodeDatabase:
				n.NodeType = vizceral.NodeStorage
			case NodeUser:
				n.NodeType = vizceral.NodeUsers
			}
			rn.Nodes = append(rn.Nodes, n)

			for target, stats := range node.Outgoing {
				rn.Connections = append(rn.Connections, vizceral.Connection{
					Source: nodeName,
					Target: target,
					Metrics: vizceral.Metrics{
						Normal:  stats.GoodRate(),
						Warning: stats.WarningRate(),
						Danger:  stats.BadRate(),
					},
				})
				this := stats.GoodRate() + stats.WarningRate() + stats.BadRate()
				if this > rn.MaxVolume {
					rn.MaxVolume = this
				}
			}
		}

		for targetRegion, s := range region.Cross {
			root.Connections = append(root.Connections, vizceral.Connection{
				Source: regionName,
				Target: targetRegion,
				Metrics: vizceral.Metrics{
					Normal:  s.GoodRate(),
					Warning: s.WarningRate(),
					Danger:  s.BadRate(),
				},
			})
		}
		root.Nodes = append(root.Nodes, rn)
	}

	for region, s := range g.Incoming {
		root.Connections = append(root.Connections, vizceral.Connection{
			Source: "INTERNET",
			Target: region,
			Metrics: vizceral.Metrics{
				Normal:  s.GoodRate(),
				Warning: s.WarningRate(),
				Danger:  s.BadRate(),
			},
		})
	}
	return root
}
