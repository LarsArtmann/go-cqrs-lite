#!/usr/bin/env bash
# check-formatters.sh — Pins .golangci.yml formatters.enable to the documented
# state, so a config reformat cannot silently resurrect a formatter that
# fights treefmt.
#
# Problem (seen three times now): .golangci.yml reformats silently mutated
# linter semantics — depguard was disabled by a re-indent (2026-08-30), and
# gci was re-added to formatters.enable twice (2026-09-05, 2026-09-06, both
# times by an automated repo-wide config reformat). Import grouping is owned
# by treefmt's goimports -local flag; gci in golangci makes the LINTER fight
# the FORMATTER over the same import blocks (it re-broke 95+ files once).
#
# This script now SELF-HEALS: when gci is present it removes it in place and
# re-verifies, instead of only reporting — the regression is machine-authored
# and recurs faster than manual removal. Third-party config edits still fail
# the run when a pinned formatter is MISSING (that is never auto-repaired).
#
# Usage: bash scripts/check-formatters.sh
# Exit:  0 if the formatters list matches the pinned state (after any
#        self-repair), 1 otherwise.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CONFIG=.golangci.yml

# Extract the formatters.enable list (between "formatters:" and the next
# top-level key), tolerating any indentation.
ENABLE=$(awk '
	/^formatters:/ { in_block = 1; next }
	in_block && /^[^ \t#]/ { in_block = 0 }
	in_block && /^[ \t]+-[ \t]+[a-z]/ {
		sub(/^[ \t]+-[ \t]+/, "")
		print $1
	}
' "$CONFIG")

fail=0

for wanted in goimports gofumpt golines; do
	if ! grep -qx "$wanted" <<<"$ENABLE"; then
		echo "FICTION: $CONFIG formatters.enable is missing '$wanted' (documented state: goimports, gofumpt, golines)."
		fail=1
	fi
done

if grep -qx gci <<<"$ENABLE"; then
	echo "REPAIR: removing 'gci' from $CONFIG formatters.enable (re-added by an automated"
	echo "        config reformat; import grouping is owned by treefmt's goimports -local)."
	# Drop only the gci list item inside the formatters block — the linters
	# section never legitimately contains a bare "- gci" list item of its own
	# under formatters:; this awk pass is scoped to that block.
	awk '
		/^formatters:/ { in_block = 1; print; next }
		in_block && /^[^ \t#]/ { in_block = 0 }
		in_block && /^[ \t]+-[ \t]+gci[ \t]*$/ { next }
		{ print }
	' "$CONFIG" > "$CONFIG.tmp" && mv "$CONFIG.tmp" "$CONFIG"

	ENABLE=$(awk '
		/^formatters:/ { in_block = 1; next }
		in_block && /^[^ \t#]/ { in_block = 0 }
		in_block && /^[ \t]+-[ \t]+[a-z]/ {
			sub(/^[ \t]+-[ \t]+/, "")
			print $1
		}
	' "$CONFIG")

	if grep -qx gci <<<"$ENABLE"; then
		echo "FICTION: gci still present after repair attempt."
		fail=1
	fi
fi

if [ "$fail" -ne 0 ]; then
	exit 1
fi

echo "✓ formatters.enable matches the documented state (no gci; goimports, gofumpt, golines present)."
