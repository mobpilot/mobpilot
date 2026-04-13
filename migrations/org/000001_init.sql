-- Atlas managed migration: 000001_init
-- Org service initial schema

CREATE SCHEMA IF NOT EXISTS org;

SET search_path = org;

CREATE TABLE organizations (
    id            UUID        NOT NULL PRIMARY KEY,
    name          TEXT        NOT NULL,
    slug          TEXT        NOT NULL,
    owner_user_id UUID        NOT NULL,
    plan          TEXT        NOT NULL DEFAULT 'free',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    suspended_at  TIMESTAMPTZ,
    UNIQUE (slug)
);

CREATE TABLE org_members (
    org_id     UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    role       TEXT        NOT NULL,
    invited_by UUID,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id),
    CONSTRAINT chk_role CHECK (role IN ('owner', 'admin', 'member'))
);

CREATE INDEX idx_org_members_org ON org_members (org_id);

CREATE TABLE org_invitations (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    email       TEXT        NOT NULL,
    role        TEXT        NOT NULL,
    token       TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_role CHECK (role IN ('owner', 'admin', 'member')),
    UNIQUE (token)
);

CREATE TABLE groups (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_groups_org ON groups (org_id);

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id  UUID NOT NULL,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE roles (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    name        TEXT        NOT NULL,
    permissions TEXT[]      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_roles_org ON roles (org_id);

CREATE TABLE outbox (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    aggregate_id UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
