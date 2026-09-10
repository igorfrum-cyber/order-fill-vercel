package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"order-fill/backend/services/passkey-service/internal/domain"
)

type RelyingParty struct {
	displayName string
	rpID        string
}

func New(displayName, rpID string) *RelyingParty {
	if displayName == "" {
		displayName = "Order Fill"
	}
	return &RelyingParty{displayName: displayName, rpID: rpID}
}

func (r *RelyingParty) BeginRegistration(origin string, user Account, existing []domain.PasskeyCredential) ([]byte, []byte, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return nil, nil, err
	}
	wrapped := wrapUser(user, existing)
	creation, session, err := wa.BeginRegistration(wrapped,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
		webauthn.WithExclusions(webauthn.Credentials(wrapped.creds).CredentialDescriptors()),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithExtensions(webauthn.WithExtensionCredProps()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("begin passkey registration: %w", err)
	}
	return marshalJSON(creation), marshalJSON(session), nil
}

func (r *RelyingParty) FinishRegistration(origin string, user Account, session, response []byte) (domain.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return domain.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	cred, err := wa.CreateCredential(wrapUser(user, nil), data, parsed)
	if err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	return toDomain(user.ID, cred)
}

func (r *RelyingParty) BeginLogin(origin string, user Account, existing []domain.PasskeyCredential) ([]byte, []byte, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return nil, nil, err
	}
	assertion, session, err := wa.BeginLogin(wrapUser(user, existing))
	if err != nil {
		return nil, nil, fmt.Errorf("begin passkey login: %w", err)
	}
	return marshalJSON(assertion), marshalJSON(session), nil
}

func (r *RelyingParty) FinishLogin(origin string, user Account, existing []domain.PasskeyCredential, session, response []byte) (domain.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return domain.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	cred, err := wa.ValidateLogin(wrapUser(user, existing), data, parsed)
	if err != nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	return toDomain(user.ID, cred)
}

func (r *RelyingParty) BeginDiscoverableLogin(origin string) ([]byte, []byte, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return nil, nil, err
	}
	assertion, session, err := wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin discoverable passkey login: %w", err)
	}
	return marshalJSON(assertion), marshalJSON(session), nil
}

func (r *RelyingParty) FinishDiscoverableLogin(origin string, session, response []byte, lookup DiscoverableLookup) (Account, domain.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return Account{}, domain.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return Account{}, domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return Account{}, domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	var found Account
	user, cred, err := wa.ValidatePasskeyLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		account, creds, lookupErr := lookup(rawID, userHandle)
		if lookupErr != nil {
			return nil, lookupErr
		}
		found = account
		return wrapUser(account, creds), nil
	}, data, parsed)
	if err != nil {
		return Account{}, domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	if wrapped, ok := user.(*wrappedUser); ok {
		found = wrapped.user
	}
	converted, err := toDomain(found.ID, cred)
	if err != nil {
		return Account{}, domain.PasskeyCredential{}, err
	}
	return found, converted, nil
}

func (r *RelyingParty) instance(origin string) (*webauthn.WebAuthn, error) {
	rpID, err := resolveRPID(origin, r.rpID)
	if err != nil {
		return nil, err
	}
	return webauthn.New(&webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         r.displayName,
		RPOrigins:             []string{origin},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementPreferred,
			RequireResidentKey: protocol.ResidentKeyNotRequired(),
			UserVerification:   protocol.VerificationPreferred,
		},
	})
}

type wrappedUser struct {
	user  Account
	creds []webauthn.Credential
}

func wrapUser(user Account, existing []domain.PasskeyCredential) *wrappedUser {
	return &wrappedUser{user: user, creds: toLibraryCreds(existing)}
}

func (u *wrappedUser) WebAuthnID() []byte          { return []byte(u.user.ID) }
func (u *wrappedUser) WebAuthnName() string        { return u.user.Login }
func (u *wrappedUser) WebAuthnDisplayName() string { return u.user.Login }
func (u *wrappedUser) WebAuthnCredentials() []webauthn.Credential {
	return u.creds
}

func toLibraryCreds(existing []domain.PasskeyCredential) []webauthn.Credential {
	out := make([]webauthn.Credential, 0, len(existing))
	for _, item := range existing {
		cred, ok := libraryCredential(item)
		if !ok {
			continue
		}
		out = append(out, cred)
	}
	return out
}

func libraryCredential(item domain.PasskeyCredential) (webauthn.Credential, bool) {
	var cred webauthn.Credential
	if len(item.Raw) > 0 {
		if err := json.Unmarshal(item.Raw, &cred); err == nil && len(cred.ID) > 0 {
			return cred, true
		}
	}
	id, err := base64.RawURLEncoding.DecodeString(item.ID)
	if err != nil || len(id) == 0 {
		return webauthn.Credential{}, false
	}
	cred.ID = id
	cred.PublicKey = item.PublicKey
	cred.Authenticator.SignCount = item.SignCount
	return cred, true
}

func toDomain(userID string, cred *webauthn.Credential) (domain.PasskeyCredential, error) {
	if cred == nil {
		return domain.PasskeyCredential{}, domain.ErrUnauthorized
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return domain.PasskeyCredential{}, err
	}
	if err := domain.AssertPasskeyCredentialJSON(raw); err != nil {
		return domain.PasskeyCredential{}, err
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, transport := range cred.Transport {
		transports = append(transports, string(transport))
	}
	return domain.PasskeyCredential{
		ID:         base64.RawURLEncoding.EncodeToString(cred.ID),
		UserID:     userID,
		PublicKey:  cred.PublicKey,
		SignCount:  cred.Authenticator.SignCount,
		Transports: transports,
		AAGUID:     cred.Authenticator.AAGUID,
		Raw:        raw,
	}, nil
}

func marshalJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}
