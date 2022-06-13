package ops

import (
	"github.com/luno/gridlock/api"
	"sort"
)

func SummariseTraffic(ml []api.Metrics) []api.Traffic {
	var ret []api.Traffic
	for _, m := range ml {
		ret = append(ret, api.Traffic{
			From:         m.Source,
			To:           m.Target,
			CountGood:    m.CountGood,
			CountWarning: m.CountWarning,
			CountBad:     m.CountBad,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].From == ret[j].From {
			return ret[i].To < ret[j].To
		}
		return ret[i].From < ret[j].From
	})
	return ret
}
