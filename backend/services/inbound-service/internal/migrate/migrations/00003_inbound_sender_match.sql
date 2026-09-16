-- Add subject column to inbound_message
ALTER TABLE inbound_message ADD COLUMN IF NOT EXISTS subject TEXT NOT NULL DEFAULT '';

-- Add receive_address (platform-wide CloudMailin address) to inbound_settings
ALTER TABLE inbound_settings ADD COLUMN IF NOT EXISTS receive_address TEXT NOT NULL DEFAULT '';

-- Add sender_email (single email per company) to company_inbound
ALTER TABLE company_inbound ADD COLUMN IF NOT EXISTS sender_email TEXT NOT NULL DEFAULT '';

-- Migrate existing data: first element of allowed_from array → sender_email
UPDATE company_inbound
SET sender_email = allowed_from[1]
WHERE sender_email = ''
  AND array_length(allowed_from, 1) > 0;
