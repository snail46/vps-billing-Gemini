-- name: CreateSubscription :one
INSERT INTO subscriptions (
    id, user_id, plan_id, status, billing_cycle, price_minor, currency,
    started_at, current_period_start, current_period_end, next_due_at, grace_until,
    cancel_at_period_end, ended_at, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, 1, now(), now()
) RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions
WHERE id = $1 LIMIT 1;

-- name: GetSubscriptionForUpdate :one
SELECT * FROM subscriptions
WHERE id = $1 LIMIT 1 FOR UPDATE;

-- name: ListSubscriptionsByUserID :many
SELECT * FROM subscriptions
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListAllSubscriptions :many
SELECT * FROM subscriptions
ORDER BY created_at DESC;

-- name: ListDueSubscriptions :many
SELECT * FROM subscriptions
WHERE status = 'active' AND next_due_at <= $1
ORDER BY next_due_at ASC
LIMIT $2;

-- name: ListGraceExpiredSubscriptions :many
SELECT * FROM subscriptions
WHERE status = 'past_due' AND grace_until IS NOT NULL AND grace_until <= $1
ORDER BY grace_until ASC
LIMIT $2;

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions
SET status = $2,
    ended_at = CASE WHEN $2 IN ('cancelled', 'expired', 'terminated') THEN now() ELSE ended_at END,
    version = version + 1,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RenewSubscription :one
UPDATE subscriptions
SET status = 'active',
    current_period_start = $2,
    current_period_end = $3,
    next_due_at = $4,
    grace_until = NULL,
    version = version + 1,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetCancelAtPeriodEnd :one
UPDATE subscriptions
SET cancel_at_period_end = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $1
RETURNING *;
