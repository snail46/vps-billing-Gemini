-- name: CreateInvoice :one
INSERT INTO invoices (
    id, invoice_no, user_id, subscription_id, order_id, status,
    amount_minor, currency, due_at, paid_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, now(), now()
) RETURNING *;

-- name: GetInvoiceByID :one
SELECT * FROM invoices
WHERE id = $1 LIMIT 1;

-- name: GetInvoiceByInvoiceNo :one
SELECT * FROM invoices
WHERE invoice_no = $1 LIMIT 1;

-- name: GetInvoiceByOrderID :one
SELECT * FROM invoices
WHERE order_id = $1 LIMIT 1;

-- name: ListInvoicesByUserID :many
SELECT * FROM invoices
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListAllInvoices :many
SELECT * FROM invoices
ORDER BY created_at DESC;

-- name: UpdateInvoiceStatus :one
UPDATE invoices
SET status = $2,
    paid_at = CASE WHEN $2 = 'paid' THEN now() ELSE paid_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateInvoiceItem :one
INSERT INTO invoice_items (
    id, invoice_id, description_i18n, quantity, unit_amount_minor, total_minor, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
) RETURNING *;

-- name: ListInvoiceItemsByInvoiceID :many
SELECT * FROM invoice_items
WHERE invoice_id = $1
ORDER BY created_at ASC;
