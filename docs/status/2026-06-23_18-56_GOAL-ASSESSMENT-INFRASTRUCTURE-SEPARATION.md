# Goal Assessment: Consumer/Deployer Infrastructure Separation

**Date:** 2026-06-23 18:56
**Goal:** *"Consumers should NOT decide on infrastructure — deployer chooses where data lives. Library should have recommendations (which DB fits which concern). Must support SQLite + Memory, ideally multiple SQLite DBs (1 for Command+ES, 1 for Query, 1 for views)."*

---

## Executive Summary

**Verdict: Architecture = strong (B+); Adoption = zero (F); Flagship feature was broken (now fixed).**

The `stack.Bundle` abstraction is an excellent realization of the goal — 5 presets, contract-tested parity, one-line backend swap. But **none of the three real consumers use it** (DiscordSync, SEC, cqrs-htmx/usermgmt all hand-wire infrastructure), and the goal's flagship feature — the multi-DB SQLite split — **was silently broken**: `WithEventDB`/`WithQueryDB` both dumped all 5 stores into one DB, and the event DB sat unused.

**Fixed this session:** Multi-DB routing bug (sqlite + turso), stale docs, missing recommendations guide.

---

## a) FULLY DONE ✓

| Item | Status | Evidence |
|------|--------|----------|
| Multi-DB routing fix (sqlite) | ✅ Fixed + tested | `openEventStores` + `openQueryStores` replace `openSecondaryStores`; `TestMultiDB_Routing` proves events land in event DB, commands in query DB |
| Multi-DB routing fix (turso) | ✅ Fixed + tested | Same fix, same regression test pattern |
| Infrastructure Recommendations doc | ✅ Written | `docs/INFRASTRUCTURE_RECOMMENDATIONS.md` — the "which DB for which concern" guide |
| PRESETS.md updated | ✅ Fixed | Turso added, multi-DB split documented, stale `readmodel` refs replaced with `kv` |
| sqlite doc.go fixed | ✅ Fixed | Phantom `AppendLog`/`Views` replaced with real `WithEventDB`/`WithQueryDB`/`WithViewDB` |
| docs/README.md index | ✅ Updated | New guides linked |
| All stack tests pass | ✅ Verified | sqlite, turso, memory, pebble, postgres — all green |

---

## b) PARTIALLY DONE ⚠️

| Item | What's done | What's missing |
|------|-------------|----------------|
| Stack preset coverage | 5/5 presets exist (memory, sqlite, pebble, postgres, turso) | Postgres lacks multi-DB split options (sqlite + turso have them) |
| Multi-DB contract testing | SQLite + Turso have routing tests | No multi-DB contract test in `contracttest.RunSuite` |
| Consumer gap documentation | Identified all 3 consumers' pain points | No migration guide written |
| Research → user-facing promotion | `INFRASTRUCTURE_RECOMMENDATIONS.md` distills key insights | `docs/research/storage-environment-mapping.md` still marked "not user-facing" |

---

## c) NOT STARTED ❌

| Item | Impact | Effort |
|------|--------|--------|
| Migrate SEC to `stack/sqlite` (fixes silent in-memory prod bug) | P0 — active data-loss risk | Medium (380 lines of boilerplate to remove) |
| Migrate usermgmt to `stack/sqlite` | P1 — reusable library others consume | Medium (200+ line SQLSessionStore, hand-rolled projections) |
| Migrate DiscordSync to `stack/` | P1 — 260-line custom projection runner | Medium |
| Consumer migration guide doc | P2 — lowers activation energy | Small |
| Postgres multi-DB split | P2 — parity with sqlite/turso | Medium |
| Multi-DB contract test in `contracttest` | P2 — prevents future routing regressions | Small |
| Auto-migrate by default for raw `storage.NewSQLiteEventStore` | P2 — every consumer hits this | Small |

---

## d) TOTALLY FUCKED UP 💥

1. **The multi-DB routing bug was the flagship goal feature — and it was broken since inception.** `WithEventDB("events.db")` opened a database, WAL-enabled it, schema-migrated it, then **never used it** — all stores landed in whichever DB called `openSecondaryStores` last. The existing test (`TestMultiDB_AllSeparate`) only checked `bundle.EventSink != nil` — it literally could not catch the bug. This is a testament to the danger of "asserted non-nil" tests vs behavioral tests.

2. **SEC's Dockerfile omits `-tags turso`** — so production Docker deployments silently use in-memory storage. All data is lost on restart. This is a live production bug in a consumer project, caused by the build-tag-based storage switching pattern that the stack presets were designed to eliminate.

3. **All 3 consumers reimplement the same boilerplate** that the stack layer eliminates: projection replay + live + dedup (DiscordSync: 260 lines, usermgmt: 153 lines), schema migration discovery, dialect mapping, journal type-assertion, bus wiring.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture / Type Model

1. **Postgres preset lacks multi-DB split** — sqlite and turso support `WithEventDB`/`WithQueryDB`/`WithViewDB`, but postgres does not. This is an asymmetry. Postgres could use separate schemas (`events_schema`, `audit_schema`, `views_schema`) or separate databases for the same isolation.

2. **No shared multi-DB builder** — sqlite and turso have nearly identical `openEventStores`/`openQueryStores`/`openSecondaryBackend` code (109 lines duplicated). Could extract into `stack` as a shared `SQLOptionBuilder` that both presets use.

3. **Config structs use plain strings** — `config.eventDSN string`, `config.queryDSN string` etc. No compile-time safety against swapping event and query DSNs. Could use branded types or a builder pattern that enforces ordering.

4. **No `stack.SessionStore`** — usermgmt built a 200-line `SQLSessionStore` from scratch. If sessions are a common consumer need, the stack should provide one (or document the pattern).

5. **No auto-migrate for raw `storage.NewSQLiteEventStore`** — every consumer discovers this the hard way ("upstream does not auto-migrate"). The stack presets handle it, but consumers who don't use stacks are surprised.

### Well-Established Libraries / Patterns

6. **`CatchUpSubscriber` already solves projection replay + dedup** — but all 3 consumers reimplement it. The `example/deployer-first` shows the correct pattern. We need a **migration guide** that shows consumers how to replace hand-rolled projection runners with `bundle.CatchUpSubscriber()` + `stack.Materialize`.

7. **`decider.Repository` already handles the event sourcing lifecycle** — but SEC reimplements `FoldEvents` + `persistAndPublish` manually (112 lines). The consumer should use `decider.NewRepository(store, bus, decider)` or `stack.Repository[State](bundle, decider)`.

### Testing

8. **`contracttest.RunSuite` doesn't test multi-DB** — it tests single-DB only. A `RunMultiDBSuite` that verifies routing correctness would prevent future regressions across all presets that support it.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **impact / effort ratio** (highest first).

| # | Task | Impact | Effort | Ratio |
|---|------|--------|--------|-------|
| 1 | Migrate SEC to `stack/sqlite` (fixes prod data-loss bug) | 🔴 Critical | Medium | ⭐⭐⭐⭐⭐ |
| 2 | Write "Hand-wired → Stack migration guide" doc | 🔴 High | Small | ⭐⭐⭐⭐⭐ |
| 3 | Add `contracttest.RunMultiDBSuite` (routing proof for all presets) | 🟡 Medium | Small | ⭐⭐⭐⭐⭐ |
| 4 | Document that raw `storage.NewSQLiteEventStore` needs manual migration | 🟡 Medium | Tiny | ⭐⭐⭐⭐⭐ |
| 5 | Add `WithEventDB`/`WithQueryDB`/`WithViewDB` to Postgres preset | 🟡 Medium | Medium | ⭐⭐⭐⭐ |
| 6 | Extract shared multi-DB builder into `stack/` (deduplicate sqlite + turso) | 🟡 Medium | Medium | ⭐⭐⭐⭐ |
| 7 | Migrate usermgmt to `stack/sqlite` (removes 200+ line SQLSessionStore) | 🟠 High | Medium | ⭐⭐⭐⭐ |
| 8 | Migrate DiscordSync to `stack/` (removes 260-line projection runner) | 🟠 High | Medium | ⭐⭐⭐⭐ |
| 9 | Promote `docs/research/storage-environment-mapping.md` to user-facing | 🟡 Medium | Small | ⭐⭐⭐ |
| 10 | Add multi-DB bench test (single-DB vs split performance) | 🟢 Low | Small | ⭐⭐⭐ |
| 11 | Consider `stack.SessionStore` or documented pattern | 🟡 Medium | Medium | ⭐⭐⭐ |
| 12 | Add example: multi-DB split with `example/deployer-first` variant | 🟡 Medium | Small | ⭐⭐⭐ |
| 13 | Consider branded types for DSN paths in config structs | 🟢 Low | Small | ⭐⭐ |
| 14 | Audit all doc.go files for phantom function references (lint check) | 🟡 Medium | Tiny | ⭐⭐⭐⭐ |
| 15 | Write ADR for multi-DB split design rationale | 🟡 Medium | Small | ⭐⭐⭐ |
| 16 | Add `go vet` or linter rule for broken godoc links | 🟢 Low | Small | ⭐⭐⭐ |
| 17 | Document `CatchUpSubscriber` as the canonical projection pattern | 🟠 High | Small | ⭐⭐⭐⭐ |
| 18 | Consider `stack.Materialize` support for SQL-backed views (not just KV) | 🟡 Medium | Large | ⭐⭐ |
| 19 | Add contract test for `turso.NewSync` (currently untested) | 🟡 Medium | Medium | ⭐⭐⭐ |
| 20 | Consider columnar/graph DB recommendation doc for advanced read models | 🟢 Low | Medium | ⭐⭐ |
| 21 | Add `deployer-first` example with Turso sync mode | 🟢 Low | Small | ⭐⭐ |
| 22 | Review if `stack.Bundle` needs a `SessionStore` field | 🟡 Medium | Medium | ⭐⭐ |
| 23 | Consider `go generate` for preset boilerplate (options, config, etc.) | 🟢 Low | Large | ⭐ |
| 24 | Audit consumer projects for other SDK gaps (what else do they build from scratch?) | 🟡 Medium | Medium | ⭐⭐⭐ |
| 25 | Consider a `stack.Validate()` method that checks multi-DB config coherence | 🟢 Low | Small | ⭐⭐ |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the library provide a `SessionStore` abstraction (and therefore a `stack.Bundle.SessionStore` field), or is session management fundamentally outside the CQRS/ES boundary?**

- **usermgmt** built a 200-line `SQLSessionStore` from scratch with DDL, migrations, placeholder dialect mapping, and JSON marshaling — entirely outside the SDK.
- This suggests sessions are a common enough consumer need that the library should at minimum document the pattern, or at maximum provide a `SessionSink`/`SessionSource` interface + SQL implementation.
- But sessions are arguably an infrastructure concern (like auth middleware) that doesn't belong in a CQRS library.
- **The answer depends on whether you consider session management part of the "CQRS infrastructure" boundary or purely application-layer concern.** This is a domain philosophy question I can't resolve without your input.

---

## Consumer Analysis Summary

| Consumer | Uses `stack/`? | Lines of reimplemented boilerplate | Biggest pain point |
|----------|--------------|-------------------------------------|-------------------|
| **SEC** | ❌ | 380 (turso wrapper + fold + projection) | Dockerfile omits `-tags turso` → **prod data loss** |
| **DiscordSync** | ❌ | 260 (custom projection runner) + 28-method query layer | 4 copies of projection-runner setup boilerplate |
| **usermgmt** | ❌ | 200+ (SQLSessionStore) + 153 (projection lifecycle) | Dialect mapping, schema migration discovery |

**Every consumer reimplements the same projection replay + dedup logic that `CatchUpSubscriber` already provides.** This is the single largest source of duplicated consumer-side boilerplate.

---

## Architecture Health Scorecard

| Goal Component | Score | Rationale |
|----------------|-------|-----------|
| Consumers should NOT decide infrastructure | **B+** (arch) / **F** (adoption) | Stack abstraction is excellent; 0/3 consumers use it |
| Simple API for deployers | **A** | 5 presets, one-line swap, contract-tested |
| Recommendations (which DB for which concern) | **B** | New `INFRASTRUCTURE_RECOMMENDATIONS.md`; was buried in research/ |
| Run fully with SQLite + Memory | **A** | Both fully wired, all capabilities |
| Multiple SQLite DBs (1 ES, 1 query, 1 views) | **A** (was **F**) | Fixed routing bug; tested with row-count proof |
