package ops

import (
	"github.com/luno/gridlock/api"
	"github.com/luno/gridlock/server/db"
	"sort"
	"time"
)

func SummariseTraffic(ml []api.Metrics, ts time.Time) []api.Traffic {
	var ret []api.Traffic
	if !ts.IsZero() {
		ts = db.BucketFromTime(ts).Time
	}
	for _, m := range ml {
		buck := time.Unix(m.Timestamp, 0)
		if !ts.IsZero() && !buck.Equal(ts) {
			continue
		}
		ret = append(ret, api.Traffic{
			From:         m.Source,
			To:           m.Target,
			Ts:           m.Timestamp,
			Duration:     int(db.BucketDuration.Seconds()),
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
