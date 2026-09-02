package passkeys

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

// RelyingParty is a WebAuthn adapter that accepts platform authenticators,
// password managers, and security keys. Authenticator attachment is left
// unset so the browser can offer every compatible app.
type RelyingParty struct {
	displayName string
	rpID        string
}

func New(displayName string, rpID string) *RelyingParty {
	name := displayName
	if name == "" {
		name = defaultDisplayName
	}
	return &RelyingParty{displayName: name, rpID: rpID}
}

func (r *RelyingParty) BeginRegistration(origin string, user identity.User, existing []identity.PasskeyCredential) (json.RawMessage, json.RawMessage, error) {
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

func (r *RelyingParty) FinishRegistration(origin string, user identity.User, session json.RawMessage, response json.RawMessage) (identity.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return identity.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	cred, err := wa.CreateCredential(wrapUser(user, nil), data, parsed)
	if err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	return toDomain(user.ID, cred)
}

func (r *RelyingParty) BeginLogin(origin string, user identity.User, existing []identity.PasskeyCredential) (json.RawMessage, json.RawMessage, error) {
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

func (r *RelyingParty) FinishLogin(origin string, user identity.User, existing []identity.PasskeyCredential, session json.RawMessage, response json.RawMessage) (identity.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return identity.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	cred, err := wa.ValidateLogin(wrapUser(user, existing), data, parsed)
	if err != nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	return toDomain(user.ID, cred)
}

func (r *RelyingParty) BeginDiscoverableLogin(origin string) (json.RawMessage, json.RawMessage, error) {
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

func (r *RelyingParty) FinishDiscoverableLogin(origin string, session json.RawMessage, response json.RawMessage, lookup port.DiscoverablePasskeyUser) (identity.User, identity.PasskeyCredential, error) {
	wa, err := r.instance(origin)
	if err != nil {
		return identity.User{}, identity.PasskeyCredential{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(session, &data); err != nil {
		return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	var found identity.User
	user, cred, err := wa.ValidatePasskeyLogin(func(rawID []byte, userHandle []byte) (webauthn.User, error) {
		account, creds, lookupErr := lookup(rawID, userHandle)
		if lookupErr != nil {
			return nil, lookupErr
		}
		found = account
		return wrapUser(account, creds), nil
	}, data, parsed)
	if err != nil {
		return identity.User{}, identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	if wrapped, ok := user.(*wrappedUser); ok {
		found = wrapped.user
	}
	converted, err := toDomain(found.ID, cred)
	if err != nil {
		return identity.User{}, identity.PasskeyCredential{}, err
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
		EncodeUserIDAsString: true,
	})
}

type wrappedUser struct {
	user  identity.User
	creds []webauthn.Credential
}

func wrapUser(user identity.User, existing []identity.PasskeyCredential) *wrappedUser {
	return &wrappedUser{user: user, creds: toLibraryCreds(existing)}
}

func (u *wrappedUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u *wrappedUser) WebAuthnName() string {
	return u.user.Login
}

func (u *wrappedUser) WebAuthnDisplayName() string {
	return u.user.Login
}

func (u *wrappedUser) WebAuthnCredentials() []webauthn.Credential {
	return u.creds
}

func toLibraryCreds(existing []identity.PasskeyCredential) []webauthn.Credential {
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

func libraryCredential(item identity.PasskeyCredential) (webauthn.Credential, bool) {
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

func toDomain(userID string, cred *webauthn.Credential) (identity.PasskeyCredential, error) {
	if cred == nil {
		return identity.PasskeyCredential{}, identity.ErrUnauthorized
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return identity.PasskeyCredential{}, err
	}
	if err := identity.AssertPasskeyCredentialJSON(raw); err != nil {
		return identity.PasskeyCredential{}, err
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, transport := range cred.Transport {
		transports = append(transports, string(transport))
	}
	return identity.PasskeyCredential{
		ID:         base64.RawURLEncoding.EncodeToString(cred.ID),
		UserID:     userID,
		PublicKey:  cred.PublicKey,
		SignCount:  cred.Authenticator.SignCount,
		Transports: transports,
		AAGUID:     cred.Authenticator.AAGUID,
		Raw:        raw,
	}, nil
}

func marshalJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}
