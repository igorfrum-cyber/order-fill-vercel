CREATE TABLE IF NOT EXISTS audit_events (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL DEFAULT '',
	actor_id TEXT NOT NULL DEFAULT '',
	company_id TEXT NOT NULL DEFAULT '',
	job_id TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ,
	payload TEXT NOT NULL DEFAULT ''
);

ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS company_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS job_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS payload TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'audit_events' AND column_name = 'at'
	) THEN
		UPDATE audit_events SET created_at = COALESCE(created_at, at);
	END IF;
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'audit_events' AND column_name = 'action'
	) THEN
		UPDATE audit_events SET type = COALESCE(NULLIF(type, ''), action);
	END IF;
END $$;

UPDATE audit_events SET created_at = NOW() WHERE created_at IS NULL;

CREATE INDEX IF NOT EXISTS audit_events_company_id_idx ON audit_events (company_id, created_at DESC);
