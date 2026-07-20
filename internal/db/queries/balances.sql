-- name: GetBalance :one
SELECT * FROM balances WHERE user_id = $1 AND asset = $2;

-- name: ListBalances :many
SELECT * FROM balances WHERE user_id = $1 ORDER BY asset;

-- name: UpsertBalance :one
-- Creates a zero-balance row on first touch, otherwise no-ops (use Increase/Decrease below to mutate)
INSERT INTO balances (user_id, asset, available, locked)
VALUES ($1, $2, 0, 0)
ON CONFLICT (user_id, asset) DO UPDATE SET updated_at = balances.updated_at
RETURNING *;

-- name: IncreaseAvailableBalance :one
-- Use inside a transaction: e.g. crediting a confirmed deposit
UPDATE balances
SET available = available + $3, updated_at = NOW()
WHERE user_id = $1 AND asset = $2
RETURNING *;

-- name: DecreaseAvailableBalance :one
-- Fails (0 rows) if it would go negative, thanks to the CHECK constraint —
-- calling code must check rows-affected / returned row.
UPDATE balances
SET available = available - $3, updated_at = NOW()
WHERE user_id = $1 AND asset = $2 AND available >= $3
RETURNING *;

-- name: LockBalance :execrows
UPDATE balances
SET available = available - sqlc.arg(amount),
    locked = locked + sqlc.arg(amount)
WHERE user_id = sqlc.arg(user_id)
  AND asset = sqlc.arg(asset)
  AND available >= sqlc.arg(amount);

-- name: UnlockBalance :one
-- Moves funds from locked -> available (e.g. order cancelled)
UPDATE balances
SET locked = locked - $3, available = available + $3, updated_at = NOW()
WHERE user_id = $1 AND asset = $2 AND locked >= $3
RETURNING *;

-- name: ConsumeLockedBalance :one
-- Removes funds from locked entirely (e.g. order filled, funds spent)
UPDATE balances
SET locked = locked - $3, updated_at = NOW()
WHERE user_id = $1 AND asset = $2 AND locked >= $3
RETURNING *;
-- name: ForfeitLocked :one
-- Clears locked funds WITHOUT crediting available — used when a bet is lost
-- and the stake should not come back to the user.
UPDATE balances
SET locked = locked - sqlc.arg(amount),
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND asset = sqlc.arg(asset) AND locked >= sqlc.arg(amount)
RETURNING *;