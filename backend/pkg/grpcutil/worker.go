package grpcutil

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	WorkerTokenMetadataKey = "x-worker-token"
	minWorkerTokenBytes    = 16
)

// #nosec G101 -- local-dev sentinel; CheckWorkerToken rejects it outside local.
const DefaultWorkerToken = "local-dev-worker-token"

func WithWorkerToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, WorkerTokenMetadataKey, token)
}

func WorkerToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(WorkerTokenMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func WorkerAuthorized(ctx context.Context, expected string) bool {
	if expected == "" {
		return false
	}
	got := WorkerToken(ctx)
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func CheckWorkerToken(environment, token string) error {
	if localEnvironment(environment) {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" || token == DefaultWorkerToken || len(token) < minWorkerTokenBytes {
		return fmt.Errorf("WORKER_TOKEN must be a non-default secret of at least %d bytes outside local environment", minWorkerTokenBytes)
	}
	return nil
}
