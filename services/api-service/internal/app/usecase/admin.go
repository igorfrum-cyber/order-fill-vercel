package usecase

import (
	"context"
	"fmt"
	"strings"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/authz"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

type Admin struct {
	store port.IdentityStore
	newID port.IDGenerator
	now   port.Clock
}

func NewAdmin(store port.IdentityStore, newID port.IDGenerator, now port.Clock) *Admin {
	return &Admin{store: store, newID: newID, now: now}
}

func (a *Admin) CreateCompany(ctx context.Context, actor identity.User, name string) (identity.Company, error) {
	if !authz.CanCreatePlatformCompany(actor) {
		return identity.Company{}, identity.ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return identity.Company{}, fmt.Errorf("%w: company name is required", job.ErrInvalid)
	}
	company := identity.Company{ID: a.newID(), Name: name, CreatedAt: a.now()}
	slug, err := a.uniqueLoginSlug(ctx, name, company.ID)
	if err != nil {
		return identity.Company{}, err
	}
	company.LoginSlug = slug
	if err := a.store.CreateCompany(ctx, company); err != nil {
		return identity.Company{}, err
	}
	return company, nil
}

func (a *Admin) uniqueLoginSlug(ctx context.Context, name string, companyID string) (string, error) {
	companies, err := a.store.ListCompanies(ctx)
	if err != nil {
		return "", err
	}
	taken := make(map[string]struct{}, len(companies))
	for _, company := range companies {
		if company.LoginSlug != "" {
			taken[company.LoginSlug] = struct{}{}
		}
	}
	return identity.UniqueLoginSlug(name, companyID, func(slug string) bool {
		_, ok := taken[slug]
		return ok
	}), nil
}

func (a *Admin) PublicCompanyLogin(ctx context.Context, slug string) (identity.Company, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return identity.Company{}, identity.ErrNotFound
	}
	company, err := a.store.GetCompanyByLoginSlug(ctx, slug)
	if err != nil {
		return identity.Company{}, identity.ErrNotFound
	}
	if company.Disabled() {
		return identity.Company{}, identity.ErrNotFound
	}
	return company, nil
}

func (a *Admin) ListCompanies(ctx context.Context, actor identity.User) ([]identity.Company, error) {
	if !authz.CanCreatePlatformCompany(actor) {
		return nil, identity.ErrNotFound
	}
	return a.store.ListCompanies(ctx)
}

func (a *Admin) DisableCompany(ctx context.Context, actor identity.User, companyID string) error {
	if !authz.CanCreatePlatformCompany(actor) {
		return identity.ErrNotFound
	}
	if err := a.store.DisableCompany(ctx, companyID, a.now()); err != nil {
		return err
	}
	a.RecordAudit(ctx, actor, port.AuditCompanyDisabled, companyID, "")
	return nil
}

func (a *Admin) CreateUser(ctx context.Context, actor identity.User, companyID string, login string, role identity.Role) (identity.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return identity.User{}, "", fmt.Errorf("%w: login is required", job.ErrInvalid)
	}
	if actor.Role == identity.RoleCompanyAdmin {
		companyID = actor.CompanyID
	}
	if !authz.CanManageCompany(actor, companyID) || companyID == "" {
		return identity.User{}, "", identity.ErrNotFound
	}
	if !authz.CanInviteRole(actor, role) {
		return identity.User{}, "", fmt.Errorf("%w: unsupported role", job.ErrInvalid)
	}
	if _, err := a.store.GetCompany(ctx, companyID); err != nil {
		return identity.User{}, "", identity.ErrNotFound
	}
	user := identity.User{
		ID:        a.newID(),
		CompanyID: companyID,
		Login:     login,
		Role:      role,
		CreatedAt: a.now(),
	}
	if err := a.store.CreateUser(ctx, user); err != nil {
		return identity.User{}, "", err
	}
	raw, err := identity.NewSecret()
	if err != nil {
		return identity.User{}, "", err
	}
	if err := a.store.CreateInvite(ctx, identity.HashSecret(raw), user.ID, a.now().Add(inviteTTL)); err != nil {
		return identity.User{}, "", err
	}
	a.RecordAudit(ctx, actor, port.AuditInviteCreated, companyID, "")
	return user, raw, nil
}

func (a *Admin) ListUsers(ctx context.Context, actor identity.User, companyID string) ([]identity.User, error) {
	if actor.Role == identity.RoleCompanyAdmin {
		companyID = actor.CompanyID
	}
	if !authz.CanManageCompany(actor, companyID) {
		return nil, identity.ErrNotFound
	}
	return a.store.ListUsers(ctx, companyID)
}

func (a *Admin) DisableUser(ctx context.Context, actor identity.User, userID string) error {
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil {
		return identity.ErrNotFound
	}
	if !canManageUser(actor, user) {
		return identity.ErrNotFound
	}
	if err := a.store.DisableUser(ctx, userID, a.now()); err != nil {
		return err
	}
	a.RecordAudit(ctx, actor, port.AuditUserDisabled, user.CompanyID, "")
	return nil
}

func (a *Admin) ListAudit(ctx context.Context, actor identity.User) ([]port.AuditEvent, error) {
	if actor.Role != identity.RolePlatformAdmin {
		return nil, identity.ErrNotFound
	}
	return a.store.ListAudit(ctx, 100)
}

func (a *Admin) RecordAudit(ctx context.Context, actor identity.User, action string, companyID string, jobID string) {
	if a.store == nil {
		return
	}
	_ = a.store.InsertAudit(ctx, port.AuditEvent{
		ID:        a.newID(),
		At:        a.now().UTC(),
		ActorID:   actor.ID,
		Action:    action,
		CompanyID: companyID,
		JobID:     jobID,
	})
}

type ListJobs struct {
	repository interface {
		List(ctx context.Context, filter port.JobListFilter) ([]port.JobListRow, error)
	}
}

func NewListJobs(repository interface {
	List(ctx context.Context, filter port.JobListFilter) ([]port.JobListRow, error)
}) *ListJobs {
	return &ListJobs{repository: repository}
}

func (u *ListJobs) Execute(ctx context.Context, actor identity.User, companyID string) ([]port.JobListRow, error) {
	filter := port.JobListFilter{Limit: 200}
	switch actor.Role {
	case identity.RolePlatformAdmin:
		filter.CompanyID = strings.TrimSpace(companyID)
	case identity.RoleCompanyAdmin:
		filter.CompanyID = actor.CompanyID
	case identity.RolePurchaser:
		filter.CompanyID = actor.CompanyID
		filter.CreatedBy = actor.ID
	default:
		return nil, identity.ErrUnauthorized
	}
	return u.repository.List(ctx, filter)
}
