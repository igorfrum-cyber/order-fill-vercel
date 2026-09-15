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

func (c *Client) Actor(ctx context.Context, userID string) (domain.Actor, error) {
	if userID == "" {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	resp, err := c.api.GetMe(ctx, &identityv1.GetMeRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: userID},
	})
	if err != nil || resp.GetUser() == nil || resp.GetUser().GetId() == "" {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	user := resp.GetUser()
	return domain.Actor{
		UserID:    user.GetId(),
		CompanyID: user.GetCompanyId(),
		Role:      domain.RoleName(user.GetRole()),
	}, nil
}

func (c *Client) Config(ctx context.Context, actor domain.Actor) (domain.CompanyConfig, error) {
	resp, err := c.api.ListCompanies(ctx, &identityv1.ListCompaniesRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: actor.UserID, CompanyId: actor.CompanyID},
	})
	if err != nil {
		return domain.CompanyConfig{}, err
	}
	return companyConfigOf(resp.GetCompanies(), actor.CompanyID), nil
}

func companyConfigOf(companies []*identityv1.Company, companyID string) domain.CompanyConfig {
	for _, company := range companies {
		if company.GetId() != companyID {
			continue
		}
		config := domain.CompanyConfig{MatchingMode: domain.MatchingModeStandard}
		if company.GetMatchingMode() == commonv1.MatchingMode_MATCHING_MODE_SMART {
			config.MatchingMode = domain.MatchingModeSmart
		}
		profile := company.GetOrderProfile()
		if profile != nil {
			config.OrderProfile = domain.OrderProfile{
				LegalName: profile.GetLegalName(), Consignee: profile.GetConsignee(), Address: profile.GetAddress(),
				ContactName: profile.GetContactName(), ContactPhone: profile.GetContactPhone(), Carrier: profile.GetCarrier(),
				DeliveryPayer: profile.GetDeliveryPayer(), DeliveryDestination: profile.GetDeliveryDestination(),
			}
			for _, item := range profile.GetBrandTerms() {
				config.OrderProfile.BrandTerms = append(config.OrderProfile.BrandTerms, domain.BrandTerms{
					Brand: item.GetBrand(), DealerName: item.GetDealerName(), PaymentMethod: item.GetPaymentMethod(),
					PaymentControl: item.GetPaymentControl(), CustomerType: item.GetCustomerType(),
					DiscountBasisPoints: item.GetDiscountBasisPoints(),
					DiscountSet:         item.GetDiscountSet(),
				})
			}
		}
		return config
	}
	return domain.CompanyConfig{MatchingMode: domain.MatchingModeStandard}
}
