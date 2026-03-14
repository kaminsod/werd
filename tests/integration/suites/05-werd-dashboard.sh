# Suite 05 — Werd Dashboard
#
# Validates that the dashboard serves its React SPA through Caddy, that
# SPA client-side routing works (arbitrary paths return index.html, not 404),
# and that the dashboard is not directly accessible from the host.

suite "Werd Dashboard"

# ── Dashboard serves HTML via Caddy ──
response=$(curl -sf "$CADDY_DASHBOARD" 2>/dev/null || echo "")
if echo "$response" | grep -qi '<!doctype html\|<html'; then
  pass "Dashboard serves HTML via Caddy"
else
  fail "Dashboard did not return HTML (got: $(echo "$response" | head -c 80))"
fi

# ── SPA routing: arbitrary path returns 200 (not 404) ──
status=$(curl -sf -o /dev/null -w '%{http_code}' "$CADDY_DASHBOARD/projects/123/settings" 2>/dev/null || echo "000")
assert_eq "200" "$status" "SPA routing: /projects/123/settings returns 200"

# ── SPA routing: arbitrary path returns index.html content ──
response=$(curl -sf "$CADDY_DASHBOARD/some/deep/route" 2>/dev/null || echo "")
if echo "$response" | grep -qi '<!doctype html\|<html'; then
  pass "SPA routing: arbitrary path returns index.html content"
else
  fail "SPA routing: arbitrary path did not return HTML"
fi

# ── Dashboard not directly reachable from host ──
if port_open 3000; then
  fail "Port 3000 should NOT be exposed on host (dashboard direct access)"
else
  pass "Port 3000 not exposed (dashboard only reachable via Caddy)"
fi
