package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/domain/identity"
)

func TestPasskeyLoginBeginRequiresOrigin(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Passkeys:       stubPasskeys{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/login/begin", strings.NewReader(`{"login":"buyer"}`))
	request.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body %s", response.Code, response.Body.String())
	}
}

func TestPasskeyLoginBeginIsPublic(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Passkeys: stubPasskeys{
			beginLogin: func(_ context.Context, origin string, login string) (usecase.PasskeyBegin, error) {
				if origin != "http://127.0.0.1:3200" || login != "buyer" {
					t.Fatalf("origin %q login %q", origin, login)
				}
				return usecase.PasskeyBegin{ChallengeID: "ch-1", Options: json.RawMessage(`{"publicKey":{}}`)}, nil
			},
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/login/begin", strings.NewReader(`{"login":"buyer"}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["challenge_id"] != "ch-1" {
		t.Fatalf("payload %#v", payload)
	}
}

func TestPasskeyRegisterBeginRequiresSession(t *testing.T) {
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Passkeys:       stubPasskeys{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/begin", strings.NewReader(`{}`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestPasskeyFinishRejectsInvalidJSON(t *testing.T) {
	token := "owner-token"
	router := NewRouter(Config{
		AllowedOrigins: []string{"http://127.0.0.1:3200"},
		Auth: stubAuth{users: map[string]identity.User{
			identity.HashSecret(token): {ID: "user-a", Login: "buyer", Role: identity.RolePurchaser},
		}},
		Passkeys: stubPasskeys{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/finish", strings.NewReader(`{`))
	request.Header.Set("X-Requested-With", "fetch")
	request.Header.Set("Origin", "http://127.0.0.1:3200")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body %s", response.Code, response.Body.String())
	}
}

type stubPasskeys struct {
	beginLogin func(context.Context, string, string) (usecase.PasskeyBegin, error)
}

func (s stubPasskeys) BeginPasskeyRegistration(context.Context, identity.User, string, string) (usecase.PasskeyBegin, error) {
	return usecase.PasskeyBegin{}, identity.ErrUnauthorized
}

func (s stubPasskeys) FinishPasskeyRegistration(context.Context, identity.User, string, string, json.RawMessage, string) (identity.PasskeyPublicView, error) {
	return identity.PasskeyPublicView{}, identity.ErrUnauthorized
}

func (s stubPasskeys) ListPasskeys(context.Context, identity.User) ([]identity.PasskeyPublicView, error) {
	return nil, nil
}

func (s stubPasskeys) DeletePasskey(context.Context, identity.User, string) error {
	return identity.ErrNotFound
}

func (s stubPasskeys) BeginPasskeyLogin(ctx context.Context, origin string, login string) (usecase.PasskeyBegin, error) {
	if s.beginLogin != nil {
		return s.beginLogin(ctx, origin, login)
	}
	return usecase.PasskeyBegin{}, identity.ErrUnauthorized
}

func (s stubPasskeys) FinishPasskeyLogin(context.Context, string, string, json.RawMessage) (usecase.Session, error) {
	return usecase.Session{}, identity.ErrUnauthorized
}
