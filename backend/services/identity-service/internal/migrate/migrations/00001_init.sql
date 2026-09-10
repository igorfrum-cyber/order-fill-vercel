CREATE TABLE IF NOT EXISTS companies (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	disabled_at TIMESTAMPTZ,
	login_slug TEXT NOT NULL,
	logo_content_type TEXT NOT NULL DEFAULT '',
	matching_mode TEXT NOT NULL DEFAULT 'standard'
);

ALTER TABLE companies ADD COLUMN IF NOT EXISTS login_slug TEXT NOT NULL DEFAULT '';
ALTER TABLE companies ADD COLUMN IF NOT EXISTS logo_content_type TEXT NOT NULL DEFAULT '';
ALTER TABLE companies ADD COLUMN IF NOT EXISTS matching_mode TEXT NOT NULL DEFAULT 'standard';
UPDATE companies SET login_slug = 'company-' || lower(left(replace(id, '-', ''), 8))
	WHERE login_slug IS NULL OR login_slug = '';

CREATE UNIQUE INDEX IF NOT EXISTS companies_login_slug_uidx ON companies (login_slug);

CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	company_id TEXT REFERENCES companies (id),
	login TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	disabled_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	id TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	user_agent TEXT NOT NULL DEFAULT '',
	ip TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS invite_tokens (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS login_challenges (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	expires_at TIMESTAMPTZ NOT NULL
);
