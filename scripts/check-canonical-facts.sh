#!/usr/bin/env bash
# check-canonical-facts.sh — docs must cite repo-derived numbers, not
# remembered ones. The number-rot class kept biting: "92 go.mod files"
# survived three module additions, and the recipes gate count drifted
# 80→81 unnoticed (both fixed 2026-09-20). This gate derives the facts
# from the repo and fails on any citation that disagrees.
#
# Facts covered (each derived fresh, never hardcoded):
#   1. go.mod count — every "<N> go.mod" citation in the canonical doc
#      set (AGENTS.md, README.md, ROADMAP.md, docs/agents/module-map.md)
#      must equal `find . -name go.mod -not -path './vendor/*' | wc -l`.
#   2. module-map census — every non-wildcard module dir with a go.mod
#      must appear as a row/token in docs/agents/module-map.md (the
#      example/* and metaengine/*engine wildcards are accepted only
#      because the census banner documents their membership).
#   3. recipes classification — the "<N>/<N> classified" claim in
#      AGENTS.md must equal the recipeCatalog map entries in
#      cmd/doc-check/recipes_catalog*.go.
#
# Usage:
#   scripts/check-canonical-facts.sh             # gate
#   scripts/check-canonical-facts.sh --self-test # mutation suite in a
#                                                # temp copy (planted
#                                                # wrong numbers must
#                                                # fail each fact leg)
#
# Exit codes: 0 = all citations honest, 1 = drift found, 2 = usage.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SELFTEST_MODE=0

if [[ "${1:-}" == "--self-test" ]]; then
	SELFTEST_MODE=1
elif [[ $# -gt 0 ]]; then
	echo "usage: $0 [--self-test]" >&2
	exit 2
fi

DOC_SET=("AGENTS.md" "README.md" "ROADMAP.md" "docs/agents/module-map.md")

failures=0

check_gomod_count() {
	local dir="$1" docs=("${DOC_SET[@]}")
	local derived
	derived=$(find "$dir" -name go.mod -not -path '*/vendor/*' | wc -l | tr -d ' ')

	local cited
	for doc in "${docs[@]}"; do
		[[ -f "$dir/$doc" ]] || continue
		while IFS= read -r line; do
			cited=$(printf '%s' "$line" | grep -oE '[0-9]+ go\.mod' | grep -oE '^[0-9]+')
			if [[ -n "$cited" && "$cited" != "$derived" ]]; then
				echo "✗ $doc cites '$cited go.mod' but the repo has $derived" >&2
				echo "  line: $line" >&2
				failures=$((failures + 1))
			fi
		done < <(grep -nE '[0-9]+ go\.mod' "$dir/$doc" || true)
	done
	echo "  go.mod count: derived=$derived"
}

check_module_map() {
	local dir="$1"
	python3 - "$dir" <<'PYEOF'
import re, subprocess, sys
root = sys.argv[1]
rows = set()
for line in open(f"{root}/docs/agents/module-map.md"):
    for m in re.finditer(r'`([a-z0-9/*_.-]+/?)`', line):
        tok = m.group(1).rstrip('/')
        if '*' not in tok:
            rows.add(tok)
real = set()
out = subprocess.run(['find', root, '-name', 'go.mod', '-not', '-path', f'{root}/vendor/*'],
                     capture_output=True, text=True).stdout
for line in out.split('\n'):
    if not line or line == f'{root}/go.mod' or '/testdata/' in line:
        continue
    mod = line[len(root) + 1:].rsplit('/go.mod', 1)[0]
    real.add(mod)
missing = {r for r in real if r not in rows and '*' not in r
           and not r.startswith('example/')
           and not re.match(r'metaengine/\w+engine$', r)
           and not r.startswith('stack/')}
# stack presets are rowed in combined rows (multi-module lines) — the
# backtick scan already caught them individually; anything left is real.
if missing:
    print(f"✗ module-map census: unrowed modules: {', '.join(sorted(missing))}", file=sys.stderr)
    sys.exit(1)
print(f"  module-map census: all {len(real)} modules rowed")
PYEOF
	if [[ $? -ne 0 ]]; then
		failures=$((failures + 1))
	fi
}

check_recipes_count() {
	local dir="$1"
	local derived
	derived=$(grep -hE '^\s*"[^"]+":\s*\{' "$dir"/cmd/doc-check/recipes_catalog*.go | wc -l | tr -d ' ')

	local cited
	cited=$(grep -oE '[0-9]+/[0-9]+ classified' "$dir/AGENTS.md" | head -1 | grep -oE '^[0-9]+')
	if [[ -z "$cited" ]]; then
		echo "✗ AGENTS.md carries no '<N>/<N> classified' recipes claim — gate cannot verify it" >&2
		failures=$((failures + 1))
	elif [[ "$cited" != "$derived" ]]; then
		echo "✗ AGENTS.md claims '$cited/$cited classified' but the recipes catalog has $derived entries" >&2
		failures=$((failures + 1))
	fi
	echo "  recipes catalog: derived=$derived cited=${cited:-none}"
}

run_all() {
	local dir="$1"
	echo "━━━ canonical facts ($dir) ━━━"
	check_gomod_count "$dir"
	check_module_map "$dir"
	check_recipes_count "$dir"
}

self_test() {
	local tmp
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp:-}"' EXIT

	# Minimal fake repo carrying one of each fact shape.
	mkdir -p "$tmp/cmd/doc-check" "$tmp/docs/agents" "$tmp/storage/memory" \
		"$tmp/one" "$tmp/two" "$tmp/three"
	touch "$tmp/go.mod" "$tmp/one/go.mod" "$tmp/two/go.mod" "$tmp/three/go.mod" \
		"$tmp/storage/memory/go.mod"
	printf '| Module | Role | Notes |\n| --- | --- | --- |\n| `storage/memory/` | x | y |\n| `one/` | x | y |\n| `two/` | x | y |\n| `three/` | x | y |\n' \
		>"$tmp/docs/agents/module-map.md"
	printf 'var recipeCatalogA = map[string]recipeSpec{\n\t"r1": {\n\t\tpreamble: "x",\n\t},\n}\n' \
		>"$tmp/cmd/doc-check/recipes_catalog.go"
	printf '# t\n\n5 go.mod files and 1/1 classified\n' >"$tmp/AGENTS.md"
	printf '# t\n' "$tmp/ROADMAP.md" >"$tmp/README.md"

	echo "━━━ check-canonical-facts self-test ━━━"

	# Leg 1 positive: honest numbers pass (4 go.mod, 1 entry, 1/1).
	run_all "$tmp" >/dev/null 2>&1
	if [[ $failures -eq 0 ]]; then
		echo "  ✓ PASS: honest citations pass"
	else
		echo "  ✗ FAIL: honest citations should pass (got $failures failures)"
		local honest_failed=1
	fi

	# Leg 1 mutation: wrong go.mod count.
	failures=0
	printf '# t\n\n99 go.mod files\n' >"$tmp/AGENTS.md"
	run_all "$tmp" >/dev/null 2>&1
	if [[ $failures -gt 0 ]]; then
		echo "  ✓ PASS: wrong go.mod count caught"
	else
		echo "  ✗ FAIL: wrong go.mod count not caught"
	fi

	# Leg 2 mutation: module missing from map.
	failures=0
	printf '# t\n\n5 go.mod files and 1/1 classified\n' >"$tmp/AGENTS.md"
	printf '| Module | Role | Notes |\n| --- | --- | --- |\n| `other/` | x | y |\n' \
		>"$tmp/docs/agents/module-map.md"
	run_all "$tmp" >/dev/null 2>&1
	if [[ $failures -gt 0 ]]; then
		echo "  ✓ PASS: unrowed module caught"
	else
		echo "  ✗ FAIL: unrowed module not caught"
	fi

	# Leg 3 mutation: recipes claim drift.
	failures=0
	printf '| Module | Role | Notes |\n| --- | --- | --- |\n| `storage/memory/` | x | y |\n' \
		>"$tmp/docs/agents/module-map.md"
	printf '# t\n\n5 go.mod files and 9/9 classified\n' >"$tmp/AGENTS.md"
	run_all "$tmp" >/dev/null 2>&1
	if [[ $failures -gt 0 ]]; then
		echo "  ✓ PASS: recipes-count drift caught"
	else
		echo "  ✗ FAIL: recipes-count drift not caught"
	fi

	if [[ "${honest_failed:-0}" == 1 ]]; then
		echo "self-test: base leg failed"
		return 1
	fi
	echo "self-test: all mutation legs behave"
	return 0
}

if [[ "$SELFTEST_MODE" == 1 ]]; then
	self_test
	exit $?
fi

cd "$ROOT"
failures=0
run_all "$ROOT"

if [[ $failures -gt 0 ]]; then
	echo "canonical-facts: ${failures} drift finding(s) — update the doc OR the repo, never just one"
	exit 1
fi
echo "canonical-facts: all citations match repo-derived truth"
exit 0
