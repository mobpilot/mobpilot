#!/usr/bin/env bash
# dev-setup.sh — Set up a development environment for testing the Streaks app
# against a running Mobpilot instance.
#
# Prerequisites:
#   - Mobpilot infra + identity service running (see deploy/docker-compose.yml)
#   - curl and jq installed
#
# Usage: ./scripts/dev-setup.sh
set -euo pipefail

HYDRA_ADMIN=${HYDRA_ADMIN_URL:-http://localhost:4445}
HYDRA_PUBLIC=${HYDRA_PUBLIC_URL:-http://localhost:4444}
KRATOS_PUBLIC=${KRATOS_PUBLIC_URL:-http://localhost:4433}
IDENTITY_URL=${IDENTITY_URL:-http://localhost:8090}

GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
info() { echo -e "${GREEN}[streaks]${NC} $*"; }
error() { echo -e "${RED}[streaks]${NC} $*" >&2; }

# Step 1: Get a client credentials token
info "Requesting OAuth2 token from Hydra..."
TOKEN_RESP=$(curl -sf -X POST "$HYDRA_PUBLIC/oauth2/token" \
  -d "grant_type=client_credentials" \
  -d "client_id=mobpilot-admin" \
  -d "client_secret=mobpilot-admin-dev-secret" \
  -d "scope=mobpilot:admin" 2>&1)

TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty')

if [[ -z "$TOKEN" ]]; then
  error "Failed to get token. Is Hydra running?"
  echo "$TOKEN_RESP"
  exit 1
fi

info "Got dev token: ${TOKEN:0:20}..."

# Step 2: Test the Identity API
info "Testing Identity API at $IDENTITY_URL..."
RESP=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$IDENTITY_URL/v1/identity/me" 2>&1)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')

if [[ "$HTTP_CODE" == "200" ]]; then
  info "Identity API is working!"
  echo "$BODY" | jq .
elif [[ "$HTTP_CODE" == "500" ]]; then
  error "Identity API returned 500."
  error "This is likely because the client_credentials token doesn't"
  error "carry the mobpilot_app_id claim that the identity service needs."
  error ""
  error "KNOWN ISSUE: The dev setup needs a Hydra token hook to inject"
  error "the mobpilot_app_id claim into access tokens."
  error "See: https://github.com/mobpilot/mobpilot/issues"
  error ""
  error "WORKAROUND: The Flutter app can run in test mode with the login"
  error "screen. Register via Kratos and the app will handle auth."
else
  error "Identity API returned HTTP $HTTP_CODE"
  echo "$BODY"
fi

# Print Flutter launch command
echo ""
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
info "To run the Streaks app with this token:"
info ""
info "  flutter run -d linux -- \\"
info "    --base-url $IDENTITY_URL/v1/identity \\"
info "    --dev-token $TOKEN \\"
info "    --test-mode"
info ""
info "Or run without a token (uses login screen):"
info ""
info "  flutter run -d linux -- \\"
info "    --base-url $IDENTITY_URL/v1/identity \\"
info "    --kratos-url $KRATOS_PUBLIC \\"
info "    --hydra-url $HYDRA_PUBLIC"
info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
