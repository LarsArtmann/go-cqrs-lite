#!/usr/bin/env bash
# check-adr-numbering.sh — ADR numbering integrity gate.
#
# FAILS on:
#   - duplicate ADR numbers (any files on disk — tracked OR untracked; this is
#     the parallel-session collision class that produced the double-0155 on
#     2026-10-10)
#   - index lockstep drift vs docs/adr/README.md:
#       * an ADR file without exactly one index row (matched by link target)
#       * an index row whose link target does not exist
#       * index title != the file's H1 title (both minus the "ADR-NNNN:" prefix)
# WARNS (exit 0) on:
#   - numbering gaps outside the documented never-assigned set (README note);
#     gaps are history, not errors — but undocumented ones deserve a note.
#
# Usage:
#   scripts/check-adr-numbering.sh [--adr-dir DIR]   # gate (default docs/adr)
#   scripts/check-adr-numbering.sh --self-test       # fixture mutations, no repo writes
#
# The filesystem scan (not `git ls-files`) is deliberate: an untracked ADR from
# a parallel session is exactly the collision this gate must catch pre-commit.
set -euo pipefail

ADR_DIR="docs/adr"
# Documented in docs/adr/README.md: "ADRs 0036, 0041, and 0138 were never
# assigned (gaps in numbering)."
NEVER_ASSIGNED="0036 0041 0138"
# Historical suffix-numbered ADR (addendum to 0099); renaming would break
# links from frozen archived status reports.
SUFFIX_NUMBERS="0099a"

fail() {
	echo "FAIL: $*" >&2
	exit 1
}
info() { echo "check-adr-numbering: $*"; }

usage() {
	grep '^#' "$0" | sed 's/^# \{0,1\}//' | tail -n +2
	exit 0
}

while [ $# -gt 0 ]; do
	case "$1" in
	--adr-dir)
		ADR_DIR="${2:?}"
		shift 2
		;;
	--self-test)
		SELF_TEST=1
		shift
		;;
	-h | --help) usage ;;
	*) fail "unknown flag: $1" ;;
	esac
done

# run_checks <adr-dir> — prints "WARN: ..." lines for gaps, exits nonzero on
# any hard violation. Operates on the given directory so --self-test can point
# it at fixtures.
run_checks() {
	local dir="$1"
	local index="$dir/README.md"
	[ -f "$index" ] || {
		echo "FAIL: no index at $index" >&2
		return 1
	}

	local files numbers dups strays
	files=$(cd "$dir" && ls [0-9][0-9][0-9][0-9]-*.md [0-9][0-9][0-9][0-9][a-z]-*.md 2>/dev/null | sort -u || true)
	[ -n "$files" ] || {
		echo "FAIL: no ADR files found in $dir" >&2
		return 1
	}

	# 0. Non-conforming filenames: an ADR that escapes the NNNN[-a]- prefix
	#    escapes this gate entirely (the 0099a class found on first run).
	strays=$(cd "$dir" && ls *.md 2>/dev/null | grep -vxE 'README\.md|[0-9]{4}([a-z])?-[A-Za-z0-9._-]+' || true)
	if [ -n "$strays" ]; then
		echo "FAIL: non-conforming ADR filename(s) in $dir (want NNNN-slug.md): $strays" >&2
		return 1
	fi

	# 1. Duplicate numbers (filesystem-wide: tracked + untracked).
	numbers=$(printf '%s\n' "$files" | sed 's/-.*//' | sort)
	dups=$(printf '%s\n' "$numbers" | uniq -d)
	if [ -n "$dups" ]; then
		echo "FAIL: duplicate ADR number(s): $dups" \
			"(files: $(printf '%s\n' "$files" | grep -E "^($(printf '%s' "$dups" | tr '\n' '|'))" | tr '\n' ' '))" >&2
		return 1
	fi

	# 2. Index lockstep: every file has exactly one row (matched by link
	#    target basename); every row's target exists; titles match.
	local f num h1_title rows_for target index_title
	while IFS= read -r f; do
		num=${f%%-*}
		rows_for=$(grep -cE "\| \[$num\]\($f\) +\|" "$index" || true)
		if [ "$rows_for" -eq 0 ]; then
			echo "FAIL: $dir/$f has no index row in README.md" >&2
			return 1
		fi
		if [ "$rows_for" -gt 1 ]; then
			echo "FAIL: $dir/$f has $rows_for index rows (want exactly 1)" >&2
			return 1
		fi
		# Tolerant H1 extraction: historical ADRs use "# ADR-NNNN: T",
		# "# ADR NNNN: T", "# ADR-NNNNa: T", and bare "# T".
		h1_title=$(head -1 "$dir/$f" | sed 's/^# //' | sed 's/^ADR-\{0,1\} \{0,1\}[0-9]\{4\}[a-z]\{0,1\}: //' | sed 's/^ *//;s/ *$//')
		index_title=$(grep -E "\| \[$num\]\($f\) +\|" "$index" | head -1 | awk -F'|' '{print $3}' | sed 's/^ \+\| \+$//g')
		if [ -z "$h1_title" ]; then
			echo "FAIL: $dir/$f H1 does not start with '# ' (or first line is empty)" >&2
			return 1
		fi
		if [ "$h1_title" != "$index_title" ]; then
			echo "FAIL: title mismatch for $f:" >&2
			echo "  file H1:  $h1_title" >&2
			echo "  index:    $index_title" >&2
			return 1
		fi
	done <<<"$files"

	while IFS= read -r target; do
		[ -n "$target" ] || continue
		if [ ! -f "$dir/$target" ]; then
			echo "FAIL: index row links to missing file: $target" >&2
			return 1
		fi
	done < <(grep -oE '\| \[[0-9]{4}\]\([^)]+\)' "$index" | sed -E 's/.*\(([^)]+)\)/\1/')

	# 3. Gaps: warn-only, outside the documented never-assigned set. Gap
	#    arithmetic uses the 4-digit part (0099a counts as 0099).
	local max missing gap undocumented=0
	max=$(printf '%s\n' "$numbers" | cut -c1-4 | sort -n | tail -1)
	missing=$(comm -23 <(seq -f '%04g' 1 "$max") <(printf '%s\n' "$numbers" | cut -c1-4 | sort -u))
	for gap in $missing; do
		if printf '%s\n' $NEVER_ASSIGNED | grep -qxF "$gap"; then
			continue
		fi
		echo "WARN: numbering gap at $gap is not in the documented never-assigned set (update the README note or the NEVER_ASSIGNED list in scripts/check-adr-numbering.sh)"
		undocumented=1
	done
	[ "$undocumented" -eq 1 ] && info "gaps noted above are warnings only"
	return 0
}

self_test() {
	local tmp fixture
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	mk_fixture() {
		fixture="$tmp/$1"
		mkdir -p "$fixture"
		printf '# ADR-0001: First Decision\n' >"$fixture/0001-first.md"
		printf '# ADR-0002: Second Decision\n' >"$fixture/0002-second.md"
		{
			printf '| ADR | Title | Date | Status |\n'
			printf '| --- | --- | --- | --- |\n'
			printf '| [0001](0001-first.md) | First Decision | 2026-01-01 | Accepted |\n'
			printf '| [0002](0002-second.md) | Second Decision | 2026-01-02 | Proposed |\n'
		} >"$fixture/README.md"
	}

	expect_ok() {
		if run_checks "$1" >/dev/null 2>"$tmp/err"; then
			echo "  PASS: $2"
		else
			echo "  FAIL(self-test): expected clean fixture to pass: $2" >&2
			cat "$tmp/err" >&2
			return 1
		fi
	}

	expect_fail() {
		local want="$3"
		if run_checks "$1" >"$tmp/out" 2>&1; then
			echo "  FAIL(self-test): expected violation, got clean exit: $2" >&2
			return 1
		fi
		if ! grep -q "$want" "$tmp/out"; then
			echo "  FAIL(self-test): violation fired but message missing '$want'" >&2
			cat "$tmp/out" >&2
			return 1
		fi
		echo "  PASS: $2"
	}

	info "self-test: fixture mutations"
	mk_fixture clean
	expect_ok "$fixture" "clean fixture passes"

	mk_fixture stray
	printf '# Stray Notes\n' >"$fixture/notes.md"
	expect_fail "$fixture" "non-conforming filename fails" "non-conforming ADR filename"

	mk_fixture dup
	printf '# ADR-0002: Collision\n' >"$fixture/0002-collision.md"
	expect_fail "$fixture" "duplicate number fails (the parallel-session class)" "duplicate ADR number"

	mk_fixture untracked
	# Simulates an untracked file from a concurrent session: present on disk only.
	printf '# ADR-0001: Untracked\n' >"$fixture/0001-untracked.md"
	expect_fail "$fixture" "untracked collision fails" "duplicate ADR number"

	mk_fixture norow
	grep -v '0001-first' "$fixture/README.md" >"$fixture/README.new" && mv "$fixture/README.new" "$fixture/README.md"
	expect_fail "$fixture" "file without index row fails" "no index row"

	mk_fixture dangling
	printf '| [0003](0003-missing.md) | Ghost | 2026-01-03 | Accepted |\n' >>"$fixture/README.md"
	expect_fail "$fixture" "dangling index row fails" "missing file"

	mk_fixture titlemismatch
	sed -i 's/| First Decision |/| Renamed Decision |/' "$fixture/README.md"
	expect_fail "$fixture" "index/H1 title mismatch fails" "title mismatch"

	mk_fixture gap
	printf '# ADR-0004: Fourth Decision\n' >"$fixture/0004-fourth.md"
	printf '| [0004](0004-fourth.md) | Fourth Decision | 2026-01-04 | Accepted |\n' >>"$fixture/README.md"
	if run_checks "$fixture" >"$tmp/out" 2>&1; then
		grep -q "WARN: numbering gap at 0003" "$tmp/out" &&
			echo "  PASS: undocumented gap warns but does not fail" ||
			{
				echo "  FAIL(self-test): gap warning missing" >&2
				cat "$tmp/out" >&2
				return 1
			}
	else
		echo "  FAIL(self-test): gap fixture must not hard-fail" >&2
		cat "$tmp/out" >&2
		return 1
	fi

	rm -rf "$tmp"
	trap - EXIT
	info "self-test: all mutations detected"
}

if [ "${SELF_TEST:-0}" = "1" ]; then
	self_test
	exit 0
fi

run_checks "$ADR_DIR"
local_count=$(cd "$ADR_DIR" && ls [0-9][0-9][0-9][0-9]-*.md [0-9][0-9][0-9][0-9][a-z]-*.md 2>/dev/null | sort -u | wc -l | tr -d ' ')
info "OK: no duplicates, index in lockstep ($local_count ADRs)"
