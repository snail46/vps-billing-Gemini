-- name: CreateResourceReservation :one
INSERT INTO resource_reservations (
  id, node_id, operation_id, cpu_cores, memory_mb, disk_gb,
  ipv4_count, ipv6_count, nat_port_count, status, expires_at, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), now()
)
RETURNING *;

-- name: GetResourceReservationByID :one
SELECT * FROM resource_reservations
WHERE id = $1;

-- name: GetResourceReservationByOperationID :one
SELECT * FROM resource_reservations
WHERE operation_id = $1;

-- name: UpdateResourceReservationStatus :exec
UPDATE resource_reservations
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: ListExpiredReservations :many
SELECT * FROM resource_reservations
WHERE status = 'reserved' AND expires_at < now();
