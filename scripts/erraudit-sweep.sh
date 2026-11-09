#!/usr/bin/env bash
# Local mirror of the CI error-audit gate (ci.yml job "error-audit"):
# runs `erraudit lint ./... --enforce-go-error-family --type-aware` in every
# module and reports rc + finding counts per module. rc legend: 0 clean,
# 2 findings (gate failure), 1 analysis error, other = crash.
#
# Usage: scripts/erraudit-sweep.sh [erraudit-binary] (default: built from the
#        SAME vendored tarball CI pins — tools/erraudit/*-src.tgz; pass a path
#        to override, e.g. a dev checkout)
# Env:   GOEXPERIMENT=jsonv2 is REQUIRED (encoding/json/v2); GOWORK is forced off.
#
# Version-skew guard (2026-11-04): a NEWER erraudit on PATH started auditing
# vendored JS (catalog/v4/docserver/static) that the pinned v0.7.0 does not see —
# the mirror went red while CI stayed green. The mirror therefore builds from
# the vendored source by default so local and CI verdicts share one binary.
set -uo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
vendored_tarball=$(find "$repo_root/tools/erraudit" -maxdepth 1 -name 'erraudit-*-src.tgz' 2>/dev/null | sort | tail -1)
report_dir="$repo_root/build/erraudit-sweep"
# Extract the vendored source OUTSIDE the repo tree: a go.mod under build/
# pollutes every module-count predicate (canonical-facts 103-vs-104, sweep's
# own module total — both bitten 2026-11-04).
build_dir="${XDG_CACHE_HOME:-$HOME/.cache}/erraudit-sweep"
mkdir -p "$report_dir" "$build_dir"

if [ $# -ge 1 ]; then
	ERRAUDIT="$1"
elif [ -n "$vendored_tarball" ] && [ -x "$build_dir/erraudit" ] &&
	[ "$build_dir/erraudit" -nt "$vendored_tarball" ]; then
	ERRAUDIT="$build_dir/erraudit"
elif [ -n "$vendored_tarball" ]; then
	rm -rf "$build_dir/src"
	mkdir -p "$build_dir/src"
	tar xzf "$vendored_tarball" -C "$build_dir/src"
	(cd "$build_dir/src" &&
		env GOWORK=off GOFLAGS= go build -o "$build_dir/erraudit" ./cmd/erraudit) || {
		echo "failed to build vendored erraudit from $vendored_tarball" >&2
		exit 1
	}
	ERRAUDIT="$build_dir/erraudit"
else
	ERRAUDIT="erraudit"
fi
command -v "$ERRAUDIT" >/dev/null 2>&1 || [ -x "$ERRAUDIT" ] || {
	echo "erraudit not found: $ERRAUDIT" >&2
	exit 1
}

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
