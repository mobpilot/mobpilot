#!/usr/bin/env bash
# bootstrap-dev.sh — One-command local Driftbase dev setup.
# Usage: ./scripts/bootstrap-dev.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_DIR="$ROOT_DIR/deploy"

# ── Colours ────────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[driftbase]${NC} $*"; }
warn()  { echo -e "${YELLOW}[driftbase]${NC} $*"; }
error() { echo -e "${RED}[driftbase]${NC} $*" >&2; }

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
  if docker compose exec -T postgres pg_isready -U driftbase &>/dev/null; then
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
info "Creating MinIO bucket 'driftbase-media' ..."
docker compose exec -T minio \
  mc alias set local http://localhost:9000 driftbase driftbase_dev_secret 2>/dev/null || true
docker compose exec -T minio \
  mc mb --ignore-existing local/driftbase-media 2>/dev/null || true
info "MinIO bucket ready."

# ── Bootstrap Hydra OAuth2 clients ────────────────────────────────────────────
info "Bootstrapping Hydra OAuth2 clients ..."

# Admin/MCP client (machine-to-machine, client credentials)
if ! curl -sf http://localhost:4445/admin/clients/driftbase-admin &>/dev/null; then
  curl -sf -X POST http://localhost:4445/admin/clients \
    -H "Content-Type: application/json" \
    -d '{
      "client_id": "driftbase-admin",
      "client_name": "Driftbase Admin / MCP",
      "client_secret": "driftbase-admin-dev-secret",
      "grant_types": ["client_credentials"],
      "scope": "driftbase:admin driftbase:apps:write driftbase:org:write",
      "token_endpoint_auth_method": "client_secret_post"
    }' | jq .client_id
  info "Created Hydra client: driftbase-admin"
fi

# Dev app client (PKCE flow for mobile)
if ! curl -sf http://localhost:4445/admin/clients/driftbase-dev-app &>/dev/null; then
  curl -sf -X POST http://localhost:4445/admin/clients \
    -H "Content-Type: application/json" \
    -d '{
      "client_id": "driftbase-dev-app",
      "client_name": "Driftbase Dev App",
      "redirect_uris": ["http://localhost:3000/callback", "driftbasedev://callback"],
      "grant_types": ["authorization_code", "refresh_token"],
      "response_types": ["code"],
      "scope": "openid profile offline_access driftbase:social driftbase:media",
      "token_endpoint_auth_method": "none",
      "metadata": {"driftbase_app_id": "00000000-0000-0000-0000-000000000001"}
    }' | jq .client_id
  info "Created Hydra client: driftbase-dev-app"
fi

info "Hydra clients configured."

# ── Run database migrations ────────────────────────────────────────────────────
info "Applying database migrations ..."
cd "$ROOT_DIR"
if command -v atlas &>/dev/null; then
  for svc in identity; do
    if [[ -d "migrations/$svc" ]]; then
      atlas migrate apply \
        --dir "file://migrations/$svc" \
        --url "postgres://driftbase:driftbase_dev@localhost:5432/driftbase?sslmode=disable&search_path=$svc" \
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
info "Driftbase local stack is ready!"
info ""
info "  Hydra (public)     http://localhost:4444"
info "  Hydra (admin)      http://localhost:4445"
info "  Kratos (public)    http://localhost:4433"
info "  Kratos (admin)     http://localhost:4434"
info "  Keto (read)        http://localhost:4466"
info "  MinIO console      http://localhost:9001"
info "  MailHog            http://localhost:8025"
info "  NATS monitor       http://localhost:8222"
info ""
info "  To start application services:"
info "    docker compose --profile services up --build"
info ""
info "  Or start everything:"
info "    docker compose --profile all up --build"
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
