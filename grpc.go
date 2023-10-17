package gridlock

import (
	"context"
	"strings"

	"github.com/luno/gridlock/api"
	"google.golang.org/grpc"
)

func GRPCClientReporter(c Client) grpc.UnaryClientInterceptor {
	return func(ctx context.Context,
		method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)

		success := CallGood
		if err != nil {
			success = CallBad
		}
		service, _ := splitMethodName(method)
		c.Record(Method{Target: service, Transport: api.TransportGRPC}, success)

		return err
	}
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}
