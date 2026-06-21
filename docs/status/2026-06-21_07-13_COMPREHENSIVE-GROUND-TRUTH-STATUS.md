# Comprehensive Status Report — 2026-06-21 07:13 CEST

> Ground-truth verified by `go build`, `go vet`, `go test`, `go test -cover`,
> and shell scripts. Not based on historical docs (which are mostly stale).

---

## Executive Summary

**The library is technically healthy but strategically lost.** The codebase
compiles cleanly, all 59 test packages pass, coverage on core modules is
excellent (86–98%), and there are zero TODO/FIXME markers in production code.
However, the project has **zero external consumers**, **668 doc files
(225k lines) vs 415 production Go files (40k lines)** — a 5.6:1 doc-to-code
ratio that signals severe process overhead. 840 commits in 21 days (40/day)
with no real-world validation.

**The code doesn't need more features. It needs a consumer.**

---

## a) FULLY DONE (verified passing)

| Area                           | Status                       | Evidence                                                                                                      |
| ------------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------------- | --- | ------------- |
| **Build**                      | ✅ Clean                     | `go build ./...` exits 0 with experimental tags                                                               |
| **Vet**                        | ✅ Clean                     | `go vet ./...` exits 0                                                                                        |
| **Tests**                      | ✅ 59/59 pass                | Zero failures across all modules + examples                                                                   |
| **Race detector**              | ✅ Clean on modified modules | SSE, stack, event pass `-race`                                                                                |
| **File-size gate**             | ✅ Zero violations           | No production file exceeds 350 lines                                                                          |
| **TODO/FIXME markers**         | ✅ Zero in production        | Only acceptable codec init panics remain                                                                      |
| **All 11 v3 breaking changes** | ✅ Complete                  | ghost bus removed, `readmodel/` deleted, `projection/` dissolved, `Event` concrete type, `Fold`→`Apply`, etc. |
| **Zero-panic migration**       | ✅ Complete                  | 26 production panics → error returns                                                                          |
| **Error taxonomy**             | ✅ Complete                  | 5-family classification (Rejection/Conflict/Transient/Infrastructure/Corruption)                              |
| **Dead build tags cleaned**    | ✅ Done                      | Only `goexperiment.arenas` + `goexperiment.jsonv2` remain                                                     |
| **Security theater removed**   | ✅ Done                      | gosec `-no-fail`, `                                                                                           |     | true` removed |
| **Ghost code eliminated**      | ✅ Done                      | `readmodel/`, `projection/`, `wasm/`, `transaction_id.go`, ghost bus — all deleted                            |
| **CI file-size gate fixed**    | ✅ Done                      | Subshell bug fixed, gate actually works                                                                       |
| **API stability golden**       | ✅ Current                   | 1,605 exports, `TestAPISurfaceCheck` passes                                                                   |

### Core Module Coverage

| Module           | Coverage | Verdict                      |
| ---------------- | -------- | ---------------------------- |
| `decider`        | 98.3%    | Excellent                    |
| `id`             | 97.6%    | Excellent                    |
| `storage/memory` | 94.1%    | Excellent                    |
| `event`          | 91.6%    | Excellent                    |
| `command`        | 90.5%    | Excellent                    |
| `query`          | 86.1%    | Good                         |
| `storage/pebble` | 82.5%    | Good                         |
| `storage/turso`  | 83.3%    | Good                         |
| `codec`          | 80.9%    | Acceptable                   |
| `storage`        | 79.3%    | Needs work (SQL error paths) |

### Features Shipped This Session

| Feature                                                              | Module           | Status                     |
| -------------------------------------------------------------------- | ---------------- | -------------------------- |
| Pebble `Checkpoint(dir)` + `Metrics()` + `Flush()` + `NewSnapshot()` | `stack/pebble`   | ✅ Done, tested            |
| `Bundle.GracefulClose(ctx)`                                          | `stack`          | ✅ Done, tested            |
| SSE `Last-Event-ID` reconnection with dedup                          | `transport/http` | ✅ Done, tested under race |
| SQLite preset split (CI compliance)                                  | `stack/sqlite`   | ✅ Done                    |

---

## b) PARTIALLY DONE

| Item                             | % Done                    | What's Left                                                                                                                                                                 | Impact                                        |
| -------------------------------- | ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| **Stack preset coverage**        | 40% avg                   | `stack/postgres` 21.2% (needs `POSTGRES_TEST_DSN`), `stack/pebble` 52.6%, `stack/sqlite` 69.9%                                                                              | Medium — presets are the consumer entry point |
| **`event/streaming_source.go`**  | 90%                       | Dead duplicate of `event/stream.go` — `EventIterator` interface lives in `streaming_source.go`, actual implementations live elsewhere. Works but is a split brain.          | Low — functional, just messy                  |
| **codec CBOR init panics**       | 95%                       | 4 panics in `codec/cbor.go` and `codec/cbor_compact.go` during `sync.Once` init. Acceptable (startup-only, unrecoverable config error) but technically violates zero-panic. | Low                                           |
| **`example/deployer-first`**     | 90%                       | Working end-to-end example but not referenced from README prominently enough.                                                                                               | Medium — consumer onboarding                  |
| **`prometheus/` module**         | 80% coverage, 0 consumers | Built, tested, but not wired into any preset or example. Island export.                                                                                                     | Low until someone needs it                    |
| **Per-module coverage CI floor** | 0%                        | No CI gate prevents coverage regression. Proposal exists but not implemented.                                                                                               | Medium                                        |

---

## c) NOT STARTED

| Item                                   | Effort    | Impact   | Notes                                                     |
| -------------------------------------- | --------- | -------- | --------------------------------------------------------- |
| **Tag v3.0.0**                         | 5 min     | HIGH     | Code is ready. Irreversible. Pointless without consumers. |
| **Build a real application**           | Days      | CRITICAL | Zero external consumers. Library is unvalidated.          |
| **Collapse 38→6 modules**              | Days      | HIGH     | 119 replace directives is the #1 consumer friction        |
| **Real-Postgres CI integration**       | 1 hour    | Medium   | `stack/postgres` tests skip without `POSTGRES_TEST_DSN`   |
| **gRPC/NATS/Redis transport adapters** | Days each | LOW      | YAGNI — no consumer signal                                |
| **jsonv2 codec**                       | Blocked   | None     | Blocked on Go stdlib stabilization                        |
| **Arena allocation**                   | Blocked   | None     | Blocked on Go stdlib stabilization                        |
| **Distributed projection runner**      | Deferred  | None     | Single-machine is sufficient; explicitly deferred         |
| **Secondary indexes for kv.Store**     | 1 day     | Low      | No consumer needs ranged scans yet                        |
| **Event stream compaction**            | 1 week    | Low      | No design, no consumer need                               |

---

## d) TOTALLY FUCKED UP

| Issue                                           | Severity          | Root Cause                                                                                                                                                                                       |
| ----------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **668 doc files / 225k lines of docs**          | 🔴 Critical waste | Every session generates 5-10 status/planning/review HTMLs. 110 status files, 88 planning files, 50 HTML reports. Nobody reads them. This is the biggest liability in the project — not the code. |
| **5.6:1 doc-to-code ratio**                     | 🔴 Critical       | 225,884 lines of docs vs 40,173 lines of production Go. The project produces more documentation than code by a 5.6x margin.                                                                      |
| **840 commits in 21 days, 0 consumers**         | 🔴 Critical       | Prototype velocity with no validation loop. The library has never been imported by another project.                                                                                              |
| **119 replace directives across 38 modules**    | 🟠 High           | "Lightweight CQRS library" that requires 119 lines of `replace` directives to work locally. This is the opposite of lightweight for a consumer.                                                  |
| **`event/streaming_source.go` split brain**     | 🟡 Medium         | `EventIterator` interface defined here, but `stream.go` has the actual streaming implementations. Historical artifact from a design pivot that was never cleaned up.                             |
| **1,605 API exports, never shrinking**          | 🟡 Medium         | Every session adds exports. Zero `Deprecated` or `Removed` in changelog. The API surface only grows.                                                                                             |
| **2 self-published dependencies in 38 modules** | 🟡 Medium         | `go-error-family` (36 modules) and `go-branded-id` (28 modules). Bus factor = 1. If either repo disappears, the entire library breaks.                                                           |

---

## e) WHAT WE SHOULD IMPROVE

### Architectural

1. **Collapse modules** — `event`, `command`, `decider`, `id`, `codec`, `dispatcher` should be ONE module (`cqrs-core`). `storage/*` should be ONE module. `stack/*` should be ONE module. Target: ~5 modules, ~5 replace directives.

2. **Delete speculative scope** — `catalog/docserver/` (720 LOC, zero consumers), `turso/indexing/` (1,400 LOC, works on any SQLite not just Turso), `prometheus/` (141 LOC, zero consumers). Move to `experimental/` branch or delete.

3. **Inline `go-error-family`** — 3 types, 5 constructors. Inlining eliminates a bus-factor-1 dependency and reduces the replace directive count by ~36.

4. **Clean up `streaming_source.go`** — Merge `EventIterator` into `stream.go` or delete one of them. This is a 10-minute fix that's been deferred for days.

### Process

5. **Stop generating status/planning/review docs** — Adopt a rule: no new doc file unless it's an ADR, a README, or a migration guide. Status reports go in commit messages.

6. **Quarterly doc prune** — Archive or delete docs older than 30 days that aren't ADRs or READMEs. 583 `.md` files + 50 HTML reports is unacceptable.

7. **Tag releases** — The repo has been "v3-ready" for days with no tag. Either tag it or stop claiming readiness.

### Consumer Trust

8. **Build a real application** — A todo backend, a banking example, anything that exercises the full stack. This will surface more real issues in 2 hours than 50 self-reviews.

9. **Dogfood the presets** — `stack/sqlite.New("app.db")` is the consumer entry point. Use it to build something real. If it doesn't work smoothly, fix it.

10. **Reduce API surface** — 1,605 exports is too many for a "lightweight" library. Mark internal types as unexported. Consolidate overlapping types.

---

## f) Top 25 Things to Get Done Next

### Tier 1: Validate (do FIRST, blocks everything)

| #   | Task                                            | Impact   | Effort |
| --- | ----------------------------------------------- | -------- | ------ |
| 1   | **Build a real application using the library**  | CRITICAL | Days   |
| 2   | **Dogfood `stack/sqlite.New()` end-to-end**     | CRITICAL | Hours  |
| 3   | **Publish to a consumer project, get feedback** | CRITICAL | Days   |

### Tier 2: Simplify (high value, reduces friction)

| #   | Task                                                  | Impact | Effort   |
| --- | ----------------------------------------------------- | ------ | -------- |
| 4   | **Collapse 38→5-6 modules**                           | HIGH   | Days     |
| 5   | **Delete `catalog/docserver/`**                       | Medium | 5 min    |
| 6   | **Move `turso/indexing/` to `storage/sql/indexing/`** | Medium | 1 hour   |
| 7   | **Delete or wire `prometheus/`**                      | Medium | Decision |
| 8   | **Inline `go-error-family` (3 types)**                | Medium | 2 hours  |
| 9   | **Clean up `event/streaming_source.go` split brain**  | Low    | 10 min   |
| 10  | **Archive 500+ historical doc files**                 | Medium | 1 hour   |

### Tier 3: Ship (makes it real)

| #   | Task                                               | Impact | Effort  |
| --- | -------------------------------------------------- | ------ | ------- |
| 11  | **Tag v3.0.0** (after validation)                  | HIGH   | 5 min   |
| 12  | **Write `V3_MIGRATION.md` (fix broken example)**   | Medium | 30 min  |
| 13  | **Add godoc `Example_*` functions for presets**    | Medium | 1 hour  |
| 14  | **Reduce API surface (mark internals unexported)** | Medium | Ongoing |

### Tier 4: Harden (after consumers exist)

| #   | Task                                                | Impact | Effort  |
| --- | --------------------------------------------------- | ------ | ------- |
| 15  | **Per-module coverage CI floor**                    | Medium | 1 hour  |
| 16  | **Real-Postgres CI integration**                    | Medium | 1 hour  |
| 17  | **Property-based tests for Version arithmetic**     | Low    | 30 min  |
| 18  | **Add race tests for PgxListener concurrent Close** | Medium | 45 min  |
| 19  | **Full `art-dupl` dedup pass on ~72 clone groups**  | Low    | 3 hours |

### Tier 5: Polish (nice to have)

| #   | Task                                                     | Impact | Effort  |
| --- | -------------------------------------------------------- | ------ | ------- |
| 20  | **`fanOut[T]` generic pattern for SSE**                  | Medium | 30 min  |
| 21  | **Typed `SecurityEnvelope` metadata field**              | Low    | 30 min  |
| 22  | **Property-based test for `command.Metadata` isolation** | Low    | 1 hour  |
| 23  | **Add `CONTRIBUTING.md`**                                | Low    | 1 hour  |
| 24  | **Benchmark hot paths (post concrete-type baseline)**    | Low    | 2 hours |
| 25  | **Version drift detection in CI**                        | Low    | 30 min  |

---

## g) The #1 Question I Cannot Answer

> **What is the actual consumer for this library?**
>
> Is it:
>
> - **A) A learning/showcase project?** (Then the 668 docs make sense as a
>   portfolio piece, and we should stop calling it "production-ready".)
> - **B) A library for Lars's own future projects?** (Then we need to build
>   one of those projects NOW to validate the API.)
> - **C) A library for external consumers?** (Then we need a marketing plan,
>   real documentation site, and the 38-module structure is a fatal flaw.)
>
> Without knowing the target audience, every decision about module count, API
> surface, documentation depth, and feature prioritization is a guess.
> **This question has been open since session 1 and remains unanswered.**

---

## Metrics Snapshot

| Metric             | Value      | Trend                         |
| ------------------ | ---------- | ----------------------------- |
| Build              | ✅ Clean   | Stable                        |
| Test packages      | 59/59 pass | Stable                        |
| Core coverage      | 86–98%     | Stable                        |
| Stack coverage     | 21–75%     | Needs work                    |
| Modules            | 38         | Too many                      |
| Replace directives | 119        | Too many                      |
| API exports        | 1,605      | Growing                       |
| Production LOC     | 40,173     | Stable                        |
| Doc LOC            | 225,884    | Growing (bad)                 |
| Doc-to-code ratio  | 5.6:1      | Getting worse                 |
| External consumers | 0          | **Unchanged since day 1**     |
| Commits in 21 days | 840        | High velocity, low validation |
| Contributors       | 1          | Bus factor = 1                |
| TODO/FIXME in prod | 0          | Clean                         |
| Files >350 lines   | 0          | Clean                         |

---

_Generated 2026-06-21 07:13 CEST. Ground-truth verified by build+test+vet. Working tree clean on `master` at `e968c799`._
