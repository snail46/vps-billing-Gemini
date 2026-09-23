-- name: CreateOperation :one
INSERT INTO operations (
  id, type, resource_type, resource_id, status, phase, progress,
  message_key, provider_id, provider_operation_id, idempotency_key,
  retryable, retry_count, max_retries, trace_id, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11,
  $12, $13, $14, $15, now(), now()
)
RETURNING *;

-- name: GetOperationByID :one
SELECT * FROM operations
WHERE id = $1;

-- name: GetOperationByIdempotencyKey :one
SELECT * FROM operations
WHERE idempotency_key = $1;

-- name: ListOperationsByResource :many
SELECT * FROM operations
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at DESC;

-- name: ListPendingOperations :many
SELECT * FROM operations
WHERE status IN ('queued', 'retrying')
ORDER BY created_at ASC
LIMIT $1;

-- name: ListRecentOperations :many
SELECT * FROM operations
ORDER BY created_at DESC
LIMIT $1;

-- name: ListStuckOperations :many
SELECT * FROM operations
WHERE status = 'running' AND updated_at < $1
ORDER BY updated_at ASC;

-- name: UpdateOperationProgress :exec
UPDATE operations
SET status = $2,
    phase = $3,
    progress = $4,
    message_key = $5,
    started_at = COALESCE(started_at, $6),
    updated_at = now()
WHERE id = $1;

-- name: CompleteOperation :exec
UPDATE operations
SET status = 'succeeded',
    progress = 100,
    phase = 'completed',
    message_key = 'operations.completed',
    finished_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: FailOperation :exec
UPDATE operations
SET status = 'failed',
    error_code = $2,
    error_message = $3,
    finished_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: SetOperationProviderInfo :exec
UPDATE operations
SET provider_id = $2,
    provider_operation_id = $3,
    updated_at = now()
WHERE id = $1;

-- name: CreateOperationStep :one
INSERT INTO operation_steps (
  id, operation_id, step_key, step_order, status, progress, attempt, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, now(), now()
)
RETURNING *;

-- name: ListOperationSteps :many
SELECT * FROM operation_steps
WHERE operation_id = $1
ORDER BY step_order ASC;

-- name: UpdateOperationStep :exec
UPDATE operation_steps
SET status = $3,
    progress = $4,
    attempt = attempt + 1,
    error_code = $5,
    error_message = $6,
    started_at = COALESCE(started_at, $7),
    finished_at = $8,
    updated_at = now()
WHERE operation_id = $1 AND step_key = $2;
