// Atlas declarative schema for the appstore service.
// Run: atlas schema apply --env appstore
// Diff: atlas schema diff --env appstore

schema "appstore" {}

table "apps" {
  schema = schema.appstore
  column "id"               { type = uuid    default = sql("gen_random_uuid()") }
  column "org_id"           { type = uuid }
  column "name"             { type = text }
  column "slug"             { type = text }
  column "description"      { type = text    default = "" }
  column "icon_url"         { type = text    default = "" }
  column "enabled_features" { type = sql("text[]")  default = sql("'{}'") }
  column "config"           { type = jsonb   default = "{}" }
  column "push_config"      { type = jsonb   default = "{}" }
  column "db_schema"        { type = text    default = "" }
  column "status"           { type = text    default = "active" }
  column "created_at"       { type = timestamptz  default = sql("now()") }
  column "updated_at"       { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id] }
  unique "uq_apps_slug"       { columns = [column.slug] }
  index  "idx_apps_org"       { columns = [column.org_id] }
  index  "idx_apps_status"    { columns = [column.status] }

  check "chk_status" { expr = "status IN ('active', 'suspended', 'deleted')" }
}

table "outbox" {
  schema = schema.appstore
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
