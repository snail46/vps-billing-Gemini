-- name: CreateWallet :one
INSERT INTO wallets (
    id, user_id, currency, available_balance_minor, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, now(), now()
) RETURNING *;

-- name: GetWalletByUserID :one
SELECT * FROM wallets
WHERE user_id = $1 AND currency = $2 LIMIT 1;

-- name: GetWalletByUserIDForUpdate :one
SELECT * FROM wallets
WHERE user_id = $1 AND currency = $2 LIMIT 1 FOR UPDATE;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET available_balance_minor = $3,
    updated_at = now()
WHERE user_id = $1 AND currency = $2
RETURNING *;
