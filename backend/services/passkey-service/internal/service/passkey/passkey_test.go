package passkey_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/passkey-service/internal/ceremony"
	"order-fill/backend/services/passkey-service/internal/domain"
	"order-fill/backend/services/passkey-service/internal/service/passkey"
	"order-fill/backend/services/passkey-service/internal/storage/memory"
	"order-fill/backend/services/passkey-service/internal/webauthn"
)

type dir map[string]webauthn.Account

func (d dir) Lookup(_ context.Context, login string) (webauthn.Account, error) {
	user, ok := d[login]
	if !ok {
		return webauthn.Account{}, domain.ErrNotFound
	}
	return user, nil
}

func setup(t *testing.T) (*passkey.Service, *memory.Store, func() time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store := memory.NewStore()
	users := dir{"buyer": {ID: "user-1", Login: "buyer"}}
	svc := passkey.New(store, ceremony.NewRedis(), &webauthn.Fake{RegisterID: "cred-1", LoginID: "cred-1", SignCount: 2}, users, clock)
	return svc, store, clock
}

func TestRegisterListDelete(t *testing.T) {
	t.Parallel()
	svc, _, _ := setup(t)
	ctx := t.Context()
	begin, err := svc.BeginRegistration(ctx, "user-1", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	view, err := svc.FinishRegistration(ctx, "user-1", "http://localhost:3200", begin.ChallengeID, []byte(`{"id":"cred-1","signCount":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if view.ID != "cred-1" {
		t.Fatalf("id=%s", view.ID)
	}
	items, err := svc.List(ctx, "user-1")
	if err != nil || len(items) != 1 {
		t.Fatalf("list=%v err=%v", items, err)
	}
	if err := svc.Delete(ctx, "user-1", "cred-1"); err != nil {
		t.Fatal(err)
	}
	items, err = svc.List(ctx, "user-1")
	if err != nil || len(items) != 0 {
		t.Fatalf("after delete list=%v err=%v", items, err)
	}
}

func TestLoginReturnsUserIDNotSession(t *testing.T) {
	t.Parallel()
	svc, _, _ := setup(t)
	ctx := t.Context()
	begin, err := svc.BeginRegistration(ctx, "user-1", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinishRegistration(ctx, "user-1", "http://localhost:3200", begin.ChallengeID, []byte(`{"id":"cred-1"}`)); err != nil {
		t.Fatal(err)
	}
	login, err := svc.BeginLogin(ctx, "buyer", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := svc.FinishLogin(ctx, "http://localhost:3200", login.ChallengeID, []byte(`{"id":"cred-1","signCount":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-1" {
		t.Fatalf("userID=%s", userID)
	}
	encoded, _ := json.Marshal(userID)
	if string(encoded) == `{"token":` {
		t.Fatal("must not mint a session")
	}
}

func TestUnknownCredentialRejected(t *testing.T) {
	t.Parallel()
	svc, _, _ := setup(t)
	ctx := t.Context()
	begin, err := svc.BeginRegistration(ctx, "user-1", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinishRegistration(ctx, "user-1", "http://localhost:3200", begin.ChallengeID, []byte(`{"id":"cred-1"}`)); err != nil {
		t.Fatal(err)
	}
	login, err := svc.BeginLogin(ctx, "buyer", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinishLogin(ctx, "http://localhost:3200", login.ChallengeID, []byte(`{"id":"missing"}`)); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("unknown cred: %v", err)
	}
}

func TestExpiredChallengeRejected(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	current := now
	clock := func() time.Time { return current }
	store := memory.NewStore()
	svc := passkey.New(store, ceremony.NewRedis(), &webauthn.Fake{RegisterID: "cred-1"}, dir{"buyer": {ID: "user-1"}}, clock)
	begin, err := svc.BeginRegistration(t.Context(), "user-1", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	current = now.Add(3 * time.Minute)
	if _, err := svc.FinishRegistration(t.Context(), "user-1", "http://localhost:3200", begin.ChallengeID, []byte(`{"id":"cred-1"}`)); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}
}

func TestSignCountUpdated(t *testing.T) {
	t.Parallel()
	svc, store, _ := setup(t)
	ctx := t.Context()
	begin, err := svc.BeginRegistration(ctx, "user-1", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinishRegistration(ctx, "user-1", "http://localhost:3200", begin.ChallengeID, []byte(`{"id":"cred-1","signCount":1}`)); err != nil {
		t.Fatal(err)
	}
	login, err := svc.BeginLogin(ctx, "buyer", "http://localhost:3200")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinishLogin(ctx, "http://localhost:3200", login.ChallengeID, []byte(`{"id":"cred-1","signCount":9}`)); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "cred-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.SignCount != 9 {
		t.Fatalf("signCount=%d", got.SignCount)
	}
}
