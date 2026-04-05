-- ─── user_profiles ───────────────────────────────────────────────────────────

-- name: UpsertUserProfile :exec
INSERT INTO user_profiles (id, app_id, display_name, avatar_url, bio, location, website_url, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id, app_id)
DO UPDATE SET
    display_name = EXCLUDED.display_name,
    avatar_url   = EXCLUDED.avatar_url,
    bio          = EXCLUDED.bio,
    location     = EXCLUDED.location,
    website_url  = EXCLUDED.website_url,
    settings     = EXCLUDED.settings,
    updated_at   = EXCLUDED.updated_at;

-- name: UpdateUserProfile :exec
UPDATE user_profiles
SET
    display_name = $3,
    avatar_url   = $4,
    bio          = $5,
    location     = $6,
    website_url  = $7,
    settings     = $8,
    updated_at   = $9
WHERE id = $1 AND app_id = $2;

-- name: GetUserProfile :one
SELECT id, app_id, display_name, avatar_url, bio, location, website_url, settings, created_at, updated_at
FROM user_profiles
WHERE id = $1 AND app_id = $2;

-- ─── device_tokens ───────────────────────────────────────────────────────────

-- name: UpsertDeviceToken :exec
INSERT INTO device_tokens (id, user_id, app_id, platform, token_type, token, device_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (app_id, device_id)
DO UPDATE SET
    user_id    = EXCLUDED.user_id,
    platform   = EXCLUDED.platform,
    token_type = EXCLUDED.token_type,
    token      = EXCLUDED.token,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens
WHERE app_id = $1 AND device_id = $2;

-- name: GetDeviceTokensByUser :many
SELECT id, user_id, app_id, platform, token_type, token, device_id, created_at, updated_at
FROM device_tokens
WHERE user_id = $1 AND app_id = $2;

-- ─── artifact_storage ────────────────────────────────────────────────────────

-- name: UpsertArtifact :exec
INSERT INTO artifact_storage (id, user_id, app_id, key, value, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, app_id, key)
DO UPDATE SET
    value      = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;

-- name: GetArtifact :one
SELECT id, user_id, app_id, key, value, created_at, updated_at
FROM artifact_storage
WHERE user_id = $1 AND app_id = $2 AND key = $3;

-- name: ListArtifacts :many
SELECT id, user_id, app_id, key, value, created_at, updated_at
FROM artifact_storage
WHERE user_id = $1 AND app_id = $2
ORDER BY key;

-- name: DeleteArtifact :exec
DELETE FROM artifact_storage
WHERE user_id = $1 AND app_id = $2 AND key = $3;

-- ─── outbox ──────────────────────────────────────────────────────────────────

-- name: InsertOutboxEvent :exec
INSERT INTO outbox (aggregate_id, event_type, payload)
VALUES ($1, $2, $3);

-- name: GetUnpublishedOutboxEvents :many
SELECT id, aggregate_id, event_type, payload, created_at
FROM outbox
WHERE published_at IS NULL
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox SET published_at = now()
WHERE id = $1;
