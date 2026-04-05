-- Identity service schema
-- Managed by Atlas; this file is the source of truth for sqlc code generation.

CREATE TABLE user_profiles (
    id           UUID        NOT NULL,
    app_id       UUID        NOT NULL,
    display_name TEXT        NOT NULL DEFAULT '',
    avatar_url   TEXT        NOT NULL DEFAULT '',
    bio          TEXT        NOT NULL DEFAULT '',
    location     TEXT        NOT NULL DEFAULT '',
    website_url  TEXT        NOT NULL DEFAULT '',
    settings     JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, app_id)
);

CREATE INDEX idx_user_profiles_app ON user_profiles (app_id);

CREATE TABLE device_tokens (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id    UUID        NOT NULL,
    app_id     UUID        NOT NULL,
    platform   TEXT        NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    token_type TEXT        NOT NULL CHECK (token_type IN ('fcm', 'apns')),
    token      TEXT        NOT NULL,
    device_id  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (app_id, device_id)
);

CREATE INDEX idx_device_tokens_user ON device_tokens (user_id, app_id);

CREATE TABLE artifact_storage (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id    UUID        NOT NULL,
    app_id     UUID        NOT NULL,
    key        TEXT        NOT NULL,
    value      JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, app_id, key)
);

CREATE INDEX idx_artifact_storage_user ON artifact_storage (user_id, app_id);

-- Transactional outbox: domain events are written here in the same transaction
-- as aggregate persistence; a River job polls and publishes to NATS.
CREATE TABLE outbox (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    aggregate_id UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
