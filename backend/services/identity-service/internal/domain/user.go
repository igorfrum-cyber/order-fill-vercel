package domain

import "time"

// User is an authenticated account. CompanyID is empty for platform admins.
type User struct {
	ID               string
	CompanyID        string
	CompanyName      string
	CompanyLoginSlug string
	CompanyHasLogo   bool
	Login            string
	PasswordHash     string
	Role             Role
	CreatedAt        time.Time
	DisabledAt       *time.Time
	CompanyDisabled  bool
	TwoFactorEnabled bool
	HasPasskey       bool
	LastSeenAt       *time.Time
}

func (u User) Disabled() bool {
	return u.DisabledAt != nil || u.CompanyDisabled
}
