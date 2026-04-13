// Atlas declarative schema for the notification service.
// Run: atlas schema apply --env notification
// Diff: atlas schema diff --env notification

schema "notification" {}

table "notifications" {
  schema = schema.notification
  column "id"              { type = uuid  default = sql("gen_random_uuid()") }
  column "user_id"         { type = uuid }
  column "app_id"          { type = uuid }
  column "group_id"        { type = uuid         null = true }
  column "org_id"          { type = uuid         null = true }
  column "title"           { type = text }
  column "body"            { type = text }
  column "data"            { type = jsonb        default = sql("'{}'::jsonb") }
  column "delivery_status" { type = text         default = "pending" }
  column "read_at"         { type = timestamptz  null = true }
  column "created_at"      { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.id] }
  index "idx_notifications_user" {
    columns = [column.user_id, column.app_id, column.created_at]
  }

  check "chk_delivery_status" {
    expr = "delivery_status IN ('pending', 'delivered', 'failed')"
  }
}

table "notification_preferences" {
  schema = schema.notification
  column "user_id"    { type = uuid }
  column "app_id"     { type = uuid }
  column "channel"    { type = text }
  column "event_type" { type = text }
  column "enabled"    { type = boolean      default = true }
  column "updated_at" { type = timestamptz  default = sql("now()") }

  primary_key { columns = [column.user_id, column.app_id, column.channel, column.event_type] }

  check "chk_channel" { expr = "channel IN ('push', 'email', 'in_app')" }
}
