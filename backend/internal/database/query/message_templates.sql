-- name: CreateMessageTemplate :one
INSERT INTO message_templates (
    name, protocol, statement, domain, uri, chain_id, version, nonce_ttl_secs
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetMessageTemplate :one
SELECT * FROM message_templates
WHERE id = $1;

-- name: GetMessageTemplateByName :one
SELECT * FROM message_templates
WHERE name = $1;

-- name: ListMessageTemplates :many
SELECT * FROM message_templates
ORDER BY created_at DESC;

-- name: UpdateMessageTemplate :one
UPDATE message_templates
SET
    name = COALESCE(NULLIF($2, ''), name),
    statement = COALESCE(NULLIF($3, ''), statement),
    domain = COALESCE(NULLIF($4, ''), domain),
    uri = COALESCE(NULLIF($5, ''), uri),
    chain_id = COALESCE($6, chain_id),
    version = COALESCE(NULLIF($7, ''), version),
    nonce_ttl_secs = COALESCE($8, nonce_ttl_secs),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteMessageTemplate :exec
DELETE FROM message_templates
WHERE id = $1;

-- name: CountAppClientsByTemplateID :one
SELECT COUNT(*) FROM app_clients
WHERE message_template_id = $1;

-- name: GetAppClientWithTemplate :one
SELECT ac.*, mt.id AS template_id, mt.name AS template_name,
       mt.protocol AS template_protocol, mt.statement AS template_statement,
       mt.domain AS template_domain, mt.uri AS template_uri,
       mt.chain_id AS template_chain_id, mt.version AS template_version,
       mt.nonce_ttl_secs AS template_nonce_ttl_secs
FROM app_clients ac
LEFT JOIN message_templates mt ON ac.message_template_id = mt.id
WHERE ac.id = $1;
