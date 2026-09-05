CREATE TABLE IF NOT EXISTS user_totp (
	user_id TEXT PRIMARY KEY,
	secret TEXT NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT false,
	recovery_code_hashes JSONB NOT NULL DEFAULT '[]'
);

ALTER TABLE user_totp ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT false;
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user_totp' AND column_name = 'enabled_at'
	) THEN
		UPDATE user_totp SET enabled = true WHERE enabled_at IS NOT NULL;
	END IF;
END $$;
