# Phase 1: Core Infrastructure

Foundation: compose file, networking, databases, reverse proxy.

## Tasks

| # | Task | Status | Details | Dependencies |
|---|---|---|---|---|
| 1.1 | Project scaffolding | Done | Directory structure, compose skeleton, .env.example, Makefiles, CI scripts | — |
| 1.2 | PostgreSQL deployment | Not started | Shared PostgreSQL 17 with init script for all databases | 1.1 |
| 1.3 | Redis deployment | Not started | Shared Redis 7 with password auth, AOF persistence | 1.1 |
| 1.4 | Caddy reverse proxy | Not started | Caddyfile with subdomain routing, auto-TLS, security headers | 1.1 |
| 1.5 | Container networking | Not started | werd-net bridge, DNS resolution, rootless Podman config | 1.1 |
| 1.6 | Health checks & restart policies | Not started | Probes for all services, depends_on with conditions | 1.2–1.5 |
| 1.7 | Secret generation script | Done | tools/generate-secrets.sh | 1.1 |
| 1.8 | ClickHouse + Temporal | Not started | ClickHouse for Plausible, Temporal for Postiz | 1.2 |
