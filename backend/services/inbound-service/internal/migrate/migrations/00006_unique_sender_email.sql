CREATE UNIQUE INDEX IF NOT EXISTS company_inbound_sender_email_lower_uidx
    ON company_inbound (LOWER(sender_email))
    WHERE sender_email <> '';
