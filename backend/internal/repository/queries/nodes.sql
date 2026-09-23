-- name: CreateNode :one
INSERT INTO nodes (
  id, provider_id, node_group_id, provider_node_id, name, region, status,
  cpu_total, memory_total_mb, disk_total_gb,
  cpu_allocated, memory_allocated_mb, disk_allocated_gb,
  cpu_reserved, memory_reserved_mb, disk_reserved_gb,
  weight, capabilities, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10,
  $11, $12, $13,
  $14, $15, $16,
  $17, $18, now(), now()
)
RETURNING *;

-- name: GetNodeByID :one
SELECT * FROM nodes
WHERE id = $1;

-- name: GetNodeForUpdate :one
SELECT * FROM nodes
WHERE id = $1
FOR UPDATE;

-- name: ListNodes :many
SELECT * FROM nodes
ORDER BY created_at ASC;

-- name: ListActiveNodes :many
SELECT * FROM nodes
WHERE status = 'active'
ORDER BY weight DESC, created_at ASC;

-- name: ListNodesByNodeGroup :many
SELECT * FROM nodes
WHERE node_group_id = $1 AND status = 'active'
ORDER BY weight DESC, created_at ASC;

-- name: UpdateNodeHeartbeat :exec
UPDATE nodes
SET last_seen_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateNodeReservationDelta :exec
UPDATE nodes
SET cpu_reserved = cpu_reserved + $2,
    memory_reserved_mb = memory_reserved_mb + $3,
    disk_reserved_gb = disk_reserved_gb + $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1;

-- name: CommitNodeReservation :exec
UPDATE nodes
SET cpu_reserved = cpu_reserved - $2,
    memory_reserved_mb = memory_reserved_mb - $3,
    disk_reserved_gb = disk_reserved_gb - $4,
    cpu_allocated = cpu_allocated + $2,
    memory_allocated_mb = memory_allocated_mb + $3,
    disk_allocated_gb = disk_allocated_gb + $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1;

-- name: UpdateNodeStatus :exec
UPDATE nodes
SET status = $2, updated_at = now()
WHERE id = $1;
