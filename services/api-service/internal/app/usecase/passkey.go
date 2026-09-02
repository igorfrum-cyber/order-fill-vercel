package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

const passkeyTTL = 2 * time.Minute

type PasskeyBegin struct {
	ChallengeID string
	Options     json.RawMessage
}

func (a *Auth) WithPasskeys(passkeys port.PasskeyCeremony) *Auth {
	a.passkeys = passkeys
	return a
}

func (a *Auth) BeginPasskeyRegistration(ctx context.Context, actor identity.User, origin string, name string) (PasskeyBegin, error) {
	if a.passkeys == nil {
		return PasskeyBegin{}, identity.ErrUnauthorized
	}
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() {
		return PasskeyBegin{}, identity.ErrUnauthorized
	}
	existing, err := a.store.ListPasskeys(ctx, user.ID)
	if err != nil {
		return PasskeyBegin{}, err
	}
	options, session, err := a.passkeys.BeginRegistration(origin, user, existing)
	if err != nil {
		return PasskeyBegin{}, err
	}
	return a.savePasskeyBegin(ctx, user.ID, identity.PasskeyPurposeRegister, options, session, name)
}

func (a *Auth) FinishPasskeyRegistration(ctx context.Context, actor identity.User, origin string, challengeID string, response json.RawMessage, name string) (identity.PasskeyPublicView, error) {
	if a.passkeys == nil {
		return identity.PasskeyPublicView{}, identity.ErrUnauthorized
	}
	challenge, err := a.store.ConsumePasskeyChallenge(ctx, strings.TrimSpace(challengeID), a.now())
	if err != nil || challenge.Purpose != identity.PasskeyPurposeRegister || challenge.UserID != actor.ID {
		return identity.PasskeyPublicView{}, identity.ErrUnauthorized
	}
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() {
		return identity.PasskeyPublicView{}, identity.ErrUnauthorized
	}
	credential, err := a.passkeys.FinishRegistration(origin, user, unwrapPasskeySession(challenge.Session), response)
	if err != nil {
		return identity.PasskeyPublicView{}, identity.ErrUnauthorized
	}
	credential.UserID = user.ID
	credential.Name = firstNonEmpty(name, challengeLabel(challenge), identity.PasskeyCredential{}.DisplayName())
	credential.CreatedAt = a.now().UTC()
	if err := a.store.SavePasskey(ctx, credential); err != nil {
		if errors.Is(err, identity.ErrConflict) {
			return identity.PasskeyPublicView{}, identity.ErrConflict
		}
		return identity.PasskeyPublicView{}, err
	}
	return credential.PublicView(), nil
}

func (a *Auth) ListPasskeys(ctx context.Context, actor identity.User) ([]identity.PasskeyPublicView, error) {
	existing, err := a.store.ListPasskeys(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]identity.PasskeyPublicView, 0, len(existing))
	for _, item := range existing {
		out = append(out, item.PublicView())
	}
	return out, nil
}

func (a *Auth) DeletePasskey(ctx context.Context, actor identity.User, id string) error {
	if err := a.store.DeletePasskey(ctx, actor.ID, strings.TrimSpace(id)); err != nil {
		return err
	}
	return nil
}

func (a *Auth) BeginPasskeyLogin(ctx context.Context, origin string, login string) (PasskeyBegin, error) {
	if a.passkeys == nil {
		return PasskeyBegin{}, identity.ErrUnauthorized
	}
	login = strings.TrimSpace(login)
	if login != "" {
		user, err := a.store.GetUserByLogin(ctx, login)
		if err == nil && !user.Disabled() {
			existing, listErr := a.store.ListPasskeys(ctx, user.ID)
			if listErr != nil {
				return PasskeyBegin{}, listErr
			}
			if len(existing) > 0 {
				options, session, beginErr := a.passkeys.BeginLogin(origin, user, existing)
				if beginErr != nil {
					return PasskeyBegin{}, beginErr
				}
				return a.savePasskeyBegin(ctx, user.ID, identity.PasskeyPurposeLogin, options, session, "")
			}
		}
	}
	options, session, err := a.passkeys.BeginDiscoverableLogin(origin)
	if err != nil {
		return PasskeyBegin{}, err
	}
	userID := ""
	if login != "" {
		if user, err := a.store.GetUserByLogin(ctx, login); err == nil && !user.Disabled() {
			userID = user.ID
		}
	}
	return a.savePasskeyBegin(ctx, userID, identity.PasskeyPurposeLogin, options, session, "")
}

func (a *Auth) FinishPasskeyLogin(ctx context.Context, origin string, challengeID string, response json.RawMessage) (Session, error) {
	if a.passkeys == nil {
		return Session{}, identity.ErrUnauthorized
	}
	challenge, err := a.store.ConsumePasskeyChallenge(ctx, strings.TrimSpace(challengeID), a.now())
	if err != nil || challenge.Purpose != identity.PasskeyPurposeLogin {
		return Session{}, identity.ErrUnauthorized
	}
	user, credential, err := a.finishPasskeyAssertion(ctx, origin, challenge, response)
	if err != nil || user.Disabled() {
		return Session{}, identity.ErrUnauthorized
	}
	now := a.now().UTC()
	stored, err := a.store.GetPasskey(ctx, credential.ID)
	if err != nil || stored.UserID != user.ID {
		return Session{}, identity.ErrUnauthorized
	}
	credential.UserID = user.ID
	credential.Name = stored.Name
	credential.CreatedAt = stored.CreatedAt
	credential.LastUsedAt = &now
	if len(credential.Raw) == 0 {
		credential.Raw = stored.Raw
	}
	if err := a.store.UpdatePasskey(ctx, credential); err != nil {
		return Session{}, err
	}
	session, err := a.issueSession(ctx, user)
	if err != nil {
		return Session{}, err
	}
	a.recordAudit(ctx, user, port.AuditLoginSuccess, user.CompanyID, "")
	return session, nil
}

func (a *Auth) finishPasskeyAssertion(ctx context.Context, origin string, challenge identity.PasskeyChallenge, response json.RawMessage) (identity.User, identity.PasskeyCredential, error) {
	if challenge.UserID != "" {
		user, err := a.store.GetUserByID(ctx, challenge.UserID)
		if err != nil {
			return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
		}
		existing, err := a.store.ListPasskeys(ctx, user.ID)
		if err != nil {
			return identity.User{}, identity.PasskeyCredential{}, err
		}
		if len(existing) == 0 {
			found, credential, err := a.passkeys.FinishDiscoverableLogin(origin, unwrapPasskeySession(challenge.Session), response, a.discoverableLookup(ctx))
			if err != nil || found.ID != user.ID {
				return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
			}
			return found, credential, nil
		}
		credential, err := a.passkeys.FinishLogin(origin, user, existing, unwrapPasskeySession(challenge.Session), response)
		if err != nil {
			return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
		}
		return user, credential, nil
	}
	found, credential, err := a.passkeys.FinishDiscoverableLogin(origin, unwrapPasskeySession(challenge.Session), response, a.discoverableLookup(ctx))
	if err != nil {
		return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	return found, credential, nil
}

func (a *Auth) discoverableLookup(ctx context.Context) port.DiscoverablePasskeyUser {
	return func(rawID []byte, userHandle []byte) (identity.User, []identity.PasskeyCredential, error) {
		if len(userHandle) == 0 {
			return identity.User{}, nil, identity.ErrUnauthorized
		}
		user, err := a.store.GetUserByID(ctx, string(userHandle))
		if err != nil || user.Disabled() {
			return identity.User{}, nil, identity.ErrUnauthorized
		}
		existing, err := a.store.ListPasskeys(ctx, user.ID)
		if err != nil {
			return identity.User{}, nil, err
		}
		_ = rawID
		return user, existing, nil
	}
}

func (a *Auth) savePasskeyBegin(ctx context.Context, userID string, purpose string, options json.RawMessage, session json.RawMessage, name string) (PasskeyBegin, error) {
	raw, err := identity.NewSecret()
	if err != nil {
		return PasskeyBegin{}, err
	}
	payload, err := json.Marshal(passkeySessionBlob{Session: session, Name: strings.TrimSpace(name)})
	if err != nil {
		return PasskeyBegin{}, err
	}
	if err := a.store.SavePasskeyChallenge(ctx, identity.PasskeyChallenge{
		ID:        raw,
		UserID:    userID,
		Purpose:   purpose,
		Session:   payload,
		ExpiresAt: a.now().Add(passkeyTTL),
	}); err != nil {
		return PasskeyBegin{}, err
	}
	return PasskeyBegin{ChallengeID: raw, Options: options}, nil
}

type passkeySessionBlob struct {
	Session json.RawMessage `json:"session"`
	Name    string          `json:"name,omitempty"`
}

func challengeLabel(challenge identity.PasskeyChallenge) string {
	var blob passkeySessionBlob
	if err := json.Unmarshal(challenge.Session, &blob); err != nil {
		return ""
	}
	return blob.Name
}

func unwrapPasskeySession(raw []byte) json.RawMessage {
	var blob passkeySessionBlob
	if err := json.Unmarshal(raw, &blob); err != nil || len(blob.Session) == 0 {
		return raw
	}
	return blob.Session
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
