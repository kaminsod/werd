#!/bin/bash
set -euo pipefail

echo "=== Werd CI Runner ==="
echo "Running: $*"

exec "$@"
