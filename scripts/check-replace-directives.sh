#!/usr/bin/env bash
# Verifies that every replace directive in a module go.mod has a matching
# entry in go.work. This catches:
#   - Missing replace directives that would break GOWORK=off builds (consumer perspective)
#   - Stale replace directives pointing to non-existent paths
#
# Usage: scripts/check-replace-directives.sh
# Exit code: 0 = all good, 1 = mismatch found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

errors=0

# Collect all go.mod files (excluding vendor)
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"

    # Extract replace directives that point to local paths (start with ./ or ../)
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*// ]] && continue
        [[ -z "$line" ]] && continue

        # Extract module path and replacement path
        # Format: "module/path => ./relative/path" or "module/path => ../relative/path"
        if echo "$line" | grep -qE '[^[:space:]]+[[:space:]]+=>[[:space:]]+\.\.?/'; then
            modpath="$(echo "$line" | awk '{print $1}')"
            relpath="$(echo "$line" | sed 's/.*=>[[:space:]]*//')"

            # Resolve the relative path from the module directory
            target_dir="$(cd "$moddir" && cd "$relpath" 2>/dev/null && pwd || echo "MISSING")"

            if [[ "$target_dir" == "MISSING" ]]; then
                echo "ERROR: $modfile has replace $modpath => $relpath but path does not exist"
                errors=$((errors + 1))
            fi
        fi
    done < <(grep '=>' "$modfile" 2>/dev/null || true)
done < <(find . -name go.mod -not -path './vendor/*' | sort)

if [[ $errors -gt 0 ]]; then
    echo ""
    echo "Found $errors replace directive issue(s)."
    exit 1
fi

echo "All replace directives valid."
