# Progress

## Current Phase: 2 — Werd API Server

### Completed
- [x] 1.1 Project scaffolding — directory structure, compose skeleton, .env.example, scripts, CI, Makefiles
- [x] 1.7 Secret generation script — tools/generate-secrets.sh
- [x] 2.1 Go project scaffolding — Go module, net/http server, sqlc config, Dockerfile, compose service
- [x] 4.1 React project scaffolding — Vite, React 19, TypeScript, Tailwind, directory structure
- [x] 1.2 PostgreSQL deployment — per-service users/databases, tuned config, init-db.sh
- [x] 1.3 Redis deployment — tuned config, per-service DB numbering, AOF persistence
- [x] 1.4 Caddy reverse proxy — security headers, CORS, WebSocket, production + local Caddyfiles
- [x] 1.5 Container networking — werd-net bridge, DNS diagnostics, rootless Podman checker
- [x] 1.6 Health checks & restart policies — start_period, distroless healthcheck binary, probes for all services
- [x] 2.2 Database migrations — goose initial schema: 10 enums, 11 tables, updated_at trigger, 12 indexes
- [x] 2.3 Authentication system — chi router, bcrypt, JWT, admin seeding, login/me/change-password
- [x] 2.4 Multi-project CRUD — project + member CRUD, role-based access, RequireProjectMember middleware
- [x] 2.6 Webhook ingestion — upsert dedup, keyword matching, alert/keyword CRUD, internal API key auth
- [x] 2.7 Notification routing engine — rule CRUD, async dispatch to ntfy/webhooks, severity filtering
- [x] 2.8 Social platform integration — connections CRUD, posts CRUD, Bluesky adapter, synchronous publish
- [x] Monitor sources CRUD — 5 endpoints for configuring per-project monitoring sources
- [x] 3.1-3.4 Phase 3 services — ntfy, changedetection.io, RSSHub, Umami activated in compose + Caddy

### In Progress
- [ ] Phase 4: Dashboard UI (Phase A foundation complete)

### Not Started
- [ ] 2.9 Post scheduling (river job queue)
- [ ] Dashboard Phase B: alerts + keywords pages
- [ ] Dashboard Phase C: sources + rules pages
- [ ] Dashboard Phase D: connections + posts pages
- [ ] Dashboard Phase E: members + settings pages

## Phase Overview

| Phase | Name | Status |
|---|---|---|
| 1 | Core Infrastructure | Done |
| 2 | Werd API Server | In progress |
| 3 | Lightweight Services | Done |
| 4 | Werd Dashboard | In progress (Phase A done) |
| 5 | Monitoring Pipeline | Not started |
| 6 | Notification & Routing | Not started |
| 7 | Publishing Pipeline | Not started |
| 8 | Network Access & Tunnel | Not started |
| 9 | Kubernetes Deployment | Not started |
| 10 | Hardening & Documentation | Not started |

## Architecture Simplification (2026-03-13)

Removed heavy dependencies (Mattermost, Postiz, Temporal, Elasticsearch, Activepieces, Plausible, ClickHouse, Folo). Cross-posting, notification routing, and scheduling built directly into the Werd API. Plausible replaced by Umami (PostgreSQL-only). See `design/DESIGN_LOG.md` for full rationale.
