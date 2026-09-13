#!/usr/bin/env bash
# check-modsums.sh — per-module `go mod tidy -diff` gate.
#
# Kills the missing-go.sum-hash class at the root: a go.mod edited without a
# matching go.sum update builds fine under a warm module cache but fails in
# any cold-cache consumer or CI leg. `go mod tidy -diff` asserts go.mod and
# go.sum are exactly what the module graph requires WITHOUT writing anything,
# so the gate cannot paper over drift the way a writing tidy would.
#
# Run: nix run .#check-modsums
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

failed=0
checked=0

while IFS= read -r gomod; do
	moddir="$(dirname "$gomod")"
	checked=$((checked + 1))

	out="$(cd "$moddir" && GOWORK=off go mod tidy -diff 2>&1)" || {
		failed=1
		echo "❌ $moddir: go.mod/go.sum not tidy (cold-cache builds will fail)"
		echo "$out" | head -20
		echo ""
	}
done < <(find . -name go.mod -not -path './vendor/*' | sort)

if [ "$failed" -ne 0 ]; then
	echo "❌ check-modsums: $checked modules checked, some not tidy — run 'GOWORK=off go mod tidy' in each failing module and commit both files"
	exit 1
fi

echo "✅ check-modsums: $checked modules tidy (go.mod/go.sum complete for cold cache)"
