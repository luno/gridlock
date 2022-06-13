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
	for _, m := range ml {
		g.AddMetric(m)
	}
	ts := db.GetBucket(at).Unix()

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
			if node.Type == NodeDatabase {
				n.NodeType = vizceral.NodeStorage
			}
			rn.Nodes = append(rn.Nodes, n)
		}

		for _, m := range ml {
			if m.SourceRegion != m.TargetRegion || m.SourceRegion != regionName {
				continue
			}
			rn.Connections = append(rn.Connections, vizceral.Connection{
				Source: m.Source,
				Target: m.Target,
				Metrics: vizceral.Metrics{
					Normal:  float64(m.CountGood) / 60,
					Warning: float64(m.CountWarning) / 60,
					Danger:  float64(m.CountBad) / 60,
				},
			})
			this := float64(m.CountGood+m.CountWarning+m.CountBad) / 60
			if this > rn.MaxVolume {
				rn.MaxVolume = this
			}
		}

		root.Nodes = append(root.Nodes, rn)
		root.Connections = append(root.Connections, vizceral.Connection{
			Source: "INTERNET",
			Target: rn.Name,
		})
	}
	return root
}
