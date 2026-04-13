# Mobpilot

Open-source universal mobile app backend. Self-host with Docker Compose or Kubernetes, or use the managed cloud offering.

## What is Mobpilot?

Mobpilot provides the backend infrastructure every mobile app needs out of the box:

- **Identity & Auth** — user registration, login, OAuth2/OIDC via Ory Hydra + Kratos
- **Multi-tenant App Management** — isolate data per app with `app_id` on every resource
- **Groups** — ephemeral groups (game sessions, live events) and permanent groups (teams, communities) with membership management
- **Notifications** — multi-channel delivery: WebSocket, SSE, FCM push (Android), APNs push (iOS), email
- **Real-time** — WebSocket and SSE fan-out via Centrifugo, backed by NATS JetStream
- **Authorization** — fine-grained RBAC via Ory Keto (Zanzibar model)
- **File Storage** — S3-compatible via MinIO
- **Search** — full-text search via Meilisearch
- **Observability** — OpenTelemetry traces + structured logging

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Mobile / Web Clients (iOS · Android · Web)                  │
└────────┬──────────────────────────────────────┬─────────────┘
         │ WebSocket / SSE                       │ REST / gRPC
         ▼                                       ▼
┌─────────────────────┐           ┌──────────────────────────┐
│  Centrifugo         │           │  Traefik API Gateway     │
│  (real-time hub)    │           └──────────┬───────────────┘
│  group: + user:     │                      │
│  channels           │         ┌────────────▼───────────────┐
└─────────┬───────────┘         │  Application Services      │
          │ NATS broker         │  ├── identity (:8080)       │
          │                     │  ├── org      (:8081)       │
          ▼                     │  └── notification (:8082)  │
┌─────────────────────┐         └────────────┬───────────────┘
│  NATS JetStream     │                      │
│  Stream: MOBPILOT_  │◄─────────────────────┘
│  EVENTS             │   publish domain events
│  subjects:          │
│  mobpilot.events.>  │
└─────────────────────┘
          │
          │ consume (notification-processor consumer)
          ▼
┌─────────────────────┐   ┌──────────────────────────────────┐
│  Notification Svc   │──►│  FCM  ·  APNs  ·  SMTP           │
│  fan-out per member │   │  (offline / email delivery)      │
└─────────────────────┘   └──────────────────────────────────┘
```

Hexagonal architecture throughout. Strict dependency rule: `infrastructure → application → domain`. Nothing in `domain/` or `application/` imports from `infrastructure/`.

## Repository layout

```
services/
  identity/      user profiles, device tokens
  org/           organizations, members, groups, roles
  notification/  multi-channel notification delivery

internal/platform/
  authz/         Ory Keto client (ReBAC)
  httpmw/        auth middleware (Hydra token introspection)
  nats/          NATS JetStream stream setup helper
  otel/          OpenTelemetry provider
  postgres/      pgxpool factory

api/
  openapi/       OpenAPI 3.1 specs (source of truth for REST codegen)
  proto/         Buf-managed .proto files (Connect-RPC)

migrations/      Atlas-managed versioned SQL migrations
tools/atlas/     Atlas HCL schema definitions
deploy/          Docker Compose + config
docs/            Feature documentation
```

## Quick start

```bash
# Copy env and start infrastructure
cp deploy/.env.example deploy/.env
cd deploy && docker compose --profile all up -d

# Run DB migrations (once infrastructure is healthy)
go run tools/atlas/org/main.go
go run tools/atlas/notification/main.go
```

Services are available via Traefik at `http://localhost:80`:
| Route prefix        | Service      |
|---------------------|--------------|
| `/v1/identity`      | identity     |
| `/v1/org`           | org          |
| `/v1/notifications` | notification |
| `/connection`       | Centrifugo WebSocket/SSE |

## Technology stack

| Concern              | Tool                       |
|----------------------|----------------------------|
| Auth                 | Ory Hydra (OAuth2) + Kratos (identity) + Keto (RBAC) |
| Database             | PostgreSQL 18, sqlc + pgx/v5 |
| Schema migrations    | Atlas (ariga.io) HCL       |
| Real-time            | Centrifugo v5 (WebSocket · SSE) |
| Messaging backbone   | NATS JetStream             |
| Push notifications   | FCM (Android) · APNs (iOS) |
| File storage         | MinIO (S3-compatible)      |
| Search               | Meilisearch                |
| Observability        | OpenTelemetry → Grafana Tempo |
| HTTP codegen         | ogen (from OpenAPI 3.1)    |
| RPC                  | Connect-RPC (Buf)          |
| Job queue            | River (Postgres-native)    |
| DI                   | Wire (compile-time)        |

## Key features

### Groups

Mobpilot supports two group types:

**Ephemeral groups** — short-lived sessions (card games, live Q&A, collaborative editing). Created with `ephemeral: true` and an `expires_at` TTL. Authorization uses Centrifugo subscription JWTs — no Ory Keto writes, zero auth overhead for high-churn scenarios.

**Permanent groups** — long-lived communities, teams, channels. Authorization via Ory Keto relation tuples (`member`, `admin`). Full RBAC integration.

See [docs/groups-and-notifications.md](docs/groups-and-notifications.md) for the full design.

### Notifications

When a message is sent to a group, every member receives it through the best available channel:

1. **WebSocket / SSE** — delivered instantly via Centrifugo to connected clients
2. **Push notification** — FCM (Android) or APNs (iOS) for offline users
3. **Email** — opt-in, via SMTP

Per-user, per-channel, per-event-type preferences. Notification history stored in PostgreSQL.

See [docs/groups-and-notifications.md](docs/groups-and-notifications.md) for delivery flow details.

## Documentation

- [Groups and Notifications](docs/groups-and-notifications.md)

## Contributing

Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`

Run codegen after changing source files:
```bash
make generate-sqlc   # after editing query.sql or schema.sql
make generate-buf    # after editing .proto files
make generate-ogen   # after editing api/openapi/*.yaml
```
