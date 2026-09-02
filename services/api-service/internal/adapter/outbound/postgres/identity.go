package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

const userSelect = `u.id, COALESCE(u.company_id, ''), COALESCE(c.name, ''), u.login, u.password_hash, u.role, u.created_at, u.disabled_at,
		COALESCE(c.disabled_at IS NOT NULL, false)`

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateCompany(ctx context.Context, company identity.Company) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO companies (id, name, created_at, disabled_at) VALUES ($1, $2, $3, $4)`,
		company.ID, company.Name, company.CreatedAt.UTC(), company.DisabledAt)
	if err != nil {
		return fmt.Errorf("insert company: %w", mapConflict(err))
	}
	return nil
}

func (r *Repository) GetCompany(ctx context.Context, id string) (identity.Company, error) {
	var company identity.Company
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, created_at, disabled_at FROM companies WHERE id = $1`, id,
	).Scan(&company.ID, &company.Name, &company.CreatedAt, &company.DisabledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Company{}, identity.ErrNotFound
		}
		return identity.Company{}, fmt.Errorf("select company: %w", err)
	}
	company.CreatedAt = company.CreatedAt.UTC()
	return company, nil
}

func (r *Repository) ListCompanies(ctx context.Context) ([]identity.Company, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, created_at, disabled_at FROM companies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()
	companies := make([]identity.Company, 0)
	for rows.Next() {
		var company identity.Company
		if err := rows.Scan(&company.ID, &company.Name, &company.CreatedAt, &company.DisabledAt); err != nil {
			return nil, err
		}
		company.CreatedAt = company.CreatedAt.UTC()
		companies = append(companies, company)
	}
	return companies, rows.Err()
}

func (r *Repository) DisableCompany(ctx context.Context, id string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE companies SET disabled_at = $2 WHERE id = $1`, id, at.UTC())
	if err != nil {
		return fmt.Errorf("disable company: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateUser(ctx context.Context, user identity.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, company_id, login, password_hash, role, created_at, disabled_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, nullIfEmpty(user.CompanyID), user.Login, user.PasswordHash, string(user.Role), user.CreatedAt.UTC(), user.DisabledAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", mapConflict(err))
	}
	return nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (identity.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id WHERE u.id = $1`, id))
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (identity.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id WHERE u.login = $1`, login))
}

func (r *Repository) ListUsers(ctx context.Context, companyID string) ([]identity.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id
		 WHERE ($1 = '' OR u.company_id = $1) ORDER BY u.created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	users := make([]identity.User, 0)
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) SetPasswordHash(ctx context.Context, userID string, hash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) ClearPasswordHash(ctx context.Context, userID string) error {
	return r.SetPasswordHash(ctx, userID, "")
}

func (r *Repository) DisableUser(ctx context.Context, id string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET disabled_at = $2 WHERE id = $1`, id, at.UTC())
	if err != nil {
		return fmt.Errorf("disable user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateSession(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *Repository) GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (identity.User, error) {
	user, err := r.scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 LEFT JOIN companies c ON c.id = u.company_id
		 WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now.UTC()))
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return identity.User{}, identity.ErrUnauthorized
		}
		return identity.User{}, err
	}
	return user, nil
}

func (r *Repository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *Repository) DeleteSessionsForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) CreateInvite(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invite_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}
	return nil
}

func (r *Repository) DeleteInvitesForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM invite_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) ConsumeInvite(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`DELETE FROM invite_tokens WHERE token_hash = $1 AND expires_at > $2 RETURNING user_id`,
		tokenHash, now.UTC()).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", identity.ErrUnauthorized
		}
		return "", fmt.Errorf("consume invite: %w", err)
	}
	return userID, nil
}

func (r *Repository) InsertAudit(ctx context.Context, event port.AuditEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_events (id, at, actor_id, action, company_id, job_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.At.UTC(), event.ActorID, event.Action, nullIfEmpty(event.CompanyID), nullIfEmpty(event.JobID))
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

func (r *Repository) ListAudit(ctx context.Context, limit int) ([]port.AuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, at, actor_id, action, COALESCE(company_id, ''), COALESCE(job_id, '')
		 FROM audit_events ORDER BY at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	events := make([]port.AuditEvent, 0)
	for rows.Next() {
		var event port.AuditEvent
		if err := rows.Scan(&event.ID, &event.At, &event.ActorID, &event.Action, &event.CompanyID, &event.JobID); err != nil {
			return nil, err
		}
		event.At = event.At.UTC()
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) scanUser(row pgx.Row) (identity.User, error) {
	user, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.User{}, identity.ErrNotFound
		}
		return identity.User{}, fmt.Errorf("select user: %w", err)
	}
	return user, nil
}

func scanUserRow(row pgx.Row) (identity.User, error) {
	var (
		user identity.User
		role string
	)
	err := row.Scan(&user.ID, &user.CompanyID, &user.CompanyName, &user.Login, &user.PasswordHash, &role, &user.CreatedAt, &user.DisabledAt, &user.CompanyDisabled)
	if err != nil {
		return identity.User{}, err
	}
	user.Role = identity.Role(role)
	user.CreatedAt = user.CreatedAt.UTC()
	return user, nil
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return identity.ErrConflict
	}
	return err
}
