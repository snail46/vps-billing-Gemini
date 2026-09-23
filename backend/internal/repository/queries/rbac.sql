-- name: CreateRole :one
INSERT INTO roles (id, key, name_key, created_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (key) DO UPDATE SET name_key = EXCLUDED.name_key
RETURNING *;

-- name: GetRoleByKey :one
SELECT * FROM roles
WHERE key = $1 LIMIT 1;

-- name: CreatePermission :one
INSERT INTO permissions (id, key)
VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING
RETURNING *;

-- name: AssignRoleToAdmin :exec
INSERT INTO admin_roles (admin_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetAdminRoles :many
SELECT r.* FROM roles r
JOIN admin_roles ar ON ar.role_id = r.id
WHERE ar.admin_id = $1;

-- name: GetAdminPermissions :many
SELECT DISTINCT p.key FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN admin_roles ar ON ar.role_id = rp.role_id
WHERE ar.admin_id = $1;
