#!/usr/bin/env bash
# check-test-matrix.sh — the test matrix must be COMPLETE, never curated.
#
# Why: `#test` and `#test-race` cover "all modules" only as long as their
# GT_PATHS keeps pointing at ${modulePaths} and modulePaths keeps deriving from
# testModules. Both links are one edit away from silent narrowing — a
# hand-picked path list, a module dropped "temporarily", a lintModules list that
# quietly stops matching. None of those break anything visible: the suite just
# covers less, and every gate downstream (coverage floors, races, shuffles)
# keeps passing over the smaller set.
#
# Checks (all mechanical, no builds):
#   A. #test       GT_PATHS is ${modulePaths}  (every module, not a subset)
#   B. #test-race  GT_PATHS is ${modulePaths} AND sets GT_RACE=1
#   C. modulePaths is derived from testModules
#   D. lintModules == testModules
#   E. repeatModules is a SUBSET of testModules (a repeat leg over a deleted
#      module silently tests nothing)
#
# Usage: bash scripts/check-test-matrix.sh [flags]
#   --flake FILE  inspect this flake (default flake.nix)
#   --quiet       only the verdict
#   --self-test   fault-inject narrowed/correct matrices
#   --help

set -euo pipefail
export LC_ALL=C

cd "$(git rev-parse --show-toplevel)"

FLAKE="flake.nix"
QUIET=0
MODE=gate

die() {
	echo "::error::$1" >&2
	exit 1
}

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
	--flake)
		shift
		[ $# -gt 0 ] || die "--flake requires a file"
		FLAKE="$1"
		;;
	--quiet) QUIET=1 ;;
	--self-test) MODE=selftest ;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown flag: $1 (try --help)" ;;
	esac
	shift
done

# app_body FLAKE NAME — the lines of `NAME = mkApp ...` (or `mkApp "NAME"`),
# up to its closing `'';`. Printed with leading whitespace stripped.
app_text() { # $1=flake $2=app name
	awk -v want="$2" '
		!inapp && ($0 ~ ("mkApp \"" want "\"") || $0 ~ ("^[[:space:]]*" want "[[:space:]]*=")) {
			inapp = 1
		}
		inapp { print }
		inapp && /^[[:space:]]*\.\.\./ { }
		inapp && /'"'"';[[:space:]]*$/ { exit }
	' "$1"
}

# audit FLAKE — print "CODE<TAB>detail" findings; empty output means complete.
audit() { # $1=flake
	local mods=""
	# C: modulePaths must be built from testModules.
	if ! grep -qE 'modulePaths[[:space:]]*=[^;]*testModules' "$1"; then
		printf 'C\tmodulePaths no longer derives from testModules\n'
	fi
	# D: lintModules must equal testModules (not a hand-written subset).
	if ! grep -qE 'lintModules[[:space:]]*=[[:space:]]*testModules' "$1"; then
		printf 'D\tlintModules is not testModules\n'
	fi
	# A: the plain suite covers every module.
	app_text "$1" test | grep -q 'GT_PATHS="\${modulePaths}"' ||
		printf 'A\t#test GT_PATHS is not ${modulePaths}\n'
	# B: the race suite covers every module AND actually enables -race.
	app_text "$1" test-race | grep -q 'GT_PATHS="\${modulePaths}"' ||
		printf 'B\t#test-race GT_PATHS is not ${modulePaths}\n'
	app_text "$1" test-race | grep -q 'GT_RACE=1' ||
		printf 'B\t#test-race does not set GT_RACE=1\n'
	# E: every repeat module still exists in testModules.
	mods="$(sed -n '/testModules = \[/,/^[[:space:]]*\];/p' "$1" | grep -oE '"[^"]+"' | tr -d '"' | sort -u)"
	local missing=""
	local m
	while IFS= read -r m; do
		[ -n "$m" ] || continue
		printf '%s\n' "$mods" | grep -qx "$m" || missing="$missing $m"
	done < <(sed -n '/repeatModules = map/,/^[[:space:]]*\];/p' "$1" | grep -oE '"[^"]+"' | tr -d '"' |
		grep -vF '${m}' | sort -u)
	if [ -n "$missing" ]; then
		printf 'E\trepeatModules names modules absent from testModules:%s\n' "$missing"
	fi
}

self_test() {
	local tmp failures=0 findings
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT

	cat >"$tmp/good.nix" <<'EOF'
          testModules = [
            "event"
            "command"
          ];
          repeatModules = map (m: "./${m}/...") [
            "event"
          ];
          lintModules = testModules;
          modulePaths = map (m: "./${m}/...") testModules;
            test = mkApp "test" goModules ''
              export GT_LABEL=test GT_PATHS="${modulePaths}"
            '';
            test-race = mkApp "test-race" goModules ''
              export GT_LABEL=test-race GT_RACE=1 GT_PATHS="${modulePaths}"
            '';
EOF
	if [ -n "$(audit "$tmp/good.nix")" ]; then
		echo "self-test FAIL: a complete matrix must produce no findings, got:" >&2
		audit "$tmp/good.nix" >&2
		failures=$((failures + 1))
	fi

	# Fault 1: the race leg narrowed to one hand-picked path.
	sed 's|export GT_LABEL=test-race GT_RACE=1 GT_PATHS="${modulePaths}"|export GT_LABEL=test-race GT_RACE=1 GT_PATHS="./event/..."|' \
		"$tmp/good.nix" >"$tmp/narrow-race.nix"
	findings="$(audit "$tmp/narrow-race.nix")"
	printf '%s\n' "$findings" | grep -q '^B	#test-race GT_PATHS' || {
		echo "self-test FAIL: a narrowed race matrix must be caught" >&2
		printf '%s\n' "$findings" >&2
		failures=$((failures + 1))
	}

	# Fault 2: the race leg forgot -race.
	sed 's| GT_RACE=1||' "$tmp/good.nix" >"$tmp/no-race.nix"
	findings="$(audit "$tmp/no-race.nix")"
	printf '%s\n' "$findings" | grep -q '^B	#test-race does not set GT_RACE=1' || {
		echo "self-test FAIL: a race leg without GT_RACE=1 must be caught" >&2
		failures=$((failures + 1))
	}

	# Fault 3: modulePaths hand-written instead of derived.
	sed 's|modulePaths = map (m: "./${m}/...") testModules;|modulePaths = [ "./event/..." ];|' \
		"$tmp/good.nix" >"$tmp/handwritten.nix"
	findings="$(audit "$tmp/handwritten.nix")"
	printf '%s\n' "$findings" | grep -q '^C	' || {
		echo "self-test FAIL: a hand-written modulePaths must be caught" >&2
		failures=$((failures + 1))
	}

	# Fault 4: lintModules narrowed.
	sed 's|lintModules = testModules;|lintModules = [ "event" ];|' "$tmp/good.nix" >"$tmp/lint.nix"
	findings="$(audit "$tmp/lint.nix")"
	printf '%s\n' "$findings" | grep -q '^D	' || {
		echo "self-test FAIL: a narrowed lint set must be caught" >&2
		failures=$((failures + 1))
	}

	# Fault 5: repeatModules names a module that no longer exists.
	cat >"$tmp/repeat.nix" <<'EOF2'
          testModules = [
            "event"
            "command"
          ];
          repeatModules = map (m: "./${m}/...") [
            "event"
            "deleted-module"
          ];
          lintModules = testModules;
          modulePaths = map (m: "./${m}/...") testModules;
            test = mkApp "test" goModules ''
              export GT_LABEL=test GT_PATHS="${modulePaths}"
            '';
            test-race = mkApp "test-race" goModules ''
              export GT_LABEL=test-race GT_RACE=1 GT_PATHS="${modulePaths}"
            '';
EOF2
	findings="$(audit "$tmp/repeat.nix")"
	printf '%s\n' "$findings" | grep -q '^E	' || {
		echo "self-test FAIL: a repeat module absent from testModules must be caught" >&2
		printf '%s\n' "$findings" >&2
		failures=$((failures + 1))
	}

	if [ "$failures" -gt 0 ]; then
		echo "::error::check-test-matrix self-test: $failures case(s) failed" >&2
		exit 1
	fi
	echo "✓ check-test-matrix self-test: 6/6 fault-injection cases behave"
}

if [ "$MODE" = "selftest" ]; then
	self_test
	exit 0
fi

[ -f "$FLAKE" ] || die "--flake '$FLAKE' does not exist"
findings="$(audit "$FLAKE")"

if [ -z "$findings" ]; then
	[ "$QUIET" -eq 1 ] && echo "✓ test matrix complete" && exit 0
	echo "✓ test matrix is complete: #test and #test-race cover every module,"
	echo "  both derive from testModules, lint covers the same set, and every"
	echo "  repeat module still exists."
	exit 0
fi

echo "::error::the test matrix is no longer complete:"
printf '%s\n' "$findings" | sed 's/^/::error::  /'
echo ""
echo "A curated matrix passes every downstream gate over a SMALLER set —"
echo "that is the failure mode this check exists for. Restore the derivation"
echo "from testModules rather than listing paths by hand."
exit 1
