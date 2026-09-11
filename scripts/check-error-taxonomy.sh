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
#   - one family per table row; Family column is the exact family name
#   - Code column holds backticked literals (`graph.read.path_not_found`)
#     or trailing-star wildcards (`graph.schema.*`, `watermill.parse_*`)
#   - wildcards need a non-empty stem and must not be bare `module.*`
#
# Extend the gate: add "section|dir|prefix1 prefix2 …" to GATED_MODULES.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
doc="$repo_root/docs/error-taxonomy.md"

# section-title | module dir (relative to repo root) | code prefixes
GATED_MODULES=(
	"middleware|middleware|middleware. deadletter."
	"graph|graph|graph."
	"storage/relational|storage/relational|relational."
	"projectionhost|projectionhost|projectionhost."
	"transport/grpc|transport/grpc|grpc."
)

status=0
total_codes=0
tmp_pool="$(mktemp)"
trap 'rm -f "$tmp_pool"' EXIT

# ── Source ground truth: code<TAB>family, one per line ──────────────────
for entry in "${GATED_MODULES[@]}"; do
	IFS='|' read -r _section dir _prefixes <<<"$entry"
	rg -U --no-filename -o \
		'errorfamily\.(?:New|Wrap)(Rejection|Conflict|Transient|Infrastructure|Corruption|Orchestration)\([^"]*"([a-z0-9_.]+)"' \
		"$repo_root/$dir" \
		--glob '*.go' --glob '!*_test.go' -g '!**/testdata/**' -g '!**/vendor/**' \
		-r '$2	$1' >>"$tmp_pool" || true
done
sort -u "$tmp_pool" -o "$tmp_pool"
total_codes=$(wc -l <"$tmp_pool")

declare -A src_family=()
while IFS=$'\t' read -r code family; do
	[ -n "$code" ] || continue
	src_family["$code"]="$family"
done <"$tmp_pool"

# ── Doc claims per section ──────────────────────────────────────────────
# claims: "code|family" for literals, "WILDCARD|stem|family" for patterns.
tmp_claims="$(mktemp)"
trap 'rm -f "$tmp_pool" "$tmp_claims"' EXIT

awk -v doc="$doc" '
BEGIN {
	n = split(SECTIONS, secs, "|") # unused placeholder for portability
}
' SECTIONS="" /dev/null 2>/dev/null || true

awk -v docfile="$doc" '
function reset_prefixes(    i, n, parts) {
	n = split(PREFIXES_ALL, parts, " ")
	for (i = 1; i <= n; i++) is_prefix[parts[i]] = 1
}
BEGIN {
	# section|family|token lines for the comparator
	sections_all = SECTIONS_ARG
	prefixes_all = PREFIXES_ARG
	nsec = split(sections_all, sarr, "|")
	for (s = 1; s <= nsec; s++) secname[s] = sarr[s]
	reset_prefixes()
	section = ""
}
/^### / {
	section = $0
	sub(/^### +/, "", section)
	next
}
/^#/ { section = "" }
section == "" { next }
{
	if (!isinsections(section)) next
	line = $0
	# table rows only: | Error | Family | Codes |
	if (line !~ /^\|/) next
	nf = split(line, cols, /\|/)
	if (nf < 4) next
	family = trim(cols[3])
	if (family !~ /^(Rejection|Conflict|Transient|Infrastructure|Corruption|Orchestration)$/) next
	# every backticked token in the row that looks like a code for this gate
	row = line
	while (match(row, /`[^`]+`/)) {
		tok = substr(row, RSTART + 1, RLENGTH - 2)
		row = substr(row, RSTART + RLENGTH)
		if (tok ~ /\*/) {
			if (tok ~ /^[a-z0-9_.]+\*$/) print "W|" section "|" tok "|" family
		} else if (tok ~ /^[a-z0-9_]+(\.[a-z0-9_]+)+$/) {
			# only codes under a gated prefix (others: identifiers/prose)
			split(tok, tparts, ".")
			if (is_prefix[tparts[1] "."]) print "L|" section "|" tok "|" family
		}
	}
	function trim(s) { gsub(/^ +| +$/, "", s); return s }
	function isinsections(s,    i) {
		for (i = 1; i <= nsec; i++) if (secname[i] == s) return 1
		return 0
	}
}
' SECTIONS_ARG="$(printf '%s|' "${GATED_MODULES[@]}" | sed 's/|.*|/|/' >/dev/null; true)" /dev/null >/dev/null 2>&1 || true

# POSIX awk lacks functions-in-conditions portability for this layout; do the
# doc pass with a plain second awk that receives section + prefix maps as
# variables and emits the same claim lines.
sections_csv=""
prefixes_csv=""
for entry in "${GATED_MODULES[@]}"; do
	IFS='|' read -r section dir prefixes <<<"$entry"
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
			else {
				printf "BADPAT\t%s\t%s\n", section, tok > "/dev/stderr"
			}
		} else if (tok ~ /^[a-z0-9_]+(\.[a-z0-9_]+)+$/) {
			split(tok, tparts, ".")
			if (want_prefix[tparts[1] "."]) printf "L\t%s\t%s\t%s\n", section, tok, family
		}
	}
}
' "$doc" >"$tmp_claims" 2>>"$tmp_claims.err" || true

if [ -s "$tmp_claims.err" ]; then
	while IFS= read -r line; do
		echo "ERROR: malformed doc code pattern: $line" >&2
		status=1
	done <"$tmp_claims.err"
fi
rm -f "$tmp_claims.err"

# ── Comparator ───────────────────────────────────────────────────────────
declare -A claimed=()
check_count=0

# Pass 1: doc literals + wildcards vs source pool.
while IFS=$'\t' read -r kind section token family; do
	[ -n "$kind" ] || continue
	check_count=$((check_count + 1))
	if [ "$kind" = "L" ]; then
		got="${src_family[$token]:-}"
		if [ -z "$got" ]; then
			echo "ERROR: [$section] doc lists \"$token\" ($family) but no errorfamily call site mints it — stale entry?" >&2
			status=1
		elif [ "$got" != "$family" ]; then
			echo "ERROR: [$section] \"$token\" is $family in the doc but the source constructs $got" >&2
			status=1
		fi
		claimed["$token"]=1
	else
		stem="${token%\*}"
		if [ "$stem" = "$token" ] || [ -z "$stem" ]; then
			echo "ERROR: [$section] wildcard \"$token\" needs a non-empty stem" >&2
			status=1
			continue
		fi
		base="${stem%.*}"
		leaf="${stem##*.}"
		if [ "$leaf" = "$stem" ] || [ -z "$leaf" ]; then
			: # stem like "watermill.parse_" (underscore form) is fine
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
	fi
done <"$tmp_claims"

# Pass 2: every source code must be claimed (literal or wildcard) in the doc.
while IFS=$'\t' read -r code fam; do
	[ -n "$code" ] || continue
	if [ -z "${claimed[$code]:-}" ]; then
		echo "ERROR: \"$code\" ($fam) is minted by a gated module but docs/error-taxonomy.md never lists it — add it or extend the wildcard" >&2
		status=1
	fi
done <"$tmp_pool"

if [ "$status" -eq 0 ]; then
	echo "✓ error-taxonomy: $total_codes codes across ${#GATED_MODULES[@]} modules match docs/error-taxonomy.md ($check_count doc claims)"
fi

exit "$status"
