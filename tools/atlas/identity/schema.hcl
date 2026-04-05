// Atlas declarative schema for the identity service.
// Run: atlas schema apply --env identity
// Diff: atlas schema diff --env identity

schema "identity" {}

table "user_profiles" {
  schema = schema.identity
  column "id"           { type = uuid }
  column "app_id"       { type = uuid }
  column "display_name" { type = text    default = "" }
  column "avatar_url"   { type = text    default = "" }
  column "bio"          { type = text    default = "" }
  column "location"     { type = text    default = "" }
  column "website_url"  { type = text    default = "" }
  column "settings"     { type = jsonb   default = "{}" }
  column "created_at"   { type = timestamptz  default = sql("now()") }
  column "updated_at"   { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id, column.app_id] }
  index "idx_user_profiles_app" { columns = [column.app_id] }
}

table "device_tokens" {
  schema = schema.identity
  column "id"         { type = uuid    default = sql("gen_random_uuid()") }
  column "user_id"    { type = uuid }
  column "app_id"     { type = uuid }
  column "platform"   { type = text }
  column "token_type" { type = text }
  column "token"      { type = text }
  column "device_id"  { type = text }
  column "created_at" { type = timestamptz default = sql("now()") }
  column "updated_at" { type = timestamptz default = sql("now()") }

  primary_key { columns = [column.id] }
  unique "uq_device_tokens_app_device" { columns = [column.app_id, column.device_id] }
  index  "idx_device_tokens_user"      { columns = [column.user_id, column.app_id] }

  check "chk_platform"   { expr = "platform IN ('ios', 'android', 'web')" }
  check "chk_token_type" { expr = "token_type IN ('fcm', 'apns')" }
}

table "artifact_storage" {
  schema = schema.identity
  column "id"         { type = uuid    default = sql("gen_random_uuid()") }
  column "user_id"    { type = uuid }
  column "app_id"     { type = uuid }
  column "key"        { type = text }
  column "value"      { type = jsonb   null = true }
  column "created_at" { type = timestamptz default = sql("now()") }
  column "updated_at" { type = timestamptz default = sql("now()") }

  primary_key { columns = [column.id] }
  unique "uq_artifact_user_app_key" { columns = [column.user_id, column.app_id, column.key] }
  index  "idx_artifact_storage_user" { columns = [column.user_id, column.app_id] }
}

table "outbox" {
  schema = schema.identity
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
