#!/bin/bash
set -euo pipefail

echo "=== CI Summary ==="
echo "Commit: $(git rev-parse --short HEAD)"
echo "Branch: $(git rev-parse --abbrev-ref HEAD)"
echo "Date:   $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "==================="
