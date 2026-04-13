// Atlas declarative schema for the org service.
// Run: atlas schema apply --env org
// Diff: atlas schema diff --env org

schema "org" {}

table "organizations" {
  schema = schema.org
  column "id"            { type = uuid }
  column "name"          { type = text }
  column "slug"          { type = text }
  column "owner_user_id" { type = uuid }
  column "plan"          { type = text    default = "free" }
  column "created_at"    { type = timestamptz  default = sql("now()") }
  column "updated_at"    { type = timestamptz  default = sql("now()") }
  column "suspended_at"  { type = timestamptz  null = true }

  primary_key { columns = [column.id] }
  unique "uq_organizations_slug" { columns = [column.slug] }
}

table "org_members" {
  schema = schema.org
  column "org_id"      { type = uuid }
  column "user_id"     { type = uuid }
  column "role"        { type = text }
  column "invited_by"  { type = uuid    null = true }
  column "joined_at"   { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.org_id, column.user_id] }
  index "idx_org_members_org" { columns = [column.org_id] }

  check "chk_role" { expr = "role IN ('owner', 'admin', 'member')" }
}

table "org_invitations" {
  schema = schema.org
  column "id"          { type = uuid    default = sql("gen_random_uuid()") }
  column "org_id"      { type = uuid }
  column "email"       { type = text }
  column "role"        { type = text }
  column "token"       { type = text }
  column "expires_at"  { type = timestamptz }
  column "accepted_at" { type = timestamptz  null = true }
  column "created_at"  { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id] }
  unique "uq_org_invitations_token" { columns = [column.token] }

  check "chk_role" { expr = "role IN ('owner', 'admin', 'member')" }
}

table "groups" {
  schema = schema.org
  column "id"          { type = uuid         default = sql("gen_random_uuid()") }
  column "org_id"      { type = uuid }
  column "app_id"      { type = uuid }
  column "name"        { type = text }
  column "description" { type = text         default = "" }
  column "ephemeral"   { type = boolean      default = false }
  column "expires_at"  { type = timestamptz  null = true }
  column "created_by"  { type = uuid }
  column "created_at"  { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id] }
  index "idx_groups_org"        { columns = [column.org_id] }
  index "idx_groups_app"        { columns = [column.app_id, column.org_id] }
  index "idx_groups_expires_at" {
    columns = [column.expires_at]
    where   = "expires_at IS NOT NULL"
  }
}

table "group_members" {
  schema = schema.org
  column "group_id" { type = uuid }
  column "user_id"  { type = uuid }
  column "app_id"   { type = uuid }

  primary_key { columns = [column.group_id, column.user_id] }
  index "idx_group_members_app" { columns = [column.app_id, column.group_id] }
}

table "roles" {
  schema = schema.org
  column "id"          { type = uuid    default = sql("gen_random_uuid()") }
  column "org_id"      { type = uuid }
  column "name"        { type = text }
  column "permissions" { type = sql("text[]") }
  column "created_at"  { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id] }
  index "idx_roles_org" { columns = [column.org_id] }
}

table "outbox" {
  schema = schema.org
  column "id"           { type = uuid    default = sql("gen_random_uuid()") }
  column "aggregate_id" { type = uuid }
  column "event_type"   { type = text }
  column "payload"      { type = jsonb }
  column "created_at"   { type = timestamptz default = sql("now()") }
  column "published_at" { type = timestamptz null = true }

  primary_key { columns = [column.id] }
  index "idx_outbox_unpublished" {
    columns = [column.created_at]
    where   = "published_at IS NULL"
  }
}
