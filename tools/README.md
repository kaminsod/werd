# Tools

Repo-wide developer scripts.

| Script | Purpose |
|---|---|
| `runner.sh` | Start/stop/restart the full Werd stack locally |
| `generate-secrets.sh` | Generate secure random secrets for `.env` |
| `dev-setup.sh` | Bootstrap local dev environment (install deps, sync workspace) |
| `check-podman.sh` | Check container runtime prerequisites |
| `ci/runner.sh` | CI runner management |

## runner.sh — Local Stack Runner

Manages the full Werd stack (9 services) in containers using docker compose or podman-compose.

### Quick Start

```bash
# Start everything (builds images, generates secrets, starts services):
./tools/runner.sh start

# Check health:
./tools/runner.sh status

# Stop:
./tools/runner.sh stop

# Restart (stop + start):
./tools/runner.sh restart
```

### What It Does

On `start`:
1. Creates `.env` from `.env.example` if missing (sets `WERD_ACCESS_MODE=local`)
2. Generates secure random secrets for any `changeme` values
3. Builds container images (werd-api from Go, werd-dashboard from React)
4. Starts all 9 services with health checks and dependency ordering
5. Waits for the stack to become healthy (up to 120s)
6. Prints access URLs with LAN IP

### Services and Ports (Local Mode)

| Port | Service | URL |
|------|---------|-----|
| 3080 | Dashboard | `http://<LAN-IP>:3080` |
| 3081 | API | `http://<LAN-IP>:3081` |
| 3082 | changedetection.io | `http://<LAN-IP>:3082` |
| 3083 | RSSHub | `http://<LAN-IP>:3083` |
| 3084 | ntfy | `http://<LAN-IP>:3084` |
| 3085 | Umami | `http://<LAN-IP>:3085` |

All services are accessible from any device on the LAN.

### Options

```bash
# Use podman-compose instead of docker compose:
./tools/runner.sh start --podman-compose

# Force docker compose (plugin v2):
./tools/runner.sh start --docker-compose

# Production mode (subdomain routing + auto-TLS, requires domain):
./tools/runner.sh start --production
```

### Compose Tool Detection

By default, the runner auto-detects the compose tool:
1. `docker compose` (v2 plugin) — preferred
2. `podman-compose` — fallback

Use `--podman-compose` or `--docker-compose` to override.

### Local vs Production Mode

**Local mode** (default): Uses `Caddyfile.local` with port-based routing on high ports (3080-3085). No TLS, no domain required. Accessible from LAN.

**Production mode** (`--production`): Uses the production `Caddyfile` with subdomain routing and automatic TLS via Let's Encrypt. Requires `WERD_DOMAIN` set in `.env` and DNS pointing to the server.

### Requirements

- **docker compose** (v2 plugin) or **podman-compose**
- **openssl** (for secret generation)
- **curl** (for health checks)
- ~2-4 GB RAM (all 9 services)

### Troubleshooting

| Problem | Fix |
|---|---|
| "No compose tool found" | Install docker compose or podman-compose |
| Build fails | Check `docker compose build` output, ensure Dockerfiles are valid |
| Services won't start | Run `./tools/runner.sh status`, check `docker compose logs` |
| Port already in use | Stop conflicting services, or check for a previous runner instance |
| Caddy can't bind port 80 (production) | `sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80` |
| Health check timeout | Services may still be building/starting; run `status` again |
