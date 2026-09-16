-- Make inbound_settings a proper singleton with a primary-key conflict target.
ALTER TABLE inbound_settings ADD COLUMN IF NOT EXISTS id SMALLINT;
UPDATE inbound_settings SET id = 1 WHERE id IS NULL;
ALTER TABLE inbound_settings ALTER COLUMN id SET NOT NULL;
ALTER TABLE inbound_settings ALTER COLUMN id SET DEFAULT 1;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'inbound_settings_pkey'
          AND conrelid = 'inbound_settings'::regclass
    ) THEN
        ALTER TABLE inbound_settings ADD CONSTRAINT inbound_settings_pkey PRIMARY KEY (id);
    END IF;
END
$$;