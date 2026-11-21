#!/usr/bin/env bash
# lint-rename-id.sh — drive a module's varnamelen 'id' findings to zero by
# block-scoped rename, lint-verifying between passes (lint-campaign helper,
# 2026-11-21). Usage: lint-rename-id.sh <module> <NewName> [max-passes]
set -uo pipefail
repo="$(git rev-parse --show-toplevel)"
cd "$repo"
mod="${1:?module}"
name="${2:?new name}"
passes="${3:-4}"
found="$(mktemp)"

for pass in $(seq 1 "$passes"); do
	out="$(GT_LINT_MODULES="$mod" nix run .#lint-scope 2>&1 || true)"
	if ! echo "$out" | grep -q "varnamelen"; then
		echo "pass $pass: no varnamelen left"
		echo "$out" | tail -2
		rm -f "$found"
		exit 0
	fi
	echo "$out" | grep "varnamelen" | sed 's/ (varnamelen)//' > "$found"
	python3 - "$name" "$found" <<'PYEOF'
import re, sys
name, found = sys.argv[1], sys.argv[2]
sites = {}
for line in open(found):
	m = re.match(r"(\S+\.go):(\d+):\d+: ", line)
	if m:
		sites.setdefault(m.group(1), []).append(int(m.group(2)))
for f, decls in sites.items():
	lines = open(f).read().split("\n")
	for decl in decls:
		end = len(lines)
		for i in range(decl, len(lines)):
			if lines[i] == "}":
				end = i + 1
				break
		for i in range(decl - 1, end):
			lines[i] = re.sub(r"\bid\b", name, lines[i])
	open(f, "w").write("\n".join(lines))
	print(f"{f}: {len(decls)} sites -> {name}")
PYEOF
done
echo "still varnamelen after $passes passes — manual review needed"
rm -f "$found"
exit 1
