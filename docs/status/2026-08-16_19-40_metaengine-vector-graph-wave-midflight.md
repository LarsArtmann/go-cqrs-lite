# Status: Metaengine Vector/Graph Feature Wave — 2026-08-16 19:40

> Mid-flight snapshot. Five TODO_LIST items in flight (badger parity, vector
> at scale, filtered k-NN, GraphRemoveEdge, undirected traversal). Engine
> implementations are in and building; test coverage and bookkeeping are
> partially outstanding. No gates run since the wave started.

## What was done

### Research & environment

- Read the full metaengine capability surface (folds, dispatch contracts,
  executor, capability audit, keycodec) and all 9 engine backends relevant
  to graph/vector work before writing code.
- Badger parity audit concluded: pebble/bbolt precedent = brute-force
  JSON vectors under `vec\x00` keys + no native graph (multimap fallback).
  Decision: implement BOTH for badger — vector for parity, graph because a
  dual adjacency index (fwd+rev marker keys) gives O(degree) prefix-seek
  BFS, strictly better than the O(N·degree^depth) multimap fallback and it
  directly serves the removal + undirected work.
- Build env: `GOTOOLCHAIN=auto` + all caches redirected to /tmp
  (/mnt/buildcache still corrupted). Baseline `go build` + quick tests green
  before any change.

### Core metaengine (module root)

- **Filtered k-NN (Task 3)**: `Embedding.Metadata`, `VectorFilter{Field,
  Op, Value}` reusing `FilterOp` constants, exported
  `VectorMatchesFilters` evaluator, new optional `VectorFilterBackend`
  (embeds `VectorBackend`) with `VectorSearchFiltered`. Executor extracts a
  `Filters []VectorFilter` input field and dispatches; engines without the
  capability fail explicitly (`errUnsupportedVectorFilters`) — no silent
  unfiltered results.
- **Edge removal (Task 4)**: `EdgeRemoval` fold sentinel → `edgeRemoveFold`
  (`FoldEdgeRemove`) classified in both `On` and `OnRecord` paths;
  `store.applyFoldEdgeRemove` dispatches via new optional
  `graphEdgeRemover` contract. No degraded fallback (MultimapBackend has no
  targeted delete) — explicit `errEdgeRemoval` instead.
- **Undirected traversal (Task 5)**: optional `undirectedGraphBackend`
  (`GraphNeighborsUndirected`), `Undirected: true` input-field extraction,
  executor dispatch with explicit `errUndirectedGraph` when unsupported
  (never silently degrade to directed). Exported capability checks:
  `HasGraphEdgeRemoval`, `HasUndirectedGraphSupport`.
- Memory engine: `GraphRemoveEdge`, `GraphNeighborsUndirected` (reverse
  scan), `VectorSearchFiltered`; `MemoryVectorIndex` is now
  collection-namespaced (was a global id-keyed map — latent cross-collection
  pollution bug found and fixed during the rework).
- `Edge`/`EdgeRemoval` doc comments now pin WHY From/To stay untyped
  (runtime-heterogeneous collections, no Go existential types, reflect-based
  fold classification needs one concrete sentinel).
- keycodec: `VectorMetaKey/Prefix` (metadata in its own `vecm` family —
  vector bytes unchanged for old engines) + `GraphEdgeFwd/Rev` key helpers.

### Engines

| Engine | VectorFiltered | GraphRemoveEdge | GraphNeighborsUndirected | State |
|---|---|---|---|---|
| memory | yes | yes | yes (reverse scan) | built, adttest |
| pebble | yes (vecm keys) | n/a | n/a | built, tests TODO |
| bbolt | yes (vecm keys) | n/a | n/a | built, tests TODO |
| badger | yes | yes (dual-key delete) | yes (dual prefix scan) | built, **13 tests green** |
| sqlite | n/a | yes (DELETE) | yes (CTE + iterative) | built, tests green |
| pg | n/a | yes | yes (dual-direction CTE) | built, tests TODO (server) |
| mysql | n/a | yes | yes (CTE + iterative) | built, tests TODO (server) |
| dgraph | n/a | yes (DelNquads both dirs) | alias (symmetric storage) | built, tests TODO (server) |
| graphadapter | n/a | yes (sink.RemoveEdge) | deliberately NO (Traverse is outgoing-only; honest error) | built, tests TODO |
| iroh | passthrough | passthrough | passthrough | built |

- SQL engines: new `idx_graph_edges_to (collection, to_node)` reverse index
  (mysql: bare CREATE INDEX with duplicate-key tolerance since MySQL lacks
  IF NOT EXISTS).
- adttest matrix: new `GraphRemove`, `GraphUndirected`, `VectorFiltered`
  scenarios with capability-gated `Requires` (engines lacking the optional
  contracts auto-skip).

## Task scorecard (5 requested items)

1. **Badger vector+graph parity (S)** — DONE (implemented + tested).
2. **Vector at scale spike (L)** — NOT STARTED.
3. **Filtered k-NN (M)** — ~70% (core+4 engines done; core/pebble/bbolt
   tests missing).
4. **GraphRemoveEdge (M)** — ~80% (all 10 engines done; pg/mysql/dgraph/
   graphadapter/iroh tests missing).
5. **Directed-vs-undirected (S)** — ~80% (same engines as 4).

## What went wrong (and was fixed)

- Twice an `old_string` covering a whole function silently REPLACED it
  instead of appending (memory `GraphNeighbors`, sqlite
  `queryGraphNeighbors`) — both caught immediately by function-list checks
  and restored. Lesson: append via a unique tail anchor, never a full
  function body.
- First `executeVectorSearchFiltered` fallback filtered on nil metadata
  (impossible — results carry no metadata): replaced with the explicit
  unsupported error.
- sqlite graph.go hit 366 lines (350 CI cap) → split `graph_undirected.go`.
- `VectorFilterBackend` initially omitted `VectorBackend` embedding — test
  compile error, fixed by embedding.
- dgo field is `DelNquads`, not `DeleteNquads`.
- A CONCURRENT session's half-written `capability_surface_test.go`
  (references `metaengine.ComplexityNative`) temporarily broke core package
  test compilation; resolved when that session finished. Their files
  (capability_audit.go, explain.go, catalog/, ci.yml, flake.nix) are theirs
  — untouched by this session.
- Auto-commit daemon committed this session's in-flight work mid-wave
  (53e0f3263) — expected behavior.

## Open / next steps (in order)

1. Core metaengine feature tests: EdgeRemoval fold e2e, undirected query
   dispatch, filtered k-NN through the store (was mid-writing when this
   report was cut).
2. pebble + bbolt VectorSearchFiltered tests (mirror badger's).
3. pg/mysql/dgraph/graphadapter removal+undirected tests (skip cleanly
   without servers) + run adttest matrix across engines.
4. Vector-at-scale spike doc (quantization/HNSW) with measured brute-force
   baseline (Task 2 — only untouched item).
5. Bookkeeping gate: api-stability golden regen (exported surface changed —
   REQUIRED), TODO_LIST/CHANGELOG, skill references, doc-check, `nix fmt`,
   per-module tests, `nix run .#verify-fast`.

## Verification status

- Builds: metaengine, keycodec, adttest + all 10 engine modules — OK
  (`go build -tags "goexperiment.jsonv2"`, GOWORK=off per module).
- Tests run so far: badgerengine (full, green), sqliteengine (full, green).
- NOT yet run: core metaengine suite (was blocked, now unblocked), pebble,
  bbolt, iroh, adttest matrices, `#verify-fast`. No GREEN claim beyond the
  two engine suites.
