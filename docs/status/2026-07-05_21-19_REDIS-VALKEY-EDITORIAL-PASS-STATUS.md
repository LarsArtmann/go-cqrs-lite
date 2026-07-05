# Status Report — Redis → ValKey Editorial Pass

**Date:** 2026-07-05 21:19
**Session Focus:** Annotate every Redis mention in the repo with the author's editorial stance (not a fan; ValKey recommended) and give ValKey equal billing
**Status:** PARTIALLY DONE — 8/49 files committed, 41 files reverted/lost

---

## Executive Summary

The user asked to find every mention of Redis and make the editorial stance clear: the author is not a fan, may support it one day, but would still recommend ValKey instead.

A comprehensive 85-file edit pass was executed across all living docs, source code, research papers, ADRs, status/planning docs, and archived material. All builds passed. Doc-check validated 790 references.

**However:** 77 of those 85 file edits were silently reverted between conversation turns. The working tree is clean at HEAD `33f94d74`. Only 8 files survived (committed as `e022a767` in a prior turn). **The bulk of the work needs to be redone.**

---

## a) FULLY DONE ✅

These 8 files are committed (`e022a767`) and carry the full editorial stance:

| File                                                                | What was added                                                     |
| ------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `ROADMAP.md`                                                        | Blockquote editorial note after transport/redis section            |
| `TODO_LIST.md`                                                      | Inline note on NATS/Redis adapter item                             |
| `docs/adr/0025-transport-adapter-strategy.md`                       | Blockquote editorial note in Status section                        |
| `docs/design/transport-redis.md`                                    | Top-of-file blockquote + ValKey mentions in Problem + Dependencies |
| `docs/research/2026-06-23_DEPLOYER_FIRST_ARCHITECTURE_AUDIT.md`     | Comment annotation on Redis checkpoint line                        |
| `docs/status/2026-06-28_00-17_RELATIONAL-GRAPH-PROJECTION-TIERS.md` | RedisGraph line annotated                                          |
| `docs/status/2026-06-28_05-06_POST-COMMAND-BRIDGE-STATUS.md`        | Executive summary annotated                                        |
| `watermill/doc.go`                                                  | Redis Streams code example annotated with ValKey note              |

**Coverage:** 8 of 49 living files with Redis mentions = **16%**.

---

## b) PARTIALLY DONE 🔨

### Source code (5 files — ALL REVERTED)

All five `.go` source files were edited with ValKey annotations and built clean. All edits are gone:

| File                    | Lines edited                   | What was lost                                                |
| ----------------------- | ------------------------------ | ------------------------------------------------------------ |
| `graph/graph.go`        | 2 RedisGraph mentions          | "author: not a fan of Redis — ValKey preferred" inline notes |
| `idempotency/store.go`  | 1 Redis SET NX mention         | "Redis or ValKey store ... (author prefers ValKey)"          |
| `kv/kv.go`              | 1 Redis SET NX mention         | "Redis/ValKey SET NX" + infrastructure doc cross-ref         |
| `transport/http/doc.go` | 1 future transports mention    | "Redis/ValKey" + author stance note                          |
| `watermill/doc.go`      | 2 more mentions (lines 66, 86) | Broker backends heading + package doc annotation             |

### Living documentation (5 files — ALL REVERTED)

| File                                     | What was lost                                                |
| ---------------------------------------- | ------------------------------------------------------------ |
| `AGENTS.md`                              | 2 code-comment annotations (RedisGraph + CommandBus backend) |
| `FEATURES.md`                            | Transport adapters row updated with "author prefers ValKey"  |
| `docs/DOMAIN_LANGUAGE.md`                | 2 table rows (EventBus + Message Broker) with ValKey         |
| `CHANGELOG.md`                           | 3 entries updated with Redis/ValKey                          |
| `docs/INFRASTRUCTURE_RECOMMENDATIONS.md` | 2 edits — "Never use" section + KV store recommendation      |

### Research docs (7 files — ALL REVERTED)

| File                                                                   | What was lost                                                              |
| ---------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `docs/research/database-architecture-taxonomy.md`                      | 4 table cells: Redis → Redis/ValKey                                        |
| `docs/research/kv-store-abstraction-research.md`                       | 3 backend rows: Redis → Redis/ValKey                                       |
| `docs/research/storage-environment-mapping.md`                         | 8 mentions across 6 sections — full ValKey treatment + section 6.4 rewrite |
| `docs/research/storage-first-principles-analysis.md`                   | 2 comparison table rows                                                    |
| `docs/research/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md`          | EventHorizon backend row                                                   |
| `docs/research/2026-05-28_SINK_SOURCE_SPLIT_AND_GENERIC_BOUNDARIES.md` | Pub/sub direction mention                                                  |
| `docs/research/scheduled-event.md`                                     | Sorted sets backend option                                                 |

### ADRs + design docs (3 files — ALL REVERTED)

| File                                           | What was lost                                 |
| ---------------------------------------------- | --------------------------------------------- |
| `docs/adr/0018-distributed-checkpointing.md`   | 3 Redis mentions → Redis/ValKey + author note |
| `docs/adr/0038-graph-projection-tier.md`       | RedisGraph line annotated                     |
| `docs/design/distributed-projection-runner.md` | Work queue mention annotated                  |

### READMEs (2 files — ALL REVERTED)

| File                    | What was lost                                        |
| ----------------------- | ---------------------------------------------------- |
| `graph/README.md`       | 2 mentions (MERGE backends + driver table) annotated |
| `idempotency/README.md` | Redis backend bullet → Redis/ValKey                  |

### Skill references (1 file — REVERTED)

| File                                                 | What was lost                           |
| ---------------------------------------------------- | --------------------------------------- |
| `.agents/skills/go-cqrs-lite/references/advanced.md` | RedisGraph in graph projections section |

---

## c) NOT STARTED ⬜

### Planning + status docs (22 files — batch replacement never landed)

These 22 historical docs were batch-edited (`Redis` → `Redis/ValKey`) in the reverted pass. None of those edits survived:

**Planning docs (7):**

- `docs/planning/2026-06-11_22-00_COMPREHENSIVE-EXECUTION-PLAN-v3.md`
- `docs/planning/2026-06-16_22-30_COMPREHENSIVE_REMAINING_WORK.md`
- `docs/planning/2026-06-17_02-25_COMPREHENSIVE_EXECUTION_PLAN.md`
- `docs/planning/2026-06-17_19-00_MASTER_TODO_PLAN.md`
- `docs/planning/2026-06-19_18-30_postgres-bus-wiring-plan.md`
- `docs/planning/2026-06-19_21-00_postgres-bus-production-hardening-plan.md`
- `docs/planning/2026-06-29_00-54_LIBRARY-PARETO-EXECUTION-PLAN.md`

**Status docs (15):**

- `docs/status/2026-06-22_12-40_V3-RELEASE-TAGGED.md` (+ 14 more)

**Archive docs (~20 files):**

- All `.md` files in `docs/planning/archive/` and `docs/status/archive/` with Redis mentions

---

## d) TOTALLY FUCKED UP 💥

### The Great Reversion of 77 Files

**What happened:** I executed a comprehensive 85-file editorial pass. Every edit tool call returned success. I verified builds passed on all 5 Go modules. Doc-check validated 790 references. I confirmed via `rg` that all mentions were annotated.

**Then between conversation turns**, the working tree was reset to HEAD. 77 of 85 file edits vanished. The git log shows 3 commits made during this session window, but none contain the bulk work:

```
33f94d74 fix: correct SKILL.md doc-check refs and remove 3 dead code symbols  ← not mine
57f7bc67 docs: add truth reconciliation plan for docs-vs-code drift            ← not mine
e022a767 docs: add ValKey recommendation alongside Redis references            ← only 8 files
```

**Root cause hypothesis:** An external process (auto-commit, git hook, or session boundary) reset uncommitted working-tree changes. The initial 8-file pass survived because it was committed; the remaining 77 files were uncommitted and got wiped.

**Impact:** ~3 hours of work lost. The user's request is 16% complete.

**Lesson:** Commit incrementally during large edit passes. Don't accumulate 85 files of uncommitted changes.

### Pre-existing uncommitted go.mod/go.sum drift

During the session, `git status` showed ~23 `go.mod`/`go.sum` files modified with:

- `go 1.26.3` → `go 1.26.4` (toolchain bump)
- `go-error-family v0.5.1` → `v0.6.1` (dependency upgrade)

These were NOT made by this session and NOT committed. They appeared and disappeared between turns. This is pre-existing dependency drift that should be investigated separately.

---

## e) WHAT WE SHOULD IMPROVE 📈

1. **Commit early, commit often** — The single biggest improvement. If I had committed after every batch of 10-15 files, at most one batch would have been lost. 77 files of uncommitted edits is irresponsible.

2. **Batch the replacable edits first** — The 22 planning/status docs and ~20 archive docs are mechanical `Redis → Redis/ValKey` replacements. These should be a single `perl -pi -e` + commit, done in 30 seconds, not hand-edited file by file.

3. **Separate "editorial note" edits from "equal billing" edits** — The prominent blockquotes (ROADMAP, ADR-0025, design docs, INFRASTRUCTURE_RECOMMENDATIONS) need careful human-written prose. The `Redis → Redis/ValKey` replacements are mechanical. These are two different commits with two different review levels.

4. **Verify persisted state between turns** — I should have run `git diff --stat` at the START of the status-report turn to detect the reversion before writing a report based on stale assumptions.

5. **The `docs/planning/2026-07-05_21-11_truth-reconciliation.md` plan** (from a sibling session) identifies 10 tasks / 48 subtasks for docs-vs-code drift. The Redis/ValKey pass intersects with this — several files that need ValKey annotations also need truth reconciliation.

---

## f) Up to 25 Things We Should Get Done Next 🎯

### Tier 1: REDO THE LOST WORK (commit after each batch!)

| #   | Task                                                                     | Impact | Effort | Approach                                             |
| --- | ------------------------------------------------------------------------ | ------ | ------ | ---------------------------------------------------- |
| 1   | **Batch-replace `Redis → Redis/ValKey` in 22 planning/status .md files** | H      | XS     | `perl -pi -e` one-liner + commit immediately         |
| 2   | **Batch-replace in ~20 archive .md files**                               | M      | XS     | Same approach, separate commit                       |
| 3   | **Re-edit 5 source `.go` files** (graph.go, kv.go, store.go, doc.go × 2) | H      | S      | Hand-edit + `go build` + commit                      |
| 4   | **Re-edit AGENTS.md** (2 code-comment annotations)                       | H      | XS     | Hand-edit + commit                                   |
| 5   | **Re-edit FEATURES.md** (transport adapters row)                         | M      | XS     | Hand-edit + commit                                   |
| 6   | **Re-edit DOMAIN_LANGUAGE.md** (2 table rows)                            | M      | XS     | Hand-edit + commit                                   |
| 7   | **Re-edit CHANGELOG.md** (3 entries)                                     | L      | XS     | Hand-edit + commit                                   |
| 8   | **Re-edit INFRASTRUCTURE_RECOMMENDATIONS.md** (2 sections)               | H      | S      | Hand-edit + commit — this is the deployer-facing doc |
| 9   | **Re-edit 7 research docs**                                              | M      | S      | Batch hand-edit + commit                             |
| 10  | **Re-edit 3 ADRs + design docs**                                         | M      | S      | Hand-edit + commit                                   |
| 11  | **Re-edit 2 READMEs** (graph/, idempotency/)                             | L      | XS     | Hand-edit + commit                                   |
| 12  | **Re-edit skill references/advanced.md**                                 | L      | XS     | Hand-edit + run doc-check + commit                   |

### Tier 2: Related improvements noticed during the pass

| #   | Task                                                                               | Impact | Effort | Notes                                                                                              |
| --- | ---------------------------------------------------------------------------------- | ------ | ------ | -------------------------------------------------------------------------------------------------- |
| 13  | **Investigate go.mod/go.sum drift** (go 1.26.3→1.26.4, go-error-family bump)       | M      | S      | 23 modules affected; appeared/disappeared between turns                                            |
| 14  | **Add ValKey to SKILL.md module decision matrix**                                  | M      | S      | The AI consumer guide doesn't mention ValKey at all                                                |
| 15  | **Add ValKey to the broker plugin recipe** in `watermill/doc.go` section heading   | L      | XS     | Currently says "Broker Backends (NATS, Redis, Kafka)"                                              |
| 16  | **Consider a `docs/EDITORIAL_STANCE.md`**                                          | L      | M      | Centralize the Redis/ValKey stance so every doc can link to it instead of repeating the blockquote |
| 17  | **Add ValKey mention to `cmd/cqrs-gen` templates** if they generate doc references | L      | S      | Check if code generator outputs any Redis references                                               |

### Tier 3: From the truth-reconciliation plan (intersection)

| #   | Task                                                                   | Impact | Effort | Notes                                                              |
| --- | ---------------------------------------------------------------------- | ------ | ------ | ------------------------------------------------------------------ |
| 18  | **Truth-reconcile ROADMAP.md** (module count, transport section)       | H      | XS     | Also needs ValKey — combine both passes                            |
| 19  | **Truth-reconcile TODO_LIST.md** (stale genproto reference)            | H      | XS     | Already touched for ValKey — combine                               |
| 20  | **Truth-reconcile FEATURES.md** (transport adapters row is stale)      | M      | XS     | Already touched for ValKey — combine                               |
| 21  | **Remove dead `docs/design/transport-redis.md` or mark as superseded** | M      | XS     | The design doc describes a module that will never ship as designed |

### Tier 4: Structural improvements

| #   | Task                                                                                     | Impact | Effort | Notes                                                            |
| --- | ---------------------------------------------------------------------------------------- | ------ | ------ | ---------------------------------------------------------------- |
| 22  | **Add a CI lint that detects Redis mentions without ValKey**                             | L      | M      | Grep-based check; prevents editorial drift                       |
| 23  | **Consolidate the "not a fan of Redis" stance into a single canonical paragraph**        | L      | S      | Currently reworded ~10 different ways across docs                |
| 24  | **Evaluate whether ValKey should get its own transport adapter design doc**              | L      | M      | `docs/design/transport-valkey.md` — or just rename the Redis one |
| 25  | **Run `nix run .#lint` + `nix run .#test`** to confirm no regressions from the full pass | H      | S      | Should be done after the redo is complete                        |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**What actually reverted the 77 files?**

The working tree went from 85 modified files (verified by `git diff --stat`, builds, and `rg` checks) to clean-at-HEAD between two conversation turns. I did not run `git restore`, `git checkout`, `git reset`, or any reversion command. The git log shows two commits I did not author (`57f7bc67`, `33f94d74`) landing in the same window.

Possible explanations:

1. **An external auto-commit process** committed a subset and discarded the rest
2. **A git hook** (pre-commit or post-commit) reset uncommitted changes
3. **A session boundary** triggered a working-tree cleanup
4. **The `go.mod`/`go.sum` drift** triggered a `go mod tidy` that somehow reset the tree
5. **Another conversation/session** operated on the same repo concurrently

**Why it matters:** If this happens every session boundary, then ANY large uncommitted edit pass is at risk. The fix (committing incrementally) is a workaround, but understanding the root cause would prevent future data loss across all tasks, not just this one.

**What I need:** The git reflog for the session window, or knowledge of any auto-commit/cleanup hooks configured in the dev environment or Crush session lifecycle.
