package grpcapi_test

import (
	"testing"
	"time"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
	"order-fill/backend/services/identity-service/internal/transport/grpcapi"
)

func TestGetMeResolvesActorWithoutSession(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	if err := store.CreateCompany(t.Context(), domain.Company{ID: "co-1", Name: "Acme", LoginSlug: "acme", MatchingMode: domain.MatchingModeStandard}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(t.Context(), domain.User{
		ID: "u1", Login: "buyer", Role: domain.RolePurchaser, CompanyID: "co-1",
	}); err != nil {
		t.Fatal(err)
	}
	clock := func() time.Time { return now }
	srv := grpcapi.NewServer(auth.New(store, nil, nil, clock), users.New(store, clock), companies.New(store, clock))
	resp, err := srv.GetMe(t.Context(), &identityv1.GetMeRequest{Meta: &commonv1.RequestMeta{ActorUserId: "u1"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetUser().GetId() != "u1" || resp.GetUser().GetRole() != string(domain.RolePurchaser) {
		t.Fatalf("%+v", resp.GetUser())
	}
}
