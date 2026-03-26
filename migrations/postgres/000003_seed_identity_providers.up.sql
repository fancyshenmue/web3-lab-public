-- Idempotent seed: default identity providers
-- Can be run multiple times safely (ON CONFLICT DO NOTHING)
INSERT INTO identity_providers (provider_id, provider_name, provider_type, enabled) VALUES
    ('google',   'Google',          'oidc',   true),
    ('github',   'GitHub',          'oauth2', true),
    ('facebook', 'Facebook',        'oauth2', true),
    ('apple',    'Apple',           'oidc',   true),
    ('email',    'Email/Password',  'email',  true),
    ('eoa',      'Ethereum EOA',    'web3',   true)
ON CONFLICT (provider_id) DO NOTHING;
