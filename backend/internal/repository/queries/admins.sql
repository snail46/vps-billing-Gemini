-- name: CreateAdmin :one
INSERT INTO admins (
    id, email, password_hash, status, display_name, two_factor_enabled, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now(), now()
) RETURNING *;

-- name: GetAdminByID :one
SELECT * FROM admins
WHERE id = $1 LIMIT 1;

-- name: GetAdminByEmail :one
SELECT * FROM admins
WHERE email = $1 LIMIT 1;

-- name: UpdateAdminStatus :one
UPDATE admins
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateAdminLastLogin :exec
UPDATE admins
SET last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateAdminTwoFactor :exec
UPDATE admins
SET two_factor_enabled = $2, two_factor_secret = $3, updated_at = now()
WHERE id = $1;

-- name: ListAdmins :many
SELECT * FROM admins
ORDER BY created_at DESC;

