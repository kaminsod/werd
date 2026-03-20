#!/usr/bin/env bash
# First-time provisioning for the email verification VPS.
# Installs podman, configures UFW, deploys Mailpit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Load deploy config
if [ -f "${SCRIPT_DIR}/.env" ]; then
  set -a; source "${SCRIPT_DIR}/.env"; set +a
fi

HOST="${DEPLOY_HOST:?Set DEPLOY_HOST in .env}"
USER="${DEPLOY_USER:-root}"
DIR="${DEPLOY_DIR:-/opt/email-verification}"

echo "=== Provisioning ${USER}@${HOST} ==="

ssh "${USER}@${HOST}" bash -s -- "${DIR}" <<'REMOTE'
set -euo pipefail
DIR="$1"

echo "── Installing podman + podman-compose ──"
apt-get update -qq
apt-get install -y -qq podman python3-pip >/dev/null
pip3 install --break-system-packages podman-compose 2>/dev/null || pip3 install podman-compose

echo "── Configuring container registries ──"
mkdir -p /etc/containers
if ! grep -q 'unqualified-search-registries' /etc/containers/registries.conf 2>/dev/null; then
  echo 'unqualified-search-registries = ["docker.io"]' >> /etc/containers/registries.conf
fi

echo "── Configuring UFW ──"
apt-get install -y -qq ufw >/dev/null
ufw --force reset >/dev/null
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment "SSH"
ufw allow 25/tcp   comment "SMTP"
ufw allow 8025/tcp comment "Mailpit API/UI"
# Podman uses DNAT for port forwarding — traffic traverses the FORWARD chain,
# so we need route rules in addition to the INPUT rules above.
ufw route allow proto tcp from any to any port 25  comment "SMTP forward to container"
ufw route allow proto tcp from any to any port 8025 comment "Mailpit API forward to container"
ufw --force enable

echo "── Creating ${DIR} ──"
mkdir -p "${DIR}"

echo "── Done with remote setup ──"
REMOTE

echo "── Copying files to ${HOST}:${DIR} ──"
scp "${SCRIPT_DIR}/docker-compose.yml" "${USER}@${HOST}:${DIR}/docker-compose.yml"
[ -f "${SCRIPT_DIR}/.env" ] && scp "${SCRIPT_DIR}/.env" "${USER}@${HOST}:${DIR}/.env"

echo "── Generating TLS certificate for STARTTLS ──"
ssh "${USER}@${HOST}" bash -s -- "${DIR}" <<'TLS'
set -euo pipefail
DIR="$1"

# Create TLS volume if needed
podman volume create email-verification_mailpit-tls 2>/dev/null || true
TLS_DIR=$(podman volume inspect email-verification_mailpit-tls | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['Mountpoint'])")

if [ ! -f "$TLS_DIR/cert.pem" ]; then
  openssl req -x509 -newkey rsa:2048 -keyout "$TLS_DIR/key.pem" -out "$TLS_DIR/cert.pem" \
    -days 3650 -nodes -subj "/CN=mail.datazo.net" 2>/dev/null
  echo "  Generated self-signed cert for STARTTLS"
else
  echo "  TLS cert already exists, skipping"
fi
TLS

echo "── Starting Mailpit ──"
ssh "${USER}@${HOST}" "cd ${DIR} && podman-compose up -d"

echo "── Waiting for health check ──"
sleep 5

API_PORT="${API_PORT:-8025}"
if curl -sf "http://${HOST}:${API_PORT}/api/v1/info" > /dev/null 2>&1; then
  echo "OK: Mailpit is running on ${HOST}:${API_PORT}"
else
  echo "WARNING: Mailpit API not yet responding on ${HOST}:${API_PORT}"
  echo "  Check with: ssh ${USER}@${HOST} 'cd ${DIR} && podman-compose logs'"
fi
