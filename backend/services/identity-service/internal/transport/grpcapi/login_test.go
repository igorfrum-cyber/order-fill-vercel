package grpcapi_test

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/password"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
	"order-fill/backend/services/identity-service/internal/transport/grpcapi"
)

func TestLoginMapsMissingAndWrongPasswordToUnauthenticated(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	hash, err := password.HashPassword("password10")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCompany(t.Context(), domain.Company{ID: "co-1", Name: "Acme", LoginSlug: "acme", MatchingMode: domain.MatchingModeStandard}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(t.Context(), domain.User{
		ID: "u1", Login: "buyer", PasswordHash: hash, Role: domain.RolePurchaser, CompanyID: "co-1",
	}); err != nil {
		t.Fatal(err)
	}
	clock := func() time.Time { return now }
	srv := grpcapi.NewServer(auth.New(store, nil, nil, clock), users.New(store, clock), companies.New(store, clock))

	_, missing := srv.Login(t.Context(), &identityv1.LoginRequest{Login: "nobody", Password: "password10"})
	_, wrong := srv.Login(t.Context(), &identityv1.LoginRequest{Login: "buyer", Password: "wrong-password-that-is-long"})
	if status.Code(missing) != codes.Unauthenticated || status.Code(wrong) != codes.Unauthenticated {
		t.Fatalf("missing=%v wrong=%v", missing, wrong)
	}
	if status.Convert(missing).Message() != status.Convert(wrong).Message() {
		t.Fatalf("messages differ: %q vs %q", missing, wrong)
	}
}
