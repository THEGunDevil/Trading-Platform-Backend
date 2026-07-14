-- name: CreateEvent :one
INSERT INTO events (object_id, object_title, type, title, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: CreateNotificationForUser :one
INSERT INTO user_notification_status (user_id, event_id)
VALUES ($1, $2)
RETURNING *;

-- name: ListUserNotifications :many
SELECT e.*, uns.is_read, uns.read_at
FROM user_notification_status uns
JOIN events e ON e.id = uns.event_id
WHERE uns.user_id = $1
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM user_notification_status WHERE user_id = $1 AND is_read = false;

-- name: MarkNotificationRead :exec
UPDATE user_notification_status SET is_read = true, read_at = NOW()
WHERE user_id = $1 AND event_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE user_notification_status SET is_read = true, read_at = NOW()
WHERE user_id = $1 AND is_read = false;