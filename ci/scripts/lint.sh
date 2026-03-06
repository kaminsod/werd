#!/bin/bash
set -euo pipefail

echo "=== Linting ==="

echo "Go vet (api)..."
cd src/go/api && go vet ./... && cd -

echo "Go vet (monitor-reddit)..."
cd src/go/monitor-reddit && go vet ./... && cd -

echo "Go vet (monitor-hn)..."
cd src/go/monitor-hn && go vet ./... && cd -

echo "Lint OK"
