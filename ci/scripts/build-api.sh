#!/bin/bash
set -euo pipefail

echo "=== Building API Server ==="
cd src/go/api
go build -o /dev/null ./cmd/werd-api
echo "API build OK"
