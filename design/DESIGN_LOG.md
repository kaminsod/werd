# Design Decision Log

Chronological record of key architectural and technology decisions.

## 2026-03-05: Backend Language — Go

**Decision:** Use Go for the API server and monitoring bots.

**Rationale:**
- 15-30 MB RSS, 5-15 MB container images — critical when sharing a box with 10+ services
- Goroutines are the natural fit for concurrent API polling + webhook ingestion + background sync
- pgx v5 is the strongest PostgreSQL driver across ecosystems (LISTEN/NOTIFY, JSONB, arrays)
- Single static binary — no runtime, no node_modules
- Official Mattermost Go client library

**Alternatives considered:** TypeScript/Node.js (shared language with frontend, but higher memory, weaker PG integration), Rust (over-engineered for I/O-bound workload).

## 2026-03-05: Frontend — React + TypeScript SPA

**Decision:** React 19 + TypeScript with Vite, TanStack Query, Zustand, Tailwind + shadcn/ui.

**Rationale:** User requirement. Large ecosystem, strong tooling, good developer pool. Type contract with backend via OpenAPI spec → openapi-typescript.

## 2026-03-05: Workflow Automation — Activepieces (MIT CE)

**Decision:** Use Activepieces Community Edition instead of n8n.

**Rationale:** n8n's "Sustainable Use License" is not OSI-approved open source. Activepieces CE is MIT-licensed. Multi-tenancy limitation (CE is single-project) handled by Werd API orchestration layer.

## 2026-03-05: Container Runtime — Podman-first

**Decision:** Target Podman as primary, Docker as secondary.

**Rationale:** Rootless by default, no daemon, compatible with docker-compose via podman socket. Same compose file works for both.

## 2026-03-05: Residential Access — FRP Tunnel

**Decision:** Use FRP (Apache-2.0) for residential/NAT deployments.

**Rationale:** Most mature tunnel (105K stars), pairs well with Caddy, fully self-hostable. Requires a small relay VPS ($3-5/mo). Alternatives: Pangolin (all-in-one but uses Traefik), Rathole (faster but fewer features).

## 2026-03-05: Database — PostgreSQL as Core Store

**Decision:** PostgreSQL for Werd core data and shared instance for sub-services.

**Rationale:** Relational model maps naturally to project → resources hierarchy. JSONB for flexible service configs. LISTEN/NOTIFY for real-time events. RLS for defense-in-depth isolation. Already required by Postiz, Activepieces, Mattermost, Plausible.

## 2026-03-05: Kubernetes Strategy — Compose-first, k8s for Scale

**Decision:** Primary deployment via compose, Kubernetes (k3s) as scale-out path.

**Rationale:** k3s adds ~750 MB RAM overhead. Compose is simpler for single-box. Helm charts exist for all infrastructure components. Migration path: Kompose → manual refinement → Helm chart.
