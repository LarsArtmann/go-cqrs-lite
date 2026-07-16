#!/usr/bin/env bash
set -euo pipefail

# Adds a "Historical session artifact" banner after the first line of each
# .md file matching the pattern, and after <body> for .html files.
# Skips files that already contain the marker.
#
# Usage: bash scripts/add-historical-banners.sh

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MARKER="<!-- historical-artifact-banner -->"

add_md_banner() {
  local file="$1"
  local depth_prefix="$2"

  if grep -qF "$MARKER" "$file" 2>/dev/null; then
    echo "SKIP (already has banner): $file"
    return
  fi

  local banner
  banner=$(cat <<'BANNER'

<!-- historical-artifact-banner -->
> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](BANNER_ROOT_CHANGELOG) and
> [TODO_LIST.md](BANNER_ROOT_TODOLIST) for current state.
> Last documentation health audit: 2026-07-16.
BANNER
)
  banner="${banner//BANNER_ROOT_CHANGELOG/${depth_prefix}CHANGELOG.md}"
  banner="${banner//BANNER_ROOT_TODOLIST/${depth_prefix}TODO_LIST.md}"

  # Insert after the first line (the H1 title)
  local tmp
  tmp=$(mktemp)
  head -n1 "$file" > "$tmp"
  echo "$banner" >> "$tmp"
  tail -n +2 "$file" >> "$tmp"
  mv "$tmp" "$file"
  echo "DONE: $file"
}

add_html_banner() {
  local file="$1"
  local depth_prefix="$2"

  if grep -qF "$MARKER" "$file" 2>/dev/null; then
    echo "SKIP (already has banner): $file"
    return
  fi

  local banner
  banner='<!-- historical-artifact-banner --><div style="background:#f4d35e;color:#0e0e10;padding:0.6rem 1rem;font-size:0.85rem;font-weight:600;margin:0 0 1rem 0;border-radius:4px">Historical session artifact — items marked TODO/Open may have since been resolved. See CHANGELOG.md and TODO_LIST.md for current state. Last audit: 2026-07-16.</div>'

  local tmp
  tmp=$(mktemp)
  # Insert after the first <body> tag
  sed "s|<body>|<body>${banner}|" "$file" > "$tmp" || {
    # If no <body> tag, insert after </title>
    sed "s|</title>|</title>${banner}|" "$file" > "$tmp"
  }
  mv "$tmp" "$file"
  echo "DONE (html): $file"
}

# Process files by directory depth
# depth 2: docs/X/*.md -> ../../
# depth 3: docs/X/Y/*.md -> ../../../

find "$ROOT/docs" -name "*2026-07-1*" -type f | sort | while IFS= read -r f; do
  rel="$(realpath --relative-to="$ROOT" "$f")"
  # Count directory depth from root (excluding the filename)
  dir_count=$(echo "$rel" | tr '/' '\n' | wc -l)
  # dir_count includes the filename, so dirs = dir_count - 1
  dirs=$((dir_count - 1))

  if [[ "$f" == *.html ]]; then
    if [ "$dirs" -eq 3 ]; then
      add_html_banner "$f" "../../../"
    else
      add_html_banner "$f" "../../"
    fi
  else
    if [ "$dirs" -eq 3 ]; then
      add_md_banner "$f" "../../../"
    elif [ "$dirs" -eq 2 ]; then
      add_md_banner "$f" "../../"
    else
      echo "UNEXPECTED depth ($dirs) for: $f"
    fi
  fi
done

echo "---"
echo "Banner insertion complete."
