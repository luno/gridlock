package gridlock

import (
	"github.com/adamhicks/gridlock/api"
)

type CallSuccess int

const (
	CallGood    CallSuccess = 1
	CallWarning CallSuccess = 2
	CallBad     CallSuccess = 3
)

type Method struct {
	Source       string
	SourceRegion string
	Target       string
	TargetRegion string

	Transport api.Transport
}

func (m Method) Merge(def Method) Method {
	if m.Source == "" {
		m.Source = def.Source
	}
	if m.SourceRegion == "" {
		m.SourceRegion = def.SourceRegion
	}
	if m.Target == "" {
		m.Target = def.Target
	}
	if m.TargetRegion == "" {
		m.TargetRegion = def.TargetRegion
	}
	if m.Transport == "" {
		m.Transport = def.Transport
	}
	return m
}
