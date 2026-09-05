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
		Id:           c.ID,
		Name:         c.Name,
		LoginSlug:    c.LoginSlug,
		HasLogo:      c.HasLogo(),
		MatchingMode: protoMatchingMode(c.MatchingMode),
	}
	if !c.CreatedAt.IsZero() {
		out.CreatedAt = c.CreatedAt.UTC().Format(time.RFC3339)
	}
	if c.DisabledAt != nil {
		out.DisabledAt = c.DisabledAt.UTC().Format(time.RFC3339)
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

func metaActorID(meta *commonv1.RequestMeta) string {
	if meta == nil {
		return ""
	}
	return meta.GetActorUserId()
}
