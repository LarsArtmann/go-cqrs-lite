#!/usr/bin/env bash
# Error-taxonomy drift gate: every errorfamily code minted by the gated
# modules must be documented in docs/error-taxonomy.md with the family the
# source actually constructs, and every documented code must still exist.
# The watermill table "lied for weeks" (parse failures labeled Corruption;
# actually Rejection) because the tables were hand-maintained — this gate
# extracts the ground truth from errorfamily.(New|Wrap)<Family>("code", …)
# call sites and diffs it against the doc in both directions (the
# check-linter-names.sh pattern).
#
# Doc conventions enforced:
#   - one family per table row; the Family column is the exact family name
#   - the Code column holds backticked literals (`graph.read.path_not_found`)
#     or trailing-star wildcards (`graph.schema.*`, `middleware.cb_invalid_*`)
#   - wildcards need a non-empty stem (bare `module.*` is rejected: it would
#     hide family drift behind one pattern)
#
# Extend the gate: add "section|dir|prefix1 prefix2 …" to GATED_MODULES.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
doc="$repo_root/docs/error-taxonomy.md"

# doc-section-title | module dir (relative to repo root) | code prefixes
GATED_MODULES=(
	"middleware|middleware|middleware. deadletter."
	"graph|graph|graph."
	"storage/relational|storage/relational|relational."
	"projectionhost|projectionhost|projectionhost."
	"transport/grpc|transport/grpc|grpc."
	"claiming|claiming|claiming."
	"watermill|watermill|watermill."
	"storage/pebble|storage/pebble|pebble."
	"core/event|event|event."
	"core/command|command|command."
	"core/query|query|query."
	"storage/view|storage/view|storage.view.|10"
	"stack|stack|stack. bbolt_preset. duckdb. duckdb_preset. memory. mysql. mysql_preset. pebble_preset. postgres. postgres_preset. sqlite. sqlite_preset. turso. turso_preset.|50"
	"deriver|deriver|deriver.|2"
	"storage (SQL facade)|storage|storage. backend. listing.|30|1"
	"storage (SQL facade)|storage/sql|storage. sql.|24"
	"storage (SQL facade)|storage/eventstore|storage.|22"
	"storage (SQL facade)|storage/readmodel|storage. kv_sql.|6"
	"metaengine|metaengine|metaengine."
)

# Per-module pool-size floor: a gated module whose extraction yields fewer
# than this many distinct codes means the extraction pattern broke (or the
# module vanished) — fail loudly instead of silently diffing an empty pool.
# Optional 4th field, default 1.
DEFAULT_FLOOR=1

EXTRACT_PATTERN='errorfamily\.(?:New|Wrap)(Rejection|Conflict|Transient|Infrastructure|Corruption|Orchestration)\(\s*(?:[^,"]*,\s*)?"([a-z0-9_.]+)"'

# --self-test: run the extraction against the planted mutation fixture
# (scripts/testdata/error-taxonomy-selftest/) and assert the scanner sees
# what it must see — and nothing more. Makes the 2026-09-11 hand-planted
# mutation proof a permanent CI fact instead of a dying session anecdote.
if [ "${1:-}" = "--self-test" ]; then
	fixture="$repo_root/scripts/testdata/error-taxonomy-selftest"
	tmp="$(mktemp)"
	trap 'rm -f "$tmp"' EXIT

	rg -U --no-filename -o "$EXTRACT_PATTERN" "$fixture" -r '$2	$1' >"$tmp"
	sort -u "$tmp" -o "$tmp"

	fail=0
	for want in \
		$'selftest.simple\tRejection' \
		$'selftest.multiline\tInfrastructure' \
		$'selftest.wrap\tCorruption'; do
		grep -qxF "$want" "$tmp" || {
			echo "SELF-TEST FAIL: planted code not extracted: $want" >&2
			fail=1
		}
	done

	for banned in "limit" "never.extract" ".suffix" "selftest.suffix"; do
		grep -q "^${banned}	" "$tmp" && {
			echo "SELF-TEST FAIL: junk captured as a code: $banned" >&2
			fail=1
		}
	done

	if [ "$fail" -eq 0 ]; then
		echo "✓ error-taxonomy self-test: scanner extraction matches the planted mutation fixture"
	fi
	exit "$fail"
fi

status=0
tmp_pool="$(mktemp)"
tmp_claims="$(mktemp)"
trap 'rm -f "$tmp_pool" "$tmp_claims"' EXIT

# ── Source ground truth: code<TAB>family, one per line ──────────────────
for entry in "${GATED_MODULES[@]}"; do
	IFS='|' read -r _section dir _prefixes floor _maxdepth <<<"$entry"
	floor="${floor:-$DEFAULT_FLOOR}"

	depth_args=()
	[ -n "${_maxdepth:-}" ] && depth_args=(--max-depth "$_maxdepth")

	tmp_mod="$(mktemp)"
	rg -U --no-filename -o "$EXTRACT_PATTERN" \
		"$repo_root/$dir" \
		"${depth_args[@]}" \
		--glob '*.go' --glob '!*_test.go' -g '!**/testdata/**' -g '!**/vendor/**' -g '!**/eventtest/**' \
		-r '$2	$1' >"$tmp_mod" || true
	sort -u "$tmp_mod" -o "$tmp_mod"
	mod_codes=$(wc -l <"$tmp_mod")

	if [ "$mod_codes" -lt "$floor" ]; then
		echo "ERROR: [$dir] extraction yielded $mod_codes codes (floor $floor) — the rg pattern or the module path is broken" >&2
		status=1
	fi

	cat "$tmp_mod" >>"$tmp_pool"
	rm -f "$tmp_mod"
done
sort -u "$tmp_pool" -o "$tmp_pool"
total_codes=$(wc -l <"$tmp_pool")

declare -A src_family=()
while IFS=$'\t' read -r code family; do
	[ -n "$code" ] || continue
	src_family["$code"]="$family"
done <"$tmp_pool"

# ── Doc claims ───────────────────────────────────────────────────────────
# Emits: L<TAB>section<TAB>code<TAB>family   (literal claim)
#        W<TAB>section<TAB>pattern<TAB>family (wildcard claim)
#        BADPAT<TAB>section<TAB>token         (malformed pattern)
sections_csv=""
prefixes_csv=""
for entry in "${GATED_MODULES[@]}"; do
	# 4th var absorbs the floor (and 5th the max-depth) so prefixes never
	# swallow them — a swallowed floor corrupts the LAST prefix ("x.|50"),
	# silently unclaiming every literal code under it (2026-09-16).
	IFS='|' read -r section _dir prefixes _claim_floor _claim_depth <<<"$entry"
	sections_csv+="${sections_csv:+,}${section}"
	prefixes_csv+="${prefixes_csv:+ }${prefixes}"
done

awk -v sections_csv="$sections_csv" -v prefixes_csv="$prefixes_csv" '
BEGIN {
	nsec = split(sections_csv, sarr, ",")
	for (s = 1; s <= nsec; s++) want_section[sarr[s]] = 1
	npfx = split(prefixes_csv, parr, " ")
	for (p = 1; p <= npfx; p++) want_prefix[parr[p]] = 1
	section = ""
}
/^### / {
	section = $0
	sub(/^### +/, "", section)
	next
}
/^#/ { section = "" }
!(section in want_section) { next }
/^\|/ {
	nf_ = split($0, cols, /\|/)
	if (nf_ < 4) next
	family = cols[3]
	gsub(/^ +| +$/, "", family)
	if (family !~ /^(Rejection|Conflict|Transient|Infrastructure|Corruption|Orchestration)$/) next
	row = $0
	while (match(row, /`[^`]+`/)) {
		tok = substr(row, RSTART + 1, RLENGTH - 2)
		row = substr(row, RSTART + RLENGTH)
		if (tok ~ /\*/) {
			if (tok ~ /^[a-z0-9_.]+\*$/) printf "W\t%s\t%s\t%s\n", section, tok, family
			else printf "BADPAT\t%s\t%s\n", section, tok
		} else if (tok ~ /^[a-z0-9_]+(\.[a-z0-9_]+)+$/) {
			split(tok, tparts, ".")
			if (want_prefix[tparts[1] "."]) printf "L\t%s\t%s\t%s\n", section, tok, family
		}
	}
}
' "$doc" >"$tmp_claims"

# ── Comparator ───────────────────────────────────────────────────────────
declare -A claimed=()
check_count=0

# Pass 1: doc claims vs source pool.
while IFS=$'\t' read -r kind section token family; do
	[ -n "$kind" ] || continue
	check_count=$((check_count + 1))

	case "$kind" in
	BADPAT)
		echo "ERROR: [$section] malformed doc code pattern \"$token\" — literal or trailing-star wildcard only" >&2
		status=1
		continue
		;;
	L)
		got="${src_family[$token]:-}"
		if [ -z "$got" ]; then
			echo "ERROR: [$section] doc lists \"$token\" ($family) but no errorfamily call site mints it — stale entry?" >&2
			status=1
		elif [ "$got" != "$family" ]; then
			echo "ERROR: [$section] \"$token\" is $family in the doc but the source constructs $got" >&2
			status=1
		fi
		claimed["$token"]=1
		;;
	W)
		stem="${token%\*}"
		if [ -z "$stem" ]; then
			echo "ERROR: [$section] wildcard \"$token\" needs a non-empty stem" >&2
			status=1
			continue
		fi
		matched=0
		while IFS=$'\t' read -r code fam; do
			case "$code" in
			"$stem"*)
				matched=$((matched + 1))
				claimed["$code"]=1
				if [ "$fam" != "$family" ]; then
					echo "ERROR: [$section] wildcard \"$token\" claims $family but \"$code\" is $fam in source" >&2
					status=1
				fi
				;;
			esac
		done <"$tmp_pool"
		if [ "$matched" -eq 0 ]; then
			echo "ERROR: [$section] wildcard \"$token\" matches no source code — stale pattern?" >&2
			status=1
		fi
		;;
	esac
done <"$tmp_claims"

# Pass 2: every source code must be claimed (literal or wildcard) in the doc.
uncovered=0
while IFS=$'\t' read -r code fam; do
	[ -n "$code" ] || continue
	if [ -z "${claimed[$code]:-}" ]; then
		echo "ERROR: \"$code\" ($fam) is minted by a gated module but docs/error-taxonomy.md never lists it — add it or extend a wildcard" >&2
		status=1
		uncovered=$((uncovered + 1))
	fi
done <"$tmp_pool"

if [ "$status" -eq 0 ]; then
	echo "✓ error-taxonomy: $total_codes codes across ${#GATED_MODULES[@]} modules match docs/error-taxonomy.md ($check_count doc claims)"
fi

exit "$status"
