# Design Decision Log

Chronological record of key architectural and technology decisions.

## 2026-03-05: Backend Language — Go

**Decision:** Use Go for the API server and monitoring bots.

**Rationale:**
- 15-30 MB RSS, 5-15 MB container images — critical when sharing a box with multiple services
- Goroutines are the natural fit for concurrent API polling + webhook ingestion + background sync
- pgx v5 is the strongest PostgreSQL driver across ecosystems (LISTEN/NOTIFY, JSONB, arrays)
- Single static binary — no runtime, no node_modules

**Alternatives considered:** TypeScript/Node.js (shared language with frontend, but higher memory, weaker PG integration), Rust (over-engineered for I/O-bound workload).

## 2026-03-05: Frontend — React + TypeScript SPA

**Decision:** React 19 + TypeScript with Vite, TanStack Query, Zustand, Tailwind + shadcn/ui.

**Rationale:** User requirement. Large ecosystem, strong tooling, good developer pool. Type contract with backend via OpenAPI spec → openapi-typescript.

## 2026-03-05: Container Runtime — Podman-first

**Decision:** Target Podman as primary, Docker as secondary.

**Rationale:** Rootless by default, no daemon, compatible with docker-compose via podman socket. Same compose file works for both.

## 2026-03-05: Residential Access — FRP Tunnel

**Decision:** Use FRP (Apache-2.0) for residential/NAT deployments.

**Rationale:** Most mature tunnel (105K stars), pairs well with Caddy, fully self-hostable. Requires a small relay VPS ($3-5/mo). Alternatives: Pangolin (all-in-one but uses Traefik), Rathole (faster but fewer features).

## 2026-03-05: Database — PostgreSQL as Core Store

**Decision:** PostgreSQL for Werd core data and shared instance for sub-services.

**Rationale:** Relational model maps naturally to project → resources hierarchy. JSONB for flexible service configs. LISTEN/NOTIFY for real-time events. RLS for defense-in-depth isolation.

## 2026-03-05: Kubernetes Strategy — Compose-first, k8s for Scale

**Decision:** Primary deployment via compose, Kubernetes (k3s) as scale-out path.

**Rationale:** k3s adds ~750 MB RAM overhead. Compose is simpler for single-box. Helm charts exist for all infrastructure components. Migration path: Kompose → manual refinement → Helm chart.

## 2026-03-13: Stack Simplification — Remove Heavy Dependencies

**Decision:** Remove Mattermost, Postiz (+ Temporal + Elasticsearch), Activepieces, Plausible (+ ClickHouse), and Folo. Replace Plausible with Umami. Build cross-posting and notification routing directly into the Werd API.

**Rationale:**

The original stack included 8+ heavyweight third-party services consuming 5-11 GB RAM. Analysis showed most functionality was either already planned for the Werd API or could be built with far less complexity than the dependency overhead:

- **Mattermost** (~250 MB) — Full team chat suite used only for posting automated alerts. The Werd Dashboard alert feed + ntfy push notifications cover this entirely.
- **Postiz** (~350 MB) + **Temporal** (~200 MB) + **Elasticsearch** (~500 MB-1 GB) — Three services for cross-posting. Postiz API limited to 3 endpoints / 30 req/hr. Direct social platform API integration in Go is straightforward and removes the rate limit.
- **Activepieces** (1.5-8 GB) — Workflow automation engine. The heaviest single dependency. All routing logic was already planned for the Werd API notification routing engine (task 2.7). LLM drafting is a single HTTP call.
- **Plausible** (~175 MB) + **ClickHouse** (~150 MB) — Two services for web analytics. Replaced by Umami (MIT, PostgreSQL-only, ~240 MB), eliminating ClickHouse entirely.
- **Folo** (~100 MB) — RSS reader for manual browsing, redundant with the dashboard alert feed.

**Impact:**
- 7 containers removed, 4 PostgreSQL databases eliminated
- RAM reduced from ~5-11 GB (third-party) to ~440 MB
- Minimum server requirement dropped from 8 GB to 2-4 GB RAM
- Secrets reduced from ~12 to ~5

**What we build instead:**
- Social platform posting adapters in `internal/integration/` (~100-200 lines per platform)
- OAuth token management + encrypted storage in `werd` DB
- Post scheduling via river persistent job queue
- LLM response drafting via HTTP client call

**Services retained:** ntfy (push notifications), RSSHub (feed generation), changedetection.io (web monitoring), Umami (web analytics) — all lightweight, purpose-built, hard to replicate.
