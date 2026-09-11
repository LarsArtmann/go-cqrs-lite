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

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== Turso Version Citation Check ==="

CANON=$(sed -n 's/^[[:space:]]*TursoGoIVMVerifiedThrough[[:space:]]*= *"\(.*\)"/\1/p' metaengine/materialized_view_versions.go | head -1)
if [ -z "$CANON" ]; then
	echo "FAIL: cannot read TursoGoIVMVerifiedThrough from metaengine/materialized_view_versions.go"
	exit 1
fi

RANGE_PAT='(<=|≤|through) v[0-9]+\.[0-9]+\.[0-9]+(-pre\.[0-9]+)?'

# Live citation sites: docs and code an operator or release check reads TODAY.
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

status=0
for f in "${LIVE_FILES[@]}"; do
	if [ ! -f "$f" ]; then
		echo "FAIL: live-citation file missing: $f (update scripts/check-turso-version.sh)"
		status=1
		continue
	fi

	while IFS= read -r hit; do
		[ -n "$hit" ] || continue
		line=${hit%%:*}
		content=${hit#*:}
		while IFS= read -r m; do
			[ -n "$m" ] || continue
			cited=${m##* }
			if [ "$cited" != "$CANON" ]; then
				echo "FAIL: $f:$line cites '$cited' but canonical TursoGoIVMVerifiedThrough is '$CANON'"
				status=1
			fi
		done < <(printf '%s\n' "$content" | grep -oE "$RANGE_PAT" || true)
	done < <(grep -nE "$RANGE_PAT" "$f" || true)
done

if [ "$status" -ne 0 ]; then
	echo ""
	echo "Fix: re-verify live (metaengine/tursoengine suite behind -tags ivmrepro), then"
	echo "bump TursoGoIVMVerifiedThrough in metaengine/materialized_view_versions.go and"
	echo "update the files above in the same change — see docs/turso-go-ivm-fix-flip-runbook.md."
	exit 1
fi

echo "All live turso-go IVM citations match TursoGoIVMVerifiedThrough=$CANON."
exit 0
