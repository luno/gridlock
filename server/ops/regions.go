package ops

import (
	"github.com/luno/gridlock/api"
	"strings"
)

func isFromInternet(m api.Metrics) bool {
	return strings.ToLower(m.Source) == "internet"
}

type RegionalTraffic map[string]map[string]Stats

func NewRegionalTraffic() RegionalTraffic {
	return make(RegionalTraffic)
}

func (t RegionalTraffic) ensure(key string) map[string]Stats {
	_, ok := t[key]
	if !ok {
		t[key] = make(map[string]Stats)
	}
	return t[key]
}

func (t RegionalTraffic) AddMetric(m api.Metrics) {
	rgn := t.ensure(m.SourceRegion)
	if isFromInternet(m) {
		rgn = t.ensure("INTERNET")
	} else if m.SourceRegion == m.TargetRegion {
		return
	}
	s := rgn[m.TargetRegion].Add(Stats{
		Good:    m.CountGood,
		Warning: m.CountWarning,
		Bad:     m.CountBad,
	})
	rgn[m.TargetRegion] = s
}
