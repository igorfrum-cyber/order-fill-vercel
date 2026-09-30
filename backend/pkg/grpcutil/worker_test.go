package grpcutil

import (
	"context"
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

func TestWorkerTokenMetadataAndEmptyToken(t *testing.T) {
	t.Parallel()

	base := metadata.NewIncomingContext(t.Context(), metadata.Pairs(WorkerTokenMetadataKey, "incoming-token"))
	if got := WorkerToken(base); got != "incoming-token" {
		t.Fatalf("WorkerToken() = %q", got)
	}
	if got := WorkerToken(WithWorkerToken(t.Context(), "")); got != "" {
		t.Fatalf("empty outgoing token became %q", got)
	}
	withoutMetadata := WithWorkerToken(t.Context(), "worker-secret-123")
	if got := WorkerToken(withoutMetadata); got != "" {
		t.Fatalf("outgoing token should not be read as incoming: %q", got)
	}
	if WorkerAuthorized(t.Context(), "worker-secret-token") {
		t.Fatal("missing metadata must not authorize")
	}
}

func TestActorRoleAndCompanyContext(t *testing.T) {
	t.Parallel()

	ctx := WithActorRole(t.Context(), "company_admin")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(md.Get(ActorRoleMetadataKey)) != 1 || md.Get(ActorRoleMetadataKey)[0] != "company_admin" {
		t.Fatalf("actor role metadata = %v", md)
	}
	ctx = metadata.NewIncomingContext(t.Context(), md)
	if got := ActorRole(ctx); got != "company_admin" {
		t.Fatalf("ActorRole() = %q", got)
	}
	if got := ActorRole(WithActorRole(t.Context(), "")); got != "" {
		t.Fatalf("empty actor role = %q", got)
	}
	if got := ActorRole(t.Context()); got != "" {
		t.Fatalf("missing actor role = %q", got)
	}

	companyCtx := WithCompany(t.Context(), "company-1")
	if got := Company(companyCtx); got != "company-1" {
		t.Fatalf("Company() = %q", got)
	}
	if got := Company(WithCompany(t.Context(), "")); got != "" {
		t.Fatalf("empty company = %q", got)
	}
	if got := Company(t.Context()); got != "" {
		t.Fatalf("missing company = %q", got)
	}
}

func TestRequestIDContextAndMetadata(t *testing.T) {
	t.Parallel()

	if id := NewID(); len(id) != 32 {
		t.Fatalf("NewID() length = %d, value %q", len(id), id)
	}
	if got := FromContext(t.Context()); got != "" {
		t.Fatalf("FromContext() on empty context = %q", got)
	}
	provided := NewContext(t.Context(), "request-1")
	if got := FromContext(provided); got != "request-1" {
		t.Fatalf("FromContext() = %q", got)
	}
	if got := FromContext(NewContext(t.Context(), "")); got == "" {
		t.Fatal("NewContext must generate an id for an empty value")
	}

	incoming := metadata.NewIncomingContext(t.Context(), metadata.Pairs(MetadataKey, "incoming-1"))
	if got := FromContext(FromIncoming(incoming)); got != "incoming-1" {
		t.Fatalf("FromIncoming() = %q", got)
	}
	if got := FromContext(FromIncoming(provided)); got != "request-1" {
		t.Fatalf("FromIncoming replaced existing id with %q", got)
	}
	if got := FromContext(FromIncoming(t.Context())); got == "" {
		t.Fatal("FromIncoming must create a request id when missing")
	}

	outgoing := ToOutgoing(t.Context())
	if got := FromContext(outgoing); got == "" {
		t.Fatal("ToOutgoing must set the context request id")
	}
	md, ok := metadata.FromOutgoingContext(outgoing)
	if !ok || len(md.Get(MetadataKey)) != 1 || md.Get(MetadataKey)[0] != FromContext(outgoing) {
		t.Fatalf("outgoing request id metadata = %v", md)
	}
	withID := ToOutgoing(NewContext(context.Background(), "existing"))
	if got := mdValue(t, withID, MetadataKey); got != "existing" {
		t.Fatalf("ToOutgoing changed existing id to %q", got)
	}
}

func mdValue(t *testing.T, ctx context.Context, key string) string {
	t.Helper()
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(md.Get(key)) == 0 {
		t.Fatalf("missing outgoing metadata %q", key)
	}
	return md.Get(key)[0]
}
