package identity

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/job-service/internal/domain"
)

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

func (c *Client) MatchingMode(ctx context.Context, actor domain.Actor) (domain.MatchingMode, error) {
	resp, err := c.api.ListCompanies(ctx, &identityv1.ListCompaniesRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: actor.UserID, CompanyId: actor.CompanyID},
	})
	if err != nil {
		return "", err
	}
	return matchingModeOf(resp.GetCompanies(), actor.CompanyID), nil
}

func matchingModeOf(companies []*identityv1.Company, companyID string) domain.MatchingMode {
	for _, company := range companies {
		if company.GetId() != companyID {
			continue
		}
		if company.GetMatchingMode() == commonv1.MatchingMode_MATCHING_MODE_SMART {
			return domain.MatchingModeSmart
		}
		return domain.MatchingModeStandard
	}
	return domain.MatchingModeStandard
}
