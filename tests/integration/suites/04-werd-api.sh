# Suite 04 — Werd API
#
# Validates the /healthz endpoint via Caddy and confirms the API is NOT
# directly reachable from the host (only through the reverse proxy).

suite "Werd API"

# ── /healthz returns 200 via Caddy ──
status=$(curl -sf -o /dev/null -w '%{http_code}' "$CADDY_API/healthz" 2>/dev/null || echo "000")
assert_eq "200" "$status" "/healthz returns 200 via Caddy"

# ── /healthz returns correct JSON body ──
body=$(curl -sf "$CADDY_API/healthz" 2>/dev/null || echo "")
assert_eq '{"status":"ok"}' "$body" '/healthz returns {"status":"ok"}'

# ── API not directly reachable from host ──
# Port 8090 should not be exposed — all access goes through Caddy.
if port_open 8090; then
  fail "Port 8090 should NOT be exposed on host (API direct access)"
else
  pass "Port 8090 not exposed (API only reachable via Caddy)"
fi
