#!/usr/bin/env bash
# check-readme-deprecated.sh — READMEs must not cite deprecated symbols as
# living API without a deprecation banner (row 634e).
#
# Discovery is source-derived (zero drift): exported funcs/types whose Go doc
# comment contains "Deprecated:" are collected from every non-test .go file,
# together with the declaring package name and file path. Citations are then
# scoped so generic names (New, Load, Handler, ...) cannot cross-wire modules:
#   - qualified citation `pkg.Sym` (possibly after deeper qualifier segments,
#     e.g. `a.b.pkg.Sym`) flags when package pkg declares Sym deprecated;
#   - bare citation `Sym` flags only when a declaring file lives inside the
#     README's own directory subtree (the citation reads as local API).
# Prose word-boundary matches ("handler" in a sentence) are deliberately
# ignored — only code citations inside backticks count. A citation is
# INTENTIONAL (not flagged) when its own line or the following line carries a
# deprecation marker (Deprecated / removed in v5 / v5 removal): READMEs are
# allowed — encouraged — to name the symbol they are telling readers to stop
# using; only citations that present deprecated API as living API flag.
# A README whose first
# DEPRECATED_BANNER_LINES lines carry a deprecation banner ("> **Deprecated"
# or "Deprecated: removed in") is skipped entirely: the package-level notice
# already tells the reader, so per-symbol markers would be noise.
#
# Historical mentions are grandfathered in
# scripts/readme-deprecated-baseline.txt (sorted "file citation" lines);
# NEW mentions fail CI. Shrink the baseline by adding banners or replacing
# the citations; re-pin only via --write-baseline after an honest review.
# (Known limit: full-import-path citations like `.../stack/v4.New` are only
# matched when the path's second-to-last segment equals the package name.)
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

# Emits "pkg\tsymbol\tabsolute-file" rows for every deprecated export.
# A symbol counts as deprecated only when a doc-comment LINE starts with
# "Deprecated:" (the Go convention) — passing mid-line mentions of other
# deprecated things (e.g. "(Deprecated: removed in v5)" field notes) do not
# deprecate the symbol itself.
discover_symbols() {
	find "$src_root" -name '*.go' -not -path '*/.git/*' \
		-not -path '*/vendor/*' -not -name '*_test.go' | while read -r f; do
		awk -v abs="$f" '
			/^package / { pkg = $2 }
			/^\/\// { doc = doc "\n" $0; next }
			/^func [A-Z]/ || /^func \([^)]*\) [A-Z]/ || /^type [A-Z]/ {
				if (doc ~ /\n\/\/ Deprecated:/) {
					name = $0
					sub(/^func \([^)]*\) /, "", name)
					sub(/^func /, "", name)
					sub(/^type /, "", name)
					sub(/[\[ (].*$/, "", name)
					if (name != "") print pkg "\t" name "\t" abs
				}
			}
			{ doc = "" }
		' "$f"
	done | sort -u
}

scan_readmes() {
	# $1 = path to the "pkg\tsymbol\tfile" table produced by discover_symbols
	local table="$1"
	[ -s "$table" ] || {
		echo "check-readme-deprecated: no deprecated symbols discovered" >&2
		return 0
	}

	find "$readme_root" -name README.md -not -path '*/.git/*' \
		-not -path '*/vendor/*' -not -path '*/docs/*' 2>/dev/null | sort | while read -r f; do
		# Bannered READMEs already warn the reader at the top.
		if head -n "$banner_lines" "$f" | grep -qE '^> ?\*\*Deprecated|Deprecated: removed in'; then
			continue
		fi
		dir="$(cd "$(dirname "$f")" && pwd)"
		awk -v ROWS="$table" -v DIR="$dir" -v FILE="$f" '
			BEGIN {
				while ((getline row < ROWS) > 0) {
					n = split(row, r, "\t")
					qual[r[1] "." r[2]] = 1
					decl[r[2]] = decl[r[2]] "\n" r[3]
				}
			}
			{
				L[FNR] = $0
				if ($0 ~ /[Dd]eprecated|removed in v5|v5 removal/) M[FNR] = 1
				n = FNR
			}
			END {
				for (i = 1; i <= n; i++) {
					# Deprecation-framed citations (marker on this line or an
					# adjacent wrapped line) are intentional disclosure.
					if (M[i - 1] || M[i] || M[i + 1]) continue
					line = L[i]
					while (match(line, /`[^`]+`/)) {
						span = substr(line, RSTART + 1, RLENGTH - 2)
						line = substr(line, RSTART + RLENGTH)
						cand = span
						sub(/[[(].*$/, "", cand)
						sub(/^[^A-Za-z0-9_]*/, "", cand)
						m = split(cand, seg, ".")
						if (m >= 2) {
							s = seg[m]; q = seg[m - 1]
							if (qual[q "." s]) print FILE " " cand
						} else if (m == 1 && decl[cand] != "") {
							n2 = split(decl[cand], paths, "\n")
							for (k = 1; k <= n2; k++) {
								if (paths[k] != "" && index(paths[k], DIR "/") == 1) {
									print FILE " " cand
									break
								}
							}
						}
					}
				}
			}
		' "$f"
	done
}

self_test() {
	local fixture
	fixture="$(mktemp -d /tmp/readme-deprecated-fixture.XXXXXX)" || return 1
	trap 'rm -rf "$fixture"' RETURN

	mkdir -p "$fixture/mod/lib" "$fixture/mod/app" "$fixture/mod/clean" "$fixture/mod/bannered" \
		"$fixture/mod/legacy" "$fixture/mod/wrapped"
	cat >"$fixture/mod/lib/api.go" <<'EOF'
package lib

// Old is going away.
//
// Deprecated: removed in v5; use Fresh.
func Old() {}

// Fresh is the replacement.
func Fresh() {}

// Fresh2 maps legacy fields (CausationID is Deprecated: removed in v5) but
// is itself current — a mid-line mention must NOT mark the symbol deprecated.
func Fresh2() {}
EOF
	# Bare citation inside the declaring subtree: MUST flag.
	printf '# lib readme\nUse `Old` today.\n' >"$fixture/mod/lib/README.md"
	# Bare citation outside the declaring subtree: must NOT flag (scoping);
	# qualified citation of the same symbol: MUST flag.
	printf '# app readme\nUse `Old` freely, or `lib.Old`.\n' >"$fixture/mod/app/README.md"
	printf '# clean readme\nUse `Fresh` or `lib.Fresh2` only.\n' >"$fixture/mod/clean/README.md"
	printf '# bannered readme\n\n> **Deprecated:** removed in v5. Use system.New.\n\nUse `lib.Old` freely.\n' >"$fixture/mod/bannered/README.md"
	# Deprecation-framed citations are intentional disclosure, not drift:
	# same-line marker and wrapped next-line marker must both be exempt.
	printf '# legacy readme\nUse `lib.Old` while migrating (Deprecated: removed in v5).\n' >"$fixture/mod/legacy/README.md"
	printf '# wrapped readme\nThe symbols\n`lib.Old`\nare deprecated and will be removed in v5.\nAlso deprecated (removed in v5):\n`lib.Old` again.\n' >"$fixture/mod/wrapped/README.md"

	# Run the engine once; capture output so grep -q early-exit cannot
	# SIGPIPE the child under `set -o pipefail` and masquerade as a failure.
	local out
	out="$(DEP_SRC_ROOT="$fixture/mod" DEP_README_ROOT="$fixture/mod" \
		DEP_BASELINE="$fixture/empty-baseline.txt" \
		bash "$0" 2>/dev/null)" || true

	# Leg 1 (non-vacuous detection): bare citation of a deprecated symbol
	# inside the declaring subtree MUST be flagged.
	if ! grep -q 'lib/README.md Old' <<<"$out"; then
		echo "self-test FAILED: in-subtree bare citation not flagged" >&2
		return 1
	fi
	# Leg 2 (scoping): the same bare citation OUTSIDE the declaring subtree
	# must not be flagged — generic names cannot cross-wire modules.
	if grep -qE 'app/README\.md Old( |$)' <<<"$out"; then
		echo "self-test FAILED: out-of-subtree bare citation flagged" >&2
		return 1
	fi
	# Leg 3 (qualified detection): `pkg.Sym` citations flag via package match.
	if ! grep -q 'app/README.md lib.Old' <<<"$out"; then
		echo "self-test FAILED: qualified citation not flagged" >&2
		return 1
	fi
	# Leg 4 (no false positive): clean README must not be flagged.
	if grep -q 'clean/README' <<<"$out"; then
		echo "self-test FAILED: clean README flagged" >&2
		return 1
	fi
	# Leg 5 (banner escape hatch): bannered README must be skipped.
	if grep -q 'bannered/README' <<<"$out"; then
		echo "self-test FAILED: bannered README flagged" >&2
		return 1
	fi
	# Leg 6 (deprecation framing): a citation whose line says it is deprecated
	# is intentional disclosure, not a living-API citation.
	if grep -q 'legacy/README' <<<"$out"; then
		echo "self-test FAILED: deprecation-framed citation flagged" >&2
		return 1
	fi
	# Leg 7 (wrapped framing): marker on the following (wrapped) line exempts too.
	if grep -q 'wrapped/README' <<<"$out"; then
		echo "self-test FAILED: wrapped deprecation-framed citation flagged" >&2
		return 1
	fi

	echo "check-readme-deprecated self-test: all seven legs green (detect + scope + qualified + clean + banner + framed + wrapped)"
	return 0
}

[ "${1:-}" = "--self-test" ] && {
	self_test
	exit $?
}

table="$(mktemp /tmp/readme-deprecated-table.XXXXXX)" || exit 1
trap 'rm -f "$table"' EXIT
discover_symbols >"$table"
findings="$(scan_readmes "$table" | sort -u)"

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
