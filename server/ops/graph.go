package ops

import "time"

type Graph struct {
	LatestBucket time.Time
	Regions      map[string]*Region
}

func NewGraph() Graph {
	return Graph{Regions: make(map[string]*Region)}
}

func (t Graph) EnsureRegion(region string) *Region {
	r, ok := t.Regions[region]
	if !ok {
		r = &Region{Nodes: make(map[string]struct{})}
		t.Regions[region] = r
	}
	return r
}

type Region struct {
	Nodes map[string]struct{}
}

func (r *Region) EnsureNode(name string) {
	_, ok := r.Nodes[name]
	if !ok {
		r.Nodes[name] = struct{}{}
	}
}
