package ops

import (
	"github.com/luno/gridlock/api"
	"sort"
)

func SummariseTraffic(t Traffic) []api.Traffic {
	var ret []api.Traffic
	for _, region := range t.Regions {
		for _, n := range region.Nodes {
			for target, stats := range n.Outgoing {
				ret = append(ret, api.Traffic{
					From:         n.Name,
					To:           target,
					CountGood:    stats.Good,
					CountWarning: stats.Warning,
					CountBad:     stats.Bad,
				})

			}
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].From == ret[j].From {
			return ret[i].To < ret[j].To
		}
		return ret[i].From < ret[j].From
	})
	return ret
}
