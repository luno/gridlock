package handlers

import "github.com/luno/gridlock/server/ops"

type Deps interface {
	TrafficStats() ops.TrafficStats
	NodeRegistry() ops.NodeRegistry
}
