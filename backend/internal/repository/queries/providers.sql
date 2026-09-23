-- name: CreateProvider :one
INSERT INTO providers (
  id, name, provider_type, endpoint, credential_ref, status, version, config, capabilities, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now()
)
RETURNING *;

-- name: GetProviderByID :one
SELECT * FROM providers
WHERE id = $1;

-- name: ListProviders :many
SELECT * FROM providers
ORDER BY created_at ASC;

-- name: UpdateProviderHealth :exec
UPDATE providers
SET status = $2, last_health_check_at = now(), updated_at = now()
WHERE id = $1;
