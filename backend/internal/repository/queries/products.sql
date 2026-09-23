-- name: CreateProduct :one
INSERT INTO products (
    id, slug, name_i18n, description_i18n, status, sort_order, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now(), now()
) RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1 LIMIT 1;

-- name: GetProductBySlug :one
SELECT * FROM products
WHERE slug = $1 LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
ORDER BY sort_order ASC, created_at DESC;

-- name: ListActiveProducts :many
SELECT * FROM products
WHERE status = 'active'
ORDER BY sort_order ASC, created_at DESC;

-- name: UpdateProduct :one
UPDATE products
SET name_i18n = $2,
    description_i18n = $3,
    status = $4,
    sort_order = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;
