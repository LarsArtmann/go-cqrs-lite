# Status Report — go-cqrs-lite v3.4.0+ (Storage Split + Deriver Idempotency)

> **Date:** 2026-06-29 08:34
> **Head:** `e82e5c6e`
> **Modules:** 54 go.mod files
> **ADRs:** 42 | **Research docs:** 34 | **Status reports:** 32
> **Build:** ✅ | **Vet:** ✅ | **check-arch:** ✅ | **check-layers:** ✅
> **Session commits:** 13 (authored) + concurrent session commits interleaved
> **Consumers:** 27 projects across ~20 codebases

---

## a) FULLY DONE ✅

### Deriver Idempotency (complete end-to-end)

| Component                      | Status     | Detail                                                                                          |
| ------------------------------ | ---------- | ----------------------------------------------------------------------------------------------- |
| `id.DeriveCommandID`           | ✅ Shipped | SHA-256 → ULID with zeroed timestamp (epoch sentinel). `IsDerivedCommandID()` predicate.        |
| `deriver.Deriver.Idempotent()` | ✅ Shipped | Combinator that re-stamps derived commands with deterministic IDs from source event.            |
| `example/deriver` integration  | ✅ Shipped | Ghost system fixed — example now calls `.Idempotent()`, runs deriver twice, shows matching IDs. |
| Tests                          | ✅ 10 new  | 5 deriver Idempotent tests, 3 DeriveCommandID tests, 2 example tests.                           |

### Storage/ God-Package Split (2 of ~5 clusters extracted)

| Cluster                      | Status     | Files                                           | Root LOC removed                                    |
| ---------------------------- | ---------- | ----------------------------------------------- | --------------------------------------------------- |
| `storage/relational/`        | ✅ Shipped | 5 source + 3 test                               | ~1,200 LOC                                          |
| `storage/view/`              | ✅ Shipped | 7 source + 10 test                              | ~1,800 LOC                                          |
| Type aliases at root         | ✅ Working | `relational_aliases.go`, `view_aliases.go`      | Zero consumer migration                             |
| Generic constructor wrappers | ✅ Working | `NewSQLiteViewStore[V,K]`, `AutoMapper[V]` etc. | Go can't alias generic funcs; thin wrappers instead |

**Before:** 38 root files, 5,392 LOC in one package
**After:** 28 root files + 2 sub-packages with 12 files each

### CI / Architecture

| Item                        | Status                                                                           |
| --------------------------- | -------------------------------------------------------------------------------- |
| `check-module-layers.sh`    | ✅ All 54 modules enforced with correct LAYER + DEP_BUDGET                       |
| `transport/http` dep budget | ✅ Fixed (3→4)                                                                   |
| 7 missing modules added     | ✅ deriver, graph, projection, projectionhost, scheduling, scenario, idempotency |
| `check-arch`                | ✅ Passes                                                                        |
| `check-layers`              | ✅ Passes                                                                        |

### Tooling / DX

| Item                       | Status                                                   |
| -------------------------- | -------------------------------------------------------- |
| `scripts/install-hooks.sh` | ✅ Shared BuildFlow pre-commit hook with scope detection |
| `docs/v4-WISHLIST.md`      | ✅ 7 candidate breaks tracked, 4 triggers defined        |
| AGENTS.md                  | ✅ Synced to 54 modules, storage sub-packages documented |
| CONTRIBUTING.md            | ✅ Hook install step documented                          |

### Event/ Split: **DECISION: DO NOT SPLIT**

Analyzed and rejected. The data:

- **469 files** import `event/v4` (vs ~50 for storage/)
- **High cross-file coupling**: Metadata in 7 files, Option in 9 files
- **2,321 LOC / 29 files** — under 350-line CI limit
- **Cohesion is real**: every file serves the Event type

This is NOT a god-package. It's a cohesive core package. Splitting would create circular deps and break 469 importers for zero structural benefit.

---

## b) PARTIALLY DONE 🟡

### Storage/ split — 3 more clusters identified but not extracted

| Cluster                   | Files | LOC  | Dependency          | Priority                    |
| ------------------------- | ----- | ---- | ------------------- | --------------------------- |
| **event store**           | 6     | ~800 | sql/ only           | Medium                      |
| **command store**         | 5     | ~600 | sql/ only           | Medium                      |
| **query store**           | 4     | ~500 | sql/ only           | Low                         |
| **snapshot + checkpoint** | 2     | ~300 | sql/ only           | Low                         |
| **pg_bus**                | 3     | ~500 | sql/ only           | Low                         |
| **backend + helpers**     | 4     | ~400 | All stores (facade) | Last (depends on all above) |

The root package still has 28 files but they're now more cohesive — core SQL stores + the `SQLBackend` facade that ties them together. The biggest wins (relational + view, the two most independent clusters) are done.

### PG Integration Tests

Scaffold exists at `storage/relational/pg_test.go` (build-tagged `postgres_integration`). CI job missing — needs GitHub Actions service container with `POSTGRES_TEST_DSN`.

### Durability Profiles

Design done at `docs/research/2026-06-28_DURABILITY_PROFILES.md`. Sync/BatchedSync/Async not implemented.

---

## c) NOT STARTED ⬜

| ID  | Task                                         | Impact   | Effort                  |
| --- | -------------------------------------------- | -------- | ----------------------- |
| N01 | Neo4j/Cypher GraphDriver (`graph/neo4j/`)    | High     | 3-4h                    |
| N02 | NATS JetStream transport adapter             | Medium   | 3h                      |
| N03 | FTS5 full-text search for RelationalStore    | Medium   | 2h                      |
| N04 | Versioned schema migrations (goose/atlas)    | Medium   | 2h                      |
| N05 | Outbox DLQ + reference-based outbox          | Medium   | 2h                      |
| N06 | Bi-temporal model (`ValidAt`)                | Low      | Large                   |
| N07 | Event redaction middleware                   | Low      | Medium                  |
| N08 | Documentation site (Docusaurus/MkDocs)       | Low      | 4h+                     |
| N09 | DiscordSync → RelationalProjection migration | Critical | Blocked (separate repo) |

---

## d) TOTALLY FUCKED UP 💥 (Honest Assessment)

### D1: DeriveCommandID timestamp footgun (FIXED)

Initial implementation packed all 16 SHA-256 bytes into ULID including the 6-byte timestamp. `id.ULID()` returned year-2681 garbage dates that looked plausible. **Fix:** Zeroed timestamp portion (epoch 1970 sentinel), added `IsDerivedCommandID()`.

### D2: Ghost system — Idempotent() not in example (FIXED)

Shipped `Deriver.Idempotent()` but example/deriver didn't call it. Docstring promised idempotency; code didn't deliver. **Fix:** Added `.Idempotent()` to composition chain.

### D3: Banned libs false positive (SELF-CORRECTED)

Reported testify/yaml.v3/pkg/errors as policy violations. Grep filter bug — all were `// indirect` transitive deps via modernc.org/sqlite. Not violations.

### D4: Consumer count claim was wrong (CORRECTED)

Initially claimed "1 external consumer (DiscordSync)". User corrected with data: **27 consumer projects** across ~20 codebases. This fundamentally changed the v3.x-vs-v4 recommendation from "easy migration" to "27 projects would need path changes."

---

## e) WHAT WE SHOULD IMPROVE 🚀

### E1: Finish the storage/ split

28 root files remain. The next 3 clusters (event store, command store, query store) are straightforward — same pattern as relational/ and view/. ~3h to complete. The backend facade stays in root.

### E2: Tests for 5 zero-test modules

`cmd/doc-check`, `example/deployer-first-heterogeneous`, `example/deployer-first-multidb`, `example/encryption`, `example/projectionhost` — all have production code, zero tests. Examples are consumer-facing demos.

### E3: Stabilize concurrent session workflow

Multiple sessions committing to `master` caused commit-message mismatches and HEAD lock failures. Feature branches or serialized sessions would prevent this.

### E4: Generic constructor alias pattern is verbose

Go doesn't allow storing generic functions in package-level vars. The `view_aliases.go` wrapper functions work but add ~45 lines of boilerplate. A future Go version with generic type aliases might simplify this. Document the pattern for future extractions.

### E5: `buildWhereClause` duplicated across 3 packages

The 37-line WHERE-clause builder is now in `storage/view/count.go`, `storage/relational/sink.go`, and root `storage/view_store_count.go` (wait — that moved). It's in 2 packages now. Small, pure, stable logic — duplication is acceptable, but a shared `sqlutil` package could centralize it if it grows.

---

## f) Top #25 Things to Get Done Next

Sorted by **impact/effort** (Pareto order).

### P0 — Critical

| #   | Task                                                                                | Impact   | Effort |
| --- | ----------------------------------------------------------------------------------- | -------- | ------ |
| 1   | **Finish storage/ split** — extract event/command/query store clusters              | High     | 3h     |
| 2   | **Tests for example/projectionhost** — consumer-facing reliability demo, zero tests | High     | 30min  |
| 3   | **Stabilize concurrent session workflow** — use branches or serialize               | Critical | 5min   |

### P1 — High Leverage

| #   | Task                                                                           | Impact   | Effort  |
| --- | ------------------------------------------------------------------------------ | -------- | ------- |
| 4   | **PG integration tests in CI** — scaffold exists, needs service container      | High     | 1h      |
| 5   | **Neo4j GraphDriver** (`graph/neo4j/`) — Schema.Indexes ready, consumer-pulled | High     | 3-4h    |
| 6   | **Durability profiles** — design done, ADR exists                              | Medium   | 1.5h    |
| 7   | **Outbox pattern DLQ** — ADR-0042/0043 discuss direction                       | Medium   | 2h      |
| 8   | **DiscordSync → RelationalProjection** — now that relational/ is extracted     | Critical | Blocked |

### P2 — Quality

| #   | Task                                                       | Impact | Effort |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 9   | **FTS5 full-text search** for RelationalStore              | Medium | 2h     |
| 10  | **Versioned schema migrations** (goose/atlas)              | Medium | 2h     |
| 11  | **Tests for cmd/doc-check** — CI tool, zero tests          | Medium | 20min  |
| 12  | **Tests for example/encryption** — crypto demo, zero tests | Medium | 20min  |
| 13  | **Tests for example/deployer-first-multidb**               | Low    | 30min  |
| 14  | **Tests for example/deployer-first-heterogeneous**         | Low    | 30min  |
| 15  | **NATS JetStream transport adapter**                       | Medium | 3h     |

### P3 — Polish / Long-Term

| #   | Task                                            | Impact | Effort                                       |
| --- | ----------------------------------------------- | ------ | -------------------------------------------- |
| 16  | **Documentation site** (Docusaurus/MkDocs)      | Low    | 4h+                                          |
| 17  | **Hot-state cache for decider** — profile first | Low    | Large                                        |
| 18  | **Event redaction middleware**                  | Low    | Medium                                       |
| 19  | **Bi-temporal model** (`ValidAt`)               | Low    | Large                                        |
| 20  | **Event-history visualization**                 | Low    | Large                                        |
| 21  | **Scheduler expansion**                         | Low    | Large                                        |
| 22  | **Graph read API on real backends**             | Low    | Large                                        |
| 23  | **Event/ split**                                | None   | **DO NOT DO** — 469 importers, high coupling |
| 24  | **Transport/grpc genproto fix**                 | Medium | Blocked (cockroachdb/errors#79)              |
| 25  | **v4 major version cut**                        | None   | **DO NOT DO** — v4-WISHLIST tracks triggers  |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should the storage/ split continue into separate go.mod files, or stay as sub-packages within one module?"

**Context:** Right now `storage/relational/` and `storage/view/` are sub-packages within the single `storage/v4` go.mod. This means a consumer who only needs `storage/relational/` still pulls in the full storage module dependency tree (modernc.org/sqlite, all the event/command/etc deps).

**Option A (current):** Sub-packages within one go.mod. Simple. Consumers import `storage/v4/relational`. Full dep tree comes along.

**Option B:** Separate go.mod files. `storage/relational/v4` becomes its own module. Consumers who only need relational projections import just that module + its minimal deps (event, kv, projection, sql). Lighter dependency footprint.

**Why I can't decide:** The library's design principle says "minimal dependencies — each module has its own go.mod with only needed deps." Option B aligns with this. BUT: splitting go.mod files for sub-packages within a directory creates the `eventtest` nested-module problem (go mod tidy warnings, tooling friction). And the type aliases at root would need replace directives to work across module boundaries.

The tradeoff is between dependency minimization (Option B) and tooling simplicity (Option A). This is an architecture decision that affects every future cluster extraction.
