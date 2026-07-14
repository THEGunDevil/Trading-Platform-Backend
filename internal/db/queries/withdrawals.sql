-- name: CreateWithdrawal :one
INSERT INTO withdrawals (user_id, asset, network, destination_address, amount, fee, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending')
RETURNING *;

-- name: GetWithdrawalByID :one
SELECT * FROM withdrawals WHERE id = $1;

-- name: ListUserWithdrawals :many
SELECT * FROM withdrawals WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateWithdrawalStatus :one
UPDATE withdrawals SET status = $2 WHERE id = $1
RETURNING *;

-- name: MarkWithdrawalCompleted :one
UPDATE withdrawals SET status = 'completed', tx_hash = $2, completed_at = NOW() WHERE id = $1
RETURNING *;

-- name: MarkWithdrawalRejected :one
UPDATE withdrawals SET status = 'rejected' WHERE id = $1
RETURNING *;

-- name: ListPendingWithdrawals :many
SELECT * FROM withdrawals WHERE status = 'pending' ORDER BY created_at ASC;