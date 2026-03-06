# Hacker News Monitor

Polls the Hacker News API for new stories and comments matching per-project keyword lists. Sends alerts to the Werd API via webhook.

## Configuration

Environment variables:

- `WERD_API_URL` — Werd API server URL for webhook delivery
- `WERD_API_KEY` — Authentication key
- `HN_POLL_INTERVAL` — Polling interval (default: `60s`)

Keywords are fetched from the Werd API at startup and refreshed periodically.
