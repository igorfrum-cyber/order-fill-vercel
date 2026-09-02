package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/authz"
	"order-fill/services/api-service/internal/domain/identity"
	"order-fill/services/api-service/internal/domain/job"
)

type Admin struct {
	store port.IdentityStore
	files port.ObjectStore
	newID port.IDGenerator
	now   port.Clock
}

func NewAdmin(store port.IdentityStore, newID port.IDGenerator, now port.Clock) *Admin {
	return &Admin{store: store, newID: newID, now: now}
}

func (a *Admin) WithFiles(files port.ObjectStore) *Admin {
	a.files = files
	return a
}

func (a *Admin) CreateCompany(ctx context.Context, actor identity.User, name string, loginSlug string) (identity.Company, error) {
	if !authz.CanCreatePlatformCompany(actor) {
		return identity.Company{}, identity.ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return identity.Company{}, fmt.Errorf("%w: company name is required", job.ErrInvalid)
	}
	slug, err := identity.ParseLoginSlug(loginSlug)
	if err != nil {
		return identity.Company{}, fmt.Errorf("%w: %s", job.ErrInvalid, err.Error())
	}
	if err := a.ensureLoginSlugFree(ctx, "", slug); err != nil {
		return identity.Company{}, err
	}
	company := identity.Company{ID: a.newID(), Name: name, LoginSlug: slug, CreatedAt: a.now()}
	if err := a.store.CreateCompany(ctx, company); err != nil {
		return identity.Company{}, err
	}
	return company, nil
}

func (a *Admin) SetCompanyLoginSlug(ctx context.Context, actor identity.User, companyID string, loginSlug string) (identity.Company, error) {
	if boundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !authz.CanManageCompany(actor, companyID) || companyID == "" {
		return identity.Company{}, identity.ErrNotFound
	}
	slug, err := identity.ParseLoginSlug(loginSlug)
	if err != nil {
		return identity.Company{}, fmt.Errorf("%w: %s", job.ErrInvalid, err.Error())
	}
	company, err := a.store.GetCompany(ctx, companyID)
	if err != nil {
		return identity.Company{}, identity.ErrNotFound
	}
	if err := a.ensureLoginSlugFree(ctx, company.ID, slug); err != nil {
		return identity.Company{}, err
	}
	if err := a.store.SetCompanyLoginSlug(ctx, company.ID, slug); err != nil {
		return identity.Company{}, err
	}
	company.LoginSlug = slug
	return company, nil
}

func (a *Admin) UpdateCompany(ctx context.Context, actor identity.User, companyID string, name string, loginSlug string) (identity.Company, error) {
	if boundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !authz.CanManageCompany(actor, companyID) || companyID == "" {
		return identity.Company{}, identity.ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return identity.Company{}, fmt.Errorf("%w: company name is required", job.ErrInvalid)
	}
	slug, err := identity.ParseLoginSlug(loginSlug)
	if err != nil {
		return identity.Company{}, fmt.Errorf("%w: %s", job.ErrInvalid, err.Error())
	}
	company, err := a.store.GetCompany(ctx, companyID)
	if err != nil {
		return identity.Company{}, identity.ErrNotFound
	}
	if err := a.ensureLoginSlugFree(ctx, company.ID, slug); err != nil {
		return identity.Company{}, err
	}
	if err := a.store.SetCompanyProfile(ctx, company.ID, name, slug); err != nil {
		return identity.Company{}, err
	}
	company.Name = name
	company.LoginSlug = slug
	return company, nil
}

func (a *Admin) SetCompanyLogo(ctx context.Context, actor identity.User, companyID string, content []byte) (identity.Company, error) {
	company, err := a.companyForManager(ctx, actor, companyID)
	if err != nil {
		return identity.Company{}, err
	}
	contentType, err := identity.ParseLogo(content)
	if err != nil {
		return identity.Company{}, fmt.Errorf("%w: %s", job.ErrInvalid, err.Error())
	}
	if a.files == nil {
		return identity.Company{}, identity.ErrNotFound
	}
	if err := a.files.Put(ctx, identity.CompanyLogoKey(company.ID), contentType, content); err != nil {
		return identity.Company{}, err
	}
	if err := a.store.SetCompanyLogoType(ctx, company.ID, contentType); err != nil {
		return identity.Company{}, err
	}
	company.LogoContentType = contentType
	return company, nil
}

func (a *Admin) ClearCompanyLogo(ctx context.Context, actor identity.User, companyID string) (identity.Company, error) {
	company, err := a.companyForManager(ctx, actor, companyID)
	if err != nil {
		return identity.Company{}, err
	}
	if err := a.store.SetCompanyLogoType(ctx, company.ID, ""); err != nil {
		return identity.Company{}, err
	}
	company.LogoContentType = ""
	return company, nil
}

func (a *Admin) PublicCompanyLogo(ctx context.Context, slug string) (port.Object, error) {
	company, err := a.PublicCompanyLogin(ctx, slug)
	if err != nil {
		return port.Object{}, err
	}
	if !company.HasLogo() || a.files == nil {
		return port.Object{}, identity.ErrNotFound
	}
	object, err := a.files.Get(ctx, identity.CompanyLogoKey(company.ID))
	if err != nil {
		return port.Object{}, identity.ErrNotFound
	}
	return object, nil
}

func (a *Admin) companyForManager(ctx context.Context, actor identity.User, companyID string) (identity.Company, error) {
	if boundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !authz.CanManageCompany(actor, companyID) || companyID == "" {
		return identity.Company{}, identity.ErrNotFound
	}
	company, err := a.store.GetCompany(ctx, companyID)
	if err != nil {
		return identity.Company{}, identity.ErrNotFound
	}
	return company, nil
}

func (a *Admin) ensureLoginSlugFree(ctx context.Context, companyID string, slug string) error {
	existing, err := a.store.GetCompanyByLoginSlug(ctx, slug)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return nil
		}
		return err
	}
	if existing.ID == companyID {
		return nil
	}
	return identity.ErrConflict
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
	if boundToOwnCompany(actor) {
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
	if boundToOwnCompany(actor) {
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

func boundToOwnCompany(actor identity.User) bool {
	return actor.Role == identity.RoleCompanyOwner || actor.Role == identity.RoleCompanyAdmin
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

func (a *Admin) NeedsTwoFactorNudge(actor identity.User) bool {
	return authz.NeedsTwoFactorNudge(actor)
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
	case identity.RoleCompanyOwner, identity.RoleCompanyAdmin:
		filter.CompanyID = actor.CompanyID
	case identity.RolePurchaser:
		filter.CompanyID = actor.CompanyID
		filter.CreatedBy = actor.ID
	default:
		return nil, identity.ErrUnauthorized
	}
	return u.repository.List(ctx, filter)
}
