#!/usr/bin/env bash
# Incremental deploy: push updated config and restart Mailpit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Load deploy config
if [ -f "${SCRIPT_DIR}/.env" ]; then
  set -a; source "${SCRIPT_DIR}/.env"; set +a
fi

HOST="${DEPLOY_HOST:?Set DEPLOY_HOST in .env}"
USER="${DEPLOY_USER:-root}"
DIR="${DEPLOY_DIR:-/opt/email-verification}"

echo "=== Deploying to ${USER}@${HOST}:${DIR} ==="

echo "── Copying files ──"
scp "${SCRIPT_DIR}/docker-compose.yml" "${USER}@${HOST}:${DIR}/docker-compose.yml"
[ -f "${SCRIPT_DIR}/.env" ] && scp "${SCRIPT_DIR}/.env" "${USER}@${HOST}:${DIR}/.env"

echo "── Pulling latest image and restarting ──"
ssh "${USER}@${HOST}" "cd ${DIR} && podman-compose pull && podman-compose up -d --force-recreate"

echo "── Waiting for health check ──"
sleep 5

API_PORT="${API_PORT:-8025}"
if curl -sf "http://${HOST}:${API_PORT}/api/v1/info" > /dev/null 2>&1; then
  echo "OK: Mailpit is running on ${HOST}:${API_PORT}"
else
  echo "WARNING: Mailpit API not yet responding on ${HOST}:${API_PORT}"
  echo "  Check with: ssh ${USER}@${HOST} 'cd ${DIR} && podman-compose logs'"
fi
