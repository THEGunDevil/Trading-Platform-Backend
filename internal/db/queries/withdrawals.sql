-- name: CreateWithdrawal :one
INSERT INTO withdrawals (user_id, asset, network, destination_address, amount, fee, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending')
RETURNING *;

-- name: GetWithdrawalByID :one
SELECT * FROM withdrawals WHERE id = $1;

-- name: SearchWithdrawalsByUser :many
SELECT w.*
FROM withdrawals w
JOIN users u ON w.user_id = u.id
WHERE u.user_name ILIKE '%' || $1 || '%'
   OR u.email ILIKE '%' || $1 || '%'
ORDER BY w.created_at DESC;

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
SELECT * FROM withdrawals WHERE status = 'pending' ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: ListCompletedWithdrawals :many
SELECT * FROM withdrawals WHERE status = 'completed' ORDER BY created_at ASC LIMIT $1 OFFSET $2;

-- name: ListRejectedWithdrawals :many
SELECT * FROM withdrawals WHERE status = 'rejected' ORDER BY created_at ASC LIMIT $1 OFFSET $2;

-- name: CountPendingWithdrawals :one
SELECT COUNT(*) FROM withdrawals WHERE status = 'pending';
-- name: CountCompletedWithdrawals :one
SELECT COUNT(*) FROM withdrawals WHERE status = 'completed';
-- name: CountRejectedWithdrawals :one
SELECT COUNT(*) FROM withdrawals WHERE status = 'rejected';