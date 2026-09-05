package ceremony

import (
	"testing"
	"time"

	"order-fill/backend/services/passkey-service/internal/domain"
)

func TestOpenEmptyUsesMemoryCeremony(t *testing.T) {
	t.Parallel()
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store.Put(domain.PasskeyChallenge{ID: "c1", UserID: "u1", Purpose: domain.PasskeyPurposeLogin, ExpiresAt: now.Add(time.Minute)})
	got, err := store.Consume("c1", now)
	if err != nil || got.UserID != "u1" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := store.Consume("c1", now); err != domain.ErrUnauthorized {
		t.Fatalf("second consume err=%v", err)
	}
}
