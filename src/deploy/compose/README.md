# Compose Deployment

Primary single-box deployment via `docker-compose.yml`.

## Usage

```bash
cp .env.example .env
# Edit .env with your configuration

# Generate secure secrets:
../../../tools/generate-secrets.sh

# Start core services:
podman-compose up -d

# Or with docker-compose via Podman socket:
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
docker compose up -d
```

## Service Activation

Third-party services are commented out by default. Uncomment them in `docker-compose.yml` as you work through the implementation phases.

## Database Initialization

`init-db.sh` runs on first PostgreSQL start and creates separate databases for each service (postiz, activepieces, mattermost, plausible, temporal).
