-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    id, event_type, aggregate_type, aggregate_id, payload,
    status, attempts, next_attempt_at, created_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, now()
) RETURNING *;

-- name: ListPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'pending' AND next_attempt_at <= now()
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventPublished :one
UPDATE outbox_events
SET status = 'published',
    published_at = now()
WHERE id = $1
RETURNING *;

-- name: ListOutboxEventsByAggregate :many
SELECT * FROM outbox_events
WHERE aggregate_type = $1 AND aggregate_id = $2
ORDER BY created_at ASC;
