#!/bin/bash
set -euo pipefail

# Generates secure random secrets and writes them to the compose .env file.
# Usage: ./tools/generate-secrets.sh [path/to/.env]

ENV_FILE="${1:-src/deploy/compose/.env}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: $ENV_FILE not found. Copy .env.example first:"
  echo "  cp src/deploy/compose/.env.example src/deploy/compose/.env"
  exit 1
fi

gen_secret() {
  openssl rand -base64 32 | tr -d '/+=' | head -c 48
}

echo "Generating secrets..."

secrets=(
  POSTGRES_PASSWORD
  UMAMI_DB_PASSWORD
  REDIS_PASSWORD
  WERD_JWT_SECRET
  WERD_ADMIN_PASSWORD
)

for key in "${secrets[@]}"; do
  value=$(gen_secret)
  if grep -q "^${key}=changeme" "$ENV_FILE" 2>/dev/null; then
    sed -i "s|^${key}=changeme|${key}=${value}|" "$ENV_FILE"
    echo "  Generated: $key"
  elif grep -q "^${key}=$" "$ENV_FILE" 2>/dev/null; then
    sed -i "s|^${key}=$|${key}=${value}|" "$ENV_FILE"
    echo "  Generated: $key"
  else
    echo "  Skipped:   $key (already set or not found)"
  fi
done

echo "Done. Secrets written to $ENV_FILE"
