-- Notification service schema
-- Managed by Atlas; this file is the source of truth for sqlc code generation.

CREATE TABLE notifications (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id         UUID        NOT NULL,
    app_id          UUID        NOT NULL,
    group_id        UUID,
    org_id          UUID,
    title           TEXT        NOT NULL,
    body            TEXT        NOT NULL,
    data            JSONB       NOT NULL DEFAULT '{}',
    delivery_status TEXT        NOT NULL DEFAULT 'pending',
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications (user_id, app_id, created_at DESC);

CREATE TABLE notification_preferences (
    user_id    UUID    NOT NULL,
    app_id     UUID    NOT NULL,
    channel    TEXT    NOT NULL,
    event_type TEXT    NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, app_id, channel, event_type)
);
