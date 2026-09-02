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
	case identity.RoleCompanyAdmin:
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
	return actor.Role == identity.RoleCompanyAdmin && actor.CompanyID != "" && actor.CompanyID == companyID
}

func CanInviteRole(actor identity.User, role identity.Role) bool {
	if actor.Disabled() {
		return false
	}
	switch actor.Role {
	case identity.RolePlatformAdmin:
		return role == identity.RoleCompanyAdmin
	case identity.RoleCompanyAdmin:
		return role == identity.RoleCompanyAdmin || role == identity.RolePurchaser
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
	case identity.RolePurchaser, identity.RoleCompanyAdmin:
		return actor.CompanyID != ""
	default:
		return false
	}
}
