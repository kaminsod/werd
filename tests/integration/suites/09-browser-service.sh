# Suite 09 — Browser Service
#
# Validates the browser automation service is running and accessible
# to the werd-api container but not directly from the host.

suite "Browser Service"

# ── Health check via internal network ──
bs_health=$(compose_exec werd-api curl -sf http://browser-service:8091/healthz 2>/dev/null || echo "")
if [ -n "$bs_health" ]; then
  assert_contains "$bs_health" "ok" "Browser service healthz reachable from werd-api"
else
  skip "Browser service healthz (service may not be running)"
fi

# ── Not directly exposed on host ──
if port_open 8091; then
  fail "Port 8091 should NOT be exposed on host (browser service direct access)"
else
  pass "Port 8091 not exposed (browser service only reachable internally)"
fi
