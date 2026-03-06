# Werd API Server

Go backend for the Werd platform. Handles authentication, multi-project orchestration, webhook ingestion, notification routing, and service aggregation.

## Stack

- **Router:** chi
- **Database:** pgx v5 + sqlc
- **Migrations:** goose
- **Config:** envconfig

## Development

```bash
make dev          # Run with go run
make build        # Build binary
make test         # Run tests
make generate     # Regenerate sqlc code
make migrate-up   # Apply migrations
```

## Structure

```
cmd/werd-api/       Entry point
internal/
  config/           Environment/config loading
  handler/          HTTP route handlers
  middleware/       Auth, CORS, logging, project-scoping
  model/            Domain types
  router/           chi route definitions
  service/          Business logic
  storage/          sqlc-generated PostgreSQL queries
  webhook/          Webhook ingestion handlers
  integration/      API clients (Mattermost, Postiz, Activepieces, etc.)
migrations/         goose SQL migration files
queries/            sqlc .sql query files
```
