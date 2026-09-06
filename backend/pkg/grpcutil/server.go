package grpcutil

import (
	"context"

	"google.golang.org/grpc"
)

const MaxMsgSize = 64 << 20

func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	serverOpts, err := serverTransportOptions()
	if err != nil {
		panic(err)
	}
	chain := []grpc.UnaryServerInterceptor{requestIDUnary}
	opts = append([]grpc.ServerOption{
		grpc.ChainUnaryInterceptor(chain...),
		grpc.MaxRecvMsgSize(MaxMsgSize),
		grpc.MaxSendMsgSize(MaxMsgSize),
	}, opts...)
	opts = append(serverOpts, opts...)
	return grpc.NewServer(opts...)
}

func requestIDUnary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(FromIncoming(ctx), req)
}
