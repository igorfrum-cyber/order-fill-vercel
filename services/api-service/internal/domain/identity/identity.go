package identity

import "time"

type Role string

const (
	RolePlatformAdmin Role = "platform_admin"
	RoleCompanyAdmin  Role = "company_admin"
	RolePurchaser     Role = "purchaser"
)

// User is an authenticated account. CompanyID is empty for platform admins.
type User struct {
	ID              string
	CompanyID       string
	CompanyName     string
	Login           string
	PasswordHash    string
	Role            Role
	CreatedAt       time.Time
	DisabledAt      *time.Time
	CompanyDisabled bool
}

func (u User) Disabled() bool {
	return u.DisabledAt != nil || u.CompanyDisabled
}

type Company struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	DisabledAt *time.Time
}

func (c Company) Disabled() bool {
	return c.DisabledAt != nil
}
