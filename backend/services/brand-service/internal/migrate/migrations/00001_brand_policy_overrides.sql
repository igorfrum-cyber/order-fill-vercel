CREATE TABLE IF NOT EXISTS brand_policy_overrides (
	brand TEXT PRIMARY KEY,
	policy JSONB NOT NULL,
	updated_by TEXT NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);
