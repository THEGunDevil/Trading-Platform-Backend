-- name: CreateOrder :one
INSERT INTO orders (user_id, symbol, side, order_type, leverage, price, quantity, margin, fee, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'open')
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: ListUserOrders :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListOpenOrdersByUser :many
SELECT * FROM orders WHERE user_id = $1 AND status = 'open' ORDER BY created_at DESC;

-- name: ListOpenOrdersBySymbol :many
-- For a matching engine scanning open orders on a pair
SELECT * FROM orders WHERE symbol = $1 AND status = 'open' ORDER BY created_at ASC;

-- name: MarkOrderFilled :one
UPDATE orders SET status = 'filled', filled_at = NOW() WHERE id = $1
RETURNING *;

-- name: CancelOrder :one
-- Only lets the owning user cancel their own still-open order
UPDATE orders SET status = 'cancelled' WHERE id = $1 AND user_id = $2 AND status = 'open'
RETURNING *;

-- name: RejectOrder :one
UPDATE orders SET status = 'rejected' WHERE id = $1
RETURNING *;