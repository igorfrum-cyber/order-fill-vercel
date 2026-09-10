package grpcutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"google.golang.org/grpc/metadata"
)

const MetadataKey = "x-request-id"

type ctxKey struct{}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}

func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func NewContext(ctx context.Context, id string) context.Context {
	if id == "" {
		id = NewID()
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

func FromIncoming(ctx context.Context) context.Context {
	if id := FromContext(ctx); id != "" {
		return ctx
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get(MetadataKey); len(values) > 0 && values[0] != "" {
			return NewContext(ctx, values[0])
		}
	}
	return NewContext(ctx, NewID())
}

func ToOutgoing(ctx context.Context) context.Context {
	id := FromContext(ctx)
	if id == "" {
		id = NewID()
		ctx = NewContext(ctx, id)
	}
	return metadata.AppendToOutgoingContext(ctx, MetadataKey, id)
}
