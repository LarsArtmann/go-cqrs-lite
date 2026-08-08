#!/usr/bin/env bash
# ephemeral-redis.sh — Start an ephemeral Redis instance from nixpkgs.
#
# For Watermill adapter testing with a real Redis Streams broker.
# No Docker daemon, no VM — just a process from the Nix store.
#
# Usage:
#   bash scripts/ephemeral-redis.sh                # run Watermill tests
#   bash scripts/ephemeral-redis.sh go test ./...  # arbitrary go command
#
# Environment:
#   REDIS_PORT — override the port (default: auto-select free port)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Pick a free port if not overridden
if [ -z "${REDIS_PORT:-}" ]; then
    REDIS_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null \
        || echo "6390")
fi

REDIS_PID=""
REDIS_DIR=$(mktemp -d /tmp/cqrs-redis-XXXXXX)

cleanup() {
    if [ -n "$REDIS_PID" ] && kill -0 "$REDIS_PID" 2>/dev/null; then
        kill "$REDIS_PID" 2>/dev/null || true
        wait "$REDIS_PID" 2>/dev/null || true
    fi
    rm -rf "$REDIS_DIR"
}
trap cleanup EXIT INT TERM

echo "==> Starting ephemeral Redis (port $REDIS_PORT)"
redis-server --port "$REDIS_PORT" --dir "$REDIS_DIR" --save "" --appendonly no --daemonize no --logfile "" &
REDIS_PID=$!

# Wait for Redis to accept connections
for i in $(seq 1 30); do
    if redis-cli -p "$REDIS_PORT" ping 2>/dev/null | grep -q PONG; then
        echo "==> Redis ready"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: Redis did not become ready within 30s"
        exit 1
    fi
    sleep 0.5
done

export REDIS_URL="redis://127.0.0.1:$REDIS_PORT"
echo "==> REDIS_URL=$REDIS_URL"

# Run the requested command
if [ $# -gt 0 ]; then
    echo "==> Running: $*"
    "$@"
else
    echo "==> No command specified. Redis is running at $REDIS_URL"
    echo "    Set REDIS_URL and run your tests in another terminal."
    echo "    Press Ctrl+C to stop."
    wait "$REDIS_PID"
fi
