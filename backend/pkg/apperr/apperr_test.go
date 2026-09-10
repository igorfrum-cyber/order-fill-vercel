package apperr

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestHTTPStatusAndGRPCCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind Kind
		http int
		grpc codes.Code
	}{
		{name: "invalid", kind: KindInvalid, http: http.StatusBadRequest, grpc: codes.InvalidArgument},
		{name: "unauthenticated", kind: KindUnauthenticated, http: http.StatusUnauthorized, grpc: codes.Unauthenticated},
		{name: "permission", kind: KindPermissionDenied, http: http.StatusForbidden, grpc: codes.PermissionDenied},
		{name: "not found", kind: KindNotFound, http: http.StatusNotFound, grpc: codes.NotFound},
		{name: "conflict", kind: KindConflict, http: http.StatusConflict, grpc: codes.AlreadyExists},
		{name: "failed precondition", kind: KindFailedPrecondition, http: http.StatusConflict, grpc: codes.FailedPrecondition},
		{name: "unavailable", kind: KindUnavailable, http: http.StatusServiceUnavailable, grpc: codes.Unavailable},
		{name: "internal", kind: KindInternal, http: http.StatusInternalServerError, grpc: codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := New(tt.kind, string(tt.kind))
			if got := HTTPStatus(err); got != tt.http {
				t.Fatalf("HTTPStatus=%d want %d", got, tt.http)
			}
			if got := GRPCCode(err); got != tt.grpc {
				t.Fatalf("GRPCCode=%v want %v", got, tt.grpc)
			}
		})
	}
}

func TestKindOfWrapped(t *testing.T) {
	t.Parallel()
	inner := errors.New("sql: no rows")
	err := Wrap(KindNotFound, "job missing", inner)
	if KindOf(err) != KindNotFound {
		t.Fatalf("KindOf=%q", KindOf(err))
	}
	if !errors.Is(err, inner) {
		t.Fatal("expected unwrap to match")
	}
	plain := errors.New("boom")
	if KindOf(plain) != KindInternal {
		t.Fatalf("plain error kind=%q", KindOf(plain))
	}
}

func TestAsError(t *testing.T) {
	t.Parallel()
	err := error(New(KindConflict, "login taken"))
	got, ok := AsError(err)
	if !ok || got.Kind != KindConflict {
		t.Fatalf("AsError got %#v ok=%v", got, ok)
	}
}
