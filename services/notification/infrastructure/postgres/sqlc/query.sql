-- ─── notifications ────────────────────────────────────────────────────────────

-- name: InsertNotification :exec
INSERT INTO notifications (id, user_id, app_id, group_id, org_id, title, body, data, delivery_status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetNotificationByID :one
SELECT id, user_id, app_id, group_id, org_id, title, body, data, delivery_status, read_at, created_at
FROM notifications
WHERE id = $1;

-- name: ListNotificationsForUser :many
SELECT id, user_id, app_id, group_id, org_id, title, body, data, delivery_status, read_at, created_at
FROM notifications
WHERE user_id = $1 AND app_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateNotificationStatus :exec
UPDATE notifications SET delivery_status = $2 WHERE id = $1;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read_at = now() WHERE id = $1 AND read_at IS NULL;

-- ─── notification_preferences ─────────────────────────────────────────────────

-- name: UpsertNotificationPreference :exec
INSERT INTO notification_preferences (user_id, app_id, channel, event_type, enabled, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (user_id, app_id, channel, event_type)
DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now();

-- name: GetNotificationPreferences :many
SELECT user_id, app_id, channel, event_type, enabled, updated_at
FROM notification_preferences
WHERE user_id = $1 AND app_id = $2;

-- name: GetPreferenceEnabled :one
SELECT enabled FROM notification_preferences
WHERE user_id = $1 AND app_id = $2 AND channel = $3
  AND (event_type = $4 OR event_type = '*')
ORDER BY event_type DESC  -- exact match wins over wildcard
LIMIT 1;
