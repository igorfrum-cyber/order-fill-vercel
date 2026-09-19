package grpcapi

import (
	"time"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/service/auth"
)

func protoUser(u domain.User) *identityv1.User {
	out := &identityv1.User{
		Id:               u.ID,
		Login:            u.Login,
		Role:             string(u.Role),
		CompanyId:        u.CompanyID,
		CompanyName:      u.CompanyName,
		LoginSlug:        u.CompanyLoginSlug,
		HasLogo:          u.CompanyHasLogo,
		TwoFactorEnabled: u.TwoFactorEnabled,
		HasPasskey:       u.HasPasskey,
		Activated:        u.PasswordHash != "",
		IsPrimaryAdmin:   u.IsPrimaryAdmin,
	}
	if u.LastSeenAt != nil {
		out.LastSeenAt = u.LastSeenAt.UTC().Format(time.RFC3339)
	}
	if u.DisabledAt != nil {
		out.DisabledAt = u.DisabledAt.UTC().Format(time.RFC3339)
	}
	return out
}

func protoCompany(c domain.Company) *identityv1.Company {
	out := &identityv1.Company{
		Id:                 c.ID,
		Name:               c.Name,
		LoginSlug:          c.LoginSlug,
		HasLogo:            c.HasLogo(),
		MatchingMode:       protoMatchingMode(c.MatchingMode),
		ChristinaProffMode: protoChristinaProffMode(c.ChristinaProffMode),
		OrderProfile:       protoOrderProfile(c.OrderProfile),
	}
	if !c.CreatedAt.IsZero() {
		out.CreatedAt = c.CreatedAt.UTC().Format(time.RFC3339)
	}
	if c.DisabledAt != nil {
		out.DisabledAt = c.DisabledAt.UTC().Format(time.RFC3339)
	}
	return out
}

func protoPublicCompany(c domain.Company) *identityv1.Company {
	out := protoCompany(c)
	out.OrderProfile = nil
	return out
}

func protoOrderProfile(profile domain.OrderProfile) *identityv1.CompanyOrderProfile {
	out := &identityv1.CompanyOrderProfile{
		LegalName: profile.LegalName, Consignee: profile.Consignee, Address: profile.Address,
		ContactName: profile.ContactName, ContactPhone: profile.ContactPhone, Carrier: profile.Carrier,
		DeliveryPayer: profile.DeliveryPayer, DeliveryDestination: profile.DeliveryDestination,
	}
	for _, item := range profile.BrandTerms {
		out.BrandTerms = append(out.BrandTerms, &identityv1.CompanyBrandTerms{
			Brand: item.Brand, DealerName: item.DealerName, PaymentMethod: item.PaymentMethod,
			PaymentControl: item.PaymentControl, CustomerType: item.CustomerType,
			DiscountBasisPoints: item.DiscountBasisPoints,
			DiscountSet:         item.DiscountSet,
		})
	}
	return out
}

func domainOrderProfile(profile *identityv1.CompanyOrderProfile) domain.OrderProfile {
	if profile == nil {
		return domain.OrderProfile{}
	}
	out := domain.OrderProfile{
		LegalName: profile.GetLegalName(), Consignee: profile.GetConsignee(), Address: profile.GetAddress(),
		ContactName: profile.GetContactName(), ContactPhone: profile.GetContactPhone(), Carrier: profile.GetCarrier(),
		DeliveryPayer: profile.GetDeliveryPayer(), DeliveryDestination: profile.GetDeliveryDestination(),
	}
	for _, item := range profile.GetBrandTerms() {
		out.BrandTerms = append(out.BrandTerms, domain.BrandTerms{
			Brand: item.GetBrand(), DealerName: item.GetDealerName(), PaymentMethod: item.GetPaymentMethod(),
			PaymentControl: item.GetPaymentControl(), CustomerType: item.GetCustomerType(),
			DiscountBasisPoints: item.GetDiscountBasisPoints(),
			DiscountSet:         item.GetDiscountSet(),
		})
	}
	return out
}

func protoSession(s auth.Session) *identityv1.Session {
	return &identityv1.Session{
		Id:     s.ID,
		UserId: s.User.ID,
		Token:  s.RawToken,
	}
}

func protoMatchingMode(mode domain.MatchingMode) commonv1.MatchingMode {
	if mode == domain.MatchingModeSmart {
		return commonv1.MatchingMode_MATCHING_MODE_SMART
	}
	return commonv1.MatchingMode_MATCHING_MODE_STANDARD
}

func domainMatchingMode(mode commonv1.MatchingMode) domain.MatchingMode {
	if mode == commonv1.MatchingMode_MATCHING_MODE_SMART {
		return domain.MatchingModeSmart
	}
	return domain.MatchingModeStandard
}

func protoChristinaProffMode(mode domain.ChristinaProffMode) commonv1.ChristinaProffMode {
	switch mode {
	case domain.ChristinaProffModeFast:
		return commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_FAST
	case domain.ChristinaProffModeCompare:
		return commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE
	default:
		return commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_STANDARD
	}
}

func domainChristinaProffMode(mode commonv1.ChristinaProffMode) domain.ChristinaProffMode {
	switch mode {
	case commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_FAST:
		return domain.ChristinaProffModeFast
	case commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE:
		return domain.ChristinaProffModeCompare
	default:
		return domain.ChristinaProffModeStandard
	}
}

func metaActorID(meta *commonv1.RequestMeta) string {
	if meta == nil {
		return ""
	}
	return meta.GetActorUserId()
}
