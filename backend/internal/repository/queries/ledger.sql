-- name: CreateLedgerTransaction :one
INSERT INTO ledger_transactions (
    id, type, reference_type, reference_id, description, created_at
) VALUES (
    $1, $2, $3, $4, $5, now()
) RETURNING *;

-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (
    id, transaction_id, account_type, account_id, direction, amount_minor, currency, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, now()
) RETURNING *;

-- name: ListLedgerEntriesByTransactionID :many
SELECT * FROM ledger_entries
WHERE transaction_id = $1
ORDER BY created_at ASC;

-- name: ListLedgerTransactionsByReference :many
SELECT * FROM ledger_transactions
WHERE reference_type = $1 AND reference_id = $2
ORDER BY created_at ASC;

-- name: ListLedgerEntriesByAccount :many
SELECT * FROM ledger_entries
WHERE account_type = $1 AND account_id = $2
ORDER BY created_at DESC;

-- name: ListRecentLedgerTransactions :many
SELECT * FROM ledger_transactions
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
