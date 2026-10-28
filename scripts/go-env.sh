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
# Behavior: overrides GOTOOLCHAIN (local -> auto), forces the cache chain
# onto go's real defaults under ~/.cache and ~/go (the ambient session env
# exports /mnt/buildcache paths), echoes one line per
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

# Cache chain pinned to go's REAL defaults (retargeted 2026-10-28, SystemNix
# docs/todo/storage.md row 184): the ambient session env exports
# /mnt/buildcache paths (87% full — the 2026-09-22 ENOSPC class), so the
# defaults must be FORCED, never left to the ambient env. The old
# /home/lars/projects/.gocache-disk chain churned the snapshotted QLC root
# subvol; ~/.cache is the unsnapshotted @cache-home subvol. GOLANGCI_LINT_CACHE
# must stay explicit: golangci derives it from GOCACHE's parent otherwise and
# lands on /mnt/buildcache (ambient bleed).
go_env_set GOCACHE "$HOME/.cache/go-build"
go_env_set GOMODCACHE "$HOME/go/pkg/mod"
go_env_set GOPATH "$HOME/go"
go_env_set GOLANGCI_LINT_CACHE "$HOME/.cache/golangci-lint"

# Build scratch stays off the shared 48G /tmp tmpfs (the ENOSPC link-job
# class) and off /mnt/buildcache: a private dir under unsnapshotted
# ~/.cache. go does NOT create GOTMPDIR itself, so probe-then-set with a
# system-tmp fallback keeps foreign hosts (CI runners) working.
cqrs_tmpdir="$HOME/.cache/go-tmp"
if mkdir -p "$cqrs_tmpdir" 2>/dev/null && [ -w "$cqrs_tmpdir" ]; then
	go_env_set GOTMPDIR "$cqrs_tmpdir"
	go_env_set TMPDIR "$cqrs_tmpdir"
else
	go_env_set GOTMPDIR /tmp
	go_env_set TMPDIR /tmp
fi
unset -v cqrs_tmpdir

unset -f go_env_set go_env_repo_root
