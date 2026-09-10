package domain

func CanCreatePlatformCompany(actor User) bool {
	return !actor.Disabled() && actor.Role == RolePlatformAdmin
}

func CanManageCompany(actor User, companyID string) bool {
	if actor.Disabled() {
		return false
	}
	if actor.Role == RolePlatformAdmin {
		return true
	}
	return companyActor(actor) && actor.CompanyID != "" && actor.CompanyID == companyID
}

func CanInviteRole(actor User, role Role) bool {
	if actor.Disabled() {
		return false
	}
	switch actor.Role {
	case RolePlatformAdmin:
		return role == RoleCompanyOwner || role == RoleCompanyAdmin || role == RolePurchaser
	case RoleCompanyOwner:
		return role == RoleCompanyAdmin || role == RolePurchaser
	case RoleCompanyAdmin:
		return role == RolePurchaser
	default:
		return false
	}
}

func CanManageUser(actor User, target User) bool {
	if actor.Disabled() {
		return false
	}
	if actor.Role == RolePlatformAdmin {
		return true
	}
	if !companyActor(actor) || actor.CompanyID == "" || actor.CompanyID != target.CompanyID {
		return false
	}
	if target.Role == RolePlatformAdmin {
		return false
	}
	if actor.Role == RoleCompanyAdmin && target.Role == RoleCompanyOwner {
		return false
	}
	return true
}

func BoundToOwnCompany(actor User) bool {
	return companyActor(actor)
}

func companyActor(actor User) bool {
	return actor.Role == RoleCompanyOwner || actor.Role == RoleCompanyAdmin
}
