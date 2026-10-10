#!/usr/bin/env bash
# check-engine-capabilities.sh — engine capability single-source gate.
#
# The per-engine capability matrix used to live in five places (ADR-0157 D2,
# advanced.md §6.13, system/README, COOKBOOK.md, modules.md) kept in lockstep
# by nothing — on 2026-10-10 that split brain held two live lies (COOKBOOK
# claimed duckdb undirected+edge-removal it does not implement; system/README
# listed `iroh` as a blank-import registry driver when irohengine has no
# RegisterDriver at all).
#
# This script DERIVES the truth from source into docs/engine-capabilities.md:
#   - registry drivers: metaengine.RegisterDriver("name", ...) in
#     metaengine/register.go + metaengine/*/register.go (filesystem scan —
#     an untracked engine module from a parallel session is caught too)
#   - Graph / Edge removal / Undirected: the engine implements the method the
#     metaengine capability asserts assert (GraphAddEdge / GraphRemoveEdge /
#     GraphNeighborsUndirected — the same probes behind HasGraphEdgeRemoval,
#     HasUndirectedGraphSupport, and the graphBackend assertion)
#   - delegation: a register.go constructing another engine module's engine
#     (turso → sqliteengine) inherits that engine's capabilities, noted
#   - CGo: a //go:build cgo constraint in the module's files
#
# Usage:
#   scripts/check-engine-capabilities.sh              # gate: regen + diff
#   scripts/check-engine-capabilities.sh --update     # regenerate the doc
#   scripts/check-engine-capabilities.sh --self-test  # fixture mutations
set -euo pipefail

DOC="docs/engine-capabilities.md"

fail() { echo "FAIL: $*" >&2; exit 1; }
info() { echo "check-engine-capabilities: $*"; }

while [ $# -gt 0 ]; do
  case "$1" in
    --doc) DOC="${2:?}"; shift 2 ;;
    --update) UPDATE=1; shift ;;
    --self-test) SELF_TEST=1; shift ;;
    *) fail "unknown flag: $1" ;;
  esac
done

prod_files() { # module dir → its non-test .go files
  local f
  for f in "$1"/*.go; do
    case "$f" in *_test.go) continue ;; esac
    [ -f "$f" ] && printf '%s\n' "$f"
  done
}

# has_method <module-dir> <MethodName> — a non-test .go file declares
# func (…) MethodName(.
has_method() {
  local f
  [ -n "$(prod_files "$1" | xargs -r grep -lE "func \([^)]+\) $2\(" 2>/dev/null || true)" ]
}

# registry_driver <module-dir> — the driver name registered by this module's
# register.go (the documented convention: metaengine/register.go +
# metaengine/*/register.go), or empty. RegisterDriver("name" may span lines,
# so scan 3 lines after each call; scanning only register.go keeps comment
# mentions of other driver names in core files out of the census.
registry_driver() {
  local hit
  [ -f "$1/register.go" ] || return 0
  hit=$(grep -h -A3 "RegisterDriver(" "$1/register.go" 2>/dev/null | grep -oE '"[a-z-]+"' | head -1 || true)
  printf '%s' "${hit//\"/}"
}

# cgo_gated <module-dir> — any production file carries a cgo build constraint.
cgo_gated() {
  [ -n "$(prod_files "$1" | xargs -r grep -l "^//go:build cgo" 2>/dev/null || true)" ]
}

# delegates_to <root> <module-dir> — engine module whose constructor this
# module's production files call (the turso→sqlite pattern), or empty.
delegates_to() {
  local root="$1" dir="$2" other pkg
  for other in "$root"/metaengine/*engine; do
    [ -d "$other" ] || continue
    [ "$other" = "$dir" ] && continue
    pkg=$(basename "$other")
    if [ -n "$(prod_files "$dir" | xargs -r grep -lE "${pkg}\.New[A-Z]" 2>/dev/null || true)" ]; then
      printf '%s' "$pkg"
      return 0
    fi
  done
  return 0
}

# Hand-maintained annotations (the only hand-written table cells; everything
# else is derived). Keyed by module dir name.
note_for() {
  case "$1" in
    metaengine)        printf 'in-process default; no persistence' ;;
    sqliteengine)      printf 'recursive CTE + iterative fallback; modernc pure-Go driver' ;;
    tursoengine)       printf 'wraps the sqlite engine (embedded Turso Database driver)' ;;
    pgengine)          printf 'WITH RECURSIVE; pgx' ;;
    mysqlengine)       printf 'WITH RECURSIVE 8.0+, probed iterative fallback' ;;
    duckdbengine)      printf 'WITH RECURSIVE; CGo — registration behind //go:build cgo' ;;
    badgerengine)      printf 'prefix-scan BFS over adjacency keys' ;;
    bboltengine)       printf 'no ADTGraph — graph queries fail unsupported' ;;
    pebbleengine)      printf 'no ADTGraph — graph queries fail unsupported' ;;
    dgraphengine)      printf 'native n(depth:) @recurse; dgo gRPC' ;;
    bigtableengine)    printf 'heavy GCP deps (~3 direct); no ADTGraph; excluded from convenience-set discussion (ADR-0157)' ;;
    irohengine)        printf 'CRDT-replicated wrapper (Replicated(local)); NO registry driver — programmatic only' ;;
    graphadapter)      printf 'graph-memory; programmatic only (not a DeploymentConfig driver); directed-only, flat node identity' ;;
    *)                 printf '' ;;
  esac
}

# derive_table <root> — prints the generated markdown for the tree at <root>.
derive_table() {
  local root="$1" rows="" dir mod driver delegate graph removal undirected build note driver_disp
  for dir in "$root"/metaengine "$root"/metaengine/*engine "$root"/metaengine/graphadapter; do
    [ -d "$dir" ] || continue
    [ -n "$(prod_files "$dir")" ] || continue
    mod=$(basename "$dir")
    driver=$(registry_driver "$dir")
    graph=no; removal=no; undirected=no
    has_method "$dir" GraphAddEdge && graph=yes
    has_method "$dir" GraphRemoveEdge && removal=yes
    has_method "$dir" GraphNeighborsUndirected && undirected=yes
    if [ "$graph" = no ] && [ "$mod" != metaengine ]; then
      delegate=$(delegates_to "$root" "$dir" || true)
      if [ -n "$delegate" ]; then
        has_method "$root/metaengine/$delegate" GraphAddEdge && graph="via $delegate"
        has_method "$root/metaengine/$delegate" GraphRemoveEdge && removal="via $delegate"
        has_method "$root/metaengine/$delegate" GraphNeighborsUndirected && undirected="via $delegate"
      fi
    fi
    if cgo_gated "$dir"; then build=CGo; else build='pure Go'; fi
    note=$(note_for "$mod")
    if [ -n "$driver" ]; then driver_disp="\`$driver\`"; else driver_disp='— (programmatic)'; fi
    rows="$rows
| $mod | $driver_disp | $graph | $removal | $undirected | $build | $note |"
  done

  cat <<EOF
<!-- GENERATED by scripts/check-engine-capabilities.sh — do not hand-edit the
     table. Regenerate: scripts/check-engine-capabilities.sh --update
     (flake: nix run .#check-engine-capabilities -- --update). The gate fails
     when source and this table drift — re-run after touching any
     metaengine/*/register.go or engine graph method. -->

# Engine Capabilities — single source

Membership classes: **registry drivers** self-register via
\`metaengine.RegisterDriver\` at init time and are selectable in
\`DeploymentConfig\`/\`cqrs.yaml\` after a blank import; **programmatic
engines** (irohengine, graphadapter) are constructed in code and have no
registry driver. Capabilities are interface-detected at runtime the same way
this table detects them at rest: \`GraphAddEdge\` (graph), \`GraphRemoveEdge\`
(edge removal — required by \`EdgeRemoval\` folds), \`GraphNeighborsUndirected\`
(undirected traversal — \`metaengine.HasUndirectedGraphSupport\`).
Delegating engines (turso) inherit their delegate's capabilities.

Other docs LINK here instead of restating per-engine facts: ADR-0157 D2,
advanced.md §6.13, system/README, metaengine/COOKBOOK.md.

| Module | Registry driver | Graph | Edge removal | Undirected | Build | Notes |
| --- | --- | --- | --- | --- | --- | --- |$rows
EOF
}

# run_gate <root> <doc>
run_gate() {
  local root="$1" doc="$2" tmpdiff
  [ -f "$doc" ] || { echo "FAIL: missing $doc (run with --update to generate)" >&2; return 1; }
  tmpdiff=$(mktemp)
  if ! derive_table "$root" | diff -u "$doc" - > "$tmpdiff" 2>&1; then
    echo "FAIL: $doc is out of sync with the source census:" >&2
    sed -n '1,40p' "$tmpdiff" >&2
    echo "" >&2
    echo "Regenerate with: scripts/check-engine-capabilities.sh --update" >&2
    rm -f "$tmpdiff"
    return 1
  fi
  rm -f "$tmpdiff"
}

self_test() {
  local tmp root doc
  tmp=$(mktemp -d)
  root="$tmp/repo"; doc="$tmp/doc.md"
  trap 'rm -rf "$tmp"' EXIT

  mk_fixture() {
    rm -rf "$root"
    mkdir -p "$root/metaengine/sqliteengine" "$root/metaengine/bboltengine"
    printf 'package metaengine\nfunc init() { RegisterDriver("memory", nil) }\n' > "$root/metaengine/register.go"
    printf 'package metaengine\nfunc (e *memEngine) GraphAddEdge() {}\nfunc (e *memEngine) GraphRemoveEdge() {}\nfunc (e *memEngine) GraphNeighborsUndirected() {}\n' > "$root/metaengine/graph.go"
    printf 'package sqliteengine\nfunc init() {\n\tmetaengine.RegisterDriver(\n\t\t"sqlite",\n\t\tnil)\n}\n' > "$root/metaengine/sqliteengine/register.go"
    printf 'package sqliteengine\nfunc (e *s) GraphAddEdge() {}\nfunc (e *s) GraphRemoveEdge() {}\nfunc (e *s) GraphNeighborsUndirected() {}\n' > "$root/metaengine/sqliteengine/graph.go"
    printf 'package bboltengine\nfunc init() { metaengine.RegisterDriver("bbolt", nil) }\n' > "$root/metaengine/bboltengine/register.go"
  }

  info "self-test: fixture mutations"

  mk_fixture
  derive_table "$root" > "$doc"
  if ! run_gate "$root" "$doc" >/dev/null 2>&1; then
    echo "  FAIL(self-test): pristine fixture must pass" >&2; return 1
  fi
  echo "  PASS: pristine fixture in sync"

  printf 'package bboltengine\nfunc (e *b) GraphAddEdge() {}\n' > "$root/metaengine/bboltengine/graph.go"
  if run_gate "$root" "$doc" >/dev/null 2>&1; then echo "  FAIL(self-test): bbolt graph growth undetected" >&2; return 1; fi
  echo "  PASS: graph capability growth detected"
  rm -f "$root/metaengine/bboltengine/graph.go"

  sed -i '/GraphNeighborsUndirected/d' "$root/metaengine/sqliteengine/graph.go"
  if run_gate "$root" "$doc" >/dev/null 2>&1; then echo "  FAIL(self-test): sqlite undirected loss undetected" >&2; return 1; fi
  echo "  PASS: undirected capability loss detected"
  mk_fixture; derive_table "$root" > "$doc"

  mkdir -p "$root/metaengine/newengine"
  printf 'package newengine\nfunc init() {\n\tmetaengine.RegisterDriver(\n\t\t"new",\n\t\tnil)\n}\n' > "$root/metaengine/newengine/register.go"
  if run_gate "$root" "$doc" >/dev/null 2>&1; then echo "  FAIL(self-test): new registry driver undetected" >&2; return 1; fi
  echo "  PASS: new registry driver detected"
  rm -rf "$root/metaengine/newengine"

  sed -i 's/in-process default/hand edit/' "$doc"
  if run_gate "$root" "$doc" >/dev/null 2>&1; then echo "  FAIL(self-test): hand edit undetected" >&2; return 1; fi
  echo "  PASS: hand-edited doc detected"

  rm -rf "$tmp"; trap - EXIT
  info "self-test: all mutations detected"
}

if [ "${SELF_TEST:-0}" = "1" ]; then
  self_test
  exit 0
fi

if [ "${UPDATE:-0}" = "1" ]; then
  mkdir -p "$(dirname "$DOC")"
  derive_table "." > "$DOC"
  info "regenerated $DOC"
  exit 0
fi

run_gate "." "$DOC"
info "OK: $DOC matches source census"
