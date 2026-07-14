-- name: CreateSupportConversation :one
INSERT INTO support_conversations (user_id, status)
VALUES ($1, 'open')
RETURNING *;

-- name: GetOpenConversationForUser :one
SELECT * FROM support_conversations WHERE user_id = $1 AND status = 'open'
ORDER BY created_at DESC LIMIT 1;

-- name: ListUserConversations :many
SELECT * FROM support_conversations WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CloseConversation :exec
UPDATE support_conversations SET status = 'closed', updated_at = NOW() WHERE id = $1;

-- name: CreateSupportMessage :one
INSERT INTO support_messages (conversation_id, sender, body, image_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListConversationMessages :many
SELECT * FROM support_messages WHERE conversation_id = $1 ORDER BY created_at ASC;

-- name: TouchConversation :exec
-- Bump updated_at whenever a new message lands, so conversations sort by recent activity
UPDATE support_conversations SET updated_at = NOW() WHERE id = $1;