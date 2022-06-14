package ops

import (
	"context"
	"github.com/luno/gridlock/api"
)

type NodeRegistry interface {
	RegisterNodes(context.Context, ...api.NodeInfo) error
	GetNodes(context.Context) ([]api.NodeInfo, error)
}
