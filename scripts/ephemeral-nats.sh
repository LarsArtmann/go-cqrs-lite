#!/usr/bin/env bash
# ephemeral-nats.sh — Start an ephemeral NATS server from nixpkgs.
#
# For Watermill adapter testing with a real NATS JetStream broker.
# No Docker daemon, no VM — just a process from the Nix store.
#
# Usage:
#   bash scripts/ephemeral-nats.sh                # run Watermill tests
#   bash scripts/ephemeral-nats.sh go test ./...  # arbitrary go command
#
# Environment:
#   NATS_PORT — override the port (default: auto-select free port)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Pick a free port if not overridden
if [ -z "${NATS_PORT:-}" ]; then
    NATS_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null \
        || echo "4222")
fi

NATS_PID=""
JETSTREAM_DIR=$(mktemp -d /tmp/cqrs-nats-XXXXXX)

cleanup() {
    if [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
        kill "$NATS_PID" 2>/dev/null || true
        wait "$NATS_PID" 2>/dev/null || true
    fi
    rm -rf "$JETSTREAM_DIR"
}
trap cleanup EXIT INT TERM

echo "==> Starting ephemeral NATS (port $NATS_PORT, JetStream enabled)"
nats-server \
    --port "$NATS_PORT" \
    --jetstream \
    --store_dir "$JETSTREAM_DIR" \
    --log level=warn \
    &
NATS_PID=$!

# Wait for NATS to accept connections
for i in $(seq 1 30); do
    if (echo > /dev/tcp/127.0.0.1/"$NATS_PORT") 2>/dev/null; then
        echo "==> NATS ready"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: NATS did not become ready within 30s"
        exit 1
    fi
    sleep 0.5
done

export NATS_URL="nats://127.0.0.1:$NATS_PORT"
echo "==> NATS_URL=$NATS_URL"

# Run the requested command
if [ $# -gt 0 ]; then
    echo "==> Running: $*"
    "$@"
else
    echo "==> No command specified. NATS is running at $NATS_URL"
    echo "    Set NATS_URL and run your tests in another terminal."
    echo "    Press Ctrl+C to stop."
    wait "$NATS_PID"
fi
