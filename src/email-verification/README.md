# Email Verification Server

Standalone [Mailpit](https://mailpit.axllent.org/) instance for receiving platform verification emails (Reddit, Bluesky account signups). Runs on a dedicated VPS where DNS MX records point.

## Architecture

```
Internet → MX datazo.net → mail.datazo.net (165.232.136.251)
                                ↓
                          Mailpit :25 (SMTP)
                                ↓
Browser Service polls → Mailpit :8025 (REST API) → extracts verification links
```

## DNS Prerequisites

| Record | Name | Value | Priority |
|--------|------|-------|----------|
| MX | `datazo.net` | `mail.datazo.net` | 0 |
| A | `mail.datazo.net` | `165.232.136.251` | — |

## Quick Start (Local)

```bash
cp .env.example .env
make run           # Start Mailpit on localhost:25 + localhost:8025
make test          # Send test email and verify via API
make logs          # Tail logs
make stop          # Stop
```

Web UI: http://localhost:8025

## Production Deployment

### First-Time Setup

```bash
cp .env.example .env
# Edit .env: set DEPLOY_HOST, optionally set MP_UI_AUTH for basic auth
make setup-server  # Installs podman, UFW, deploys Mailpit
```

This will:
1. Install `podman` + `podman-compose` on the VPS
2. Configure UFW (allow ports 22, 25, 8025 only)
3. Copy compose + .env to `/opt/email-verification/`
4. Start Mailpit and verify health

### Incremental Deploy

```bash
# After editing docker-compose.yml or .env:
make deploy        # Push config + restart
```

### Test Production

```bash
./scripts/test-email.sh 165.232.136.251
# Or from any machine:
# echo "test" | mail -s "test" anything@datazo.net
# curl http://165.232.136.251:8025/api/v1/messages
```

## Connecting to Werd

In `src/deploy/compose/.env`, set:

```env
MAILPIT_URL=http://165.232.136.251:8025
EMAIL_DOMAIN=datazo.net
```

The browser service will poll this remote Mailpit instance for verification emails instead of the local compose-internal one.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_PORT` | `25` | Host port for inbound SMTP |
| `API_PORT` | `8025` | Host port for Web UI + REST API |
| `MP_MAX_MESSAGES` | `5000` | Max stored messages before pruning |
| `MP_UI_AUTH` | _(empty)_ | Basic auth for Web UI (`user:pass`) |
| `DEPLOY_HOST` | — | VPS IP for deployment |
| `DEPLOY_USER` | `root` | SSH user |
| `DEPLOY_DIR` | `/opt/email-verification` | Remote install path |

## Useful API Endpoints

```bash
# Server info
curl http://HOST:8025/api/v1/info

# List messages
curl http://HOST:8025/api/v1/messages

# Search by recipient
curl "http://HOST:8025/api/v1/search?query=to:user@datazo.net"

# Delete all messages
curl -X DELETE http://HOST:8025/api/v1/messages
```
