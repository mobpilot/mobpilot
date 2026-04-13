-- Org service schema
-- Managed by Atlas; this file is the source of truth for sqlc code generation.

CREATE TABLE organizations (
    id            UUID        NOT NULL PRIMARY KEY,
    name          TEXT        NOT NULL,
    slug          TEXT        NOT NULL UNIQUE,
    owner_user_id UUID        NOT NULL,
    plan          TEXT        NOT NULL DEFAULT 'free',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    suspended_at  TIMESTAMPTZ
);

CREATE TABLE org_members (
    org_id     UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    role       TEXT        NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    invited_by UUID,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX idx_org_members_org ON org_members (org_id);

CREATE TABLE org_invitations (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    email       TEXT        NOT NULL,
    role        TEXT        NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    token       TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    app_id      UUID        NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    ephemeral   BOOLEAN     NOT NULL DEFAULT false,
    expires_at  TIMESTAMPTZ,
    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_groups_org ON groups (org_id);
CREATE INDEX idx_groups_app ON groups (app_id, org_id);
CREATE INDEX idx_groups_expires_at ON groups (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id  UUID NOT NULL,
    app_id   UUID NOT NULL,
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_app ON group_members (app_id, group_id);

CREATE TABLE roles (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    org_id      UUID        NOT NULL,
    name        TEXT        NOT NULL,
    permissions TEXT[]      NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_roles_org ON roles (org_id);

-- Transactional outbox: domain events are written here in the same transaction
-- as aggregate persistence; a background worker polls and publishes to NATS.
CREATE TABLE outbox (
    id           UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    aggregate_id UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
