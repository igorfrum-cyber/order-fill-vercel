package grpcutil

import (
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestWorkerAuthorizedRejectsEmptyExpected(t *testing.T) {
	t.Parallel()
	if WorkerAuthorized(t.Context(), "") {
		t.Fatal("empty expected token must not authorize")
	}
}

func TestWorkerAuthorizedAcceptsMatchingMetadata(t *testing.T) {
	t.Parallel()
	ctx := WithWorkerToken(t.Context(), "worker-secret-token")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	ctx = metadata.NewIncomingContext(t.Context(), md)
	if !WorkerAuthorized(ctx, "worker-secret-token") {
		t.Fatal("matching token must authorize")
	}
	if WorkerAuthorized(ctx, "other-secret-token") {
		t.Fatal("wrong token must not authorize")
	}
}

func TestCheckWorkerTokenRejectsDefaultOutsideLocal(t *testing.T) {
	t.Parallel()
	if err := CheckWorkerToken("production", DefaultWorkerToken); err == nil {
		t.Fatal("expected default token error")
	}
	if err := CheckWorkerToken("production", "short"); err == nil {
		t.Fatal("expected short token error")
	}
	if err := CheckWorkerToken("local", ""); err != nil {
		t.Fatal(err)
	}
	if err := CheckWorkerToken("production", "production-worker-token"); err != nil {
		t.Fatal(err)
	}
}
