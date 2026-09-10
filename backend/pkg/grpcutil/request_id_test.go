package grpcutil

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestFromIncomingUsesMetadata(t *testing.T) {
	t.Parallel()
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs(MetadataKey, "abc"))
	ctx = FromIncoming(ctx)
	if FromContext(ctx) != "abc" {
		t.Fatalf("id=%q", FromContext(ctx))
	}
}

func TestFromIncomingGeneratesWhenMissing(t *testing.T) {
	t.Parallel()
	ctx := FromIncoming(t.Context())
	if FromContext(ctx) == "" {
		t.Fatal("empty id")
	}
}

func TestToOutgoingAppendsMetadata(t *testing.T) {
	t.Parallel()
	ctx := NewContext(t.Context(), "req-1")
	ctx = ToOutgoing(ctx)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || md.Get(MetadataKey)[0] != "req-1" {
		t.Fatalf("outgoing md=%v ok=%v", md, ok)
	}
}

func TestUnaryTimeoutSetsDeadline(t *testing.T) {
	t.Parallel()
	err := unaryTimeout(t.Context(), "/test", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > DefaultCallTimeout+time.Second {
			t.Fatalf("deadline ok=%v until=%v", ok, time.Until(deadline))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
