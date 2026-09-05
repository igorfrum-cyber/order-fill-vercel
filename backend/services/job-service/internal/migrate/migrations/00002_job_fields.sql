ALTER TABLE jobs ADD COLUMN IF NOT EXISTS progress DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS progress_message TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS input_files JSONB NOT NULL DEFAULT '[]';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS output_files JSONB NOT NULL DEFAULT '[]';

CREATE TABLE IF NOT EXISTS job_reports (
    job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

ALTER TABLE job_reports ADD COLUMN IF NOT EXISTS payload JSONB;
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'job_reports' AND column_name = 'summary'
	) THEN
		UPDATE job_reports SET payload = jsonb_build_object(
			'summary', COALESCE(summary, '{}'::jsonb),
			'rows', COALESCE("rows", '[]'::jsonb)
		) WHERE payload IS NULL;
	END IF;
END $$;
UPDATE job_reports SET payload = '{}'::jsonb WHERE payload IS NULL;
