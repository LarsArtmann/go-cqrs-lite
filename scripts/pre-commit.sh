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
if grep -R 'fmt\.Printf' --include='*.go' . |
	grep -v '_test.go' |
	grep -v '/example/' |
	grep -v '/testdata/' |
	grep -v '/cmd/' |
	grep -v 'doc.go'; then
	echo "ERROR: fmt.Printf found in production code (allowed in tests/examples/cmd/doc comments)"
	exit 1
fi

echo "==> Checking api_surface.txt is up to date"
if ! (cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" .); then
	echo "ERROR: docs/api_surface.txt is stale. Run: cd cmd/api-stability && GOWORK=off go run -tags 'goexperiment.jsonv2' . --update"
	exit 1
fi

echo "==> Syntax gate: staged .go files must parse and be gofmt-clean"
bash scripts/check-staged-go.sh

# ── Staged-aware cheap gates (seconds, not minutes) ─────────────────────────
# Each gate runs only when the commit touches files it can actually judge.
staged() {
	git diff --cached --name-only --diff-filter=ACMR | grep -E "$1" || true
}

if [ -n "$(staged 'scripts/.*\.sh$')" ]; then
	echo "==> shellcheck on scripts/ (commit touches scripts/)"
	if ! nix-shell -p shellcheck --run 'shellcheck scripts/*.sh'; then
		echo "ERROR: shellcheck findings in scripts (repo gate is zero findings)"
		exit 1
	fi
fi

if [ -n "$(staged 'go\.(mod|work|sum)$|flake\.nix$')" ]; then
	echo "==> Workspace sync check (go.work ↔ flake.nix)"
	if ! bash scripts/check-workspace-sync.sh; then
		echo "ERROR: go.work and flake.nix testModules are out of sync. Run: go work sync + update flake.nix"
		exit 1
	fi
fi

if [ -n "$(staged '^CHANGELOG\.md$')" ]; then
	echo "==> CHANGELOG symbol citations"
	if ! bash scripts/check-changelog-symbols.sh; then
		echo "ERROR: CHANGELOG [Unreleased] cites pkg.Symbol entries that do not exist"
		exit 1
	fi
fi

echo "✅ Pre-commit checks passed"
