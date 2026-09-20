#!/usr/bin/env bash
# verify-lock.sh — advisory flock for verify/release windows (W3 Q5, 2026-09-20).
#
# Source this, then call verify_lock_acquire. On contention the acquirer
# fails LOUD with the holder's PID instead of two sessions interleaving
# writes into the same tree (the 10:31 corruption incident cost ~45 min).
# Advisory semantics: nothing outside these consumers enforces it, and
# VERIFY_LOCK=0 skips it entirely — the lock coordinates willing sessions.
#
# The lock lives OUTSIDE the worktree (${TMPDIR}/go-cqrs-lite-verify-<roothash>.lock):
# machine-local scheduling state must not pollute `git status` (the
# tag-release smoke test asserts a fully clean tree), and a shared host's
# clones of this repo then serialize against ONE lock, which is the point.
# flock releases on process death, so there is no stale-lock state to
# clean: if acquire fails, the holder is ALIVE right now.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/verify-lock.sh"
#   verify_lock_acquire                 # non-blocking; exit 1 on contention
#   verify_lock_acquire 120             # block up to 120s first
#   verify_lock_release                 # optional; trap EXIT does it anyway
#
# VERIFY_LOCK=0 ./whatever   # opt out (documented escape hatch)

VERIFY_LOCK_FD=9

verify_lock_path() {
	local root
	root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
	printf '%s/go-cqrs-lite-verify-%s.lock' \
		"${TMPDIR:-/tmp}" \
		"$(printf '%s' "$root" | sha256sum | cut -c1-12)"
}

verify_lock_acquire() {
	if [[ "${VERIFY_LOCK:-1}" == 0 ]]; then
		echo "verify-lock: VERIFY_LOCK=0 — advisory lock skipped"
		return 0
	fi
	if ! command -v flock >/dev/null 2>&1; then
		echo "verify-lock: flock(1) not found — skipping advisory lock (install util-linux)" >&2
		return 0
	fi
	local lock_path wait_secs=0
	lock_path="$(verify_lock_path)"
	if [[ "${1:-}" =~ ^[0-9]+$ ]]; then
		wait_secs="$1"
	fi
	exec {VERIFY_LOCK_FD}>>"$lock_path"
	if [[ "$wait_secs" -gt 0 ]]; then
		flock -w "$wait_secs" "$VERIFY_LOCK_FD"
	else
		flock -n "$VERIFY_LOCK_FD"
	fi
	local rc=$?
	if [[ "$rc" -ne 0 ]]; then
		local holder
		holder="$(cat "$lock_path" 2>/dev/null || echo unknown)"
		echo "verify-lock: REFUSED — another verify/release window is active ($lock_path, holder: pid $holder)." >&2
		echo "  Coordinate with that session, or VERIFY_LOCK=0 to opt out of the advisory lock." >&2
		eval "exec ${VERIFY_LOCK_FD}>&-"
		return 1
	fi
	printf '%s\n' "$$" >"$lock_path"
	trap 'verify_lock_release' EXIT
	return 0
}

verify_lock_release() {
	flock -u "$VERIFY_LOCK_FD" 2>/dev/null || true
	eval "exec ${VERIFY_LOCK_FD}>&-"
}
