#!/bin/bash
set -euo pipefail

# Bootstrap local development environment.

echo "=== Werd Dev Setup ==="

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "Error: Go is not installed"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Error: Node.js is not installed"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "Error: npm is not installed"; exit 1; }

echo "Go:   $(go version)"
echo "Node: $(node --version)"
echo "npm:  $(npm --version)"

# Go workspace sync
echo ""
echo "Syncing Go workspace..."
cd src/go && go work sync && cd -

# Install web dependencies
echo ""
echo "Installing web dependencies..."
cd src/web && npm install && cd -

# Copy .env if needed
if [ ! -f src/deploy/compose/.env ]; then
  echo ""
  echo "Creating .env from template..."
  cp src/deploy/compose/.env.example src/deploy/compose/.env
  echo "  Edit src/deploy/compose/.env with your configuration."
fi

echo ""
echo "=== Dev setup complete ==="
echo ""
echo "Next steps:"
echo "  make dev-api   # Start API server"
echo "  make dev-web   # Start dashboard dev server"
echo "  make compose-up # Start all services via compose"
