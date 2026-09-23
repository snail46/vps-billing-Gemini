-- name: CreateOrder :one
INSERT INTO orders (
    id, order_no, user_id, status, subtotal_minor, discount_minor, total_minor, currency, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, now(), now()
) RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: GetOrderByOrderNo :one
SELECT * FROM orders
WHERE order_no = $1 LIMIT 1;

-- name: ListOrdersByUserID :many
SELECT * FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListAllOrders :many
SELECT * FROM orders
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    paid_at = CASE WHEN $2 = 'paid' THEN now() ELSE paid_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    id, order_id, product_id, plan_id, quantity, unit_price_minor, total_minor,
    product_snapshot, plan_snapshot, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, now()
) RETURNING *;

-- name: ListOrderItemsByOrderID :many
SELECT * FROM order_items
WHERE order_id = $1
ORDER BY created_at ASC;
