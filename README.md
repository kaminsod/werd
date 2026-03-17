# Werd

**Spreading the word** — an open-source, self-hosted social media automation and monitoring platform.

Werd is a unified stack for managing your online presence across social media platforms, technical blogs, discussion forums, and community channels. It centralizes cross-posting, scheduling, monitoring, sentiment tracking, and notification routing into a single self-hosted deployment — no paid SaaS dependencies.

**Core design goals:**
- **Single-box deployable** — runs on a single machine via `docker-compose.yml`, on any network (cloud VPS, residential behind NAT, or local-only)
- **Podman-first** — targets Podman as the primary container runtime, with Docker support as secondary
- **Lightweight** — core stack runs in ~1 GB RAM; heavy third-party services replaced by built-in Go modules and purpose-built lightweight tools
- **Scales out when needed** — each component can be independently scaled in a distributed Kubernetes environment (tested locally with k3s)
- **Multi-project isolation** — supports multiple fully-isolated projects, each with its own platform connections, keywords, notification preferences, and team members
- **Fully open source** — every component is self-hostable with an OSI-approved license (the only external dependency is an optional LLM API)

## What It Does

- **Manage everything from one UI** — a custom React-based web dashboard provides a single pane of glass for all projects, services, alerts, and publishing across the entire stack
- **Run multiple isolated projects** — each project has its own keyword sets, monitored sources, platform connections, notification channels, and team members with role-based access
- **Cross-post everywhere from one place** — schedule and publish to LinkedIn, X, Bluesky, Reddit, Mastodon, and more via direct platform API integration built into the Werd API
- **Monitor mentions and keywords** — track brand mentions, competitor activity, and relevant discussions across Reddit, Hacker News, the web, news sites, and RSS feeds
- **Route notifications intelligently** — funnel alerts to push notifications (ntfy) and the dashboard alert feed, with configurable per-project routing rules
- **Draft AI-assisted responses** — automatically generate contextual response drafts for human review and approval via any LLM API
- **Syndicate blog content** — publish once on your blog, auto-distribute to Dev.to, Hashnode, and social platforms with canonical URLs preserved
- **Track analytics** — privacy-friendly web analytics via Umami (no cookies, no ClickHouse — just PostgreSQL)

## Architecture

```
                          +------------------------+
                          |     Werd Dashboard     |  React + TypeScript SPA
                          |  (multi-project mgmt)  |  (projects, alerts, publish, config)
                          +-----------+------------+
                                      |
                          +-----------v------------+
                          |     Werd API Server    |  Go backend
                          |  (orchestration, auth, |  (chi + pgx + sqlc + river)
                          |   cross-posting, route)|
                          +-----------+------------+
                                      |
                    +-----------------+-----------------+
                    |                 |                 |
                    v                 v                 v
          +--------+------+  +-------+-------+  +------+-------+
          |   Caddy       |  |  PostgreSQL   |  |    Redis     |
          |  (reverse     |  | (werd + umami)|  |  (caching,   |
          |   proxy, TLS) |  |               |  |   queues)    |
          +---------------+  +---------------+  +--------------+
                   |
    +--------------+----------+----------+----------+
    |              |          |          |          |
+---v---+    +----v----+ +---v---+ +---v---+  +---v--------+
| ntfy  |    | change  | | RSS  | | Umami |  | LinkedIn,  |
| (push)|    | detect  | | Hub  | | (ana- |  | X, Bluesky,|
|       |    | .io     | |      | | lytics|  | Reddit,    |
+-------+    +---------+ +------+ +-------+  | Mastodon...|
                                              +------------+
                                                (direct API)

    [Residential / NAT access]
    +-------------+          +-------------+
    | FRP client  | -------> | FRP server  |  (on relay VPS)
    | (home box)  |  tunnel  | + Caddy     |  ($3-5/mo VPS)
    +-------------+          +-------------+
```

## Stack

Every component is open source (OSI-approved licenses) and fully self-hostable. The only external dependency is an LLM API for AI-assisted response drafting (optional).

### Custom Components (This Project)

| Component | Tech | Role |
|---|---|---|
| **Werd API Server** | Go (chi, pgx, sqlc, river) | Core backend — multi-project orchestration, auth, cross-posting, notification routing, webhook ingestion, background jobs |
| **Werd Dashboard** | React + TypeScript (Vite) | SPA frontend — project management, unified alert feed, publishing, configuration, analytics |

### Lightweight Services

| Component | Tool | Role | License |
|---|---|---|---|
| Reverse proxy | [Caddy](https://github.com/caddyserver/caddy) | Automatic HTTPS, reverse proxy, subdomain routing | Apache-2.0 |
| Web & page monitoring | [changedetection.io](https://github.com/dgtlmoon/changedetection.io) | Track changes on any web page, keyword alerts | Apache-2.0 |
| RSS feed generation | [RSSHub](https://github.com/DIYgod/RSSHub) | Turn any website into an RSS feed (1000+ routes) | MIT |
| Push notifications | [ntfy](https://github.com/binwiederhier/ntfy) | Instant push alerts to phone/desktop via HTTP | Apache-2.0 |
| Web analytics | [Umami](https://github.com/umami-software/umami) | Privacy-friendly analytics (no cookies, PostgreSQL-only) | MIT |
| Reddit monitoring | Custom Go service using [Reddit API](https://www.reddit.com/dev/api/) | Stream subreddits for keyword matches | Apache-2.0 |
| Hacker News monitoring | Custom poller using [HN API](https://github.com/HackerNews/API) | Poll new stories/comments for keyword matches | Public API |
| Blog syndication | [cross-post CLI](https://github.com/shahednasser/cross-post) | Auto-publish to Dev.to and Hashnode | MIT |

### Infrastructure

| Component | Tool | Role | License |
|---|---|---|---|
| Database | [PostgreSQL 17](https://github.com/postgres/postgres) | Shared DB server — Werd core DB + Umami analytics | PostgreSQL License |
| Cache / queue | [Redis 7](https://github.com/redis/redis) | Shared cache, session store, and RSSHub cache | BSD-3-Clause |
| Tunnel (residential) | [FRP](https://github.com/fatedier/frp) | Expose services from behind NAT/CGNAT to the internet | Apache-2.0 |

### How the pieces connect

**Monitoring pipeline:**
1. **Reddit** — Go service streams submissions and comments from target subreddits, matches against per-project keyword lists, sends alerts via webhook to Werd API
2. **Hacker News** — Custom poller checks new stories/comments against keywords via the public HN API; RSSHub generates keyword-filtered HN feeds
3. **Web/News** — changedetection.io watches web pages (competitor blogs, procurement sites, news) for keyword-triggered changes
4. **RSS** — RSSHub turns platforms without native RSS into feeds; changedetection.io monitors feeds for keyword matches
5. **GitHub** — Native webhooks push events (stars, issues, PRs, discussions) to Werd API

**Routing pipeline:**
1. All monitoring sources send webhooks to the **Werd API Server**, which tags alerts with the matching project(s)
2. Werd API evaluates per-project **notification rules** and fans out to destinations (ntfy topics, dashboard alert feed, external webhooks)
3. High-priority alerts push to **ntfy** (per-project topic) for instant mobile/desktop notifications
4. Optionally, Werd API calls an **LLM API** to draft contextual responses, surfaced in the dashboard for human review
5. The **Werd Dashboard** aggregates alerts from all sources into a unified, filterable, per-project feed via WebSocket

**Publishing pipeline:**
1. Content is created and scheduled via the **Werd Dashboard**
2. Werd API publishes to connected social platforms via **direct API integration** (X, LinkedIn, Bluesky, Reddit, Mastodon, etc.)
3. Blog posts are syndicated to Dev.to and Hashnode via **cross-post CLI**
4. **Umami** tracks resulting traffic and referral sources (per-project site)

## Technical Details

### Multi-Project Architecture

Multi-project isolation is a core design principle. The **Werd API Server** is the orchestration layer that maps each project to isolated resources across all sub-services:

| Sub-Service | Per-Project Isolation Strategy |
|---|---|
| Social platforms | Per-project OAuth connections and credentials stored in `platform_connections` table (encrypted at rest) |
| ntfy | One **topic prefix** per project (e.g., `werd-proj1-high`, `werd-proj1-github`). ACL rules restrict access per project. |
| changedetection.io | Watches **tagged** by project ID. Webhook URLs include project routing info. |
| RSSHub | Feed URLs parameterized per project (keyword sets). |
| Umami | One **site** per project. Shared API key scoped by site. |
| Reddit/HN monitors | Per-project keyword sets and subreddit lists, managed by Werd API, stored in PostgreSQL. |

The Werd API Server is the source of truth for project configuration. It provisions and deprovisions sub-service resources via their APIs when projects are created/modified/deleted.

### Data Model & PostgreSQL

PostgreSQL is the right choice for Werd's core data store:

- **Relational model** maps naturally to the project → resources hierarchy (projects, users, memberships, configurations, alerts, posts)
- **JSONB columns** provide flexibility for service-specific configuration without schema changes (each sub-service has different config shapes)
- **Row-level security (RLS)** can enforce project isolation at the database level, preventing cross-project data leaks even in case of application bugs
- **LISTEN/NOTIFY** enables real-time event propagation to the API server (e.g., new alert → push to connected dashboard clients via WebSocket)

**Werd core database schema (conceptual):**

```
projects
  id              UUID PK
  name            TEXT
  slug            TEXT UNIQUE
  settings        JSONB          -- project-wide preferences
  created_at      TIMESTAMPTZ

users
  id              UUID PK
  email           TEXT UNIQUE
  password_hash   TEXT
  name            TEXT
  created_at      TIMESTAMPTZ

project_members
  project_id      UUID FK → projects
  user_id         UUID FK → users
  role            ENUM (owner, admin, member, viewer)
  UNIQUE(project_id, user_id)

service_instances                 -- maps projects to sub-service resources
  id              UUID PK
  project_id      UUID FK → projects
  service         ENUM (ntfy, changedetect, umami, ...)
  external_id     TEXT           -- topic prefix, site ID, etc. in the sub-service
  config          JSONB          -- service-specific configuration
  status          ENUM (provisioning, active, error, deprovisioning)

monitor_sources
  id              UUID PK
  project_id      UUID FK → projects
  type            ENUM (reddit, hn, web, rss, github)
  config          JSONB          -- subreddits, URLs, feed URLs, repo/org, etc.
  enabled         BOOLEAN

keywords
  id              UUID PK
  project_id      UUID FK → projects
  keyword         TEXT
  match_type      ENUM (exact, substring, regex)

platform_connections
  id              UUID PK
  project_id      UUID FK → projects
  platform        TEXT           -- linkedin, x, bluesky, reddit, mastodon, ...
  credentials     JSONB          -- encrypted at rest (pgcrypto or app-level)
  enabled         BOOLEAN

alerts
  id              UUID PK
  project_id      UUID FK → projects
  source_type     ENUM (reddit, hn, web, rss, github)
  source_id       TEXT           -- external ID (reddit post ID, HN item ID, etc.)
  title           TEXT
  content         TEXT
  url             TEXT
  matched_keywords TEXT[]
  severity        ENUM (low, medium, high, critical)
  status          ENUM (new, seen, triaged, dismissed, responded)
  created_at      TIMESTAMPTZ
  UNIQUE(project_id, source_type, source_id)  -- deduplication

notification_rules
  id              UUID PK
  project_id      UUID FK → projects
  source_type     ENUM (reddit, hn, web, rss, github, all)
  min_severity    ENUM (low, medium, high, critical)
  destination     ENUM (ntfy, email, webhook)
  config          JSONB          -- topic, email addr, URL, etc.

published_posts
  id              UUID PK
  project_id      UUID FK → projects
  content         TEXT
  platforms       TEXT[]
  scheduled_at    TIMESTAMPTZ
  published_at    TIMESTAMPTZ
  status          ENUM (draft, scheduled, publishing, published, failed)

audit_log
  id              BIGSERIAL PK
  project_id      UUID FK → projects
  user_id         UUID FK → users (nullable for system actions)
  action          TEXT
  details         JSONB
  created_at      TIMESTAMPTZ
```

**Database layout (single PostgreSQL instance):**

| Database | Owner | Purpose |
|---|---|---|
| `werd` | Werd API Server | Core project/user/alert/config data (schema above) |
| `umami` | Umami | Privacy-friendly web analytics |

### Backend Architecture (Go)

The Werd API Server is written in **Go**, chosen for this workload because:

- **Low resource footprint** — 15-30 MB RSS, 5-15 MB container image (static binary in distroless). Critical when sharing a single box with other services.
- **Native concurrency** — goroutines are the natural fit for the core loop: poll service APIs concurrently, ingest webhooks, run background sync/dedup jobs, serve HTTP — all in one process without a job framework.
- **Best PostgreSQL integration** — pgx v5 is the strongest PostgreSQL driver across ecosystems, with full LISTEN/NOTIFY, JSONB, arrays, COPY protocol support. Paired with sqlc for compile-time-checked SQL with generated Go types.
- **Single binary deployment** — no runtime, no `node_modules`, no V8 engine. One static binary, one minimal Dockerfile.
- **Direct platform API integration** — social media platform APIs are straightforward HTTP/OAuth2 calls, well-suited to Go's standard library and available client packages.

**Go stack:**

| Concern | Tool |
|---|---|
| HTTP routing | [chi](https://github.com/go-chi/chi) |
| PostgreSQL | [pgx v5](https://github.com/jackc/pgx) + [sqlc](https://github.com/sqlc-dev/sqlc) |
| Migrations | [goose](https://github.com/pressly/goose) |
| Background jobs | [river](https://github.com/riverqueue/river) for persistent job queue (post scheduling, sync) |
| Config | [envconfig](https://github.com/kelseyhightower/envconfig) |
| WebSocket | `nhooyr.io/websocket` (real-time alert push to dashboard) |
| OpenAPI spec generation | [swag](https://github.com/swaggo/swag) (consumed by frontend via openapi-typescript) |
| Container | Multi-stage: `golang:1.23` builder → `gcr.io/distroless/static` runtime |

**Key responsibilities:**
- Authentication (local accounts, session management, optional OIDC/SSO)
- Multi-project orchestration (provision/deprovision sub-service resources per project)
- Cross-posting to social platforms (direct API integration — X, LinkedIn, Bluesky, Reddit, Mastodon)
- Post scheduling via river persistent job queue
- OAuth token management for platform connections (encrypted storage)
- Webhook ingestion (receive alerts from monitoring bots, tag with project, deduplicate, persist)
- Notification routing (evaluate rules, fan out to ntfy topics / dashboard / webhooks)
- LLM-assisted response drafting (direct HTTP call to configurable LLM API)
- Background sync (poll services for state changes, sync to Werd DB)
- Real-time push (WebSocket to dashboard clients for live alert feed)

### Frontend Architecture (React + TypeScript)

The Werd Dashboard is a **React + TypeScript SPA**, built with Vite:

| Concern | Tool |
|---|---|
| Framework | [React 19](https://react.dev) + TypeScript |
| Build | [Vite](https://vitejs.dev) |
| Routing | [React Router](https://reactrouter.com) |
| Data fetching | [TanStack Query](https://tanstack.com/query) |
| State management | [Zustand](https://github.com/pmndrs/zustand) |
| Styling | [Tailwind CSS](https://tailwindcss.com) |
| Component library | [shadcn/ui](https://ui.shadcn.com) |
| API types | Auto-generated from Go backend's OpenAPI spec via [openapi-typescript](https://github.com/openapi-ts/openapi-typescript) |
| Real-time | WebSocket connection to Werd API for live alerts |
| Container | Multi-stage: `node:22-alpine` builder → `caddy:alpine` serving static files (or served by Werd API) |

**Dashboard views:**
- **Project switcher** — top-level navigation between isolated projects
- **Overview** — per-project service health, recent alerts, upcoming scheduled posts
- **Alert feed** — unified, filterable, searchable alert list with triage actions (dismiss, flag, respond)
- **Publishing** — create/schedule/manage posts across platforms, preview, platform-specific formatting
- **Monitoring config** — manage keywords, subreddits, watched URLs, RSS feeds per project
- **Platforms** — connect/disconnect social media accounts per project (direct OAuth flows)
- **Notifications** — configure routing rules, priority thresholds per project
- **Analytics** — embedded Umami stats, cross-platform engagement metrics
- **Settings** — user management, roles, project settings, service configuration

### Container Runtime

**Podman (primary):**
- All services defined in a single `docker-compose.yml` compatible with `podman-compose`
- Runs rootless by default — no daemon, no root privileges required
- Services can be grouped into Podman pods for shared networking
- For users who prefer `docker-compose`, the Podman socket provides full compatibility

**Docker (secondary):**
- The compose file is compatible with Docker / Docker Compose out of the box
- Docker support is a secondary target — tested but not the primary development environment

### Kubernetes Strategy

Kubernetes is a **good fit for scaling Werd** beyond a single box, but adds significant complexity (~750 MB RAM overhead for k3s, steep learning curve). The strategy is **compose-first, Kubernetes when you need to scale**.

**Why Kubernetes works for Werd:**
- Each component is a separate Deployment, scalable independently (e.g., scale Reddit monitors without scaling the API)
- Helm charts exist for all major infrastructure components (PostgreSQL via CloudNativePG/Bitnami, Redis via Bitnami)
- Namespace-based isolation can map to projects for additional separation (RBAC, NetworkPolicies, ResourceQuotas)
- Rolling updates, self-healing, and health check-driven restarts built in
- Horizontal Pod Autoscaler (HPA) for automatic scaling based on load

**k3s for local/single-node testing:**
- Lightweight single-binary Kubernetes distribution
- ~500-768 MB RAM baseline overhead (acceptable on 8+ GB machines)
- Use SQLite backend (default), disable unused components (Traefik if using Caddy, ServiceLB, metrics-server)
- `k3s server --disable traefik --disable servicelb`

**Migration path from compose to Kubernetes:**
1. Use [Kompose](https://kompose.io) to generate initial manifests from `docker-compose.yml` as scaffolding
2. Manually refine: add probes, resource limits, Secrets, Ingress rules
3. Migrate incrementally — stateless services first, databases last
4. Use operators for stateful services (CloudNativePG for PostgreSQL, Redis operator)
5. Package as Helm chart for reproducible deployments

**When to use what:**

| Scenario | Recommended Runtime |
|---|---|
| Single box, getting started | `podman-compose` / `docker-compose` |
| Single box, want self-healing | k3s (single node) |
| Multiple boxes, need scaling | k3s cluster or full Kubernetes |
| Production, high availability | Kubernetes with managed control plane or multi-master k3s |

### Networking & Access Modes

Werd is designed to run on any network — cloud VPS, residential, or local-only.

**Internal networking:**

All services communicate over an internal container network (`werd-net`). Only the reverse proxy exposes ports to the host:

| Service | Internal Port | External Access |
|---|---|---|
| Caddy | 80, 443 | Host-mapped (entry point for all HTTP traffic) |
| Werd Dashboard | 3000 | Via Caddy: `werd.yourdomain.com` |
| Werd API | 8090 | Via Caddy: `api.yourdomain.com` |
| changedetection.io | 5000 | Via Caddy: `monitor.yourdomain.com` |
| RSSHub | 1200 | Via Caddy: `rss.yourdomain.com` |
| ntfy | 2586 | Via Caddy: `ntfy.yourdomain.com` |
| Umami | 3000 | Via Caddy: `analytics.yourdomain.com` |
| PostgreSQL | 5432 | Internal only |
| Redis | 6379 | Internal only |

**Access modes:**

| Mode | Network Type | How It Works |
|---|---|---|
| **Cloud / VPS** | Public IP, open ports | Caddy binds to 80/443 directly, automatic Let's Encrypt TLS via HTTP-01 challenge. Standard DNS A record pointing to VPS IP. |
| **Residential** | Behind NAT / CGNAT | FRP client on home box tunnels to FRP server on a small relay VPS ($3-5/mo). Caddy on the relay VPS handles TLS and proxies traffic through the tunnel. DNS points to relay VPS IP. |
| **Local-only** | LAN access only | Caddy serves with self-signed certs or HTTP-only. Access via `http://local-ip:port`. No domain required. |

**Residential deployment (FRP tunnel):**

```
[Internet] → [Relay VPS: Caddy + FRP Server] → [FRP Tunnel] → [Home Box: Werd Stack]
```

- **[FRP](https://github.com/fatedier/frp)** (Apache-2.0, 105K+ stars) is the most mature, battle-tested open-source tunnel
- FRP client runs as a container in the Werd compose stack
- FRP server + Caddy run on a minimal relay VPS (512 MB RAM, $3-5/mo)
- Supports TCP, UDP, HTTP, HTTPS, and WebSocket
- Alternative: [Pangolin](https://github.com/fosrl/pangolin) (AGPL-3.0) provides an all-in-one tunnel + proxy + dashboard but uses Traefik internally instead of Caddy

**Kubernetes networking:**
- Ingress controller (Caddy Ingress or nginx-ingress) replaces standalone Caddy
- Use DNS-01 challenge for Let's Encrypt behind NAT (not HTTP-01, which requires inbound port 80)
- k3s: use `--disable traefik` and deploy your own Ingress controller for consistency with compose setup

### Storage & Volumes

Persistent data is stored in named volumes:

```
werd/
  data/
    postgres/        # All PostgreSQL databases (werd + umami)
    redis/           # Redis persistence (AOF)
    changedetect/    # Watch configurations, page snapshots
    ntfy/            # Notification cache
    caddy/           # TLS certificates, Caddy config
    werd/            # Werd API local state (uploaded assets, etc.)
```

For Kubernetes, each volume maps to a PersistentVolumeClaim (PVC). Use a StorageClass appropriate for your environment (local-path for single-node k3s, network-attached storage for multi-node).

### Configuration

All service configuration is driven through environment variables in `src/deploy/compose/.env` (copied from `.env.example`):

```bash
# ── Domain & Access Mode ──
WERD_DOMAIN=yourdomain.com
WERD_EMAIL=admin@yourdomain.com         # For Let's Encrypt
WERD_ACCESS_MODE=cloud                  # cloud | residential | local

# ── Residential tunnel (only if WERD_ACCESS_MODE=residential) ──
FRP_SERVER_ADDR=relay-vps.example.com
FRP_SERVER_PORT=7000
FRP_TOKEN=<generated>

# ── Database (werd superuser — owns werd DB, runs migrations) ──
POSTGRES_PASSWORD=<generated>

# ── Per-service database passwords ──
UMAMI_DB_PASSWORD=<generated>

# ── Redis ──
REDIS_PASSWORD=<generated>

# ── Service secrets ──
WERD_JWT_SECRET=<generated>

# ── Platform API keys (configure per project via Dashboard) ──
# Instance-wide OAuth app credentials (shared across projects):
REDDIT_CLIENT_ID=
REDDIT_CLIENT_SECRET=
TWITTER_CLIENT_ID=
TWITTER_CLIENT_SECRET=
LINKEDIN_CLIENT_ID=
LINKEDIN_CLIENT_SECRET=
# ... per-platform OAuth app registration

# ── Optional: LLM for AI-assisted drafting ──
LLM_API_URL=
LLM_API_KEY=
LLM_MODEL=

# ── Initial project setup (subsequent projects created via Dashboard) ──
WERD_ADMIN_EMAIL=admin@yourdomain.com
WERD_ADMIN_PASSWORD=<generated>
```

Per-project settings (keywords, subreddits, watched URLs, notification rules, platform connections) are managed through the **Werd Dashboard**, not environment variables. The Werd API Server stores these in PostgreSQL and provisions sub-service resources accordingly.

## Repository Layout

This is a monorepo. All application source code lives under `src/`, with language-specific code further nested (e.g., `src/go/` for all Go modules). Top-level directories serve distinct purposes:

```
werd/
├── src/                               # All application source and deployment code
│   ├── go/                            # Go workspace (all Go modules)
│   │   ├── go.work                    # Go workspace file linking all modules
│   │   ├── api/                       # Werd API Server
│   │   │   ├── cmd/werd-api/          #   Entry point (main.go)
│   │   │   ├── internal/              #   Internal packages
│   │   │   │   ├── config/            #     Environment/config loading
│   │   │   │   ├── handler/           #     HTTP route handlers
│   │   │   │   ├── middleware/        #     Auth, CORS, logging, project-scoping
│   │   │   │   ├── model/            #     Domain types
│   │   │   │   ├── router/            #     chi route definitions
│   │   │   │   ├── service/           #     Business logic
│   │   │   │   ├── storage/           #     sqlc-generated PostgreSQL queries
│   │   │   │   ├── webhook/           #     Webhook ingestion handlers
│   │   │   │   └── integration/       #     Social platform API clients (X, LinkedIn, etc.)
│   │   │   ├── migrations/            #   goose SQL migration files
│   │   │   ├── queries/               #   sqlc .sql query files
│   │   │   ├── sqlc.yaml              #   sqlc codegen config
│   │   │   ├── Makefile               #   build, test, lint, migrate, generate
│   │   │   └── Dockerfile             #   Multi-stage: golang → distroless
│   │   ├── monitor-reddit/            # Reddit monitoring bot
│   │   │   ├── cmd/monitor-reddit/    #   Entry point
│   │   │   ├── internal/              #   Reddit API, keyword matching, webhooks
│   │   │   ├── Makefile
│   │   │   └── Dockerfile
│   │   └── monitor-hn/                # Hacker News poller
│   │       ├── cmd/monitor-hn/        #   Entry point
│   │       ├── internal/              #   HN API, keyword matching, webhooks
│   │       ├── Makefile
│   │       └── Dockerfile
│   │
│   ├── web/                           # Werd Dashboard (React + TypeScript SPA)
│   │   ├── src/
│   │   │   ├── components/            #   React components (shadcn/ui based)
│   │   │   ├── pages/                 #   Route pages
│   │   │   ├── hooks/                 #   Custom React hooks
│   │   │   ├── lib/                   #   Utilities, API client helpers
│   │   │   ├── stores/                #   Zustand state stores
│   │   │   └── types/                 #   openapi-typescript generated API types
│   │   ├── vite.config.ts
│   │   ├── package.json
│   │   ├── Makefile
│   │   └── Dockerfile                 #   Multi-stage: node → caddy (static serve)
│   │
│   └── deploy/                        # All deployment / runtime configuration
│       ├── compose/                   #   Docker/Podman compose (primary deployment)
│       │   ├── docker-compose.yml     #     Full stack definition
│       │   ├── .env.example           #     Environment variable template
│       │   └── init-db.sh             #     PostgreSQL multi-database init script
│       ├── caddy/                     #   Reverse proxy configuration
│       │   ├── Caddyfile              #     Production (subdomain routing, auto-TLS)
│       │   └── Caddyfile.local        #     Local-only (no TLS)
│       ├── relay/                     #   Residential tunnel (relay VPS setup)
│       │   ├── docker-compose.yml     #     FRP server + Caddy on relay VPS
│       │   ├── frps.toml              #     FRP server config
│       │   └── Caddyfile              #     Relay Caddy config
│       └── k8s/                       #   Kubernetes manifests / Helm chart (future)
│           └── helm/werd/             #     Helm chart
│
├── ci/                                # CI/CD infrastructure
│   ├── Containerfile                  #   CI runner container definition
│   ├── runner/                        #   Runner entrypoint
│   └── scripts/                       #   Per-concern CI scripts (build, test, lint, detect-changes)
│
├── tools/                             # Repo-wide developer scripts
│   ├── generate-secrets.sh            #   Generate all .env secrets
│   ├── dev-setup.sh                   #   One-command dev environment bootstrap
│   └── ci/                            #   CI runner management
│
├── design/                            # Architecture & design documentation
│   ├── ARCHITECTURE.md                #   System architecture overview
│   ├── DESIGN_LOG.md                  #   Chronological decision log
│   ├── DATA_MODEL.md                  #   Database schema design
│   └── diagrams/                      #   Mermaid/PNG architecture diagrams
│
├── spec/                              # API specifications
│   └── openapi.yaml                   #   Werd API OpenAPI spec (source of truth for frontend types)
│
├── research/                          # Technical & market research
│   ├── competitors/                   #   Competitor analysis
│   └── tools/                         #   Tool evaluations
│
├── plan/                              # Planning & progress tracking
│   ├── PROGRESS.md                    #   Current status
│   ├── BLOCKERS.md                    #   Known blockers
│   └── phases/                        #   Per-phase detailed plans (01 through 10)
│
├── marketing/                         # Marketing materials
│
├── tests/                             # Integration & end-to-end tests
│   └── integration/                   #   Phase 1 infrastructure tests (Bash)
│       ├── run.sh                     #     Test harness (setup → run → teardown)
│       ├── lib.sh                     #     Shared assertions and compose helpers
│       ├── docker-compose.test.yml    #     Compose override for test environment
│       └── suites/                    #     Test suites (01-08)
│
├── Makefile                           # Top-level build orchestration
├── PLAN.md                            # High-level phase overview
├── README.md                          # This file
└── .github/workflows/ci.yml          # GitHub Actions CI
```

### Key conventions

- **`src/go/`** — All Go code lives here as a [Go workspace](https://go.dev/doc/tutorial/workspaces). Modules share dependencies via `go.work` but build independently (each has its own `Dockerfile`).
- **`src/web/`** — React SPA. API types are auto-generated from `spec/openapi.yaml` via openapi-typescript.
- **`src/deploy/`** — All runtime/deployment configuration. The `docker-compose.yml` references Dockerfiles in sibling directories via relative build contexts.
- **`tools/`** — Scripts developers run locally. `ci/scripts/` — scripts that run only in CI.
- **`design/`** — Architecture docs and decision log. Separate from `plan/` which tracks implementation progress.
- **Per-package Makefiles** — each package under `src/` has its own `Makefile`. The root `Makefile` delegates to them (e.g., `make build-api` → `make -C src/go/api build`).

## Deployment

### Prerequisites

- **Podman** 4.0+ with `podman-compose` 1.0+, **or** Docker with Docker Compose v2
- A Linux server (2 vCPU, 2-4 GB RAM, 30 GB SSD minimum — 8 GB RAM recommended for comfort)
- A domain name pointed at your server (for automatic TLS; not required for local-only mode)

### Quick Start (Podman — Single Box)

```bash
# Clone the repo
git clone https://github.com/your-org/werd.git
cd werd

# Copy and configure environment
cp src/deploy/compose/.env.example src/deploy/compose/.env
# Edit src/deploy/compose/.env with your domain, access mode, and credentials

# Generate all secrets automatically
./tools/generate-secrets.sh

# Start all services
make compose-up

# Check status
make compose-ps

# Open the dashboard
# Cloud:       https://werd.yourdomain.com
# Local-only:  http://localhost:3000
```

### Quick Start (Docker — Single Box)

```bash
git clone https://github.com/your-org/werd.git
cd werd
cp src/deploy/compose/.env.example src/deploy/compose/.env
./tools/generate-secrets.sh
docker compose -f src/deploy/compose/docker-compose.yml --env-file src/deploy/compose/.env up -d
```

### docker-compose with Podman Socket

```bash
# Enable the Podman socket
systemctl --user enable --now podman.socket

# Point docker-compose at Podman
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock

# Use make targets as usual (they use podman-compose internally)
make compose-up
```

### Kubernetes (k3s — Single Node)

```bash
# Install k3s without built-in Traefik (we use Caddy)
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable traefik --disable servicelb" sh -

# Deploy Werd via Helm
helm repo add werd https://your-org.github.io/werd-charts
helm install werd werd/werd \
  --namespace werd --create-namespace \
  --values values.yaml

# Or generate manifests from compose
kompose convert -f src/deploy/compose/docker-compose.yml -o src/deploy/k8s/
# Then manually refine and apply
kubectl apply -f src/deploy/k8s/
```

### Residential Deployment (Behind NAT)

**On the relay VPS** ($3-5/mo, 512 MB RAM):

```bash
# Install FRP server + Caddy
# See src/deploy/relay/ for ready-to-use configs
cd src/deploy/relay
docker compose up -d
```

**On your home box:**

```bash
# Set access mode to residential in src/deploy/compose/.env
WERD_ACCESS_MODE=residential
FRP_SERVER_ADDR=relay-vps.example.com
FRP_SERVER_PORT=7000
FRP_TOKEN=your-secret-token

# Start — FRP client container included automatically
podman-compose -f src/deploy/compose/docker-compose.yml --profile residential up -d
```

### Compose Structure (`src/deploy/compose/docker-compose.yml`)

```yaml
services:
  # ── Core (Werd) ──
  werd-api:          # Go API server (cross-posting, routing, scheduling built-in)
  werd-dashboard:    # React SPA (served by Caddy or werd-api)

  # ── Infrastructure ──
  caddy:             # Reverse proxy & auto-TLS
  postgres:          # Shared PostgreSQL (werd + umami DBs)
  redis:             # Shared Redis (Werd sessions, RSSHub cache)

  # ── Lightweight Services ──
  ntfy:              # Push notifications (~20 MB)
  changedetect:      # Web page monitoring (~100 MB)
  rsshub:            # RSS feed generation (~80 MB)
  umami:             # Web analytics (~240 MB)

  # ── Custom Monitors ──
  reddit-monitor:    # Custom Go Reddit monitor
  hn-monitor:        # Custom HN poller

  # ── Residential tunnel (profile: residential) ──
  frpc:              # FRP client — only started with --profile residential

networks:
  werd-net:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
  caddy-data:
  caddy-config:
  # ntfy-data:
  # changedetect-data:
```

## Integration Tests

End-to-end tests that spin up the full compose stack and validate that all Phase 1 infrastructure works correctly as a unit.

```bash
# Run the full suite (builds images, starts stack, runs tests, tears down):
make integration-test

# Same, but leave the stack running for debugging:
make integration-test-keep
```

**Requirements:** Podman with podman-compose (or Docker with docker compose), curl, openssl. No domain or TLS setup needed — tests use `Caddyfile.local` with port-based routing on localhost.

### What's tested (~44 assertions)

| Suite | Coverage |
|---|---|
| Stack Lifecycle | All 5 services healthy, startup ordering, `werd-net` network name deterministic, internal ports not exposed |
| PostgreSQL | Database/user creation by `init-db.sh`, cross-database isolation, `postgres.conf` tuning loaded |
| Redis | AUTH enforcement, per-DB read/write and isolation, `maxmemory`, eviction policy, AOF enabled |
| Werd API | `/healthz` through Caddy, JSON response body, direct port blocked |
| Werd Dashboard | HTML serving, SPA `try_files` routing (deep paths return `index.html`), direct port blocked |
| Caddy Proxy | Security headers, `Server` header removed, CORS origin/methods/headers, preflight, 10MB body limit |
| DNS Resolution | All services resolvable by name from multiple containers |
| Persistence | PostgreSQL data survives restart, Redis AOF data survives restart |

See [`tests/integration/README.md`](tests/integration/README.md) for architecture details, per-test documentation, and debugging tips.

## Implementation Milestones

| # | Milestone | Status | Details | Dependencies |
|---|---|---|---|---|
| **1** | **Core Infrastructure** | | **Foundation: compose, networking, databases, proxy** | |
| 1.1 | Project scaffolding | Done | Directory structure, `docker-compose.yml` skeleton, `.env.example`, Makefiles, CI scripts, Dockerfiles | — |
| 1.2 | PostgreSQL deployment | Done | Shared PostgreSQL 17 with per-service users/databases, tuned config, init-db.sh with grants | 1.1 |
| 1.3 | Redis deployment | Done | Shared Redis 7 with tuned config, per-service DB isolation, AOF persistence | 1.1 |
| 1.4 | Caddy reverse proxy | Done | Caddyfile with subdomain routing, auto-TLS, security headers, CORS, WebSocket proxy, local mode variant | 1.1 |
| 1.5 | Container networking | Not started | `werd-net` bridge network, DNS resolution between services, rootless Podman config | 1.1 |
| 1.6 | Health checks & restart policies | Not started | Liveness/readiness probes for all services, `restart: unless-stopped`, dependency ordering (`depends_on` with health conditions) | 1.2–1.5 |
| 1.7 | Secret generation script | Done | `tools/generate-secrets.sh` — generates all passwords, JWT secrets, writes to `.env` | 1.1 |
| | | | | |
| **2** | **Werd API Server (Go Backend)** | | **Core backend — auth, multi-project, cross-posting, routing** | |
| 2.1 | Go project scaffolding | Done | Go module, chi router, pgx connection pool, sqlc config, Dockerfile (multi-stage → distroless), compose service | 1.1 |
| 2.2 | Database migrations | Not started | goose migration files for Werd core schema (projects, users, project_members, service_instances, alerts, etc.) | 2.1, 1.2 |
| 2.3 | Authentication system | Not started | Local user accounts, bcrypt password hashing, JWT session tokens, middleware for route protection | 2.2 |
| 2.4 | Multi-project CRUD | Not started | Create/read/update/delete projects, member management, role-based access control (owner/admin/member/viewer) | 2.3 |
| 2.5 | Service provisioning engine | Not started | On project create: provision ntfy topic, changedetection watches, Umami site. On delete: deprovision. | 2.4, 3.x |
| 2.6 | Webhook ingestion | Not started | HTTP endpoints to receive alerts from monitors, tag with project, deduplicate (UNIQUE constraint on source_type + source_id), persist | 2.4 |
| 2.7 | Notification routing engine | Not started | Evaluate per-project notification rules against incoming alerts, fan out to ntfy topics / dashboard / webhooks / LLM drafting | 2.6 |
| 2.8 | Social platform integration | Not started | Per-platform posting adapters (X, LinkedIn, Bluesky, Reddit, Mastodon), OAuth token management, encrypted credential storage | 2.4 |
| 2.9 | Post scheduling | Not started | river persistent job queue for scheduled posts, status tracking, retry logic | 2.8 |
| 2.10 | Background sync jobs | Not started | Poll sub-services for state changes, sync engagement metrics from social platforms | 2.5 |
| 2.11 | WebSocket real-time push | Not started | WebSocket endpoint for live alert feed — push new alerts to connected dashboard clients via LISTEN/NOTIFY | 2.6 |
| 2.12 | OpenAPI spec generation | Not started | swag annotations on all endpoints, auto-generated OpenAPI spec for frontend type generation | 2.4–2.11 |
| | | | | |
| **3** | **Lightweight Service Deployment** | | **Get sub-services running and accessible** | |
| 3.1 | ntfy | Not started | Deploy with SQLite cache, configure ACL for per-project topics, test push notifications | 1.4 |
| 3.2 | changedetection.io | Not started | Deploy, configure Caddy route, verify webhook output on page change | 1.4 |
| 3.3 | RSSHub | Not started | Deploy with Redis cache, verify feed generation for target platforms | 1.3, 1.4 |
| 3.4 | Umami | Not started | Deploy with PostgreSQL backend, configure site creation API, verify tracking | 1.2, 1.4 |
| | | | | |
| **4** | **Werd Dashboard (React Frontend)** | | **SPA for project management and unified control** | |
| 4.1 | React project scaffolding | Done | Vite + React 19 + TypeScript, Tailwind CSS, shadcn/ui, React Router, TanStack Query, Dockerfile | 1.1 |
| 4.2 | API type generation pipeline | Not started | openapi-typescript consuming Werd API's OpenAPI spec, automated in build | 2.12, 4.1 |
| 4.3 | Auth flow | Not started | Login/logout, JWT token management, protected routes, user context | 2.3, 4.1 |
| 4.4 | Project switcher + overview | Not started | Project list, create/switch projects, per-project dashboard with service health and recent alerts | 2.4, 4.3 |
| 4.5 | Alert feed view | Not started | Unified alert list with filtering (source, severity, status, keyword), search, triage actions, WebSocket live updates | 2.6, 2.11, 4.4 |
| 4.6 | Monitoring configuration | Not started | Manage keywords, subreddits, watched URLs, RSS feeds per project — CRUD forms writing to Werd API | 2.4, 4.4 |
| 4.7 | Platform connections | Not started | Connect/disconnect social accounts per project via OAuth flows, status display | 2.8, 4.4 |
| 4.8 | Publishing interface | Not started | Create/schedule/manage posts across platforms, preview, platform-specific formatting | 2.8, 2.9, 4.7 |
| 4.9 | Notification rules | Not started | Configure routing rules per project: source type × severity → destination (ntfy topic / webhook) | 2.7, 4.4 |
| 4.10 | Analytics dashboard | Not started | Embedded Umami stats, cross-platform engagement metrics per project | 2.10, 3.4, 4.4 |
| 4.11 | Settings & user management | Not started | Invite users to projects, manage roles, project settings, service configuration | 2.4, 4.3 |
| | | | | |
| **5** | **Monitoring Pipeline** | | **Custom bots for keyword monitoring across sources** | |
| 5.1 | Reddit monitoring bot | Not started | Go service using Reddit API. Streams target subreddits per project, keyword matching, webhook to Werd API. | 2.6 |
| 5.2 | Hacker News poller | Not started | Go service polling HN API (stories, comments, Ask HN). Per-project keyword matching, webhook to Werd API. | 2.6 |
| 5.3 | changedetection.io integration | Not started | Werd API provisions/manages watches per project via changedetection.io API. Webhook alerts routed to Werd API. | 2.5, 3.2 |
| 5.4 | RSSHub feed configuration | Not started | Werd API generates RSSHub URLs per project keyword sets. Feeds consumed by changedetection.io. | 3.3 |
| 5.5 | GitHub webhook receiver | Not started | Werd API endpoint receives GitHub webhooks (stars, issues, PRs, discussions), routes per project by repo mapping. | 2.6 |
| | | | | |
| **6** | **Notification & Routing** | | **Wire monitoring outputs to notification destinations** | |
| 6.1 | ntfy alert rules | Not started | High-priority alerts trigger ntfy push to per-project topics. Configurable priority levels via notification_rules. | 2.7, 3.1 |
| 6.2 | LLM response drafting | Not started | On relevant mention: Werd API calls LLM API → generates draft → surfaces in dashboard for human review. | 2.7 |
| 6.3 | Alert deduplication | Not started | UNIQUE constraint on (project_id, source_type, source_id). Cross-monitor dedup for overlapping sources. | 2.6 |
| 6.4 | External webhook destinations | Not started | Configure arbitrary webhook URLs as notification destinations (Slack, Discord, etc. via incoming webhooks). | 2.7 |
| | | | | |
| **7** | **Publishing Pipeline** | | **Cross-posting, scheduling, and syndication** | |
| 7.1 | Platform OAuth connections | Not started | Dashboard OAuth flow per platform per project. Werd API manages token storage and refresh. | 2.8, 4.7 |
| 7.2 | Post creation & scheduling | Not started | Dashboard create/schedule → Werd API → platform APIs. Per-project scoping. river job queue for scheduling. | 2.9, 4.8, 7.1 |
| 7.3 | Blog syndication | Not started | cross-post CLI integration for Dev.to and Hashnode with canonical URL preservation | 7.1 |
| 7.4 | Umami tracking integration | Not started | Auto-generate UTM parameters for published links. Per-project Umami site for referral tracking. | 2.5, 3.4 |
| | | | | |
| **8** | **Network Access & Tunnel** | | **Residential and flexible network support** | |
| 8.1 | Cloud access mode | Not started | Default Caddyfile with direct Let's Encrypt HTTP-01 challenge. Standard DNS setup documentation. | 1.4 |
| 8.2 | Local-only access mode | Not started | Caddyfile variant with self-signed certs or HTTP-only. Compose profile `local`. | 1.4 |
| 8.3 | FRP tunnel client container | Not started | `frpc` container in compose (profile: `residential`). Config template reading from `.env`. | 1.1 |
| 8.4 | Relay VPS setup | Not started | `src/deploy/relay/` — compose file with FRP server + Caddy. Setup script and documentation. | 8.3 |
| 8.5 | DNS-01 TLS for NAT/k8s | Not started | Caddy DNS-01 challenge plugin (Cloudflare, Route53, etc.) for environments where HTTP-01 is not possible. | 1.4 |
| | | | | |
| **9** | **Kubernetes Deployment** | | **Helm charts and k8s manifests for distributed scaling** | |
| 9.1 | Kompose baseline generation | Not started | Generate initial k8s manifests from `docker-compose.yml` via Kompose | 1.x |
| 9.2 | Manifest refinement | Not started | Add resource limits, probes, Secrets, ConfigMaps, Ingress rules to generated manifests | 9.1 |
| 9.3 | Helm chart | Not started | Package all manifests into a Helm chart with `values.yaml` for configuration | 9.2 |
| 9.4 | Database operators | Not started | CloudNativePG for PostgreSQL, Redis operator — replacing raw StatefulSets | 9.2 |
| 9.5 | k3s single-node testing | Not started | Documented k3s deployment with optimized settings (`--disable traefik`, GOMEMLIMIT, SQLite backend) | 9.3 |
| 9.6 | Multi-node scaling guide | Not started | Documentation for scaling individual services (HPA config, node affinity, PVC across nodes) | 9.5 |
| | | | | |
| **10** | **Hardening & Documentation** | | **Production readiness, security, operations** | |
| 10.1 | Backup & restore | Not started | `scripts/backup.sh` — pg_dump all databases, snapshot volumes. `scripts/restore.sh` for recovery. Documented schedule. | 1.2 |
| 10.2 | Security hardening | Not started | Rate limiting (Caddy), CORS policies, CSP headers, non-root container users, encrypted credentials (pgcrypto or app-level AES), secret rotation script | 1.4, 2.3 |
| 10.3 | Logging & observability | Not started | Centralized log aggregation (Loki + Promtail or simple file-based). Service-level error alerting via ntfy. | 1.6, 3.1 |
| 10.4 | Setup documentation | Not started | Step-by-step deployment guide for each access mode. Configuration reference. Troubleshooting FAQ. | All above |
| 10.5 | Update/migration tooling | Not started | `scripts/update.sh` — pull new images, run goose migrations, restart services with rollback. Compose and k8s variants. | 10.1 |
| 10.6 | CI/CD pipeline | Not started | GitHub Actions: build Werd API + Dashboard images, run tests, push to container registry, Helm chart release | 2.12, 4.2 |

### Priority Order

**Critical path:** 1.x → 2.1–2.4 → 3.1–3.4 → 4.1–4.5 → 5.1–5.2 → 6.1–6.3

1. **Phase 1 (Core Infra)** — compose file, databases, proxy, networking
2. **Phase 2 (Werd API)** — Go backend with auth, multi-project, cross-posting, webhook ingestion, notification routing
3. **Phase 3 (Lightweight Services)** — get sub-services running and API-accessible (ntfy, changedetection, RSSHub, Umami)
4. **Phase 4 (Dashboard)** — React SPA providing the unified management UI (can be developed in parallel with Phase 3)
5. **Phase 5 + 6 (Monitoring + Routing)** — the core value proposition: monitor → alert → notify
6. **Phase 7 (Publishing)** — cross-posting and content distribution
7. **Phase 8 (Network Access)** — residential tunnel support, access mode variants
8. **Phase 9 (Kubernetes)** — Helm charts and distributed deployment (can begin after Phase 3)
9. **Phase 10 (Hardening)** — production polish, security, documentation, CI/CD

## Status

MVP complete. Phases 1 (Infrastructure), 3 (Service Deployment), 4 (Dashboard), 5 (Monitoring), and 6 (Notifications) are done. Phase 2 (API Server) is mostly done. The platform has ~35+ API endpoints, 10 dashboard pages, dual method support (API/browser) for all platforms, Reddit+HN monitors, a processing rules pipeline with LLM classification, and notification routing to ntfy/webhooks. Stack simplified (2026-03-13): heavy dependencies removed, core runs in ~440 MB RAM.

## License

Apache-2.0
