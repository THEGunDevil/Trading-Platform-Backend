-- name: CreateDepositAddress :one
INSERT INTO deposit_addresses (user_id, asset, network, address)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, asset, network) DO UPDATE SET address = EXCLUDED.address
RETURNING *;

-- name: GetDepositAddress :one
SELECT * FROM deposit_addresses WHERE user_id = $1 AND asset = $2 AND network = $3;

-- name: CreateDeposit :one
INSERT INTO deposits (user_id, asset, network, amount, tx_hash, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING *;

-- name: GetDepositByID :one
SELECT * FROM deposits WHERE id = $1;

-- name: ListUserDeposits :many
SELECT * FROM deposits WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateDepositConfirmations :one
UPDATE deposits SET confirmations = $2 WHERE id = $1
RETURNING *;

-- name: MarkDepositConfirmed :one
UPDATE deposits SET status = 'confirmed', confirmed_at = NOW() WHERE id = $1
RETURNING *;

-- name: MarkDepositFailed :one
UPDATE deposits SET status = 'failed' WHERE id = $1
RETURNING *;

-- name: ListPendingDeposits :many
-- For a background worker polling chain confirmations
SELECT * FROM deposits WHERE status = 'pending' ORDER BY created_at ASC;