#!/usr/bin/env bash
# check-coverage.sh — coverage FLOORS for every module and every package.
#
# Why this exists: before 2026-10-26 this gate pinned 11 of 89 test modules,
# so 78 modules could lose their tests entirely and nothing went red. A gate
# that measures 12% of the surface is a claim, not a check. The floors live in
# scripts/coverage-baseline.txt (one row per module in flake.nix `testModules`)
# and scripts/coverage-package-baseline.txt (one row per package), and the gate
# fails on:
#
#   * a module/package regression beyond the lower tolerance,
#   * a module/package that has no baseline row (unmeasured surface),
#   * a baseline row that no longer resolves to a module/package (dangling key),
#   * a package that lost its tests (the module percentage can hold while a
#     package rots, because the module denominator grows with the rest),
#   * coverage data vanishing for a module that had some (tests deleted).
#
# Upward drift beyond the tolerance is reported as STALE but does not fail:
# new tests are never a defect, and a force-refreshed baseline would train
# people to re-pin until green.
#
# Usage: bash scripts/check-coverage.sh [flags]
#   (default)         measure, compare modules AND packages, exit non-zero on drift
#   --update          re-measure and REWRITE both baseline files
#   --profile FILE    consume an existing merged coverprofile instead of running tests
#   --report DIR      also write TSV/markdown artifacts into DIR
#   --quiet           print only rows that are not ok
#   --self-test       fault-inject the compare logic against fixtures (no Go needed)
#   --help
#
# Environment:
#   COVERAGE_BASELINE          module baseline (default scripts/coverage-baseline.txt)
#   COVERAGE_PACKAGE_BASELINE  package baseline (default scripts/coverage-package-baseline.txt)
#   COVERAGE_WORKDIR           scratch dir (default build/coverage)
#   COVERAGE_TOLERANCE         module upward tolerance in points (default 2.0)
#   COVERAGE_PACKAGE_TOLERANCE package upward tolerance (default 1.0)
#   COVERAGE_LOWER_TOLERANCE   override the lower tolerance for both
#   COVERAGE_TEST_P            go test -p value for the measurement run (optional)
#
# Measurement mode is `aggregate`: one workspace-wide test run emits a merged
# coverprofile, and a module's percentage is covered/total STATEMENTS for the
# files it owns (deepest module directory wins, so nested modules such as
# metaengine/pgengine are never double counted). The pre-2026-10-26 version
# took the FIRST package's coverage line, which in a multi-package module is an
# arbitrary package, not the module.
#
# The measurement is load-sensitive (timeout-skipped tests measure 3-5 points
# lower on a loaded box, deterministically on GitHub runners), so the baselines
# are pinned from quiet-window runs and CI widens only the LOWER bound. That
# also makes the 1-point PACKAGE rule a local/quiet-window rule: on CI a skipped
# test in a tiny package moves the number further than that.

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
PKG_BASELINE="${COVERAGE_PACKAGE_BASELINE:-scripts/coverage-package-baseline.txt}"
WORKDIR="${COVERAGE_WORKDIR:-build/coverage}"
TOLERANCE="${COVERAGE_TOLERANCE:-2.0}"
PKG_TOLERANCE="${COVERAGE_PACKAGE_TOLERANCE:-1.0}"
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
		# In-place reuse (the profile already IS $wd/merged.out) must not cp
		# onto itself.
		if [ ! "$PROFILE_IN" -ef "$wd/merged.out" ]; then
			cp "$PROFILE_IN" "$wd/merged.out"
		fi
	else
		echo "==> Measuring coverage across $(wc -l <"$wd/modules.txt") modules (this is the test run)" >&2
		# SOAK_SKIP_BOLT keeps the 8-20 minute bbolt soak out of the gate the
		# same way #verify does; the soak has dedicated runs.
		#
		# A failing test does NOT abort the measurement: coverage is a
		# measurement, and test success is already gated by #test and
		# #test-race (the pre-2026-10-26 script tolerated failures for the
		# same reason). Load-sensitive suites — benchkit's 90s journey budget
		# is the documented example — would otherwise turn the coverage gate
		# into a second, flakier test gate. The warning is loud so a suite
		# that is systematically broken cannot hide behind a green number.
		local -a pargs=()
		if [ -n "${COVERAGE_TEST_P:-}" ]; then
			pargs=(-p "$COVERAGE_TEST_P")
		fi
		if ! SOAK_SKIP_BOLT=1 CGO_ENABLED=1 go test "${paths[@]}" \
			-count=1 -timeout=30m -covermode=set \
			-coverprofile="$wd/merged.out" "${pargs[@]}" >"$wd/test.log" 2>&1; then
			local failed
			failed="$(grep -c '^--- FAIL' "$wd/test.log" || true)"
			echo "::warning::$failed test(s) failed during the coverage measurement; coverage is still measured (test success is gated by #test / #test-race):" >&2
			grep '^--- FAIL' "$wd/test.log" | head -20 >&2
		fi
		[ -s "$wd/merged.out" ] || die "go test produced no coverprofile — see $wd/test.log"
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
# and the package is the dirname of <importpath>/<file>.go. Every attributed
# package gets a row, including ones that contribute no statements (pct `none`),
# so "this package lost its tests" is visible instead of absent.
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
				if (mod[p] != "")
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

# --- floor files -----------------------------------------------------------
# Both baseline files are TAB separated data rows after a '#' comment header:
#   module	pct	packages	notest	measured	mode
#   package	pct	notest			measured	mode
# `mode` is `aggregate` for a numeric floor, `none` for a unit with no test
# statements at all (an explicit record instead of an invisible gap).

module_actual() { # wd -> key|pct|notest
	awk -F'\t' '$1 != "" { print $1 "|" $2 "|" $6 }' "$1/module-coverage.txt"
}

package_actual() { # wd -> key|pct|notest
	awk -F'|' '$2 != "" {
		pct = ($4 + 0 > 0) ? sprintf("%.1f", 100 * $3 / $4) : "none"
		printf "%s|%s|%d\n", $1, pct, ($5 + 0 == 0) ? 1 : 0
	}' "$1/pkg-coverage.txt"
}

module_baseline_rows() { # file -> key|pct|notest|mode
	awk -F'\t' '/^#/ || NF == 0 { next } { print $1 "|" $2 "|" $4 "|" $6 }' "$1"
}

package_baseline_rows() { # file -> key|pct|notest|mode
	awk -F'\t' '/^#/ || NF == 0 { next } { print $1 "|" $2 "|" $3 "|" $5 }' "$1"
}

write_module_baseline() {
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

write_package_baseline() {
	local wd="$1" out="$2" date
	date="$(date +%Y-%m-%d)"
	{
		printf '%s\n' \
			"# Package-level coverage floors — one row per package in a testModules module." \
			"# Generated by: bash scripts/check-coverage.sh --update" \
			"# Do NOT hand-edit. A module floor cannot see a single rotting package," \
			"# because the module denominator grows with the rest of the code." \
			"#" \
			"# package	pct	notest	measured	mode" \
			"# notest  = 1 when the package has no test files at all" \
			"# mode    = aggregate (numeric floor) | none (no test statements)"
		awk -F'|' -v d="$date" '
			{
				has = $5 + 0
				pct = ($4 + 0 > 0) ? sprintf("%.1f", 100 * $3 / $4) : "none"
				mode = (pct == "none") ? "none" : "aggregate"
				printf "%s\t%s\t%d\t%s\t%s\n", $1, pct, (has == 0) ? 1 : 0, d, mode
			}' "$wd/pkg-coverage.txt" | sort
	} >"$out"
}

# compare_floors joins a key list, a prepared baseline and prepared actuals.
# Args: KEYS PREPARED_BASELINE PREPARED_ACTUAL QUIET UP_TOL LOWER_TOL LABEL
# Exit: 0 when nothing regressed, 1 on any drift/dangling/unmeasured row.
compare_floors() {
	local keys="$1" bl_in="$2" act_in="$3" quiet="$4" tol="$5" ltol="$6" label="$7"
	local out rc=0
	out="$(mktemp)"
	awk -F'|' -v tol="$tol" -v ltol="$ltol" -v label="$label" \
		-v keys_file="$keys" -v bl_file="$bl_in" '
		FILENAME == keys_file { keys[++n] = $1; inlist[$1] = 1; next }
		FILENAME == bl_file { bl[$1] = $2; bnotest[$1] = $3; bmode[$1] = $4; bseen[$1] = 1; next }
		{ act[$1] = $2; anotest[$1] = $3 }
		END {
			fail = 0
			for (i = 1; i <= n; i++) {
				k = keys[i]
				if (!(k in bseen)) {
					printf "0|%s|%s|-|%s|MISSING|no baseline row (unmeasured surface); add it with --update\n", label, k, act[k]
					fail = 1
					continue
				}
				status = "ok"; note = ""; a = act[k]; b = bl[k]
				aclass = (a == "none" || a == "") ? "none" : "num"
				if (bmode[k] == "none") {
					if (aclass == "num") { status = "STALE"; note = "gained tests; refresh with --update" }
				} else if (aclass == "none") {
					status = "DRIFT"; note = "coverage data vanished (tests deleted or code removed)"
				} else {
					d = a - b
					if (-d > ltol) { status = "DRIFT"; note = "regression beyond lower tolerance" }
					else if (d > tol) { status = "STALE"; note = "improved; refresh with --update" }
				}
				if (status != "DRIFT" && anotest[k] + 0 > bnotest[k] + 0) {
					status = "DRIFT"
					note = "lost its tests (notest " bnotest[k] " -> " anotest[k] ")"
				}
				if (status == "DRIFT") fail = 1
				flag = (status == "ok") ? 1 : 0
				printf "%d|%s|%s|%s|%s|%s|%s\n", flag, label, k, b, a, status, note
			}
			for (k in bseen)
				if (!(k in inlist)) {
					printf "0|%s|%s|%s|-|DANGLING|baseline row resolves to nothing\n", label, k, bl[k]
					fail = 1
				}
			exit fail
		}' "$keys" "$bl_in" "$act_in" >"$out" || rc=$?

	sort -t'|' -k1,1n -k3,3 "$out" |
		awk -F'|' -v quiet="$quiet" '
			$6 != "ok" { printf "%-7s %-58s %6s %6s  %-9s %s\n", $2, $3, $4, $5, $6, $7; next }
			quiet == 0 { printf "%-7s %-58s %6s %6s  %-9s\n", $2, $3, $4, $5, $6 }
		'
	rm -f "$out"
	return "$rc"
}

# --- self-test -------------------------------------------------------------
# Fault-injection proof that the compare logic can go red. The fixtures replay
# the production code path (keys + baseline + actuals), so a green self-test
# means the gate is wired to fail for the right reasons.
self_test() {
	local tmp failures=0 rc out
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064  # $tmp must expand now, not at trap time
	trap "rm -rf '$tmp'" EXIT

	_fixture() { # $1=dir $2=keys $3=actual rows $4=baseline rows
		mkdir -p "$1"
		printf '%s\n' "$2" >"$1/keys.txt"
		printf '%s\n' "$3" >"$1/actual.txt"
		printf '%s\n' "$4" >"$1/baseline.txt"
	}

	_gate() { # $1=dir $2=up tolerance $3=lower tolerance
		compare_floors "$1/keys.txt" "$1/baseline.txt" "$1/actual.txt" 1 "$2" "$3" unit
	}

	_expect_green() { # $1=dir $2=label
		if ! _gate "$1" 0.5 0.5 >/dev/null; then
			echo "self-test FAIL: $2 must be green" >&2
			failures=$((failures + 1))
		fi
	}

	_expect_red() { # $1=dir $2=label $3=expected status
		rc=0
		out="$(_gate "$1" 0.5 0.5)" || rc=$?
		if [ "$rc" -eq 0 ]; then
			echo "self-test FAIL: $2 must be red" >&2
			failures=$((failures + 1))
		elif ! printf '%s' "$out" | grep -q "$3"; then
			echo "self-test FAIL: $2 must report $3, got: $out" >&2
			failures=$((failures + 1))
		fi
	}

	# Happy path: matching numbers.
	_fixture "$tmp/ok" "alpha" "alpha|80.0|0" "alpha|80.0|0|aggregate"
	_expect_green "$tmp/ok" "matching numbers"

	# Regression beyond the lower tolerance.
	_fixture "$tmp/drift" "alpha" "alpha|77.0|0" "alpha|80.0|0|aggregate"
	_expect_red "$tmp/drift" "a 3-point regression" "DRIFT"

	# Improvement: green, labelled STALE.
	_fixture "$tmp/stale" "alpha" "alpha|90.0|0" "alpha|80.0|0|aggregate"
	rc=0
	out="$(_gate "$tmp/stale" 0.5 0.5)" || rc=$?
	if [ "$rc" -ne 0 ] || ! printf '%s' "$out" | grep -q STALE; then
		echo "self-test FAIL: an improvement must be green and STALE" >&2
		failures=$((failures + 1))
	fi

	# Dangling baseline row: a deleted unit.
	_fixture "$tmp/dangling" "alpha" "alpha|80.0|0" \
		"alpha|80.0|0|aggregate
ghost|50.0|0|aggregate"
	_expect_red "$tmp/dangling" "a dangling baseline row" "DANGLING"

	# Unmeasured unit: present in the tree, absent from the baseline.
	_fixture "$tmp/unmeasured" "alpha
beta" "alpha|80.0|0
beta|70.0|0" "alpha|80.0|0|aggregate"
	_expect_red "$tmp/unmeasured" "an unmeasured unit" "MISSING"

	# A package that lost its tests while the percentage held.
	_fixture "$tmp/notest" "alpha" "alpha|80.0|1" "alpha|80.0|0|aggregate"
	_expect_red "$tmp/notest" "a unit that lost its tests" "DRIFT"

	# Data vanished: unit had numbers, now contributes no statements.
	_fixture "$tmp/vanished" "alpha" "alpha|none|1" "alpha|80.0|0|aggregate"
	_expect_red "$tmp/vanished" "vanished coverage data" "DRIFT"

	# Explicitly untested unit stays green (a recorded decision, not a gap).
	_fixture "$tmp/none" "alpha" "alpha|none|1" "alpha|none|1|none"
	_expect_green "$tmp/none" "an allowlisted untested unit"

	# Tolerance is actually read: the same -3.0 drop is green at -6.0.
	_fixture "$tmp/ci" "alpha" "alpha|77.0|0" "alpha|80.0|0|aggregate"
	if ! _gate "$tmp/ci" 0.5 6.0 >/dev/null; then
		echo "self-test FAIL: the lower tolerance must widen the allowance on CI" >&2
		failures=$((failures + 1))
	fi

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-coverage self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-coverage self-test: 9/9 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

measure "$WORKDIR"

if [ "$MODE" = "update" ]; then
	write_module_baseline "$WORKDIR" "$BASELINE"
	write_package_baseline "$WORKDIR" "$PKG_BASELINE"
	echo "✓ Wrote $(grep -vc '^#' "$BASELINE") module floors to $BASELINE"
	echo "✓ Wrote $(grep -vc '^#' "$PKG_BASELINE") package floors to $PKG_BASELINE"
	echo "  Quote the measured numbers in the commit message; do not hand-edit rows."
	printf '%-52s %8s %10s %8s %8s\n' MODULE COVERED TOTAL PCT NOTEST
	awk -F'\t' '{ printf "%-52s %8d %10d %8s %8d\n", $1, $3, $4, $2, $6 }' "$WORKDIR/module-coverage.txt"
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

[ -f "$BASELINE" ] || die "module baseline '$BASELINE' is missing — run: bash scripts/check-coverage.sh --update"
[ -f "$PKG_BASELINE" ] || die "package baseline '$PKG_BASELINE' is missing — run: bash scripts/check-coverage.sh --update"

bl_tmp="$(mktemp)"
pkgbl_tmp="$(mktemp)"
act_tmp="$(mktemp)"
pkgact_tmp="$(mktemp)"
pkgkeys_tmp="$(mktemp)"
trap 'rm -f "$bl_tmp" "$pkgbl_tmp" "$act_tmp" "$pkgact_tmp" "$pkgkeys_tmp"' EXIT

module_baseline_rows "$BASELINE" >"$bl_tmp"
package_baseline_rows "$PKG_BASELINE" >"$pkgbl_tmp"
module_actual "$WORKDIR" >"$act_tmp"
package_actual "$WORKDIR" >"$pkgact_tmp"
package_actual "$WORKDIR" | cut -d'|' -f1 >"$pkgkeys_tmp"

printf '%-7s %-58s %6s %6s  %-9s %s\n' SCOPE UNIT BASELINE ACTUAL STATUS NOTE
printf '%-7s %-58s %6s %6s  %-9s %s\n' ----- ---- -------- ------ ------ ----

status=0
compare_floors "$WORKDIR/modules.txt" "$bl_tmp" "$act_tmp" "$QUIET" \
	"$TOLERANCE" "$LOWER_TOLERANCE" module || status=1
compare_floors "$pkgkeys_tmp" "$pkgbl_tmp" "$pkgact_tmp" "$QUIET" \
	"$PKG_TOLERANCE" "$LOWER_TOLERANCE" package || status=1

if [ "$status" -eq 0 ]; then
	echo ""
	echo "✓ All $(grep -vc '^#' "$BASELINE") module and $(grep -vc '^#' "$PKG_BASELINE") package floors hold (+${TOLERANCE}%/${PKG_TOLERANCE}% / -${LOWER_TOLERANCE}%)."
	exit 0
fi
echo ""
echo "::error::coverage drifted beyond the tolerances (+${TOLERANCE}% module / +${PKG_TOLERANCE}% package, -${LOWER_TOLERANCE}%)."
echo "Regressions: fix the tests. Intentional re-baseline: bash scripts/check-coverage.sh --update"
exit 1
