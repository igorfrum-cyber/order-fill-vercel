package grpcutil

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const ActorRoleMetadataKey = "x-actor-role"

func WithActorRole(ctx context.Context, role string) context.Context {
	if role == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, ActorRoleMetadataKey, role)
}

func ActorRole(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(ActorRoleMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
