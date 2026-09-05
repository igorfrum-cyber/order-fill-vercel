package grpcutil

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const DefaultCallTimeout = 60 * time.Second

func Dial(_ context.Context, target string) (*grpc.ClientConn, error) {
	// ponytail: insecure until services leave the compose network; add TLS then.
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(MaxMsgSize),
			grpc.MaxCallSendMsgSize(MaxMsgSize),
		),
		grpc.WithChainUnaryInterceptor(outgoingRequestID, unaryTimeout),
	)
}

func outgoingRequestID(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(ToOutgoing(ctx), method, req, reply, cc, opts...)
}

func unaryTimeout(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCallTimeout)
		defer cancel()
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func DeadlineBound(ctx context.Context, bound time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, bound)
}
