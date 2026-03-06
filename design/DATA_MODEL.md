# Werd Data Model

See [README.md](../README.md) for the full conceptual schema.

## Databases (single PostgreSQL instance)

| Database | Owner | Purpose |
|---|---|---|
| `werd` | Werd API Server | Core project/user/alert/config data |
| `postiz` | Postiz | Organizations, integrations, scheduled posts |
| `activepieces` | Activepieces | Flows, connections, executions |
| `mattermost` | Mattermost | Teams, channels, messages, users |
| `plausible` | Plausible | Sites, goals, configuration |
| `temporal` | Temporal | Workflow engine state (Postiz dependency) |

## Multi-Project Isolation

The `werd` database is the source of truth. Every table (except `users`) is scoped by `project_id`. The API server maps each project to isolated resources in sub-services:

| Sub-Service | Isolation Mechanism | Tracked In |
|---|---|---|
| Postiz | Organization per project | `service_instances` |
| Mattermost | Team + channels per project | `service_instances` |
| ntfy | Topic prefix per project | `service_instances` |
| Plausible | Site per project | `service_instances` |
| changedetection.io | Watch tags per project | `service_instances` |
| Activepieces | Naming convention (CE) | `service_instances` |

## Core Tables (werd database)

Key tables in the `werd` database:

- **projects** — Multi-tenant project definitions
- **users** — Local user accounts (bcrypt passwords, JWT sessions)
- **project_members** — User-to-project membership with roles (owner/admin/member/viewer)
- **service_instances** — Maps project_id to external service resource IDs
- **monitor_sources** — Per-project monitoring configuration (subreddits, URLs, keywords)
- **keywords** — Per-project keyword sets for matching
- **alerts** — Incoming alerts from all monitoring sources, tagged by project
- **notification_rules** — Per-project routing rules (source type x severity -> destination)
- **published_posts** — Cross-platform post tracking (wraps Postiz data)

All tables except `users` include a `project_id` foreign key for tenant isolation.
