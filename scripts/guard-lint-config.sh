#!/usr/bin/env bash
# guard-lint-config.sh — self-healing watcher body for .golangci.yml corruption.
#
# Problem (incidents 2026-08-30 .. 2026-10-14, four recurrences of the
# depguard-block deletion alone): parallel sessions run golangci-lint
# langservers whose clients apply config "fixes", rewriting .golangci.yml to
# a shape an older schema view likes (depguard.rules stripped, gci re-added).
# The auto-commit daemon absorbs the rewrite with --no-verify (hooks are dead
# on its path), so corruption reaches origin and CI reds on #check-lint-config
# within minutes, ~45 min of manual surgery per incident.
#
# This script is the tree-level healer the daemon CANNOT bypass: it runs from
# a push hook (never-blocking), a systemd timer (2-min cadence, see
# scripts/nightly/), and any interactive loop. It heals the working tree and
# re-stages the file; the daemon's next cycle commits the repair. It never
# blocks a push and never touches the golden.
#
# Heal ladder (judged ONLY by the hash tripwire, never by healer exit codes):
#   1. hash matches golden            -> healthy, exit 0
#   2. golden dirty vs HEAD           -> a re-pin is in flight, back off
#   3. shape healers (best effort)    -> check-formatters.sh, restore-depguard.sh
#   4. last-resort git restore        -> newest commit whose .golangci.yml
#                                        hashes to the golden
#   5. nothing worked                 -> loud exit 1 (human decides)
#
# Usage:
#   scripts/guard-lint-config.sh             # heal-or-report, safe any time
#   scripts/guard-lint-config.sh --self-test # fixture fault-injection suite
#
# Env hooks (self-test only):
#   GUARD_ROOT  repo root   (default: this script's repo)
#   GUARD_LOG   log file    (default: /var/tmp/cqrs-lint-config-guard.log)
#   GUARD_SKIP_HEALERS=1    skip step 3 (fixture mode: healers are root-bound)
#
# Exit: 0 healthy/healed/backed-off, 1 unhealable, 2 usage.

set -euo pipefail

ROOT="${GUARD_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
CONFIG="$ROOT/.golangci.yml"
GOLDEN="$ROOT/scripts/golangci-config-hash.golden.txt"
LOG="${GUARD_LOG:-/var/tmp/cqrs-lint-config-guard.log}"

usage() {
	sed -n '2,8p' "${BASH_SOURCE[0]}"
	exit 2
}

hash_of() {
	sha256sum "$1" | cut -d' ' -f1
}

golden_hash() {
	# sha256sum two-column format (hash + filename), same parse as the tripwire.
	cut -d' ' -f1 "$GOLDEN"
}

log_event() {
	printf '%s %s\n' "$(date -Is)" "$*" >>"$LOG" 2>/dev/null || true
}

golden_clean_vs_head() {
	# A golden with uncommitted changes means a human is mid-re-pin: the
	# hash tripwire's target itself is moving, so healing against it could
	# clobber intentional work. Back off instead.
	git -C "$ROOT" cat-file -e "HEAD:scripts/golangci-config-hash.golden.txt" 2>/dev/null ||
		return 1
	git -C "$ROOT" diff --quiet HEAD -- scripts/golangci-config-hash.golden.txt
}

last_good_config_sha() {
	# Newest commit whose .golangci.yml hashes to the CURRENT golden.
	local want have commit
	want="$(golden_hash)"
	while IFS= read -r commit; do
		have="$(git -C "$ROOT" show "$commit:.golangci.yml" 2>/dev/null | sha256sum | cut -d' ' -f1 || true)"
		if [ "$have" = "$want" ]; then
			echo "$commit"
			return 0
		fi
	done < <(git -C "$ROOT" log --format=%H -n 200 -- .golangci.yml)
	return 1
}

stage_config() {
	git -C "$ROOT" add -- .golangci.yml 2>/dev/null || true
}

heal() {
	local via=""
	if [ "${GUARD_SKIP_HEALERS:-0}" != "1" ]; then
		bash "$ROOT/scripts/check-formatters.sh" >/dev/null 2>&1 || true
		bash "$ROOT/scripts/restore-depguard.sh" >/dev/null 2>&1 || true
		if [ "$(hash_of "$CONFIG")" = "$(golden_hash)" ]; then
			via="shape-healers"
		fi
	fi
	if [ -z "$via" ]; then
		local commit
		commit="$(last_good_config_sha)" || {
			log_event "UNHEALABLE no-last-good config=$(hash_of "$CONFIG") golden=$(golden_hash)"
			echo "guard-lint-config: no committed .golangci.yml matches the golden; heal manually:" >&2
			echo "  scripts/check-golangci-hash.sh --update && git add .golangci.yml scripts/golangci-config-hash.golden.txt" >&2
			return 1
		}
		git -C "$ROOT" restore --source="$commit" -- .golangci.yml
		via="last-resort:$commit"
	fi
	if [ "$(hash_of "$CONFIG")" != "$(golden_hash)" ]; then then
		log_event "UNHEALABLE heal-miss via=$via"
		echo "guard-lint-config: healer $via produced a config that still mismatches the golden" >&2
		return 1
	fi
	stage_config
	log_event "HEALED via=$via"
	echo "guard-lint-config: healed .golangci.yml (via=$via), re-staged for the next daemon cycle"
	return 0
}

guard_main() {
	if [ ! -f "$CONFIG" ] || [ ! -f "$GOLDEN" ]; then
		return 0
	fi
	if [ "$(hash_of "$CONFIG")" = "$(golden_hash)" ]; then
		return 0
	fi
	if ! golden_clean_vs_head; then
		log_event "BACKOFF dirty-golden (re-pin in flight)"
		return 0
	fi
	heal
}

# ── self-test: fixture git repos with planted corruption shapes ─────────────
# Mutation rule: every FAIL below must fire on a broken guard (verified by
# planting a no-op heal before shipping; see gotchas entry 2026-10-14).
self_test() {
	local tmp rc=0
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' RETURN

	new_fixture() {
		# $1 = name; sets F to a git repo containing ONLY the golden.
		F="$tmp/$1"
		mkdir -p "$F/scripts"
		cp "$GOLDEN" "$F/scripts/golangci-config-hash.golden.txt"
		cp "$CONFIG" "$F/.golangci.yml.expected"
		git -C "$F" init -q
		git -C "$F" add scripts
		git -C "$F" -c user.email=t@t -c user.name=t commit -qm golden
	}

	expect() { # $1=desc $2=want_rc $3=want_log_grep(optional)
		local out want="$2" got
		out="$(GUARD_ROOT="$F" GUARD_LOG="$tmp/log" GUARD_SKIP_HEALERS=1 bash "${BASH_SOURCE[0]}")" && got=0 || got=$?
		if [ "$got" != "$want" ]; then
			echo "  ✗ FAIL: $1 (rc=$got want=$want)" >&2
			rc=1
		fi
		if [ -n "${3:-}" ] && ! grep -q "$3" "$tmp/log" 2>/dev/null; then
			echo "  ✗ FAIL: $1 (log missing '$3')" >&2
			rc=1
		fi
		rm -f "$tmp/log"
	}

	# T1: healthy tree (config matches golden, both committed) -> 0, silent.
	new_fixture t1
	cp "$F/.golangci.yml.expected" "$F/.golangci.yml"
	git -C "$F" add .golangci.yml && git -C "$F" -c user.email=t@t -c user.name=t commit -qm good-config
	expect "healthy tree exits 0" 0
	[ -z "$(git -C "$F" status --porcelain)" ] || {
		echo "  ✗ FAIL: healthy tree was modified" >&2
		rc=1
	}

	# T2: corrupted tree, golden clean -> last-resort heal + re-stage.
	new_fixture t2
	cp "$F/.golangci.yml.expected" "$F/.golangci.yml"
	git -C "$F" add .golangci.yml && git -C "$F" -c user.email=t@t -c user.name=t commit -qm good-config
	sed -i '/depguard:/,$d' "$F/.golangci.yml"
	expect "corrupt tree heals via last-resort" 0 "HEALED via=last-resort"
	cmp -s "$F/.golangci.yml" "$F/.golangci.yml.expected" || {
		echo "  ✗ FAIL: healed bytes wrong" >&2
		rc=1
	}
	git -C "$F" diff --cached --quiet -- .golangci.yml && {
		echo "  ✗ FAIL: heal not re-staged" >&2
		rc=1
	}

	# T3: golden dirty vs HEAD (re-pin in flight) -> back off, no heal.
	new_fixture t3
	cp "$F/.golangci.yml.expected" "$F/.golangci.yml"
	sed -i '/depguard:/,$d' "$F/.golangci.yml"
	echo "deadbeef" >"$F/scripts/golangci-config-hash.golden.txt"
	expect "dirty golden backs off" 0 "BACKOFF dirty-golden"
	git -C "$F" diff --quiet -- .golangci.yml 2>/dev/null || {
		echo "  ✗ FAIL: back-off still healed" >&2
		rc=1
	}

	# T4: no committed config ever matches golden -> loud 1, no heal.
	new_fixture t4
	cp "$F/.golangci.yml.expected" "$F/.golangci.yml"
	sed -i '/depguard:/,$d' "$F/.golangci.yml"
	expect "no last-good fails loudly" 1 "UNHEALABLE"

	if [ "$rc" = 0 ]; then
		echo "  ✓ guard-lint-config self-test passed"
	else
		echo "  self-test FAILED" >&2
	fi
	return $rc
}

case "${1:-}" in
"") guard_main ;;
--self-test) self_test ;;
*) usage ;;
esac
