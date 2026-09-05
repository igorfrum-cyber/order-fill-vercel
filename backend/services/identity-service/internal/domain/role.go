package domain

import "fmt"

type Role string

const (
	RolePlatformAdmin Role = "platform_admin"
	RoleCompanyOwner  Role = "company_owner"
	RoleCompanyAdmin  Role = "company_admin"
	RolePurchaser     Role = "purchaser"
)

func ParseRole(raw string) (Role, error) {
	switch Role(raw) {
	case RolePlatformAdmin, RoleCompanyOwner, RoleCompanyAdmin, RolePurchaser:
		return Role(raw), nil
	default:
		return "", fmt.Errorf("%w: unsupported role", ErrInvalid)
	}
}

func (u User) CanSetMatchingMode() bool {
	return !u.Disabled() && u.Role == RolePlatformAdmin
}
