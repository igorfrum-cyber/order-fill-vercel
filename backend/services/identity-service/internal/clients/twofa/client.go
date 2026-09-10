package twofa

import "context"

type Client interface {
	IsEnabled(ctx context.Context, userID string) (bool, error)
	Verify(ctx context.Context, userID, code string) error
}
