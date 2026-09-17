-- The CloudMailin receive address is configured globally in inbound_settings.
-- Company settings may repeat that address or leave it empty.
ALTER TABLE company_inbound
    DROP CONSTRAINT IF EXISTS company_inbound_receive_address_key;
