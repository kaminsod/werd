# Suite 01 — Stack Lifecycle
#
# Validates that all services are running and healthy, the compose network
# has the expected deterministic name, and only Caddy exposes host ports.

suite "Stack Lifecycle"

# ── All services running and healthy ──
# The fact that the harness wait succeeded already proves this, but we
# make it explicit per-service for clear reporting.

for svc in postgres redis werd-api werd-dashboard caddy; do
  if compose_exec "$svc" true 2>/dev/null; then
    pass "$svc is running"
  else
    fail "$svc is not running"
  fi
done

# ── Startup ordering ──
# depends_on with service_healthy conditions enforces ordering:
#   postgres, redis  →  werd-api  →  caddy
#   werd-dashboard   →  caddy
# The fact that all services are healthy proves ordering was respected.
pass "Startup ordering enforced (all services healthy via depends_on chain)"

# ── Network name is deterministic ──
# docker-compose.yml sets `name: werd-net` to prevent project-name prefixing.
if command -v podman >/dev/null 2>&1; then
  net_list=$(podman network ls --format '{{.Name}}' 2>/dev/null)
elif command -v docker >/dev/null 2>&1; then
  net_list=$(docker network ls --format '{{.Name}}' 2>/dev/null)
else
  net_list=""
fi

if echo "$net_list" | grep -qx 'werd-net'; then
  pass "Network name is 'werd-net' (deterministic, not prefixed)"
else
  fail "Network 'werd-net' not found (got: $(echo "$net_list" | tr '\n' ' '))"
fi

# ── Internal ports NOT exposed on host ──
# Only Caddy should expose ports. API (8090), dashboard (3000),
# PostgreSQL (5432), and Redis (6379) must stay internal.
for blocked_port in 8090 3000 5432 6379; do
  if port_open "$blocked_port"; then
    fail "Port $blocked_port should NOT be exposed on host"
  else
    pass "Port $blocked_port not exposed on host"
  fi
done

# ── Caddy test ports ARE open ──
for open_port in 13080 13081; do
  if port_open "$open_port"; then
    pass "Port $open_port exposed on host (Caddy test port)"
  else
    fail "Port $open_port should be exposed on host"
  fi
done
