#!/usr/bin/env bash
# pin-sweep.sh — Bump every sibling go.mod pin to the latest published tag,
# refresh the cqrs-lint goldens that pin the version set, and verify each
# changed module standalone.
#
# Encodes the same-wave rule ("bump pins → refresh cqrs-lint goldens →
# stale-pin check → standalone verify") mechanically, so the golden refresh
# cannot be forgotten — forgetting it broke verify-ci twice (2026-08-22 and
# 2026-08-29 tag waves; the rule previously lived only in prose).
#
# Usage:
#   bash scripts/pin-sweep.sh --check      # report-only; exit 1 on stale pins (CI leg)
#   bash scripts/pin-sweep.sh --dry-run    # preview the sweep: report planned bumps, change nothing
#   bash scripts/pin-sweep.sh --no-build   # sweep without per-module compile checks
#   bash scripts/pin-sweep.sh              # full sweep (default)
#
# --remote (with --check or --dry-run): compare against `git ls-remote --tags origin`
# instead of local tags — catches the "tag pushed but not fetched locally" blind spot,
# where --check stays green while every consumer pin is already stale upstream.
#
# Notes:
# - "Latest" = highest LOCAL git tag `<dir>/v4.*` (sort -V) unless --remote is given.
#   Push tags before cutting dependent tags: GOPRIVATE consumers resolve via VCS, not
#   local refs.
# - event/v4/eventtest is skipped: its v4.x tags are dead (module path lacks
#   the /vN suffix, so the module proxy rejects them — 2026-08-28 finding).
# - Per-module verification compiles test files without running tests
#   (`go test -run ZZNONE`), because test files can require sibling symbols
#   production code never imports.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

MODE=sweep

NO_BUILD=0
REMOTE=0

for arg in "$@"; do
	case "$arg" in
	--check)
		MODE=check
		;;
	--dry-run)
		MODE=dryrun
		;;
	--no-build)
		NO_BUILD=1
		;;
	--remote)
		REMOTE=1
		;;
	*)
		echo "usage: pin-sweep.sh [--check] [--dry-run] [--no-build] [--remote]" >&2
		exit 2
		;;
	esac
done

GO_TAGS=""

# latest_tag prints the highest version tag for a module dir. Source: local
# refs by default, `git ls-remote --tags origin` when REMOTE=1 (fetched ONCE
# into a cache file — one network round trip per run, not per module).
latest_tag() {
	local moddir="$1"

	if [ "$REMOTE" -eq 1 ]; then
		if [ -z "${REMOTE_TAGS_FETCHED:-}" ]; then
			REMOTE_TAGS_FILE=$(mktemp)
			git ls-remote --tags origin >"$REMOTE_TAGS_FILE"
			REMOTE_TAGS_FETCHED=1
		fi

		grep -E "refs/tags/${moddir}/v4\.[0-9]+\.[0-9]+$" "$REMOTE_TAGS_FILE" |
			sed 's|.*refs/tags/||' |
			sort -V |
			tail -1

		return 0
	fi

	git tag -l "${moddir}/v4.*" | sort -V | tail -1
}

# collect_stale emits "<dir>\t<dep>\t<ver>\t<latest>" for every sibling pin
# older than the latest tag (local refs, or origin with --remote).
collect_stale() {
	find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while read -r gomod; do
		dir="${gomod#./}"
		dir="${dir%/go.mod}"

		grep -E '^[[:space:]]*github\.com/larsartmann/go-cqrs-lite/[^ ]+ v[0-9]+\.[0-9]+\.[0-9]+$' \
			"$gomod" 2>/dev/null |
			while read -r dep ver; do
				case "$dep" in
				*/eventtest) continue ;;
				esac

				case "$ver" in
				v4.*) ;;
				*) continue ;;
				esac

				moddir="${dep#github.com/larsartmann/go-cqrs-lite/}"
				moddir="${moddir%/*}"

				latest=$(latest_tag "$moddir")

				[ -z "$latest" ] && continue

				latest="${latest#"${moddir}"/}"

				[ "$ver" = "$latest" ] && continue

				oldest=$(printf '%s\n%s\n' "$ver" "$latest" | sort -V | head -1)

				[ "$oldest" = "$latest" ] && continue

				printf '%s\t%s\t%s\t%s\n' "$dir" "$dep" "$ver" "$latest"
			done
	done

	return 0
}

# external_stale emits "<dir>\t<dep>\t<ver>\t<latest>" for cqrs-lint's pins on
# EXTERNAL larsartmann modules (go-finding): the sibling sweep above cannot see
# these, and a stale external pin silently freezes the behavior pins the
# cqrs-lint suite depends on. Latest = highest version on the module proxy.
external_stale() {
	local gomod="cmd/cqrs-lint/go.mod"
	[ -f "$gomod" ] || return 0

	local dep ver latest candidates
	while read -r dep ver; do
		[ -z "$dep" ] && continue
		candidates=$(GOWORK=off GOPROXY=https://proxy.golang.org go list -m -versions "$dep" 2>/dev/null |
			awk '{for (i = 2; i <= NF; i++) print $i}')
		latest=$(printf '%s\n' "$candidates" | sort -V | tail -1)
		[ -z "$latest" ] && continue
		oldest=$(printf '%s\n%s\n' "$ver" "$latest" | sort -V | head -1)
		[ "$oldest" = "$latest" ] && continue
		printf 'cmd/cqrs-lint\t%s\t%s\t%s\n' "$dep" "$ver" "$latest"
	done < <(grep -E '^[[:space:]]*github\.com/larsartmann/go-finding(/pipeline)? v[0-9]+\.[0-9]+\.[0-9]+$' "$gomod")
}

# sweep_dir bumps one pin and returns success when the go.mod changed.
sweep_dir() {
	local dir="$1" dep="$2" latest="$3"

	(cd "$dir" && GOWORK=off go mod edit -require="${dep}@${latest}")
}

# compile_check validates a bumped module standalone, including test files.
# go mod tidy runs first: go mod edit leaves the module "not tidy" and the
# build refuses until re-resolved.
compile_check() {
	local dir="$1"

	echo "  verifying $dir (GOWORK=off tidy + build + test-compile)"

	(cd "$dir" && GOWORK=off go mod tidy && GOWORK=off go build -tags "$GO_TAGS" ./... && GOWORK=off go test -tags "$GO_TAGS" -run ZZNONE -count=1 ./... >/dev/null)
}

# refresh_goldens regenerates both cqrs-lint goldens that pin the version set.
refresh_goldens() {
	echo "==> refreshing cqrs-lint goldens (taskmanager finding list + rule profile)"

	(cd cmd/cqrs-lint && CQRS_LINT_UPDATE_GOLDEN=1 GOWORK=off go test -tags "$GO_TAGS" -run TestLintExampleTaskmanager . >/dev/null)

	(cd cmd/cqrs-lint && CQRS_LINT_UPDATE_GOLDEN=1 GOWORK=off go test -tags "$GO_TAGS" -run TestIntegration_TaskmanagerExpectedFindings ./pkg/rules/ >/dev/null)
}

stale=$(collect_stale || true)

stale_count=$(printf '%s' "$stale" | grep -c . || true)

if [ "$MODE" = check ]; then
	external=$(external_stale || true)

	if [ "$stale_count" -gt 0 ]; then
		printf '%s\n' "$stale" | while IFS=$'\t' read -r dir dep ver latest; do
			echo "::error::stale pin: $dir requires $dep@$ver but $latest is tagged"
		done

		echo "$stale_count stale pin(s) — run scripts/pin-sweep.sh"
		exit 1
	fi

	if [ -n "$external" ]; then
		printf '%s\n' "$external" | while IFS=$'\t' read -r dir dep ver latest; do
			echo "::error::stale external pin: $dir requires $dep@$ver but $latest is published"
		done

		echo "stale external pin(s) — bump cmd/cqrs-lint's go-finding requires"
		exit 1
	fi

	echo "All sibling pins are at their latest tags (source: local refs)."

	if [ "$REMOTE" -eq 1 ]; then
		echo "All sibling pins are at their latest origin tags."
	fi

	exit 0
fi

if [ "$MODE" = dryrun ]; then
	if [ "$stale_count" -eq 0 ]; then
		echo "Dry-run: nothing to sweep, all pins current."
		exit 0
	fi

	echo "==> dry-run: would bump $stale_count stale pin(s)"
	printf '%s\n' "$stale" | while IFS=$'\t' read -r dir dep ver latest; do
		echo "  $dir: $dep $ver → $latest"
	done
	echo "==> dry-run: would standalone-verify each changed module, then refresh cqrs-lint goldens"
	echo "Dry-run complete: no changes made."

	exit 0
fi

if [ "$stale_count" -eq 0 ]; then
	echo "Nothing to sweep: all sibling pins are at their latest tags."
	exit 0
fi

echo "==> bumping $stale_count stale pin(s)"

printf '%s\n' "$stale" | while IFS=$'	' read -r dir dep ver latest; do
	echo "  $dir: $dep $ver → $latest"
	sweep_dir "$dir" "$dep" "$latest"
done

# Modules to verify: every directory the stale list names. NOT git diff —
# a sweep that restores a pin to its HEAD state produces a correct zero diff
# and would false-abort here.
mapfile -t changed < <(printf '%s\n' "$stale" | cut -f1 | sort -u)

if [ "${#changed[@]}" -eq 0 ]; then
	echo "ERROR: pins were stale but no go.mod changed — aborting"
	exit 1
fi

if [ "$NO_BUILD" -eq 0 ]; then
	for d in "${changed[@]}"; do
		compile_check "$d"
	done
fi

refresh_goldens

remaining=$(collect_stale | grep -c . || true)
if [ "$remaining" -gt 0 ]; then
	echo "ERROR: $remaining pin(s) still stale after sweep (tag missing or loop bug)"
	exit 1
fi

echo "✅ pin sweep complete: ${#changed[@]} module(s) bumped, goldens refreshed, all pins current"
