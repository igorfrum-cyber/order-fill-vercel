package identity

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/audit-service/internal/domain"
)

type Actor struct {
	UserID    string
	CompanyID string
	Role      string
}

type Client struct {
	api identityv1.IdentityServiceClient
}

func Dial(ctx context.Context, addr string) (*Client, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &Client{api: identityv1.NewIdentityServiceClient(conn)}, nil
}

func (c *Client) Actor(ctx context.Context, userID string) (Actor, error) {
	if userID == "" {
		return Actor{}, domain.ErrUnauthorized
	}
	resp, err := c.api.GetMe(ctx, &identityv1.GetMeRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: userID},
	})
	if err != nil || resp.GetUser() == nil || resp.GetUser().GetId() == "" {
		return Actor{}, domain.ErrUnauthorized
	}
	user := resp.GetUser()
	return Actor{UserID: user.GetId(), CompanyID: user.GetCompanyId(), Role: user.GetRole()}, nil
}
