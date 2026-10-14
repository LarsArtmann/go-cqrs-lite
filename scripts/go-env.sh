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
# the verify apps use — all on / with headroom. The /home/lars-backed caches
# are conditional: GitHub-hosted runners have no /home/lars, and a GOCACHE
# pointing at an uncreatable path kills every go build ("mkdir /home/lars:
# permission denied" — the 2026-10-14 Benchmark Regression Detection class,
# same probe as the tmpdir fallback below). Runners keep go's defaults.
cqrs_cache_root=/home/lars/projects
if mkdir -p "$cqrs_cache_root" 2>/dev/null && [ -w "$cqrs_cache_root" ]; then
	mkdir -p "$cqrs_cache_root/.gocache-disk" "$cqrs_cache_root/.golangci-disk" 2>/dev/null || true
	go_env_set GOCACHE "$cqrs_cache_root/.gocache-disk"
	go_env_set GOLANGCI_LINT_CACHE "$cqrs_cache_root/.golangci-disk"
fi
unset -v cqrs_cache_root
go_env_set GOMODCACHE /tmp/gomod-verify
go_env_set GOPATH /tmp/gopath-verify

# Temp dirs: prefer the shared disk path, but fall back to the system default
# when it cannot exist — GitHub-hosted runners have no /home/lars, and a
# GOTMPDIR pointing at an uncreatable path breaks every mktemp in the job
# (the 2026-10-13 Benchmark Regression Detection failure class).
cqrs_tmpdir=/home/lars/projects/.gotmp
if mkdir -p "$cqrs_tmpdir" 2>/dev/null && [ -w "$cqrs_tmpdir" ]; then
	go_env_set GOTMPDIR "$cqrs_tmpdir"
	go_env_set TMPDIR "$cqrs_tmpdir"
fi
unset -v cqrs_tmpdir

unset -f go_env_set go_env_repo_root
