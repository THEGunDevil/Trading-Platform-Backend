-- internal/db/queries/admin.sql

-- name: GetAllUsersWithDetails :many
SELECT 
    id, user_name, email, role, 
    is_banned, is_permanent_ban, ban_reason, ban_until,
    created_at, token_version
FROM users
ORDER BY created_at DESC;

-- name: GetPlatformBalances :many
SELECT 
    b.asset,
    SUM(b.available)::FLOAT8 AS available,
    SUM(b.locked)::FLOAT8 AS locked
FROM balances b
GROUP BY b.asset
ORDER BY b.asset;

-- name: GetRecentTrades :many
SELECT 
    t.id, t.symbol, o.side, t.price, t.quantity, t.fee, t.executed_at AS created_at
FROM trades t
JOIN orders o ON t.order_id = o.id
ORDER BY t.executed_at DESC
LIMIT 50;

-- name: GetActiveSupportSessions :many
SELECT 
    ss.id, u.user_name, ss.subject, ss.status, 
    ss.assigned_agent_id, ss.created_at
FROM support_sessions ss
JOIN users u ON ss.user_id = u.id
WHERE ss.status IN ('open', 'assigned', 'in_progress')
ORDER BY ss.created_at DESC;

-- name: GetBalancesForUsers :many
SELECT user_id, asset, available, locked FROM balances WHERE user_id = ANY($1::uuid[]);


-- name: ListAgentUsersPaginated :many
SELECT id, user_name, email, role, is_banned, is_permanent_ban, ban_reason, ban_until, created_at, token_version
FROM users
WHERE role = 'agent'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAgentUsers :one
SELECT COUNT(*) FROM users WHERE role = 'agent';



-- name: GetPlatformSetting :one
SELECT value FROM platform_settings WHERE key = $1;

-- name: UpsertPlatformSetting :exec
INSERT INTO platform_settings (key, value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key)
DO UPDATE SET value = $2, updated_at = NOW();

-- name: UpdateUserWillProfit :exec
UPDATE users 
SET will_profit = $2, updated_at = NOW()
WHERE id = $1;
-- name: GetUserWillProfit :one

SELECT will_profit FROM users WHERE id = $1;

-- name: SearchUsers :many
SELECT id, user_name, email, role, is_banned, is_permanent_ban, ban_reason, ban_until, created_at, token_version, will_profit
FROM users
WHERE user_name ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%'
   OR role ILIKE '%' || $1 || '%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;