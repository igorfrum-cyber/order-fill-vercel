package memory

import (
	"context"
	"maps"
	"sync"
	"time"

	"order-fill/backend/services/identity-service/internal/domain"
)

type Store struct {
	mu         sync.Mutex
	users      map[string]domain.User
	logins     map[string]string
	companies  map[string]domain.Company
	slugs      map[string]string
	sessions   map[string]domain.LoginSession
	invites    map[string]invite
	challenges map[string]challenge
	twoFA      map[string]bool
}

type invite struct {
	UserID    string
	ExpiresAt time.Time
}

type challenge struct {
	UserID    string
	ExpiresAt time.Time
}

func NewStore() *Store {
	return &Store{
		users:      map[string]domain.User{},
		logins:     map[string]string{},
		companies:  map[string]domain.Company{},
		slugs:      map[string]string{},
		sessions:   map[string]domain.LoginSession{},
		invites:    map[string]invite{},
		challenges: map[string]challenge{},
		twoFA:      map[string]bool{},
	}
}

func (s *Store) CreateCompany(_ context.Context, company domain.Company) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.slugs[company.LoginSlug]; ok {
		return domain.ErrConflict
	}
	s.companies[company.ID] = company
	s.slugs[company.LoginSlug] = company.ID
	return nil
}

func (s *Store) GetCompany(_ context.Context, id string) (domain.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.companies[id]
	if !ok {
		return domain.Company{}, domain.ErrNotFound
	}
	return c, nil
}

func (s *Store) GetCompanyByLoginSlug(_ context.Context, slug string) (domain.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.slugs[slug]
	if !ok {
		return domain.Company{}, domain.ErrNotFound
	}
	return s.companies[id], nil
}

func (s *Store) ListCompanies(_ context.Context) ([]domain.Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Company, 0, len(s.companies))
	for _, c := range s.companies {
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) SetCompanyProfile(_ context.Context, id, name, slug string, mode domain.MatchingMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.companies[id]
	if !ok {
		return domain.ErrNotFound
	}
	if other, ok := s.slugs[slug]; ok && other != id {
		return domain.ErrConflict
	}
	delete(s.slugs, c.LoginSlug)
	c.Name = name
	c.LoginSlug = slug
	c.MatchingMode = mode
	s.companies[id] = c
	s.slugs[slug] = id
	return nil
}

func (s *Store) DisableCompany(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.companies[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.DisabledAt = &at
	s.companies[id] = c
	return nil
}

func (s *Store) CountUsers(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.users), nil
}

func (s *Store) CreateUser(_ context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.logins[user.Login]; ok {
		return domain.ErrConflict
	}
	s.users[user.ID] = user
	s.logins[user.Login] = user.ID
	return nil
}

func (s *Store) GetUserByID(_ context.Context, id string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userLocked(id)
}

func (s *Store) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.logins[login]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return s.userLocked(id)
}

func (s *Store) ListUsers(_ context.Context, companyID string) ([]domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.User, 0)
	for _, u := range s.users {
		if u.CompanyID == companyID {
			got, _ := s.userLocked(u.ID)
			out = append(out, got)
		}
	}
	return out, nil
}

func (s *Store) SetPasswordHash(_ context.Context, userID, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return domain.ErrNotFound
	}
	u.PasswordHash = hash
	s.users[userID] = u
	return nil
}

func (s *Store) ClearPasswordHash(ctx context.Context, userID string) error {
	return s.SetPasswordHash(ctx, userID, "")
}

func (s *Store) DisableUser(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.DisabledAt = &at
	s.users[id] = u
	return nil
}

func (s *Store) CreateSession(_ context.Context, session domain.LoginSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *Store) GetSessionUser(_ context.Context, tokenHash string, now time.Time) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[tokenHash]
	if !ok || !sess.ExpiresAt.After(now) {
		return domain.User{}, domain.ErrNotFound
	}
	return s.userLocked(sess.UserID)
}

func (s *Store) ListSessions(_ context.Context, userID string, now time.Time) ([]domain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.LoginSession, 0)
	for _, sess := range s.sessions {
		if sess.UserID == userID && sess.ExpiresAt.After(now) {
			out = append(out, sess)
		}
	}
	return out, nil
}

func (s *Store) DeleteSession(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tokenHash)
	return nil
}

func (s *Store) DeleteSessionsForUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	maps.DeleteFunc(s.sessions, func(_ string, sess domain.LoginSession) bool {
		return sess.UserID == userID
	})
	return nil
}

func (s *Store) CreateInvite(_ context.Context, tokenHash, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invites[tokenHash] = invite{UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (s *Store) DeleteInvitesForUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	maps.DeleteFunc(s.invites, func(_ string, inv invite) bool {
		return inv.UserID == userID
	})
	return nil
}

func (s *Store) ConsumeInvite(_ context.Context, tokenHash string, now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invites[tokenHash]
	if !ok || !inv.ExpiresAt.After(now) {
		return "", domain.ErrUnauthorized
	}
	delete(s.invites, tokenHash)
	return inv.UserID, nil
}

func (s *Store) CreateLoginChallenge(_ context.Context, tokenHash, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[tokenHash] = challenge{UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (s *Store) GetLoginChallenge(_ context.Context, tokenHash string, now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.challenges[tokenHash]
	if !ok || !ch.ExpiresAt.After(now) {
		return "", domain.ErrUnauthorized
	}
	return ch.UserID, nil
}

func (s *Store) ConsumeLoginChallenge(_ context.Context, tokenHash string, now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.challenges[tokenHash]
	if !ok || !ch.ExpiresAt.After(now) {
		return "", domain.ErrUnauthorized
	}
	delete(s.challenges, tokenHash)
	return ch.UserID, nil
}

func (s *Store) SetTwoFA(_ context.Context, userID string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.twoFA[userID] = enabled
}

func (s *Store) TwoFAEnabled(_ context.Context, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.twoFA[userID]
}

func (s *Store) SessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func (s *Store) userLocked(id string) (domain.User, error) {
	u, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if u.CompanyID != "" {
		if c, ok := s.companies[u.CompanyID]; ok {
			u.CompanyName = c.Name
			u.CompanyLoginSlug = c.LoginSlug
			u.CompanyHasLogo = c.HasLogo()
			u.CompanyDisabled = c.Disabled()
		}
	}
	u.TwoFactorEnabled = s.twoFA[u.ID]
	return u, nil
}
