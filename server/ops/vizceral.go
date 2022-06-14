package ops

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/api/vizceral"
	"github.com/luno/gridlock/server/db"
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

func CompileVizceralGraph(ml []api.Metrics, at time.Time) vizceral.Node {
	g := NewGraph()
	ts := db.GetBucket(at).Previous().Unix()
	for _, m := range ml {
		g.AddMetric(m, m.Timestamp == ts)
	}

	internet := createNode("INTERNET", vizceral.RendererRegion, ts)
	root := vizceral.Node{
		Renderer:         vizceral.RendererGlobal,
		Name:             "edge",
		ServerUpdateTime: ts,
		Nodes:            []vizceral.Node{internet},
		Connections:      []vizceral.Connection{},
	}

	for regionName, region := range g.Regions {
		rn := createNode(regionName, vizceral.RendererRegion, ts)

		for nodeName, node := range region.Nodes {
			n := createNode(nodeName, vizceral.RendererFocusedChild, ts)
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
						Normal:  float64(stats.Good) / 60,
						Warning: float64(stats.Warning) / 60,
						Danger:  float64(stats.Bad) / 60,
					},
				})
				this := float64(stats.Good+stats.Warning+stats.Bad) / 60
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
					Normal:  float64(s.Good) / 60,
					Warning: float64(s.Warning) / 60,
					Danger:  float64(s.Bad) / 60,
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
				Normal:  float64(s.Good) / 60,
				Warning: float64(s.Warning) / 60,
				Danger:  float64(s.Bad) / 60,
			},
		})
	}
	return root
}
