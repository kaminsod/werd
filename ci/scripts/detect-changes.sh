#!/bin/bash
set -euo pipefail

# Detect which packages changed relative to the base branch.
# Outputs space-separated list of changed packages.

BASE=${1:-origin/master}

changed=""

if git diff --name-only "$BASE"...HEAD | grep -q '^src/go/api/'; then
  changed="$changed api"
fi

if git diff --name-only "$BASE"...HEAD | grep -q '^src/web/'; then
  changed="$changed web"
fi

if git diff --name-only "$BASE"...HEAD | grep -q '^src/go/monitor-reddit/'; then
  changed="$changed monitor-reddit"
fi

if git diff --name-only "$BASE"...HEAD | grep -q '^src/go/monitor-hn/'; then
  changed="$changed monitor-hn"
fi

if git diff --name-only "$BASE"...HEAD | grep -q '^src/deploy/'; then
  changed="$changed deploy"
fi

echo "${changed# }"
