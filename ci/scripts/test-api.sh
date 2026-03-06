#!/bin/bash
set -euo pipefail

echo "=== Testing API Server ==="
cd src/go/api
go test ./... -v -count=1
echo "API tests OK"
