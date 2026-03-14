# Compose Deployment

Primary single-box deployment via `docker-compose.yml`.

## Quick Start

```bash
cp .env.example .env
# Edit .env with your domain and access mode

# Generate secure secrets (all passwords, JWT keys, etc.):
../../../tools/generate-secrets.sh
# Or from repo root: make generate-secrets

# Start core services:
podman-compose up -d

# Or with docker-compose via Podman socket:
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
docker compose up -d
```

## Service Activation

Lightweight services are commented out by default. Uncomment them in `docker-compose.yml` as you work through the implementation phases.

## PostgreSQL

Shared PostgreSQL 17 instance hosting databases for Werd and sub-services. Each service gets a dedicated database user restricted to its own database.

### Databases and Users

| Database | User | Purpose |
|---|---|---|
| `werd` | `werd` (superuser) | Core project/user/alert/config data. Also runs migrations. |
| `umami` | `umami` | Privacy-friendly web analytics |

### Initialization

On first start, `init-db.sh` runs automatically (via `docker-entrypoint-initdb.d`) and:
1. Creates a dedicated PostgreSQL user for each service
2. Creates each service database with that user as owner
3. Revokes `PUBLIC` access so users can only reach their own database

The `werd` database and user are created by the postgres image itself (`POSTGRES_USER` / `POSTGRES_DB`).

**Important:** `init-db.sh` only runs on first container start (when the data volume is empty). If you need to re-initialize, remove the volume first:
```bash
podman-compose down
podman volume rm compose_postgres-data   # name may vary — check with: podman volume ls
podman-compose up -d
```

### Configuration Tuning

`postgres.conf` provides tuned settings for a multi-service single-box deployment. The defaults target a host with **4 GB RAM**. Adjust for your hardware:

| Host RAM | `shared_buffers` | `effective_cache_size` |
|---|---|---|
| 2 GB | 256MB | 1GB |
| 4 GB | 512MB (default) | 2GB (default) |
| 8 GB | 1GB | 4GB |
| 16 GB | 2GB | 8GB |

Edit `postgres.conf` directly — changes take effect on container restart.

### Host Access (Development)

To connect to PostgreSQL from your host (psql, pgAdmin, DBeaver, etc.), uncomment the port mapping in `docker-compose.yml`:

```yaml
postgres:
  ports:
    - "${POSTGRES_PORT:-5432}:5432"
```

Then connect:
```bash
psql -h localhost -U werd -d werd
# Password: value of POSTGRES_PASSWORD from .env
```

### Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `POSTGRES_PASSWORD` | Password for `werd` superuser | `changeme` |
| `UMAMI_DB_PASSWORD` | Password for `umami` database user | `changeme` |
| `POSTGRES_PORT` | Host port for dev access (commented out by default) | `5432` |

All passwords should be generated via `tools/generate-secrets.sh` — never use the defaults in production.

## Redis

Shared Redis 7 instance used for caching, sessions, and job queues. Services are isolated by database number to avoid key collisions.

### Database Numbering

| DB | Service | Usage |
|---|---|---|
| 0 | Werd API | Sessions, cache |
| 2 | RSSHub | Route cache |

All services share a single `REDIS_PASSWORD`. Redis ACLs don't support per-database access restrictions, so isolation is by convention (separate DB numbers). Each service's connection URL includes its assigned database number (e.g., `redis://:pass@redis:6379/2`).

### Configuration Tuning

`redis.conf` sets memory limits and eviction policy. The default `maxmemory` of **256 MB** targets a 4 GB host. Adjust for your hardware:

| Host RAM | `maxmemory` |
|---|---|
| 2 GB | 128mb |
| 4 GB | 256mb (default) |
| 8 GB | 512mb |
| 16 GB | 1gb |

The eviction policy is `allkeys-lru` — when memory is full, Redis evicts the least-recently-used keys. This is safe for cache/session workloads.

Edit `redis.conf` directly — changes take effect on container restart.

### Host Access (Development)

To connect from your host (redis-cli, RedisInsight, etc.), uncomment the port mapping in `docker-compose.yml`:

```yaml
redis:
  ports:
    - "${REDIS_PORT:-6379}:6379"
```

Then connect:
```bash
redis-cli -h localhost -a <REDIS_PASSWORD from .env>
# Or use RedisInsight at localhost:6379
```

### Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `REDIS_PASSWORD` | Shared password for all Redis connections | `changeme` |
| `REDIS_PORT` | Host port for dev access (commented out by default) | `6379` |

## Networking

All services share a single bridge network called `werd-net`.

### Network Topology

```
                          ┌─────────────┐
                          │   Caddy      │ ← only service exposing host ports (80, 443)
                          └──────┬───────┘
                                 │ werd-net (bridge)
        ┌──────────┬─────────────┼──────────────┬──────────┐
        │          │             │              │          │
   werd-api   werd-dashboard  postgres       redis    (other services)
```

- **Single flat network** — all services can reach each other by compose service name. Split frontend/backend networks are unnecessary for a single-user single-box deployment. PostgreSQL and Redis are already password-protected.
- **Deterministic name** — the compose file sets `name: werd-net` explicitly, so the network is always called `werd-net` regardless of project directory name (without this, Podman/Docker would prefix it, e.g., `compose_werd-net`).
- **Port exposure** — only Caddy binds to host ports (80/443). All other services communicate internally on `werd-net`. Uncomment port mappings in `docker-compose.yml` for direct host access during development.

### DNS Resolution

Services reference each other by their compose service name. For example, `werd-api` connects to PostgreSQL at `postgres:5432` and Redis at `redis:6379`. Docker/Podman compose provides this DNS resolution automatically on the bridge network.

To verify DNS resolution is working:
```bash
make compose-check-dns
```

### Rootless Podman Setup

Podman runs rootless by default, which means containers cannot bind to privileged ports (< 1024) without configuration. Since Caddy needs ports 80 and 443:

```bash
# Allow unprivileged binding to port 80+
sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80

# Persist across reboots
echo 'net.ipv4.ip_unprivileged_port_start=80' | sudo tee /etc/sysctl.d/podman-privileged-ports.conf
```

Run `tools/check-podman.sh` to verify your setup, or `tools/dev-setup.sh` which includes the check automatically.

### Docker Compatibility

If using `docker compose` with Podman as the backend, enable the Podman socket:

```bash
systemctl --user enable --now podman.socket
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
docker compose up -d
```

### Troubleshooting

| Problem | Diagnosis | Fix |
|---|---|---|
| Services can't reach each other | `make compose-check-dns` | Ensure all services are on `werd-net` |
| Caddy can't bind port 80/443 | `sysctl net.ipv4.ip_unprivileged_port_start` | See "Rootless Podman Setup" above |
| Network name has project prefix | `podman network ls` | Ensure `name: werd-net` is in compose file |
| SELinux denies volume mounts (Fedora/RHEL) | `ausearch -m avc -ts recent` | Add `:Z` suffix to volume mounts |
| `docker compose` can't connect to Podman | Check `$DOCKER_HOST` | Enable Podman socket (see above) |
