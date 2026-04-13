-- Atlas managed migration: 000001_init
-- Appstore service initial schema

CREATE SCHEMA IF NOT EXISTS appstore;

SET search_path = appstore;

CREATE TABLE apps (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id           UUID        NOT NULL,
    name             TEXT        NOT NULL,
    slug             TEXT        NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    icon_url         TEXT        NOT NULL DEFAULT '',
    enabled_features TEXT[]      NOT NULL DEFAULT '{}',
    config           JSONB       NOT NULL DEFAULT '{}',
    push_config      JSONB       NOT NULL DEFAULT '{}',
    db_schema        TEXT        NOT NULL DEFAULT '',
    status           TEXT        NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_status CHECK (status IN ('active', 'suspended', 'deleted')),
    UNIQUE (slug)
);

CREATE INDEX idx_apps_org    ON apps (org_id);
CREATE INDEX idx_apps_status ON apps (status);

CREATE TABLE outbox (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    aggregate_id UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
