#!/bin/bash
set -euo pipefail

echo "=== Building Monitors ==="
for mod in monitor-reddit monitor-hn; do
  echo "Building $mod..."
  cd "src/go/$mod"
  go build -o /dev/null "./cmd/$mod"
  cd -
done
echo "Monitor builds OK"
