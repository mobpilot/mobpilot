#!/usr/bin/env bash
# bootstrap-dev.sh — One-command local Mobpilot dev setup.
# Usage: ./scripts/bootstrap-dev.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_DIR="$ROOT_DIR/deploy"

# ── Colours ────────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[mobpilot]${NC} $*"; }
warn()  { echo -e "${YELLOW}[mobpilot]${NC} $*"; }
error() { echo -e "${RED}[mobpilot]${NC} $*" >&2; }

# ── Prereq checks ──────────────────────────────────────────────────────────────
for cmd in docker curl jq; do
  if ! command -v "$cmd" &>/dev/null; then
    error "Required tool not found: $cmd"
    exit 1
  fi
done

# ── .env setup ─────────────────────────────────────────────────────────────────
if [[ ! -f "$DEPLOY_DIR/.env" ]]; then
  info "Creating deploy/.env from .env.example ..."
  cp "$DEPLOY_DIR/.env.example" "$DEPLOY_DIR/.env"
  warn "Review deploy/.env before deploying to production!"
fi

# ── Start infrastructure ───────────────────────────────────────────────────────
info "Starting infrastructure (profile: infra) ..."
cd "$DEPLOY_DIR"
docker compose --profile infra up -d

# ── Wait for PostgreSQL ────────────────────────────────────────────────────────
info "Waiting for PostgreSQL 18 to be ready ..."
for i in $(seq 1 30); do
  if docker compose exec -T postgres pg_isready -U mobpilot &>/dev/null; then
    break
  fi
  if [[ $i -eq 30 ]]; then
    error "PostgreSQL did not become ready in time."
    exit 1
  fi
  sleep 2
done
info "PostgreSQL ready."

# ── Wait for Hydra ─────────────────────────────────────────────────────────────
info "Waiting for Ory Hydra to be ready ..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:4445/health/ready &>/dev/null; then
    break
  fi
  if [[ $i -eq 30 ]]; then
    error "Hydra did not become ready in time."
    exit 1
  fi
  sleep 3
done
info "Hydra ready."

# ── Wait for Kratos ────────────────────────────────────────────────────────────
info "Waiting for Ory Kratos to be ready ..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:4434/health/ready &>/dev/null; then
    break
  fi
  if [[ $i -eq 30 ]]; then
    error "Kratos did not become ready in time."
    exit 1
  fi
  sleep 3
done
info "Kratos ready."

# ── Create MinIO bucket ────────────────────────────────────────────────────────
info "Creating MinIO bucket 'mobpilot-media' ..."
docker compose exec -T minio \
  mc alias set local http://localhost:9000 mobpilot mobpilot_dev_secret 2>/dev/null || true
docker compose exec -T minio \
  mc mb --ignore-existing local/mobpilot-media 2>/dev/null || true
info "MinIO bucket ready."

# ── Bootstrap Hydra OAuth2 clients ────────────────────────────────────────────
info "Bootstrapping Hydra OAuth2 clients ..."

# Admin/MCP client (machine-to-machine, client credentials)
if ! curl -sf http://localhost:4445/admin/clients/mobpilot-admin &>/dev/null; then
  curl -sf -X POST http://localhost:4445/admin/clients \
    -H "Content-Type: application/json" \
    -d '{
      "client_id": "mobpilot-admin",
      "client_name": "Mobpilot Admin / MCP",
      "client_secret": "mobpilot-admin-dev-secret",
      "grant_types": ["client_credentials"],
      "scope": "mobpilot:admin mobpilot:apps:write mobpilot:org:write",
      "token_endpoint_auth_method": "client_secret_post"
    }' | jq .client_id
  info "Created Hydra client: mobpilot-admin"
fi

# Dev app client (PKCE flow for mobile)
if ! curl -sf http://localhost:4445/admin/clients/mobpilot-dev-app &>/dev/null; then
  curl -sf -X POST http://localhost:4445/admin/clients \
    -H "Content-Type: application/json" \
    -d '{
      "client_id": "mobpilot-dev-app",
      "client_name": "Mobpilot Dev App",
      "redirect_uris": ["http://localhost:3000/callback", "mobpilotdev://callback"],
      "grant_types": ["authorization_code", "refresh_token"],
      "response_types": ["code"],
      "scope": "openid profile offline_access mobpilot:social mobpilot:media",
      "token_endpoint_auth_method": "none",
      "metadata": {"mobpilot_app_id": "00000000-0000-0000-0000-000000000001"}
    }' | jq .client_id
  info "Created Hydra client: mobpilot-dev-app"
fi

info "Hydra clients configured."

# ── Run database migrations ────────────────────────────────────────────────────
info "Applying database migrations ..."
cd "$ROOT_DIR"
if command -v atlas &>/dev/null; then
  for svc in identity org appstore notification; do
    if [[ -d "migrations/$svc" ]]; then
      atlas migrate apply \
        --dir "file://migrations/$svc" \
        --url "postgres://mobpilot:mobpilot_dev@localhost:5432/mobpilot?sslmode=disable&search_path=$svc" \
        2>/dev/null || warn "Migrations for $svc may already be applied."
      info "Migrations applied: $svc"
    fi
  done
else
  warn "Atlas not installed — skipping migrations. Install: https://atlasgo.io/docs/cli/install"
fi

# ── Done ───────────────────────────────────────────────────────────────────────
echo ""
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
info "Mobpilot local stack is ready!"
info ""
info "  Hydra (public)     http://localhost:4444"
info "  Hydra (admin)      http://localhost:4445"
info "  Kratos (public)    http://localhost:4433"
info "  Kratos (admin)     http://localhost:4434"
info "  Keto (read)        http://localhost:4466"
info "  MinIO console      http://localhost:9001"
info "  MailHog            http://localhost:8025"
info "  NATS monitor       http://localhost:8222"
info "  Centrifugo admin   http://localhost:8002"
info "  Traefik dashboard  http://localhost:8080"
info ""
info "  Application services (after infra is up):"
info "    identity service   http://localhost:8090"
info "    org service        http://localhost:8091"
info "    notification svc   http://localhost:8093"
info ""
info "  To start application services:"
info "    docker compose --profile services up --build"
info ""
info "  Or start everything:"
info "    docker compose --profile all up --build"
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
