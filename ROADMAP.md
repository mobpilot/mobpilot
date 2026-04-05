# Mobpilot Roadmap

## Current status: Phase 1 complete

Phase 1 foundation is built and committed. See [ARCHITECTURE.md](ARCHITECTURE.md) for the full tech stack and design decisions.

**What exists today:**
- Go monorepo with module `github.com/mobpilot/mobpilot`
- `internal/platform/` — shared postgres pool, OTel, HTTP middleware
- `services/identity/` — full hexagonal implementation (profiles, device tokens, artifact storage)
- `deploy/docker-compose.yml` — PostgreSQL 18, Redis, NATS, MinIO, Meilisearch, Hydra, Kratos, Keto, Traefik
- `scripts/bootstrap-dev.sh` — one-command local setup
- `tools/atlas/identity/schema.hcl` + `migrations/identity/`
- `api/openapi/identity.yaml`
- `.github/workflows/ci.yml` — lint, buf, sqlc check, test (real PG18), Docker build → GHCR

**To start developing:**
```bash
git clone https://github.com/mobpilot/mobpilot
cd mobpilot
cp deploy/.env.example deploy/.env
./scripts/bootstrap-dev.sh
docker compose -f deploy/docker-compose.yml --profile all up --build
```

---

## Phase 2 — Organizations, RBAC, App Creation (Issues #1–#8)

Goal: multi-tenant model works, apps can be created, Keto enforces permissions, MCP stub is running.

### Issue #1: `services/org` — Organizations service

Implement the `org` service following the same hexagonal pattern as `identity`:

**Domain:**
- `Organization` aggregate (`id`, `name`, `slug`, `owner_user_id`, `plan`, `suspended_at`)
- `OrgMember` value object (`org_id`, `user_id`, `role: owner|admin|member`)
- `Group` aggregate + `GroupMember`
- `Role` aggregate (`id`, `org_id`, `name`, `permissions []string`)
- Domain errors: `ErrOrgNotFound`, `ErrMemberNotFound`, `ErrSlugTaken`
- Domain events: `OrgCreatedEvent`, `MemberInvitedEvent`, `MemberJoinedEvent`

**Application use cases:**
- `CreateOrg(cmd)` → validates slug uniqueness, persists, raises event
- `GetOrg(id, callerID)` → authorization check via Keto
- `InviteOrgMember(cmd)` → creates invitation record, publishes event (notification service will email)
- `AcceptInvitation(token)` → resolves token, adds member
- `RemoveMember(orgID, userID, callerID)`
- `CreateGroup(cmd)`, `AddGroupMember`, `RemoveGroupMember`
- `CreateRole(cmd)`, `DeleteRole`

**Infrastructure:**
- `infrastructure/postgres/sqlc/` — schema.sql + query.sql for all org tables
- `infrastructure/postgres/` — `OrgRepo`, `MemberRepo`, `GroupRepo`, `RoleRepo`
- `infrastructure/nats/` — event publisher
- `infrastructure/http/` — REST handler
- `tools/atlas/org/schema.hcl` + `migrations/org/000001_init.sql`
- `api/openapi/org.yaml`

**Key tables:**
```sql
organizations(id, name, slug UNIQUE, owner_user_id, plan, created_at, suspended_at)
org_members(org_id, user_id, role, invited_by, joined_at)  -- composite PK
org_invitations(id, org_id, email, role, token, expires_at, accepted_at)
groups(id, org_id, name, description, created_at)
group_members(group_id, user_id)  -- composite PK
roles(id, org_id, name, permissions TEXT[], created_at)
```

---

### Issue #2: `services/appstore` — App Registry service

**Domain:**
- `App` aggregate (`id`, `org_id`, `name`, `slug`, `description`, `icon_url`, `enabled_features []string`, `config JSONB`, `push_config JSONB`, `db_schema`, `status`)
- `EnabledFeature` enum: `social`, `media`, `notifications`, `storage`, `search`
- Domain errors: `ErrAppNotFound`, `ErrAppSlugTaken`
- Domain event: `AppCreatedEvent`, `AppUpdatedEvent`

**Application use cases:**
- `CreateApp(cmd)` → provisions DB schema (`CREATE SCHEMA app_<slug>`), registers Hydra OAuth2 client via Hydra admin API, persists app record
- `GetApp(id, callerID)`, `ListAppsByOrg(orgID, callerID)`
- `UpdateApp(cmd)` — update features, config, push_config
- `DeleteApp(id, callerID)` — soft delete
- `ConfigurePush(cmd)` — stores FCM/APNs credential references in `push_config`
- `GenerateAPIKey(appID, callerID)` → calls Hydra to create client credentials for SDK use

**Hydra integration:**
- When `CreateApp` is called, the use case calls `HydraAdminClient` (outbound port) to `POST /admin/clients`
- The new client gets `metadata.mobpilot_app_id` set so the claims hook enriches tokens
- Hydra client ID convention: `app-{slug}`

**Infrastructure:**
- `infrastructure/postgres/`, `infrastructure/http/`, `infrastructure/hydra/` (Hydra admin client adapter)
- `tools/atlas/appstore/schema.hcl` + `migrations/appstore/000001_init.sql`
- `api/openapi/appstore.yaml`

---

### Issue #3: Ory Keto integration — `internal/platform/authz`

Wire Ory Keto for fine-grained RBAC across all services.

**Namespace definitions** (update `deploy/config/keto/keto.yaml`):
```
Organization: owner, admin, member
App: owner, viewer, editor
Post: author, viewer
Group: member, admin
```

**`internal/platform/authz/` package:**
```go
type Checker interface {
    Check(ctx context.Context, subject, relation, object string) (bool, error)
    WriteRelation(ctx context.Context, subject, relation, object string) error
    DeleteRelation(ctx context.Context, subject, relation, object string) error
}
```
- Implementation wraps the Ory Keto gRPC client (`github.com/ory/keto-client-go`)
- Used by HTTP middleware and application services for authorization

**Integration points:**
- `org` service writes tuples on member join/leave
- `appstore` service writes tuples on app creation
- `identity` service HTTP handler checks `mobpilot:read` on app for `GET /users/{id}`

---

### Issue #4: Hydra claims hook — enrich tokens with `mobpilot_app_id`

Hydra supports a token hook that calls an external HTTP endpoint to add custom claims.

**Implementation:**
- Add `services/auth/` with a minimal Go HTTP server (no hexagonal needed — pure infrastructure)
- Implements Hydra's token hook interface: `POST /token-hook`
- Receives `{ client_id, subject, requested_scope }`, looks up the app by `client_id` → `appstore` service, returns `{ extra: { mobpilot_app_id: "..." } }`
- Configure Hydra via env: `OAUTH2_TOKEN_HOOK_URL=http://auth:8090/token-hook`
- Add `auth` service to `deploy/docker-compose.yml`

**Note:** For the dev Hydra client `mobpilot-dev-app`, the `metadata.mobpilot_app_id` is set directly at client creation (see `bootstrap-dev.sh`), so the hook is only strictly needed when apps are dynamically provisioned.

---

### Issue #5: `services/mcp` — MCP server (Phase 2 stub)

Implement the MCP server with Phase 2 tools: `create_app`, `list_apps`, `create_org`, `get_platform_status`.

**Transport:** HTTP + SSE (primary), stdio (secondary)

**Tech:** Use `github.com/mark3labs/mcp-go` or implement the MCP protocol directly (it's a simple JSON-RPC 2.0 over HTTP/SSE).

**Architecture:** The MCP server is a thin infrastructure layer — no domain, no application layer. It authenticates the caller (Hydra Bearer token, scope `mobpilot:admin`), then calls the internal services via Connect-RPC clients.

**Tools for Phase 2:**
```json
create_org        — POST /v1/org/organizations
list_orgs         — GET  /v1/org/organizations
create_app        — POST /v1/apps
list_apps         — GET  /v1/apps?org_id=...
get_platform_status — health checks for all services
```

**Directory:**
```
services/mcp/
├── cmd/mcp/main.go
├── server.go        — MCP server setup, tool registration
├── tools/
│   ├── org.go       — create_org, list_orgs
│   ├── appstore.go  — create_app, list_apps
│   └── health.go    — get_platform_status
└── Dockerfile
```

**Claude Code integration:**
```bash
claude mcp add --transport http http://localhost:9000/mcp
# or for self-hosted cloud:
claude mcp add --transport http https://your-mobpilot.com/mcp
```

---

### Issue #6: `api/proto/` — Define Connect-RPC .proto files

Define `.proto` service definitions for all Phase 2 services so they can call each other internally via Connect-RPC (not HTTP). The `identity` service proto was deferred; do all three now.

**Files to create:**
- `api/proto/identity/v1/identity.proto` — `GetUser`, `GetDeviceTokens`
- `api/proto/org/v1/org.proto` — `GetOrg`, `IsMember`, `GetMemberRole`
- `api/proto/appstore/v1/appstore.proto` — `GetApp`, `GetAppByClientID`

**Run `make generate-buf`** after creating these to generate Connect-RPC Go code in `gen/go/`.

**Wire the Connect handlers** in `infrastructure/connect/` for each service (alongside the existing REST handlers).

---

### Issue #7: Database migrations for org + appstore

- `tools/atlas/org/schema.hcl`
- `migrations/org/000001_init.sql`
- `tools/atlas/appstore/schema.hcl`
- `migrations/appstore/000001_init.sql`
- Update `scripts/bootstrap-dev.sh` to apply these migrations on startup

---

### Issue #8: Integration test — full Phase 2 flow

Write an integration test (tag: `//go:build integration`) that:
1. Starts postgres + nats via testcontainers-go
2. Creates an organization
3. Creates an app (verifies Hydra client created + DB schema provisioned)
4. Calls `GET /v1/identity/me` with an app-scoped token
5. Verifies `mobpilot_app_id` claim is present and scoped correctly

File: `tests/integration/phase2_test.go`

---

## Phase 3 — Social features + Media (Issues #9–#14)

### Issue #9: `services/social` — Core social service

Full hexagonal implementation. See `ARCHITECTURE.md` for detailed domain model.

**Domain aggregates:** `Post`, `Follow`, `Reaction`
**Feed strategy:** fan-out on write → Redis sorted sets for ≤1000 followers; pull-on-read for larger accounts
**Transactional outbox** for `PostCreatedEvent`, `PostLikedEvent`, `FollowedEvent`
**Key API endpoints:**
```
POST   /v1/social/posts
GET    /v1/social/posts/{id}
DELETE /v1/social/posts/{id}
GET    /v1/social/feed           (cursor-based)
POST   /v1/social/posts/{id}/reactions
POST   /v1/social/follows/{userID}
DELETE /v1/social/follows/{userID}
GET    /v1/social/users/{id}/followers
GET    /v1/social/users/{id}/following
```

### Issue #10: Feed fan-out worker

`services/social/infrastructure/redis/feed_cache.go` — Redis sorted set operations for feed fan-out.
When a post is created (NATS event `mobpilot.events.social.post.created`), a River job fans out to all followers' Redis feed keys.

### Issue #11: `services/media` — File upload + transcoding

**Pre-signed upload flow:** client calls `POST /v1/media/upload-url` → gets S3 pre-signed PUT URL → uploads directly to MinIO → calls `POST /v1/media/{id}/confirm` → triggers River transcoding job.

**Transcoding:** River job runs ffmpeg (sidecar container) for video; sharp/libvips via CGO for image thumbnails. Stores variant URLs in `media_objects.variants JSONB`.

### Issue #12: OpenAPI specs for social + media

`api/openapi/social.yaml` + `api/openapi/media.yaml`

### Issue #13: `services/social` Connect-RPC proto

`api/proto/social/v1/social.proto` for cross-service calls (e.g. appstore checking post counts for analytics).

### Issue #14: Integration test — social loop

Test: create post with media → follow user → see post in feed → react → verify feed updated.

---

## Phase 4 — Push Notifications (Issues #15–#18)

### Issue #15: `services/notification` — Unified push worker

**NATS consumer:** subscribes to `mobpilot.events.social.*`, `mobpilot.events.org.*`
**FCM:** `firebase.google.com/go/v4/messaging`
**APNs:** `github.com/sideshow/apns2`
**Per-app credentials:** loaded from Kubernetes Secrets (self-hosted) or AWS Secrets Manager (cloud), referenced by `apps.push_config.fcm.service_account_secret_ref`
**Per-user preferences:** stored in `identity.artifact_storage` under key `notification_prefs`
**Token rotation:** on `messaging.ErrRegistrationTokenNotRegistered` from FCM → delete token from `device_tokens`

### Issue #16: Notification history API

```
GET /v1/notify/me/history          (cursor-based)
PATCH /v1/notify/me/preferences    (opt-in/out per event type)
```

### Issue #17: MCP tools for notifications

Add to `services/mcp/tools/`:
- `send_test_notification(app_id, user_id, title, body)`
- `configure_push_notifications(app_id, fcm_config?, apns_config?)`

### Issue #18: Integration test — push delivery

Mock FCM/APNs in test; verify notification record created and token rotation on expiry.

---

## Phase 5 — BFF + Admin Frontend (Issues #19–#24)

### Issue #19: `services/bff` — Backend for Frontend

Aggregates data from all services for the admin dashboard. See `ARCHITECTURE.md` (BFF section).

**OpenAPI spec:** `api/openapi/bff.yaml` — designed for admin UI needs (denormalized, no N+1).

**Key endpoints:**
```
GET /bff/v1/dashboard             — org stats, app list, recent activity (parallel fan-out)
GET /bff/v1/apps                  — apps with usage metrics
POST /bff/v1/apps                 — delegates to appstore service
GET /bff/v1/apps/{id}             — app detail with push config status
GET /bff/v1/users                 — user list with device count
GET /bff/v1/social/posts          — post moderation view
GET /bff/v1/orgs/{id}/members     — member list with roles
```

### Issue #20: Admin frontend scaffold

```bash
cd admin && npm create vite@latest . -- --template react-ts
npm install @tanstack/react-router @tanstack/react-query @tanstack/react-table @tanstack/react-form
npm install @biomejs/biome -D
npx shadcn@latest init
```

Set up TanStack Router file-based routing, QueryClient, Biome config, Tailwind v4, auth guard using Ory Kratos session.

### Issue #21: Admin dashboard page

`admin/src/routes/index.tsx` — dashboard with stats cards (total users, posts, apps, storage used). Uses TanStack Query to fetch from `GET /bff/v1/dashboard`.

### Issue #22: App management UI

`admin/src/routes/apps/index.tsx` + `admin/src/routes/apps/$appId.tsx`
- TanStack Table for app list
- TanStack Form for "Create new app" (calls `POST /bff/v1/apps`)
- Push notification configuration form (FCM/APNs)

### Issue #23: `openapi-typescript` codegen for admin

`package.json` script: `"gen:api": "openapi-typescript api/openapi/bff.yaml -o src/gen/bff-api.d.ts"`
Wire `openapi-fetch` client in `admin/src/lib/api.ts`.
Add to CI (`admin-check` job in `.github/workflows/ci.yml`).

### Issue #24: Org + user management UI

- `admin/src/routes/orgs/index.tsx` — org list, invite member form
- `admin/src/routes/users/index.tsx` — user table with device count, last seen

---

## Phase 6 — Kubernetes + Self-Hosted (Issues #25–#28)

### Issue #25: Kubernetes Kustomize base manifests

`deploy/k8s/base/` — one `Deployment` + `Service` + `HorizontalPodAutoscaler` per service.
All services use distroless images from GHCR.
`deploy/k8s/overlays/local/` — k3s-compatible, single node, reduced replicas.
`deploy/k8s/overlays/production/` — multi-replica, resource limits, PodDisruptionBudgets.

### Issue #26: KEDA autoscaling for workers

`deploy/k8s/base/notification/keda-scaledobject.yaml` — scale notification worker by NATS consumer pending message count.
`deploy/k8s/base/social/keda-scaledobject.yaml` — scale feed fan-out worker by NATS queue depth.

### Issue #27: Helm chart + Terraform AWS modules

`deploy/helm/` — Helm chart wrapping the Kustomize base for cloud deployment.
`deploy/terraform/modules/rds/` — PostgreSQL 18 on AWS RDS.
`deploy/terraform/modules/elasticache/` — Redis on ElastiCache.
`deploy/terraform/modules/s3/` — S3 bucket with lifecycle policies.
`deploy/terraform/modules/eks/` — EKS cluster with managed node groups.

### Issue #28: Self-hosted documentation

`docs/self-hosted.md` — complete walkthrough from zero to running.
`docs/getting-started.md` — 5-minute quickstart with Docker Compose.
`docs/mcp-guide.md` — how to connect Claude Code / Cursor to the MCP server.

---

## Phase 7 — Open Source Launch (Issues #29–#32)

### Issue #29: Go SDK

`sdk/go/` — typed client for the Mobpilot REST API + Connect-RPC client.
Published as `github.com/mobpilot/mobpilot-go`.

### Issue #30: TypeScript SDK

`sdk/typescript/` — typed fetch client generated from OpenAPI specs.
Covers identity, social, media endpoints. Published to npm as `@mobpilot/sdk`.

### Issue #31: Example apps

`examples/react-native/` — React Native starter app using the TypeScript SDK.
`examples/flutter/` — Flutter starter app (manual REST client, until a Dart SDK exists).

### Issue #32: GitHub community setup

- `CONTRIBUTING.md` (already exists — fill in details)
- `.github/ISSUE_TEMPLATE/bug_report.yml`
- `.github/ISSUE_TEMPLATE/feature_request.yml`
- `.github/pull_request_template.md`
- GitHub Discussions enabled
- GitHub Projects board mirroring this roadmap
- First release: `v0.1.0-alpha` tag after Phase 5 is complete
