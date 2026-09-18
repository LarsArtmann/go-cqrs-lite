#!/usr/bin/env bash
# install-hooks.sh, installs THE canonical pre-commit hook.
#
# Reconciliation (2026-09-18, TODO "pre-commit hook hardening"): there used to
# be two competing hook sources, this script's BuildFlow heredoc (written to
# .git/hooks/pre-commit) and the repo gate chain (scripts/pre-commit.sh →
# .githooks/pre-commit). With core.hooksPath=.githooks the BuildFlow copy was
# silently dead; on fresh clones (no hooksPath) the repo gates were dead.
# Now there is exactly one hook: scripts/pre-commit.sh, installed to the
# hooksPath directory, WITH BuildFlow chained inside it (so `buildflow
# precommit install` can no longer wipe the repo gates, it writes
# .git/hooks/pre-commit, which hooksPath overrides).
#
# Fresh-clone bootstrapping: `nix develop` sets core.hooksPath and installs
# the hook automatically (see flake.nix devShell shellHook); this script is
# the manual fallback.
#
# Usage: ./scripts/install-hooks.sh

set -euo pipefail

if [ ! -d .git ] && [ ! -f .git ]; then
	echo "Error: not a git repository root" >&2
	exit 1
fi

# One hook directory, referenced by config, never per-clone .git/hooks state.
git config core.hooksPath .githooks

mkdir -p .githooks
cp scripts/pre-commit.sh .githooks/pre-commit
chmod +x .githooks/pre-commit

if command -v buildflow &>/dev/null; then
	echo "Installed canonical pre-commit hook to .githooks/pre-commit (hooksPath set)."
	echo "BuildFlow found: its pre-commit gate is chained inside the hook."
else
	echo "Installed canonical pre-commit hook to .githooks/pre-commit (hooksPath set)."
	echo "NOTE: buildflow not in PATH, run 'nix develop' to chain the BuildFlow gate."
fi
