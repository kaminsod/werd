# Werd — Go Workspace

This directory contains all Go source code for the Werd platform, managed as a [Go workspace](https://go.dev/doc/tutorial/workspaces).

## Modules

| Module | Description |
|---|---|
| `api/` | Werd API Server — core backend handling auth, multi-project orchestration, webhook ingestion, and service aggregation |
| `monitor-reddit/` | Reddit monitoring bot — streams subreddits for keyword matches, sends webhooks to the API |
| `monitor-hn/` | Hacker News poller — polls HN API for keyword matches, sends webhooks to the API |

## Development

```bash
# From this directory:
go work sync

# Or use the root Makefile:
make build-api
make test-api
```
