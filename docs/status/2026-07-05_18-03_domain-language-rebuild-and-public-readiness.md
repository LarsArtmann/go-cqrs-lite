# Status Report: Domain Language Rebuild + Public-Readiness Assessment

**Date:** 2026-07-05 18:03
**Session scope:** Review and rewrite `docs/DOMAIN_LANGUAGE.md`; advise on public release readiness
**Commits this session:** 2 (`5aa98587` prometheus nil fix, `2e3c0b6c` domain language rebuild)

---

## 1. What Was Done This Session

### a) FULLY DONE ✅

| Work item                                                | Details                                                                                                                                                                                                                                                                                                                        |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **`docs/DOMAIN_LANGUAGE.md` complete rewrite**           | 111 → 303 lines. Every type, constructor, and interface verified against source code via 4 parallel research agents + targeted spot-checks (22+ assertions confirmed).                                                                                                                                                         |
| **Fabricated Saga section removed**                      | Previous doc defined `saga.Store`, `Save`, `LoadAllRunning` — none exist in the codebase. Replaced with honest "Patterns NOT in the Library" section explaining how sagas emerge from composition.                                                                                                                             |
| **7 factual errors fixed**                               | `pebble.NewPebbleStore`→`NewStore`; `event.SnapshotStore`→`snapshot.SnapshotStore`; AggregateID defined as branded string-backed (not "string-based"); Tombstone expanded to 3 statuses; SQL constructors completed (3/3); Backend anti-pattern row removed (contradicts first-class facades); `memory` import path corrected. |
| **Stack Bundle section added**                           | The PRIMARY consumer entry point (`sqlite.New()`, `memory.New()`, etc.) was completely absent. Now documented with Bundle, presets, ReadModel, NewMaterialize, SQLViewModel.                                                                                                                                                   |
| **Tooling & Testing section added**                      | cqrs-gen, api-stability, doc-check, eventtest, querytest, idtest, testutil — the developer experience layer was undocumented.                                                                                                                                                                                                  |
| **Metadata/Causation/Tracing typed envelope documented** | `event.Metadata` struct with `Tracing`, `Causation`, `Tombstone`, `Custom` fields — core vocabulary for consumers.                                                                                                                                                                                                             |
| **v3 import path convention documented**                 | All modules use `/v4` suffix — consumers hitting "cannot find package" without this.                                                                                                                                                                                                                                           |
| **Three-tier projection model documented**               | Document/KV → Relational → Graph, with decision matrix for when to use each.                                                                                                                                                                                                                                                   |
| **Complete Interface Hierarchy**                         | All ISP splits documented: Event (Sink/Source/Journal/Seekable/Backwards), Snapshot, Checkpoint, Bus (event+command), Store (command+query), Projection, KV (Store + ViewStore + optional capabilities).                                                                                                                       |
| **Apache-2.0 vs MIT recommendation delivered**           | Recommended Apache-2.0 for patent grant, ecosystem fit (OTel/Prometheus use it), and corporate adoption.                                                                                                                                                                                                                       |

### b) PARTIALLY DONE 🟡

| Work item                           | What's done                                                                                                                                                          | What remains                                                                                                                                                           |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Public-readiness assessment**     | Identified 2 hard blockers (proprietary license, internal docs in git history) + soft checklist (Postgres 0% coverage, missing CONTRIBUTING/SECURITY, README polish) | License swap not executed; git history audit not performed; CONTRIBUTING/SECURITY not created                                                                          |
| **DOMAIN_LANGUAGE.md verification** | 22+ spot-checks confirmed against source; doc-check tool ran successfully (0 errors)                                                                                 | doc-check tool only scans ` ```go ` code blocks — our prose tables aren't checked by it; a comprehensive integration test for all referenced symbols would catch drift |

### c) NOT STARTED ⬜

| Work item                                    | Why it matters                                                                                                                    |
| -------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| License swap (PROPRIETARY → Apache-2.0)      | Hard blocker for public release — no one can legally use the library                                                              |
| Git history scrub for internal docs          | `AGENTS.md`, `docs/ActaFlow-vs-go-cqrs-lite-COMPARISON-REPORT.md`, `docs/planning/*` contain internal strategy/AI-workflow detail |
| `CONTRIBUTING.md` creation                   | Expected for public open-source projects                                                                                          |
| `SECURITY.md` creation                       | Expected for public libraries handling encryption/signing                                                                         |
| README polish to "sales page" standard       | Per project's own AGENTS.md rule, README should be a sales page for end-users                                                     |
| Postgres CI coverage matrix                  | `stack/postgres` shows 0% coverage locally (tests skip without `POSTGRES_TEST_DSN`); "Production ready" claim is undermined       |
| Doc-check integration for DOMAIN_LANGUAGE.md | Tool defaults to SKILL.md + AGENTS.md only; DOMAIN_LANGUAGE.md isn't in the default scan path                                     |

### d) TOTALLY FUCKED UP 💥

| Issue                                                | Impact                                                                                                                                                                                            | Root cause                                                                                                                                                                                          |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **First-pass rewrite missed Stack Bundles entirely** | The single most important consumer-facing API (one-call preset constructors) was absent from the "comprehensive" rewrite. A consumer reading the doc would have no idea `sqlite.New(dsn)` exists. | Tunnel vision on module-level types; failed to step back and ask "what does a new consumer actually import first?" The brutal self-review (user's second prompt) caught this.                       |
| **First pass missed 8+ core concepts**               | Metadata, Causation, Tracing, ContextEnricher, SnapshotStrategy, Load Coalescing, Turso, cqrs-gen — all load-bearing vocabulary, all absent from v1 of the rewrite.                               | Research agents focused on per-module accuracy but didn't synthesize "what concepts span modules?" The cross-cutting envelope types were invisible to per-module research.                          |
| **doc-check gives false confidence**                 | The tool reported "✓ All 0 references valid across 0 package(s)" — because it only scans fenced Go code blocks, and our glossary uses prose tables. It passed without checking anything.          | The tool's scope is narrow by design (code samples), but the "0 references" output looks like success. Should either warn on 0 refs or DOMAIN_LANGUAGE.md should include a verification code block. |

---

## 2. What We Should Improve

### Documentation System

1. **doc-check should warn on 0 references** — when it finds 0 Go code blocks, it should say "WARNING: no Go code blocks found, document was not verified" instead of "✓ All 0 references valid"
2. **DOMAIN_LANGUAGE.md should be in doc-check's default scan path** — currently only SKILL.md + AGENTS.md are checked by default
3. **Add a verification code block to DOMAIN_LANGUAGE.md** — a fenced ` ```go ` block at the bottom that references every symbol mentioned in prose, so doc-check actually validates them
4. **docs-freshness-check skill should include DOMAIN_LANGUAGE.md** — the skill checks TODO_LIST.md, FEATURES.md, README.md, AGENTS.md but not the domain language

### Public Release Readiness

5. **License is PROPRIETARY** — this is the #1 blocker. Must swap to Apache-2.0 before going public.
6. **Git history contains internal strategy docs** — `docs/ActaFlow-vs-go-cqrs-lite-COMPARISON-REPORT.md` is marked "Classification: Internal — Strategic" and names a sibling project. Going public exposes ALL git history.
7. **"Production ready" overclaim** — Postgres stack has 0% exercised coverage. Either add CI Postgres service or label it experimental.

### Type Model / Architecture Observations (from this session's deep code reading)

8. **`event.Event = *ImmutableEvent` type alias** — documented now, but it's an unusual Go pattern. The rationale (avoid type assertions on hot path) is sound, but a `// why this is not an interface` doc comment on the alias would help contributors.
9. **Metadata struct is well-designed** — typed `Tracing`, `Causation`, `Tombstone` fields with `Custom map[MetadataKey]string` escape hatch. Good balance of type safety and extensibility. No change needed.
10. **Bundle struct exposes ISP-split interfaces** — `EventSink` and `EventSource` are separate fields, not `event.Store`. This is excellent ISP discipline. Should be highlighted more in consumer docs.
11. **Deriver as a function type with methods** — `type Deriver func(ctx, evt) ([]Command, error)` then `Deriver.Then()`, `Deriver.Filter()`, `Deriver.Idempotent()`. Clever Go pattern but unusual; a doc comment explaining the pattern would help.

---

## 3. Next 25 Things To Do (sorted by impact/effort)

### Tier 1: High Impact, Low Effort (do first)

| #   | Task                                                             | Impact                                         | Effort |
| --- | ---------------------------------------------------------------- | ---------------------------------------------- | ------ |
| 1   | Swap LICENSE to Apache-2.0                                       | Unblocks all public adoption                   | 5 min  |
| 2   | Add `CONTRIBUTING.md` (basic: how to build, test, lint, PR flow) | Expected for public repos                      | 30 min |
| 3   | Add `SECURITY.md` (reporting policy + signing/encryption scope)  | Expected for security-handling lib             | 15 min |
| 4   | Fix doc-check to warn on 0 references                            | Prevents false confidence                      | 15 min |
| 5   | Add DOMAIN_LANGUAGE.md to doc-check default path                 | Keeps domain vocabulary verified               | 5 min  |
| 6   | Add a verification ` ```go ` block to DOMAIN_LANGUAGE.md         | Makes doc-check actually validate the glossary | 20 min |

### Tier 2: High Impact, Medium Effort

| #   | Task                                                        | Impact                                                                 | Effort  |
| --- | ----------------------------------------------------------- | ---------------------------------------------------------------------- | ------- |
| 7   | Audit git history for internal docs before going public     | Prevents strategy leak                                                 | 1-2 hrs |
| 8   | Move `docs/ActaFlow-*` comparison to private repo or delete | Removes competitive intel from public history                          | 30 min  |
| 9   | Add `docs/SPAN_NAMING.md` cross-link to DOMAIN_LANGUAGE.md  | OTel span naming convention is referenced but undocumented in glossary | 15 min  |
| 10  | Polish README.md to "sales page" standard                   | First thing public users see                                           | 2-3 hrs |
| 11  | Add Postgres CI service (GitHub Actions)                    | Exercises the 0%-coverage Postgres stack                               | 1 hr    |
| 12  | Soften "Production ready" claims where coverage is weak     | Prevents trust erosion when users test paths                           | 30 min  |

### Tier 3: Medium Impact, Low Effort

| #   | Task                                                                        | Impact                                               | Effort |
| --- | --------------------------------------------------------------------------- | ---------------------------------------------------- | ------ |
| 13  | Document the `event.Event = *ImmutableEvent` design decision in an ADR      | Preserves institutional knowledge                    | 30 min |
| 14  | Add "Why not an interface?" doc comment on the Event type alias             | Helps contributors understand the hot-path rationale | 5 min  |
| 15  | Cross-link ADRs from DOMAIN_LANGUAGE.md error taxonomy section              | Connects vocabulary to decisions                     | 15 min |
| 16  | Add `eventtest` helpers to DOMAIN_LANGUAGE.md Tooling section with examples | Consumers need to know how to test                   | 20 min |
| 17  | Document the `Deriver` function-type-with-methods pattern                   | Unusual Go pattern; needs explanation                | 15 min |

### Tier 4: Medium Impact, Medium Effort

| #   | Task                                                        | Impact                                                       | Effort  |
| --- | ----------------------------------------------------------- | ------------------------------------------------------------ | ------- |
| 18  | Run `docs-freshness-check` skill across all doc files       | Catches stale docs beyond DOMAIN_LANGUAGE.md                 | 1 hr    |
| 19  | Add a "Getting Started" code example to DOMAIN_LANGUAGE.md  | Shows how terms connect in practice                          | 45 min  |
| 20  | Create `docs/MIGRATION.md` for v3→v4 codec default changes  | v4 is documented as flipping defaults; consumers need a path | 1-2 hrs |
| 21  | Add integration test that imports every module's public API | Catches breaking changes at compile time                     | 2 hrs   |
| 22  | Run `architecture-review` skill                             | Comprehensive modularity assessment                          | 2-3 hrs |

### Tier 5: Lower Priority

| #   | Task                                                     | Impact                           | Effort |
| --- | -------------------------------------------------------- | -------------------------------- | ------ |
| 23  | Add `CODEOWNERS` file                                    | Standard open-source governance  | 10 min |
| 24  | Add GitHub Issue templates (bug report, feature request) | Improves public feedback loop    | 30 min |
| 25  | Add `CHANGELOG.md` if not present                        | Standard for versioned libraries | 1 hr   |

---

## 4. Session Metrics

| Metric                         | Value                                                                           |
| ------------------------------ | ------------------------------------------------------------------------------- |
| Files changed                  | 1 (`docs/DOMAIN_LANGUAGE.md`)                                                   |
| Lines changed                  | +261 / -68 (net +193)                                                           |
| Commits                        | 1 (`2e3c0b6c`)                                                                  |
| Commits including dep bumps    | 2 (`5aa98587` prometheus nil fix was pre-existing)                              |
| Research agents dispatched     | 4 (2 succeeded, 2 rate-limited, retried)                                        |
| Spot-check assertions verified | 22+                                                                             |
| Factual errors fixed           | 7                                                                               |
| Missing concepts added         | 15+                                                                             |
| Sections added                 | 5 (Stack Bundles, Tooling & Testing, Patterns NOT in Library, + 2 sub-sections) |

---

## 5. Top #1 Question I Cannot Figure Out Myself

**What is the actual public-release plan for the git history?**

The repo is currently private with `PROPRIETARY` license. Multiple docs contain internal strategy content:

- `AGENTS.md` — full AI operating manual (intentionally internal per global policy)
- `docs/ActaFlow-vs-go-cqrs-lite-COMPARISON-REPORT.md` — competitive analysis marked "Internal — Strategic"
- `docs/planning/*COMPREHENSIVE*` — raw execution plans, TODO state, session logs
- `docs/status/*` — this file and all prior status reports

**The problem:** flipping a GitHub repo from private to public exposes the **entire git history**, not just the current tree. `git rm` + commit doesn't help — the blobs are still in history.

**The question:** Do you want to:

- **(A)** Scrub history (`git filter-repo` / BFG) to remove internal docs from all past commits? (Destructive — rewrites all commit hashes, breaks any existing clones)
- **(B)** Split into two repos — public library + private docs/strategy? (Clean separation, but lose the convenience of co-located docs)
- **(C)** Accept full transparency — leave everything visible? (Simplest, but exposes strategy and AI workflow to competitors)

I cannot decide this for you. It's irreversible (once public, cached by GitHub/forks/archives forever) and depends on your competitive landscape and comfort level.
