# Status Report — go-cqrs-lite v3.4.0+

> **Date:** 2026-06-29 06:28
> **Head:** `ecabb39a`
> **Modules:** 54 go.mod files
> **ADRs:** 42 | **Research docs:** 34
> **Build:** ✅ | **Vet:** ✅ | **check-arch:** ✅ | **check-layers:** ✅

---

## a) FULLY DONE ✅

### This Session (7 commits)

| Commit | Description |
|--------|-------------|
| `259fca9b` | **DeriveCommandID** in id/ (SHA-256→ULID) + **Idempotent()** combinator on Deriver + 7 missing modules in check-module-layers.sh + transport/http budget fix |
| `abb7d2ae` | BuildFlow golangci-lint auto-fixes (sync.OnceValue, named ErrMeterRequired, redundant bounds check) |
| `343c96d6` | `scripts/install-hooks.sh` — shared BuildFlow pre-commit hook with scope detection |
| `cb374587` | AGENTS.md synced to 54 modules: module list, test command, layer graph |
| `e3fc6ee5` | nolint directives for intentional patterns |
| `ce786145` | (concurrent) DeriveCommandID timestamp fix (zeroed ULID timestamp as sentinel) + scheduler/watermill/middleware fixes |
| `ecabb39a` | (concurrent) example/deriver Idempotent() integration + encryption nonce warning |

### Key Deliverables

1. **Deriver idempotency closed** — `id.DeriveCommandID(namespace, keys...)` produces deterministic CommandIDs from SHA-256. The `Deriver.Idempotent()` combinator re-stamps derived commands with deterministic IDs. `id.IsDerivedCommandID()` distinguishes derived (epoch timestamp) from fresh IDs. Example/deriver demonstrates it end-to-end.

2. **CI blockers resolved** — `check-module-layers.sh` now enforces all 54 modules with correct LAYER + DEP_BUDGET values. Both `check-arch` and `check-layers` pass.

3. **BuildFlow hook shared** — `scripts/install-hooks.sh` installs the scope-detection pre-commit hook. New contributors get it via one command.

4. **Timestamp footgun fixed** — DeriveCommandID zeroes the ULID timestamp portion (epoch 1970 sentinel) so `id.ULID()` returns a recognizable "not a timestamp" instead of garbage dates (year 2681).

---

## b) PARTIALLY DONE 🟡

| Item | Status | Detail |
|------|--------|--------|
| **Storage/ split** | Design done | 39 files → 8 sub-packages with type aliases. 15 importers. Deferred — needs dedicated session. |
| **PG integration tests** | Scaffold exists | `storage/relational_pg_test.go` build-tagged. CI job missing. Needs POSTGRES_TEST_DSN service container. |
| **Durability profiles** | Design done | `docs/research/2026-06-28_DURABILITY_PROFILES.md`. Sync/BatchedSync/Async not implemented. |
| **Documentation site** | Not started | 54 modules need browsable docs. Docusaurus/MkDocs. |
| **DiscordSync migration** | Blocked | Separate repo. RelationalProjection is the target. |

---

## c) NOT STARTED ⬜

| ID | Task | Impact | Effort |
|----|------|--------|--------|
| N01 | Neo4j/Cypher GraphDriver (`graph/neo4j/`) | High | 3-4h |
| N02 | NATS JetStream transport adapter | Medium | 3h |
| N03 | FTS5 full-text search for RelationalStore | Medium | 2h |
| N04 | Versioned schema migrations (goose/atlas) | Medium | 2h |
| N05 | Outbox DLQ + reference-based outbox | Medium | 2h |
| N06 | Bi-temporal model (`ValidAt`) | Low | Large |
| N07 | Event redaction middleware | Low | Medium |
| N08 | Event-history visualization tools | Low | Large |

---

## d) TOTALLY FUCKED UP 💥 (Honest Assessment)

### D1: DeriveCommandID timestamp footgun (FIXED)
**What:** Initial `DeriveCommandID` packed all 16 SHA-256 bytes into the ULID, including the 6-byte timestamp. `id.ULID()` returned year-2681 garbage dates that looked plausible.

**Fix:** Zeroed the timestamp portion. `id.ULID()` now returns epoch 1970. Added `IsDerivedCommandID()` predicate.

**Lesson:** When the type system forces a ULID-backed type, the timestamp portion MUST be handled explicitly for derived IDs. Don't silently produce plausible-looking garbage.

### D2: Ghost system — Idempotent() not used in example (FIXED)
**What:** Shipped `Deriver.Idempotent()` in the library but the example/deriver (the demo that's supposed to showcase it) didn't call it. The docstring PROMISED idempotency ("re-processing the same event yields the same commands") but the code didn't deliver.

**Fix:** Added `.Idempotent()` to the composition chain. Example now runs the deriver twice and shows matching IDs.

**Lesson:** A feature that isn't demonstrated in its own example module is a ghost system. Always wire new features into their demos before committing.

### D3: Initial scan false positive on banned libs
**What:** Reported testify/yaml.v3/pkg/errors as "banned library violations" in examples. The grep filter (`grep -v "indirect"`) was applied to filenames, not dep lines. ALL deps were actually `// indirect` (transitive via modernc.org/sqlite).

**Lesson:** Verify scan results before reporting. The grep was structurally wrong — filtering filenames through a content filter.

---

## e) WHAT WE SHOULD IMPROVE 🚀

### E1: Concurrent session collision risk
Multiple sessions committing to `master` simultaneously caused:
- Commit messages mismatched to staged files (concurrent commits swept up my staged changes)
- HEAD lock failures
- BuildFlow auto-fixes re-applied to already-committed files

**Fix:** Use feature branches (`git switch -c my-feature`) or serialize sessions. The CONTRIBUTING.md already warns about this.

### E2: 5 modules with zero tests
`cmd/doc-check`, `example/deployer-first-heterogeneous`, `example/deployer-first-multidb`, `example/encryption`, `example/projectionhost` have production code but no test files. Examples are consumer-facing demos — they should at minimum have smoke tests.

### E3: DeriveCommandID entropy reduced to 80 bits
The timestamp-zeroing fix reduces the hash space from 128 bits to 80 bits (10 bytes randomness). This is still 2^80 — far beyond any collision risk for idempotency keys — but worth documenting for consumers who might worry.

### E4: `command.Metadata.Custom` is `map[MetadataKey]string`
All custom metadata values are strings. This is stringly-typed. A future improvement could use generics or a typed value union. Low priority — the current design is simple and works.

### E5: Type model — `DeriveCommandID` can't be distinguished at the type level
Unlike `DeriveAggregateID` (which produces a string-backed ID, structurally different from ULID-backed), `DeriveCommandID` produces a ULID-backed ID that's the same type as `NewCommandID`. The only way to distinguish is `IsDerivedCommandID()` at runtime. A phantom-type approach (like the marker pattern used throughout the id package) could make this compile-time, but would require splitting CommandID into two types — breaking change.

---

## f) Top #25 Things to Get Done Next

Sorted by **impact/effort** (Pareto order).

### P0 — Critical (blocks correctness or CI)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 1 | **Stabilize concurrent session workflow** | Critical | 5min | Use branches or serialize. Prevents commit collision/data loss. |
| 2 | **Tests for example/projectionhost** | High | 30min | DLQ demo has zero tests — consumer-facing reliability demo. |
| 3 | **Tests for example/encryption** | Medium | 20min | Encryption demo has zero tests. |

### P1 — High Leverage

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 4 | **PG integration tests in CI** (GitHub Actions service container) | High | 1h | Scaffold exists at `storage/relational_pg_test.go`. Just needs CI wiring. |
| 5 | **Storage/ god-package split** (39 files → 8 sub-packages + aliases) | High | 4h | Biggest god-package. 15 importers. Type aliases keep backward compat. |
| 6 | **Neo4j GraphDriver** (`graph/neo4j/`) | High | 3-4h | Graph Schema.Indexes ready. Consumer-pulled (DiscordSync). |
| 7 | **Durability profiles implementation** (Sync/BatchedSync/Async) | Medium | 1.5h | Design done. ADR exists. |
| 8 | **Outbox pattern DLQ** | Medium | 2h | ADR-0042/0043 discuss direction. |

### P2 — Quality Improvements

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 9 | **FTS5 full-text search** for RelationalStore | Medium | 2h | DiscordSync SearchMessages. |
| 10 | **Versioned schema migrations** (goose/atlas-style) | Medium | 2h | Currently embeds raw .sql DDL. |
| 11 | **Tests for cmd/doc-check** | Medium | 20min | Zero tests on a CI tool. |
| 12 | **Tests for example/deployer-first-multidb** | Low | 30min | Zero tests. |
| 13 | **Tests for example/deployer-first-heterogeneous** | Low | 30min | Zero tests. |
| 14 | **NATS JetStream transport adapter** | Medium | 3h | ADR-0025. |
| 15 | **DiscordSync → RelationalProjection migration** | Critical | 2-3h | Blocked — separate repo. |

### P3 — Polish

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 16 | **Documentation site** (Docusaurus/MkDocs) | Low | 4h+ | 54 modules need browsable docs. |
| 17 | **projection.Runner** standalone | Low | 30min | YAGNI — bundle.RunProjections covers common case. |
| 18 | **Hot-state cache for decider** | Low | Large | Snapshot+page-cache may suffice. Profile first. |
| 19 | **Event redaction middleware** | Low | Medium | Design reviewed. |
| 20 | **Bi-temporal model** (`ValidAt`) | Low | Large | Niche — finance/HR/healthcare. |
| 21 | **Event-history visualization** | Low | Large | Research doc OPEN. |
| 22 | **Scheduler expansion** | Low | Large | scheduling/ already shipped Timer[P] generics. |
| 23 | **Graph read API on real driver backends** | Low | Large | Cypher abstraction rejected (ADR-0038). |
| 24 | **event/ god-package split** | Low | 3h | NOT RECOMMENDED — 197 importers, high blast radius. |
| 25 | **Transport/grpc genproto fix** | Medium | Blocked | cockroachdb/errors#79. Pre-existing, weeks old. |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should `DeriveCommandID` use a phantom type to distinguish derived IDs at compile time?"

**Context:** Currently `CommandID = Of[CommandMarker]` — one type for all command IDs. `DeriveCommandID` produces the same type. The only way to tell them apart is `IsDerivedCommandID()` at runtime.

**Option A (current):** Runtime predicate. Simple. One CommandID type everywhere. Zero breaking changes.

**Option B (phantom type):** Introduce `DerivedCommandMarker` so `id.Of[DerivedCommandMarker]` is a distinct type. Compile-time safety: you literally cannot use a derived ID where a fresh one is expected (or vice versa). BUT: this is a breaking change for every consumer that uses `CommandID` as a parameter type, and the two types would need conversion functions.

**Option C (generic wrapper):** `CommandID` stays as-is. Add `type Derived[T any] struct{ ID CommandID }` as a wrapper type. Non-breaking. But adds ceremony.

**Why I can't decide:** The library's design philosophy says "make impossible states unrepresentable" — Option B delivers that. But the AGENTS.md also says "deleting external-facing API is breaking the product" — Option B breaks every consumer. The tradeoff between type safety and backward compatibility is a product decision, not a technical one.

**What I recommend:** Stay with Option A (runtime predicate) for v3.x. If we ever do a v4 with breaking changes, Option B is the right call. Document `IsDerivedCommandID()` prominently so consumers know to check.

---

## Self-Review Checklist

| Question | Answer |
|----------|--------|
| What did you forget? | Integrated Idempotent() into example/deriver only AFTER the brutal review caught it as a ghost system. Should have done it in the same commit. |
| What's stupid? | DeriveCommandID initially produced garbage timestamps (year 2681). Fixed by zeroing the timestamp portion. |
| What could be better? | DeriveCommandID has 80 bits of entropy instead of 128. Still safe (2^80 >> collision risk) but worth noting. |
| Did you lie? | Initial "banned libs in examples" report was a false positive from a grep bug. Corrected. |
| Ghost systems? | Idempotent() was a ghost — shipped in library but not wired into example. Fixed. |
| Split brains? | None found. check-module-layers.sh is now the single source of truth for layer assignments. |
| Tests? | Added 8 new tests (5 deriver Idempotent, 3 DeriveCommandID). 5 modules still have zero tests. |
| Scope creep? | No. Stayed focused on the deriver idempotency story + CI fixes + docs sync. |
