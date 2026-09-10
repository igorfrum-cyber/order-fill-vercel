package passkey

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	passkeyv1 "order-fill/backend/proto/gen/go/orderfill/passkey/v1"
)

type GRPC struct {
	client passkeyv1.PasskeyServiceClient
}

func Dial(ctx context.Context, addr string) (*GRPC, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GRPC{client: passkeyv1.NewPasskeyServiceClient(conn)}, nil
}

func (c *GRPC) FinishLogin(ctx context.Context, challengeID, origin string, credential []byte) (string, error) {
	resp, err := c.client.FinishLogin(ctx, &passkeyv1.FinishLoginRequest{
		ChallengeId: challengeID, CredentialJson: credential, Origin: origin,
	})
	if err != nil {
		return "", err
	}
	return resp.GetUserId(), nil
}

func (c *GRPC) HasCredentials(ctx context.Context, userID string) (bool, error) {
	resp, err := c.client.ListCredentials(ctx, &passkeyv1.ListCredentialsRequest{ActorUserId: userID})
	if err != nil {
		return false, err
	}
	return len(resp.GetCredentials()) > 0, nil
}
