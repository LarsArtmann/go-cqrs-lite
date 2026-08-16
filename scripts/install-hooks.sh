#!/usr/bin/env bash
# install-hooks.sh
# Installs the BuildFlow pre-commit hook with scope detection.
#
# Usage: ./scripts/install-hooks.sh
#
# The hook skips lint for doc-only commits (.md, .html, .d2, .svg, .txt, .yaml)
# and runs BuildFlow in pre-commit mode (--staged-only, 300s timeout) otherwise.

set -euo pipefail

HOOK_PATH=".git/hooks/pre-commit"

if [ ! -d .git/hooks ]; then
	echo "Error: not a git repository root (.git/hooks not found)" >&2
	exit 1
fi

if ! command -v buildflow &>/dev/null; then
	echo "Error: buildflow not found in PATH. Run 'nix develop' first." >&2
	exit 1
fi

cat >"$HOOK_PATH" <<'HOOK_EOF'
#!/usr/bin/env bash
# BuildFlow Pre-Commit Hook
# Installed by scripts/install-hooks.sh
#
# Scope detection: skip lint for doc-only commits to save ~45s per commit.
# NOTE: `buildflow precommit install` REGENERATES this file and wipes the
# appended blocks below (api-stability + staged-syntax gates) — re-run
# scripts/install-hooks.sh after reinstalling BuildFlow's hook.

set -e

STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACMR)
NON_DOC_FILES=$(echo "$STAGED_FILES" | grep -vE '\.(md|html|d2|svg|txt|yaml|yml)$' || true)

if [ -z "$NON_DOC_FILES" ]; then
  echo "BuildFlow: skipping lint for doc-only commit"
  exit 0
fi

echo "Running BuildFlow pre-commit checks (max-time: 300s)..."

START=$(date +%s)

buildflow --build-mode pre-commit --staged-only --max-time 300s

git diff --cached --name-only --diff-filter=ACMR -z | xargs -0 -r git add

END=$(date +%s)
DURATION=$((END-START))

echo "BuildFlow completed in ${DURATION}s"

# ── Post-BuildFlow appended gates (wiped by `buildflow precommit install`;
#    this script is the canonical restorer) ─────────────────────────────────

# API surface golden check — prevents shipping exports without regenerating
# docs/api_surface.txt. Fast (~1s GOWORK=off run). Only when .go files staged.
ROOT="$(git rev-parse --show-toplevel)"
if echo "$STAGED_FILES" | grep -qE '\.go$' && echo "$STAGED_FILES" | grep -vqE '^(docs/|\.agents/|scripts/)'; then
	echo "==> Checking api_surface.txt is up to date"
	if ! (cd "$ROOT/cmd/api-stability" && GOWORK=off go run -tags "goexperiment.jsonv2" .); then
		echo "ERROR: docs/api_surface.txt is stale. Run: cd cmd/api-stability && GOWORK=off go run -tags 'goexperiment.jsonv2' . --update"
		exit 1
	fi
fi

# Staged .go syntax gate — blocks concurrent-session mid-write corruption
# ("func (w *workor)", "fojection." landed staged twice on 2026-08-16).
if echo "$STAGED_FILES" | grep -qE '\.go$'; then
	bash "$ROOT/scripts/check-staged-go.sh"
fi
HOOK_EOF

chmod +x "$HOOK_PATH"

echo "Installed pre-commit hook to $HOOK_PATH"
echo "Scope detection: doc-only commits skip lint, all others run BuildFlow."
echo "Includes post-BuildFlow gates: api-stability golden + staged-syntax."
