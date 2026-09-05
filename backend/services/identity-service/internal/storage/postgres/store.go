package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/identity-service/internal/domain"
)

const userSelect = `u.id, COALESCE(u.company_id, ''), COALESCE(c.name, ''), COALESCE(c.login_slug, ''),
	COALESCE(c.logo_content_type, '') <> '', u.login, u.password_hash, u.role, u.created_at, u.disabled_at,
	c.disabled_at IS NOT NULL, false, false`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) CreateCompany(ctx context.Context, company domain.Company) error {
	created := company.CreatedAt.UTC()
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO companies (id, name, created_at, login_slug, logo_content_type, matching_mode)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		company.ID, company.Name, created, company.LoginSlug, company.LogoContentType, string(company.MatchingMode))
	if err != nil {
		return mapConflict(err)
	}
	return nil
}

func (s *Store) GetCompany(ctx context.Context, id string) (domain.Company, error) {
	return s.scanCompany(s.pool.QueryRow(ctx,
		`SELECT id, name, login_slug, logo_content_type, matching_mode, created_at, disabled_at FROM companies WHERE id = $1`, id))
}

func (s *Store) GetCompanyByLoginSlug(ctx context.Context, slug string) (domain.Company, error) {
	return s.scanCompany(s.pool.QueryRow(ctx,
		`SELECT id, name, login_slug, logo_content_type, matching_mode, created_at, disabled_at FROM companies WHERE login_slug = $1`, slug))
}

func (s *Store) ListCompanies(ctx context.Context) ([]domain.Company, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, login_slug, logo_content_type, matching_mode, created_at, disabled_at FROM companies ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Company, 0)
	for rows.Next() {
		c, err := scanCompanyRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) SetCompanyProfile(ctx context.Context, id, name, slug string, mode domain.MatchingMode) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE companies SET name = $2, login_slug = $3, matching_mode = $4 WHERE id = $1`,
		id, name, slug, string(mode))
	if err != nil {
		return mapConflict(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DisableCompany(ctx context.Context, id string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE companies SET disabled_at = $2 WHERE id = $1`, id, at.UTC())
	if err != nil {
		return fmt.Errorf("disable company: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) CreateUser(ctx context.Context, user domain.User) error {
	created := user.CreatedAt.UTC()
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, company_id, login, password_hash, role, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, nullIfEmpty(user.CompanyID), user.Login, user.PasswordHash, string(user.Role), created)
	if err != nil {
		return mapConflict(err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	return s.scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id WHERE u.id = $1`, id))
}

func (s *Store) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	return s.scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id WHERE u.login = $1`, login))
}

func (s *Store) ListUsers(ctx context.Context, companyID string) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id
		 WHERE COALESCE(u.company_id, '') = $1 ORDER BY u.created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	out := make([]domain.User, 0)
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, rows.Err()
}

func (s *Store) SetPasswordHash(ctx context.Context, userID, hash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ClearPasswordHash(ctx context.Context, userID string) error {
	return s.SetPasswordHash(ctx, userID, "")
}

func (s *Store) DisableUser(ctx context.Context, id string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET disabled_at = $2 WHERE id = $1`, id, at.UTC())
	if err != nil {
		return fmt.Errorf("disable user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, session domain.LoginSession) error {
	created := session.CreatedAt.UTC()
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, id, created_at, user_agent, ip)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		session.TokenHash, session.UserID, session.ExpiresAt.UTC(), session.ID, created, session.UserAgent, session.IP)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *Store) GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	return s.scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 LEFT JOIN companies c ON c.id = u.company_id
		 WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now.UTC()))
}

func (s *Store) ListSessions(ctx context.Context, userID string, now time.Time) ([]domain.LoginSession, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, token_hash, user_id, user_agent, ip, created_at, expires_at
		 FROM sessions WHERE user_id = $1 AND expires_at > $2
		 ORDER BY created_at DESC`, userID, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	out := make([]domain.LoginSession, 0)
	for rows.Next() {
		var item domain.LoginSession
		if err := rows.Scan(&item.ID, &item.TokenHash, &item.UserID, &item.UserAgent, &item.IP, &item.CreatedAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		item.CreatedAt = item.CreatedAt.UTC()
		item.ExpiresAt = item.ExpiresAt.UTC()
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) DeleteSessionsForUser(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (s *Store) CreateInvite(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO invite_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}
	return nil
}

func (s *Store) DeleteInvitesForUser(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM invite_tokens WHERE user_id = $1`, userID)
	return err
}

func (s *Store) ConsumeInvite(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`DELETE FROM invite_tokens WHERE token_hash = $1 AND expires_at > $2 RETURNING user_id`,
		tokenHash, now.UTC()).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) CreateLoginChallenge(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO login_challenges (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt.UTC())
	return err
}

func (s *Store) GetLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM login_challenges WHERE token_hash = $1 AND expires_at > $2`,
		tokenHash, now.UTC()).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) ConsumeLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`DELETE FROM login_challenges WHERE token_hash = $1 AND expires_at > $2 RETURNING user_id`,
		tokenHash, now.UTC()).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) scanUser(row pgx.Row) (domain.User, error) {
	user, err := scanUserRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	return user, err
}

func (s *Store) scanCompany(row pgx.Row) (domain.Company, error) {
	c, err := scanCompanyRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Company{}, domain.ErrNotFound
	}
	return c, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUserRow(row scanner) (domain.User, error) {
	var user domain.User
	var role string
	err := row.Scan(
		&user.ID, &user.CompanyID, &user.CompanyName, &user.CompanyLoginSlug, &user.CompanyHasLogo,
		&user.Login, &user.PasswordHash, &role, &user.CreatedAt, &user.DisabledAt,
		&user.CompanyDisabled, &user.TwoFactorEnabled, &user.HasPasskey,
	)
	if err != nil {
		return domain.User{}, err
	}
	user.Role = domain.Role(role)
	user.CreatedAt = user.CreatedAt.UTC()
	return user, nil
}

func scanCompanyRow(row scanner) (domain.Company, error) {
	var c domain.Company
	var mode string
	if err := row.Scan(&c.ID, &c.Name, &c.LoginSlug, &c.LogoContentType, &mode, &c.CreatedAt, &c.DisabledAt); err != nil {
		return domain.Company{}, err
	}
	c.MatchingMode = domain.ParseMatchingMode(mode)
	c.CreatedAt = c.CreatedAt.UTC()
	return c, nil
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
