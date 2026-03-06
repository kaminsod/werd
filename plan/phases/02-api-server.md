# Phase 2: Werd API Server (Go Backend)

Core backend — auth, multi-project orchestration, webhook ingestion.

## Tasks

| # | Task | Status | Details | Dependencies |
|---|---|---|---|---|
| 2.1 | Go project scaffolding | Not started | chi router, pgx pool, sqlc config, Dockerfile | 1.1 |
| 2.2 | Database migrations | Not started | goose migration files for core schema | 2.1, 1.2 |
| 2.3 | Authentication system | Not started | Local accounts, bcrypt, JWT sessions | 2.2 |
| 2.4 | Multi-project CRUD | Not started | Projects, members, roles | 2.3 |
| 2.5 | Service provisioning engine | Not started | Provision sub-service resources per project | 2.4, 3.x |
| 2.6 | Webhook ingestion | Not started | Receive, tag, deduplicate, persist alerts | 2.4 |
| 2.7 | Notification routing engine | Not started | Evaluate rules, fan out to destinations | 2.6 |
| 2.8 | Background sync jobs | Not started | Poll services for state changes | 2.5 |
| 2.9 | WebSocket real-time push | Not started | Live alert feed via LISTEN/NOTIFY | 2.6 |
| 2.10 | OpenAPI spec generation | Not started | swag annotations, spec for frontend types | 2.4-2.9 |
