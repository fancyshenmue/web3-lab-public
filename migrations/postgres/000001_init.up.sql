-- Account-System PostgreSQL Schema Initialization
-- Consolidated from 8 Spanner migrations into a single baseline.
-- Tables: accounts, identity_providers, account_identities,
--         account_sessions, account_merge_history, account_audit_logs

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- 1. accounts
-- =============================================================================
CREATE TABLE accounts (
    account_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ,
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata      JSONB
);

CREATE INDEX idx_accounts_status ON accounts (status);
CREATE INDEX idx_accounts_created_at ON accounts (created_at DESC);

-- =============================================================================
-- 2. identity_providers
-- =============================================================================
CREATE TABLE identity_providers (
    provider_id    VARCHAR(50)  PRIMARY KEY,
    provider_name  VARCHAR(100) NOT NULL,
    provider_type  VARCHAR(50)  NOT NULL,
    enabled        BOOLEAN      NOT NULL DEFAULT true,
    configuration  JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- 3. account_identities
-- =============================================================================
CREATE TABLE account_identities (
    identity_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id         UUID         NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    kratos_identity_id UUID,
    provider_id        VARCHAR(50)  NOT NULL REFERENCES identity_providers(provider_id),
    provider_user_id   VARCHAR(255) NOT NULL,
    display_name       VARCHAR(255),
    avatar_url         VARCHAR(500),
    attributes         JSONB,
    raw_data           JSONB,
    verified           BOOLEAN      NOT NULL DEFAULT false,
    is_primary         BOOLEAN      NOT NULL DEFAULT false,
    linked_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at       TIMESTAMPTZ,
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unlinked_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_account_identities_provider_user ON account_identities (provider_id, provider_user_id);
CREATE UNIQUE INDEX idx_account_identities_kratos_id ON account_identities (kratos_identity_id) WHERE kratos_identity_id IS NOT NULL;
CREATE INDEX idx_account_identities_account_id ON account_identities (account_id);
CREATE INDEX idx_account_identities_provider_id ON account_identities (provider_id);
CREATE INDEX idx_account_identities_linked_at ON account_identities (linked_at DESC);

-- =============================================================================
-- 4. account_sessions
-- =============================================================================
CREATE TABLE account_sessions (
    session_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id        UUID         NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    identity_id       UUID         NOT NULL REFERENCES account_identities(identity_id),
    kratos_session_id UUID,
    ip_address        VARCHAR(45),
    user_agent        VARCHAR(500),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at        TIMESTAMPTZ  NOT NULL,
    revoked_at        TIMESTAMPTZ,
    last_activity_at  TIMESTAMPTZ
);

CREATE INDEX idx_account_sessions_account_id ON account_sessions (account_id);
CREATE INDEX idx_account_sessions_kratos_session_id ON account_sessions (kratos_session_id);
CREATE INDEX idx_account_sessions_expires_at ON account_sessions (expires_at);
CREATE INDEX idx_account_sessions_created_at ON account_sessions (created_at DESC);

-- =============================================================================
-- 5. account_merge_history
-- =============================================================================
CREATE TABLE account_merge_history (
    merge_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_account_id      UUID        NOT NULL REFERENCES accounts(account_id),
    target_account_id      UUID        NOT NULL REFERENCES accounts(account_id),
    merged_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    merged_by              UUID,
    reason                 VARCHAR(500),
    identities_transferred INTEGER,
    metadata               JSONB
);

CREATE INDEX idx_account_merge_history_source ON account_merge_history (source_account_id);
CREATE INDEX idx_account_merge_history_target ON account_merge_history (target_account_id);
CREATE INDEX idx_account_merge_history_merged_at ON account_merge_history (merged_at DESC);

-- =============================================================================
-- 6. account_audit_logs
-- =============================================================================
CREATE TABLE account_audit_logs (
    log_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id        UUID,
    identity_id       VARCHAR(255),
    event_type        VARCHAR(50)  NOT NULL,
    event_status      VARCHAR(20)  NOT NULL,
    event_message     TEXT,
    session_id        UUID,
    kratos_session_id VARCHAR(255),
    ip_address        VARCHAR(45),
    user_agent        VARCHAR(500),
    provider_id       VARCHAR(50),
    event_data        JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_account_audit_logs_account_id ON account_audit_logs (account_id, created_at DESC);
CREATE INDEX idx_account_audit_logs_event_type ON account_audit_logs (event_type, created_at DESC);
CREATE INDEX idx_account_audit_logs_session_id ON account_audit_logs (session_id, created_at DESC);
CREATE INDEX idx_account_audit_logs_failed_attempts ON account_audit_logs (event_status, event_type, created_at DESC)
    INCLUDE (ip_address, identity_id);

-- =============================================================================
-- Seed data moved to 000003_seed_identity_providers.up.sql (idempotent)
-- =============================================================================
