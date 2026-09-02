package port

import (
	"encoding/json"

	"order-fill/services/api-service/internal/domain/identity"
)

// PasskeyCeremony is the WebAuthn relying-party operations. Origin is the
// browser origin of the current request so company subdomains stay valid.
type PasskeyCeremony interface {
	BeginRegistration(origin string, user identity.User, existing []identity.PasskeyCredential) (options json.RawMessage, session json.RawMessage, err error)
	FinishRegistration(origin string, user identity.User, session json.RawMessage, response json.RawMessage) (identity.PasskeyCredential, error)
	BeginLogin(origin string, user identity.User, existing []identity.PasskeyCredential) (options json.RawMessage, session json.RawMessage, err error)
	FinishLogin(origin string, user identity.User, existing []identity.PasskeyCredential, session json.RawMessage, response json.RawMessage) (identity.PasskeyCredential, error)
	BeginDiscoverableLogin(origin string) (options json.RawMessage, session json.RawMessage, err error)
	FinishDiscoverableLogin(origin string, session json.RawMessage, response json.RawMessage, lookup DiscoverablePasskeyUser) (identity.User, identity.PasskeyCredential, error)
}

// DiscoverablePasskeyUser resolves a passkey user handle to the account and
// its stored public credentials.
type DiscoverablePasskeyUser func(rawID []byte, userHandle []byte) (identity.User, []identity.PasskeyCredential, error)
