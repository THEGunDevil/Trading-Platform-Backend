-- internal/db/queries/support.sql

-- name: CreateSupportSession :one
INSERT INTO support_sessions (user_id, subject, status)
VALUES ($1, $2, 'open')
RETURNING *;

-- name: GetSupportSessionByID :one
SELECT * FROM support_sessions WHERE id = $1;

-- name: GetOpenSessionForUser :one
SELECT * FROM support_sessions
WHERE user_id = $1 AND status IN ('open', 'assigned')
ORDER BY created_at DESC LIMIT 1;

-- name: GetAvailableSessions :many
SELECT ss.*, u.user_name, u.email as user_email
FROM support_sessions ss
JOIN users u ON ss.user_id = u.id
WHERE ss.status = 'open' AND ss.assigned_agent_id IS NULL
ORDER BY ss.created_at ASC;

-- name: AssignAgentToSession :one
UPDATE support_sessions
SET assigned_agent_id = $1,
    status = 'assigned',
    assigned_at = NOW(),
    updated_at = NOW()
WHERE id = $2 AND status = 'open'
RETURNING *;

-- name: SendSupportMessage :one
INSERT INTO support_messages (session_id, sender_id, content, image_url, is_agent)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListSessionMessages :many
SELECT * FROM support_messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: CreateSessionNotification :one
INSERT INTO session_notifications (session_id, agent_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetAgentNotifications :many
SELECT sn.*, ss.user_id, u.user_name, u.email as user_email,
       ss.subject, ss.created_at as session_created_at
FROM session_notifications sn
JOIN support_sessions ss ON sn.session_id = ss.id
JOIN users u ON ss.user_id = u.id
WHERE sn.agent_id = $1
  AND sn.is_read = FALSE
  AND sn.is_expired = FALSE
ORDER BY ss.created_at DESC;

-- name: MarkNotificationAsRead :exec
UPDATE session_notifications
SET is_read = TRUE
WHERE id = $1 AND agent_id = $2;

-- name: ExpireAllNotifications :execrows
UPDATE session_notifications
SET is_expired = TRUE, is_read = TRUE
WHERE session_id = $1 AND is_expired = FALSE;

-- name: ListAllSessionsWithUser :many
SELECT 
    ss.id, ss.user_id, ss.subject, ss.status,
    ss.assigned_agent_id, ss.created_at, ss.updated_at,
    ss.assigned_at, ss.closed_at,
    u.user_name, u.email as user_email
FROM support_sessions ss
JOIN users u ON ss.user_id = u.id
ORDER BY ss.created_at DESC;

-- name: CloseSession :exec
UPDATE support_sessions
SET status = 'closed', closed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'assigned';