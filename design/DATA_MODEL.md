# Werd Data Model

See [README.md](../README.md) for the full conceptual schema.

## Databases (single PostgreSQL instance)

| Database | Owner | Purpose |
|---|---|---|
| `werd` | Werd API Server | Core project/user/alert/config/post data |
| `umami` | Umami | Privacy-friendly web analytics |

## Multi-Project Isolation

The `werd` database is the source of truth. Every table (except `users`) is scoped by `project_id`. The API server maps each project to isolated resources in sub-services:

| Sub-Service | Isolation Mechanism | Tracked In |
|---|---|---|
| ntfy | Topic prefix per project | `service_instances` |
| changedetection.io | Watch tags per project | `service_instances` |
| Umami | Site per project | `service_instances` |
| RSSHub | Feed URLs parameterized per project | `monitor_sources` |

## Core Tables (werd database)

Key tables in the `werd` database:

- **projects** — Multi-tenant project definitions
- **users** — Local user accounts (bcrypt passwords, JWT sessions)
- **project_members** — User-to-project membership with roles (owner/admin/member/viewer)
- **service_instances** — Maps project_id to external service resource IDs
- **monitor_sources** — Per-project monitoring configuration (subreddits, URLs, keywords)
- **keywords** — Per-project keyword sets for matching
- **alerts** — Incoming alerts from all monitoring sources, tagged by project (includes `tags`, `classification_reason`, `monitor_source_id` columns added for processing rules pipeline)
- **notification_rules** — Per-project routing rules (source type x severity -> destination)
- **processing_rules** — Per-project filter and classify rules for the monitoring pipeline (keyword, regex, LLM)
- **platform_connections** — Per-project OAuth credentials for social platforms (stored as JSONB; encryption TODO)
- **published_posts** — Cross-platform post tracking with scheduling support
- **post_platform_results** — Per-platform publish results for each post (success/failure, external IDs)

All tables except `users` include a `project_id` foreign key for tenant isolation.
