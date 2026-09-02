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

const userSelect = `u.id, COALESCE(u.company_id, ''), COALESCE(c.name, ''), COALESCE(c.login_slug, ''), COALESCE(c.logo_content_type, '') <> '', u.login, u.password_hash, u.role, u.created_at, u.disabled_at,
		COALESCE(c.disabled_at IS NOT NULL, false), COALESCE(t.enabled_at IS NOT NULL, false),
		EXISTS (SELECT 1 FROM passkey_credentials p WHERE p.user_id = u.id)`

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateCompany(ctx context.Context, company identity.Company) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO companies (id, name, created_at, disabled_at, login_slug) VALUES ($1, $2, $3, $4, $5)`,
		company.ID, company.Name, company.CreatedAt.UTC(), company.DisabledAt, nullIfEmpty(company.LoginSlug))
	if err != nil {
		return fmt.Errorf("insert company: %w", mapConflict(err))
	}
	return nil
}

func (r *Repository) GetCompany(ctx context.Context, id string) (identity.Company, error) {
	company, err := scanCompany(r.pool.QueryRow(ctx,
		`SELECT id, name, created_at, disabled_at, COALESCE(login_slug, ''), COALESCE(logo_content_type, '') FROM companies WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Company{}, identity.ErrNotFound
		}
		return identity.Company{}, fmt.Errorf("select company: %w", err)
	}
	return company, nil
}

func (r *Repository) ListCompanies(ctx context.Context) ([]identity.Company, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, created_at, disabled_at, COALESCE(login_slug, ''), COALESCE(logo_content_type, '') FROM companies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()
	companies := make([]identity.Company, 0)
	for rows.Next() {
		company, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		companies = append(companies, company)
	}
	return companies, rows.Err()
}

func scanCompany(row pgx.Row) (identity.Company, error) {
	var company identity.Company
	if err := row.Scan(&company.ID, &company.Name, &company.CreatedAt, &company.DisabledAt, &company.LoginSlug, &company.LogoContentType); err != nil {
		return identity.Company{}, err
	}
	company.CreatedAt = company.CreatedAt.UTC()
	return company, nil
}

func (r *Repository) GetCompanyByLoginSlug(ctx context.Context, slug string) (identity.Company, error) {
	company, err := scanCompany(r.pool.QueryRow(ctx,
		`SELECT id, name, created_at, disabled_at, COALESCE(login_slug, ''), COALESCE(logo_content_type, '') FROM companies WHERE login_slug = $1`, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Company{}, identity.ErrNotFound
		}
		return identity.Company{}, fmt.Errorf("select company by login slug: %w", err)
	}
	return company, nil
}

func (r *Repository) SetCompanyLoginSlug(ctx context.Context, id string, slug string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE companies SET login_slug = $2 WHERE id = $1`, id, slug)
	if err != nil {
		return fmt.Errorf("set company login slug: %w", mapConflict(err))
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) SetCompanyProfile(ctx context.Context, id string, name string, slug string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE companies SET name = $2, login_slug = $3 WHERE id = $1`, id, name, slug)
	if err != nil {
		return fmt.Errorf("set company profile: %w", mapConflict(err))
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) SetCompanyLogoType(ctx context.Context, id string, contentType string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE companies SET logo_content_type = NULLIF($2, '') WHERE id = $1`, id, contentType)
	if err != nil {
		return fmt.Errorf("set company logo type: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
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
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id LEFT JOIN user_totp t ON t.user_id = u.id WHERE u.id = $1`, id))
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (identity.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id LEFT JOIN user_totp t ON t.user_id = u.id WHERE u.login = $1`, login))
}

func (r *Repository) ListUsers(ctx context.Context, companyID string) ([]identity.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelect+` FROM users u LEFT JOIN companies c ON c.id = u.company_id LEFT JOIN user_totp t ON t.user_id = u.id
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

func (r *Repository) CreateSession(ctx context.Context, session identity.LoginSession) error {
	created := session.CreatedAt.UTC()
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, id, created_at, user_agent, ip)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		session.TokenHash, session.UserID, session.ExpiresAt.UTC(), session.ID, created, session.UserAgent, session.IP)
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
		 LEFT JOIN user_totp t ON t.user_id = u.id
		 WHERE s.token_hash = $1 AND s.expires_at > $2`, tokenHash, now.UTC()))
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return identity.User{}, identity.ErrUnauthorized
		}
		return identity.User{}, err
	}
	return user, nil
}

func (r *Repository) ListSessions(ctx context.Context, userID string, now time.Time) ([]identity.LoginSession, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(id, token_hash), token_hash, user_id, COALESCE(user_agent, ''), COALESCE(ip, ''), COALESCE(created_at, expires_at), expires_at
		 FROM sessions WHERE user_id = $1 AND expires_at > $2
		 ORDER BY created_at DESC NULLS LAST`, userID, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	out := make([]identity.LoginSession, 0)
	for rows.Next() {
		var item identity.LoginSession
		if err := rows.Scan(&item.ID, &item.TokenHash, &item.UserID, &item.UserAgent, &item.IP, &item.CreatedAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		item.CreatedAt = item.CreatedAt.UTC()
		item.ExpiresAt = item.ExpiresAt.UTC()
		out = append(out, item)
	}
	return out, rows.Err()
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

func (r *Repository) SaveTOTPSetup(ctx context.Context, settings identity.TOTP) error {
	hashes, err := marshalJSON(recoveryHashes(settings.RecoveryCodeHashes), "recovery_code_hashes")
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO user_totp (user_id, secret, enabled_at, recovery_code_hashes)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE SET
			secret = EXCLUDED.secret,
			enabled_at = EXCLUDED.enabled_at,
			recovery_code_hashes = EXCLUDED.recovery_code_hashes`,
		settings.UserID, settings.Secret, settings.EnabledAt, hashes)
	if err != nil {
		return fmt.Errorf("save totp: %w", err)
	}
	return nil
}

func (r *Repository) GetTOTP(ctx context.Context, userID string) (identity.TOTP, error) {
	var (
		settings identity.TOTP
		raw      []byte
	)
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, secret, enabled_at, recovery_code_hashes FROM user_totp WHERE user_id = $1`,
		userID).Scan(&settings.UserID, &settings.Secret, &settings.EnabledAt, &raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.TOTP{}, identity.ErrNotFound
		}
		return identity.TOTP{}, fmt.Errorf("select totp: %w", err)
	}
	if err := unmarshalJSON(raw, &settings.RecoveryCodeHashes, "recovery_code_hashes"); err != nil {
		return identity.TOTP{}, err
	}
	if settings.RecoveryCodeHashes == nil {
		settings.RecoveryCodeHashes = []string{}
	}
	return settings, nil
}

func (r *Repository) EnableTOTP(ctx context.Context, userID string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE user_totp SET enabled_at = $2 WHERE user_id = $1`, userID, at.UTC())
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) DisableTOTP(ctx context.Context, userID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_totp WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	encoded, err := marshalJSON(recoveryHashes(hashes), "recovery_code_hashes")
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE user_totp SET recovery_code_hashes = $2 WHERE user_id = $1`, userID, encoded)
	if err != nil {
		return fmt.Errorf("replace recovery codes: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateLoginChallenge(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO login_challenges (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("insert login challenge: %w", err)
	}
	return nil
}

func (r *Repository) GetLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM login_challenges WHERE token_hash = $1 AND expires_at > $2`,
		tokenHash, now.UTC()).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", identity.ErrUnauthorized
		}
		return "", fmt.Errorf("select login challenge: %w", err)
	}
	return userID, nil
}

func (r *Repository) ConsumeLoginChallenge(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`DELETE FROM login_challenges WHERE token_hash = $1 AND expires_at > $2 RETURNING user_id`,
		tokenHash, now.UTC()).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", identity.ErrUnauthorized
		}
		return "", fmt.Errorf("consume login challenge: %w", err)
	}
	return userID, nil
}

func (r *Repository) SavePasskey(ctx context.Context, credential identity.PasskeyCredential) error {
	if err := identity.AssertPasskeyCredentialJSON(credential.Raw); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO passkey_credentials (id, user_id, name, credential, created_at, last_used_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		credential.ID, credential.UserID, credential.DisplayName(), credential.Raw, credential.CreatedAt.UTC(), credential.LastUsedAt)
	if err != nil {
		return fmt.Errorf("insert passkey: %w", mapConflict(err))
	}
	return nil
}

func (r *Repository) ListPasskeys(ctx context.Context, userID string) ([]identity.PasskeyCredential, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, credential, created_at, last_used_at
		 FROM passkey_credentials WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	defer rows.Close()
	out := make([]identity.PasskeyCredential, 0)
	for rows.Next() {
		credential, err := scanPasskey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, credential)
	}
	return out, rows.Err()
}

func (r *Repository) GetPasskey(ctx context.Context, id string) (identity.PasskeyCredential, error) {
	credential, err := scanPasskey(r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, credential, created_at, last_used_at FROM passkey_credentials WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.PasskeyCredential{}, identity.ErrNotFound
		}
		return identity.PasskeyCredential{}, fmt.Errorf("select passkey: %w", err)
	}
	return credential, nil
}

func (r *Repository) DeletePasskey(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM passkey_credentials WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete passkey: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) UpdatePasskey(ctx context.Context, credential identity.PasskeyCredential) error {
	if err := identity.AssertPasskeyCredentialJSON(credential.Raw); err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE passkey_credentials SET name = $3, credential = $4, last_used_at = $5 WHERE id = $1 AND user_id = $2`,
		credential.ID, credential.UserID, credential.DisplayName(), credential.Raw, credential.LastUsedAt)
	if err != nil {
		return fmt.Errorf("update passkey: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) SavePasskeyChallenge(ctx context.Context, challenge identity.PasskeyChallenge) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO passkey_challenges (id, user_id, purpose, challenge, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		challenge.ID, nullIfEmpty(challenge.UserID), challenge.Purpose, challenge.Session, challenge.ExpiresAt.UTC())
	if err != nil {
		return fmt.Errorf("insert passkey challenge: %w", err)
	}
	return nil
}

func (r *Repository) GetPasskeyChallenge(ctx context.Context, id string, now time.Time) (identity.PasskeyChallenge, error) {
	challenge, err := scanPasskeyChallenge(r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(user_id, ''), purpose, challenge, expires_at
		 FROM passkey_challenges WHERE id = $1 AND expires_at > $2`, id, now.UTC()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.PasskeyChallenge{}, identity.ErrUnauthorized
		}
		return identity.PasskeyChallenge{}, fmt.Errorf("select passkey challenge: %w", err)
	}
	return challenge, nil
}

func (r *Repository) ConsumePasskeyChallenge(ctx context.Context, id string, now time.Time) (identity.PasskeyChallenge, error) {
	challenge, err := scanPasskeyChallenge(r.pool.QueryRow(ctx,
		`DELETE FROM passkey_challenges WHERE id = $1 AND expires_at > $2
		 RETURNING id, COALESCE(user_id, ''), purpose, challenge, expires_at`, id, now.UTC()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.PasskeyChallenge{}, identity.ErrUnauthorized
		}
		return identity.PasskeyChallenge{}, fmt.Errorf("consume passkey challenge: %w", err)
	}
	return challenge, nil
}

func scanPasskey(row pgx.Row) (identity.PasskeyCredential, error) {
	var credential identity.PasskeyCredential
	if err := row.Scan(&credential.ID, &credential.UserID, &credential.Name, &credential.Raw, &credential.CreatedAt, &credential.LastUsedAt); err != nil {
		return identity.PasskeyCredential{}, err
	}
	credential.CreatedAt = credential.CreatedAt.UTC()
	fillPasskeyFromRaw(&credential)
	return credential, nil
}

func scanPasskeyChallenge(row pgx.Row) (identity.PasskeyChallenge, error) {
	var challenge identity.PasskeyChallenge
	if err := row.Scan(&challenge.ID, &challenge.UserID, &challenge.Purpose, &challenge.Session, &challenge.ExpiresAt); err != nil {
		return identity.PasskeyChallenge{}, err
	}
	challenge.ExpiresAt = challenge.ExpiresAt.UTC()
	return challenge, nil
}

func fillPasskeyFromRaw(credential *identity.PasskeyCredential) {
	var parsed struct {
		PublicKey     []byte   `json:"publicKey"`
		Transport     []string `json:"transport"`
		Authenticator struct {
			AAGUID    []byte `json:"aaguid"`
			SignCount uint32 `json:"signCount"`
		} `json:"authenticator"`
	}
	if err := unmarshalJSON(credential.Raw, &parsed, "passkey"); err != nil {
		return
	}
	credential.PublicKey = parsed.PublicKey
	credential.Transports = parsed.Transport
	credential.AAGUID = parsed.Authenticator.AAGUID
	credential.SignCount = parsed.Authenticator.SignCount
}

func recoveryHashes(hashes []string) []string {
	if hashes == nil {
		return []string{}
	}
	return hashes
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
	err := row.Scan(&user.ID, &user.CompanyID, &user.CompanyName, &user.CompanyLoginSlug, &user.CompanyHasLogo, &user.Login, &user.PasswordHash, &role, &user.CreatedAt, &user.DisabledAt, &user.CompanyDisabled, &user.TwoFactorEnabled, &user.HasPasskey)
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
