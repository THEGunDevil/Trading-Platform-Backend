
-- name: CreatePrediction :one
INSERT INTO predictions (
    user_id, coin_id, symbol, amount, direction, 
    duration_seconds, start_price, payout_rate, 
    status, expires_at
) VALUES (
    $1, $2, $3, $4, $5, 
    $6, $7, $8, 
    'active', NOW() + ($6 * INTERVAL '1 second')
) RETURNING *;

-- name: GetPredictionByID :one
SELECT * FROM predictions 
WHERE id = $1 AND user_id = $2;

-- name: GetActivePredictionsByUser :many
SELECT * FROM predictions 
WHERE user_id = $1 AND status = 'active' 
ORDER BY created_at DESC;

-- name: GetUserPredictions :many
SELECT * FROM predictions 
WHERE user_id = $1 
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: GetActivePredictions :many
SELECT * FROM predictions 
WHERE status = 'active' AND expires_at <= NOW();

-- name: ResolvePrediction :one
UPDATE predictions 
SET 
    status = $2,
    final_price = $3,
    profit = CASE WHEN $2 = 'won' THEN amount * payout_rate ELSE 0 END,
    payout = CASE WHEN $2 = 'won' THEN amount + (amount * payout_rate) ELSE 0 END,
    resolved_at = NOW()
WHERE id = $1 AND status = 'active' 
RETURNING *;

-- name: CancelPrediction :exec
UPDATE predictions 
SET status = 'cancelled', resolved_at = NOW() 
WHERE id = $1 AND user_id = $2 AND status = 'active';

-- name: ExpirePredictions :many
UPDATE predictions 
SET 
    status = 'expired',
    resolved_at = NOW()
WHERE status = 'active' AND expires_at <= NOW()
RETURNING *;

-- name: CountActivePredictionsByUser :one
SELECT COUNT(*) FROM predictions 
WHERE user_id = $1 AND status = 'active';

-- name: GetUserBalance :one
SELECT COALESCE(available, 0)::float8 as balance 
FROM balances 
WHERE user_id = $1 AND asset = 'USDT';

-- name: UpdateUserBalance :exec
INSERT INTO balances (user_id, asset, available) 
VALUES ($1, 'USDT', $2)
ON CONFLICT (user_id, asset) 
DO UPDATE SET available = balances.available + $2, updated_at = NOW();