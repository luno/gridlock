package handlers

import "github.com/adamhicks/gridlock/server/ops"

type Deps interface {
	TrafficStats() ops.TrafficStats
}
