#!/bin/bash
set -euo pipefail

# ============================================================================
# Quick Admin Login Test
# ============================================================================
#
# Tests admin login against a running Werd instance. Can be run standalone
# against either the local dev stack or the integration test stack.
#
# Usage:
#   ./tests/test-admin-login.sh                         # default: http://localhost:3081
#   ./tests/test-admin-login.sh http://kbox:3081        # custom API base
#   API_BASE=http://kbox:3081 ./tests/test-admin-login.sh
#
# Reads credentials from src/deploy/compose/.env by default.
# Override with ADMIN_EMAIL and ADMIN_PASSWORD env vars.
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$REPO_ROOT/src/deploy/compose/.env"

API_BASE="${1:-${API_BASE:-http://localhost:3081}}"
API_URL="$API_BASE/api"

# Read credentials from .env if not overridden.
if [ -z "${ADMIN_EMAIL:-}" ] && [ -f "$ENV_FILE" ]; then
  ADMIN_EMAIL=$(grep '^WERD_ADMIN_EMAIL=' "$ENV_FILE" | cut -d= -f2)
fi
if [ -z "${ADMIN_PASSWORD:-}" ] && [ -f "$ENV_FILE" ]; then
  ADMIN_PASSWORD=$(grep '^WERD_ADMIN_PASSWORD=' "$ENV_FILE" | cut -d= -f2)
fi

ADMIN_EMAIL="${ADMIN_EMAIL:?ADMIN_EMAIL not set and not found in .env}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:?ADMIN_PASSWORD not set and not found in .env}"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); printf "  \033[32mPASS\033[0m %s\n" "$1"; }
fail() { FAIL=$((FAIL + 1)); printf "  \033[31mFAIL\033[0m %s\n" "$1"; }

echo "============================================"
echo "  Admin Login Test"
echo "============================================"
echo "  API:      $API_URL"
echo "  Email:    $ADMIN_EMAIL"
echo "  Password: ${ADMIN_PASSWORD:0:2}****"
echo ""

# ── 1. API healthcheck ──

health_status=$(curl -sf -o /dev/null -w '%{http_code}' "$API_URL/../healthz" 2>/dev/null || echo "000")
if [ "$health_status" = "200" ]; then
  pass "API is reachable (/healthz → 200)"
else
  fail "API is NOT reachable (/healthz → $health_status)"
  echo ""
  echo "  Cannot reach $API_URL/../healthz"
  echo "  Make sure the stack is running: ./tools/runner.sh status"
  exit 1
fi

# ── 2. Login attempt ──

login_resp=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" 2>/dev/null)

login_body=$(echo "$login_resp" | head -1)
login_status=$(echo "$login_resp" | tail -1)

if [ "$login_status" = "200" ]; then
  pass "POST /auth/login → 200"
else
  fail "POST /auth/login → $login_status (expected 200)"
  echo ""
  echo "  Response body: $login_body"
  echo ""
  echo "  Diagnostics: checking database state..."

  # Try to query postgres directly via compose.
  COMPOSE_DIR="$REPO_ROOT/src/deploy/compose"
  if command -v docker >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v podman-compose >/dev/null 2>&1; then
    COMPOSE_CMD="podman-compose"
  else
    echo "  (no compose tool found, cannot query database)"
    echo ""
    echo "  Results: $PASS passed, $FAIL failed"
    exit 1
  fi

  echo ""
  echo "  Users in database:"
  $COMPOSE_CMD -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.local.yml" \
    --env-file "$COMPOSE_DIR/.env" \
    exec -T postgres psql -U werd -d werd -c \
    "SELECT id, email, name, length(password_hash) as hash_len, created_at FROM users;" 2>/dev/null \
    | sed 's/^/    /' || echo "    (could not query database)"

  echo ""
  echo "  Project memberships:"
  $COMPOSE_CMD -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.local.yml" \
    --env-file "$COMPOSE_DIR/.env" \
    exec -T postgres psql -U werd -d werd -c \
    "SELECT pm.project_id, u.email, pm.role FROM project_members pm JOIN users u ON pm.user_id = u.id;" 2>/dev/null \
    | sed 's/^/    /' || echo "    (could not query database)"

  echo ""
  echo "  Orphaned projects (no members):"
  $COMPOSE_CMD -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.local.yml" \
    --env-file "$COMPOSE_DIR/.env" \
    exec -T postgres psql -U werd -d werd -c \
    "SELECT p.id, p.slug FROM projects p WHERE NOT EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id);" 2>/dev/null \
    | sed 's/^/    /' || echo "    (could not query database)"

  echo ""
  echo "  werd-api logs (last 20 lines):"
  $COMPOSE_CMD -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.local.yml" \
    --env-file "$COMPOSE_DIR/.env" \
    logs --tail=20 werd-api 2>/dev/null \
    | sed 's/^/    /' || echo "    (could not read logs)"

  echo ""
  echo "  Results: $PASS passed, $FAIL failed"
  exit 1
fi

# ── 3. Verify token ──

token=$(echo "$login_body" | jq -r '.token // empty')
if [ -n "$token" ]; then
  pass "Response contains JWT token"
else
  fail "Response missing JWT token"
fi

user_email=$(echo "$login_body" | jq -r '.user.email // empty')
if [ "$user_email" = "$ADMIN_EMAIL" ]; then
  pass "Response email matches ($user_email)"
else
  fail "Response email mismatch (expected $ADMIN_EMAIL, got $user_email)"
fi

# ── 4. /auth/me works ──

me_status=$(curl -sf -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $token" "$API_URL/auth/me" 2>/dev/null || echo "000")
if [ "$me_status" = "200" ]; then
  pass "GET /auth/me → 200 (token is valid)"
else
  fail "GET /auth/me → $me_status (expected 200)"
fi

# ── 5. Can list projects ──

projects=$(curl -sf -H "Authorization: Bearer $token" "$API_URL/projects" 2>/dev/null || echo "[]")
project_count=$(echo "$projects" | jq 'if type == "array" then length else 0 end' 2>/dev/null || echo "0")
pass "Can list projects ($project_count found)"

# ── 6. Wrong password returns proper error ──

wrong_resp=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"wrongpassword\"}" 2>/dev/null)

wrong_body=$(echo "$wrong_resp" | head -1)
wrong_status=$(echo "$wrong_resp" | tail -1)

if [ "$wrong_status" = "401" ]; then
  pass "Wrong password → 401"
else
  fail "Wrong password → $wrong_status (expected 401)"
fi

wrong_msg=$(echo "$wrong_body" | jq -r '.message // empty')
if [ "$wrong_msg" = "invalid email or password" ]; then
  pass "401 body has correct error message"
else
  fail "401 body message: '$wrong_msg' (expected 'invalid email or password')"
fi

# ── Summary ──

echo ""
echo "============================================"
printf "  Results: \033[32m%d passed\033[0m" "$PASS"
[ "$FAIL" -gt 0 ] && printf ", \033[31m%d failed\033[0m" "$FAIL"
echo ""
echo "============================================"

[ "$FAIL" -gt 0 ] && exit 1 || exit 0
