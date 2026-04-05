#!/usr/bin/env bash
# create-github-issues.sh — Creates all Mobpilot roadmap issues on GitHub.
#
# Usage:
#   ./scripts/create-github-issues.sh owner/repo
#
# Prerequisites:
#   gh auth login   (GitHub CLI, authenticated)
#
# This script is idempotent: it skips issues whose titles already exist.
set -euo pipefail

REPO="${1:-}"
if [[ -z "$REPO" ]]; then
  echo "Usage: $0 <owner/repo>"
  echo "Example: $0 mobpilot/mobpilot"
  exit 1
fi

if ! command -v gh &>/dev/null; then
  echo "GitHub CLI (gh) not found. Install: https://cli.github.com"
  exit 1
fi

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[issues]${NC} $*"; }
skip()  { echo -e "${YELLOW}[skip]${NC}  $*"; }

create_issue() {
  local title="$1"
  local body="$2"
  local labels="$3"
  local milestone="$4"

  # Check if issue already exists
  if gh issue list --repo "$REPO" --search "\"$title\" in:title" --json title --jq '.[].title' 2>/dev/null \
      | grep -qF "$title"; then
    skip "$title"
    return
  fi

  gh issue create \
    --repo "$REPO" \
    --title "$title" \
    --body "$body" \
    --label "$labels" \
    --milestone "$milestone" \
    2>/dev/null && info "Created: $title" || info "Created (no milestone/label): $title"
}

# ── Create labels ──────────────────────────────────────────────────────────────
info "Creating labels ..."
for label_def in \
  "phase-1:Foundation:0075ca" \
  "phase-2:Orgs + RBAC + App Creation:e4e669" \
  "phase-3:Social + Media:d93f0b" \
  "phase-4:Push Notifications:fbca04" \
  "phase-5:BFF + Admin Frontend:0e8a16" \
  "phase-6:Kubernetes + Self-Hosted:5319e7" \
  "phase-7:Open Source Launch:b60205" \
  "backend:Go backend service:1d76db" \
  "frontend:Admin frontend:f9d0c4" \
  "infrastructure:Deploy + infra:c5def5" \
  "good first issue:Good for newcomers:7057ff"; do
  IFS=: read -r name desc color <<< "$label_def"
  gh label create "$name" --repo "$REPO" --description "$desc" --color "$color" 2>/dev/null || true
done

# ── Create milestones ──────────────────────────────────────────────────────────
info "Creating milestones ..."
for milestone in \
  "Phase 2: Orgs + RBAC + App Creation" \
  "Phase 3: Social + Media" \
  "Phase 4: Push Notifications" \
  "Phase 5: BFF + Admin Frontend" \
  "Phase 6: Kubernetes + Self-Hosted" \
  "Phase 7: Open Source Launch"; do
  gh api repos/"$REPO"/milestones \
    -f title="$milestone" \
    -f state=open \
    2>/dev/null | true
done

# ── Phase 2 issues ────────────────────────────────────────────────────────────
info "Creating Phase 2 issues ..."

create_issue \
  "feat(org): Implement organizations service (hexagonal)" \
  "$(cat <<'EOF'
Implement \`services/org/\` following the same hexagonal architecture as \`services/identity/\`.

## Domain
- \`Organization\` aggregate (id, name, slug, owner_user_id, plan, suspended_at)
- \`OrgMember\` (org_id, user_id, role: owner|admin|member)
- \`Group\` aggregate + \`GroupMember\`
- \`Role\` aggregate (id, org_id, name, permissions []string)
- Errors: \`ErrOrgNotFound\`, \`ErrMemberNotFound\`, \`ErrSlugTaken\`
- Events: \`OrgCreatedEvent\`, \`MemberInvitedEvent\`, \`MemberJoinedEvent\`

## Application use cases
- \`CreateOrg\`, \`GetOrg\`, \`InviteOrgMember\`, \`AcceptInvitation\`, \`RemoveMember\`
- \`CreateGroup\`, \`AddGroupMember\`, \`RemoveGroupMember\`
- \`CreateRole\`, \`DeleteRole\`

## Infrastructure
- \`infrastructure/postgres/sqlc/\` — schema.sql + query.sql
- Postgres repositories for all aggregates
- NATS event publisher
- REST HTTP handler
- \`tools/atlas/org/schema.hcl\` + \`migrations/org/000001_init.sql\`
- \`api/openapi/org.yaml\`

## Key tables
\`\`\`sql
organizations(id, name, slug UNIQUE, owner_user_id, plan, created_at, suspended_at)
org_members(org_id, user_id, role, invited_by, joined_at)
org_invitations(id, org_id, email, role, token, expires_at, accepted_at)
groups(id, org_id, name, description, created_at)
group_members(group_id, user_id)
roles(id, org_id, name, permissions TEXT[], created_at)
\`\`\`

See [ROADMAP.md](../ROADMAP.md) Issue #1 for full details.
EOF
)" \
  "phase-2,backend" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "feat(appstore): Implement app registry service" \
  "$(cat <<'EOF'
Implement \`services/appstore/\` — app registry with tenant provisioning.

## Domain
- \`App\` aggregate: id, org_id, name, slug, enabled_features, config JSONB, push_config JSONB, db_schema, status
- \`EnabledFeature\` enum: social, media, notifications, storage, search
- Events: \`AppCreatedEvent\`, \`AppUpdatedEvent\`

## Key use cases
- \`CreateApp(cmd)\` → provisions DB schema (\`CREATE SCHEMA app_<slug>\`), registers Hydra OAuth2 client, persists record
- \`UpdateApp\`, \`DeleteApp\`, \`ConfigurePush\`, \`GenerateAPIKey\`

## Hydra integration
When \`CreateApp\` is called:
1. Calls Hydra admin API (\`POST /admin/clients\`) to register new OAuth2 client
2. Sets \`metadata.mobpilot_app_id\` on the client so claims hook enriches tokens
3. Client ID convention: \`app-{slug}\`

See [ROADMAP.md](../ROADMAP.md) Issue #2 for full details.
EOF
)" \
  "phase-2,backend" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "feat(authz): Wire Ory Keto RBAC — internal/platform/authz" \
  "$(cat <<'EOF'
Implement \`internal/platform/authz/\` — Ory Keto client wrapper for fine-grained RBAC.

## Interface
\`\`\`go
type Checker interface {
    Check(ctx context.Context, subject, relation, object string) (bool, error)
    WriteRelation(ctx context.Context, subject, relation, object string) error
    DeleteRelation(ctx context.Context, subject, relation, object string) error
}
\`\`\`

## Namespaces (update deploy/config/keto/keto.yaml)
- Organization: owner, admin, member
- App: owner, viewer, editor
- Post: author, viewer
- Group: member, admin

## Integration points
- \`org\` service writes tuples on member join/leave
- \`appstore\` service writes tuples on app creation
- HTTP middleware in all services uses Keto for authorization

See [ROADMAP.md](../ROADMAP.md) Issue #3 for full details.
EOF
)" \
  "phase-2,backend,infrastructure" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "feat(auth): Hydra token hook — inject mobpilot_app_id claim" \
  "$(cat <<'EOF'
Add \`services/auth/\` — a minimal HTTP server implementing Hydra's token hook.

**Endpoint:** \`POST /token-hook\`
- Receives \`{ client_id, subject, requested_scope }\`
- Looks up app by \`client_id\` via appstore service
- Returns \`{ extra: { mobpilot_app_id: "..." } }\`

**Config:** Set \`OAUTH2_TOKEN_HOOK_URL=http://auth:8090/token-hook\` in Hydra env.

Add \`auth\` service to \`deploy/docker-compose.yml\`.

See [ROADMAP.md](../ROADMAP.md) Issue #4 for full details.
EOF
)" \
  "phase-2,backend,infrastructure" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "feat(mcp): MCP server Phase 2 stub — create_app, create_org, get_platform_status" \
  "$(cat <<'EOF'
Implement \`services/mcp/\` — the Model Context Protocol server.

## Transport
- HTTP + SSE (primary)
- stdio (secondary — for \`claude mcp add\`)

## Tools for Phase 2
| Tool | Delegates to |
|---|---|
| \`create_org\` | \`POST /v1/org/organizations\` |
| \`list_orgs\` | \`GET /v1/org/organizations\` |
| \`create_app\` | \`POST /v1/apps\` |
| \`list_apps\` | \`GET /v1/apps?org_id=...\` |
| \`get_platform_status\` | health checks on all services |

## Architecture
Thin infrastructure-only layer — no domain, no application. Authenticates via Hydra (scope: \`mobpilot:admin\`), then calls services via Connect-RPC clients.

## Claude Code usage
\`\`\`bash
claude mcp add --transport http http://localhost:9000/mcp
\`\`\`

See [ROADMAP.md](../ROADMAP.md) Issue #5 for full details.
EOF
)" \
  "phase-2,backend" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "feat(proto): Define Connect-RPC .proto files for identity, org, appstore" \
  "$(cat <<'EOF'
Define Buf-managed .proto service definitions for internal service-to-service communication.

## Files
- \`api/proto/identity/v1/identity.proto\` — GetUser, GetDeviceTokens
- \`api/proto/org/v1/org.proto\` — GetOrg, IsMember, GetMemberRole
- \`api/proto/appstore/v1/appstore.proto\` — GetApp, GetAppByClientID

Run \`make generate-buf\` to generate Connect-RPC Go code in \`gen/go/\`.

Add Connect-RPC handlers in \`infrastructure/connect/\` for each service.

See [ROADMAP.md](../ROADMAP.md) Issue #6.
EOF
)" \
  "phase-2,backend,infrastructure" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "chore(migrations): Atlas schemas + migrations for org and appstore" \
  "$(cat <<'EOF'
- \`tools/atlas/org/schema.hcl\`
- \`migrations/org/000001_init.sql\`
- \`tools/atlas/appstore/schema.hcl\`
- \`migrations/appstore/000001_init.sql\`
- Update \`scripts/bootstrap-dev.sh\` to apply these on startup

See [ROADMAP.md](../ROADMAP.md) Issue #7.
EOF
)" \
  "phase-2,infrastructure" \
  "Phase 2: Orgs + RBAC + App Creation"

create_issue \
  "test(integration): Phase 2 full flow test" \
  "$(cat <<'EOF'
Integration test (tag: \`//go:build integration\`) covering the full Phase 2 flow:

1. Start postgres + nats via testcontainers-go
2. Create organization
3. Create app (verify Hydra client created + DB schema provisioned)
4. Get app-scoped Hydra token
5. Call \`GET /v1/identity/me\` — verify \`mobpilot_app_id\` claim present

File: \`tests/integration/phase2_test.go\`

See [ROADMAP.md](../ROADMAP.md) Issue #8.
EOF
)" \
  "phase-2,backend" \
  "Phase 2: Orgs + RBAC + App Creation"

# ── Phase 3 issues ────────────────────────────────────────────────────────────
info "Creating Phase 3 issues ..."

create_issue \
  "feat(social): Implement social service — posts, feeds, follows, reactions" \
  "Full hexagonal implementation of \`services/social/\`. See ROADMAP.md Issue #9." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

create_issue \
  "feat(social): Feed fan-out worker — Redis sorted sets + River jobs" \
  "Fan-out on write to Redis sorted sets; River job triggered by NATS PostCreatedEvent. See ROADMAP.md Issue #10." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

create_issue \
  "feat(media): Implement media service — pre-signed upload + transcoding" \
  "Pre-signed S3 upload URL, River transcoding job, MinIO integration. See ROADMAP.md Issue #11." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

create_issue \
  "chore(openapi): OpenAPI specs for social + media services" \
  "\`api/openapi/social.yaml\` + \`api/openapi/media.yaml\`. See ROADMAP.md Issue #12." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

create_issue \
  "feat(proto): Connect-RPC proto for social service" \
  "\`api/proto/social/v1/social.proto\`. See ROADMAP.md Issue #13." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

create_issue \
  "test(integration): Social loop test — post, follow, feed, react" \
  "Integration test: create post with media → follow → see in feed → react. See ROADMAP.md Issue #14." \
  "phase-3,backend" \
  "Phase 3: Social + Media"

# ── Phase 4 issues ────────────────────────────────────────────────────────────
info "Creating Phase 4 issues ..."

create_issue \
  "feat(notification): Unified push notification worker — FCM + APNs" \
  "NATS consumer, FCM via firebase-admin-go, APNs via sideshow/apns2, token rotation. See ROADMAP.md Issue #15." \
  "phase-4,backend" \
  "Phase 4: Push Notifications"

create_issue \
  "feat(notification): Notification history API + user preferences" \
  "GET /v1/notify/me/history, PATCH /v1/notify/me/preferences. See ROADMAP.md Issue #16." \
  "phase-4,backend" \
  "Phase 4: Push Notifications"

create_issue \
  "feat(mcp): MCP tools for push notifications" \
  "send_test_notification, configure_push_notifications tools in services/mcp. See ROADMAP.md Issue #17." \
  "phase-4,backend" \
  "Phase 4: Push Notifications"

create_issue \
  "test(integration): Push notification delivery test" \
  "Mock FCM/APNs, verify notification record created and token rotation. See ROADMAP.md Issue #18." \
  "phase-4,backend" \
  "Phase 4: Push Notifications"

# ── Phase 5 issues ────────────────────────────────────────────────────────────
info "Creating Phase 5 issues ..."

create_issue \
  "feat(bff): Backend for Frontend service — admin dashboard API" \
  "Aggregates data from all services. See ROADMAP.md Issue #19 for API endpoints." \
  "phase-5,backend" \
  "Phase 5: BFF + Admin Frontend"

create_issue \
  "feat(admin): Admin frontend scaffold — Vite + TanStack + Shadcn + Tailwind v4 + Biome" \
  "See ROADMAP.md Issue #20 for setup commands and directory structure." \
  "phase-5,frontend" \
  "Phase 5: BFF + Admin Frontend"

create_issue \
  "feat(admin): Dashboard page — stats cards + app overview" \
  "TanStack Query + Recharts. GET /bff/v1/dashboard. See ROADMAP.md Issue #21." \
  "phase-5,frontend" \
  "Phase 5: BFF + Admin Frontend"

create_issue \
  "feat(admin): App management UI — create app, push config form" \
  "TanStack Table + TanStack Form + Zod. See ROADMAP.md Issue #22." \
  "phase-5,frontend" \
  "Phase 5: BFF + Admin Frontend"

create_issue \
  "chore(admin): openapi-typescript codegen from bff.yaml" \
  "Generate typed client from api/openapi/bff.yaml, wire openapi-fetch. Add to CI. See ROADMAP.md Issue #23." \
  "phase-5,frontend,infrastructure" \
  "Phase 5: BFF + Admin Frontend"

create_issue \
  "feat(admin): Org + user management UI" \
  "Org list, invite member form, user table with device count. See ROADMAP.md Issue #24." \
  "phase-5,frontend" \
  "Phase 5: BFF + Admin Frontend"

# ── Phase 6 issues ────────────────────────────────────────────────────────────
info "Creating Phase 6 issues ..."

create_issue \
  "chore(k8s): Kubernetes Kustomize base manifests for all services" \
  "deploy/k8s/base/ + overlays/local + overlays/production. Distroless images from GHCR. See ROADMAP.md Issue #25." \
  "phase-6,infrastructure" \
  "Phase 6: Kubernetes + Self-Hosted"

create_issue \
  "chore(k8s): KEDA autoscaling for notification + social workers" \
  "Scale by NATS consumer pending count. See ROADMAP.md Issue #26." \
  "phase-6,infrastructure" \
  "Phase 6: Kubernetes + Self-Hosted"

create_issue \
  "chore(infra): Helm chart + Terraform AWS modules (RDS, ElastiCache, S3, EKS)" \
  "See ROADMAP.md Issue #27." \
  "phase-6,infrastructure" \
  "Phase 6: Kubernetes + Self-Hosted"

create_issue \
  "docs: Self-hosted + getting-started documentation" \
  "docs/self-hosted.md, docs/getting-started.md, docs/mcp-guide.md. See ROADMAP.md Issue #28." \
  "phase-6,infrastructure" \
  "Phase 6: Kubernetes + Self-Hosted"

# ── Phase 7 issues ────────────────────────────────────────────────────────────
info "Creating Phase 7 issues ..."

create_issue \
  "feat(sdk): Go SDK — typed client for Mobpilot API" \
  "sdk/go/ — published as github.com/mobpilot/mobpilot-go. See ROADMAP.md Issue #29." \
  "phase-7,backend" \
  "Phase 7: Open Source Launch"

create_issue \
  "feat(sdk): TypeScript SDK — generated from OpenAPI specs" \
  "sdk/typescript/ — published to npm as @mobpilot/sdk. See ROADMAP.md Issue #30." \
  "phase-7,frontend" \
  "Phase 7: Open Source Launch"

create_issue \
  "feat(examples): React Native + Flutter starter apps" \
  "examples/react-native/ + examples/flutter/. See ROADMAP.md Issue #31." \
  "phase-7,frontend" \
  "Phase 7: Open Source Launch"

create_issue \
  "chore(community): GitHub community setup — issue templates, PR template, Discussions, Projects" \
  "See ROADMAP.md Issue #32." \
  "phase-7,infrastructure" \
  "Phase 7: Open Source Launch"

echo ""
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
info "All issues created on $REPO"
info "View: https://github.com/$REPO/issues"
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
