-- name: CreateAccount :one
INSERT INTO accounts (account_id, status, metadata)
VALUES (@account_id, @status, @metadata)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE account_id = @account_id;

-- name: GetAccountByKratosIdentityID :one
SELECT a.*
FROM accounts a
JOIN account_identities ai ON a.account_id = ai.account_id
WHERE ai.kratos_identity_id = @kratos_identity_id
  AND ai.unlinked_at IS NULL
LIMIT 1;

-- name: UpdateAccount :exec
UPDATE accounts
SET last_login_at = @last_login_at,
    status        = @status,
    metadata      = @metadata,
    updated_at    = CURRENT_TIMESTAMP
WHERE account_id = @account_id;

-- name: UpdateAccountStatus :exec
UPDATE accounts
SET status     = @status,
    updated_at = CURRENT_TIMESTAMP
WHERE account_id = @account_id;

-- name: UpdateLastLogin :exec
UPDATE accounts
SET last_login_at = @last_login_at,
    updated_at    = CURRENT_TIMESTAMP
WHERE account_id = @account_id;
