package authz

import (
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

func CanAccessJob(actor identity.User, entity job.Job) bool {
	if actor.Disabled() {
		return false
	}
	switch actor.Role {
	case identity.RolePlatformAdmin:
		return true
	case identity.RoleCompanyOwner, identity.RoleCompanyAdmin:
		return entity.CompanyID != "" && entity.CompanyID == actor.CompanyID
	case identity.RolePurchaser:
		return entity.CompanyID != "" && entity.CompanyID == actor.CompanyID && entity.CreatedBy == actor.ID
	default:
		return false
	}
}

func CanManageCompany(actor identity.User, companyID string) bool {
	if actor.Disabled() {
		return false
	}
	if actor.Role == identity.RolePlatformAdmin {
		return true
	}
	return companyActor(actor) && actor.CompanyID != "" && actor.CompanyID == companyID
}

func CanInviteRole(actor identity.User, role identity.Role) bool {
	if actor.Disabled() {
		return false
	}
	switch actor.Role {
	case identity.RolePlatformAdmin:
		return role == identity.RoleCompanyOwner || role == identity.RoleCompanyAdmin || role == identity.RolePurchaser
	case identity.RoleCompanyOwner:
		return role == identity.RoleCompanyAdmin || role == identity.RolePurchaser
	case identity.RoleCompanyAdmin:
		return role == identity.RolePurchaser
	default:
		return false
	}
}

func CanCreatePlatformCompany(actor identity.User) bool {
	return !actor.Disabled() && actor.Role == identity.RolePlatformAdmin
}

func CanCreateJob(actor identity.User) bool {
	if actor.Disabled() {
		return false
	}
	switch actor.Role {
	case identity.RolePurchaser, identity.RoleCompanyAdmin, identity.RoleCompanyOwner:
		return actor.CompanyID != ""
	default:
		return false
	}
}

func CanManageUser(actor identity.User, target identity.User) bool {
	if actor.Disabled() {
		return false
	}
	if actor.Role == identity.RolePlatformAdmin {
		return true
	}
	if !companyActor(actor) || actor.CompanyID == "" || actor.CompanyID != target.CompanyID {
		return false
	}
	if target.Role == identity.RolePlatformAdmin {
		return false
	}
	if actor.Role == identity.RoleCompanyAdmin && target.Role == identity.RoleCompanyOwner {
		return false
	}
	return true
}

func companyActor(actor identity.User) bool {
	return actor.Role == identity.RoleCompanyOwner || actor.Role == identity.RoleCompanyAdmin
}

// NeedsTwoFactorNudge is true until the person has a passkey or an app code.
// Every role can set this up; a passkey is enough for daily work.
func NeedsTwoFactorNudge(actor identity.User) bool {
	if actor.Disabled() || actor.TwoFactorEnabled || actor.HasPasskey {
		return false
	}
	return true
}
