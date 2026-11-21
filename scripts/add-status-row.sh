#!/usr/bin/env bash
# add-status-row.sh — insert a row into docs/status/README.md's live index.
#
# Why this exists: three sessions in a row hand-inserted the row and each
# time burned 15+ minutes on the same three bugs — hyphen vs COLON in the
# timestamp link text (the row says "2026-11-21 19:21", the filename says
# 2026-11-21_19-21), forgetting the em-dash title suffix, and misjudging
# the pipe-cell padding (every row is padded to the table's fixed width so
# an unpadded row visibly dents the rendered table).
#
# Usage:
#   scripts/add-status-row.sh <docs/status/report-file.md> <title> <summary>
#
# Behavior:
#   - derives the row timestamp from the filename (colons in the row text,
#     underscores in the file, em-dash before the title)
#   - inserts the row directly under the table's separator line
#   - pads the row to the table's current fixed width
#   - refuses to run when the README has uncommitted changes (the same
#     discipline as the art-dupl baseline re-pin)
#   - reminds about check-readme-links.sh
#
# Self-test: scripts/add-status-row.sh --self-test
set -euo pipefail

README_REL="docs/status/README.md"

die() {
	echo "add-status-row: $*" >&2
	exit 2
}

# insert_row <readme> <report-basename.md> <title> <summary>
insert_row() {
	local readme="$1" base="$2" title="$3" summary="$4"

	local date part stamp
	date="$(printf '%s' "$base" | cut -c1-10)"  # 2026-11-21
	part="$(printf '%s' "$base" | cut -c12-16)" # 19-21
	stamp="$date ${part//-/:}"                  # 2026-11-21 19:21

	local link="[$stamp]($base) — $title"

	local header_line
	header_line="$(grep -m1 -n '^| Report' "$readme" | cut -d: -f1)" ||
		die "no table header (line starting '| Report') in $readme"

	local width
	width="$(awk -v h="$((header_line + 2))" 'NR==h {print length($0)}' "$readme")"
	[ "$width" -gt 100 ] || die "could not measure a sane table width ($width) in $readme"

	local row="| $link | $summary"
	local pad=$((width - ${#row} - 1))
	[ "$pad" -ge 0 ] || die "row content (${#row} chars) exceeds table width $width — shorten the title or summary"
	row="$row$(printf '%*s' "$pad" '')|"

	awk -v at="$((header_line + 2))" -v row="$row" '
		NR == at { print row }
		{ print }
	' "$readme" >"$readme.tmp" && mv "$readme.tmp" "$readme"
}

case "${1:-}" in
--self-test)
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	mkdir -p "$tmp/docs/status"
	{
		echo "# Status Reports"
		echo
		echo "## Live reports index"
		echo
		echo "| Report$(printf '%*s' 54 '') | What$(printf '%*s' 58 '') |"
		echo "|$(printf '%*s' 62 '-' | tr ' ' '-')|$(printf '%*s' 64 '-' | tr ' ' '-')|"
		echo "| [2026-11-20 08:00](2026-11-20_08-00_old.md) — Old$(printf '%*s' 39 '') | padded summary row here matching width exactly$(printf '%*s' 21 '') |"
	} >"$tmp/$README_REL"
	printf 'old report\n' >"$tmp/docs/status/2026-11-20_08-00_old.md"
	printf 'new report\n' >"$tmp/docs/status/2026-11-21_19-21_fixture.md"

	insert_row "$tmp/$README_REL" "2026-11-21_19-21_fixture.md" "Fixture title" "fixture summary"

	grep -q '\[2026-11-21 19:21\](2026-11-21_19-21_fixture.md) — Fixture title' \
		"$tmp/$README_REL" || die "self-test: row text wrong (colon/em-dash)"
	grep -n "Fixture title" "$tmp/$README_REL" | grep -q "^7:" ||
		die "self-test: row not inserted directly under the separator"

	local_w="$(awk 'NR==7{print length($0)}' "$tmp/$README_REL")"
	old_w="$(awk 'NR==8{print length($0)}' "$tmp/$README_REL")"
	[ "$local_w" = "$old_w" ] || die "self-test: widths differ ($local_w vs $old_w) — padding wrong"

	echo "add-status-row self-test: OK (row under separator, widths aligned at $local_w)"
	;;
*)
	[ $# -eq 3 ] || die "usage: $0 <docs/status/report.md> <title> <summary>   (or --self-test)"
	file="$1"
	title="$2"
	summary="$3"
	base="$(basename "$file")"
	[ -f "docs/status/$base" ] || die "report not found under docs/status/: $base"
	git diff --quiet -- "$README_REL" || die "$README_REL has uncommitted changes — commit or stash first (baseline discipline)"

	insert_row "$README_REL" "$base" "$title" "$summary"
	echo "row inserted; now run: bash scripts/check-readme-links.sh && nix fmt"
	;;
esac
