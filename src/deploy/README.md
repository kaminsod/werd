# Werd — Deployment

All runtime deployment configuration for the Werd platform.

## Directories

| Directory | Purpose |
|---|---|
| `compose/` | Docker/Podman compose — primary single-box deployment |
| `caddy/` | Caddy reverse proxy configuration (Caddyfile templates) |
| `relay/` | Residential relay VPS setup (FRP server + Caddy) |
| `k8s/` | Kubernetes manifests and Helm chart (future) |

## Quick Start

```bash
# From repo root:
cd src/deploy/compose
cp .env.example .env
# Edit .env with your configuration
../../tools/generate-secrets.sh  # or: make generate-secrets (from repo root)
podman-compose up -d
```

See the [main README](../../README.md) for full deployment instructions.
