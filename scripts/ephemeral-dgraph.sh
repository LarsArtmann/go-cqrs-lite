#!/usr/bin/env bash
# ephemeral-dgraph.sh — Start an ephemeral Dgraph (Zero + Alpha) from nixpkgs.
#
# For dgraphengine integration testing with a real Dgraph instance.
# No Docker daemon, no VM — just processes from the Nix store.
#
# Usage:
#   bash scripts/ephemeral-dgraph.sh                # run dgraphengine tests
#   bash scripts/ephemeral-dgraph.sh go test ./...  # arbitrary go command
#
# Environment:
#   DGRAPH_ALPHA_GRPC — override Alpha gRPC port (default: auto-select)
#   DGRAPH_ZERO_GRPC  — override Zero gRPC port (default: auto-select)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

pick_free_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()' 2>/dev/null
}

# Pick free ports if not overridden.
if [ -z "${DGRAPH_ZERO_GRPC:-}" ]; then
    DGRAPH_ZERO_GRPC=$(pick_free_port || echo "15580")
fi
if [ -z "${DGRAPH_ALPHA_GRPC:-}" ]; then
    DGRAPH_ALPHA_GRPC=$(pick_free_port || echo "19080")
fi

# Derive port offsets from defaults (Zero: 5080, Alpha: 9080).
ZERO_OFFSET=$((DGRAPH_ZERO_GRPC - 5080))
ALPHA_OFFSET=$((DGRAPH_ALPHA_GRPC - 9080))

DGRAPH_DIR=$(mktemp -d /tmp/cqrs-dgraph-XXXXXX)
ZERO_PID=""
ALPHA_PID=""

cleanup() {
    if [ -n "$ALPHA_PID" ] && kill -0 "$ALPHA_PID" 2>/dev/null; then
        kill "$ALPHA_PID" 2>/dev/null || true
        wait "$ALPHA_PID" 2>/dev/null || true
    fi
    if [ -n "$ZERO_PID" ] && kill -0 "$ZERO_PID" 2>/dev/null; then
        kill "$ZERO_PID" 2>/dev/null || true
        wait "$ZERO_PID" 2>/dev/null || true
    fi
    rm -rf "$DGRAPH_DIR"
}
trap cleanup EXIT INT TERM

echo "==> Starting Dgraph Zero (gRPC $DGRAPH_ZERO_GRPC, offset $ZERO_OFFSET)"
dgraph zero \
    --my="localhost:$DGRAPH_ZERO_GRPC" \
    --port_offset="$ZERO_OFFSET" \
    --wal "$DGRAPH_DIR/zw" \
    --logtostderr \
    >"$DGRAPH_DIR/zero.log" 2>&1 &
ZERO_PID=$!

# Wait for Zero to become leader.
for i in $(seq 1 60); do
    if (echo > /dev/tcp/127.0.0.1/"$DGRAPH_ZERO_GRPC") 2>/dev/null; then
        echo "==> Zero ready"
        break
    fi
    if [ "$i" -eq 60 ]; then
        echo "ERROR: Zero did not become ready within 30s"
        cat "$DGRAPH_DIR/zero.log" 2>/dev/null || true
        exit 1
    fi
    sleep 0.5
done

ALPHA_INTERNAL=$((7080 + ALPHA_OFFSET))
echo "==> Starting Dgraph Alpha (gRPC $DGRAPH_ALPHA_GRPC, internal $ALPHA_INTERNAL)"
dgraph alpha \
    --my="localhost:$ALPHA_INTERNAL" \
    --zero="localhost:$DGRAPH_ZERO_GRPC" \
    --port_offset="$ALPHA_OFFSET" \
    --postings "$DGRAPH_DIR/p" \
    --wal "$DGRAPH_DIR/w" \
    --security "whitelist=0.0.0.0/0" \
    --logtostderr \
    >"$DGRAPH_DIR/alpha.log" 2>&1 &
ALPHA_PID=$!

# Wait for Alpha to accept gRPC connections.
for i in $(seq 1 60); do
    if (echo > /dev/tcp/127.0.0.1/"$DGRAPH_ALPHA_GRPC") 2>/dev/null; then
        echo "==> Alpha ready"
        break
    fi
    if [ "$i" -eq 60 ]; then
        echo "ERROR: Alpha did not become ready within 30s"
        cat "$DGRAPH_DIR/alpha.log" 2>/dev/null || true
        cat "$DGRAPH_DIR/zero.log" 2>/dev/null || true
        exit 1
    fi
    sleep 0.5
done

# Extra settle time for Alpha to sync with Zero.
sleep 2

export DGRAPH_ADDR="localhost:$DGRAPH_ALPHA_GRPC"
echo "==> DGRAPH_ADDR=$DGRAPH_ADDR"
echo "==> Logs in $DGRAPH_DIR"

# Run the requested command.
if [ $# -gt 0 ]; then
    echo "==> Running: $*"
    "$@"
else
    echo "==> No command specified. Dgraph is running at $DGRAPH_ADDR"
    echo "    Set DGRAPH_ADDR and run your tests in another terminal."
    echo "    Press Ctrl+C to stop."
    wait "$ALPHA_PID"
fi
