CREATE TABLE IF NOT EXISTS passkey_credentials (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	public_key BYTEA,
	sign_count BIGINT NOT NULL DEFAULT 0,
	transports JSONB NOT NULL DEFAULT '[]',
	aaguid BYTEA,
	raw BYTEA NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	last_used_at TIMESTAMPTZ
);

ALTER TABLE passkey_credentials ADD COLUMN IF NOT EXISTS public_key BYTEA;
ALTER TABLE passkey_credentials ADD COLUMN IF NOT EXISTS sign_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE passkey_credentials ADD COLUMN IF NOT EXISTS transports JSONB NOT NULL DEFAULT '[]';
ALTER TABLE passkey_credentials ADD COLUMN IF NOT EXISTS aaguid BYTEA;
ALTER TABLE passkey_credentials ADD COLUMN IF NOT EXISTS raw BYTEA;
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'passkey_credentials' AND column_name = 'credential'
	) THEN
		ALTER TABLE passkey_credentials ALTER COLUMN credential DROP NOT NULL;
		ALTER TABLE passkey_credentials ALTER COLUMN credential SET DEFAULT '{}'::jsonb;
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS passkey_credentials_user_id_idx ON passkey_credentials (user_id);
