package twofa

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	twofav1 "order-fill/backend/proto/gen/go/orderfill/twofa/v1"
)

type GRPC struct {
	client twofav1.TwoFAServiceClient
}

func Dial(ctx context.Context, addr string) (*GRPC, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GRPC{client: twofav1.NewTwoFAServiceClient(conn)}, nil
}

func (c *GRPC) IsEnabled(ctx context.Context, userID string) (bool, error) {
	resp, err := c.client.IsEnabled(ctx, &twofav1.IsEnabledRequest{UserId: userID})
	if err != nil {
		return false, err
	}
	return resp.GetEnabled(), nil
}

func (c *GRPC) Verify(ctx context.Context, userID, code string) error {
	resp, err := c.client.Verify(ctx, &twofav1.VerifyRequest{UserId: userID, Code: code})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return errDisabled
	}
	return nil
}

var errDisabled = errVerifyFailed{}

type errVerifyFailed struct{}

func (errVerifyFailed) Error() string { return "twofa verify failed" }
