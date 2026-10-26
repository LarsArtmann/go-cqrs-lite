#!/usr/bin/env bash
# fuzz-rotation.sh — every fuzz target, on a schedule, not just the four the
# nightly happened to name.
#
# Why: 115 fuzz targets exist (2026-10-26 census) and exactly 4 ran nightly.
# Fuzzers only pay off when they run, and a target that never runs is a
# comment with a function signature. This script keeps the full inventory in
# scripts/fuzz-targets.txt and rotates it deterministically by day, so every
# target is exercised within SHARDS nights without a 2-hour job.
#
# Usage:
#   bash scripts/fuzz-rotation.sh --list                 # inventory + shard
#   bash scripts/fuzz-rotation.sh --update-targets       # regenerate inventory
#   bash scripts/fuzz-rotation.sh --shard 1/4 [--seconds 60]
#   bash scripts/fuzz-rotation.sh --all [--seconds 10]   # everything, once
#   bash scripts/fuzz-rotation.sh --self-test
#
# Environment:
#   FUZZ_SECONDS      per-target -fuzztime (default 60)
#   FUZZ_TARGETS_FILE inventory path (default scripts/fuzz-targets.txt)
#   FUZZ_SHARD_OVERRIDE  force a shard index (1-based) for a manual rerun
#
# Rotation: target i (0-based, sorted inventory) belongs to shard (i % SHARDS);
# the nightly runs shard (day-of-year % SHARDS) + 1. Every target therefore runs
# at least once per SHARDS nights, and a manual `--shard N/4` replays one.
#
# Corpora: go stores new inputs under <package>/testdata/fuzz/<Target>/, which
# the workflow caches between runs — a crash found on night N becomes a
# regression case on night N+1 without anyone copying files.

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

TARGETS="${FUZZ_TARGETS_FILE:-scripts/fuzz-targets.txt}"
SECONDS_PER_TARGET="${FUZZ_SECONDS:-60}"
SHARD=""
SHARDS=4
MODE=run

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

# build_targets writes "<module>\t<package-rel>\t<Target>" for every
# `func Fuzz*(f *testing.F)` in the tree. Module attribution is the longest
# testModules prefix so nested modules never absorb their parent's targets.
build_targets() { # $1=out
	local mods
	mods="$(sed -n '/testModules = \[/,/^[[:space:]]*\];/p' flake.nix |
		grep -oE '"[^"]+"' | tr -d '"' | awk '{ print length, $0 }' |
		sort -rn | cut -d' ' -f2- | tr '\n' ' ')"
	[ -n "$mods" ] || die "could not read testModules from flake.nix"

	grep -rn --include='*_test.go' '^func Fuzz' . |
		grep -v '^./vendor/' | grep -v '/testdata/' |
		sed 's|^\./||; s|:func \([A-Za-z0-9_]*\)(.*|\t\1|' |
		awk -F'\t' -v mods="$mods" '
			{
				file = $1; target = $2
				dir = file; sub(/\/[^\/]*$/, "", dir)
				n = split(mods, m, " "); mod = ""
				for (i = 1; i <= n; i++)
					if (dir == m[i] || index(dir, m[i] "/") == 1) { mod = m[i]; break }
				if (mod == "") next
				rel = dir
				if (rel == mod) rel = "."; else sub("^" mod "/", "", rel)
				printf "%s\t%s\t%s\n", mod, rel, target
			}' | sort -t$'\t' -k1,1 -k2,2 -k3,3 >"$1"
}

list_targets() { # $1=file -> one line per target with its shard
	local i=0 mod pkg target
	while IFS=$'\t' read -r mod pkg target; do
		printf '%-24s %-28s %-55s shard=%d\n' "$mod" "$pkg" "$target" $((i % SHARDS + 1))
		i=$((i + 1))
	done <"$1"
}

run_target() { # $1=module $2=package-rel $3=target
	local mod="$1" pkg="$2" target="$3"
	local pattern="./${pkg}/"
	[ "$pkg" = "." ] && pattern="."
	printf '==> %s %s %s (%ss)\n' "$mod" "$pkg" "$target" "$SECONDS_PER_TARGET"
	if (cd "$mod" && GOWORK=off GOFLAGS= GOTOOLCHAIN=auto go test \
		-run "^${target}$" -fuzz "^${target}$" \
		-fuzztime "${SECONDS_PER_TARGET}s" "$pattern"); then
		return 0
	fi
	return 1
}

self_test() {
	local tmp failures=0 out
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	printf 'm1\t.\tFuzzA\nm1\t.\tFuzzB\nm1\tsub\tFuzzC\nm2\t.\tFuzzD\n' >"$tmp/targets.txt"

	out="$(list_targets "$tmp/targets.txt")"
	if [ "$(printf '%s\n' "$out" | grep -c 'shard=')" -ne 4 ]; then
		echo "self-test FAIL: every target must get a shard" >&2
		failures=$((failures + 1))
	fi
	# Deterministic: the same inventory always yields the same assignment.
	if [ "$out" != "$(list_targets "$tmp/targets.txt")" ]; then
		echo "self-test FAIL: shard assignment must be deterministic" >&2
		failures=$((failures + 1))
	fi
	# Every shard is covered exactly once across the inventory.
	inventory_total="$(grep -c . "$tmp/targets.txt")"
	assigned=0
	for s in 1 2 3 4; do
		k="$(printf '%s\n' "$out" | grep -c "shard=$s")"
		assigned=$((assigned + k))
	done
	if [ "$assigned" -ne "$inventory_total" ]; then
		echo "self-test FAIL: shards must partition the inventory ($assigned != $inventory_total)" >&2
		failures=$((failures + 1))
	fi
	# Round-robin balances within one of each other.
	declare -A counts=()
	for s in 1 2 3 4; do
		counts[$s]="$(printf '%s\n' "$out" | grep -c "shard=$s")"
	done
	min=99
	max=0
	for s in 1 2 3 4; do
		[ "${counts[$s]}" -lt "$min" ] && min="${counts[$s]}"
		[ "${counts[$s]}" -gt "$max" ] && max="${counts[$s]}"
	done
	if [ $((max - min)) -gt 1 ]; then
		echo "self-test FAIL: round-robin shards must differ by at most one ($min..$max)" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::fuzz-rotation self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ fuzz-rotation self-test: 4/4 sharding cases behave"
}

while [ $# -gt 0 ]; do
	case "$1" in
	--list) MODE=list ;;
	--update-targets) MODE=update ;;
	--self-test) MODE=selftest ;;
	--all) SHARD="all" ;;
	--shard)
		shift
		[ $# -gt 0 ] || die "--shard requires N/M"
		SHARD="$1"
		;;
	--seconds)
		shift
		[ $# -gt 0 ] || die "--seconds requires a value"
		SECONDS_PER_TARGET="$1"
		;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

[ "$MODE" = "selftest" ] && {
	self_test
	exit 0
}

if [ "$MODE" = "update" ]; then
	build_targets "$TARGETS"
	printf '✓ Wrote %s fuzz targets to %s\n' "$(grep -c . "$TARGETS")" "$TARGETS"
	exit 0
fi

[ -f "$TARGETS" ] || die "$TARGETS is missing — run: bash scripts/fuzz-rotation.sh --update-targets"

if [ "$MODE" = "list" ]; then
	list_targets "$TARGETS"
	exit 0
fi

# Resolve the shard: explicit N/M, --all, or today's rotation slot.
if [ -z "$SHARD" ]; then
	doy="$(date -u +%j)"
	doy=$((10#$doy))
	SHARD="$((doy % SHARDS + 1))/$SHARDS"
fi

selected=()
if [ "$SHARD" = "all" ]; then
	while IFS= read -r line; do selected+=("$line"); done <"$TARGETS"
	printf '==> fuzz rotation: ALL %s targets, %ss each\n' "${#selected[@]}" "$SECONDS_PER_TARGET"
else
	n="${SHARD%%/*}"
	total="${SHARD##*/}"
	[ "$n" -ge 1 ] && [ "$n" -le "$total" ] || die "--shard must be N/M with 1 <= N <= M"
	i=0
	while IFS= read -r line; do
		if [ $((i % total + 1)) -eq "$n" ]; then selected+=("$line"); fi
		i=$((i + 1))
	done <"$TARGETS"
	printf '==> fuzz rotation: shard %s/%s, %s targets, %ss each\n' "$n" "$total" "${#selected[@]}" "$SECONDS_PER_TARGET"
fi

[ "${#selected[@]}" -gt 0 ] || die "shard is empty — check $TARGETS"

failed=0
for line in "${selected[@]}"; do
	IFS=$'\t' read -r mod pkg target <<<"$line"
	if ! run_target "$mod" "$pkg" "$target"; then
		echo "::error::fuzz target $target ($mod/$pkg) found a failure"
		failed=$((failed + 1))
	fi
done

if [ "$failed" -gt 0 ]; then
	echo "::error::$failed target(s) failed — corpus under $TARGETS's packages (testdata/fuzz/)"
	exit 1
fi
echo "✓ fuzz rotation: ${#selected[@]} target(s) clean"
