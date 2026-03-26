-- name: GetIdentityProvider :one
SELECT * FROM identity_providers WHERE provider_id = @provider_id;

-- name: ListIdentityProviders :many
SELECT * FROM identity_providers
WHERE (sqlc.arg(only_enabled)::boolean = false OR enabled = true)
ORDER BY provider_id;

-- name: UpdateIdentityProvider :exec
UPDATE identity_providers
SET provider_name = @provider_name,
    provider_type = @provider_type,
    enabled       = @enabled,
    configuration = @configuration,
    updated_at    = CURRENT_TIMESTAMP
WHERE provider_id = @provider_id;
