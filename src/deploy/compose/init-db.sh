#!/bin/bash
# Creates per-service databases and users in the shared PostgreSQL instance.
# Each service gets its own user with access restricted to its own database.
# This script runs automatically on first container start via docker-entrypoint-initdb.d.
#
# The 'werd' superuser (POSTGRES_USER) is created by the postgres image itself.
# It owns the 'werd' database and can access all databases for admin/migration tasks.

set -euo pipefail

create_service_db() {
  local db="$1"
  local user="$2"
  local pass="$3"

  echo "Creating database '$db' with user '$user'..."

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER ${user} WITH PASSWORD '${pass}';
    CREATE DATABASE ${db} OWNER ${user};
    REVOKE ALL ON DATABASE ${db} FROM PUBLIC;
    GRANT ALL PRIVILEGES ON DATABASE ${db} TO ${user};
EOSQL
}

# Service databases — each gets a dedicated user.
# Passwords are passed via environment variables from docker-compose.yml.
create_service_db "umami" "umami" "${UMAMI_DB_PASSWORD}"

echo "All service databases and users created."
