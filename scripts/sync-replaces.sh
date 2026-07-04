#!/usr/bin/env bash
# Ensures every go.mod has replace directives for ALL larsartmann/go-cqrs-lite
# sibling modules it requires (direct or indirect). Required for GOWORK=off
# per-module builds (CI lint, devShell, api-stability, etc.).
#
# Usage: bash scripts/sync-replaces.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# Map a module path suffix to its repo-relative directory.
# "event/v3"          -> "event"
# "storage/memory/v3" -> "storage/memory"
# "event/v3/eventtest" -> "event/v3/eventtest"  (no /vN at end -> keep as-is)
modpath_to_dir() {
	local suffix="$1"
	if [[ "$suffix" =~ ^(.+)/v[0-9]+$ ]]; then
		echo "${BASH_REMATCH[1]}"
	else
		echo "$suffix"
	fi
}

# Relative path from module dir (repo-relative) to target dir (repo-relative).
relative_to() {
	local from_dir="$1" target_dir="$2"
	local depth
	depth=$(tr -cd '/' <<<"$from_dir" | wc -c)
	local prefix=""
	for ((i = 0; i <= depth; i++)); do prefix+="../"; done
	echo "${prefix}${target_dir}"
}

count=0
while IFS= read -r -d '' gomod; do
	mod_dir="${gomod%/go.mod}"
	rel_dir="${mod_dir#"$REPO_ROOT"/}"
	[ "$rel_dir" = "$mod_dir" ] && rel_dir="."

	# Collect required sibling module paths (from require lines, excluding replace targets)
	mapfile -t required < <(
		grep -E '^\s+github\.com/larsartmann/go-cqrs-lite/' "$gomod" |
			grep -v '=>' |
			grep -oP 'github\.com/larsartmann/go-cqrs-lite/[^\s/]+' |
			sort -u || true
	)
	# Also catch multi-segment paths like .../storage/memory/v3, .../event/v3/eventtest
	mapfile -t required_full < <(
		grep -E '^\s+github\.com/larsartmann/go-cqrs-lite/' "$gomod" |
			grep -v '=>' |
			sed -E 's|.*github\.com/larsartmann/go-cqrs-lite/([^[:space:]]+).*|\1|' |
			sort -u || true
	)

	# Collect existing replace LHS paths
	mapfile -t existing < <(
		grep '=>' "$gomod" |
			grep 'github.com/larsartmann/go-cqrs-lite/' |
			sed -E 's|.*github\.com/larsartmann/go-cqrs-lite/([^[:space:>]]+).*=>.*|\1|' |
			sort -u || true
	)

	# Compute missing
	for req_suffix in "${required_full[@]:-}"; do
		[ -z "$req_suffix" ] && continue
		# Skip if already replaced
		if printf '%s\n' "${existing[@]:-}" | grep -qxF "$req_suffix"; then
			continue
		fi
		target_dir=$(modpath_to_dir "$req_suffix")
		rel_path=$(relative_to "$rel_dir" "$target_dir")
		full_path="github.com/larsartmann/go-cqrs-lite/$req_suffix"
		(cd "$mod_dir" && go mod edit "-replace=${full_path}=${rel_path}")
		echo "ADDED: ${rel_dir}/go.mod: $full_path => $rel_path"
		count=$((count + 1))
	done
done < <(find "$REPO_ROOT" -name go.mod -not -path '*/vendor/*' -not -path '*/.git/*' -not -path "$REPO_ROOT/go.mod" -print0)

echo "Added $count replace directive(s)."
