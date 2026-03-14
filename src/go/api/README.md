# Werd API Server

Go backend for the Werd platform. Handles authentication, multi-project orchestration, cross-posting, post scheduling, webhook ingestion, and notification routing.

## Stack

- **Router:** net/http (will migrate to chi)
- **Database:** pgx v5 + sqlc (planned)
- **Migrations:** goose (planned)
- **Background jobs:** river (persistent job queue for scheduling)
- **Config:** os.Getenv (will migrate to envconfig)

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
  integration/      Social platform API clients (X, LinkedIn, Bluesky, Reddit, Mastodon)
migrations/         goose SQL migration files
queries/            sqlc .sql query files
```
