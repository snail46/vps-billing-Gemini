-- name: CreateNodeGroup :one
INSERT INTO node_groups (
  id, name, region, status, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, now(), now()
)
RETURNING *;

-- name: GetNodeGroupByID :one
SELECT * FROM node_groups
WHERE id = $1;

-- name: ListNodeGroups :many
SELECT * FROM node_groups
ORDER BY name ASC;
