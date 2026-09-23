-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    id, actor_type, actor_id, action, resource_type, resource_id,
    before_data, after_data, ip_address, user_agent, request_id, trace_id, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now()
) RETURNING *;

-- name: ListAuditEvents :many
SELECT * FROM audit_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
