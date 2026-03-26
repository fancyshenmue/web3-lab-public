CREATE TABLE IF NOT EXISTS app_clients (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    oauth2_client_id VARCHAR(255) NOT NULL UNIQUE,
    frontend_url VARCHAR(255) NOT NULL,
    login_path VARCHAR(255) DEFAULT '/login',
    logout_url VARCHAR(255) DEFAULT '',
    allowed_cors_origins JSONB DEFAULT '[]'::jsonb,
    jwt_secret VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_clients_oauth2_client_id ON app_clients(oauth2_client_id);
