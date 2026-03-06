# Werd Data Model

See [README.md](../README.md) for the full conceptual schema.

## Databases (single PostgreSQL instance)

| Database | Owner | Purpose |
|---|---|---|
| `werd` | Werd API Server | Core project/user/alert/config data |
| `postiz` | Postiz | Organizations, integrations, scheduled posts |
| `activepieces` | Activepieces | Flows, connections, executions |
| `mattermost` | Mattermost | Teams, channels, messages |
| `plausible` | Plausible | Sites, goals, configuration |
| `temporal` | Temporal | Workflow engine state (Postiz dependency) |

## Multi-Project Isolation

The `werd` database is the source of truth. Every table (except `users`) is scoped by `project_id`. The API server maps each project to isolated resources in sub-services:

- Postiz organization
- Mattermost team + channels
- ntfy topic prefix
- Plausible site
- changedetection.io watch tags

The `service_instances` table tracks these mappings.
