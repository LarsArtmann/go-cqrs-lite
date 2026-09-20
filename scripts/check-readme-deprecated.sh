#!/usr/bin/env bash
# check-readme-deprecated.sh — READMEs must not cite deprecated symbols as
# living API without a deprecation banner (row 634e).
#
# Discovery is source-derived (zero drift): exported funcs/types whose Go doc
# comment contains "Deprecated:" are collected from every non-test .go file.
# A module README is then flagged when it cites one of those symbols inside
# backticks (prose word-boundary matches like "handler" or "execute" are
# deliberately ignored — only code citations count). A README whose first
# DEPRECATED_BANNER_LINES lines carry a deprecation banner ("> **Deprecated"
# or "Deprecated: removed in") is skipped entirely: the package-level notice
# already tells the reader, so per-symbol markers would be noise.
#
# Historical mentions are grandfathered in
# scripts/readme-deprecated-baseline.txt (sorted "file symbol" lines);
# NEW mentions fail CI. Shrink the baseline by adding banners or replacing
# the citations; re-pin only via --write-baseline after an honest review.
#
# Usage:
#   scripts/check-readme-deprecated.sh [--self-test] [--write-baseline]
#
# Env overrides (planted-fixture pattern; honored by --self-test so it never
# touches tracked files):
#   DEP_SRC_ROOT     source root to scan for deprecated symbols
#   DEP_README_ROOT  root to scan for README.md files
#   DEP_BASELINE     baseline file (default scripts/readme-deprecated-baseline.txt)
set -uo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
	echo "check-readme-deprecated.sh: must run inside the git repo" >&2
	exit 1
}
cd "$repo_root" || exit 1

src_root="${DEP_SRC_ROOT:-$repo_root}"
readme_root="${DEP_README_ROOT:-$repo_root}"
baseline="${DEP_BASELINE:-$repo_root/scripts/readme-deprecated-baseline.txt}"
banner_lines="${DEPRECATED_BANNER_LINES:-12}"

discover_symbols() {
	find "$src_root" -name '*.go' -not -path '*/.git/*' \
		-not -path '*/vendor/*' -not -name '*_test.go' | while read -r f; do
		awk '
			/^\/\// { doc = doc "\n" $0; next }
			/^func [A-Z]/ || /^func \([^)]*\) [A-Z]/ || /^type [A-Z]/ {
				if (doc ~ /Deprecated:/) {
					name = $0
					sub(/^func \([^)]*\) /, "", name)
					sub(/^func /, "", name)
					sub(/^type /, "", name)
					sub(/[\[ (].*$/, "", name)
					if (name != "") print name
				}
			}
			{ doc = "" }
		' "$f"
	done | sort -u
}

scan_readmes() {
	# $1 = newline-separated deprecated symbol list
	local symbols="$1"
	[ -n "$symbols" ] || {
		echo "check-readme-deprecated: no deprecated symbols discovered" >&2
		return 0
	}

	find "$readme_root" -name README.md -not -path '*/.git/*' \
		-not -path '*/vendor/*' -not -path '*/docs/*' 2>/dev/null | sort | while read -r f; do
		# Bannered READMEs already warn the reader at the top.
		if head -n "$banner_lines" "$f" | grep -qE '^> ?\*\*Deprecated|Deprecated: removed in'; then
			continue
		fi
		for sym in $symbols; do
			# Cite-in-backticks only: `sym` possibly qualified (pkg.sym).
			if grep -qE "\`[^\`]*\b${sym}\b[^\`]*\`" "$f"; then
				echo "$f $sym"
			fi
		done
	done
}

self_test() {
	local fixture
	fixture="$(mktemp -d /tmp/readme-deprecated-fixture.XXXXXX)" || return 1
	trap 'rm -rf "$fixture"' RETURN

	mkdir -p "$fixture/mod"
	cat >"$fixture/mod/api.go" <<'EOF'
package mod

// Old is going away.
//
// Deprecated: removed in v5; use New.
func Old() {}

// New is the replacement.
func New() {}
EOF
	printf '# clean readme\nUse `New` only.\n' >"$fixture/mod/README-clean.md"
	printf '# dirty readme\nUse `Old` today.\n' >"$fixture/mod/README-dirty.md"
	printf '# bannered readme\n\n> **Deprecated:** removed in v5. Use system.New.\n\nUse `Old` freely.\n' >"$fixture/mod/README-bannered.md"

	# Leg 1 (non-vacuous detection): backticked citation of a deprecated
	# symbol in an un-bannered README MUST be flagged.
	if ! DEP_SRC_ROOT="$fixture/mod" DEP_README_ROOT="$fixture/mod" \
		DEP_BASELINE="$fixture/empty-baseline.txt" \
		bash "$0" 2>/dev/null | grep -q 'README-dirty.md Old'; then
		echo "self-test FAILED: dirty README not flagged" >&2
		return 1
	fi
	# Leg 2 (no false positive): clean README must not be flagged.
	if DEP_SRC_ROOT="$fixture/mod" DEP_README_ROOT="$fixture/mod" \
		DEP_BASELINE="$fixture/empty-baseline.txt" \
		bash "$0" 2>/dev/null | grep -q 'README-clean'; then
		echo "self-test FAILED: clean README flagged" >&2
		return 1
	fi
	# Leg 3 (banner escape hatch): bannered README must be skipped.
	if DEP_SRC_ROOT="$fixture/mod" DEP_README_ROOT="$fixture/mod" \
		DEP_BASELINE="$fixture/empty-baseline.txt" \
		bash "$0" 2>/dev/null | grep -q 'README-bannered'; then
		echo "self-test FAILED: bannered README flagged" >&2
		return 1
	fi

	echo "check-readme-deprecated self-test: all three legs green (detect + clean + banner)"
	return 0
}

[ "${1:-}" = "--self-test" ] && {
	self_test
	exit $?
}

symbols="$(discover_symbols)"
findings="$(scan_readmes "$symbols" | sort)"

if [ "${1:-}" = "--write-baseline" ]; then
	printf '%s\n' "$findings" | sed '/^$/d' >"$baseline"
	echo "baseline written: $(grep -c . "$baseline") entries in $baseline"
	exit 0
fi

fail=0
new=0
while IFS= read -r line; do
	[ -n "$line" ] || continue
	if grep -qxF "$line" "$baseline" 2>/dev/null; then
		continue
	fi
	echo "NEW: $line (deprecated symbol cited as living API; add a banner or migrate the citation)"
	new=$((new + 1))
done <<<"$findings"

baselined=0
[ -f "$baseline" ] && baselined=$(grep -c . "$baseline" || true)

if [ "$new" -gt 0 ]; then
	echo "check-readme-deprecated: $new NEW deprecated-symbol citations ($baselined grandfathered in baseline)"
	fail=1
else
	echo "check-readme-deprecated: clean ($baselined grandfathered, 0 new)"
fi

exit "$fail"
