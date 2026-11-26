#!/usr/bin/env bash
# Local mirror of the CI error-audit gate (ci.yml job "error-audit"): runs
# `erraudit lint ./... --enforce-go-error-family --type-aware
# --disable-extensions --modules=none` in every module and ratchets each
# module's finding count against scripts/erraudit-baseline.txt.
#
# WHY A RATCHET: `--enforce-go-error-family` was decorative on `erraudit
# lint` until v0.8.0 (erraudit CHANGELOG 0.8.0, "Every --enforce-* flag now
# produces findings on erraudit lint"), so the zero this sweep reported
# against v0.7.0 was vacuous. The first real run (2026-11-26, erraudit
# cb96501) reported findings in most modules; the baseline header carries
# the pinned totals and the pinning version. The gate fails on any module
# that grows, on a new offender, on a module erraudit could not measure,
# and on a dangling baseline row.
#
# Usage:
#   scripts/erraudit-sweep.sh [erraudit-binary]   gate (default)
#   scripts/erraudit-sweep.sh --update-baseline   re-pin every count
#   scripts/erraudit-sweep.sh --self-test         mutation proof of the ratchet
# Env:   ERRAUDIT_BIN overrides the binary (a positional path wins);
#        GOEXPERIMENT=jsonv2 for encoding/json/v2; GOWORK is forced off.
#        ERRAUDIT_BASELINE points the gate at another baseline file. Fixtures
#        only, never a normal run.
#
# Binary policy (owner ruling 2026-11-26): no erraudit source is vendored in
# this repo. CI installs the latest release; locally the sweep runs whatever
# binary is installed, and prints the deciding version. Counts are
# version-dependent: re-pin after upgrading erraudit and review the diff.
set -uo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
report_dir="$repo_root/build/erraudit-sweep"
baseline="${ERRAUDIT_BASELINE:-$repo_root/scripts/erraudit-baseline.txt}"
measured_tsv="$report_dir/measured.tsv"
# One flag set for CI and the local mirror. --modules=none scopes a run to
# its own module: the default fan-out (erraudit >= v0.8.0) makes the root
# row the sum of every nested module. --exclude keeps the vendored docserver
# bundles out: the JS/TS detectors joined `lint` in v0.8.0 and are no longer
# covered by --disable-extensions, and a vendored bundle cannot gate a Go
# error-policy leg.
flags=(
	--enforce-go-error-family
	--type-aware
	--disable-extensions
	--modules=none
	--exclude
	docserver/static
	--format
	json
)

mode=gate
ERRAUDIT="${ERRAUDIT_BIN:-erraudit}"
for arg in "$@"; do
	case "$arg" in
	--update-baseline) mode=update ;;
	--self-test) mode=selftest ;;
	-*)
		echo "unknown flag: $arg (supported: --update-baseline, --self-test)" >&2
		exit 2
		;;
	*) ERRAUDIT="$arg" ;;
	esac
done

# evaluate <baseline> <measured.tsv> <lines-var>: one verdict line per module
# in the named variable; sets eval_fail to the number of failing rows (0 =
# green) and eval_failing to the modules with new findings. Results travel
# through variables, never a command substitution: a substitution would run
# this function in a subshell and lose the counter.
evaluate() {
	local base_file="$1" measured_file="$2"
	local -n lines_ref="$3"
	local -A base=() measured=()
	local -a order=() dangling=()
	local line count name base_count verdict
	eval_fail=0
	eval_failing=()
	declare -gA eval_detail=()

	while IFS= read -r line; do
		case "$line" in
		'' | '#'*) continue ;;
		esac
		if [[ ! "$line" =~ ^([0-9]+)$'\t'(.+)$ ]]; then
			printf -v line '%-8s malformed baseline row: %s' BAD-ROW "$line"
			lines_ref+=$line$'\n'
			eval_fail=$((eval_fail + 1))
			continue
		fi
		base["${BASH_REMATCH[2]}"]="${BASH_REMATCH[1]}"
	done <"$base_file"

	while IFS=$'\t' read -r count name; do
		[ -n "$name" ] || continue
		measured["$name"]="$count"
		order+=("$name")
	done <"$measured_file"

	for name in "${order[@]}"; do
		count="${measured[$name]}"
		base_count="${base[$name]:-}"
		verdict=OK
		case "$count" in
		ERR)
			verdict=ERROR
			eval_fail=$((eval_fail + 1))
			;;
		*)
			if [ -z "$base_count" ]; then
				if [ "$count" -gt 0 ]; then
					verdict=NEW
					eval_fail=$((eval_fail + 1))
					eval_failing+=("$name")
					eval_detail["$name"]="NEW $count/0"
				fi
			elif [ "$count" -gt "$base_count" ]; then
				verdict=GREW
				eval_fail=$((eval_fail + 1))
				eval_failing+=("$name")
				eval_detail["$name"]="GREW $count/$base_count"
			elif [ "$count" -lt "$base_count" ]; then
				verdict=DOWN
			fi
			;;
		esac
		printf -v line '%-8s %5s/%-5s %s' "$verdict" "$count" "${base_count:--}" "$name"
		lines_ref+=$line$'\n'
		[ "$verdict" != DOWN ] ||
			lines_ref+=$'         ratcheted down: lower the row with --update-baseline\n'
	done

	for name in "${!base[@]}"; do
		[ -n "${measured[$name]:-}" ] || dangling+=("$name")
	done
	if [ "${#dangling[@]}" -gt 0 ]; then
		mapfile -t dangling < <(printf '%s\n' "${dangling[@]}" | sort)
		for name in "${dangling[@]}"; do
			printf -v line '%-8s %5s/%-5s %s (baseline row without a module)' DANGLING '-' "${base[$name]}" "$name"
			lines_ref+=$line$'\n'
			eval_fail=$((eval_fail + 1))
		done
	fi
}

# log_name_for: the report-file stem of a module ("." is the root module).
log_name_for() {
	local name="$1" log_name
	log_name=$(echo "$name" | tr '/' '_')
	[ "$name" != "." ] || log_name=root
	printf '%s\n' "$log_name"
}

# measure: runs erraudit in every module and writes <count><TAB><module>
# rows; a module erraudit could not measure gets the count ERR.
measure() {
	local mod_dir name out_file rc count
	: >"$measured_tsv"
	while IFS= read -r mod_dir; do
		name="."
		[ "$mod_dir" = "$repo_root" ] || name=${mod_dir#"$repo_root"/}
		out_file="$report_dir/$(log_name_for "$name").json"
		rc=0
		(cd "$mod_dir" && env GOWORK=off GOEXPERIMENT=jsonv2 GOFLAGS= "$ERRAUDIT" lint ./... "${flags[@]}" >"$out_file" 2>"$out_file.log") || rc=$?
		count=ERR
		case "$rc" in
		0 | 2)
			count="$(jq -r '.findings | length' "$out_file" 2>/dev/null)" || count=ERR
			[[ "$count" =~ ^[0-9]+$ ]] || count=ERR
			;;
		esac
		if [ "$count" = ERR ]; then
			echo "erraudit could not measure $name (rc=$rc, see $out_file.log)" >&2
		fi
		printf '%s\t%s\n' "$count" "$name" >>"$measured_tsv"
	done < <(find "$repo_root" -name go.mod -not -path '*/vendor/*' -not -path '*/.git/*' -exec dirname {} \; | sort)
}

# write_baseline: re-pins the baseline from the last measurement. Rows that
# grew against the previous pin are listed so the reviewer sees them.
write_baseline() {
	local pinned_rows version
	pinned_rows="$(awk -F'\t' '$1 ~ /^[0-9]+$/ && $1 > 0' "$measured_tsv" | sort -t "$(printf '\t')" -k1,1nr -k2,2)"
	version="$("$ERRAUDIT" --version 2>&1 | head -1)"
	awk -F'\t' 'NR == FNR { old[$2] = $1; next } ($2 in old) && ($1 > old[$2]) { printf "GREW     %s: %s -> %s\n", $2, old[$2], $1 }' \
		<(grep -v '^#' "$baseline" 2>/dev/null) <(printf '%s\n' "$pinned_rows")
	{
		echo "# Error-policy ratchet baseline (see scripts/erraudit-sweep.sh)."
		echo "# Format: <findings><TAB><module-dir>, findings desc."
		echo "# Regenerate: scripts/erraudit-sweep.sh --update-baseline"
		echo "# Pinned $(date +%Y-%m-%d) with ${version}: $(awk -F'\t' 'NF == 0 { next } { s += $1; n++ } END { print s + 0 " findings in " n + 0 " modules" }' <<<"$pinned_rows")."
		printf '%s\n' "$pinned_rows"
	} >"$baseline"
	echo "re-pinned $baseline"
}

# show_findings: the first sites of a failing module, so a red gate points at
# the code instead of at a log directory.
show_findings() {
	local json total
	json="$report_dir/$(log_name_for "$1").json"
	total="$(jq -r '.findings | length' "$json" 2>/dev/null)" || total=0
	jq -r --arg root "$repo_root/" '.findings[:5][] | "  \(.position.file | ltrimstr($root)):\(.position.line): \(.rule): \(.message | split(" (")[0])"' "$json" 2>/dev/null
	[ "$total" -le 5 ] || echo "  ... and $((total - 5)) more in ${json#"$repo_root"/}"
	echo "  reproduce: (cd $1 && erraudit lint ./... ${flags[*]:0:${#flags[@]}-2})"
}

# self_test: plants each fault class in an in-memory baseline and measured
# table, and asserts evaluate's row count, verdict token, and the module it
# names. No erraudit run.
self_test() {
	local failures=0
	run_case() {
		local label="$1" want_fail="$2" token="$3" base_rows="$4" measured_rows="$5" want_failing="${6-}" out="" recorded=""
		eval_fail=0
		evaluate <(printf '%s\n' "$base_rows") <(printf '%s\n' "$measured_rows") out
		recorded="${eval_failing[*]-}"
		if [ "$eval_fail" != "$want_fail" ] || ! grep -q "^$token" <<<"$out" ||
			[ "$recorded" != "$want_failing" ]; then
			echo "self-test FAIL: $label (failures=$eval_fail, want $want_fail; token '$token' $(grep -q "^$token" <<<"$out" && echo found || echo missing); recorded '$recorded', want '$want_failing')" >&2
			failures=$((failures + 1))
		fi
	}
	local pinned=$'5\ta\n2\tb'
	run_case "equal counts pass" 0 OK "$pinned" "$pinned"
	run_case "growth fails and names the module" 1 GREW "$pinned" $'6\ta\n2\tb' a
	run_case "shrink passes" 0 DOWN "$pinned" $'3\ta\n2\tb'
	run_case "new offender fails and is named" 1 NEW "$pinned" $'5\ta\n2\tb\n1\tc' c
	run_case "unmeasurable module fails" 1 ERROR "$pinned" $'ERR\ta\n2\tb'
	run_case "dangling row fails" 1 DANGLING "$pinned" $'5\ta'
	run_case "malformed row fails" 1 BAD-ROW $'x\ta' $'0\ta'
	run_case "module born clean passes" 0 OK '' $'0\tz'

	[ "$failures" -eq 0 ] && echo "self-test: 8 fault classes verified, all green"
	[ "$failures" -eq 0 ]
}

case "$mode" in
selftest)
	self_test
	exit $?
	;;
esac

command -v "$ERRAUDIT" >/dev/null 2>&1 || [ -x "$ERRAUDIT" ] || {
	echo "erraudit not found: $ERRAUDIT" >&2
	echo "install erraudit, pass a binary path, or set ERRAUDIT_BIN" >&2
	exit 1
}
command -v jq >/dev/null 2>&1 || {
	echo "jq not found: the sweep counts findings from erraudit's JSON output" >&2
	exit 1
}
[ -f "$baseline" ] || [ "$mode" = update ] || {
	echo "baseline missing: $baseline (create it with --update-baseline)" >&2
	exit 1
}
mkdir -p "$report_dir"
echo "erraudit: $("$ERRAUDIT" --version 2>&1 | head -1) ($ERRAUDIT)"
measure

if [ "$mode" = update ]; then
	if grep -q '^ERR' "$measured_tsv"; then
		echo "refusing to re-pin: erraudit could not measure every module" >&2
		exit 1
	fi
	write_baseline
	exit 0
fi

lines=""
evaluate "$baseline" "$measured_tsv" lines
printf '%s' "$lines"
echo "---"
echo "modules=$(wc -l <"$measured_tsv") failing=$eval_fail pinned=$(grep -vc '^#' "$baseline") (logs in build/erraudit-sweep/)"
if [ "${#eval_failing[@]}" -gt 0 ]; then
	for name in "${eval_failing[@]}"; do
		echo "${eval_detail[$name]} $name (measured/pinned), first sites:"
		show_findings "$name"
	done
fi
[ "$eval_fail" -eq 0 ] || exit 2
