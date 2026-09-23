-- name: CreatePayment :one
INSERT INTO payments (
    id, payment_no, order_id, gateway, gateway_payment_id, status,
    amount_minor, currency, idempotency_key, gateway_payload, paid_at,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    now(), now()
) RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1 LIMIT 1;

-- name: GetPaymentForUpdate :one
SELECT * FROM payments
WHERE id = $1 LIMIT 1 FOR UPDATE;

-- name: GetPaymentByPaymentNo :one
SELECT * FROM payments
WHERE payment_no = $1 LIMIT 1;

-- name: GetPaymentByIdempotencyKey :one
SELECT * FROM payments
WHERE idempotency_key = $1 LIMIT 1;

-- name: GetPaymentByIdempotencyKeyForUpdate :one
SELECT * FROM payments
WHERE idempotency_key = $1 LIMIT 1 FOR UPDATE;

-- name: GetPaymentByGatewayAndExternalID :one
SELECT * FROM payments
WHERE gateway = $1 AND gateway_payment_id = $2 LIMIT 1;

-- name: GetPaymentByGatewayAndExternalIDForUpdate :one
SELECT * FROM payments
WHERE gateway = $1 AND gateway_payment_id = $2 LIMIT 1 FOR UPDATE;

-- name: ListPaymentsByOrderID :many
SELECT * FROM payments
WHERE order_id = $1
ORDER BY created_at DESC;

-- name: ListAllPayments :many
SELECT * FROM payments
ORDER BY created_at DESC;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status = $2,
    gateway_payment_id = COALESCE($3, gateway_payment_id),
    gateway_payload = COALESCE($4, gateway_payload),
    paid_at = CASE WHEN $2 = 'succeeded' THEN now() ELSE paid_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;
