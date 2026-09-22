#!/usr/bin/env bash
# go-env.sh — the ONE sourced env chain for every go/buildflow invocation
# in this repo (T05, 2026-09-22; contract: docs/agents/gowork-modes.md).
#
# Why this exists: the ambient session env on this host carries
# GOTOOLCHAIN=local (the nixpkgs Go wrapper default) plus caches pointed at
# /mnt/buildcache (which hit 100% on 2026-09-22). Under that env every
# workspace go command false-fails ("go.work requires go >= 1.27.1"),
# BuildFlow's gomod scans silently skip, and cache writes die with ENOSPC.
# Four sessions re-derived this chain by hand before it was written down
# here. Source it — never re-spell it.
#
# Usage (first line of any script or hook that runs go/buildflow):
#   source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/go-env.sh"
#   # or, from anywhere:
#   source /home/lars/projects/go-cqrs-lite/scripts/go-env.sh
#
# Behavior: overrides GOTOOLCHAIN (local -> auto), redirects the cache
# chain to the disk-backed paths from gowork-modes.md, echoes one line per
# variable it actually changed (silent when the env is already correct).
# GOFLAGS/GOWORK are deliberately NOT set here — the GOWORK decision table
# (docs/agents/gowork-modes.md) is per-command-class, not ambient.

go_env_repo_root() {
	local script="${BASH_SOURCE[0]}"
	if [[ -L "$script" ]]; then
		script="$(readlink -f "$script")"
	fi
	(cd "$(dirname "$script")/.." && pwd)
}

GO_ENV_ROOT="${GO_ENV_ROOT:-$(go_env_repo_root)}"

# Idempotent single-variable override with change notice.
go_env_set() {
	local value="$2" current="${!1:-}"
	if [[ "$current" != "$value" ]]; then
		echo "go-env: $1='$current' -> '$value'" >&2
		export "$1=$value"
	fi
}

# GOTOOLCHAIN=local pins the toolchain to the host go (1.26.7) which is
# older than the workspace contract (1.27.1) — the false-fail class.
go_env_set GOTOOLCHAIN auto

# Cache chain off /mnt/buildcache (full disk, 2026-09-22) onto the paths
# the verify apps already use — all on / with headroom.
go_env_set GOCACHE /home/lars/projects/.gocache-disk
go_env_set GOMODCACHE /tmp/gomod-verify
go_env_set GOPATH /tmp/gopath-verify
go_env_set GOLANGCI_LINT_CACHE /home/lars/projects/.golangci-disk
go_env_set GOTMPDIR /home/lars/projects/.gotmp
go_env_set TMPDIR /home/lars/projects/.gotmp

mkdir -p "$GOCACHE" "$GOLANGCI_LINT_CACHE" "$GOTMPDIR" 2>/dev/null || {
	echo "go-env: WARN — could not create cache dirs under /home/lars/projects" >&2
}

unset -f go_env_set go_env_repo_root
