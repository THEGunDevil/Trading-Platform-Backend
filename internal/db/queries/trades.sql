-- name: CreateTrade :one
INSERT INTO trades (order_id, user_id, symbol, price, quantity, fee)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListTradesByOrder :many
SELECT * FROM trades WHERE order_id = $1 ORDER BY executed_at ASC;

-- name: ListUserTrades :many
SELECT * FROM trades WHERE user_id = $1 ORDER BY executed_at DESC LIMIT $2 OFFSET $3;

-- name: ListTradesBySymbol :many
-- Recent trade feed for a given pair, e.g. a "recent trades" ticker
SELECT * FROM trades WHERE symbol = $1 ORDER BY executed_at DESC LIMIT $2;