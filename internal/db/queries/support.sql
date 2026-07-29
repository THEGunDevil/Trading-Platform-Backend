-- internal/db/queries/support.sql

-- name: CreateSupportSession :one
INSERT INTO support_sessions (user_id, subject, status)
VALUES ($1, $2, 'open')
RETURNING *;

-- name: GetSupportSessionByID :one
SELECT * FROM support_sessions WHERE id = $1;
-- name: GetSupportMessageByID :one
SELECT * FROM support_messages WHERE id = $1;
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
-- name: GetSessionWithUserByID :one
SELECT
    ss.id, ss.user_id, ss.subject, ss.status,
    ss.assigned_agent_id, ss.created_at, ss.updated_at,
    ss.assigned_at, ss.closed_at,
    u.user_name, u.email as user_email
FROM support_sessions ss
JOIN users u ON ss.user_id = u.id
WHERE ss.id = $1;
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

-- name: ListAgentUserIDs :many
SELECT id FROM users WHERE role = 'agent';

-- name: CreateSessionNotification :one
INSERT INTO session_notifications (session_id, agent_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetAgentNotifications :many
SELECT
    sn.id,
    sn.session_id,
    sn.agent_id,
    sn.created_at,
    ss.subject,
    ss.created_at AS session_created_at,
    u.user_name
FROM session_notifications sn
JOIN support_sessions ss ON ss.id = sn.session_id
JOIN users u ON u.id = ss.user_id
WHERE sn.agent_id = $1
  AND sn.is_read = false
  AND sn.is_expired = false
ORDER BY sn.created_at ASC;

-- name: MarkNotificationAsRead :exec
UPDATE session_notifications
SET is_read = true
WHERE id = $1 AND agent_id = $2;

-- name: ExpireAllNotifications :execrows
UPDATE session_notifications
SET is_expired = TRUE, is_read = TRUE
WHERE session_id = $1 AND is_expired = FALSE;


-- name: UpdateMessage :one
UPDATE support_messages
SET content = $2, image_url = $3
WHERE id = $1 AND sender_id = $4
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM support_messages
WHERE id = $1 AND sender_id = $2;