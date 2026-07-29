#!/usr/bin/env bash
set -euo pipefail

# Pre-commit checks for go-cqrs-lite.
# Install with: nix run .#install-hooks

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> Running nix fmt"
nix fmt

if ! git diff --quiet --exit-code; then
  echo "ERROR: nix fmt changed files. Stage the formatted changes and commit again."
  exit 1
fi

echo "==> Building all modules (catches broken-code commits, incl. auto-commit daemon)"
# Compile-check the whole workspace before allowing a commit. This is the gate
# that prevents the auto-commit daemon (or a human) from shipping code that does
# not build — the recurring "stale GREEN" / broken-daemon-commit class of failure.
if ! go build -tags "goexperiment.jsonv2" ./...; then
  echo "ERROR: go build failed. Fix compile errors before committing."
  exit 1
fi

echo "==> Checking for fmt.Printf in production code"
# Allow fmt.Printf in tests, examples, generated/testdata files, and cmd tooling.
if grep -R 'fmt\.Printf' --include='*.go' . \
    | grep -v '_test.go' \
    | grep -v '/example/' \
    | grep -v '/testdata/' \
    | grep -v '/cmd/' \
    | grep -v 'doc.go'; then
  echo "ERROR: fmt.Printf found in production code (allowed in tests/examples/cmd/doc comments)"
  exit 1
fi

echo "==> Checking api_surface.txt is up to date"
(cd cmd/api-stability && GOWORK=off go run main.go)

echo "✅ Pre-commit checks passed"
