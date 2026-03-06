# Phase 1: Core Infrastructure

Foundation: compose file, networking, databases, reverse proxy.

## Tasks

| # | Task | Status | Details | Dependencies |
|---|---|---|---|---|
| 1.1 | Project scaffolding | Done | Directory structure, compose skeleton, .env.example, Makefiles, CI scripts | — |
| 1.2 | PostgreSQL + Redis deployment | Not started | Shared PostgreSQL 17 with init script for all databases, Redis 7 with auth | 1.1 |
| 1.3 | Caddy reverse proxy | Not started | Caddyfile with subdomain routing, auto-TLS, security headers | 1.1 |
| 1.4 | Container networking | Not started | werd-net bridge, DNS resolution, rootless Podman config | 1.1 |
| 1.5 | Health checks & restart policies | Not started | Probes for all services, depends_on with conditions | 1.2-1.4 |
| 1.6 | Secret generation script | Not started | tools/generate-secrets.sh (done in scaffolding) | 1.1 |
| 1.7 | ClickHouse + Temporal | Not started | ClickHouse for Plausible, Temporal for Postiz | 1.2 |
