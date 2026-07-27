#!/usr/bin/env bash
# verify-docs.sh — CI assertions for documentation consistency
# Runs as part of CI to catch documentation drift before it reaches master.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
errors=0

# ── P6-29: Build must compile ──────────────────────────────────────────────
echo "=== Build check ==="
if ! nix run .#build 2>/dev/null; then
  echo "FAIL: nix run .#build failed"
  errors=$((errors + 1))
fi

# ── P6-30: Exactly one [Unreleased] in CHANGELOG.md ─────────────────────────
echo "=== CHANGELOG [Unreleased] count ==="
unreleased_count=$(grep -cE '^## \[Unreleased\]' CHANGELOG.md)
if [ "$unreleased_count" -ne 1 ]; then
  echo "FAIL: Found $unreleased_count '## [Unreleased]' sections (expected 1)"
  errors=$((errors + 1))
else
  echo "OK: exactly 1 [Unreleased] section"
fi

# ── P6-31: Module count in docs matches actual ─────────────────────────────
echo "=== Module count check ==="
actual_count=$(find . -name go.mod -not -path './vendor/*' | wc -l)
stale_refs=$(grep -rnE '\b(28|48|49|52)\s*(go\.mod|modules)' \
  --include='*.md' \
  README.md AGENTS.md FEATURES.md CONTRIBUTING.md docs/README.md \
  docs/v4-WISHLIST.md ROADMAP.md TODO_LIST.md 2>/dev/null \
  | grep -v 'archive/' || true)
if [ -n "$stale_refs" ]; then
  echo "FAIL: Stale module count references found:"
  echo "$stale_refs"
  errors=$((errors + 1))
else
  echo "OK: no stale module count references in core docs"
fi

# ── P6-32: License in README matches LICENSE file ───────────────────────────
echo "=== License consistency check ==="
license_type=$(head -1 LICENSE | awk '{print $1}')
readme_license=$(grep -iE '^(## License|.+ (MIT|PROPRIETARY|Apache-2.0))' README.md \
  | head -1 | grep -oiE '(MIT|PROPRIETARY|Apache-2.0)' | head -1 || true)
if [ -z "$readme_license" ]; then
  # Try the specific line format
  readme_license=$(grep -iE 'PROPRIETARY|MIT license' README.md | head -1 \
    | grep -oiE '(MIT|PROPRIETARY|Apache-2.0)' | head -1 || true)
fi
if echo "$license_type" | grep -qi "PROPRIETARY" && echo "$readme_license" | grep -qi "PROPRIETARY"; then
  echo "OK: LICENSE file and README both say PROPRIETARY"
elif [ -z "$readme_license" ]; then
  echo "WARN: Could not determine license from README (manual check needed)"
else
  echo "FAIL: LICENSE file says '$license_type' but README says '$readme_license'"
  errors=$((errors + 1))
fi

# ── ADR index completeness ──────────────────────────────────────────────────
echo "=== ADR index completeness ==="
adr_files=$(find docs/adr -maxdepth 1 -name '00*.md' ! -name 'README.md' | wc -l)
adr_indexed=$(grep -cE '^\| \[00[0-9]+\]' docs/README.md || true)
if [ "$adr_files" -ne "$adr_indexed" ]; then
  echo "FAIL: $adr_files ADR files exist but only $adr_indexed are indexed in docs/README.md"
  errors=$((errors + 1))
else
  echo "OK: all $adr_files ADRs indexed in docs/README.md"
fi

# ── Error family count consistency ──────────────────────────────────────────
echo "=== Error family count consistency ==="
# go-error-family v0.10.0 has 6 families (Rejection, Conflict, Transient,
# Infrastructure, Orchestration, Corruption). Living docs must say "6-family",
# never "5-family". This check prevents split-brain after a family is added.
# Excludes: archive/ (historical), CHANGELOG (historical), docs/status/ (point-in-time),
# docs/feedback/ (historical consumer feedback), and ADR amendment notes.
stale_family=$(grep -rniE '\b5[- ]family\b|\bfive families\b|\b5 error families\b' \
  --include='*.md' . 2>/dev/null \
  | grep -v '/archive/' \
  | grep -v 'CHANGELOG' \
  | grep -v 'docs/status/' \
  | grep -v 'docs/feedback/' \
  | grep -v 'docs/planning/' \
  | grep -v 'Amendment.*Five Families' \
  | grep -v '.git/' \
  || true)
if [ -n "$stale_family" ]; then
  echo "FAIL: Stale '5-family' references found (should be '6-family'):"
  echo "$stale_family"
  errors=$((errors + 1))
else
  echo "OK: no stale '5-family' references in living docs"
fi

# ── Summary ────────────────────────────────────────────────────────────────
echo ""
if [ "$errors" -eq 0 ]; then
  echo "✓ All documentation assertions passed"
else
  echo "✗ $errors documentation assertion(s) failed"
  exit 1
fi
