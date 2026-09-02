package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

func TestBeginPasskeyRegistrationReturnsOptionsWithoutSecrets(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	fake := &fakePasskeys{options: json.RawMessage(`{"publicKey":{"challenge":"abc"}}`), session: json.RawMessage(`{"challenge":"abc"}`)}
	auth.WithPasskeys(fake)

	begin, err := auth.BeginPasskeyRegistration(context.Background(), user, "http://127.0.0.1:3200", "MacBook")
	if err != nil {
		t.Fatal(err)
	}
	if begin.ChallengeID == "" || string(begin.Options) != `{"publicKey":{"challenge":"abc"}}` {
		t.Fatalf("begin %#v", begin)
	}
	if fake.lastOrigin != "http://127.0.0.1:3200" {
		t.Fatalf("origin %q", fake.lastOrigin)
	}
}

func TestFinishPasskeyRegistrationStoresPublicCredential(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	raw := []byte(`{"id":"YWJj","publicKey":"BKEi","attestationType":"none"}`)
	auth.WithPasskeys(&fakePasskeys{
		options: json.RawMessage(`{"publicKey":{"challenge":"abc"}}`),
		session: json.RawMessage(`{"challenge":"abc"}`),
		credential: identity.PasskeyCredential{
			ID:        "cred-1",
			PublicKey: []byte{1, 2, 3},
			Raw:       raw,
		},
	})
	begin, err := auth.BeginPasskeyRegistration(context.Background(), user, "http://127.0.0.1:3200", "MacBook")
	if err != nil {
		t.Fatal(err)
	}
	view, err := auth.FinishPasskeyRegistration(context.Background(), user, "http://127.0.0.1:3200", begin.ChallengeID, json.RawMessage(`{"id":"cred-1"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "MacBook" || view.ID != "cred-1" {
		t.Fatalf("view %#v", view)
	}
	stored, err := store.GetPasskey(context.Background(), "cred-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.UserID != user.ID {
		t.Fatalf("user %s", stored.UserID)
	}
}

func TestFinishPasskeyRegistrationRejectsInvalidPayload(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	auth.WithPasskeys(&fakePasskeys{finishErr: identity.ErrUnauthorized})
	begin, err := auth.BeginPasskeyRegistration(context.Background(), user, "http://127.0.0.1:3200", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.FinishPasskeyRegistration(context.Background(), user, "http://127.0.0.1:3200", begin.ChallengeID, json.RawMessage(`{}`), ""); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestPasskeyLoginIssuesSessionWithoutPassword(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	raw := []byte(`{"id":"YWJj","publicKey":"BKEi","attestationType":"none"}`)
	credential := identity.PasskeyCredential{ID: "cred-1", UserID: user.ID, Name: "MacBook", PublicKey: []byte{1}, Raw: raw}
	if err := store.SavePasskey(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	auth.WithPasskeys(&fakePasskeys{
		options:    json.RawMessage(`{"publicKey":{"challenge":"login"}}`),
		session:    json.RawMessage(`{"challenge":"login"}`),
		credential: identity.PasskeyCredential{ID: "cred-1", PublicKey: []byte{1}, Raw: raw, SignCount: 2},
	})
	begin, err := auth.BeginPasskeyLogin(context.Background(), "http://127.0.0.1:3200", user.Login)
	if err != nil {
		t.Fatal(err)
	}
	session, err := auth.FinishPasskeyLogin(context.Background(), "http://127.0.0.1:3200", begin.ChallengeID, json.RawMessage(`{"id":"cred-1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if session.User.ID != user.ID || session.RawToken == "" {
		t.Fatalf("session %#v", session)
	}
	assertAudit(t, store, "login_success", user.ID, user.CompanyID)
}

func TestPasskeyLoginUnknownUserIsGeneric(t *testing.T) {
	auth, _ := newTestAuthStore(t)
	auth.WithPasskeys(&fakePasskeys{
		options: json.RawMessage(`{"publicKey":{"challenge":"x"}}`),
		session: json.RawMessage(`{"challenge":"x"}`),
	})
	begin, err := auth.BeginPasskeyLogin(context.Background(), "http://127.0.0.1:3200", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if begin.ChallengeID == "" {
		t.Fatal("expected generic challenge")
	}
	if _, err := auth.FinishPasskeyLogin(context.Background(), "http://127.0.0.1:3200", begin.ChallengeID, json.RawMessage(`{}`)); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDeletePasskeyRemovesOnlyOwnCredential(t *testing.T) {
	auth, store := newTestAuthStore(t)
	user := seedPurchaser(t, store, "buyer", "correct-horse")
	raw := []byte(`{"id":"YWJj","publicKey":"BKEi"}`)
	if err := store.SavePasskey(context.Background(), identity.PasskeyCredential{ID: "cred-1", UserID: user.ID, Name: "A", Raw: raw}); err != nil {
		t.Fatal(err)
	}
	if err := auth.DeletePasskey(context.Background(), user, "cred-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPasskey(context.Background(), "cred-1"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

type fakePasskeys struct {
	options    json.RawMessage
	session    json.RawMessage
	credential identity.PasskeyCredential
	beginErr   error
	finishErr  error
	lastOrigin string
}

func (f *fakePasskeys) BeginRegistration(origin string, _ identity.User, _ []identity.PasskeyCredential) (json.RawMessage, json.RawMessage, error) {
	f.lastOrigin = origin
	if f.beginErr != nil {
		return nil, nil, f.beginErr
	}
	return f.optionsOrDefault(), f.sessionOrDefault(), nil
}

func (f *fakePasskeys) FinishRegistration(string, identity.User, json.RawMessage, json.RawMessage) (identity.PasskeyCredential, error) {
	if f.finishErr != nil {
		return identity.PasskeyCredential{}, f.finishErr
	}
	return f.credential, nil
}

func (f *fakePasskeys) BeginLogin(origin string, _ identity.User, _ []identity.PasskeyCredential) (json.RawMessage, json.RawMessage, error) {
	f.lastOrigin = origin
	if f.beginErr != nil {
		return nil, nil, f.beginErr
	}
	return f.optionsOrDefault(), f.sessionOrDefault(), nil
}

func (f *fakePasskeys) FinishLogin(string, identity.User, []identity.PasskeyCredential, json.RawMessage, json.RawMessage) (identity.PasskeyCredential, error) {
	if f.finishErr != nil {
		return identity.PasskeyCredential{}, f.finishErr
	}
	return f.credential, nil
}

func (f *fakePasskeys) BeginDiscoverableLogin(origin string) (json.RawMessage, json.RawMessage, error) {
	f.lastOrigin = origin
	if f.beginErr != nil {
		return nil, nil, f.beginErr
	}
	return f.optionsOrDefault(), f.sessionOrDefault(), nil
}

func (f *fakePasskeys) FinishDiscoverableLogin(string, json.RawMessage, json.RawMessage, port.DiscoverablePasskeyUser) (identity.User, identity.PasskeyCredential, error) {
	if f.finishErr != nil {
		return identity.User{}, identity.PasskeyCredential{}, f.finishErr
	}
	return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
}

func (f *fakePasskeys) optionsOrDefault() json.RawMessage {
	if len(f.options) == 0 {
		return json.RawMessage(`{"publicKey":{}}`)
	}
	return f.options
}

func (f *fakePasskeys) sessionOrDefault() json.RawMessage {
	if len(f.session) == 0 {
		return json.RawMessage(`{}`)
	}
	return f.session
}
