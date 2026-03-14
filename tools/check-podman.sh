#!/bin/bash
set -euo pipefail

# Check container runtime prerequisites for Werd deployment.
# Advisory only — does not auto-apply changes.

PASS=0
WARN=0
FAIL=0

pass() { printf "  \033[32m✓\033[0m %s\n" "$1"; PASS=$((PASS + 1)); }
warn() { printf "  \033[33m!\033[0m %s\n" "$1"; WARN=$((WARN + 1)); }
fail() { printf "  \033[31m✗\033[0m %s\n" "$1"; FAIL=$((FAIL + 1)); }

echo "=== Container Runtime Check ==="
echo ""

# ── Detect runtime ──

RUNTIME=""
if command -v podman >/dev/null 2>&1; then
    RUNTIME="podman"
    pass "Podman $(podman --version | awk '{print $NF}')"
elif command -v docker >/dev/null 2>&1; then
    RUNTIME="docker"
    pass "Docker $(docker --version | awk '{print $3}' | tr -d ',')"
else
    fail "No container runtime found (install Podman or Docker)"
fi

# ── Compose tool ──

if [ "$RUNTIME" = "podman" ]; then
    if command -v podman-compose >/dev/null 2>&1; then
        pass "podman-compose $(podman-compose --version 2>/dev/null | head -1 | awk '{print $NF}')"
    elif command -v docker-compose >/dev/null 2>&1; then
        warn "podman-compose not found, but docker-compose is available"
        echo "        Install: pip install podman-compose"
    else
        fail "No compose tool found (install podman-compose)"
        echo "        Install: pip install podman-compose"
    fi
elif [ "$RUNTIME" = "docker" ]; then
    if docker compose version >/dev/null 2>&1; then
        pass "docker compose $(docker compose version --short 2>/dev/null)"
    elif command -v docker-compose >/dev/null 2>&1; then
        pass "docker-compose (standalone)"
    else
        fail "No compose tool found"
    fi
fi

# ── Rootless Podman: privileged ports ──

if [ "$RUNTIME" = "podman" ]; then
    echo ""
    echo "Rootless Podman checks:"

    PORT_START=$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null || echo "unknown")
    if [ "$PORT_START" = "unknown" ]; then
        warn "Could not read net.ipv4.ip_unprivileged_port_start"
    elif [ "$PORT_START" -le 80 ]; then
        pass "Unprivileged port start = $PORT_START (ports 80/443 available)"
    else
        fail "Unprivileged port start = $PORT_START (Caddy needs ≤ 80)"
        echo "        Fix: sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80"
        echo "        Persist: echo 'net.ipv4.ip_unprivileged_port_start=80' | sudo tee /etc/sysctl.d/podman-privileged-ports.conf"
    fi

    # ── Podman socket ──

    SOCKET_PATH="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock"
    if [ -S "$SOCKET_PATH" ]; then
        pass "Podman socket active ($SOCKET_PATH)"
    else
        warn "Podman socket not active (needed for docker-compose compatibility)"
        echo "        Fix: systemctl --user enable --now podman.socket"
    fi
fi

# ── Summary ──

echo ""
echo "---"
printf "Results: \033[32m%d passed\033[0m" "$PASS"
[ "$WARN" -gt 0 ] && printf ", \033[33m%d warnings\033[0m" "$WARN"
[ "$FAIL" -gt 0 ] && printf ", \033[31m%d failed\033[0m" "$FAIL"
echo ""

exit "$FAIL"
