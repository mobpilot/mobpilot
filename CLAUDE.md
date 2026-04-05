# Mobpilot — Claude Code Context

## Project overview

Mobpilot is an open-source universal mobile app backend platform. It provides social media features, multi-tenant app management, push notifications, and AI-driven configuration via an MCP server. Users can self-host with Docker Compose or Kubernetes, or use the managed cloud offering.

## Architecture

Hexagonal architecture (Ports & Adapters) throughout. Strict dependency rule:
```
infrastructure → application → domain
```
Nothing in `domain/` or `application/` may import from `infrastructure/`.

## Module layout

```
services/<name>/
  domain/            pure Go types, errors, events, outbound port interfaces
  application/       use-case services, inbound port interfaces, command types
  infrastructure/    adapters: postgres (sqlc), redis, nats, connect-rpc, http (ogen)
  cmd/<name>/        main.go — DI wiring only

internal/platform/   shared infrastructure primitives (pool, otel, middleware)
api/openapi/         OpenAPI 3.1 specs (source of truth for ogen codegen)
api/proto/           Buf-managed .proto files (source of truth for connect-rpc)
tools/atlas/         Atlas HCL schemas (source of truth for migrations)
migrations/          Atlas-managed versioned SQL migrations
deploy/              Docker Compose + Kubernetes + Terraform
admin/               React 19 + TanStack admin frontend
```

## Key technology choices

| Concern | Tool |
|---|---|
| Auth | Ory Hydra (OAuth2) + Kratos (identity) + Keto (RBAC) |
| DB | PostgreSQL 18, queries via sqlc + pgx/v5 |
| Schema migrations | Atlas (ariga.io) HCL |
| HTTP codegen | ogen (from OpenAPI 3.1 specs) |
| Service RPC | Connect-RPC (Buf) from .proto files |
| Job queue | River (Postgres-native, transactional) |
| DI | Wire (compile-time) |
| Messaging | NATS JetStream |
| Logging | log/slog + OTel bridge |
| Tracing | OpenTelemetry → Grafana Tempo |

## Code generation

Always run generators after changing source files:

```bash
make generate-sqlc     # after editing query.sql or schema.sql
make generate-buf      # after editing .proto files
make generate-ogen     # after editing api/openapi/*.yaml
```

## Testing

- Use `testcontainers-go` for real PostgreSQL and Redis in integration tests.
- Tag integration tests with `//go:build integration`.
- Unit tests must not require running infrastructure.

## Error handling conventions

- Domain layer: sentinel errors (`var ErrFoo = errors.New("...")`)
- Application layer: wrap with `fmt.Errorf("Service.Method ctx: %w", err)`
- Infrastructure layer: map pg/redis errors to domain errors at the adapter boundary
- HTTP layer: translate domain errors to RFC 7807 Problem Details

## Multi-tenancy

Every DB table has `app_id UUID NOT NULL`. Every authenticated request carries `mobpilot_app_id` as a JWT custom claim (injected by Hydra claims hook). All repository queries must include `WHERE app_id = $<n>`.

## Commit style

Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`
