#!/usr/bin/env bash
# check-changelog-symbols.sh — CHANGELOG honesty gate.
#
# Every `pkg.Symbol` identifier cited in the [Unreleased] Added/Changed
# sections of CHANGELOG.md must exist in docs/api_surface.txt (the
# api-stability golden). This kills the reverted-work fiction class
# mechanically: entries describing work that was reverted before tagging cite
# symbols that no longer exist (the 2026-08-10/11 tombstone entries described
# e406edcfb, reverted by a6613ef0d before any tag; the CHANGELOG never
# recorded the reversion — corrected 2026-08-16).
#
# Scope: ONLY the [Unreleased] section. Released version sections are frozen
# history — their symbols were true at tag time and may legitimately have
# been removed since (e.g. the codec module deletion, ADR-0128).
#
# Resolution: a citation `alias.Symbol` passes if
#   a) the api-stability golden has "/alias/<kind> Symbol" (package alias ==
#      golden path segment), or
#   b) the golden has "alias/...<kind> Symbol" (cited by module root, symbol
#      in a nested package of that module), or
#   c) the repo source has a directory named `alias` declaring the exported
#      symbol — the fallback for true subpackage citations the module-root
#      golden cannot see (e.g. enginetest.RunSeqSeekableStreamLogTest: a
#      package INSIDE the metaengine module, not a module root itself).
# <kind> ∈ func|method|type|struct|interface|const|var.
#
# Usage: bash scripts/check-changelog-symbols.sh
# Exit: 0 = all cited symbols exist, 1 = fiction found (or extraction failed).

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CHANGELOG="CHANGELOG.md"
GOLDEN="docs/api_surface.txt"

# Aliases that are not repo packages: stdlib names and prose fragments that
# survive the `lowercase.Uppercase` regex (e.g. "cqrs-lint.E005" yields
# "lint.E005"). Extend, never narrow, without a matching golden entry.
SKIP_ALIASES='^(fmt|os|time|sync|context|errors|strings|strconv|bytes|io|log|testing|json|database|sql|net|http|reflect|sort|math|filepath|regexp|slog|rand|slices|maps|atomic|url|tar|zip|tls|big|lint|runtime|pkg|t|e|g|go|api|cli|yaml|toml|db|pebble|humanize|engine|bbolt)$'

# exists_in_source_dir: true when any repo directory named `alias` declares
# the exported `symbol` in a non-test .go file (func/type level; const/var
# block members are rare in citations and covered by the golden tiers).
exists_in_source_dir() {
	local alias="$1" symbol="$2"
	local dir f
	while IFS= read -r dir; do
		for f in "$dir"/*.go; do
			case "$f" in
			*_test.go) continue ;;
			esac
			[ -e "$f" ] || continue
			if grep -qE \
				-e "^func ${symbol}\\(" \
				-e "^type ${symbol} " \
				-e "^type ${symbol}\\[" \
				-e "^type ${symbol}$" \
				"$f"; then
				return 0
			fi
		done
	done < <(find . -type d -name "$alias" -not -path './.git/*' -not -path '*/testdata/*' | sort)
	return 1
}

# 1. Extract the [Unreleased] section, keep only Added/Changed subsections.
section=$(awk '
	/^## \[Unreleased\]/{sel=1; next}
	/^## \[/{sel=0}
	!sel{next}
	/^### /{keep = /^### (Added|Changed)/}
	keep
' "$CHANGELOG")

if [ -z "$section" ]; then
	echo "ERROR: no [Unreleased] Added/Changed content found in $CHANGELOG — extraction is broken or the section is empty."
	exit 1
fi

# 2. Pull backticked code spans, then qualified pkg.Symbol refs out of them.
refs=$(printf '%s\n' "$section" |
	grep -oE '`[^`]+`' |
	grep -oE '\b[a-z][a-z0-9]*\.[A-Z][A-Za-z0-9]*' |
	sort -u)

if [ -z "$refs" ]; then
	echo "✓ No pkg.Symbol citations in [Unreleased] Added/Changed — nothing to verify."
	exit 0
fi

# 3. Verify each ref against the api-stability golden.
errors=0
total=0
while IFS= read -r ref; do
	alias="${ref%%.*}"
	symbol="${ref#*.}"
	symbol="${symbol%%\[*}" # strip generic args: Metadata[K] -> Metadata

	if printf '%s' "$alias" | grep -qE "$SKIP_ALIASES"; then
		continue
	fi

	total=$((total + 1))

	if grep -qE "/${alias}/(func|method|type|struct|interface|const|var) ${symbol}$" "$GOLDEN" ||
		grep -qE "^${alias}/.*(func|method|type|struct|interface|const|var) ${symbol}$" "$GOLDEN" ||
		exists_in_source_dir "$alias" "$symbol"; then
		continue
	fi

	echo "FICTION: $ref cited in [Unreleased] Added/Changed but absent from $GOLDEN and repo source."
	errors=$((errors + 1))
done <<<"$refs"

echo "Verified $total pkg.Symbol citation(s) against the API golden."

if [ "$errors" -gt 0 ]; then
	echo ""
	echo "Found $errors cited symbol(s) that do not exist."
	echo "Either the entry describes reverted work (delete/correct it), or the"
	echo "symbol name is wrong, or docs/api_surface.txt is stale"
	echo "(cd cmd/api-stability && GOWORK=off go run -tags 'goexperiment.jsonv2' . --update)."
	exit 1
fi

echo "✓ CHANGELOG [Unreleased] citations are honest."
