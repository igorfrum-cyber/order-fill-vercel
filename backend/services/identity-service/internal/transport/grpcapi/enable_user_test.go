package grpcapi_test

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/service/auth"
	"order-fill/backend/services/identity-service/internal/service/companies"
	"order-fill/backend/services/identity-service/internal/service/users"
	"order-fill/backend/services/identity-service/internal/storage/memory"
	"order-fill/backend/services/identity-service/internal/transport/grpcapi"
)

func TestEnableUserProtectsPrimaryAdmin(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	primary := domain.User{ID: "root", Login: "root", Role: domain.RolePlatformAdmin, IsPrimaryAdmin: true}
	secondary := domain.User{ID: "ops", Login: "ops", Role: domain.RolePlatformAdmin}
	for _, user := range []domain.User{primary, secondary} {
		if err := store.CreateUser(t.Context(), user); err != nil {
			t.Fatal(err)
		}
	}
	clock := func() time.Time { return now }
	srv := grpcapi.NewServer(auth.New(store, nil, nil, clock), users.New(store, clock), companies.New(store, clock))

	_, err := srv.EnableUser(t.Context(), &identityv1.EnableUserRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "ops"}, UserId: "root",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("secondary enabling primary: %v", err)
	}
	if _, err := srv.EnableUser(t.Context(), &identityv1.EnableUserRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "root"}, UserId: "ops",
	}); err != nil {
		t.Fatalf("primary enabling secondary: %v", err)
	}
}
