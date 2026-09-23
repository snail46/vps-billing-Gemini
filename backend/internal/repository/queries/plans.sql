-- name: CreatePlan :one
INSERT INTO plans (
    id, product_id, node_group_id, slug, name_i18n, status,
    cpu_cores, memory_mb, disk_gb, traffic_gb, bandwidth_mbps,
    ipv4_count, ipv6_count, nat_port_count, virtualization,
    billing_cycle, price_minor, currency, stock_mode,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17, $18, $19,
    now(), now()
) RETURNING *;

-- name: GetPlanByID :one
SELECT * FROM plans
WHERE id = $1 LIMIT 1;

-- name: GetPlanByProductAndSlug :one
SELECT * FROM plans
WHERE product_id = $1 AND slug = $2 LIMIT 1;

-- name: ListPlansByProductID :many
SELECT * FROM plans
WHERE product_id = $1
ORDER BY price_minor ASC, created_at ASC;

-- name: ListActivePlansByProductID :many
SELECT * FROM plans
WHERE product_id = $1 AND status = 'active'
ORDER BY price_minor ASC, created_at ASC;

-- name: ListAllActivePlans :many
SELECT * FROM plans
WHERE status = 'active'
ORDER BY price_minor ASC, created_at ASC;

-- name: UpdatePlan :one
UPDATE plans
SET name_i18n = $2,
    status = $3,
    cpu_cores = $4,
    memory_mb = $5,
    disk_gb = $6,
    traffic_gb = $7,
    bandwidth_mbps = $8,
    ipv4_count = $9,
    ipv6_count = $10,
    nat_port_count = $11,
    price_minor = $12,
    currency = $13,
    stock_mode = $14,
    updated_at = now()
WHERE id = $1
RETURNING *;
