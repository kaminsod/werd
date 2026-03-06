#!/bin/bash
# Creates additional databases for each service in the shared PostgreSQL instance.
# This script runs automatically on first container start via docker-entrypoint-initdb.d.

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE postiz;
    CREATE DATABASE activepieces;
    CREATE DATABASE mattermost;
    CREATE DATABASE plausible;
    CREATE DATABASE temporal;
EOSQL

echo "All service databases created."
