#!/usr/bin/env bash
# check-file-size.sh — the 350-line file-size gate, RATCHET form.
#
# Policy (2026-09-11, S18 of the publish/reset/v5-train plan): the hard
# 350-line limit stays the target, but the gate enforces a RATCHET against
# scripts/file-size-baseline.txt instead of failing on the 58 historical
# offenders (which made the gate permanently red since 2026-08-08 and thus
# invisible). The gate fails when:
#   - a production file EXCEEDS 350 lines and is NOT in the baseline, or
#   - a baselined file GROWS beyond its recorded line count.
# Shrinking a baselined file is always allowed (ratchet down); a file that
# drops to <= 350 should be removed from the baseline (`--update-baseline`).
# New files must be born under the limit.
#
# Usage:
#   bash scripts/check-file-size.sh                # gate (default)
#   bash scripts/check-file-size.sh --update-baseline  # regenerate baseline
#
# Baseline format: "lines<TAB>path", sorted by lines descending. Review it
# like a golden: a growing entry in a diff is a policy violation caught at
# review time even before the gate runs.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

BASELINE="scripts/file-size-baseline.txt"
LIMIT=350

find_go_files() {
	find . -name "*.go" -not -name "*_test.go" \
		-not -name "*.pb.go" \
		-not -name "*.gen.go" \
		-not -name "*_templ.go" \
		-not -path "*/example/*" \
		-not -path "*/testdata/*" \
		-not -path "*/internal/cattest/*" \
		-not -path "*/.git/*"
}

if [ "${1:-}" = "--update-baseline" ]; then
	{
		echo "# File-size ratchet baseline (see scripts/check-file-size.sh for the policy)."
		echo "# Remove an entry when its file drops to <= ${LIMIT} lines."
		echo "# Regenerate: bash scripts/check-file-size.sh --update-baseline"
		find_go_files | while IFS= read -r f; do
			lines=$(wc -l < "$f")
			if [ "$lines" -gt "$LIMIT" ]; then
				printf '%s\t%s\n' "$lines" "$f"
			fi
		done | sort -rn
	} >"$BASELINE"
	echo "Baseline written: $(grep -c $'\t' "$BASELINE") offender(s) > ${LIMIT} lines."
	exit 0
fi

if [ ! -f "$BASELINE" ]; then
	echo "::error::$BASELINE missing — run 'bash scripts/check-file-size.sh --update-baseline' and commit it."
	exit 1
fi

declare -A BASE
while IFS=$'\t' read -r lines path; do
	case "$lines" in ''|\#*) continue ;; esac
	BASE["$path"]="$lines"
done <"$BASELINE"

failed=false
while IFS= read -r f; do
	lines=$(wc -l < "$f")
	if [ "$lines" -le "$LIMIT" ]; then
		continue
	fi

	base="${BASE[$f]:-}"
	if [ -z "$base" ]; then
		echo "::error::$f has $lines lines (max $LIMIT) — a NEW offender; split it."
		failed=true
		continue
	fi

	if [ "$lines" -gt "$base" ]; then
		echo "::error::$f grew $base → $lines lines (ratchet: baselined files may only shrink; max $LIMIT)."
		failed=true
	fi
done < <(find_go_files)

if [ "$failed" = true ]; then
	echo "::error::File-size ratchet violations detected (see above)."
	exit 1
fi

offenders=$(grep -c $'\t' "$BASELINE" || true)
echo "✓ file-size ratchet: no new offenders, no growth ($offenders baselined historical offender(s), target ≤ $LIMIT)."
