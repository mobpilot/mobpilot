-- Migration: add ephemeral group support
-- Adds app_id, ephemeral, expires_at, created_by to groups and app_id to group_members.

-- The zero UUID defaults ('00000000-...') are intentional: this is a greenfield
-- deployment and no existing rows are present. The defaults satisfy the NOT NULL
-- constraint during the schema migration only; application code always supplies
-- real values. A stricter migration (UPDATE then DROP DEFAULT) is unnecessary
-- here because there is no existing data to back-fill.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS app_id     UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    ADD COLUMN IF NOT EXISTS ephemeral  BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid;

ALTER TABLE group_members
    ADD COLUMN IF NOT EXISTS app_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid;

CREATE INDEX IF NOT EXISTS idx_groups_app        ON groups (app_id, org_id);
CREATE INDEX IF NOT EXISTS idx_groups_expires_at ON groups (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_group_members_app ON group_members (app_id, group_id);
