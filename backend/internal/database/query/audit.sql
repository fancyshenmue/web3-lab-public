-- name: CreateAuditLog :one
INSERT INTO account_audit_logs (
    log_id, account_id, identity_id, event_type, event_status, event_message,
    session_id, kratos_session_id, ip_address, user_agent, provider_id, event_data
)
VALUES (
    @log_id, @account_id, @identity_id, @event_type, @event_status, @event_message,
    @session_id, @kratos_session_id, @ip_address, @user_agent, @provider_id, @event_data
)
RETURNING *;

-- name: GetAuditLogsByAccountID :many
SELECT * FROM account_audit_logs
WHERE account_id = @account_id
ORDER BY created_at DESC
LIMIT @log_limit;

-- name: GetAuditLogsByEventType :many
SELECT * FROM account_audit_logs
WHERE event_type = @event_type
ORDER BY created_at DESC
LIMIT @log_limit;

-- name: GetFailedLoginAttempts :many
SELECT * FROM account_audit_logs
WHERE event_type = 'LOGIN'
  AND event_status = 'FAILURE'
  AND identity_id = @identity_id
  AND created_at >= @since
ORDER BY created_at DESC;

-- name: GetAuditLogsBySession :many
SELECT * FROM account_audit_logs
WHERE session_id = @session_id
ORDER BY created_at DESC;
