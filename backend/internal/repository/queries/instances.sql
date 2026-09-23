-- name: CreateInstance :one
INSERT INTO instances (
  id, subscription_id, node_id, provider_id, provider_instance_id, name,
  desired_state, observed_state, cpu_cores, memory_mb, disk_gb,
  traffic_limit_gb, bandwidth_mbps, image_id, primary_ipv4, primary_ipv6,
  version, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11,
  $12, $13, $14, $15, $16,
  1, now(), now()
)
RETURNING *;

-- name: GetInstanceByID :one
SELECT * FROM instances
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetInstanceBySubscriptionID :one
SELECT * FROM instances
WHERE subscription_id = $1 AND deleted_at IS NULL;

-- name: ListInstances :many
SELECT * FROM instances
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListInstancesByNodeID :many
SELECT * FROM instances
WHERE node_id = $1 AND deleted_at IS NULL;

-- name: UpdateInstanceStates :exec
UPDATE instances
SET desired_state = $2,
    observed_state = $3,
    provider_instance_id = COALESCE($4, provider_instance_id),
    last_synced_at = now(),
    version = version + 1,
    updated_at = now()
WHERE id = $1;

-- name: SoftDeleteInstance :exec
UPDATE instances
SET deleted_at = now(), updated_at = now()
WHERE id = $1;
