-- name: CreateAccountSession :one
INSERT INTO account_sessions (
    session_id, account_id, identity_id, kratos_session_id,
    ip_address, user_agent, expires_at
)
VALUES (
    @session_id, @account_id, @identity_id, @kratos_session_id,
    @ip_address, @user_agent, @expires_at
)
RETURNING *;

-- name: GetAccountSession :one
SELECT * FROM account_sessions WHERE session_id = @session_id;

-- name: GetAccountSessionByKratosSessionID :one
SELECT * FROM account_sessions
WHERE kratos_session_id = @kratos_session_id
LIMIT 1;

-- name: GetActiveSessionsByAccountID :many
SELECT * FROM account_sessions
WHERE account_id = @account_id
  AND revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC;

-- name: UpdateSessionActivity :exec
UPDATE account_sessions
SET last_activity_at = @last_activity_at
WHERE session_id = @session_id;

-- name: RevokeSession :exec
UPDATE account_sessions
SET revoked_at = @revoked_at
WHERE session_id = @session_id;

-- name: RevokeAccountSessions :exec
UPDATE account_sessions
SET revoked_at = @revoked_at
WHERE account_id = @account_id
  AND revoked_at IS NULL;

-- name: CleanupExpiredSessions :execrows
DELETE FROM account_sessions
WHERE expires_at < CURRENT_TIMESTAMP
   OR revoked_at IS NOT NULL;
