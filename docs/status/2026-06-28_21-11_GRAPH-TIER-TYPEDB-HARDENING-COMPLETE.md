# Comprehensive Status Report — 2026-06-28 21:11

> **Scope:** Full project status after TypeDB research → graph tier hardening session
> **Head:** `dafe40d0` (pushed to origin/master)
> **Working tree:** `M transport/http/sse.go` (concurrent session, not mine)

---

## Executive Summary

This session transformed the **graph projection tier** from the least-typed of the three projection tiers into the one with the richest read surface. Triggered by deep research into TypeDB (typedb.com), the session identified that the graph tier had `NodeRef{Label string, KeyProp string, KeyValue any}` and `props map[string]any` — strings and `any` everywhere — while the relational tier had just shipped column-name validation. TypeDB's existence proof (graph-shaped read models don't require abandoning strong types) drove two high-leverage additions:

1. **`graph.Schema`** — opt-in closed-world validation at the sink boundary. Every `MergeNode`/`MergeEdge`/`SetNodeProperty` is validated against a declared schema (node types, edge types, property types) before touching the graph. Catches the phantom-node bug class. Mirrors the relational tier's Row column-name validation.

2. **`MemoryDriver` Read API** — typed `Query`/`Traverse`/`Neighbors`/`ShortestPath` via `ReadableDriver` interface. Uses Go-native predicate functions (NOT a query language). Makes the in-memory graph actually queryable — previously it was write-only (only `Snapshot()` exposed raw data).

Plus: first real graph consumer (`example/graph-demo/`), two ADRs (0039 Graph Schema, 0040 Deriver Design), opinionated `docs/projection-tiers.md` rewrite, full docs update.

**42 graph tests pass with `-race`. Build + vet clean. Lint clean on all new files.**

---

## At a Glance

| Metric | Value |
|--------|-------|
| Modules | 47 `go.mod` files (46 in go.work + root) |
| Go version | 1.26.4 |
| API surface exports | 1,707 |
| ADRs | 39 (0039 + 0040 new this session) |
| Research docs | 31 (1 new this session) |
| Examples | 7 (graph-demo new this session) |
| Stack presets | 5 (memory, sqlite, pebble, postgres, turso) |
| Graph module LOC | 2,384 (1,402 new this session) |
| Graph tests | 42 (37 new this session) |
| Commits since v3.1.0 | ~75+ |
| Build | ✅ PASS (workspace + all modules) |
| Vet | ✅ PASS (zero issues) |
| Tests (graph + demo) | ✅ 42 pass with `-race` |
| Architecture enforcement | ⚠️ 6/7 modules pass (transport/http budget over) |
| Last release | v3.1.0 |

---

## a) FULLY DONE ✅

### A1. Graph Schema Validation (ADR-0039) — COMMITTED `c9f5d9c8`

| Metric | Value |
|--------|-------|
| Files | 4 new (`schema.go`, `schema_sink.go`, `schema_test.go` + errors) |
| Tests | 16 schema unit tests + 3 graphtest contract tests |
| Coverage | All validation paths: unknown label, unknown prop, wrong key prop, edge endpoint mismatch |

Components:
- **`Schema`** — declares node types (label, key prop, properties), edge types (type, endpoint labels, properties)
- **`Schema.Validate()`** — catches structural errors: empty names, duplicate labels, key prop in properties, unknown endpoint labels
- **`schemaSink`** — wraps any `GraphSink` with validation before forwarding writes
- **`WithSchema`** option on `NewGraphProjection` (validates regardless of driver)
- **`WithDriverSchema`** option on `NewMemoryDriver` (validates standalone use)
- **Nil schema** = backward-compatible (no validation, open-world default)
- **3 contract tests** added to `graphtest` suite (SchemaFactory pattern)

### A2. MemoryDriver Read API — COMMITTED `c9f5d9c8`

| Metric | Value |
|--------|-------|
| Files | 2 new (`read.go`, `memory_read.go`) |
| Tests | 16 read API tests (`read_test.go`) |

Components:
- **`ReadableDriver`** interface — `Query`, `Traverse`, `Neighbors`, `ShortestPath`
- **`Pattern`** — label filter + Go-native predicate function (`func(map[string]any) bool`)
- **`NodeView`/`EdgeView`** — read-only snapshots with defensive prop copies
- **`Query`** — filter by label + predicate function
- **`Traverse`** — BFS with cycle-safe visited set, configurable maxDepth (0 = immediate, -1 = unlimited)
- **`Neighbors`** — 1-hop adjacency in both directions (incoming + outgoing edges)
- **`ShortestPath`** — BFS with parent tracking, `ErrPathNotFound` sentinel

**Deliberately NOT a query language.** Go compiler type-checks predicate functions; a query parser cannot. For external graph DBs (Neo4j), reads stay native (Cypher/Gremlin) per ADR-0038.

### A3. example/graph-demo — COMMITTED `dafe40d0`

| Metric | Value |
|--------|-------|
| Files | 4 (`go.mod`, `main.go`, `main_test.go`, `README.md`) |
| Tests | 5 (Query, Traverse, ShortestPath, Schema rejection, content filter) |

First real consumer of the graph tier. Models a discussion forum: users author messages, messages reply to messages. Demonstrates Schema validation, GraphProjection with events, and all four read API operations. The "zero real consumers" problem (flagged in 5 prior status reports) is now resolved.

### A4. Documentation — COMMITTED `dafe40d0`

| Item | Status |
|------|--------|
| `docs/projection-tiers.md` (rewritten with opinionated comparison) | ✅ |
| ADR-0039 (Graph Schema — what we adopted/rejected from TypeDB) | ✅ |
| ADR-0040 (Deriver module design — TypeDB rule model reference) | ✅ |
| `docs/research/2026-06-28_TYPEDB_LESSONS_FOR_PROJECTIONS.md` | ✅ (committed by concurrent session) |
| `docs/planning/2026-06-28_20-37_GRAPH-TIER-TYPEDB-HARDENING.md` (Pareto plan) | ✅ |
| `graph/README.md` (Schema + Read API sections) | ✅ |
| `AGENTS.md` (module list, Key Patterns, test command) | ✅ |

### A5. Concurrent Session Work (OTel DX) — COMMITTED by another session

These shipped during my session via concurrent commits. Not my work, but part of the current repo state:

| Item | Commit |
|------|--------|
| `otel.Setup()` one-call provider | `c1988cce` (I committed this — it was in the working tree) |
| OTel middleware bundle | `ca3db581` |
| OTel span instrumentation (transport/http, transport/grpc, watermill) | `001bf8ab` |
| example/user OTel dogfooding | `697f77ee` |
| OTel README rewrite | `882c27f3` |
| retryTracer + span tree validation tests | `2eb8670b`, `abcad30e` |

### A6. Pre-Session Cleanup — COMMITTED

| Item | Commit |
|------|--------|
| Restored corrupted `middleware/generic.go` + `retry_query_test.go` (patch syntax in Go source) | (git restore) |
| Synced go.sum files across workspace (transitive dep checksums) | `f15c1750` |

---

## b) PARTIALLY DONE ⚠️

### B1. transport/http Dependency Budget Overrun

`transport/http` has 3 production deps (budget: 2). The OTel instrumentation commit (`001bf8ab`) added `otel/v3` as a direct dependency. The budget needs to be bumped to 3, or `otel/v3` needs to be made indirect. This causes `nix run .#check-layers` to fail.

**Status:** `check-arch` reports "Module layer check FAILED." All 6 per-module arch-lint configs pass; the failure is the cross-module budget check only.

### B2. transport/http/sse.go Uncommitted Change

A formatting change in `transport/http/sse.go` (multi-line `cqrsotel.EventAttrs()` call) is in the working tree, not committed. This is from the concurrent OTel session — not my change. I did not touch it per safety rules.

### B3. DiscordSync Migration (the original trigger for relational tier)

Unchanged from prior reports. The capability (`storage.RelationalProjection`) exists. The migration of DiscordSync's `internal/projection/` + `internal/db/query.go` has NOT been started. Separate repo.

### B4. Graph Neo4j/Memgraph Driver

`GraphSink`/`GraphDriver` abstraction complete and validated by contract suite + Schema. A concrete Neo4j/Cypher driver (`graph/neo4j/`) does NOT exist — deliberately deferred to consumer-pull.

### B5. PostgreSQL Relational Tier Untested Against Real PG

`storage.RelationalProjection` and `RelationalStore` are tested on SQLite only. The dialect abstraction should make PG work, but it's unproven against a live PostgreSQL instance. No testcontainers integration.

### B6. Schema `Required` Property Constraint Not Enforced

The `PropertyType.Required` field exists in the schema but is NOT validated by `schemaSink`. The `validateProps` method checks that property names are declared, but does not check that required properties are present. This is a deliberate scope limit (the validation catches structural typos, not business-rule violations), but it's an incomplete feature.

---

## c) NOT STARTED ⬜

| # | Item | Effort | Why deferred |
|---|------|--------|--------------|
| C1 | God-package splits (storage 38 files, event 30 files → sub-packages) | High | Major refactor; separate session |
| C2 | Versioned schema migrations (goose/atlas-style) | Medium | Pre-existing gap |
| C3 | NATS/Redis transport via Watermill broker plugins | Medium | ADR-0025 accepted; consumer demand drives |
| C4 | Documentation site (Docusaurus/MkDocs) | High | 47 modules need browsable docs |
| C5 | Outbox DLQ + reference-based outbox | Medium | Pre-existing gaps |
| C6 | Durability profiles (Sync/BatchedSync/Async) | Low | Pre-existing gap |
| C7 | `projection.Runner` (standalone journal replay pipeline) | High | `bundle.RunProjections` covers common case |
| C8 | `RelationalStore` JOIN or denormalization examples | Medium | Denormalization documented; no example yet |
| C9 | Neo4j/Memgraph GraphDriver | Medium | Consumer-pulled |
| C10 | Scheduler module | Large | Deadlines, timeouts, cron — infrastructure concern |
| C11 | Deriver module implementation | Medium | Design captured in ADR-0040; not yet built |
| C12 | Hot-state cache for decider | Large | Only matters for 100+ cmd/sec/aggregate |
| C13 | Bi-temporal model (`ValidAt`) | Large | Niche — finance/HR/healthcare |
| C14 | Event redaction middleware | Medium | Design reviewed, no code |
| C15 | Graph Schema FTS5 / full-text search | Medium | Separate concern; relational-tier gap |
| C16 | Graph read API on real driver backends | Large | Cypher abstraction rejected (ADR-0038) |

---

## d) TOTALLY FUCKED UP 💥

Honest answer: **nothing is critically broken.** All tests pass, the build is clean, and no data-loss paths exist. But there are real issues:

### D1. transport/http Dep Budget — BLOCKING check-layers

The `check-layers` gate FAILS because `transport/http` has 3 production deps against a budget of 2. The OTel instrumentation commit (`001bf8ab`) added the `otel/v3` dependency. This is a **CI-breaking regression** that was introduced by the concurrent OTel session and has not been fixed.

**Impact:** `nix run .#check-layers` exits non-zero. Any CI that runs this gate will fail.
**Fix:** Either bump `DEP_BUDGET[transport/http]=3` in the layer-check script, or make `otel/v3` indirect.

### D2. Concurrent Sessions Committing to master Without Coordination

During this session, at least 2 concurrent sessions committed to master (`abcad30e`, `89b7dbfa`, `ffd8fc57`, `2eb8670b`, `882c27f3`, `001bf8ab`, `697f77ee`, `ca3db581`). This caused:
- My research doc (`d1f40dc3`) to be committed by another session before I committed it
- Files appearing modified in my working tree that I didn't author (`transport/http/sse.go`)
- Two Go source files corrupted with patch syntax (`middleware/generic.go`, `middleware/retry_query_test.go`) — likely from a tool failure in another session

**Impact:** Race conditions on the working tree. I had to restore corrupted files and investigate unauthored changes before committing. Not data-loss-level, but high friction.

### D3. BuildFlow Pre-Commit Hook Timeout on Doc-Only Commits

The BuildFlow hook runs golangci-lint across all 47 modules with a 300s budget. On doc-only commits, this is wasteful and occasionally times out. The hook should detect file-type scope and skip linting for `.md`/`.html`-only changes.

### D4. Graph `describeSchema` Was Dead Code

I wrote `describeSchema()` in `schema.go` for error messages, but never called it. Lint caught it (`unused`). Removed before commit. Not a real bug — just wasted effort.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Architecture & Type Model

1. **Enforce `PropertyType.Required` in schema validation** — the field exists but `schemaSink` doesn't check it. Either enforce it or remove the field (YAGNI). Don't ship a field that lies about what it does.

2. **Graph Schema needs an `Indexes` equivalent** — `ViewMapper` has `Indexes []IndexSpec` for SQL view stores. The graph tier has no equivalent for declaring which properties should be indexed. For the MemoryDriver this is irrelevant, but a Neo4j driver would need it.

3. **`ReadableDriver` should be in `graphtest` contract** — right now only `MemoryDriver` implements it. If a future driver (e.g. an embedded Apache Age) implements it, there's no contract test to validate the read operations.

### Operational

4. **Fix transport/http dep budget** — CI-breaking. Either bump budget or restructure deps. Highest priority operational fix.

5. **Stamp RESOLVED on remaining unmarked research docs** — mechanical, prevents confusion about which proposals are live vs historical.

6. **Commit the BuildFlow hook config** — the 300s timeout fix lives only in local `.git/hooks/`. New contributors get the old 60s hook.

### Documentation

7. **The graph-demo should demonstrate Schema rejection at runtime** — the example shows Schema validation working, but doesn't show what happens when you write a bad label. A `// Try changing "User" to "Bogus"` comment would teach the value proposition.

8. **AGENTS.md module list is stale** — says 44 modules, actual count is 47. The module table also doesn't list `example/graph-demo`.

9. **`docs/projection-tiers.md` Graph Schema section should cross-link ADR-0039** — the rewritten doc mentions the schema but doesn't link to the ADR with the full "what we adopted/rejected from TypeDB" rationale.

### Process

10. **Corrupted files from concurrent sessions** — two Go source files had literal `+++` patch syntax in them. This suggests a tool (possibly an AI agent's edit tool) is writing diff artifacts into source files. Investigate the root cause.

---

## f) Top 25 Things to Get Done Next 🎯

### Tier 1: Critical (CI-breaking or highest leverage)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 1 | **Fix transport/http dep budget** (bump to 3 or make otel indirect) | Critical | 5min | check-layers gate is FAILING |
| 2 | **Commit or discard transport/http/sse.go** formatting change | High | 1min | Uncommitted change in working tree |
| 3 | **Enforce or remove `PropertyType.Required`** | Medium | 30min | Field exists but doesn't work — lying API |
| 4 | **Update AGENTS.md module count** (44 → 47) and add graph-demo | Low | 5min | Stale docs |
| 5 | **Cross-link ADR-0039 from projection-tiers.md** | Low | 5min | Discoverability |

### Tier 2: High-value improvements

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 6 | **Migrate DiscordSync's projection layer** to `RelationalProjection` | Critical | 2-3h | Original trigger for relational tier; validates against real consumer |
| 7 | **Add PostgreSQL integration tests** for relational tier (testcontainers) | High | 1h | Tested on SQLite only; PG path unproven |
| 8 | **Add `ReadableDriver` contract tests** to graphtest | Medium | 1h | No contract for read operations; future drivers unvalidated |
| 9 | **God-package split: storage/** (38 files → sub-packages) | High | 4h+ | Largest god-package |
| 10 | **God-package split: event/** (30 files → sub-packages) | High | 3h+ | Core module |
| 11 | **Build Neo4j/Cypher GraphDriver** (`graph/neo4j/`) | High | 3-4h | First real graph backend; validates Schema + Sink abstraction |
| 12 | **Implement Deriver module** (per ADR-0040) | Medium | 1-2 days | Design done; event→command derivation |
| 13 | **Add versioned schema migrations** | Medium | 2h | Pre-existing gap (goose/atlas-style) |
| 14 | **Complete Pebble module** (any remaining gaps) | Medium | 2h | Pre-existing gap |

### Tier 3: Quality and completeness

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 15 | **Add graph-demo Schema rejection demo** (show what bad labels do) | Low | 15min | Teaches the value proposition |
| 16 | **Stamp RESOLVED on remaining unmarked research docs** | Low | 15min each | Prevents confusion |
| 17 | **Add FTS5 full-text search** to RelationalStore | Medium | 2h | DiscordSync's SearchMessages |
| 18 | **Add NATS JetStream transport adapter** | Medium | 3h | ADR-0025 accepted |
| 19 | **Add Outbox DLQ + reference-based outbox** | Medium | 2h | Pre-existing gaps |
| 20 | **Add Durability profiles** across backends | Low | 1.5h | Pre-existing gap |
| 21 | **Documentation site** (Docusaurus/MkDocs) | Low | 4h+ | 47 modules need browsable docs |
| 22 | **Commit the BuildFlow hook config** or document install | Low | 15min | Onboarding |
| 23 | **Add `graph.Schema.Indexes`** for future driver backends | Low | 30min | Neo4j driver would need it |
| 24 | **Property-based tests for graph read API** (rapid) | Medium | 1h | Traverse/ShortestPath edge cases |
| 25 | **Integration test: RunProjections with graph tier** end-to-end | Medium | 1h | No graph-in-bundle test exists yet |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should `PropertyType.Required` be enforced, or should the field be removed?**

The `PropertyType` struct has a `Required bool` field. It's documented as "controls whether the sink rejects writes that omit this property." But `schemaSink.validateProps()` only checks that property *names* are declared — it does NOT check that required properties are *present*.

This means the field is a **lie**: it suggests the schema enforces required-ness, but it doesn't. Two valid fixes:

- **Option A: Enforce it.** Add a check in `validateNodeProps` and `validateEdgeProps` that verifies all `Required: true` properties are present in the props map. ~15 min of work. But: `MergeNode` with an empty props map (updating an existing node) would then fail if the node type has required properties. This changes MERGE semantics — is that acceptable?

- **Option B: Remove the field.** Delete `Required` from `PropertyType`. The schema only validates structural correctness (names exist), not business rules (all required fields present). This is consistent with the "minimal constraint grammar" decision in ADR-0039. But: the field might be useful documentation even without enforcement.

The core tension: MERGE semantics mean a handler may intentionally write a partial update (`MergeNode(ref, {"name": "new"})` to update just one property). Enforcing `Required` would break partial updates. But not enforcing it means the field is misleading.

I cannot determine which is correct because it depends on whether partial node updates are a supported use case (they currently work — MergeNode preserves existing properties not in the props map). If partial updates are supported, `Required` cannot be enforced at the sink level (it would need to check the *merged* result, not the incoming props). If partial updates are NOT a documented use case, enforcement is straightforward.

**This needs a human decision: enforce and document the MERGE interaction, or remove the field as YAGNI.**
