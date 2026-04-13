-- ─── organizations ───────────────────────────────────────────────────────────

-- name: CreateOrganization :exec
INSERT INTO organizations (id, name, slug, owner_user_id, plan, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetOrganizationByID :one
SELECT id, name, slug, owner_user_id, plan, created_at, updated_at, suspended_at
FROM organizations
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT id, name, slug, owner_user_id, plan, created_at, updated_at, suspended_at
FROM organizations
WHERE slug = $1;

-- name: UpdateOrganization :exec
UPDATE organizations
SET
    name       = $2,
    plan       = $3,
    updated_at = $4
WHERE id = $1;

-- ─── org_members ────────────────────────────────────────────────────────────

-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, user_id, role, invited_by, joined_at)
VALUES ($1, $2, $3, $4, $5);

-- name: RemoveOrgMember :exec
DELETE FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- name: GetOrgMember :one
SELECT org_id, user_id, role, invited_by, joined_at
FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- name: ListOrgMembers :many
SELECT org_id, user_id, role, invited_by, joined_at
FROM org_members
WHERE org_id = $1
ORDER BY joined_at;

-- ─── org_invitations ────────────────────────────────────────────────────────

-- name: CreateInvitation :exec
INSERT INTO org_invitations (id, org_id, email, role, token, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetInvitationByToken :one
SELECT id, org_id, email, role, token, expires_at, accepted_at, created_at
FROM org_invitations
WHERE token = $1;

-- name: MarkInvitationAccepted :exec
UPDATE org_invitations SET accepted_at = now()
WHERE id = $1;

-- ─── groups ─────────────────────────────────────────────────────────────────

-- name: CreateGroup :exec
INSERT INTO groups (id, org_id, app_id, name, description, ephemeral, expires_at, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetGroupByID :one
SELECT id, org_id, app_id, name, description, ephemeral, expires_at, created_by, created_at
FROM groups
WHERE id = $1;

-- name: ListGroupsByOrg :many
SELECT id, org_id, app_id, name, description, ephemeral, expires_at, created_by, created_at
FROM groups
WHERE org_id = $1
ORDER BY name;

-- name: ListExpiredGroups :many
SELECT id, org_id, app_id, name, description, ephemeral, expires_at, created_by, created_at
FROM groups
WHERE ephemeral = true AND expires_at IS NOT NULL AND expires_at < $1;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- ─── group_members ──────────────────────────────────────────────────────────

-- name: AddGroupMember :exec
INSERT INTO group_members (group_id, user_id, app_id) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveGroupMember :exec
DELETE FROM group_members WHERE group_id = $1 AND user_id = $2;

-- name: ListGroupMembers :many
SELECT group_id, user_id, app_id
FROM group_members
WHERE group_id = $1;

-- ─── roles ──────────────────────────────────────────────────────────────────

-- name: CreateRole :exec
INSERT INTO roles (id, org_id, name, permissions, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRoleByID :one
SELECT id, org_id, name, permissions, created_at
FROM roles
WHERE id = $1;

-- name: ListRolesByOrg :many
SELECT id, org_id, name, permissions, created_at
FROM roles
WHERE org_id = $1
ORDER BY name;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- ─── outbox ─────────────────────────────────────────────────────────────────

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
