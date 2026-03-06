#!/bin/bash
set -euo pipefail

echo "=== Building Web Dashboard ==="
cd src/web
npm ci
npm run build
echo "Web build OK"
