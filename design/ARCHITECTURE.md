# Werd Architecture

See [README.md](../README.md) for the full architecture overview, stack table, and data flow descriptions.

This document covers implementation-level architectural decisions.

## System Layers

1. **Werd API Server** (Go) — orchestration layer, source of truth for multi-project state, cross-posting, notification routing
2. **Werd Dashboard** (React SPA) — presentation layer, communicates exclusively with the API server
3. **Lightweight sub-services** (ntfy, changedetection.io, RSSHub, Umami) — managed by the API server, not directly by the dashboard
4. **Infrastructure** (PostgreSQL, Redis, Caddy) — shared resources

## API Server Responsibilities

- Authentication and session management
- Multi-project CRUD and member management
- Service provisioning/deprovisioning per project (ntfy topics, changedetection watches, Umami sites)
- Cross-posting to social media platforms (direct API integration, no intermediary service)
- Post scheduling via persistent job queue (river)
- OAuth token management for platform connections
- Webhook ingestion, deduplication, and routing
- Notification routing engine (evaluate rules, fan out to ntfy / dashboard / webhooks)
- LLM-assisted response drafting (direct API call)
- Background sync with sub-service APIs
- Real-time push to dashboard via WebSocket (backed by PostgreSQL LISTEN/NOTIFY)

## Data Flow

```
Dashboard ──REST/WS──> API Server ──> PostgreSQL (werd DB)
                           │
                           ├──> Social Platform APIs (X, LinkedIn, Bluesky, Reddit, Mastodon, ...)
                           ├──> ntfy API (per-project topics)
                           ├──> changedetection.io API (per-project watches)
                           ├──> Umami API (per-project sites)
                           ├──> RSSHub (feed generation)
                           └──> LLM API (optional, response drafting)

Monitors ──webhook──> API Server ──> alert dedup ──> notification routing
```
