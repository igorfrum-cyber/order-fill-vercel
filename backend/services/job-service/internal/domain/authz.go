package domain

type RoleName string

const (
	RolePlatformAdmin RoleName = "platform_admin"
	RoleCompanyOwner  RoleName = "company_owner"
	RoleCompanyAdmin  RoleName = "company_admin"
	RolePurchaser     RoleName = "purchaser"
)

type Actor struct {
	UserID    string
	CompanyID string
	Role      RoleName
	Disabled  bool
}

func CanCreateJob(actor Actor) bool {
	if actor.Disabled {
		return false
	}
	switch actor.Role {
	case RolePurchaser, RoleCompanyAdmin, RoleCompanyOwner:
		return actor.CompanyID != ""
	default:
		return false
	}
}

func CanAccessJob(actor Actor, entity Job) bool {
	if actor.Disabled {
		return false
	}
	switch actor.Role {
	case RolePlatformAdmin:
		return true
	case RoleCompanyOwner, RoleCompanyAdmin:
		return entity.CompanyID != "" && entity.CompanyID == actor.CompanyID
	case RolePurchaser:
		return entity.CompanyID != "" && entity.CompanyID == actor.CompanyID && entity.OwnerUserID == actor.UserID
	default:
		return false
	}
}
