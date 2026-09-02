package port

import (
	"context"
	"time"

	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

type JobListFilter struct {
	CompanyID string
	CreatedBy string
	Limit     int
}

type JobListRow struct {
	Job            job.Job
	CreatedByLogin string
}

type AuditEvent struct {
	ID          string
	At          time.Time
	ActorID     string
	ActorLogin  string
	Action      string
	CompanyID   string
	CompanyName string
	JobID       string
}

const (
	AuditLoginSuccess     = "login_success"
	AuditLogout           = "logout"
	AuditLogoutEverywhere = "logout_everywhere"
	AuditPasswordChanged  = "password_changed"
	AuditInviteCreated    = "invite_created"
	AuditAccessReset      = "access_reset"
	AuditUserDisabled     = "user_disabled"
	AuditCompanyDisabled  = "company_disabled"
	AuditJobView          = "job_view"
	AuditFileDownload     = "file_download"
	AuditArchiveDownload  = "archive_download"
)

func AccessAuditActions() []string {
	return []string{
		AuditPasswordChanged,
		AuditInviteCreated,
		AuditAccessReset,
		AuditUserDisabled,
		AuditCompanyDisabled,
	}
}

func IsAccessAudit(action string) bool {
	for _, item := range AccessAuditActions() {
		if item == action {
			return true
		}
	}
	return false
}

type IdentityStore interface {
	CountUsers(ctx context.Context) (int, error)
	CreateCompany(ctx context.Context, company identity.Company) error
	GetCompany(ctx context.Context, id string) (identity.Company, error)
	GetCompanyByLoginSlug(ctx context.Context, slug string) (identity.Company, error)
	ListCompanies(ctx context.Context) ([]identity.Company, error)
	SetCompanyLoginSlug(ctx context.Context, id string, slug string) error
	SetCompanyProfile(ctx context.Context, id string, name string, slug string) error
	SetCompanyLogoType(ctx context.Context, id string, contentType string) error
	DisableCompany(ctx context.Context, id string, at time.Time) error

	CreateUser(ctx context.Context, user identity.User) error
	GetUserByID(ctx context.Context, id string) (identity.User, error)
	GetUserByLogin(ctx context.Context, login string) (identity.User, error)
	ListUsers(ctx context.Context, companyID string) ([]identity.User, error)
	SetPasswordHash(ctx context.Context, userID string, hash string) error
	ClearPasswordHash(ctx context.Context, userID string) error
	DisableUser(ctx context.Context, id string, at time.Time) error

	CreateSession(ctx context.Context, session identity.LoginSession) error
	GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (identity.User, error)
	ListSessions(ctx context.Context, userID string, now time.Time) ([]identity.LoginSession, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteSessionsForUser(ctx context.Context, userID string) error

	CreateInvite(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error
	DeleteInvitesForUser(ctx context.Context, userID string) error
	ConsumeInvite(ctx context.Context, tokenHash string, now time.Time) (string, error)

	SaveTOTPSetup(ctx context.Context, settings identity.TOTP) error
	GetTOTP(ctx context.Context, userID string) (identity.TOTP, error)
	EnableTOTP(ctx context.Context, userID string, at time.Time) error
	DisableTOTP(ctx context.Context, userID string) error
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error

	CreateLoginChallenge(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error
	GetLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error)
	ConsumeLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error)

	SavePasskey(ctx context.Context, credential identity.PasskeyCredential) error
	ListPasskeys(ctx context.Context, userID string) ([]identity.PasskeyCredential, error)
	GetPasskey(ctx context.Context, id string) (identity.PasskeyCredential, error)
	DeletePasskey(ctx context.Context, userID string, id string) error
	UpdatePasskey(ctx context.Context, credential identity.PasskeyCredential) error

	SavePasskeyChallenge(ctx context.Context, challenge identity.PasskeyChallenge) error
	GetPasskeyChallenge(ctx context.Context, id string, now time.Time) (identity.PasskeyChallenge, error)
	ConsumePasskeyChallenge(ctx context.Context, id string, now time.Time) (identity.PasskeyChallenge, error)

	InsertAudit(ctx context.Context, event AuditEvent) error
	ListAudit(ctx context.Context, limit int, actions []string) ([]AuditEvent, error)
	LastLogins(ctx context.Context, userIDs []string) (map[string]time.Time, error)
}
