#!/usr/bin/env bash
# check-coverage.sh — coverage FLOOR for every module in the workspace.
#
# Why this exists: before 2026-10-26 this gate pinned 11 of 89 test modules,
# so 78 modules could lose their tests entirely and nothing went red. A gate
# that measures 12% of the surface is a claim, not a check. The floors live in
# scripts/coverage-baseline.txt, one row per module in flake.nix `testModules`,
# and the gate fails on:
#
#   * a module regression beyond the lower tolerance,
#   * a module in `testModules` with no baseline row (unmeasured surface),
#   * a baseline row that no longer resolves to a module (dangling key),
#   * a module that gained a package WITHOUT tests (the floor cannot be met by
#     adding untested packages),
#   * coverage data vanishing for a module that had some (tests deleted).
#
# Upward drift beyond the tolerance is reported as STALE but does not fail:
# new tests are never a defect, and a force-refreshed baseline would train
# people to re-pin until green.
#
# Usage: bash scripts/check-coverage.sh [flags]
#   (default)        measure, compare against the baseline, exit non-zero on drift
#   --update         re-measure and REWRITE scripts/coverage-baseline.txt
#   --profile FILE   consume an existing merged coverprofile instead of running tests
#   --report DIR     also write module/package TSV and a markdown table into DIR
#   --quiet          print only rows that are not ok
#   --self-test      fault-inject the compare logic against fixtures (no Go needed)
#   --help
#
# Environment:
#   COVERAGE_BASELINE   baseline path (default scripts/coverage-baseline.txt)
#   COVERAGE_WORKDIR    scratch dir (default build/coverage)
#   COVERAGE_TOLERANCE  upward tolerance in points (default 2.0)
#   COVERAGE_TEST_P     go test -p value for the measurement run (optional)
#
# Measurement mode is `aggregate`: one workspace-wide test run emits a merged
# coverprofile, and a module's percentage is covered/total STATEMENTS for the
# files it owns (deepest module directory wins, so nested modules such as
# metaengine/pgengine are never double counted). The pre-2026-10-26 version
# took the FIRST package's coverage line, which in a multi-package module is an
# arbitrary package, not the module.
#
# The measurement is load-sensitive (timeout-skipped tests measure 3-5 points
# lower on a loaded box, deterministically on GitHub runners), so the baseline
# is pinned from quiet-window runs and CI widens only the LOWER bound.

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

# Self-heal the toolchain cache env (the nix wrapper runs this script bare;
# without writable caches go test fails and coverage parses as EMPTY — the
# vacuous 0.0%-DRIFT class this script once reported). Forced chain via
# scripts/go-env.sh (T05): the old set-before-use form let an ambient
# GOTOOLCHAIN=local (nixpkgs Go wrapper) poison the run the same way.
source "$PWD/scripts/go-env.sh"

BASELINE="${COVERAGE_BASELINE:-scripts/coverage-baseline.txt}"
WORKDIR="${COVERAGE_WORKDIR:-build/coverage}"
TOLERANCE="${COVERAGE_TOLERANCE:-2.0}"
# CI lower-bound allowance (2026-10-14): GitHub runners are the documented
# "loaded box" class of the header note — timeout-skipped tests measure
# 3-5 points lower DETERMINISTICALLY (run 38619781576 measured the exact
# table a load1>60 local run produced), while quiet-dev-box fresh runs match
# the baseline exactly. The baseline stays pinned to quiet-window numbers
# per policy; on CI (GitHub sets CI=true) only the LOWER bound widens, so
# the environment shortfall passes while real regressions (beyond -6.0) and
# ANY local drift beyond -2.0 still fail.
CI_LOWER_TOLERANCE=6.0
LOWER_TOLERANCE="${COVERAGE_LOWER_TOLERANCE:-$TOLERANCE}"
if [ "${CI:-}" = "true" ] && [ -z "${COVERAGE_LOWER_TOLERANCE:-}" ]; then
	LOWER_TOLERANCE=$CI_LOWER_TOLERANCE
fi

PROFILE_IN=""
REPORT_DIR=""
QUIET=0
MODE=gate

die() {
	echo "::error::$1" >&2
	if [ $# -gt 1 ]; then
		printf '%s\n' "${@:2}" >&2
	fi
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
	--update) MODE=update ;;
	--self-test) MODE=selftest ;;
	--profile)
		shift
		[ $# -gt 0 ] || die "--profile requires a file"
		PROFILE_IN="$1"
		;;
	--report)
		shift
		[ $# -gt 0 ] || die "--report requires a directory"
		REPORT_DIR="$1"
		;;
	--quiet) QUIET=1 ;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

# list_modules prints the authoritative module list: flake.nix `testModules`.
# That list is already gated against the go.mod tree by `nix run .#check-modules`,
# so it is the one place a new module must be registered. Parsing it (instead of
# a second hand-maintained list) is what makes "every module measured" fail
# closed when someone adds a module and forgets the baseline.
list_modules() {
	local mods n
	mods="$(sed -n '/testModules = \[/,/^[[:space:]]*\];/p' flake.nix |
		grep -oE '"[^"]+"' | tr -d '"' | sort)"
	n="$(printf '%s\n' "$mods" | grep -c . || true)"
	[ "$n" -ge 50 ] || die "module list extraction from flake.nix is implausible ($n modules) — did the testModules block move?"
	printf '%s\n' "$mods"
}

# collect fills $1 with the measurement inputs:
#   modules.txt   one module path per line (sorted)
#   packages.txt  importpath|dir|testfiles|xtestfiles (go list)
#   merged.out    one merged coverprofile for the whole workspace
collect() {
	local wd="$1" mod
	mkdir -p "$wd"
	list_modules >"$wd/modules.txt"
	local -a paths=()
	while IFS= read -r mod; do
		paths+=("./$mod/...")
	done <"$wd/modules.txt"
	[ "${#paths[@]}" -gt 0 ] || die "no modules to measure"

	if [ -n "$PROFILE_IN" ]; then
		[ -f "$PROFILE_IN" ] || die "--profile '$PROFILE_IN' does not exist"
		cp "$PROFILE_IN" "$wd/merged.out"
	else
		echo "==> Measuring coverage across $(wc -l <"$wd/modules.txt") modules (this is the test run)" >&2
		local -a pargs=()
		if [ -n "${COVERAGE_TEST_P:-}" ]; then
			pargs=(-p "$COVERAGE_TEST_P")
		fi
		# SOAK_SKIP_BOLT keeps the 8-20 minute bbolt soak out of the gate the
		# same way #verify does; the soak has dedicated runs.
		if ! SOAK_SKIP_BOLT=1 CGO_ENABLED=1 go test "${paths[@]}" \
			-count=1 -timeout=30m -covermode=set \
			-coverprofile="$wd/merged.out" "${pargs[@]}" >"$wd/test.log" 2>&1; then
			echo "::error::go test failed during the coverage measurement; tail of $wd/test.log:" >&2
			tail -20 "$wd/test.log" >&2
			exit 1
		fi
	fi

	if ! go list -f '{{.ImportPath}}|{{.Dir}}|{{len .TestGoFiles}}|{{len .XTestGoFiles}}' \
		"${paths[@]}" >"$wd/packages.txt"; then
		die "go list failed; cannot attribute coverage to packages"
	fi
}

# attribute_packages maps every package to the module that owns it. Deepest
# module directory wins so nested modules are not double counted.
attribute_packages() {
	local wd="$1" mods_str
	mods_str="$(tr '\n' ' ' <"$wd/modules.txt")"
	awk -F'|' -v root="$PWD" -v mods_str="$mods_str" '
		BEGIN { n = split(mods_str, mods, " ") }
		{
			pkg = $1; dir = $2
			has = (($3 + $4) > 0) ? 1 : 0
			best = ""; bestlen = -1
			for (i = 1; i <= n; i++) {
				m = mods[i]
				if (dir == root "/" m || index(dir, root "/" m "/") == 1) {
					if (length(m) > bestlen) { best = m; bestlen = length(m) }
				}
			}
			printf "%s|%s|%d\n", pkg, best, has
		}' "$wd/packages.txt" >"$wd/pkgmap.txt"
}

# attribute_coverage sums profile statements per package. A profile entry is
#   <importpath>/<file>.go:<start>,<end> <numStmt> <count>
# and the package is the dirname of <importpath>/<file>.go.
attribute_coverage() {
	local wd="$1"
	awk -F'|' '
		NR == FNR { mod[$1] = $2; has[$1] = $3; next }
		/^mode:/ { next }
		{
			path = $0
			sub(/:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+ .*$/, "", path)
			i = length(path)
			while (i > 0 && substr(path, i, 1) != "/") i--
			pkg = substr(path, 1, i - 1)
			if (!(pkg in mod)) next
			split($0, f, " ")
			num = f[2] + 0
			cnt = f[3] + 0
			tot[pkg] += num
			if (cnt > 0) cov[pkg] += num
		}
		END {
			for (p in mod)
				printf "%s|%s|%d|%d|%d\n", p, mod[p], cov[p] + 0, tot[p] + 0, has[p]
		}' "$wd/pkgmap.txt" "$wd/merged.out" | sort >"$wd/pkg-coverage.txt"
}

# rollup_modules turns per-package rows into
#   module<TAB>pct<TAB>covered<TAB>total<TAB>packages<TAB>notest
# `pct` is `none` when the module contributes no statements to the profile
# (it has no test files at all) — that distinction is kept instead of a
# misleading 0.0, because "0 of 0 statements" is not zero coverage.
rollup_modules() {
	local wd="$1" mods_str
	mods_str="$(tr '\n' ' ' <"$wd/modules.txt")"
	awk -F'|' -v mods_str="$mods_str" '
		BEGIN { n = split(mods_str, mods, " ") }
		$2 != "" {
			m = $2
			tot[m] += $4
			cov[m] += $3
			np[m]++
			if ($5 == 0) nt[m]++
		}
		END {
			for (i = 1; i <= n; i++) {
				m = mods[i]
				t = tot[m] + 0
				c = cov[m] + 0
				pct = (t > 0) ? sprintf("%.1f", 100 * c / t) : "none"
				printf "%s\t%s\t%d\t%d\t%d\t%d\n", m, pct, c, t, np[m] + 0, nt[m] + 0
			}
		}' "$wd/pkg-coverage.txt" >"$wd/module-coverage.txt"
}

measure() {
	local wd="$1"
	collect "$wd"
	attribute_packages "$wd"
	attribute_coverage "$wd"
	rollup_modules "$wd"
}

# --- baseline format -------------------------------------------------------
# Comment lines start with '#'. Data rows are TAB separated:
#   module	pct	packages	notest	measured	mode
# `mode` is `aggregate` for a numeric floor or `none` for a module with no test
# statements at all (explicitly allowlisted, so an untested module is a
# recorded decision instead of an invisible gap).

baseline_rows() {
	[ -f "$1" ] || return 0
	awk -F'\t' '/^#/ || NF == 0 { next } { print $1 "|" $2 "|" $4 "|" $6 }' "$1"
}

write_baseline() {
	local wd="$1" out="$2" date
	date="$(date +%Y-%m-%d)"
	{
		printf '%s\n' \
			"# Coverage floors — one row per module in flake.nix testModules." \
			"# Generated by: bash scripts/check-coverage.sh --update" \
			"# Do NOT hand-edit: the gate fails on dangling or missing rows, and" \
			"# --update is the only sanctioned way to move a number (quiet window," \
			"# measured value quoted in the commit message)." \
			"#" \
			"# module	pct	packages	notest	measured	mode" \
			"# pct      = covered statements / total statements for the module" \
			"# packages = packages in the module; notest = those with no test files" \
			"# mode     = aggregate (numeric floor) | none (module has no test statements)"
		awk -F'\t' -v d="$date" '
			{
				mode = ($2 == "none") ? "none" : "aggregate"
				printf "%s\t%s\t%s\t%s\t%s\t%s\n", $1, $2, $5, $6, d, mode
			}' "$wd/module-coverage.txt" | sort
	} >"$out"
}

# run_gate compares a module-coverage file against a baseline.
# Args: WD BASELINE MODULES_FILE QUIET
# Exit: 0 when nothing regressed, 1 on any drift/dangling/unmeasured row.
run_gate() {
	local wd="$1" baseline="$2" modules_file="$3" quiet="$4"
	local bl="$wd/.baseline-join.txt" act="$wd/.actual-join.txt" out="$wd/.gate-rows.txt"
	baseline_rows "$baseline" >"$bl"
	awk -F'\t' '$1 != "" { print $1 "|" $2 "|" $6 }' "$wd/module-coverage.txt" >"$act"

	local rc=0
	awk -F'|' -v tol="$TOLERANCE" -v ltol="$LOWER_TOLERANCE" \
		-v mods_file="$modules_file" -v bl_file="$bl" '
		FILENAME == mods_file { mods[++nmod] = $1; inlist[$1] = 1; next }
		FILENAME == bl_file { bl[$1] = $2; bnotest[$1] = $3; bmode[$1] = $4; bseen[$1] = 1; next }
		{ act[$1] = $2; anotest[$1] = $3 }
		END {
			fail = 0
			for (i = 1; i <= nmod; i++) {
				m = mods[i]
				if (!(m in bseen)) {
					printf "0|%s|-|%s|MISSING|module has no baseline row (unmeasured surface)\n", m, act[m]
					fail = 1
					continue
				}
				status = "ok"; note = ""; a = act[m]; b = bl[m]
				aclass = (a == "none" || a == "") ? "none" : "num"
				if (bmode[m] == "none") {
					if (aclass == "num") { status = "STALE"; note = "module gained tests; refresh with --update" }
				} else if (aclass == "none") {
					status = "DRIFT"; note = "coverage data vanished (tests deleted or packages removed)"
				} else {
					d = a - b
					if (-d > ltol) { status = "DRIFT"; note = "regression beyond lower tolerance" }
					else if (d > tol) { status = "STALE"; note = "improved; refresh with --update" }
				}
				if (status != "DRIFT" && anotest[m] + 0 > bnotest[m] + 0) {
					status = "DRIFT"
					note = "packages without tests grew: " bnotest[m] " -> " anotest[m]
				}
				if (status == "DRIFT") fail = 1
				flag = (status == "ok") ? 1 : 0
				printf "%d|%s|%s|%s|%s|%s\n", flag, m, b, a, status, note
			}
			for (m in bseen)
				if (!(m in inlist)) {
					printf "0|%s|%s|-|DANGLING|baseline row resolves to no testModules entry\n", m, bl[m]
					fail = 1
				}
			exit fail
		}' "$modules_file" "$bl" "$act" >"$out" || rc=$?

	sort -t'|' -k1,1n -k2,2 "$out" |
		awk -F'|' -v quiet="$quiet" '
			$5 != "ok" { printf "%-44s %8s -> %-8s %-9s %s\n", $2, $3, $4, $5, $6; next }
			quiet == 0 { printf "%-44s %8s -> %-8s %-9s\n", $2, $3, $4, $5 }
		'
	return "$rc"
}

# --- self-test -------------------------------------------------------------
# Fault-injection proof that the compare logic can go red. The fixtures replay
# the production code path (module-coverage.txt + baseline + modules.txt), so a
# green self-test means the gate is wired to fail for the right reasons.
self_test() {
	local tmp failures=0 rc out
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064  # $tmp must expand now, not at trap time
	trap "rm -rf '$tmp'" EXIT

	_fixture() { # $1=dir $2=modules $3=module-coverage rows $4=baseline rows
		mkdir -p "$1"
		printf '%s\n' "$2" >"$1/modules.txt"
		printf '%s\n' "$3" >"$1/module-coverage.txt"
		{
			printf '# fixture baseline\n# module\tpct\tpackages\tnotest\tmeasured\tmode\n'
			printf '%s\n' "$4"
		} >"$1/baseline.txt"
	}

	_expect_green() { # $1=dir $2=label
		if ! run_gate "$1" "$1/baseline.txt" "$1/modules.txt" 1 >/dev/null; then
			echo "self-test FAIL: $2 must be green" >&2
			failures=$((failures + 1))
		fi
	}

	_expect_red() { # $1=dir $2=label $3=expected status
		rc=0
		out="$(run_gate "$1" "$1/baseline.txt" "$1/modules.txt" 1)" || rc=$?
		if [ "$rc" -eq 0 ]; then
			echo "self-test FAIL: $2 must be red" >&2
			failures=$((failures + 1))
		elif ! printf '%s' "$out" | grep -q "$3"; then
			echo "self-test FAIL: $2 must report $3, got: $out" >&2
			failures=$((failures + 1))
		fi
	}

	# Happy path: matching numbers.
	_fixture "$tmp/ok" "alpha" "alpha	80.0	8	10	1	0" "alpha	80.0	1	0	2026-10-26	aggregate"
	_expect_green "$tmp/ok" "matching numbers"

	# Regression: -3.0 points.
	_fixture "$tmp/drift" "alpha" "alpha	77.0	7	10	1	0" "alpha	80.0	1	0	2026-10-26	aggregate"
	_expect_red "$tmp/drift" "a 3-point regression" "DRIFT"

	# Improvement: green, labelled STALE.
	_fixture "$tmp/stale" "alpha" "alpha	90.0	9	10	1	0" "alpha	80.0	1	0	2026-10-26	aggregate"
	rc=0
	out="$(run_gate "$tmp/stale" "$tmp/stale/baseline.txt" "$tmp/stale/modules.txt" 1)" || rc=$?
	if [ "$rc" -ne 0 ] || ! printf '%s' "$out" | grep -q STALE; then
		echo "self-test FAIL: an improvement must be green and STALE" >&2
		failures=$((failures + 1))
	fi

	# Dangling baseline row: a deleted module.
	_fixture "$tmp/dangling" "alpha" "alpha	80.0	8	10	1	0" \
		"alpha	80.0	1	0	2026-10-26	aggregate
ghost	50.0	1	0	2026-10-26	aggregate"
	_expect_red "$tmp/dangling" "a dangling baseline row" "DANGLING"

	# Unmeasured module: in testModules, absent from the baseline.
	_fixture "$tmp/unmeasured" "alpha
beta" "alpha	80.0	8	10	1	0
beta	70.0	7	10	1	0" "alpha	80.0	1	0	2026-10-26	aggregate"
	_expect_red "$tmp/unmeasured" "an unmeasured module" "MISSING"

	# Untested-package growth: same percentage, more packages without tests.
	_fixture "$tmp/notest" "alpha" "alpha	80.0	8	10	3	1" "alpha	80.0	3	0	2026-10-26	aggregate"
	_expect_red "$tmp/notest" "packages without tests growing" "DRIFT"

	# Data vanished: module had numbers, now contributes no statements.
	_fixture "$tmp/vanished" "alpha" "alpha	none	0	0	1	1" "alpha	80.0	1	0	2026-10-26	aggregate"
	_expect_red "$tmp/vanished" "vanished coverage data" "DRIFT"

	# Explicitly untested module stays green (a recorded decision, not a gap).
	_fixture "$tmp/none" "alpha" "alpha	none	0	0	1	1" "alpha	none	1	1	2026-10-26	none"
	_expect_green "$tmp/none" "an allowlisted untested module"

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-coverage self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-coverage self-test: 8/8 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

measure "$WORKDIR"

if [ "$MODE" = "update" ]; then
	write_baseline "$WORKDIR" "$BASELINE"
	echo "✓ Wrote $(grep -vc '^#' "$BASELINE") module floors to $BASELINE"
	echo "  Quote the measured numbers in the commit message; do not hand-edit rows."
	printf '%-44s %8s %10s %8s %8s\n' MODULE COVERED TOTAL PCT NOTEST
	awk -F'\t' '{ printf "%-44s %8d %10d %8s %8d\n", $1, $3, $4, $2, $6 }' "$WORKDIR/module-coverage.txt"
	exit 0
fi

if [ -n "$REPORT_DIR" ]; then
	mkdir -p "$REPORT_DIR"
	cp "$WORKDIR/module-coverage.txt" "$REPORT_DIR/module-coverage.tsv"
	cp "$WORKDIR/pkg-coverage.txt" "$REPORT_DIR/pkg-coverage.tsv"
	{
		printf '| Module | Coverage | Statements | Packages | Without tests |\n'
		printf '| --- | ---: | ---: | ---: | ---: |\n'
		awk -F'\t' '{ printf "| `%s` | %s%% | %d/%d | %d | %d |\n", $1, $2, $3, $4, $5, $6 }' \
			"$WORKDIR/module-coverage.txt"
	} >"$REPORT_DIR/coverage-report.md"
	echo "==> wrote $REPORT_DIR/{module,pkg}-coverage.tsv and coverage-report.md" >&2
fi

printf '%-44s %8s %10s %-9s %s\n' MODULE BASELINE ACTUAL STATUS NOTE
printf '%-44s %8s %10s %-9s %s\n' ------ -------- ------ ------ ----
if run_gate "$WORKDIR" "$BASELINE" "$WORKDIR/modules.txt" "$QUIET"; then
	echo ""
	echo "✓ All $(grep -vc '^#' "$BASELINE") module floors hold (+${TOLERANCE}% / -${LOWER_TOLERANCE}%)."
	exit 0
fi
echo ""
echo "::error::coverage drifted beyond +${TOLERANCE}%/-${LOWER_TOLERANCE}% tolerance."
echo "Regressions: fix the tests. Intentional re-baseline: bash scripts/check-coverage.sh --update"
exit 1
