package webauthn

import (
	"encoding/base64"
	"encoding/json"

	"order-fill/backend/services/passkey-service/internal/domain"
)

type Account struct {
	ID    string
	Login string
}

type DiscoverableLookup func(rawID, userHandle []byte) (Account, []domain.PasskeyCredential, error)

type Ceremony interface {
	BeginRegistration(origin string, user Account, existing []domain.PasskeyCredential) (options, session []byte, err error)
	FinishRegistration(origin string, user Account, session, response []byte) (domain.PasskeyCredential, error)
	BeginLogin(origin string, user Account, existing []domain.PasskeyCredential) (options, session []byte, err error)
	FinishLogin(origin string, user Account, existing []domain.PasskeyCredential, session, response []byte) (domain.PasskeyCredential, error)
	BeginDiscoverableLogin(origin string) (options, session []byte, err error)
	FinishDiscoverableLogin(origin string, session, response []byte, lookup DiscoverableLookup) (Account, domain.PasskeyCredential, error)
}

type Fake struct {
	RegisterID string
	LoginID    string
	SignCount  uint32
}

func (f *Fake) BeginRegistration(origin string, _ Account, _ []domain.PasskeyCredential) ([]byte, []byte, error) {
	if _, err := resolveRPID(origin, ""); err != nil {
		return nil, nil, err
	}
	return []byte(`{"type":"register"}`), []byte(`{"session":"register"}`), nil
}

func (f *Fake) FinishRegistration(_ string, user Account, _, response []byte) (domain.PasskeyCredential, error) {
	id := f.RegisterID
	count := f.SignCount
	if len(response) > 0 {
		var payload struct {
			ID        string `json:"id"`
			SignCount uint32 `json:"signCount"`
		}
		if err := json.Unmarshal(response, &payload); err == nil {
			if payload.ID != "" {
				id = payload.ID
			}
			count = payload.SignCount
		}
	}
	raw := []byte(`{"id":"` + id + `","publicKey":"BKEi"}`)
	return domain.PasskeyCredential{ID: id, UserID: user.ID, SignCount: count, Raw: raw}, nil
}

func (f *Fake) BeginLogin(origin string, _ Account, existing []domain.PasskeyCredential) ([]byte, []byte, error) {
	if _, err := resolveRPID(origin, ""); err != nil {
		return nil, nil, err
	}
	if len(existing) == 0 {
		return nil, nil, domain.ErrUnauthorized
	}
	return []byte(`{"type":"login"}`), []byte(`{"session":"login"}`), nil
}

func (f *Fake) FinishLogin(_ string, user Account, existing []domain.PasskeyCredential, _, response []byte) (domain.PasskeyCredential, error) {
	id := f.LoginID
	count := f.SignCount
	if len(response) > 0 {
		var payload struct {
			ID        string `json:"id"`
			SignCount uint32 `json:"signCount"`
		}
		if err := json.Unmarshal(response, &payload); err == nil {
			if payload.ID != "" {
				id = payload.ID
			}
			count = payload.SignCount
		}
	}
	for _, cred := range existing {
		if cred.ID == id {
			cred.SignCount = count
			return cred, nil
		}
	}
	return domain.PasskeyCredential{ID: id, UserID: user.ID, SignCount: count}, nil
}

func (f *Fake) BeginDiscoverableLogin(origin string) ([]byte, []byte, error) {
	if _, err := resolveRPID(origin, ""); err != nil {
		return nil, nil, err
	}
	return []byte(`{"type":"discoverable"}`), []byte(`{"session":"discoverable"}`), nil
}

func (f *Fake) FinishDiscoverableLogin(_ string, _, response []byte, lookup DiscoverableLookup) (Account, domain.PasskeyCredential, error) {
	id := f.LoginID
	count := f.SignCount
	if len(response) > 0 {
		var payload struct {
			ID        string `json:"id"`
			SignCount uint32 `json:"signCount"`
		}
		if err := json.Unmarshal(response, &payload); err == nil {
			if payload.ID != "" {
				id = payload.ID
			}
			count = payload.SignCount
		}
	}
	if lookup == nil {
		return Account{ID: id}, domain.PasskeyCredential{ID: id, SignCount: count}, nil
	}
	rawID, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		rawID = []byte(id)
	}
	account, existing, err := lookup(rawID, []byte(id))
	if err != nil {
		return Account{}, domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	for _, cred := range existing {
		if cred.ID == id {
			cred.SignCount = count
			return account, cred, nil
		}
	}
	return account, domain.PasskeyCredential{ID: id, UserID: account.ID, SignCount: count}, nil
}
