# Suite 06 — Caddy Reverse Proxy
#
# Validates security headers, CORS configuration, Server header removal,
# and request body size limit enforcement.

suite "Caddy Reverse Proxy"

# ── Security headers on dashboard ──
# Caddyfile.local applies the security_headers_local snippet to all sites.
headers=$(curl -sI "$CADDY_DASHBOARD" 2>/dev/null)

check_header() {
  local name="$1" expected="$2" desc="$3"
  local value
  value=$(echo "$headers" | grep -i "^${name}:" | head -1 | sed "s/^[^:]*: *//" | tr -d '\r\n')
  if [ -n "$expected" ]; then
    assert_eq "$expected" "$value" "$desc"
  elif [ -n "$value" ]; then
    pass "$desc (value: $value)"
  else
    fail "$desc — header not present"
  fi
}

check_header "X-Content-Type-Options" "nosniff" \
  "Security header: X-Content-Type-Options = nosniff"

check_header "X-Frame-Options" "DENY" \
  "Security header: X-Frame-Options = DENY"

check_header "Referrer-Policy" "strict-origin-when-cross-origin" \
  "Security header: Referrer-Policy"

check_header "Permissions-Policy" "" \
  "Security header: Permissions-Policy present"

# ── Server header removed ──
server_hdr=$(echo "$headers" | grep -i "^Server:" | head -1 | tr -d '\r\n')
if [ -z "$server_hdr" ]; then
  pass "Server header removed"
else
  fail "Server header should be removed (got: $server_hdr)"
fi

# ── CORS headers on API ──
# Caddyfile.local sets Access-Control-Allow-Origin "*" on the API site.
api_headers=$(curl -sI "$CADDY_API/healthz" 2>/dev/null)

cors_origin=$(echo "$api_headers" | grep -i "^Access-Control-Allow-Origin:" | head -1 | sed 's/^[^:]*: *//' | tr -d '\r\n')
assert_eq "*" "$cors_origin" "CORS: Access-Control-Allow-Origin = * (local mode)"

cors_methods=$(echo "$api_headers" | grep -i "^Access-Control-Allow-Methods:" | head -1 | sed 's/^[^:]*: *//' | tr -d '\r\n')
assert_contains "$cors_methods" "POST" "CORS: Allow-Methods includes POST"
assert_contains "$cors_methods" "DELETE" "CORS: Allow-Methods includes DELETE"

cors_headers_val=$(echo "$api_headers" | grep -i "^Access-Control-Allow-Headers:" | head -1 | sed 's/^[^:]*: *//' | tr -d '\r\n')
assert_contains "$cors_headers_val" "Authorization" "CORS: Allow-Headers includes Authorization"

# ── CORS preflight (OPTIONS) ──
# Caddyfile.local doesn't have an explicit OPTIONS handler, but the header
# directives apply to all responses. The upstream werd-api may return 405
# for OPTIONS, but Caddy still injects the CORS headers.
preflight=$(curl -sI -X OPTIONS "$CADDY_API/healthz" \
  -H "Origin: http://localhost:13080" \
  -H "Access-Control-Request-Method: POST" 2>/dev/null)

pf_origin=$(echo "$preflight" | grep -i "^Access-Control-Allow-Origin:" | head -1 | sed 's/^[^:]*: *//' | tr -d '\r\n')
assert_eq "*" "$pf_origin" "CORS preflight: Allow-Origin = *"

# ── Request body limit (10MB) ──
# Caddy should reject a request body >10MB with 413 or a connection reset.
status=$(dd if=/dev/zero bs=1M count=11 2>/dev/null | \
  curl -sf -o /dev/null -w '%{http_code}' \
    -X POST "$CADDY_API/healthz" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @- 2>/dev/null || echo "000")

if [ "$status" = "413" ] || [ "$status" = "000" ]; then
  pass "Body limit enforced: 11MB POST rejected (status: $status)"
else
  fail "Body limit NOT enforced: 11MB POST returned $status (expected 413 or connection reset)"
fi
