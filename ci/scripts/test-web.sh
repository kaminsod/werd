#!/bin/bash
set -euo pipefail

echo "=== Testing Web Dashboard ==="
cd src/web
npm ci
npm run typecheck
echo "Web tests OK"
