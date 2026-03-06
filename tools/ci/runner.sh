#!/bin/bash
set -euo pipefail

# Build and run the CI container.

ACTION="${1:-run}"
IMAGE="werd-ci:latest"
CONTAINERFILE="ci/Containerfile"

case "$ACTION" in
  build)
    echo "Building CI image..."
    podman build -t "$IMAGE" -f "$CONTAINERFILE" .
    ;;
  run)
    echo "Running CI..."
    podman run --rm -v "$(pwd):/workspace:Z" "$IMAGE" bash -c "
      ci/scripts/summary.sh
      ci/scripts/build-api.sh
      ci/scripts/build-monitors.sh
      ci/scripts/test-api.sh
      ci/scripts/lint.sh
    "
    ;;
  *)
    echo "Usage: $0 {build|run}"
    exit 1
    ;;
esac
