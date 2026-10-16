#!/usr/bin/env bash
# Generates the per-module code tables in docs/error-taxonomy.md from
# erraudit's errorflow export — the companion of check-error-taxonomy.sh.
#
# Ground truth: the export's `codes` array (schemaVersion 2, erraudit
# v0.7.0): every go-error-family code minted in the module, named
# sentinels AND inline New/Newf/Wrap/Wrapf calls. The gate's rg regex was
# blind to the f-variants (5 inline-only codes in core/event alone) —
# erraudit is the strictly better pool, and this generator makes the doc a
# generated artifact instead of a hand-maintained one that lied for weeks.
#
# Both scripts share the GATED_MODULES table (parsed from
# check-error-taxonomy.sh): same sections, same code prefixes, same
# module-dir resolution. The gate verifies what this script writes; the
# doc conventions it depends on are preserved exactly —
#   - Family stays column 3 of each table row
#   - codes stay backticked literals (`event.date_parse`)
#   - prose lines below tables are ignored by the gate's parser
#
# What lands between the begin/end markers in docs/error-taxonomy.md:
#   - one `### <section>` heading per GATED_MODULES section, in
#     first-appearance order
#   - one row per code: | <sentinel name or -> | <Family> | `<code>` |
#   - dead-sentinel and inline-mint count prose lines per section
#
# Modes:
#   (no args)  rewrite the marked region in place
#   --check    exit 1 with a diff when the committed region is stale
#   --pool     dump the filtered code pool as section<TAB>code<TAB>Family
#              lines (the check-error-taxonomy.sh gate's preferred pool
#              source; rc 3 = erraudit unavailable, so the gate falls back
#              to its rg extraction)
#
# Activation: needs an erraudit whose errorflow export reports
# schemaVersion >= 2. Missing or older erraudit warns and exits 0 — the
# same activation pattern as the CI errorflow-freshness step, so the gate
# turns itself on when the vendored pin bumps to v0.7.0 and stays green
# before that.
#
# Override for local/unreleased builds: ERRAUDIT_BIN=/path/to/erraudit.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
doc="$repo_root/docs/error-taxonomy.md"
gate="$repo_root/scripts/check-error-taxonomy.sh"
erraudit="${ERRAUDIT_BIN:-erraudit}"
mode="${1:-}"

if [ "$mode" != "" ] && [ "$mode" != "--check" ] && [ "$mode" != "--pool" ]; then
	echo "usage: $0 [--check|--pool]   (regenerate docs/error-taxonomy.md module tables)" >&2
	exit 2
fi

begin='<!-- generate-error-taxonomy:begin (scripts/generate-error-taxonomy.sh writes this region — edit outside the markers) -->'
end='<!-- generate-error-taxonomy:end -->'

if ! command -v "$erraudit" >/dev/null 2>&1; then
	if [ "$mode" = "--pool" ]; then
		exit 3
	fi
	echo "WARN: erraudit not found — taxonomy doc generation skipped (needs errorflow schemaVersion 2)" >&2
	exit 0
fi

if [ "$mode" != "--pool" ] && { ! grep -qxF "$begin" "$doc" || ! grep -qxF "$end" "$doc"; }; then
	echo "ERROR: $doc is missing the generate-error-taxonomy markers — refusing to touch it" >&2
	exit 1
fi

# GATED_MODULES rows from the gate script: section|dir|prefixes|floor|maxdepth.
# The 5th field (rg --max-depth) marks entries the rg gate scopes SHALLOW
# (root files only); entries WITHOUT it scope the whole directory tree,
# nested modules included — for those, erraudit (hard module boundaries)
# needs --modules=all to see the same codes.
declare -a sections=()
declare -A section_dirs=() section_prefixes=() section_floor=() dir_deep=()
while IFS='|' read -r section dir prefixes floor maxdepth; do
	if [ -z "${section_dirs[$section]+_}" ]; then
		sections+=("$section")
		section_dirs["$section"]=""
		section_prefixes["$section"]=""
		section_floor["$section"]=1
	fi
	section_dirs["$section"]+="${section_dirs[$section]:+:}$dir"
	section_prefixes["$section"]+=" $prefixes"
	if [ -n "${floor:-}" ] && [ "$floor" -gt "${section_floor[$section]}" ]; then
		section_floor["$section"]="$floor"
	fi
	# Deep (whole-tree) scan unless EVERY entry for this dir scopes shallow.
	if [ -n "${maxdepth:-}" ]; then
		[ "${dir_deep[$dir]:-1}" -eq 0 ] || dir_deep["$dir"]=0
	else
		dir_deep["$dir"]=1
	fi
done < <(sed -n '/^GATED_MODULES=(/,/^)$/s/^[[:space:]]*"\(.*\)".*/\1/p' "$gate")

# Nested modules: a gated dir may not be a module root itself
# (storage/relational lives inside the storage module). Run erraudit from
# the enclosing module root with a subpath pattern — module boundaries are
# hard, ./... from a root never crosses them.
run_args_for_dir() {
	local dir="$1" root="$1"
	while [ ! -f "$repo_root/$root/go.mod" ] && [ "$root" != "." ]; do
		root="$(dirname "$root")"
	done
	if [ "$root" = "." ] && [ ! -f "$repo_root/go.mod" ]; then
		echo "ERROR: no module root found above $dir" >&2
		return 1
	fi
	if [ "$root" = "$dir" ]; then
		printf '%s %s\n' "$dir" "./..."
	else
		printf '%s %s\n' "$root" "./${dir#"$root"/}/..."
	fi
}

# Whole-tree entries cover every nested module under the dir. erraudit's
# --modules=all fan-out resolves patterns in the root module's context and
# dies on module boundaries (queue/mysql), so the fan-out is done here:
# one standalone run per module — nested modules only ever load their own
# go.sum that way.
modules_under_dir() {
	local dir="$1"
	printf '%s\n' "$dir"
	find "$repo_root/$dir" -mindepth 2 -name go.mod -not -path '*/vendor/*' -printf '%h\n' |
		sed "s|^$repo_root/||"
}

run_errorflow() { # $1 = module dir to cd into, $2 = package pattern
	# CGO_ENABLED=1: cgo-backed modules (duckdbengine) fail package loading
	# without it; GOWORK/GOFLAGS are cleared because caller direnv shells
	# export -mod=mod which breaks module-mode pattern analysis.
	(cd "$repo_root/$1" && env GOWORK=off GOFLAGS= GOEXPERIMENT=jsonv2 CGO_ENABLED=1 \
		"$erraudit" errorflow "$2" --type-aware 2>/dev/null)
}

schema_checked=0
check_schema() {
	if [ "$schema_checked" -eq 1 ]; then
		return 0
	fi
	schema_checked=1
	local version
	version="$(jq -r '.schemaVersion // 0' <<<"$1")"
	if [ "$version" -lt 2 ]; then
		if [ "$mode" = "--pool" ]; then
			exit 3
		fi
		echo "WARN: erraudit errorflow exports schemaVersion $version (need 2 for the codes array) — taxonomy doc generation skipped; activates at the v0.7.0 pin" >&2
		exit 0
	fi
}

# extract_rows <json> <sec_dir> <prefixes>: appends code\tfamily\tname and
# dead-sentinel rows for one module's export to the section's TSVs.
extract_rows() {
	local json="$1" sec_dir="$2" prefixes="$3"

	jq -r --arg pfx "$prefixes" '
		($pfx | split(" ") | map(select(length > 0))) as $pfx
		| ((.errors // [])
			| map(select((.code // "") != ""))
			| map({key: .code, value: .name})
			| from_entries) as $names
		| (.codes // [])
		| map(select(((.declared // "") | endswith("_test.go")) | not))
		| map(select(.family | test("^(rejection|conflict|transient|infrastructure|corruption|orchestration)$")))
		| map(select(.code as $c | any($pfx[]; . as $p | $c | startswith($p))))
		| unique_by([.code, .family])
		| .[] | [.code, .family, ($names[.code] // "-")] | @tsv
	' <<<"$json" >>"$sec_dir/codes.tsv"

	jq -r '
		(.errors // [])
		| map(select(.status == "unreferenced"))
		| sort_by(.name)
		| .[] | [.name, (.declared // "")] | @tsv
	' <<<"$json" >>"$sec_dir/dead.tsv"
}

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

status=0
for section in "${sections[@]}"; do
	sec_dir="$tmp_root/$(echo "$section" | tr '/ ()' '____')"
	mkdir -p "$sec_dir"

	: >"$sec_dir/codes.tsv"
	: >"$sec_dir/dead.tsv"

	IFS=':' read -r -a dirs <<<"${section_dirs[$section]}"
	for dir in "${dirs[@]}"; do
		if [ "${dir_deep[$dir]:-1}" -eq 1 ]; then
			while IFS= read -r mod; do
				json="$(run_errorflow "$mod" "./...")"
				check_schema "$json"
				extract_rows "$json" "$sec_dir" "${section_prefixes[$section]}"
			done < <(modules_under_dir "$dir")
		else
			run_args="$(run_args_for_dir "$dir")"
			json="$(run_errorflow "${run_args%% *}" "${run_args#* }")"
			check_schema "$json"
			extract_rows "$json" "$sec_dir" "${section_prefixes[$section]}"
		fi
	done

	# Deduplicate across the section's dirs (a code minted in two dirs of
	# one section keeps its first name) and sort by code.
	awk -F'\t' '!seen[$1 FS $2]++' "$sec_dir/codes.tsv" |
		LC_ALL=C sort -t$'\t' -k1,1 -s >"$sec_dir/codes.uniq.tsv"

	rows=$(wc -l <"$sec_dir/codes.uniq.tsv")
	if [ "$rows" -lt "${section_floor[$section]}" ]; then
		echo "ERROR: [$section] erraudit extraction yielded $rows codes (floor ${section_floor[$section]}) — the module or the prefixes moved" >&2
		status=1
	fi
done

if [ "$status" -ne 0 ]; then
	exit "$status"
fi

if [ "$mode" = "--pool" ]; then
	for section in "${sections[@]}"; do
		sec_dir="$tmp_root/$(echo "$section" | tr '/ ()' '____')"
		awk -F'\t' -v sec="$section" \
			'{printf "%s\t%s\t%s%s\n", sec, $1, toupper(substr($2, 1, 1)), substr($2, 2)}' \
			"$sec_dir/codes.uniq.tsv"
	done
	exit 0
fi

# ── Render the marked region ──────────────────────────────────────────────
region="$tmp_root/region.md"
: >"$region"

for section in "${sections[@]}"; do
	sec_dir="$tmp_root/$(echo "$section" | tr '/ ()' '____')"

	# Column widths over header + cells (all-ASCII tables: byte width ==
	# display width, prettier-stable).
	w1=${#section}
	w2=6 # "Family"
	w3=4 # "Code"
	declare -a col1=() col2=() col3=()
	while IFS=$'\t' read -r code family name; do
		[ -n "$code" ] || continue
		family="${family^}"
		col1+=("$name")
		col2+=("$family")
		col3+=("\`$code\`")
		[ ${#name} -gt "$w1" ] && w1=${#name}
		[ ${#family} -gt "$w2" ] && w2=${#family}
		[ $((${#code} + 2)) -gt "$w3" ] && w3=$((${#code} + 2))
	done <"$sec_dir/codes.uniq.tsv"

	{
		echo "### $section"
		echo
		printf '| %-*s | %-*s | %-*s |\n' "$w1" "Error" "$w2" "Family" "$w3" "Code"
		printf '| %s | %s | %s |\n' \
			"$(printf '%*s' "$w1" '' | tr ' ' '-')" \
			"$(printf '%*s' "$w2" '' | tr ' ' '-')" \
			"$(printf '%*s' "$w3" '' | tr ' ' '-')"
		for i in "${!col3[@]}"; do
			printf '| %-*s | %-*s | %-*s |\n' "$w1" "${col1[$i]}" "$w2" "${col2[$i]}" "$w3" "${col3[$i]}"
		done
		echo

		dead_names="$(awk -F'\t' '!seen[$1]++ {printf "%s%s", sep, "`" $1 "`"; sep=", "}' "$sec_dir/dead.tsv")"
		if [ -n "$dead_names" ]; then
			dead_count="$(awk -F'\t' '!seen[$1]++' "$sec_dir/dead.tsv" | wc -l)"
			echo "- Unreferenced named errors (dead-sentinel candidates): $dead_names"
		fi

		inline_count="$(awk -F'\t' '$3 == "-"' "$sec_dir/codes.uniq.tsv" | wc -l)"
		total=$(wc -l <"$sec_dir/codes.uniq.tsv")
		if [ "$inline_count" -gt 0 ]; then
			echo "- $inline_count of $total codes are inline mints (no named sentinel)"
		fi
		echo
	} >>"$region"
done

# ── Splice the region into the doc ────────────────────────────────────────
tmp_doc="$tmp_root/doc.md"
awk -v begin="$begin" -v end="$end" -v region="$region" '
	$0 == begin { print; print ""; while ((getline line < region) > 0) print line; inside = 1; next }
	$0 == end   { inside = 0; print; next }
	!inside     { print }
' "$doc" >"$tmp_doc"

if [ "$mode" = "--check" ]; then
	if diff -u "$doc" "$tmp_doc" >"$tmp_root/drift.diff"; then
		echo "✓ error-taxonomy doc is up to date with the erraudit codes pool"
	else
		echo "ERROR: docs/error-taxonomy.md region is stale — run scripts/generate-error-taxonomy.sh and commit" >&2
		cat "$tmp_root/drift.diff" >&2
		exit 1
	fi
else
	cat "$tmp_doc" >"$doc"
	echo "✓ regenerated module tables in docs/error-taxonomy.md ($(wc -l <"$region") lines)"
fi
