CREATE TABLE IF NOT EXISTS message_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(100) UNIQUE NOT NULL,
    protocol       VARCHAR(10) NOT NULL CHECK (protocol IN ('siwe', 'eip712')),
    statement      TEXT NOT NULL,
    domain         VARCHAR(255) NOT NULL,
    uri            VARCHAR(500) NOT NULL,
    chain_id       INTEGER NOT NULL DEFAULT 1,
    version        VARCHAR(10) NOT NULL DEFAULT '1',
    nonce_ttl_secs INTEGER NOT NULL DEFAULT 300,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK from app_clients to message_templates
ALTER TABLE app_clients
    ADD COLUMN message_template_id UUID REFERENCES message_templates(id);

-- Seed the default template
INSERT INTO message_templates (name, protocol, statement, domain, uri, chain_id, version, nonce_ttl_secs)
VALUES (
    'default',
    'siwe',
    'Sign in to {service_name}',
    'app.web3-local-dev.com',
    'https://app.web3-local-dev.com',
    72390,
    '1',
    300
);
