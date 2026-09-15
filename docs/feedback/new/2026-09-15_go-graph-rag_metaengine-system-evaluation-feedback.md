# go-graph-rag Consumer Feedback: metaengine/system Evaluation Findings

**Date:** 2026-09-15
**Consumer:** `github.com/larsartmann/go-graph-rag` (public GraphRAG SDK, Go 1.26.7, dep policy: stdlib + `samber/lo` + `modernc.org/sqlite`)
**Author:** Crush (AI), from a full adoption evaluation of `metaengine/v4` (v4.13.0) and `system/v4` (v4.7.0)
**Full report:** `go-graph-rag/docs/research/2026-09-15_metaengine-system-adoption.md` (277 lines, every claim file:line-cited)
**Verdict that motivated this feedback:** adopted neither — `system` is a category error for a library, `metaengine` is capable but wrong-shaped for GraphRAG. The items below are what would have changed that verdict, or what the evaluation exposed as library-level gaps independent of graphrag.

---

## Summary of requests

| # | Request                                                                                                            | Category    | Severity | Effort | Blocks whom                            |
| - | ------------------------------------------------------------------------------------------------------------------ | ----------- | -------- | ------ | -------------------------------------- |
| 1 | Ship an ANN vector engine behind `VectorBackend`                                                                   | Capability  | High     | L      | Every RAG/similarity consumer          |
| 2 | Fail closed on grouped materialized views (ADR-0135 caveat)                                                        | Correctness | High     | S      | Turso users of `WithMaterializedViews` |
| 3 | Fail closed on `EventAdapter.Save` racy fallback                                                                   | Correctness | High     | S      | Third-party engine authors             |
| 4 | Cut `system`'s engine requires from the consumer module graph                                                      | Dep hygiene | High     | M      | sqlite/memory-only `system` consumers  |
| 5 | Stamp experimental status in each module's `doc.go`                                                                | Docs truth  | Medium   | S      | All consumers (pkg.go.dev readers)     |
| 6 | Close the test-mass gap on `system`                                                                                | Reliability | High     | L      | v5 migration (ADR-0123)                |
| 7 | Richer Graph edges (relation label, weight) — v5 material                                                          | Capability  | Medium   | M      | Graph/DAG read-model consumers         |
| 8 | Smaller batch: fold-error messages, publish modules reference, plan-diff CI recipe, cross-engine apply diagnostics | DX / Docs   | Low-Med  | S each | Various                                |

---

## 1. Ship an ANN vector engine

### The Problem

Every shipped vector path is brute-force:

- Memory: "computes distances on every search — O(N·D) per query… Suitable for small
  collections (<10K vectors)… for production scale, use an engine with ANN search (HNSW, PQ)"
  (`metaengine/vector_search.go:152-156`).
- sqliteengine: pure-Go path scores in Go; libSQL pushdown via `vector_distance_*` is
  "same O(N) complexity" (`metaengine/sqliteengine/vector.go:12-27`).
- The seam exists and invites implementations — `VectorBackend` documents "brute-force
  (Memory), HNSW, or product quantization" (`vector_search.go:59-61`) — but no ANN engine
  ships anywhere in the repo.

### The Impact

This was the single decisive reason go-graph-rag did not adopt metaengine. The SDK already
has a linear cosine scan in ~120 lines; adopting a ~10-module dependency that computes the
same O(N·D) scan buys nothing. Every consumer whose read model is "embed and rank" — RAG,
dedup, similarity search, recommendation — hits this wall at 10-50K vectors and must leave
the ecosystem (sqlite-vec, hannoy, or hand-rolled HNSW) for exactly the ADT metaengine
already models (`Embedding`, collections, filtered k-NN with pre-ranking filters — the API
design is right; the engines aren't there yet).

### The Fix (options, best first)

1. `sqliteengine` optional libSQL `vector32()` tables when the driver supports F32_BLOB —
   the pushdown plumbing (`sqliteengine/vector.go:61-70`) is already half-written.
2. A separate `metaengine/vecengine` module wrapping `sqlite-vec`/`hannoy` (separate module
   per the established ADR-0086-family dep-isolation pattern — heavy deps opt-in).
3. In-process HNSW port as a `memory`-family engine.

Until then: state the <10K guidance in the module README, not only in code comments —
consumers evaluating "is metaengine's Vector production-ready?" read the README.

---

## 2. Fail closed on grouped materialized views

### The Problem

ADR-0135 (Accepted) documents that Turso grouped materialized views "return silently wrong
SUMs after a second transaction" — scalar views exact, grouped views carry an upstream
correctness caveat. The spec surface (`WithMaterializedViews(specs...)`, `(collection, fn,
column, groupBy?)`) happily accepts `groupBy` today.

### The Impact

Silent-wrong is the worst failure class a storage library can offer. The durability
subsystem already holds itself to the right standard — "a silently dropped durability tier
is a durability lie" (`metaengine/durability.go:63-93`) — but the newest feature ships a
documented-silent-wrong path as a default-on option.

### The Fix

Make grouped specs opt-in behind an explicit acknowledgment, e.g.
`WithKnownGroupedViewBug()` / `MaterializedViewSpec.UnsafeAllowGrouped = true`, with the
Doctor already surfacing which views are grouped (ADR-0135's Doctor integration makes the
visible half easy). Flip the default back to serving-from-scratch when upstream Turso fixes
it. Cheapest correctness win on this list.

---

## 3. Fail closed on `EventAdapter.Save` racy fallback

### The Problem

`system/doc.go:1-31` documents the capability ladder: `AtomicAppender` → ATOMIC,
`Transactional` → transactional, neither → **racy** bare check-then-append — "do NOT rely
on it under concurrency."

### The Impact

The library's own documentation says the fallback is unusable under concurrency, yet
construction accepts such engines silently. Third-party engine authors (exactly the audience
the driver registry invites) will ship engines without either capability and their users
will get interleaved/duplicated appends with no signal — the failure is rare, load-dependent,
and blamed on the consumer's code.

### The Fix

Reject at `system.New` construction (or at engine registration): an engine serving a
`RoleSourceOfTruth` instance without `AtomicAppender` or `Transactional` is a config error,
same family as `ErrDurabilityConflict`. Keep the documented ladder for non-source-of-truth
roles if any legitimate case exists — but the write path should never be racy-by-default.

---

## 4. Cut `system`'s engine requires from the consumer module graph

### The Problem

`system/go.mod` directly requires `metaengine/{badgerengine,pebbleengine,pgengine,sqliteengine}`
plus watermill, koanf (yaml/env/file), otter, and their transitive tree (otel, prometheus
client, sentry, cbor, pebble, badger, pgx…). ADR-0123 §3 already plans the fix for v5:
"consumers blank-import engines; `system/` never imports engine modules."

### The Impact

A memory/sqlite-only consumer of `system` — the quickstart's own default deployment —
inherits the largest dependency tree in the ecosystem for engines it will never open.
For libraries this is disqualifying (go-graph-rag's policy break was measured against
exactly this), and for apps it bloats `go.sum`, vendor dirs, nix FODs, and scanner surface.
The v5 design is right; the v4 graph doesn't have to wait for it.

### The Fix

1. Move the engine-requiring integration tests (`integration_badger_test.go`,
   `integration_postgres_test.go`, pebble/pg legs) into a separate `system/integration`
   module (or `systemtest`), the pattern `metaengine/enginetest` and `storage/backuptest`
   already established for exactly this.
2. Keep at most the `sqliteengine` require if `system` itself constructs sqlite drivers
   outside tests — otherwise rely on blank imports per ADR-0123 and ship the pattern one
   release early.

---

## 5. Stamp experimental status in each module's `doc.go`

### The Problem

`FEATURES.md:1442-1464` marks metaengine (+ every engine submodule) and system as
🧪 Experimental. Grep for "EXPERIMENTAL/Experimental" in `system/*.go` + README: **zero
hits** — same for metaengine. The status lives only in a repo-level file the module
consumer never sees.

### The Impact

pkg.go.dev is the consumer's view of a module. An `import ".../system/v4"` reader has no
signal that the API is v5-churn-exposed (`On`/`OnTyped` removal, driver-registry moves,
`RecordAwareFold` compat shim). Downstream tooling (my own session's skill reference,
for one) then disagrees with the repo about where the marking lives — two sources of truth,
one of them invisible.

### The Fix

One paragraph in each experimental module's package doc (`doc.go` / primary file): status,
what churn to expect, v5 pointer to ADR-0123. Remove the paragraph on graduation. This also
makes `cmd/doc-check` able to verify it, if desired.

---

## 6. Close the test-mass gap on `system`

### The Problem

Measured this session: metaengine — 155 test files in its root package alone, ≥355 across
subpackages, including a 10M-event soak with heap assertions, fold-classifier fuzz,
catch-up/quarantine stress, restart-idempotency. system — `system_test.go` holds 3 tests
(FullCQRSRoundtrip, Journal, Close) plus a handful of integration files (sqlite lifecycle,
badger, postgres env-gated, shutdown ordering).

### The Impact

ADR-0123 makes `system.System` the **sole** composition root at v5 and deletes `stack`.
The module carrying the library's entire future is its thinnest-tested critical path:
config-loader parsing (koanf merge, env override edge cases, `__` nesting), lifecycle
drain/shutdown ordering under load, `MultiBus`/named-publisher fan-out, role/durability
conflict detection, and `cqrs.yaml` round-trips are all correctness-critical and largely
untested relative to their blast radius.

### The Fix

Metaengine-parity pass, prioritized: (1) config-loader table + fuzz (it parses operator
input — untrusted-shaped by definition), (2) lifecycle/shutdown stress with real engines,
(3) determinism test (same domain+deployment → identical wiring), (4) `benchkit.FactoryFromSystem`
already exists — wire it into CI as a regression gate. Steal metaengine's soak pattern for
a long-running `system` endurance test.

---

## 7. Richer Graph edges (v5 material)

### The Problem

`metaengine/types.go:45-48`: `Edge{From, To any}` — no relation label, no weight. Graph
reads are `GraphNeighbors(node, depth)` (memory BFS; sqlite recursive CTEs — nice work,
`sqliteengine/graph.go:40-47`).

### The Impact

Real event-sourced graphs carry semantics on edges: causation DAGs edge types, similarity
graphs carry scores, permission graphs carry labels. Consumers today either (a) run one
graph collection per relation type (N indexes, no mixed traversal), or (b) smuggle
weight/label into side Map collections keyed by `from|to` — doubling the write path and
losing single-edge atomicity. go-graph-rag's core edge type is
`{Source, Target, Relation, Weight}` with derived `RelationSimilar` weighted edges; the
metaengine ADT cannot express it.

### The Fix

Not for v4 (breaking). v5 candidate: `Edge{From, To}` → `Edge[From, To, Payload]` or
`Edge{From, To; Kind string; Weight float64; Meta map[string]any}` with a documented
migration (fold return type still IS the ADT — the signature table grows two rows).
Alternative smaller step: `GraphAddEdgeWeighted` capability interface (like
`VectorSearchFiltered`'s optional-capability pattern) so engines can opt into weights
without breaking the base ADT.

---

## 8. Smaller batch

1. **Fold-signature error messages.** The reflection-based fold inference
   (`fold_inference.go`, `infer_*.go`) is the most magic API surface; it already needs
   `FuzzFoldClassifier` to prove no panics. On a classifier mismatch, embed the README's
   signature→ADT table verbatim in the error so the fix is self-service.
2. **Publish the module reference.** The skill's `modules.md` (one-liners + import paths +
   decision matrices) is the best consumer-facing doc in the ecosystem — and it lives in
   `~/.config`, invisible to third parties. The `catalog/docserver` infra already exists to
   host it.
3. **Plan-diff as a documented CI recipe.** `plan_diff.go` exists; a cookbook/recipe showing
   consumers how to pin `PlanResult`/`SerializablePlan` in-repo would turn engine/dep
   upgrades from "trust the planner" into a reviewable diff.
4. **Cross-engine apply diagnostics.** Cross-engine 2PC is out (documented, agreed). But
   Doctor/ExplainPlan could warn when one event type's folds span engines with no shared
   transaction — today that partial-failure window is invisible until it bites.

---

## What is already excellent (keep exactly as is)

Calibration, so the requests above aren't read as doom:

- **Durability honesty** — `RejectDurabilityTier` and "a silently dropped tier is a
  durability lie" is the best durability stance I've seen in a Go storage library. Items 2
  and 3 above are just this principle applied consistently to the two newest surfaces.
- **Dep-isolation pattern** — separate modules for heavy engines (pebbleengine, duckdbengine,
  badgerengine) is exactly why the ecosystem is consumable at all. Item 4 asks it be applied
  to `system`'s test-only requires.
- **ADR discipline** — ADR-0091/0097 (SSE de-dup), ADR-0115 (soak relocation), ADR-0135's
  honest caveats. The paper trail is why this feedback could be precise.
- **`cqrs-upgrade`** — consumer-side pin sweep + v5-removed-API reporting is rare maturity.
- **metaengine's test culture** — soak/fuzz/restart/catch-up-stress is the bar item 6 asks
  `system` to meet.

---

## Consumer context (why this lens)

go-graph-rag is a dependency-light retrieval SDK (documents → embeddings → typed weighted
graph → hybrid search), extracted from the CV repo 2026-09-15. Its evaluation asked:
"can metaengine's Vector + Graph ADTs replace our hand-rolled SQLite store and in-memory
scan?" Answer: not today — brute-force vectors (item 1) and label/weight-less edges (item 7)
are the two hard capability gaps; items 2-6 are library-level findings the evaluation
exposed along the way. The CV application (SUPERB evented-funnel-core plan) is separately
adopting `system`+`metaengine` at the app layer with T01/T29 spikes — that adoption is
unaffected by this feedback and is the right venue for it.
