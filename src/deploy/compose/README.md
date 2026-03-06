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

Third-party services are commented out by default. Uncomment them in `docker-compose.yml` as you work through the implementation phases.

## PostgreSQL

Shared PostgreSQL 17 instance hosting 6 databases. Each service gets a dedicated database user restricted to its own database.

### Databases and Users

| Database | User | Purpose |
|---|---|---|
| `werd` | `werd` (superuser) | Core project/user/alert/config data. Also runs migrations. |
| `postiz` | `postiz` | Cross-posting organizations, integrations, scheduled posts |
| `activepieces` | `activepieces` | Workflow definitions, connections, executions |
| `mattermost` | `mattermost` | Teams, channels, messages, users |
| `plausible` | `plausible` | Analytics sites, goals, configuration |
| `temporal` | `temporal` | Workflow engine state (Postiz dependency) |

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
| `POSTIZ_DB_PASSWORD` | Password for `postiz` database user | `changeme` |
| `ACTIVEPIECES_DB_PASSWORD` | Password for `activepieces` database user | `changeme` |
| `MATTERMOST_DB_PASSWORD` | Password for `mattermost` database user | `changeme` |
| `PLAUSIBLE_DB_PASSWORD` | Password for `plausible` database user | `changeme` |
| `TEMPORAL_DB_PASSWORD` | Password for `temporal` database user | `changeme` |
| `POSTGRES_PORT` | Host port for dev access (commented out by default) | `5432` |

All passwords should be generated via `tools/generate-secrets.sh` — never use the defaults in production.
