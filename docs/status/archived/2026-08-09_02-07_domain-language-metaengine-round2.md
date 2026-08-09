# Status Report: DOMAIN_LANGUAGE.md Metaengine Integration — Round 2

**Date:** 2026-08-09 02:07
**Session goal:** Execute all remaining improvement items from the prior status report (50-item list), making the domain language comprehensive and verified.
**Result:** 16 tasks completed. All gaps from the prior report's top-10 priorities are closed. doc-check passes: **174 references valid across 50 packages** (up from 144/41).

> ✅ **ARCHIVED 2026-08-09** — All 18 items shipped; 3 open questions resolved
> in the follow-up split session (`2026-08-09_02-43`). The metaengine content
> was subsequently split into `docs/METAENGINE_DOMAIN_LANGUAGE.md`. Read that
> file + the `2026-08-09_02-57` followup for current state.

---

## a) FULLY DONE

1. **Read Patterns table completed.** Added all 6 missing patterns: `ReadMembership`, `ReadMultiLookup`, `ReadLogTail`, `ReadVectorSearch`, `ReadFullTextSearch`, `ReadSpatialRange`. The table now matches the 11 patterns defined in `metaengine/types.go:20-32` exactly.

2. **Fold DSL section expanded.** Added `AutoInsert[E,R]`, `AutoUpdate[E,R]`, `AutoDelete[E]`, and `AutoCRUD[C,U,D,R]` — the reflection-based auto-folds from `metaengine/auto_fold.go`. These are the 80%-auto-generation building blocks promised by ADR-0116.

3. **Engine Capabilities section expanded.** Added `GroupedAggregateReader` (SQL GROUP BY) and `MultiAggregateReader` (batch aggregation) from `metaengine/aggregations.go`.

4. **Hot Operations section expanded.** Added `MapUpdateTyped[V]` — the typed atomic read-modify-write helper from `metaengine/dx.go`.

5. **ADT section enriched.** Added typed fold input types: `Embedding` (ID + `[]float32`) for Vector ADT and `IndexedText` (ID + content) for Search ADT, from `metaengine/vector_search.go`.

6. **Engines table enriched.** Added `Calibratable` interface note — engines accepting benchmark-calibrated `SetCalibration(CalibrationCosts)`. Currently: Memory, SQLite, Pebble, Badger, DuckDB.

7. **Cost Model section enriched.** Added the core cost formula: `estimated_latency = (ops × nsPerOp / 1e6) + NetworkRTT` — the unifying equation that connects complexity, calibration, and replication.

8. **Plan Operations section expanded.** Added `Store.Collections()`, `Store.ReplicationMode(queryName)`, and `Store.Persistence(queryName)` — the introspection API for operations and dashboards.

9. **Projection Adapter subsection added.** New subsection under Metaengine documenting `projectionadapter.New`, `WithEventDecoder`, `EventWithID[P]`, and `NewTypeDecoder` — the bridge from metaengine Store to `projection.Projection`.

10. **System (Deployer Composition Root) subsection added.** Under Stack Bundles: `system.New`, `DomainConfig`, `DeploymentConfig`, `GracefulClose`, `Health`, `Snapshot`.

11. **Tooling section expanded.** Added `cqrs-lint`, `cqrs-bench`, `enginetest` (RunTransactionalTest, RunStreamLogBackendTest, RunAtomicAppenderTest), `adttest` (RunMatrix), `benchkit` (Run/Compare), and `FlightRecorder`.

12. **"When to use Metaengine vs Manual" decision table added.** At the top of the Metaengine section — helps consumers choose between manual projection tiers and the metaengine.

13. **Design-stage vocabulary annotated.** Added implementation-status note: ADRs 0061-0098 are implemented; ADRs 0111-0117 are v2 vision (partially or planned). Terms marked **(planned)** where appropriate.

14. **Individual ADR cross-references added.** Persistence → ADR-0098, Replication → ADR-0093, Storage Layouts → ADR-0073 + ADR-0092. Previously only a range link existed.

15. **StreamRef relationship clarified.** The `record.StreamRef` vs `id.StreamRef` distinction now says: "the string form" vs "the branded typed form. Same concept, different type layer."

16. **Interface Hierarchy code block updated.** Added `GroupedAggregateReader`, `MultiAggregateReader`, and `Calibratable` to the metaengine Engine interface tree.

17. **Verification block expanded.** Added 30 new verified symbols (+9 new packages): engine constructors (sqliteengine, pebbleengine, pgengine), projectionadapter, adttest, enginetest, system, flightrecorder, benchkit, plus all new metaengine core symbols (ReadPatterns, AutoInsert/AutoUpdate/AutoDelete/AutoCRUD, MapUpdateTyped, Embedding, IndexedText, GroupedAggregateReader, MultiAggregateReader, Calibratable).

18. **doc-check passes:** 174 references valid across 50 packages. Up from 144/41 at session start.

---

## b) PARTIALLY DONE

1. **Engine submodule verification is incomplete.** The verification block imports `sqliteengine`, `pebbleengine`, `pgengine`, `projectionadapter`, `adttest`, and `enginetest` — but NOT `duckdbengine`, `badgerengine`, `dgraphengine`, `irohengine`, or `graphadapter`. These 5 engines appear in the Engines table prose but are not in the verification block. Reason: `duckdbengine` requires CGo, and the others pull heavy deps. doc-check uses static parsing so CGo isn't the issue, but the symbols were not added. They CAN be added.

2. **The (planned) annotation is a block-level note, not inline per-term.** The implementation-status blockquote at the top of the Metaengine section explains which ADRs are implemented vs planned, but individual terms in the tables below (e.g., "Auto-projection from Record types") don't carry inline "(planned)" tags. A reader scanning a specific table row wouldn't see the annotation without reading the intro note.

3. **`keycodec` and `metaengine/bench` are not in the Tooling section.** They exist in the codebase but weren't added. `keycodec` is a utility package (composite key encoding for LSM backends); `metaengine/bench` is the cross-engine benchmark module.

4. **The Saga anti-pattern entry mentions `deriver` but doesn't cross-link.** The Saga row in both the Messaging table and the Anti-Patterns table reference `deriver` by name but don't link to the CQRS section's Deriver definition. A hyperlink or section reference would improve navigability.

---

## c) NOT STARTED

1. **No table of contents.** The file is now 1019 lines with 31 section/subsection headings. A ToC at the top would help navigation significantly.

2. **No `nix run .#verify` run.** Only doc-check was run (the directly relevant gate). The full verify gate (build/vet/test/lint/race) was not run because no code changed — only a markdown doc. This is correct for this change set.

3. **No CHANGELOG.md update.** Doc-only change; CHANGELOG not touched.

4. **No SKILL.md update.** The AI consumer skill has its own `references/` files checked separately by doc-check. Not touched.

5. **No consistency check against canonical design docs.** The metaengine section was built from code, not cross-verified against `docs/planning/meta-engine-design.md`, `meta-engine-project-definition.md`, or `meta-engine-assumptions-and-query-planning.md` for vocabulary drift.

6. **No CI gate for DOMAIN_LANGUAGE.md staleness.** No automated check that new modules appear in the domain language when added to the repo.

7. **No section anchors/IDs.** Cross-referencing from other docs into specific sections would require HTML anchors, which aren't present.

---

## d) TOTALLY FUCKED UP

1. **The verification block imports `projectionadapter/v4` but the actual import path may not resolve under doc-check's static parser.** The `projectionadapter` module is `metaengine/projectionadapter/v4` — but `adttest` and `enginetest` are NOT separate modules (they're packages within `metaengine/v4`). The import paths in the verification block use `"github.com/larsartmann/go-cqrs-lite/metaengine/adttest"` (no `/v4`) for these two, which IS correct for sub-packages of the parent module. doc-check passes, confirming the paths resolve. However, I should have been more careful about the module-vs-package distinction when writing the imports — it worked, but by accident of doc-check's static parser stripping `/v4` suffixes, not because I verified the Go import semantics.

2. **The cost formula was added to the Cost Model intro but not to the Replication section.** The formula `estimated_latency = (ops × nsPerOp / 1e6) + NetworkRTT` appears once under Cost Model, but the Replication section also discusses NetworkRTT without cross-referencing the formula. A reader in the Replication section doesn't see how NetworkRTT plugs into the total cost.

---

## e) WHAT WE SHOULD IMPROVE

1. **Add remaining 5 engine submodules to the verification block.** `duckdbengine.New`, `badgerengine.NewBadgerEngine`, `dgraphengine.New`, `irohengine.Replicated`, `graphadapter` — all appear in the Engines table but aren't verified. This is a 10-minute task.

2. **Add a table of contents.** 1019 lines with 31 sections. A markdown ToC with anchor links would be the single highest-ROI navigation improvement.

3. **Add `keycodec` and `metaengine/bench` to Tooling.** Both exist in the codebase and are missing from the domain language entirely.

4. **Inline (planned) annotations on specific terms.** Instead of one block-level note, mark individual v2-vision terms inline: "Auto-projection (planned, ADR-0116)", "Tombstone-as-domain-event (planned, ADR-0114)", etc.

5. **Cross-reference the cost formula from the Replication section.** The Replication section mentions NetworkRTT but doesn't link back to the cost formula that uses it.

6. **Cross-link Saga → Deriver.** Add a `#cqrs` anchor reference from the Saga entry to the Deriver row.

7. **Standardize the Context column format.** Some entries use backtick-wrapped Go symbols (`metaengine.Plan`), others use prose ("returns `[]CollectionInfo`"). Pick one style for consistency.

8. **Add "see also" cross-references between related sections.** E.g., Snapshot ↔ SnapshotBackend, Checkpoint ↔ metaengine PlanResult, Materialize (KV) ↔ Materialize vs Replay (metaengine).

9. **Verify vocabulary consistency against the canonical design docs.** The metaengine section was built from code exports, not from `docs/planning/meta-engine-design.md`. Terms may have drifted.

10. **Add the Iroh three-tier transport testing pyramid.** `InProcessNetwork`, `loopback.LoopbackTransport`, `quic.QuicTransport` — the testing strategy for distributed engines is undocumented in the domain language.

---

## f) Up to 50 Things to Get Done Next

### Verification Block Completion (high ROI, low effort)

1. Add `duckdbengine.New` to verification block (CGo module, but doc-check is static)
2. Add `badgerengine.NewBadgerEngine` to verification block
3. Add `dgraphengine.New` to verification block
4. Add `irohengine.Replicated` to verification block
5. Add `graphadapter` to verification block (if it exports a constructor)
6. Add `keycodec.MapKey` to verification block
7. Add `metaengine/bench` symbols to verification block

### Content Completeness

8. Add `keycodec` subsection to Tooling (composite key encoding for LSM backends)
9. Add `metaengine/bench` to Tooling (cross-engine benchmark module)
10. Add Iroh three-tier transport testing pyramid to Metaengine section
11. Document `loopback.LoopbackTransport` (real TCP, no CGo)
12. Document `quic.QuicTransport` (real QUIC via iroh-go, CGo)
13. Document `InProcessNetwork` (goroutine calls, fastest, no CGo)
14. Document `metaengine/dsl.go` DX helpers (`PlanFromMemory`, `Store.LogPlan`, `sqliteengine.NewFromDSN`, `sqliteengine.PlanFromDSN`)
15. Document SSE replay ring buffer (`WithSSEReplayLimit`, `SSEReplay[V]`)
16. Document cursor-encoded prefetch (`WithCursorString`, `PrefetchCache` key matching)
17. Document `branded_units.go` (calibrated time-branded cost units)
18. Document `consistency.go` (consistency model helpers)
19. Document `ContractSuite` testing helper (exported for engine implementors)

### Navigation & Polish

20. Add a table of contents at the top (1019 lines, 31 headings)
21. Add section anchors for cross-referencing from other docs
22. Inline "(planned)" annotations on specific v2-vision terms (not just block-level)
23. Cross-reference cost formula from Replication section
24. Cross-link Saga entry → Deriver definition
25. Standardize Context column format (backtick symbols vs prose)
26. Add "see also" cross-references between related sections
27. Add Consistency Guarantees entries for metaengine (eventual consistency by default, read-your-writes not guaranteed across engines)

### Structural Improvements

28. Move Projection Adapter subsection to immediately after Engines (currently after the Calibratable note — feels slightly disconnected)
29. Consider whether `system/` belongs under Stack Bundles or warrants its own top-level section
30. Consider whether the Record subsection should move to immediately before Metaengine (it's currently under Core Concepts)

### Verification & Gates

31. Run `nix run .#verify` as a full safety net
32. Run api-stability golden check (`cmd/api-stability`)
33. Run `nix run .#doc-check` (flake version) to confirm parity with manual run
34. Add a CI meta-test that checks every `go.mod` directory has at least one reference in DOMAIN_LANGUAGE.md
35. Add a CI gate for DOMAIN_LANGUAGE.md line-count growth (prevent stale sections)

### Broader Documentation Health

36. Check if `SKILL.md` references need metaengine vocabulary updates
37. Check if `FEATURES.md` reflects the metaengine's current feature set
38. Check if `TODO_LIST.md` has stale metaengine tasks
39. Run the `docs-health` skill for a full documentation audit
40. Update `CHANGELOG.md` with the DOMAIN_LANGUAGE.md expansion
41. Check `docs/README.md` links to the domain language
42. Verify `docs/planning/meta-engine-design.md` uses consistent vocabulary
43. Verify `docs/planning/meta-engine-project-definition.md` vocabulary
44. Verify `docs/planning/meta-engine-assumptions-and-query-planning.md` vocabulary
45. Check if `docs/MIGRATION-kv-to-metaengine.md` uses consistent vocabulary

### Meta

46. Generate a "Metaengine Design Doc Vocabulary Glossary" mapping domain language terms to design docs
47. Add a "How to Read This File" guide for new consumers (suggested reading order)
48. Consider splitting the metaengine section into its own `docs/METAENGINE_DOMAIN_LANGUAGE.md`
49. Add examples to each metaengine subsection (small code snippets showing usage)
50. Add a glossary of DDIA terms referenced throughout (survivability, replication, derived data, etc.)

---

## g) Questions I Cannot Answer Myself

> **RESOLVED (2026-08-09 follow-up session):** All three questions answered.
> Q1: **Yes** — all 5 engine submodules added to the verification block in `docs/METAENGINE_DOMAIN_LANGUAGE.md` (76 references, 13 packages).
> Q2: **Stay under Stack Bundles** — `system/` remains as a subsection; it serves the same consumer need.
> Q3: **Both** — the file was split (metaengine extracted to `docs/METAENGINE_DOMAIN_LANGUAGE.md`) AND both files got table of contents.

1. **Should the 5 remaining engine submodules (duckdbengine, badgerengine, dgraphengine, irohengine, graphadapter) be in the verification block?** doc-check uses static parsing, so CGo isn't a blocker. But adding them increases the maintenance surface — every API change in those modules would need a corresponding doc-check update. Is the verification coverage worth the coupling?

2. **Should `system/` be its own top-level section or stay under Stack Bundles?** The system module is architecturally distinct (deployer-driven composition root, not a one-call preset), but it serves the same consumer need as Stack Bundles (wiring everything together). Moving it would change the file's information architecture.

3. **Should the file have a table of contents?** At 1019 lines with 31 headings, a ToC would clearly help navigation. But markdown ToCs require manual maintenance (no auto-generation in standard markdown), and the project doesn't use a static site generator that could auto-generate one. Is the maintenance cost worth it, or should the file be split instead?
