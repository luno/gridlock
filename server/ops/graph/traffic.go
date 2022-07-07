package graph

import "time"

type RateStats struct {
	Good     int64
	Warning  int64
	Bad      int64
	Duration time.Duration
}

func (s RateStats) IsZero() bool {
	return s.Duration == 0
}

func (s RateStats) Sum(o RateStats) RateStats {
	if s.Duration == 0 {
		return o
	}
	if s.Duration != o.Duration {
		panic("not matching duration")
	}
	s.Good += o.Good
	s.Warning += o.Warning
	s.Bad += o.Bad
	return s
}

func (s RateStats) Extend(o RateStats) RateStats {
	s.Good += o.Good
	s.Warning += o.Warning
	s.Bad += o.Bad
	s.Duration += o.Duration
	return s
}

func (s RateStats) GoodRate() float64 {
	return float64(s.Good) / s.Duration.Seconds()
}

func (s RateStats) WarningRate() float64 {
	return float64(s.Warning) / s.Duration.Seconds()
}

func (s RateStats) BadRate() float64 {
	return float64(s.Bad) / s.Duration.Seconds()
}

type TrafficLogs struct {
	Buckets map[time.Time]RateStats
}

func NewArc() TrafficLogs {
	return TrafficLogs{Buckets: make(map[time.Time]RateStats)}
}

func (a TrafficLogs) Add(t time.Time, s RateStats) {
	old := a.Buckets[t]
	a.Buckets[t] = old.Sum(s)
}

func (a TrafficLogs) LastTimestamp() time.Time {
	var t time.Time
	for b := range a.Buckets {
		if b.After(t) {
			t = b
		}
	}
	return t
}

func (a TrafficLogs) Max() RateStats {
	var ret RateStats
	for _, s := range a.Buckets {
		if s.Good+s.Warning+s.Bad > ret.Good+ret.Warning+ret.Bad {
			ret = s
		}
	}
	return ret
}

func (a TrafficLogs) Summary(f TimeInclusionFunc) RateStats {
	var ret RateStats
	for t, s := range a.Buckets {
		if f(t, s.Duration) > 0 {
			ret = ret.Extend(s)
		}
	}
	return ret
}

type NodeTraffic map[string]map[string]TrafficLogs

func NewTraffic() NodeTraffic {
	return make(NodeTraffic)
}

func (t NodeTraffic) Add(from, to string, ts time.Time, s RateStats) {
	tgt, ok := t[from]
	if !ok {
		tgt = make(map[string]TrafficLogs)
		t[from] = tgt
	}
	arc, ok := tgt[to]
	if !ok {
		arc = NewArc()
		tgt[to] = arc
	}
	arc.Add(ts, s)
}

func (t NodeTraffic) Flatten() []Arc {
	var r []Arc
	for from, tgt := range t {
		for to, t := range tgt {
			r = append(r, Arc{
				From:    from,
				To:      to,
				Traffic: t,
			})
		}
	}
	return r
}
