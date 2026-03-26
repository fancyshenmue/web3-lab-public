-- name: CreateAccountIdentity :one
INSERT INTO account_identities (
    identity_id, account_id, kratos_identity_id, provider_id, provider_user_id,
    display_name, avatar_url, attributes, raw_data, verified, is_primary
)
VALUES (
    @identity_id, @account_id, @kratos_identity_id, @provider_id, @provider_user_id,
    @display_name, @avatar_url, @attributes, @raw_data, @verified, @is_primary
)
RETURNING *;

-- name: GetAccountIdentity :one
SELECT * FROM account_identities WHERE identity_id = @identity_id;

-- name: GetAccountIdentityByKratosID :one
SELECT * FROM account_identities
WHERE kratos_identity_id = @kratos_identity_id
  AND unlinked_at IS NULL
LIMIT 1;

-- name: GetAccountIdentityByProviderUserID :one
SELECT * FROM account_identities
WHERE provider_id = @provider_id
  AND provider_user_id = @provider_user_id
  AND unlinked_at IS NULL
LIMIT 1;

-- name: GetAccountIdentitiesByAccountID :many
SELECT * FROM account_identities
WHERE account_id = @account_id
  AND unlinked_at IS NULL
ORDER BY linked_at DESC;

-- name: UpdateAccountIdentity :exec
UPDATE account_identities
SET account_id   = @account_id,
    display_name = @display_name,
    avatar_url   = @avatar_url,
    attributes   = @attributes,
    raw_data     = @raw_data,
    verified     = @verified,
    is_primary   = @is_primary,
    last_used_at = @last_used_at,
    updated_at   = CURRENT_TIMESTAMP
WHERE identity_id = @identity_id;

-- name: UpdateIdentityLastUsed :exec
UPDATE account_identities
SET last_used_at = @last_used_at,
    updated_at   = CURRENT_TIMESTAMP
WHERE identity_id = @identity_id;

-- name: SetPrimaryIdentityReset :exec
UPDATE account_identities
SET is_primary = false,
    updated_at = CURRENT_TIMESTAMP
WHERE account_id = @account_id;

-- name: SetPrimaryIdentitySet :exec
UPDATE account_identities
SET is_primary = true,
    updated_at = CURRENT_TIMESTAMP
WHERE identity_id = @identity_id
  AND account_id = @account_id;

-- name: SoftDeleteAccountIdentity :exec
UPDATE account_identities
SET unlinked_at = @unlinked_at,
    updated_at  = CURRENT_TIMESTAMP
WHERE identity_id = @identity_id;

-- name: DeleteAccountIdentity :exec
DELETE FROM account_identities WHERE identity_id = @identity_id;

-- name: FindAccountByEmail :one
SELECT a.* FROM accounts a
JOIN account_identities ai ON a.account_id = ai.account_id
WHERE ai.attributes->>'email' = @email
  AND ai.unlinked_at IS NULL
LIMIT 1;

-- name: FindAccountByEOA :one
SELECT a.* FROM accounts a
JOIN account_identities ai ON a.account_id = ai.account_id
WHERE ai.provider_id = 'eoa'
  AND ai.attributes->>'eoa_address' = @eoa_address
  AND ai.unlinked_at IS NULL
LIMIT 1;
