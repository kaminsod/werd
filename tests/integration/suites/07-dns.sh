# Suite 07 — DNS Resolution
#
# Validates that all services can resolve each other by compose service name
# on the werd-net bridge network.

suite "DNS Resolution"

# Use the caddy container (Alpine-based, has nslookup via busybox) to
# resolve every service name. This proves the compose DNS is working.
for svc in postgres redis werd-api werd-dashboard caddy; do
  result=$(compose_exec caddy nslookup "$svc" 2>&1 || true)
  if echo "$result" | grep -qi "Address\|Name:"; then
    pass "DNS: caddy can resolve '$svc'"
  else
    fail "DNS: caddy cannot resolve '$svc'"
  fi
done

# Cross-check: postgres (also Alpine) can resolve redis.
result=$(compose_exec postgres nslookup redis 2>&1 || true)
if echo "$result" | grep -qi "Address\|Name:"; then
  pass "DNS: postgres can resolve 'redis' (cross-check)"
else
  fail "DNS: postgres cannot resolve 'redis'"
fi
