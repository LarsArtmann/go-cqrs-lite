#!/usr/bin/env bash
# check-formatters.sh — Pins .golangci.yml formatters.enable to the documented
# state, so a config reformat cannot silently resurrect a formatter that
# fights treefmt.
#
# Problem (seen twice now): .golangci.yml reformats silently mutated linter
# semantics — depguard was disabled by a re-indent (2026-08-30), and gci was
# re-added to formatters.enable, making 366 treefmt-clean files fail
# `nix run .#verify` lint (2026-09-05). Import grouping is owned by treefmt's
# goimports -local flag; gci in golangci makes the LINTER fight the FORMATTER
# over the same import blocks (it re-broke 95+ files once before).
#
# Usage: bash scripts/check-formatters.sh
# Exit:  0 if the formatters list matches the pinned state, 1 otherwise.

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
	echo "FICTION: $CONFIG formatters.enable contains 'gci'."
	echo "         Import grouping is owned by treefmt's goimports -local flag"
	echo "         (AGENTS.md gotcha 18). gci makes the linter fight the formatter"
	echo "         over the same import blocks — 366 treefmt-clean files failed"
	echo "         verify lint when a config reformat re-added gci (2026-09-05)."
	echo "         Remove gci from formatters.enable to fix."
	fail=1
fi

if [ "$fail" -ne 0 ]; then
	exit 1
fi

echo "✓ formatters.enable matches the documented state (no gci; goimports, gofumpt, golines present)."
