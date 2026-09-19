package companies

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
	"order-fill/backend/services/identity-service/internal/secret"
)

type Store interface {
	CreateCompany(ctx context.Context, company domain.Company) error
	GetCompany(ctx context.Context, id string) (domain.Company, error)
	GetCompanyByLoginSlug(ctx context.Context, slug string) (domain.Company, error)
	ListCompanies(ctx context.Context) ([]domain.Company, error)
	SetCompanyProfile(ctx context.Context, id, name, slug string, mode domain.MatchingMode, christina domain.ChristinaProffMode) error
	SetCompanyOrderProfile(ctx context.Context, id string, profile domain.OrderProfile) error
	DisableCompany(ctx context.Context, id string, at time.Time) error
}

func (c *Companies) UpdateOrderProfile(ctx context.Context, actor domain.User, companyID string, profile domain.OrderProfile) (domain.Company, error) {
	if domain.BoundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !domain.CanManageCompany(actor, companyID) || companyID == "" {
		return domain.Company{}, domain.ErrNotFound
	}
	company, err := c.store.GetCompany(ctx, companyID)
	if err != nil {
		return domain.Company{}, domain.ErrNotFound
	}
	profile, err = domain.NormalizeOrderProfile(profile)
	if err != nil {
		return domain.Company{}, err
	}
	if err := c.store.SetCompanyOrderProfile(ctx, company.ID, profile); err != nil {
		return domain.Company{}, err
	}
	company.OrderProfile = profile
	return company, nil
}

type Companies struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Companies {
	if now == nil {
		now = time.Now
	}
	return &Companies{store: store, now: now}
}

func (c *Companies) Create(ctx context.Context, actor domain.User, name, loginSlug string, mode domain.MatchingMode, christina domain.ChristinaProffMode) (domain.Company, error) {
	if !domain.CanCreatePlatformCompany(actor) {
		return domain.Company{}, domain.ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Company{}, fmt.Errorf("%w: company name is required", domain.ErrInvalid)
	}
	slug, err := domain.ParseLoginSlug(loginSlug)
	if err != nil {
		return domain.Company{}, err
	}
	if mode == "" {
		mode = domain.MatchingModeStandard
	}
	if mode == domain.MatchingModeSmart && !actor.CanSetMatchingMode() {
		mode = domain.MatchingModeStandard
	}
	if christina == "" {
		christina = domain.ChristinaProffModeStandard
	}
	if christina != domain.ChristinaProffModeStandard && !actor.CanSetChristinaProffMode() {
		christina = domain.ChristinaProffModeStandard
	}
	id, err := secret.NewSecret()
	if err != nil {
		return domain.Company{}, err
	}
	company := domain.Company{
		ID:                 id,
		Name:               name,
		LoginSlug:          slug,
		MatchingMode:       mode,
		ChristinaProffMode: christina,
		CreatedAt:          c.now(),
	}
	if err := c.store.CreateCompany(ctx, company); err != nil {
		return domain.Company{}, err
	}
	return company, nil
}

func (c *Companies) List(ctx context.Context, actor domain.User) ([]domain.Company, error) {
	if domain.CanCreatePlatformCompany(actor) {
		return c.store.ListCompanies(ctx)
	}
	if actor.CompanyID == "" {
		return nil, domain.ErrNotFound
	}
	company, err := c.store.GetCompany(ctx, actor.CompanyID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return []domain.Company{company}, nil
}

func (c *Companies) Update(ctx context.Context, actor domain.User, companyID, name, loginSlug string, mode domain.MatchingMode, christina domain.ChristinaProffMode) (domain.Company, error) {
	if domain.BoundToOwnCompany(actor) {
		companyID = actor.CompanyID
	}
	if !domain.CanManageCompany(actor, companyID) || companyID == "" {
		return domain.Company{}, domain.ErrNotFound
	}
	company, err := c.store.GetCompany(ctx, companyID)
	if err != nil {
		return domain.Company{}, domain.ErrNotFound
	}
	name = cmp.Or(strings.TrimSpace(name), company.Name)
	if name == "" {
		return domain.Company{}, fmt.Errorf("%w: company name is required", domain.ErrInvalid)
	}
	slug := company.LoginSlug
	if strings.TrimSpace(loginSlug) != "" {
		parsed, err := domain.ParseLoginSlug(loginSlug)
		if err != nil {
			return domain.Company{}, err
		}
		slug = parsed
	}
	if mode == "" || !actor.CanSetMatchingMode() {
		mode = company.MatchingMode
	}
	if christina == "" || !actor.CanSetChristinaProffMode() {
		christina = company.ChristinaProffMode
	}
	if err := c.store.SetCompanyProfile(ctx, company.ID, name, slug, mode, christina); err != nil {
		return domain.Company{}, err
	}
	company.Name = name
	company.LoginSlug = slug
	company.MatchingMode = mode
	company.ChristinaProffMode = christina
	return company, nil
}

func (c *Companies) PublicBySlug(ctx context.Context, slug string) (domain.Company, error) {
	company, err := c.store.GetCompanyByLoginSlug(ctx, slug)
	if err != nil || company.Disabled() {
		return domain.Company{}, domain.ErrNotFound
	}
	return company, nil
}

func (c *Companies) Disable(ctx context.Context, actor domain.User, companyID string) error {
	if !domain.CanCreatePlatformCompany(actor) {
		return domain.ErrNotFound
	}
	return c.store.DisableCompany(ctx, companyID, c.now())
}
