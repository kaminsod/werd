#!/bin/bash
set -euo pipefail

# ============================================================================
# Werd — Phase 1 Integration Test Harness
# ============================================================================
#
# Spins up the full compose stack with a test-specific override, runs all
# assertion suites against it, prints a summary, and tears down.
#
# Usage:
#   ./tests/integration/run.sh          # normal run (teardown after)
#   WERD_TEST_KEEP=1 ./run.sh           # leave stack running for inspection
#
# Requirements:
#   - podman-compose (preferred) or docker compose
#   - curl, openssl
#   - No other compose stack using the werd-net network
#
# Exit code: 0 if all tests pass, 1 if any fail.
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

detect_compose_cmd

echo "============================================"
echo "  Werd Integration Tests — Phase 1"
echo "============================================"
echo "Compose:  $COMPOSE_CMD"
echo "Project:  $PROJECT_NAME"
echo "Date:     $(date -Iseconds)"
echo ""

# ── Teardown any previous run ──

echo "Cleaning up any previous test stack..."
compose_cmd down -v 2>/dev/null || true

# ── Generate test .env ──

echo "Generating test environment..."
cp "$COMPOSE_DIR/.env.example" "$TEST_ENV"

# Override for local/test mode.
sed -i 's/^WERD_DOMAIN=.*/WERD_DOMAIN=localhost/' "$TEST_ENV"
sed -i 's/^WERD_ACCESS_MODE=.*/WERD_ACCESS_MODE=local/' "$TEST_ENV"

# Generate real secrets into the test .env.
"$REPO_ROOT/tools/generate-secrets.sh" "$TEST_ENV"

# Export secrets needed by test assertions.
export REDIS_PASSWORD
REDIS_PASSWORD=$(grep '^REDIS_PASSWORD=' "$TEST_ENV" | cut -d= -f2)

# Absolute path for the Caddyfile bind mount (avoids compose path resolution issues).
export CADDYFILE_PATH="$REPO_ROOT/src/deploy/caddy/Caddyfile.local"

# ── Build ──

echo ""
echo "Building images..."
compose_cmd build 2>&1 | tail -5

# ── Start stack ──

echo ""
echo "Starting test stack..."
compose_cmd up -d

# ── Wait for healthy ──
# Caddy depends on all upstream services via service_healthy conditions, so
# if both Caddy endpoints respond the entire dependency chain is up.

echo "Waiting for stack to become healthy (timeout: 120s)..."

healthy=false
if wait_for_url "$CADDY_API/healthz" 120; then
  if wait_for_url "$CADDY_DASHBOARD" 10; then
    healthy=true
  fi
fi

if ! $healthy; then
  echo ""
  echo "ERROR: Stack did not become healthy within timeout."
  echo ""
  echo "--- compose ps ---"
  compose_cmd ps || true
  echo ""
  echo "--- compose logs (last 80 lines) ---"
  compose_cmd logs --tail=80 || true
  compose_cmd down -v 2>/dev/null || true
  rm -f "$TEST_ENV"
  exit 1
fi

echo "All services healthy."

# ── Run test suites ──

for suite_file in "$SCRIPT_DIR/suites/"*.sh; do
  source "$suite_file"
done

# ── Summary ──

echo ""
echo "============================================"
printf "  Results: \033[32m%d passed\033[0m" "$PASS_COUNT"
[ "$FAIL_COUNT" -gt 0 ] && printf ", \033[31m%d failed\033[0m" "$FAIL_COUNT"
[ "$SKIP_COUNT" -gt 0 ] && printf ", \033[33m%d skipped\033[0m" "$SKIP_COUNT"
echo ""
echo "============================================"

# ── Teardown ──

if [ "${WERD_TEST_KEEP:-0}" != "1" ]; then
  echo ""
  echo "Tearing down test stack..."
  compose_cmd down -v
  rm -f "$TEST_ENV"
else
  echo ""
  echo "WERD_TEST_KEEP=1 — stack left running for inspection."
  echo "  Dashboard: $CADDY_DASHBOARD"
  echo "  API:       $CADDY_API/healthz"
  echo "Teardown manually:  $COMPOSE_CMD -p $PROJECT_NAME down -v"
fi

[ "$FAIL_COUNT" -gt 0 ] && exit 1 || exit 0
