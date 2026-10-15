#!/usr/bin/env bash
# Local mirror of the CI error-audit gate (ci.yml job "error-audit"):
# runs `erraudit lint ./... --enforce-go-error-family --type-aware` in every
# module and reports rc + finding counts per module. rc legend: 0 clean,
# 2 findings (gate failure), 1 analysis error, other = crash.
#
# Usage: scripts/erraudit-sweep.sh [erraudit-binary] (default: erraudit on PATH)
# Env:   GOEXPERIMENT=jsonv2 is REQUIRED (encoding/json/v2); GOWORK is forced off.
set -uo pipefail

ERRAUDIT="${1:-erraudit}"
command -v "$ERRAUDIT" >/dev/null 2>&1 || { echo "erraudit not found: $ERRAUDIT" >&2; exit 1; }

repo_root=$(cd "$(dirname "$0")/.." && pwd)
report_dir="$repo_root/build/erraudit-sweep"
mkdir -p "$report_dir"

fail=0
total=0
while IFS= read -r mod_dir; do
	total=$((total + 1))
	name=${mod_dir#"$repo_root"/}
	out_file="$report_dir/$(echo "$name" | tr '/' '_').log"
	rc=$(
		cd "$mod_dir" &&
			env GOWORK=off GOEXPERIMENT=jsonv2 GOFLAGS= "$ERRAUDIT" lint ./... \
				--enforce-go-error-family --type-aware --disable-extensions \
				>"$out_file" 2>&1
		echo $?
	)
	case "$rc" in
	0) verdict=CLEAN ;;
	2) verdict=FINDINGS ;;
	*) verdict="RC=$rc" ;;
	esac
	printf '%-8s %s\n' "$verdict" "$name"
	[ "$rc" -ne 0 ] && fail=$((fail + 1))
done < <(find "$repo_root" -name go.mod -not -path '*/vendor/*' -not -path '*/.git/*' -exec dirname {} \; | sort)

echo "---"
echo "modules=$total failing=$fail (logs in build/erraudit-sweep/)"
[ "$fail" -eq 0 ] || exit 2
