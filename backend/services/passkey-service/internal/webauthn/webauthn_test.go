package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	library "github.com/go-webauthn/webauthn/webauthn"

	"order-fill/backend/services/passkey-service/internal/domain"
)

func TestFakeRegistrationAndLoginCeremonies(t *testing.T) {
	t.Parallel()

	fake := &Fake{RegisterID: "registered", LoginID: "login-id", SignCount: 4}
	user := Account{ID: "user-1", Login: "buyer"}
	options, session, err := fake.BeginRegistration("https://example.com", user, nil)
	if err != nil || string(options) != `{"type":"register"}` || string(session) != `{"session":"register"}` {
		t.Fatalf("BeginRegistration() = %s %s %v", options, session, err)
	}
	if _, _, err := fake.BeginRegistration("http://example.com", user, nil); err == nil {
		t.Fatal("BeginRegistration accepted insecure public origin")
	}
	registered, err := fake.FinishRegistration("https://example.com", user, session, []byte(`{"id":"new-id","signCount":12}`))
	if err != nil || registered.ID != "new-id" || registered.UserID != user.ID || registered.SignCount != 12 || len(registered.Raw) == 0 {
		t.Fatalf("FinishRegistration() = %+v, %v", registered, err)
	}
	registered, err = fake.FinishRegistration("https://example.com", user, session, []byte("invalid"))
	if err != nil || registered.ID != "registered" || registered.SignCount != fake.SignCount {
		t.Fatalf("FinishRegistration() with invalid JSON = %+v, %v", registered, err)
	}

	if _, _, err := fake.BeginLogin("https://example.com", user, nil); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("BeginLogin without credentials = %v", err)
	}
	options, session, err = fake.BeginLogin("https://example.com", user, []domain.PasskeyCredential{{ID: "login-id"}})
	if err != nil || string(options) != `{"type":"login"}` || string(session) != `{"session":"login"}` {
		t.Fatalf("BeginLogin() = %s %s %v", options, session, err)
	}
	if _, _, err := fake.BeginLogin("http://example.com", user, []domain.PasskeyCredential{{ID: "id"}}); err == nil {
		t.Fatal("BeginLogin accepted insecure public origin")
	}
	updated, err := fake.FinishLogin("https://example.com", user, []domain.PasskeyCredential{{ID: "login-id", UserID: user.ID}}, session, []byte(`{"id":"login-id","signCount":9}`))
	if err != nil || updated.ID != "login-id" || updated.SignCount != 9 || updated.UserID != user.ID {
		t.Fatalf("FinishLogin() existing credential = %+v, %v", updated, err)
	}
	newCredential, err := fake.FinishLogin("https://example.com", user, nil, session, []byte("invalid"))
	if err != nil || newCredential.ID != fake.LoginID || newCredential.SignCount != fake.SignCount {
		t.Fatalf("FinishLogin() fallback = %+v, %v", newCredential, err)
	}
}

func TestFakeDiscoverableLoginLookupPaths(t *testing.T) {
	t.Parallel()

	fake := &Fake{LoginID: "AQID", SignCount: 8}
	options, session, err := fake.BeginDiscoverableLogin("https://example.com")
	if err != nil || string(options) != `{"type":"discoverable"}` || string(session) != `{"session":"discoverable"}` {
		t.Fatalf("BeginDiscoverableLogin() = %s %s %v", options, session, err)
	}
	if _, _, err := fake.BeginDiscoverableLogin("http://example.com"); err == nil {
		t.Fatal("BeginDiscoverableLogin accepted insecure public origin")
	}
	account, credential, err := fake.FinishDiscoverableLogin("https://example.com", session, []byte(`{"signCount":3}`), nil)
	if err != nil || account.ID != "AQID" || credential.ID != "AQID" || credential.SignCount != 3 {
		t.Fatalf("FinishDiscoverableLogin() without lookup = %+v %+v %v", account, credential, err)
	}
	account, credential, err = fake.FinishDiscoverableLogin("https://example.com", session, nil, func(rawID, handle []byte) (Account, []domain.PasskeyCredential, error) {
		if !reflect.DeepEqual(rawID, []byte{1, 2, 3}) || string(handle) != "AQID" {
			t.Fatalf("lookup identifiers = %v / %q", rawID, handle)
		}
		return Account{ID: "user-2", Login: "owner"}, []domain.PasskeyCredential{{ID: "AQID", UserID: "user-2"}}, nil
	})
	if err != nil || account.ID != "user-2" || credential.UserID != "user-2" || credential.SignCount != fake.SignCount {
		t.Fatalf("FinishDiscoverableLogin() existing = %+v %+v %v", account, credential, err)
	}
	account, credential, err = fake.FinishDiscoverableLogin("https://example.com", session, []byte(`{"id":"not-base64!"}`), func(rawID, _ []byte) (Account, []domain.PasskeyCredential, error) {
		if string(rawID) != "not-base64!" {
			t.Fatalf("invalid base64 fallback = %q", rawID)
		}
		return Account{ID: "user-3"}, nil, nil
	})
	if err != nil || account.ID != "user-3" || credential.UserID != "user-3" {
		t.Fatalf("FinishDiscoverableLogin() new credential = %+v %+v %v", account, credential, err)
	}
	if _, _, err := fake.FinishDiscoverableLogin("https://example.com", session, nil, func([]byte, []byte) (Account, []domain.PasskeyCredential, error) {
		return Account{}, nil, errors.New("lookup failed")
	}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("lookup failure = %v", err)
	}
}

func TestCredentialConversionHelpers(t *testing.T) {
	t.Parallel()

	validRaw := []byte(`{"id":"AQID","publicKey":"AQ=="}`)
	valid := domain.PasskeyCredential{ID: "ignored", Raw: validRaw}
	if converted := toLibraryCreds([]domain.PasskeyCredential{valid}); len(converted) != 1 || !reflect.DeepEqual(converted[0].ID, []byte{1, 2, 3}) {
		t.Fatalf("Raw credential conversion = %+v", converted)
	}
	if converted := toLibraryCreds([]domain.PasskeyCredential{
		{ID: base64.RawURLEncoding.EncodeToString([]byte{4, 5}), PublicKey: []byte{6}, SignCount: 7},
		{ID: "!!!"}, {ID: ""}, {ID: "AQ=="},
	}); len(converted) != 1 || converted[0].Authenticator.SignCount != 7 || converted[0].PublicKey[0] != 6 {
		t.Fatalf("fallback credential conversion = %+v", converted)
	}
	if len(wrapUser(Account{ID: "user", Login: "buyer"}, nil).WebAuthnID()) == 0 {
		t.Fatal("wrapped user returned an empty WebAuthn ID")
	}
	wrapped := wrapUser(Account{ID: "user", Login: "buyer"}, []domain.PasskeyCredential{valid})
	if wrapped.WebAuthnName() != "buyer" || wrapped.WebAuthnDisplayName() != "buyer" || len(wrapped.WebAuthnCredentials()) != 1 {
		t.Fatalf("wrapped user details = %q %q %+v", wrapped.WebAuthnName(), wrapped.WebAuthnDisplayName(), wrapped.WebAuthnCredentials())
	}

	credential := &library.Credential{
		ID: []byte{1, 2, 3}, PublicKey: []byte{4}, Transport: []protocol.AuthenticatorTransport{"usb"},
		Authenticator: library.Authenticator{SignCount: 8},
	}
	converted, err := toDomain("user-4", credential)
	if err != nil || converted.ID != "AQID" || converted.UserID != "user-4" || converted.SignCount != 8 || !reflect.DeepEqual(converted.Transports, []string{"usb"}) {
		t.Fatalf("toDomain() = %+v, %v", converted, err)
	}
	if _, err := toDomain("user", nil); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("nil credential conversion = %v", err)
	}
	if got := marshalJSON(map[string]string{"ok": "yes"}); !json.Valid(got) {
		t.Fatalf("marshalJSON returned invalid JSON %q", got)
	}
	if got := string(marshalJSON(make(chan int))); got != "{}" {
		t.Fatalf("marshalJSON error fallback = %q", got)
	}
}

func TestRelyingPartyRejectsInvalidOriginBeforeCeremony(t *testing.T) {
	t.Parallel()

	relyingParty := New("", "example.com")
	if _, _, err := relyingParty.BeginRegistration("http://example.com", Account{ID: "user"}, nil); err == nil {
		t.Fatal("registration accepted an insecure public origin")
	}
	if _, _, err := relyingParty.BeginLogin("https://other.example", Account{ID: "user"}, nil); err == nil {
		t.Fatal("login accepted an origin outside the configured relying party")
	}
	if _, err := relyingParty.FinishRegistration("https://other.example", Account{}, nil, nil); err == nil {
		t.Fatal("registration finish accepted an origin outside the relying party")
	}
	if _, err := relyingParty.FinishLogin("https://other.example", Account{}, nil, nil, nil); err == nil {
		t.Fatal("login finish accepted an origin outside the relying party")
	}
	if _, _, err := relyingParty.BeginDiscoverableLogin("https://other.example"); err == nil {
		t.Fatal("discoverable login accepted an origin outside the relying party")
	}
	if _, _, err := relyingParty.FinishDiscoverableLogin("https://other.example", nil, nil, nil); err == nil {
		t.Fatal("discoverable login finish accepted an origin outside the relying party")
	}
}

func TestRelyingPartyBeginsRegistrationAndLoginCeremonies(t *testing.T) {
	t.Parallel()

	relyingParty := New("", "")
	user := Account{ID: "user-1", Login: "buyer"}
	registrationOptions, registrationSession, err := relyingParty.BeginRegistration("https://example.com", user, nil)
	if err != nil || !json.Valid(registrationOptions) || len(registrationSession) == 0 {
		t.Fatalf("BeginRegistration() = %s %s %v", registrationOptions, registrationSession, err)
	}
	loginOptions, loginSession, err := relyingParty.BeginLogin("https://example.com", user, []domain.PasskeyCredential{{ID: "AQID", PublicKey: []byte{4}}})
	if err != nil || !json.Valid(loginOptions) || len(loginSession) == 0 {
		t.Fatalf("BeginLogin() = %s %s %v", loginOptions, loginSession, err)
	}
	discoverableOptions, discoverableSession, err := relyingParty.BeginDiscoverableLogin("https://example.com")
	if err != nil || !json.Valid(discoverableOptions) || len(discoverableSession) == 0 {
		t.Fatalf("BeginDiscoverableLogin() = %s %s %v", discoverableOptions, discoverableSession, err)
	}
}
