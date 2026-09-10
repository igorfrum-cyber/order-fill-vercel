package auth

import (
	"testing"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/storage/memory"
)

func TestBootstrapCreatesAdminOnce(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := New(store, nil, nil, func() time.Time { return now })
	raw, created, err := svc.Bootstrap(t.Context(), "admin")
	if err != nil || !created || raw == "" {
		t.Fatalf("created=%v raw=%q err=%v", created, raw, err)
	}
	again, created, err := svc.Bootstrap(t.Context(), "admin")
	if err != nil || created || again != "" {
		t.Fatalf("second created=%v token=%q err=%v", created, again, err)
	}
	user, err := store.GetUserByLogin(t.Context(), "admin")
	if err != nil || user.Role != domain.RolePlatformAdmin {
		t.Fatalf("user=%+v err=%v", user, err)
	}
}
