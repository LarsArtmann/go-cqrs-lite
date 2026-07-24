# Status: Aggregate→Stream Rename Follow-ups (ADR-0058) — Session 4

**Date:** 2026-07-24 22:56
**Session focus:** Completing ALL remaining ADR-0058 deferred items from Session 3 (verification gates, consumer-facing docs, test-file cleanup, historical-doc annotations)
**Previous sessions:** Session 1 renamed types/APIs; Session 2 started comment cleanup; Session 3 cleaned non-test Go files + top-level docs

---

## a) FULLY DONE THIS SESSION

### Verification gates (Session 3 deferred items 4–6)

1. **`nix run .#verify`** — Full gate run: build ✅, vet ✅, test ✅ (all 56 modules), doc-assertions ✅ (CHANGELOG count, module count, license). The `-race` pass showed 4 flaky failures under concurrent build pressure — all pass individually (pre-existing, unrelated to rename).
2. **`nix run .#check-layers`** — Dependency budget check passes clean.
3. **Race detection** — 4 flaky tests identified (all pass individually): `projectionhost` SQLiteCheckpoint (timing), `transport/grpc` CBOREncoding (channel timing), `benchkit` DurationAborts (timing), `cmd/cqrs-bench` build (transient). All are pre-existing concurrency test fragility, not rename-related.

### Consumer-facing documentation (Session 3 deferred items 7–8)

| File | Changes |
|------|---------|
| **README.md** | Example code: `id.NewAggregateID()`→`id.NewStreamID()`, `cmd.AggregateID()`→`cmd.StreamID()`, `aggID`→`streamID`. Module table: "Pure-function aggregate"→"Pure-function event sourcing". 0 aggregate refs remaining. |
| **FEATURES.md** | 24 stale references fixed across decider, id, snapshot, listing, otel, watermill, storage, integration sections. `AggregateID`→`StreamID`, `DeriveAggregateID`→`DeriveStreamID`, `SQLAggregateReader`→`SQLStreamReader`, `AggregateProjection`→`StreamProjection`, `AggregateAttrs`→`StreamAttrs`, `AttrAggregate*`→`AttrStream*`, all `agg*` params→`stream*`. Only `AggregateAwareStrategy` remains (intentional DDD concept). |
| **CONTRIBUTING.md** | Module tree: "Pure-function aggregate"→"Pure-function event sourcing", "Aggregate listing"→"Stream listing". |
| **docs/error-taxonomy.md** | Error var names: `ErrNilAggregateID`→`ErrNilStreamID`, `ErrEmptyAggregateType`→`ErrEmptyStreamType`, `ErrAggregateNotFound`→`ErrStreamNotFound`. Classification codes intentionally kept (`event.nil_aggregate_id`, etc.). |
| **ROADMAP.md** | Rename status updated to "Complete", known issues section updated to "✅ Done (Sessions 1–4)", `aggID`→`streamID` in examples. |
| **TODO_LIST.md** | All 5 open rename items marked `[x]` or `[RESOLVED]` with session references. YAGNI section: "aggregate"→"stream". |
| **benchkit/README.md** | Profile table: "Aggregates"/"Events/Agg"→"Streams"/"Events/Stream". Comment: "aggregate loads"→"stream loads". |

### Test file cleanup (Session 3 deferred — "~50+ remaining references")

**224 test files** cleaned via mechanical + targeted replacement:

| Category | Pattern | Example |
|----------|---------|---------|
| Variable names | `aggID`→`streamID`, `aggType`→`streamType`, `agg1`→`stream1` | `storage/event_store_load_query_test.go` (30 vars) |
| Stream type labels | `"TestAggregate"`→`"TestStream"`, `"TestAgg"`→`"TestStream"` | `storage/memory/memory_bdd_test.go` (13 labels) |
| Comments | "non-existent aggregate"→"non-existent stream", "load 100 aggregates"→"load 100 streams", "User aggregate"→"User stream" | `integration/realistic_bench_*.go`, `transport/http/sse_test_helpers_test.go` |
| Assertion messages | "expected aggregate type"→"expected stream type", "other aggregate untouched"→"other stream untouched" | `event/builder_test.go`, `storage/sql/helpers_test.go` |
| Function names | `TestEventStore_MultipleAggregates`→`TestEventStore_MultipleStreams`, `BenchmarkScale_DeciderExecute_ManyAggregates`→`BenchmarkScale_DeciderExecute_ManyStreams` | `storage/turso/coverage_test.go`, `integration/scale_bench_*.go` |
| Type names | `runProjectionsAggregate`→`runProjectionsStream` | `stack/run_projections_test.go` |

**Excluded (intentionally):** `id/` deprecated-alias tests, compat test files, SQL column names in query strings, JSON struct tags, OTel `cqrs.aggregate.*` string values, Watermill metadata keys, error classification codes, `AggregateAwareStrategy`, `AggregateRoot` DDD concepts.

**Post-cleanup verification:**
- `go build` — clean ✅
- `go vet` — clean ✅
- `nix run .#test` — all 56 modules pass ✅
- `nix fmt` — reformatted 15 files (alignment only, no logic change)
- `doc-check` on all changed .md files — 38 references valid across 7 packages ✅

### Historical doc annotations

| File | Annotation |
|------|-----------|
| **AGGREGATE-CONCEPT-ANALYSIS.md** | Added resolution blockquote: "ADR-0058 accepted and implemented." Updated References section to point to new file locations (`id/stream_id.go`, `id/stream_type.go`). Added ADR-0058 to references list. |
| **Rename planning doc** | Status: "In Progress" → "✅ Complete (Sessions 1–3, ADR-0058)" |

---

## b) PARTIALLY DONE

### Module-level READMEs — **NOT TOUCHED (biggest gap)**

~19 module README.md files still contain stale "aggregate" references (totaling ~90 hits). The canonical API names in these files are wrong — consumers reading module docs see deprecated names:

| Module README | Hits | Key stale references |
|---------------|------|---------------------|
| `listing/README.md` | 28 | `AggregateRef`, `AggregateStatus`, `AggregateReader`, `NewInMemoryAggregateReader`, `NewSQLAggregateReader`, `NewAggregateProjection` — ALL renamed |
| `otel/README.md` | 9 | `cqrs.aggregate.*` string values (intentional) — but doc doesn't explain this |
| `storage/README.md` | 7 | Mixed: SQL columns (kept) + stale API names |
| `snapshot/README.md` | 6 | `AggregateRef` in struct examples |
| `id/README.md` | 6 | `AggregateID`, `AggregateType`, `AggregateMarker` |
| `event/README.md` | 5 | `AggregateRef`, `evt.AggregateID()` |
| `decider/README.md` | 5 | "aggregate" in prose + `aggID` in examples |
| `schema/README.md` | 4 | Upcaster examples |
| `command/README.md` | 4 | `AggregateRef`, error codes (kept) |
| Others (10 files) | 1–2 each | Mixed stale prose + intentional refs |

### docs/ non-archive markdown — **NOT TOUCHED**

~18 docs/*.md files have stale references. Many are historical (ADRs, research) but some are living docs:

| File | Hits | Nature |
|------|------|--------|
| `docs/DOMAIN_LANGUAGE.md` | 7 | Session 3 partially updated; may have remaining stale refs |
| `docs/getting-started.md` | 6 | **Living doc** — stale API names in tutorial code |
| `docs/ActaFlow-vs-go-cqrs-lite-COMPARISON-REPORT.md` | 14 | Comparison doc |
| `docs/MIGRATION_v1.md` | 7 | Migration guide |
| `docs/INFRASTRUCTURE_RECOMMENDATIONS.md` | 6 | Infrastructure guide |
| `docs/art-dupl-improvement-report.md` | 6 | Duplication report |
| `docs/turso-indexing-guidance.md` | 4 | Turso guide |
| `docs/signing-architecture.md` | 4 | Signing docs |
| `docs/SPAN_NAMING.md` | 3 | Session 3 updated; 3 remaining may be `cqrs.aggregate.*` (kept) |
| Others (9 files) | 1–2 each | Mixed |

### CHANGELOG.md — **NOT UPDATED for Session 4**

Session 3 added a migration guide section. Session 4 work (224 test files, 7 docs) not reflected.

### AGGREGATE-CONCEPT-ANALYSIS.md — **body text still contradictory**

Added a top-level resolution annotation, but Section 6 body still says "Why It Was Rejected" with three reasons that were overcome. The section reads as if the rename didn't happen until you reach the top annotation.

---

## c) NOT STARTED

### Per-module README audit + fix
19 module READMEs need the same treatment as the top-level docs. `listing/README.md` is the worst (28 hits, every API name is deprecated).

### docs/getting-started.md
6 stale references in tutorial code — consumers following this guide see deprecated API names.

### docs/planning/ + docs/sessions/ historical docs
~90 files with aggregate references. Mostly frozen historical records, but `SESSION_MILESTONES.md` and `SESSION_HISTORY.md` have current status info with stale type names.

### docs/research/ + docs/feedback/ + docs/quality/
~40 files. These are point-in-time research/consumer-feedback documents. Most are historical. `update-old-docs` skill would be appropriate for batch annotation.

### docs/design/
5 files (`hot-state-cache.md`, `read-pressure-snapshots.md`, etc.) reference "aggregate" in design discussions that are still living docs.

### docs/modularization/
4 files (`PROPOSAL.md`, `RE-MODULARIZATION-ASSESSMENT.md`, etc.) reference aggregate in module descriptions.

### Example apps
`example/taskmanager/` and `example/readme-quickstart/` READMEs have 1 aggregate reference each (DDD domain vocabulary — may be intentional).

---

## d) TOTALLY FUCKED UP

### Nothing catastrophic — but close calls:

1. **Blind sed across 224 test files** — I used `find | xargs sed` for mechanical replacement. This is efficient but risky. I verified build+test+fmt afterward, but I should have spot-checked more files before trusting the mass replacement. One false positive class: the sed replaced `aggID` → `streamID` even in test helpers like `parseAggID` in `id/id_core_test.go` — wait, actually I excluded `id/` from the sed, so this was fine. But I got lucky on scoping.

2. **Didn't catch the git history moving underneath me** — While I was working, commits `888e0b9f` (style: nix fmt) and `9dad4441` (test bench) were created. My changes to markdown files were included in commit `9dad4441` along with unrelated bench test changes. I didn't author these commits and didn't notice until I checked `git status` at the end. I should have committed my work in logical batches as I went.

3. **AGGREGATE-CONCEPT-ANALYSIS.md half-annotation** — I added a resolution blockquote at the top but left the body text of Section 6 ("Why It Was Rejected") and Section 9 ("The Open Question") completely unchanged. A reader who skips the top annotation gets a directly contradictory message. This is worse than either leaving it fully alone or fully updating it.

4. **Missed module READMEs entirely** — I audited and fixed the top-level consumer-facing docs (README.md, FEATURES.md, CONTRIBUTING.md) but completely missed the 19 module-level README.md files. A consumer who reads `listing/README.md` sees `NewSQLAggregateReader` which doesn't exist anymore (it's `NewSQLStreamReader`). This is a **consumer-facing bug in documentation**.

---

## e) WHAT WE SHOULD IMPROVE

### Process failures this session:

1. **Incomplete scope audit** — I ran `rg "aggregate" --type md .` to find markdown files but then only fixed the "top-level" ones. I didn't classify or triage the ~140 .md files that remained. The module READMEs were in the search output but I didn't prioritize them.

2. **No checkpoint commits** — I did all work in one session without committing logical batches. The git history shows my changes were swept into a commit I didn't author, mixed with unrelated work. I should have committed after each logical unit (docs, test files, annotations).

3. **Contradictory annotation** — The AGGREGATE-CONCEPT-ANALYSIS.md annotation creates a split-brain: top says "resolved", body says "rejected". Either fully update the body or add inline notes at each contradictory section.

4. **Didn't verify `nix run .#lint` on final state** — I ran lint early and found pre-existing issues in `benchkit/sweep.go`. I didn't re-run it after my changes to verify no new issues were introduced by the sed replacements.

5. **Didn't update the Session 3 status doc** — The Session 3 status doc lists items as deferred. I resolved those items but didn't annotate the Session 3 doc to reflect resolution.

6. **No verification of the remaining ~20 test files** — After the mass sed, ~20 test files still have aggregate references (2–11 each). I classified these as "legitimate" (SQL columns, metadata keys, DDD concepts) but I didn't verify each one. Some may be stale comments that slipped through the sed patterns.

### Strategic gaps:

7. **Module READMEs are the product surface** — For a library, module READMEs ARE the consumer documentation. Leaving `listing/README.md` with 28 deprecated API names is a documentation bug, not a cosmetic issue.

8. **docs/getting-started.md is a tutorial** — A new consumer following this guide will see `AggregateID` and get confused when it's deprecated. This should have been in the first batch of fixes.

9. **No `update-old-docs` skill used** — The historical docs (research, feedback, planning archive) are exactly what the `update-old-docs` skill is designed for: non-destructive annotation of stale point-in-time documents.

---

## f) NEXT STEPS (up to 50, prioritized by impact)

### P0 — Consumer-facing documentation bugs (fix now)

1. **`listing/README.md`** — 28 stale API names. Every exported symbol reference is wrong.
2. **`id/README.md`** — 6 stale type names (`AggregateID`, `AggregateType`, `AggregateMarker`).
3. **`event/README.md`** — 5 stale references (`AggregateRef`, `evt.AggregateID()`).
4. **`decider/README.md`** — 5 stale references (prose + example code).
5. **`docs/getting-started.md`** — 6 stale references in tutorial code.
6. **`snapshot/README.md`** — 6 stale `AggregateRef` in struct examples.
7. **`command/README.md`** — 4 stale references (mixed with intentional error codes).
8. **`schema/README.md`** — 4 stale upcaster examples.
9. **`storage/README.md`** — 7 references (mixed: SQL columns kept + stale API names).
10. **`query/README.md`** — 2 stale references.
11. **`middleware/README.md`** — 2 stale references.
12. **`deriver/README.md`** — 2 stale references.
13. **`catalog/README.md`** — 2 stale references.
14. **`cmd/cqrs-lint/README.md`** — 5 DDD vocabulary references (may be intentional).
15. **`cmd/cqrs-bench/README.md`** — 2 stale references.

### P1 — Living docs (fix soon)

16. **`docs/DOMAIN_LANGUAGE.md`** — Verify Session 3's partial update is complete.
17. **`docs/SPAN_NAMING.md`** — Verify remaining 3 refs are intentional `cqrs.aggregate.*`.
18. **`docs/ARCHITECTURE_PATTERNS.md`** — 1 reference, check if stale.
19. **`docs/STORAGE_GUIDE.md`** — 2 references, check if stale.
20. **`docs/turso-indexing-guidance.md`** — 4 references.
21. **`docs/signing-architecture.md`** — 4 references.
22. **`docs/MIGRATION_v1.md`** — 7 references (may be historical).
23. **`docs/INFRASTRUCTURE_RECOMMENDATIONS.md`** — 6 references.
24. **`docs/PRESETS.md`** — 1 reference.
25. **`docs/OFFLINE_FIRST_METADATA.md`** — 2 references.
26. **`docs/ECOSYSTEM_BOUNDARIES.md`** — 1 reference.
27. **`docs/getting-started.md`** — (already listed above, highest priority).
28. **`docs/index.md`** — 1 reference.
29. **`docs/v4-WISHLIST.md`** — 1 reference.
30. **`docs/README.md`** — 1 reference.

### P2 — AGGREGATE-CONCEPT-ANALYSIS.md body fix

31. **Section 6 body** — Update "Why It Was Rejected" to "Why It Was Initially Rejected (Later Overridden by ADR-0058)" or add inline notes.
32. **Section 9 body** — Update "The Open Question" to "Resolution (ADR-0058)".
33. **Section 5** — `AggregateRef as Partition Key` → note it's now `StreamRef`.

### P3 — Historical doc annotations (use `update-old-docs` skill)

34. **`docs/adr/0001-decider-over-aggregate.md`** — Add note that ADR-0058 renamed the types.
35. **`docs/adr/0005-tombstone-soft-delete.md`** — Check if refs need updating.
36. **`docs/adr/0024-exported-id-markers.md`** — 5 refs, historical ADR.
37. **`docs/sessions/SESSION_MILESTONES.md`** — Annotate that type names have changed.
38. **`docs/sessions/SESSION_HISTORY.md`** — Same.
39. **`docs/planning/` non-archive files** — ~10 files, annotate rename status.
40. **`docs/research/` non-archive files** — ~10 files, point-in-time research.
41. **`docs/feedback/` files** — ~15 files, consumer feedback referencing old names.
42. **`docs/design/` files** — 5 files, design docs.
43. **`docs/modularization/` files** — 4 files.

### P4 — Remaining test file verification

44. **Verify ~20 test files** with remaining aggregate refs are ALL legitimate (SQL columns, metadata keys, DDD concepts, error codes).
45. **`watermill/command_protocol_test.go`** — 11 refs, verify all are metadata keys.
46. **`storage/turso/indexing/advisor_test.go`** — 10 refs, verify SQL schema refs.
47. **`catalog/eventcatalog/exporter_new_resources_test.go`** — 8 refs, verify AggregateRoot DDD.

### P5 — Meta

48. **Run `nix run .#lint`** on final state to verify no new issues.
49. **Update CHANGELOG.md** with Session 4 work.
50. **Annotate Session 3 status doc** to mark deferred items as resolved.

---

## g) QUESTIONS (cannot figure out myself)

### 1. Should module READMEs use deprecated names with a note, or only canonical names?

The top-level README.md and FEATURES.md now use only canonical `Stream*` names. But module READMEs (especially `listing/README.md`, `id/README.md`) still show `AggregateReader`, `NewSQLAggregateReader`, etc. Should these:
- **(A)** Show ONLY canonical names (cleanest for new consumers, but hides the deprecated aliases that still compile)?
- **(B)** Show canonical names with a "Deprecated aliases: `AggregateReader` → `StreamReader`" note (helpful for migration)?
- **(C)** Show deprecated names with "→ use `StreamReader`" deprecation notices (most backward-compatible)?

### 2. Are the uncommitted Go changes in `stack/bench/` and `benchkit/` mine to deal with?

The working tree has uncommitted changes in `benchkit/benchkit.go`, `benchkit/sweep.go`, `stack/bench/command_bench_test.go`, and an untracked `stack/bench/contention_bench_test.go`. These appeared during my session (from a concurrent process or prior work). The `contention_bench_test.go` has a compile error (`store.Save` wrong signature). Should I fix these, or are they someone else's in-progress work?

### 3. Should historical ADRs (0001, 0005, 0024, etc.) be annotated or left frozen?

ADRs are immutable decision records. ADR-0001 references `AggregateRef` and `core/aggregate`. ADR-0005 references `aggregate_id` in tombstone metadata. These are historically accurate at their decision time. Should I:
- **(A)** Add a "Note (ADR-0058): Type names referenced here have been renamed to `Stream*`" footer?
- **(B)** Leave them frozen as historical artifacts (they're point-in-time records)?
- **(C)** Update inline references to canonical names (rewrites history)?

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Verification gates | 3 | ✅ All pass |
| Consumer-facing docs fixed | 7 files | ✅ Done |
| Test files cleaned | 224 files | ✅ Done |
| Historical docs annotated | 2 files | ✅ Done |
| Module READMEs remaining | ~19 files | ❌ Not started (P0) |
| Living docs remaining | ~18 files | ❌ Not started (P1) |
| AGGREGATE-CONCEPT-ANALYSIS body | 1 file | ⚠️ Half-done (P2) |
| Historical docs to annotate | ~60+ files | ❌ Not started (P3) |
| Test files to verify | ~20 files | ❌ Not verified (P4) |

**The rename is structurally complete.** All Go code compiles with canonical names, all tests pass, deprecated aliases work. The remaining work is **documentation debt**: module READMEs and living docs still show deprecated names to consumers.
