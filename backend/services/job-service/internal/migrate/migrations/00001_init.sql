-- jobs owns job state, matching_mode snapshot, and report categories.
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    company_id TEXT NOT NULL,
    matching_mode TEXT NOT NULL DEFAULT 'standard',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    error_message TEXT NOT NULL DEFAULT ''
);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS owner_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS company_id TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS matching_mode TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS error_message TEXT NOT NULL DEFAULT '';
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'jobs' AND column_name = 'created_by'
	) THEN
		UPDATE jobs SET owner_user_id = COALESCE(NULLIF(owner_user_id, ''), created_by, '');
	END IF;
END $$;
UPDATE jobs SET error_message = '' WHERE error_message IS NULL;
