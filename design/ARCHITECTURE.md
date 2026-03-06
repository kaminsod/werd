# Werd Architecture

See [README.md](../README.md) for the full architecture overview, stack table, and data flow descriptions.

This document covers implementation-level architectural decisions.

## System Layers

1. **Werd API Server** (Go) — orchestration layer, source of truth for multi-project state
2. **Werd Dashboard** (React SPA) — presentation layer, communicates exclusively with the API server
3. **Sub-services** (Postiz, Activepieces, Mattermost, etc.) — managed by the API server, not directly by the dashboard
4. **Infrastructure** (PostgreSQL, Redis, ClickHouse, Caddy) — shared resources

## API Server Responsibilities

- Authentication and session management
- Multi-project CRUD and member management
- Service provisioning/deprovisioning per project
- Webhook ingestion, deduplication, and routing
- Background sync with sub-service APIs
- Real-time push to dashboard via WebSocket (backed by PostgreSQL LISTEN/NOTIFY)

## Data Flow

```
Dashboard ──REST/WS──> API Server ──> PostgreSQL (werd DB)
                           │
                           ├──> Mattermost API (per-project team)
                           ├──> Postiz API (per-project org)
                           ├──> Activepieces API (flows)
                           ├──> ntfy API (per-project topics)
                           ├──> Plausible API (per-project site)
                           └──> changedetection.io API (per-project watches)

Monitors ──webhook──> API Server ──> alert dedup ──> notification routing
```
