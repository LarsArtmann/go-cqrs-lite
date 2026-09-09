#!/usr/bin/env bash
# Deprecated-linter-name tripwire: every linter named in .golangci.yml
# (linters.enable, linters.disable, and every linters.settings key) must be
# a linter the installed golangci-lint knows. `golangci-lint config verify`
# catches schema drift, but a RENAMED linter can silently become a no-op
# setting (the gci incident class, 2026-08-30) — this gate makes renames loud.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
config="$repo_root/.golangci.yml"

# All linter names this golangci-lint version knows (enabled + disabled).
mapfile -t supported < <(golangci-lint linters --json 2>/dev/null |
	jq -r '(.Enabled + .Disabled)[].name' | sort -u)

declare -A supported_set=()
for name in "${supported[@]}"; do
	supported_set["$name"]=1
done

# Names referenced by the config, extracted in one pass:
#   - list items under `linters.enable:` / `linters.disable:` (4-space "- x")
#   - keys under `linters.settings:` (4-space "x:")
mapfile -t referenced < <(awk '
  /^linters:/ { in_linters=1; mode=""; next }
  /^[A-Za-z-]+:/ { in_linters=0 }
  !in_linters { next }
  /^  (enable|disable):/ { mode="list"; next }
  /^  [A-Za-z-]+:/ { mode = ($0 ~ /^  settings:/) ? "settings" : "" ; next }
  mode == "list" && /^ +- +[A-Za-z0-9_-]+[[:space:]]*$/ {
    sub(/^ +- +/, ""); print; next
  }
  # Settings linter names sit at exactly four spaces; nested option keys are
  # deeper and must NOT be treated as linter names.
  mode == "settings" && /^    [A-Za-z0-9_-]+:/ {
    key = $1; gsub(/:/, "", key); print key
  }
' "$config")

status=0
declare -A seen=()
for name in "${referenced[@]}"; do
	[ -n "${seen[$name]:-}" ] && continue
	seen["$name"]=1
	if [ -z "${supported_set[$name]:-}" ]; then
		echo "ERROR: .golangci.yml references linter \"$name\" unknown to golangci-lint $(golangci-lint version --short 2>/dev/null || echo '?') — renamed? removed?" >&2
		status=1
	fi
done

if [ "$status" -eq 0 ]; then
	echo "✓ every linter named in .golangci.yml is known (${#seen[@]} checked)"
fi

exit "$status"
