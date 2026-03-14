# Phase 1: Core Infrastructure

Foundation: compose file, networking, databases, reverse proxy.

## Tasks

| # | Task | Status | Details | Dependencies |
|---|---|---|---|---|
| 1.1 | Project scaffolding | Done | Directory structure, compose skeleton, .env.example, Makefiles, CI scripts | — |
| 1.2 | PostgreSQL deployment | Done | Shared PostgreSQL 17, per-service users/databases, tuned config, init-db.sh | 1.1 |
| 1.3 | Redis deployment | Done | Shared Redis 7 with tuned config, per-service DB isolation, AOF persistence | 1.1 |
| 1.4 | Caddy reverse proxy | Done | Caddyfile with subdomain routing, auto-TLS, security headers, CORS, local mode variant | 1.1 |
| 1.5 | Container networking | Done | werd-net bridge, DNS resolution, rootless Podman config | 1.1 |
| 1.6 | Health checks & restart policies | Done | Probes for all services, depends_on with conditions, start_period, distroless healthcheck binary | 1.2–1.5 |
| 1.7 | Secret generation script | Done | tools/generate-secrets.sh | 1.1 |
