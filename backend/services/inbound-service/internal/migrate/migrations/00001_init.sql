CREATE TABLE IF NOT EXISTS inbound_settings (
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_webhook_at TIMESTAMPTZ,
    webhook_count  BIGINT NOT NULL DEFAULT 0,
    error_count    BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS company_inbound (
    company_id      TEXT PRIMARY KEY,
    receive_address TEXT UNIQUE NOT NULL,
    allowed_from    TEXT[] NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS inbound_message (
    id                  TEXT PRIMARY KEY,
    provider_message_id TEXT UNIQUE NOT NULL,
    envelope_from       TEXT NOT NULL DEFAULT '',
    envelope_to         TEXT NOT NULL DEFAULT '',
    received_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    company_id          TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'received',
    error_code          TEXT NOT NULL DEFAULT '',
    attachment_count    INT NOT NULL DEFAULT 0,
    total_bytes         BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inbound_attachment (
    id           TEXT PRIMARY KEY,
    message_id   TEXT NOT NULL REFERENCES inbound_message(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size         BIGINT NOT NULL DEFAULT 0,
    object_key   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inbound_message_company ON inbound_message(company_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_inbound_message_provider ON inbound_message(provider_message_id);
CREATE INDEX IF NOT EXISTS idx_inbound_attachment_message ON inbound_attachment(message_id);