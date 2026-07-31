-- name: CreateUser :one
INSERT INTO users (user_name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserNameByID :one
SELECT user_name FROM users WHERE id = $1;
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserProfile :one
-- UPDATE users
-- SET user_name = $2, updated_at = NOW()
-- WHERE id = $1
-- RETURNING *;
-- name: UpdateUserByID :one
UPDATE users
SET
  user_name            = COALESCE(sqlc.narg(user_name), user_name),
  updated_at            = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = $2, token_version = token_version + 1, updated_at = NOW()
WHERE id = $1;
-- name: UpdateUserRole :exec
UPDATE users 
SET role = $2, updated_at = NOW()
WHERE id = $1;
-- name: IncrementTokenVersion :exec
-- Invalidates all existing refresh tokens for a user (e.g. on logout-everywhere or password change)
UPDATE users SET token_version = token_version + 1 WHERE id = $1;

-- name: BanUser :exec
UPDATE users
SET is_banned = true, ban_reason = $2, ban_until = $3, is_permanent_ban = $4
WHERE id = $1;

-- name: UnbanUser :exec
UPDATE users
SET is_banned = false, ban_reason = NULL, ban_until = NULL, is_permanent_ban = false
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;
-- name: CountUsersByEmail :one
SELECT COUNT(*)
FROM users
WHERE
    (CASE 
        WHEN $1 = '' THEN TRUE
        ELSE email ILIKE '%' || $1 || '%'
    END);


-- name: CountUsers :one
SELECT COUNT(*) FROM users;
-- name: ListUsersPaginated :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBannedUsersPaginated :many
SELECT id, user_name, email, role, is_banned, is_permanent_ban, ban_reason, ban_until, created_at, token_version
FROM users
WHERE is_banned = TRUE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBannedUsers :one
SELECT COUNT(*) FROM users WHERE is_banned = TRUE;