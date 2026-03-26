-- name: CreateAppClient :one
INSERT INTO app_clients (
    id,
    name,
    oauth2_client_id,
    frontend_url,
    login_path,
    logout_url,
    allowed_cors_origins,
    jwt_secret,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
) RETURNING *;

-- name: GetAppClient :one
SELECT * FROM app_clients
WHERE id = $1;

-- name: GetAppClientByOAuth2ID :one
SELECT * FROM app_clients
WHERE oauth2_client_id = $1;

-- name: ListAppClients :many
SELECT * FROM app_clients
ORDER BY created_at DESC;

-- name: UpdateAppClient :one
UPDATE app_clients
SET
    name = $2,
    frontend_url = $3,
    login_path = $4,
    logout_url = $5,
    allowed_cors_origins = $6,
    jwt_secret = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAppClient :exec
DELETE FROM app_clients
WHERE id = $1;
