package identity

import "time"

type Role string

const (
	RolePlatformAdmin Role = "platform_admin"
	RoleCompanyOwner  Role = "company_owner"
	RoleCompanyAdmin  Role = "company_admin"
	RolePurchaser     Role = "purchaser"
)

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
}

func (u User) Disabled() bool {
	return u.DisabledAt != nil || u.CompanyDisabled
}

type Company struct {
	ID              string
	Name            string
	LoginSlug       string
	LogoContentType string
	CreatedAt       time.Time
	DisabledAt      *time.Time
}

func (c Company) Disabled() bool {
	return c.DisabledAt != nil
}

// TOTP is a user's app-based two-factor settings. Secret is the TOTP shared
// secret; recovery codes are stored only as hashes.
type TOTP struct {
	UserID             string
	Secret             string
	EnabledAt          *time.Time
	RecoveryCodeHashes []string
}

func (t TOTP) Enabled() bool {
	return t.EnabledAt != nil
}
