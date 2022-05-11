package ops

import (
	"github.com/adamhicks/gridlock/api/vizceral"
)

func createNode(name string, rend vizceral.NodeRenderer, ts int64) vizceral.Node {
	return vizceral.Node{
		Class:    vizceral.ClassNormal,
		Name:     name,
		Renderer: rend,
		Updated:  ts,
	}
}

func CompileVizceralGraph(t Traffic) vizceral.Node {
	var ts int64
	if !t.To.IsZero() {
		ts = t.To.Unix()
	}

	internet := createNode("INTERNET", vizceral.RendererRegion, ts)
	root := vizceral.Node{
		Renderer:         vizceral.RendererGlobal,
		Name:             "edge",
		ServerUpdateTime: ts,
		Nodes:            []vizceral.Node{internet},
		Connections:      []vizceral.Connection{},
	}

	for name, region := range t.Regions {
		rn := createNode(name, vizceral.RendererRegion, ts)

		for nodeName, node := range region.Nodes {
			n := createNode(nodeName, vizceral.RendererFocusedChild, ts)

			for target, stats := range node.Outgoing {
				rn.Connections = append(rn.Connections, vizceral.Connection{
					Source: nodeName,
					Target: target,
					Metrics: vizceral.Metrics{
						Normal:  stats.Good,
						Warning: stats.Warning,
						Danger:  stats.Bad,
					},
				})
				rn.MaxVolume += float64(stats.Good + stats.Warning + stats.Bad)
			}
			rn.Nodes = append(rn.Nodes, n)
		}
		root.Nodes = append(root.Nodes, rn)
		root.Connections = append(root.Connections, vizceral.Connection{
			Source: "INTERNET",
			Target: rn.Name,
		})
	}
	return root
}
