# Phase 1 Integration Tests

End-to-end tests that validate the full compose stack — PostgreSQL, Redis, Werd API, Werd Dashboard, and Caddy — works correctly as a unit.

## Quick Start

```bash
# From repo root:
make integration-test

# Or directly:
./tests/integration/run.sh

# Leave stack running after tests for debugging:
make integration-test-keep
# or: WERD_TEST_KEEP=1 ./tests/integration/run.sh
```

## Requirements

- **Container runtime**: Podman (preferred) or Docker
- **Compose tool**: podman-compose or docker compose
- **CLI tools**: curl, openssl (for secret generation)
- **No conflicting stack**: The tests create a `werd-net` bridge network, so no other compose stack using that name can be running simultaneously

## Architecture

### Test Runner

```
tests/integration/
  run.sh                          # Harness: setup → run → teardown
  lib.sh                          # Shared utilities and assertion helpers
  docker-compose.test.yml         # Compose override for test environment
  suites/
    01-stack-lifecycle.sh         # Stack health, network, port exposure
    02-postgres.sh                # DB isolation, config tuning
    03-redis.sh                   # Auth, DB isolation, config
    04-werd-api.sh                # Health endpoint, reachability
    05-werd-dashboard.sh          # SPA serving and routing
    06-caddy.sh                   # Headers, CORS, body limit
    07-dns.sh                     # Inter-service DNS resolution
    08-persistence.sh             # Data survives restart
```

### How It Works

The harness (`run.sh`) manages the full lifecycle:

1. **Detect compose tool** — podman-compose preferred, docker compose as fallback.
2. **Clean up** — tear down any leftover stack from a previous run (`down -v`).
3. **Generate test environment** — copy `.env.example` to `.env.test`, run `generate-secrets.sh` to fill in real secrets, set `WERD_DOMAIN=localhost` and `WERD_ACCESS_MODE=local`.
4. **Build images** — `werd-api` and `werd-dashboard` have `build:` directives and need to be built from source.
5. **Start stack** — `compose up -d` with project name `werd-test` for isolation.
6. **Wait for healthy** — poll Caddy endpoints (`http://localhost:13080` for dashboard, `http://localhost:13081/healthz` for API). Since Caddy has `depends_on` with `service_healthy` on all upstream services, Caddy responding means the entire chain is healthy. Timeout: 120 seconds.
7. **Run suites** — source each `suites/*.sh` file in order. Each file calls the `suite()` function to print a header, then runs assertions using helpers from `lib.sh`.
8. **Print summary** — total pass/fail/skip counts.
9. **Tear down** — `compose down -v` removes containers, volumes, and network. Skip if `WERD_TEST_KEEP=1`.
10. **Exit code** — 0 if all tests pass, 1 if any fail.

### Compose Override

The base `docker-compose.yml` uses the production Caddyfile (subdomain routing, auto-TLS). Tests need `Caddyfile.local` instead (port-based routing on localhost, no TLS, no domain). The override file (`docker-compose.test.yml`) swaps the Caddyfile volume and remaps host ports:

| Production | Test Override | Why |
|---|---|---|
| `Caddyfile` (subdomain + TLS) | `Caddyfile.local` (port-based, no TLS) | No domain/DNS needed |
| Host ports 80, 443 | Host ports 13080, 13081 | Avoids privileged ports and conflicts |
| Admin API healthcheck (:2019) | Proxy healthcheck (:3081/healthz) | Matches local Caddyfile ports |

The `CADDYFILE_PATH` environment variable is set by the harness to the absolute path of `Caddyfile.local`, avoiding compose path resolution differences between podman-compose and docker compose.

### Assertion Helpers

`lib.sh` provides these assertion functions, used by all suite files:

| Function | Purpose |
|---|---|
| `assert_eq expected actual desc` | Strict string equality |
| `assert_contains haystack needle desc` | Substring match (case-insensitive) |
| `assert_not_contains haystack needle desc` | Negative substring match |
| `assert_status code url desc` | HTTP response code check |
| `pass desc` / `fail desc` / `skip desc` | Manual pass/fail/skip |
| `port_open port` | Check if TCP port is listening on localhost |
| `compose_exec service cmd...` | Run a command inside a service container |

## Test Suites

### 01 — Stack Lifecycle

Validates the compose stack as a whole.

| Test | What It Proves |
|---|---|
| Each service is running | All 5 containers started and are accessible |
| Startup ordering | `depends_on` with `service_healthy` enforced the correct chain |
| Network name is `werd-net` | `name: werd-net` in compose prevents project-name prefixing |
| Ports 8090, 3000, 5432, 6379 NOT exposed | Internal services not leaking to host |
| Ports 13080, 13081 exposed | Caddy test ports are reachable |

### 02 — PostgreSQL

Validates `init-db.sh` database/user creation and `postgres.conf` tuning.

| Test | What It Proves |
|---|---|
| `werd` DB accessible by `werd` user | Primary database and superuser work |
| `umami` DB accessible by `umami` user | Per-service user creation by init-db.sh |
| `umami` cannot access `werd` DB | `REVOKE ALL FROM PUBLIC` isolation |
| `werd` can access `umami` DB | Superuser bypass (expected, documented) |
| `shared_buffers = 512MB` | Custom postgres.conf loaded (not defaults) |
| `log_min_duration_statement = 1s` | Slow query logging configured |
| `timezone = UTC` | Consistent timestamps across services |
| Password authentication works | Env var injection of secrets is correct |

### 03 — Redis

Validates AUTH, database isolation, and `redis.conf` tuning.

| Test | What It Proves |
|---|---|
| Unauthenticated PING rejected | `--requirepass` enforced |
| Authenticated PING succeeds | Password from `.env` works |
| Write/read on DB 0 | Werd API database functional |
| Write/read on DB 2 | RSSHub database functional |
| Key in DB 0 not visible in DB 2 | Database number isolation works |
| `maxmemory = 268435456` (256MB) | Custom redis.conf loaded |
| `maxmemory-policy = allkeys-lru` | Eviction policy configured |
| `appendonly = yes` | AOF persistence enabled |

### 04 — Werd API

Validates the health endpoint through the full proxy chain.

| Test | What It Proves |
|---|---|
| `/healthz` returns 200 via Caddy | API is live, Caddy→API proxy works |
| `/healthz` body is `{"status":"ok"}` | Endpoint returns expected JSON |
| Port 8090 not exposed on host | API only reachable through Caddy |

### 05 — Werd Dashboard

Validates SPA serving and client-side routing.

| Test | What It Proves |
|---|---|
| Dashboard serves HTML via Caddy | Build artifacts served correctly |
| `/projects/123/settings` returns 200 | SPA `try_files` routing works (not 404) |
| Arbitrary deep path returns index.html | Client-side routing supported |
| Port 3000 not exposed on host | Dashboard only reachable through Caddy |

### 06 — Caddy Reverse Proxy

Validates security headers, CORS, and request limits.

| Test | What It Proves |
|---|---|
| `X-Content-Type-Options: nosniff` | MIME sniffing protection |
| `X-Frame-Options: DENY` | Clickjacking protection |
| `Referrer-Policy` set | Referrer leakage prevention |
| `Permissions-Policy` present | Browser feature restrictions |
| `Server` header removed | Server fingerprint hidden |
| CORS `Allow-Origin: *` on API | Local dev CORS configured |
| CORS `Allow-Methods` includes POST, DELETE | Mutation methods allowed |
| CORS `Allow-Headers` includes Authorization | Auth header allowed cross-origin |
| CORS preflight has correct `Allow-Origin` | OPTIONS requests get CORS headers |
| 11MB POST rejected (413/reset) | `request_body max_size 10MB` enforced |

### 07 — DNS Resolution

Validates compose-provided service discovery.

| Test | What It Proves |
|---|---|
| Caddy can resolve all 5 service names | DNS works on werd-net bridge |
| Postgres can resolve redis (cross-check) | DNS works between non-Caddy services too |

### 08 — Persistence

Validates data durability across container restarts.

| Test | What It Proves |
|---|---|
| PostgreSQL row survives restart | `postgres-data` volume is mounted and durable |
| Redis key survives restart | AOF persistence + `redis-data` volume working |

## Debugging

### Leave stack running

```bash
make integration-test-keep
```

Then inspect:
```bash
# Service status
podman-compose -p werd-test ps

# Service logs
podman-compose -p werd-test logs werd-api

# Shell into a container
podman-compose -p werd-test exec caddy sh
podman-compose -p werd-test exec postgres psql -U werd -d werd

# Hit endpoints directly
curl http://localhost:13081/healthz
curl -I http://localhost:13080
```

### Teardown manually

```bash
podman-compose \
  -f src/deploy/compose/docker-compose.yml \
  -f tests/integration/docker-compose.test.yml \
  -p werd-test down -v
```

### Common Issues

| Problem | Cause | Fix |
|---|---|---|
| "No compose tool found" | Missing podman-compose or docker compose | `pip install podman-compose` or install Docker |
| Timeout waiting for healthy | Image build failed or service crashed | Run `make integration-test-keep`, check `compose logs` |
| Port 13080/13081 already in use | Previous test run not cleaned up | Run teardown command above, or the test harness cleans up automatically on next run |
| "werd-net" network conflict | Another compose stack using `werd-net` | Stop the other stack first |
| Caddy TLS errors | Wrong Caddyfile mounted | Verify `docker-compose.test.yml` override is being picked up |

## Design Decisions

**Why Bash, not Go/Node test frameworks?**
The infrastructure under test is compose-based. The assertions are simple (HTTP status codes, psql queries, redis-cli commands). Shell scripts with `curl`, `psql`, and `redis-cli` (via `compose exec`) are the natural tool — no extra dependencies, no compilation step, runs anywhere compose runs.

**Why Caddyfile.local instead of production Caddyfile?**
The production Caddyfile uses subdomain routing (`api.yourdomain.com`) with automatic TLS. Tests would need DNS setup, domain configuration, and TLS cert handling. The local Caddyfile uses port-based routing on localhost with no TLS — everything works out of the box.

**Why high ports (13080/13081)?**
Avoids requiring root/sysctl for privileged port binding (ports < 1024), and avoids conflicts with any running production stack.

**Why poll Caddy instead of parsing `compose ps`?**
The `compose ps --format json` output differs between podman-compose and docker compose. Polling Caddy with curl is portable and also tests the full dependency chain — Caddy only responds after all its upstream services are healthy.

**Why `--project-name werd-test`?**
Isolates test containers, volumes, and networks from any production stack. Allows safe `down -v` without affecting production data.
