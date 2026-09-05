package passkey

import "context"

type Client interface {
	FinishLogin(ctx context.Context, challengeID, origin string, credential []byte) (userID string, err error)
	HasCredentials(ctx context.Context, userID string) (bool, error)
}
