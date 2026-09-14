#!/usr/bin/env bash
#
# check-turso-version.sh — every LIVE turso-go IVM version citation must match
# the canonical constant metaengine.TursoGoIVMVerifiedThrough.
#
# The grouped-materialized-view caveat (ADR-0135) cites an upstream-verified
# driver version in many places (Doctor WARN, guard pin, gotchas, skill
# references, ADR, FEATURES, TODO_LIST). Without this gate every
# re-verification was a multi-site whack-a-mole. The canonical value lives
# ONLY in metaengine/materialized_view_versions.go; this script fails when a
# live citation names a different version (stale OR ahead-of-verified).
#
# Point-in-time records (CHANGELOG history, docs/research drafts,
# docs/status reports) are intentionally NOT gated.
#
# Exit 0 = all live citations match, exit 1 = drift found.
#
# --self-test: mutation-proves the scanner against a synthetic tree (a clean
# citation passes, a planted stale citation is caught) without touching the
# repo's real files.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

RANGE_PAT='(<=|≤|through) v[0-9]+\.[0-9]+\.[0-9]+(-pre\.[0-9]+)?'

read_canonical() {
	local root="$1"
	sed -n 's/^[[:space:]]*TursoGoIVMVerifiedThrough[[:space:]]*= *"\(.*\)"/\1/p' \
		"$root/metaengine/materialized_view_versions.go" | head -1
}

# scan_citations <root> <live-file...>: echoes FAIL lines for every citation
# that drifts from the root's canonical constant; returns 1 when any drift.
scan_citations() {
	local root="$1"
	shift

	local canon
	canon="$(read_canonical "$root")"
	if [ -z "$canon" ]; then
		echo "FAIL: cannot read TursoGoIVMVerifiedThrough from $root/metaengine/materialized_view_versions.go"
		return 1
	fi

	local status=0
	local f
	for f in "$@"; do
		if [ ! -f "$root/$f" ]; then
			echo "FAIL: live-citation file missing: $f (update scripts/check-turso-version.sh)"
			status=1
			continue
		fi

		local hit line content m cited
		while IFS= read -r hit; do
			[ -n "$hit" ] || continue
			line=${hit%%:*}
			content=${hit#*:}
			while IFS= read -r m; do
				[ -n "$m" ] || continue
				cited=${m##* }
				if [ "$cited" != "$canon" ]; then
					echo "FAIL: $f:$line cites '$cited' but canonical TursoGoIVMVerifiedThrough is '$canon'"
					status=1
				fi
			done < <(printf '%s\n' "$content" | grep -oE "$RANGE_PAT" || true)
		done < <(grep -nE "$RANGE_PAT" "$root/$f" || true)
	done

	return "$status"
}

LIVE_FILES=(
	metaengine/materialized_view_doctor_test.go
	metaengine/tursoengine/matview_property_test.go
	metaengine/tursoengine/matview_bench_test.go
	metaengine/tursoengine/ivm_repro_test.go
	metaengine/tursoengine/README.md
	FEATURES.md
	TODO_LIST.md
	docs/agents/gotchas-tooling-build.md
	docs/adr/0135-materialized-views-operator-option.md
	docs/benchmarks/2026-09-07_turso-materialized-views.md
	.agents/skills/go-cqrs-lite/references/readmodels.md
	.agents/skills/go-cqrs-lite/references/recipes.md
)

self_test() {
	local tmp
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' RETURN

	mkdir -p "$tmp/metaengine" "$tmp/docs"
	printf 'package metaengine\n\nconst (\n\tTursoGoIVMVerifiedThrough = "v0.7.2-pre.10"\n)\n' \
		>"$tmp/metaengine/materialized_view_versions.go"
	printf 'The caveat holds through v0.7.2-pre.10.\n' >"$tmp/docs/clean.md"
	printf 'Older doc says the caveat holds through v0.7.2-pre.8.\n' >"$tmp/docs/stale.md"

	if scan_citations "$tmp" docs/clean.md >/dev/null 2>&1; then
		echo "  ✓ PASS: clean citation accepted"
	else
		echo "  ✗ FAIL: clean citation was rejected"
		return 1
	fi

	local out
	out="$(scan_citations "$tmp" docs/stale.md 2>&1)" && true
	if printf '%s' "$out" | grep -q "v0.7.2-pre.8"; then
		echo "  ✓ PASS: planted stale citation caught"
	else
		echo "  ✗ FAIL: stale citation NOT caught (out: $out)"
		return 1
	fi

	return 0
}

if [ "${1:-}" = "--self-test" ]; then
	if self_test; then
		echo "check-turso-version self-test passed."
		exit 0
	fi
	echo "check-turso-version self-test FAILED."
	exit 1
fi

cd "$ROOT"

echo "=== Turso Version Citation Check ==="

if ! scan_citations "$ROOT" "${LIVE_FILES[@]}"; then
	echo ""
	echo "Fix: re-verify live (metaengine/tursoengine suite behind -tags ivmrepro), then"
	echo "bump TursoGoIVMVerifiedThrough in metaengine/materialized_view_versions.go and"
	echo "update the files above in the same change — see docs/turso-go-ivm-fix-flip-runbook.md."
	exit 1
fi

echo "All live turso-go IVM citations match TursoGoIVMVerifiedThrough=$(read_canonical "$ROOT")."
exit 0
