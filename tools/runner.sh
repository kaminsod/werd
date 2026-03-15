#!/bin/bash
set -euo pipefail

# ============================================================================
# Werd Local Runner
# ============================================================================
#
# Manages the full Werd stack locally using container compose.
#
# Usage:
#   ./tools/runner.sh start                       — build and start all services
#   ./tools/runner.sh stop                        — stop all services
#   ./tools/runner.sh restart                     — stop then start
#   ./tools/runner.sh status                      — show service health and ports
#
#   ./tools/runner.sh containers ls               — list all containers
#   ./tools/runner.sh containers status            — health status of each container
#   ./tools/runner.sh containers --name=werd-api stop    — stop a specific container
#   ./tools/runner.sh containers --name=werd-api start   — start a specific container
#   ./tools/runner.sh containers --name=werd-api restart — restart a specific container
#   ./tools/runner.sh containers --name=werd-api status  — status of a specific container
#   ./tools/runner.sh containers --name=werd-api logs    — tail logs of a specific container
#
# Options:
#   --podman-compose   Use podman-compose instead of docker compose
#   --docker-compose   Force docker compose (plugin v2)
#   --production       Use production Caddyfile (subdomain + TLS) instead of local
#
# Requires: docker compose (or podman-compose), openssl
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$REPO_ROOT/src/deploy/compose"
ENV_FILE="$COMPOSE_DIR/.env"
ENV_EXAMPLE="$COMPOSE_DIR/.env.example"
CADDY_LOCAL="$REPO_ROOT/src/deploy/caddy/Caddyfile.local"

# ── Defaults ──

FORCE_TOOL=""
MODE="local"  # local or production

# ── Parse arguments ──

SUBCOMMAND=""
CONTAINER_NAME=""
CONTAINER_ACTION=""
ARGS=()

for arg in "$@"; do
  case "$arg" in
    --podman-compose) FORCE_TOOL="podman-compose" ;;
    --docker-compose) FORCE_TOOL="docker-compose" ;;
    --production)     MODE="production" ;;
    --name=*)         CONTAINER_NAME="${arg#--name=}" ;;
    *)                ARGS+=("$arg") ;;
  esac
done

# Determine subcommand from positional args.
if [ ${#ARGS[@]} -ge 1 ]; then
  SUBCOMMAND="${ARGS[0]}"
fi
if [ ${#ARGS[@]} -ge 2 ]; then
  CONTAINER_ACTION="${ARGS[1]}"
fi

if [ -z "$SUBCOMMAND" ]; then
  echo "Usage:"
  echo "  $0 <start|stop|restart|status>"
  echo "  $0 containers <ls|status>"
  echo "  $0 containers --name=<service> <start|stop|restart|status|logs>"
  exit 1
fi

# ── Detect compose tool ──

detect_compose() {
  if [ -n "$FORCE_TOOL" ]; then
    case "$FORCE_TOOL" in
      podman-compose)
        if command -v podman-compose >/dev/null 2>&1; then
          COMPOSE_CMD="podman-compose"
          return
        fi
        echo "Error: podman-compose not found"
        exit 1
        ;;
      docker-compose)
        if docker compose version >/dev/null 2>&1; then
          COMPOSE_CMD="docker compose"
          return
        fi
        echo "Error: docker compose not found"
        exit 1
        ;;
    esac
  fi

  # Auto-detect: prefer docker compose (v2 plugin), then podman-compose
  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v podman-compose >/dev/null 2>&1; then
    COMPOSE_CMD="podman-compose"
  else
    echo "Error: No compose tool found."
    echo "  Install docker compose: https://docs.docker.com/compose/install/"
    echo "  Or podman-compose:      pip install podman-compose"
    exit 1
  fi
}

detect_compose

# ── Compose wrapper ──

compose_cmd() {
  if [ "$MODE" = "local" ]; then
    export CADDYFILE_PATH="$CADDY_LOCAL"
    $COMPOSE_CMD \
      -f "$COMPOSE_DIR/docker-compose.yml" \
      -f "$COMPOSE_DIR/docker-compose.local.yml" \
      --env-file "$ENV_FILE" \
      "$@"
  else
    $COMPOSE_CMD \
      -f "$COMPOSE_DIR/docker-compose.yml" \
      --env-file "$ENV_FILE" \
      "$@"
  fi
}

# ── Ensure .env exists with secrets ──

ensure_env() {
  if [ ! -f "$ENV_FILE" ]; then
    echo "Creating .env from template..."
    cp "$ENV_EXAMPLE" "$ENV_FILE"

    # Set local mode defaults.
    sed -i 's/^WERD_DOMAIN=.*/WERD_DOMAIN=localhost/' "$ENV_FILE"
    sed -i 's/^WERD_ACCESS_MODE=.*/WERD_ACCESS_MODE=local/' "$ENV_FILE"
  fi

  # Generate secrets for any "changeme" values.
  if grep -q '=changeme' "$ENV_FILE" 2>/dev/null; then
    echo "Generating secrets..."
    "$REPO_ROOT/tools/generate-secrets.sh" "$ENV_FILE"
  fi
}

# ── Wait for healthy ──

wait_for_healthy() {
  local url="$1"
  local timeout="${2:-120}"
  local elapsed=0

  echo "Waiting for services to become healthy (timeout: ${timeout}s)..."

  while [ $elapsed -lt "$timeout" ]; do
    if curl -sf -o /dev/null "$url" 2>/dev/null; then
      return 0
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done

  echo "WARNING: Timed out waiting for $url after ${timeout}s."
  echo "Services may still be starting. Check: $0 status"
  return 1
}

# ── Get LAN IP ──

get_lan_ip() {
  ip -4 route get 1.0.0.0 2>/dev/null | awk '{print $7; exit}' \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo "localhost"
}

# ── Print access info ──

print_access_info() {
  local lan_ip
  lan_ip=$(get_lan_ip)

  echo ""
  echo "============================================"
  echo "  Werd is running"
  echo "============================================"

  if [ "$MODE" = "local" ]; then
    echo ""
    echo "  Dashboard:      http://${lan_ip} (or :3080)"
    echo "  API:            http://${lan_ip}:3081"
    echo "  changedetect:   http://${lan_ip}:3082"
    echo "  RSSHub:         http://${lan_ip}:3083"
    echo "  ntfy:           http://${lan_ip}:3084"
    echo "  Umami:          http://${lan_ip}:3085"
    echo ""
    echo "  Custom domain:  Point any domain at ${lan_ip} to access the dashboard."
  else
    local domain
    domain=$(grep '^WERD_DOMAIN=' "$ENV_FILE" 2>/dev/null | cut -d= -f2)
    echo ""
    echo "  Dashboard:      https://werd.${domain}"
    echo "  API:            https://api.${domain}"
    echo "  changedetect:   https://monitor.${domain}"
    echo "  RSSHub:         https://rss.${domain}"
    echo "  ntfy:           https://ntfy.${domain}"
    echo "  Umami:          https://analytics.${domain}"
  fi

  echo ""
  echo "  Manage:"
  echo "    $0 status                          — service health"
  echo "    $0 stop                            — stop all services"
  echo "    $0 restart                         — restart all services"
  echo "    $0 containers ls                   — list containers"
  echo "    $0 containers --name=<svc> logs    — tail service logs"
  echo "============================================"
}

# ── Stack subcommands ──

cmd_start() {
  echo "=== Werd Runner: start ==="
  echo "Compose: $COMPOSE_CMD"
  echo "Mode:    $MODE"
  echo ""

  ensure_env

  echo "Building images..."
  compose_cmd build 2>&1 | tail -10

  # Start infrastructure first (postgres, redis).
  echo ""
  echo "Starting infrastructure..."
  compose_cmd up -d postgres redis

  echo "Waiting for PostgreSQL..."
  local pg_elapsed=0
  while [ $pg_elapsed -lt 60 ]; do
    if compose_cmd exec -T postgres pg_isready -U werd >/dev/null 2>&1; then
      break
    fi
    sleep 2
    pg_elapsed=$((pg_elapsed + 2))
  done

  # Apply all database migrations (in order).
  echo "Applying database migrations..."
  for migration_file in "$REPO_ROOT"/src/go/api/migrations/*.sql; do
    [ -f "$migration_file" ] || continue
    sed -n '/^-- +goose Up$/,/^-- +goose Down$/p' "$migration_file" \
      | sed '/^-- +goose/d' \
      | compose_cmd exec -T postgres psql -U werd -d werd -f - 2>&1 \
      | sed '/already exists/d; /^$/d' || true
  done
  echo "Migrations applied."

  # Now start all remaining services.
  echo ""
  echo "Starting all services..."
  compose_cmd up -d

  if [ "$MODE" = "local" ]; then
    wait_for_healthy "http://localhost:3081/api/healthz" 120 || true
  else
    wait_for_healthy "http://localhost:80" 120 || true
  fi

  print_access_info
}

cmd_stop() {
  echo "=== Werd Runner: stop ==="
  compose_cmd down
  echo "All services stopped."
}

cmd_restart() {
  echo "=== Werd Runner: restart ==="
  compose_cmd down
  cmd_start
}

cmd_status() {
  echo "=== Werd Runner: status ==="
  echo "Compose: $COMPOSE_CMD"
  echo "Mode:    $MODE"
  echo ""

  compose_cmd ps

  # Quick health checks.
  echo ""
  echo "Health checks:"

  if [ "$MODE" = "local" ]; then
    local checks=("3080/:Dashboard" "3081/api/healthz:API" "3082/:changedetect" "3083/:RSSHub" "3084/v1/health:ntfy" "3085/:Umami")
    for entry in "${checks[@]}"; do
      local path_port="${entry%%:*}"
      local name="${entry##*:}"
      local port="${path_port%%/*}"
      local path="/${path_port#*/}"
      if curl -sf -o /dev/null "http://localhost:${port}${path}" 2>/dev/null; then
        printf "  \033[32m%-16s\033[0m OK (:%s)\n" "$name" "$port"
      else
        printf "  \033[31m%-16s\033[0m UNREACHABLE (:%s)\n" "$name" "$port"
      fi
    done
  else
    if curl -sf -o /dev/null "http://localhost:80" 2>/dev/null; then
      printf "  \033[32mCaddy\033[0m            OK (:80)\n"
    else
      printf "  \033[31mCaddy\033[0m            UNREACHABLE (:80)\n"
    fi
  fi
}

# ── Container subcommands ──

# List all known compose service names for validation.
get_known_services() {
  compose_cmd config --services 2>/dev/null || compose_cmd ps --services 2>/dev/null || echo ""
}

# Validate that a service name exists in the compose config.
validate_service_name() {
  local name="$1"
  local known
  known=$(get_known_services)
  if [ -z "$known" ]; then
    # Can't determine services — let compose handle the error.
    return 0
  fi
  if echo "$known" | grep -qx "$name"; then
    return 0
  fi
  echo "Error: Unknown service '$name'."
  echo ""
  echo "Available services:"
  echo "$known" | sed 's/^/  /'
  exit 1
}

# Check if a service is currently running.
is_service_running() {
  local name="$1"
  local status
  status=$(compose_cmd ps "$name" --format "{{.Status}}" 2>/dev/null || echo "")
  if [ -n "$status" ] && echo "$status" | grep -qi "up\|running"; then
    return 0
  fi
  return 1
}

# Require --name for actions that need it.
require_name() {
  local action="$1"
  if [ -z "$CONTAINER_NAME" ]; then
    echo "Error: --name=<service> is required for 'containers $action'."
    echo ""
    echo "Example: $0 containers --name=werd-api $action"
    echo ""
    echo "Available services:"
    get_known_services | sed 's/^/  /'
    exit 1
  fi
}

cmd_containers() {
  local action="${CONTAINER_ACTION:-ls}"

  case "$action" in
    ls)
      echo "=== Containers ==="
      compose_cmd ps --format "table {{.Name}}\t{{.Service}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null \
        || compose_cmd ps
      ;;

    status)
      if [ -n "$CONTAINER_NAME" ]; then
        validate_service_name "$CONTAINER_NAME"
        echo "=== Container: $CONTAINER_NAME ==="
        if is_service_running "$CONTAINER_NAME"; then
          compose_cmd ps "$CONTAINER_NAME"
        else
          echo "Service '$CONTAINER_NAME' is not running."
        fi
      else
        # Status for all containers with health detail.
        echo "=== All Containers ==="
        compose_cmd ps
        echo ""
        echo "Service health:"
        local services
        services=$(compose_cmd ps --services 2>/dev/null) || services=""
        if [ -z "$services" ]; then
          echo "  No services running."
        else
          for svc in $services; do
            local state
            state=$(compose_cmd ps "$svc" --format "{{.Status}}" 2>/dev/null || echo "unknown")
            if echo "$state" | grep -qi "healthy"; then
              printf "  \033[32m%-20s\033[0m %s\n" "$svc" "$state"
            elif echo "$state" | grep -qi "starting\|running\|up"; then
              printf "  \033[33m%-20s\033[0m %s\n" "$svc" "$state"
            else
              printf "  \033[31m%-20s\033[0m %s\n" "$svc" "$state"
            fi
          done
        fi
      fi
      ;;

    start)
      require_name "start"
      validate_service_name "$CONTAINER_NAME"
      if is_service_running "$CONTAINER_NAME"; then
        echo "$CONTAINER_NAME is already running."
        exit 0
      fi
      echo "Starting $CONTAINER_NAME..."
      if compose_cmd start "$CONTAINER_NAME" 2>&1; then
        echo "$CONTAINER_NAME started."
      else
        echo "Error: Failed to start $CONTAINER_NAME."
        echo "Check logs: $0 containers --name=$CONTAINER_NAME logs"
        exit 1
      fi
      ;;

    stop)
      require_name "stop"
      validate_service_name "$CONTAINER_NAME"
      if ! is_service_running "$CONTAINER_NAME"; then
        echo "$CONTAINER_NAME is not running."
        exit 0
      fi
      echo "Stopping $CONTAINER_NAME..."
      if compose_cmd stop "$CONTAINER_NAME" 2>&1; then
        echo "$CONTAINER_NAME stopped."
      else
        echo "Error: Failed to stop $CONTAINER_NAME."
        exit 1
      fi
      ;;

    restart)
      require_name "restart"
      validate_service_name "$CONTAINER_NAME"
      echo "Restarting $CONTAINER_NAME..."
      if compose_cmd restart "$CONTAINER_NAME" 2>&1; then
        echo "$CONTAINER_NAME restarted."
      else
        echo "Error: Failed to restart $CONTAINER_NAME."
        echo "Check logs: $0 containers --name=$CONTAINER_NAME logs"
        exit 1
      fi
      ;;

    logs)
      require_name "logs"
      validate_service_name "$CONTAINER_NAME"
      compose_cmd logs --tail=50 -f "$CONTAINER_NAME"
      ;;

    *)
      echo "Error: Unknown containers action '$action'."
      echo ""
      echo "Usage:"
      echo "  $0 containers ls                            — list all containers"
      echo "  $0 containers status                        — health of all containers"
      echo "  $0 containers --name=<service> status       — status of one container"
      echo "  $0 containers --name=<service> start        — start a container"
      echo "  $0 containers --name=<service> stop         — stop a container"
      echo "  $0 containers --name=<service> restart      — restart a container"
      echo "  $0 containers --name=<service> logs         — tail container logs"
      exit 1
      ;;
  esac
}

# ── Dispatch ──

case "$SUBCOMMAND" in
  start)      cmd_start ;;
  stop)       cmd_stop ;;
  restart)    cmd_restart ;;
  status)     cmd_status ;;
  containers) cmd_containers ;;
  *)
    echo "Unknown command: $SUBCOMMAND"
    echo "Usage:"
    echo "  $0 <start|stop|restart|status>"
    echo "  $0 containers <ls|status>"
    echo "  $0 containers --name=<service> <start|stop|restart|status|logs>"
    exit 1
    ;;
esac
