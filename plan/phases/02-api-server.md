# Phase 2: Werd API Server (Go Backend)

Core backend — auth, multi-project orchestration, webhook ingestion, cross-posting, notification routing.

## Tasks

| # | Task | Status | Details | Dependencies |
|---|---|---|---|---|
| 2.1 | Go project scaffolding | Done | Go module, net/http server, sqlc config, Dockerfile, compose service | 1.1 |
| 2.2 | Database migrations | Done | goose initial schema: 10 enums, 11 tables, updated_at trigger, 12 indexes | 2.1, 1.2 |
| 2.3 | Authentication system | Done | chi router, bcrypt, JWT, admin seeding, login/me/change-password endpoints | 2.2 |
| 2.4 | Multi-project CRUD | Done | Project + member CRUD, role-based access, RequireProjectMember middleware | 2.3 |
| 2.5 | Service provisioning engine | Not started | Provision ntfy topics, changedetection watches, Umami sites per project | 2.4, 3.x |
| 2.6 | Webhook ingestion | Done | Upsert dedup, keyword matching, alert/keyword CRUD, internal API key auth | 2.4 |
| 2.7 | Notification routing engine | Done | Rule CRUD, async routing on alert ingest, ntfy + webhook dispatchers, severity comparison | 2.6 |
| 2.8 | Social platform integration | Done | Platform connections CRUD, published posts CRUD, adapter interface + Bluesky AT Protocol adapter, synchronous publish | 2.4 |
| 2.9 | Post scheduling | Not started | river persistent job queue for scheduled posts, status tracking | 2.8 |
| 2.10 | Background sync jobs | Not started | Poll sub-services for state changes, sync engagement metrics | 2.5 |
| 2.11 | WebSocket real-time push | Not started | Live alert feed via LISTEN/NOTIFY | 2.6 |
| 2.12 | OpenAPI spec generation | Not started | swag annotations, spec for frontend types | 2.4–2.11 |
