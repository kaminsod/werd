#!/bin/bash
# Shared utilities for integration tests.
# Sourced by run.sh and all suite files — not executed directly.

# ── Paths ──

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_DIR="$REPO_ROOT/src/deploy/compose"
TEST_DIR="$REPO_ROOT/tests/integration"
TEST_ENV="$TEST_DIR/.env.test"
PROJECT_NAME="werd-test"

# Test endpoints (Caddy via Caddyfile.local, remapped to high ports).
CADDY_DASHBOARD="http://localhost:13080"
CADDY_API="http://localhost:13081"

# ── Compose runtime detection ──

detect_compose_cmd() {
  if command -v podman-compose >/dev/null 2>&1; then
    COMPOSE_CMD="podman-compose"
  elif docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
  else
    echo "Error: No compose tool found (install podman-compose or docker compose)"
    exit 1
  fi
}

# ── Compose wrappers ──

# Run a compose command against the test stack.
compose_cmd() {
  $COMPOSE_CMD \
    -f "$COMPOSE_DIR/docker-compose.yml" \
    -f "$TEST_DIR/docker-compose.test.yml" \
    --env-file "$TEST_ENV" \
    -p "$PROJECT_NAME" \
    "$@"
}

# Exec a command inside a running service container (-T = no TTY, for CI).
compose_exec() {
  compose_cmd exec -T "$@"
}

# ── Output formatting ──

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf "  \033[32mPASS\033[0m %s\n" "$1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf "  \033[31mFAIL\033[0m %s\n" "$1"
}

skip() {
  SKIP_COUNT=$((SKIP_COUNT + 1))
  printf "  \033[33mSKIP\033[0m %s\n" "$1"
}

suite() {
  printf "\n\033[1m--- %s ---\033[0m\n" "$1"
}

# ── Assertion helpers ──

# assert_eq "expected" "actual" "description"
assert_eq() {
  local expected="$1" actual="$2" desc="$3"
  if [ "$expected" = "$actual" ]; then
    pass "$desc"
  else
    fail "$desc (expected '$expected', got '$actual')"
  fi
}

# assert_contains "haystack" "needle" "description"
assert_contains() {
  local haystack="$1" needle="$2" desc="$3"
  if echo "$haystack" | grep -qi "$needle"; then
    pass "$desc"
  else
    fail "$desc (expected to contain '$needle')"
  fi
}

# assert_not_contains "haystack" "needle" "description"
assert_not_contains() {
  local haystack="$1" needle="$2" desc="$3"
  if echo "$haystack" | grep -qi "$needle"; then
    fail "$desc (should not contain '$needle')"
  else
    pass "$desc"
  fi
}

# assert_status CODE URL DESCRIPTION — check HTTP response code.
assert_status() {
  local expected="$1" url="$2" desc="$3"
  local actual
  actual=$(curl -sf -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || echo "000")
  assert_eq "$expected" "$actual" "$desc"
}

# ── Network helpers ──

# port_open PORT — returns 0 if a TCP port is listening on localhost.
port_open() {
  timeout 2 bash -c "echo >/dev/tcp/localhost/$1" 2>/dev/null
}

# wait_for_url URL TIMEOUT_SECS — poll until URL returns HTTP 200.
wait_for_url() {
  local url="$1" timeout_secs="${2:-120}" elapsed=0
  while [ $elapsed -lt "$timeout_secs" ]; do
    if curl -sf -o /dev/null "$url" 2>/dev/null; then
      return 0
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  return 1
}

# ── API helpers ──

# login_and_get_token EMAIL PASSWORD — login and echo the JWT token.
login_and_get_token() {
  local email="$1" password="$2"
  local resp
  resp=$(curl -sf -X POST "$CADDY_API/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$password\"}" 2>/dev/null)
  echo "$resp" | jq -r '.token // empty'
}

# api_get TOKEN PATH — GET with JWT auth, echo response body.
api_get() {
  local token="$1" path="$2"
  curl -sf -H "Authorization: Bearer $token" "$CADDY_API/api$path" 2>/dev/null
}

# api_post TOKEN PATH BODY — POST with JWT auth, echo response body.
api_post() {
  local token="$1" path="$2" body="$3"
  curl -sf -X POST -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$body" "$CADDY_API/api$path" 2>/dev/null
}

# api_post_status TOKEN PATH BODY — POST and echo HTTP status code.
api_post_status() {
  local token="$1" path="$2" body="$3"
  curl -s -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$body" "$CADDY_API/api$path" 2>/dev/null
}

# api_put TOKEN PATH BODY — PUT with JWT auth, echo response body.
api_put() {
  local token="$1" path="$2" body="$3"
  curl -sf -X PUT -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$body" "$CADDY_API/api$path" 2>/dev/null
}

# api_put_status TOKEN PATH BODY — PUT and echo HTTP status code.
api_put_status() {
  local token="$1" path="$2" body="$3"
  curl -s -o /dev/null -w '%{http_code}' -X PUT -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$body" "$CADDY_API/api$path" 2>/dev/null
}

# api_delete TOKEN PATH — DELETE with JWT auth, echo response body.
api_delete() {
  local token="$1" path="$2"
  curl -sf -X DELETE -H "Authorization: Bearer $token" "$CADDY_API/api$path" 2>/dev/null
}

# api_delete_status TOKEN PATH — DELETE and echo HTTP status code.
api_delete_status() {
  local token="$1" path="$2"
  curl -s -o /dev/null -w '%{http_code}' -X DELETE \
    -H "Authorization: Bearer $token" "$CADDY_API/api$path" 2>/dev/null
}

# assert_json_field RESPONSE FIELD EXPECTED DESCRIPTION — check a JSON field value.
assert_json_field() {
  local response="$1" field="$2" expected="$3" desc="$4"
  local actual
  actual=$(echo "$response" | jq -r "$field" 2>/dev/null)
  assert_eq "$expected" "$actual" "$desc"
}

# generate_test_id — echo a unique identifier based on timestamp + random suffix.
generate_test_id() {
  echo "werdtest_$(date +%s)_$(head -c4 /dev/urandom | xxd -p)"
}

# Shared test state (set by login_setup, used by test suites).
TEST_TOKEN=""
TEST_PROJECT_ID=""
TEST_ADMIN_EMAIL=""
TEST_ADMIN_PASSWORD=""

# login_setup — create/login admin user and project for test suites.
# Call this once; subsequent suites reuse the exported vars.
login_setup() {
  if [ -n "$TEST_TOKEN" ]; then
    return 0  # already set up
  fi

  # Read admin creds from .env.test
  TEST_ADMIN_EMAIL=$(grep '^WERD_ADMIN_EMAIL=' "$TEST_ENV" | cut -d= -f2)
  TEST_ADMIN_PASSWORD=$(grep '^WERD_ADMIN_PASSWORD=' "$TEST_ENV" | cut -d= -f2)

  if [ -z "$TEST_ADMIN_EMAIL" ] || [ -z "$TEST_ADMIN_PASSWORD" ]; then
    echo "ERROR: WERD_ADMIN_EMAIL / WERD_ADMIN_PASSWORD not set in $TEST_ENV"
    return 1
  fi

  TEST_TOKEN=$(login_and_get_token "$TEST_ADMIN_EMAIL" "$TEST_ADMIN_PASSWORD")
  if [ -z "$TEST_TOKEN" ]; then
    echo "ERROR: Failed to login as admin"
    return 1
  fi

  # Create a test project.
  local resp
  resp=$(api_post "$TEST_TOKEN" "/projects" '{"name":"Integration Test Project","description":"Automated tests"}')
  TEST_PROJECT_ID=$(echo "$resp" | jq -r '.id // empty')
  if [ -z "$TEST_PROJECT_ID" ]; then
    echo "ERROR: Failed to create test project"
    return 1
  fi
}
