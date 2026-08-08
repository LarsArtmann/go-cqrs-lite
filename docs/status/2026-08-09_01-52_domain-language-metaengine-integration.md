# Status Report: DOMAIN_LANGUAGE.md Metaengine Integration

**Date:** 2026-08-09 01:52
**Session goal:** Make `docs/DOMAIN_LANGUAGE.md` superb and ensure it supports the metaengine.
**Result:** Substantially complete — one verification gap remains.

---

## a) FULLY DONE

1. **Record (Structural Foundation) subsection** added under Core Concepts (ADR-0111). Covers Record, StreamRef (record), CommonMetadata, ActorID, ClientCreatedAt, ServerReceivedAt, ServerStoredAt, SchemaVersion. Clarifies the Record vs Event vs Command distinction.
2. **Metaengine major section** added as a new top-level section (16 subsections):
   - Core Concepts (Engine, EngineProfile, Store, Plan/PlanResult, QueryAssignment, Collection, QueryDecl, QueryConfig)
   - ADTs table (all 11: Map, Sorted Map, Multimap, Counter, Set, Log, Stream Log, Graph, Vector, Search, Spatial) with Backend interface names
   - Fold DSL (Fold, On, OnTyped, OnRecord, FoldKind, Delta, Edge, MultiEntry, Skip, Remove, Poison)
   - Cost Model (Complexity, CostEstimate, NsPerOp, ReadCosts, ScaleThreshold, Volume, LatencyBudget, WriteAmplification)
   - Storage Layouts (Row, Columnar, LSM, KV) + Layout Planning (LayoutPlan, PlannedColumn, LayoutPlanner, LayoutPlanApplier, WithColumnarLayout)
   - Read Patterns (Point Lookup, Filtered Scan, Aggregate, Traversal, Scan)
   - Filter, Sort & Pagination (FilterSpec, SortSpec, FilterOnField, SortOnField, FilterOn, Cursor, TypedReader, QueryBuilder)
   - PlanRule Pipeline (PlanRule, RulePipeline, RuleTraceEntry, Diagnostic + 4 levels: SCREAM/DEGRADED/WARN/INFO)
   - Persistence Model (Volatile, Persistent)
   - Replication Model (None, Single-Leader, Multi-Leader, Leaderless, Replication Lag, Network RTT)
   - Materialize vs Replay (WorkloadStats, ReplayCost, MaterializeCost, ShouldMaterialize)
   - Temporal Reads (VersionedStorage, AsOfSignal)
   - Plan Operations (SerializablePlan, PlanDiff, Manifest, DryRun, ExplainPlan, Doctor)
   - Engine Capabilities (PushdownScan, StreamingScan, RawValueReader, AtomicAppender, SnapshotBackend, MapUpdater, AggregateReader)
   - Hot Operations (TieredStore, Watcher, SwapEngine, PrefetchCache)
   - Engines table (8 engines with type and module path)
3. **Cross-Cutting cleanup**: Removed the 2 stale single-row Metaengine/Planner entries that were buried in the Cross-Cutting table, replaced with a single cross-reference link to the new dedicated section.
4. **Anti-Patterns table** extended with 2 metaengine entries: "Manual query routing" → "Cost-based planner" and "Hand-written DDL" → "LayoutPlan auto-generation".
5. **Interface Hierarchy** code block extended with the full metaengine Engine interface tree: 13 optional ADT backends (ISP), 8 optional capabilities, and the PlanRule/RulePipeline lifecycle interfaces.
6. **Verification code block** updated:
   - Added `record/v4` and `metaengine/v4` imports
   - Added `errorfamily.NewOrchestration` (was missing — the error taxonomy table listed it but the verification block didn't reference it)
   - Added 3 record symbols: `record.Record{}`, `record.NewStreamRef`, `record.CommonMetadata{}`
   - Added 41 metaengine symbols spanning all 16 subsections
   - **doc-check passes: 144 references valid across 41 packages**
7. **File grew from ~572 lines to 903 lines** — the metaengine is now first-class in the domain language.

---

## b) PARTIALLY DONE

1. **Metaengine ADRs not cross-referenced individually.** The new section links to ADRs 0061, 0062, 0063, 0092, 0098, 0111, 0113, 0114, 0115, 0116, 0117 but only as a range in the intro. Individual subsections could link to their specific ADR (e.g., Persistence → ADR-0098, Replication → ADR-0093). This would improve navigability but was not done to avoid over-linking.
2. **The metaengine section references some concepts that exist only in design docs, not yet in code.** The v2 vision ADRs (0111-0117) describe auto-projection, ES-native planning, and command lifecycle as events — these are design-stage, not yet implemented. The domain language describes both the current state and the intended vocabulary. This is arguably correct (domain language should capture intent) but could be marked more clearly.

---

## c) NOT STARTED

1. **No changes to `SKILL.md` or skill references.** The AI consumer skill (`.agents/skills/go-cqrs-lite/`) was not touched. If the skill references DOMAIN_LANGUAGE.md for vocabulary, it may want updating too — but the skill has its own `references/` files that are checked by doc-check separately.
2. **No `CHANGELOG.md` update.** This was a doc-only change; CHANGELOG was not touched.
3. **No `AGENTS.md` module table update.** The metaengine entry in AGENTS.md already has extensive detail; the DOMAIN_LANGUAGE.md now complements it.
4. **No verification that the metaengine section is consistent with the canonical design docs** (`docs/planning/meta-engine-design.md`, `meta-engine-project-definition.md`, `meta-engine-assumptions-and-query-planning.md`). The domain language was built from the code and the agent's extraction; a cross-check against the design docs for vocabulary drift was not performed.
5. **No `nix run .#verify` run.** Only `doc-check` was run (the directly relevant gate). The full verify gate (build/vet/test/lint/race) was not run because no code changed — only a markdown doc.

---

## d) TOTALLY FUCKED UP

Nothing. All changes are doc-only, doc-check passes, and no code was modified.

---

## e) WHAT WE SHOULD IMPROVE

1. **Individual ADR cross-references.** Each metaengine subsection should link to its specific ADR, not just a range. Low effort, high navigability payoff.
2. **Mark design-stage vs implemented vocabulary.** The v2 vision terms (auto-projection from Record types, command lifecycle as event streams, tombstone-as-domain-event) appear in the domain language but are not yet in the codebase. A reader could be misled. Consider a "(planned)" or "(design-stage)" annotation.
3. **The Record subsection duplicates some StreamRef content.** The Core Concepts section already has a StreamRef entry under Event Sourcing (`id.NewStreamRef`), and the Record section adds `record.StreamRef`. Both are the same concept at different layers. The relationship should be made explicit (the Record section does say "distinct from `id.StreamRef` (branded ID) but same concept" — but this could be clearer).
4. **Metaengine engine submodules not in the verification block.** The Engines table lists `metaengine/sqliteengine/`, `metaengine/pebbleengine/`, `metaengine/duckdbengine/`, etc. but the verification block doesn't import or reference any of them (only core `metaengine/v4`). If a reader follows the table to import an engine, doc-check doesn't verify those paths exist.
5. **The "Saga" entry in Messaging says "Not a module"** — but the metaengine's `deriver` module IS the building block for saga patterns, and it's in the CQRS table. The saga anti-pattern could cross-reference the deriver more explicitly now that the metaengine's fold-based approach is documented.
6. **The metaengine section doesn't mention `system/` (the deployer-driven composition root).** The system module auto-wires metaengine stores into projections. This integration point is documented in AGENTS.md but not in the domain language.
7. **No "When to use Metaengine vs manual Read Models" decision guide.** The section explains what the metaengine IS but doesn't explicitly say when a consumer should choose it over the three manual projection tiers. The intro has a one-liner but a small decision table would help.

---

## f) Up to 50 Things to Get Done Next

### DOMAIN_LANGUAGE.md Polish (high ROI, low effort)
1. Add individual ADR links to each metaengine subsection (Persistence→0098, Replication→0093, Layout→0073/0092, etc.)
2. Add "(planned)" annotations to v2 vision terms not yet in code
3. Add a "When to use Metaengine vs Manual Read Models" decision table at the top of the Metaengine section
4. Clarify the `record.StreamRef` vs `id.StreamRef` relationship with a diagram or explicit note
5. Add metaengine engine submodule paths to the verification block (`sqliteengine`, `pebbleengine`, `duckdbengine`, `pgengine`, `badgerengine`, `dgraphengine`, `irohengine`)
6. Cross-reference the `deriver` module from the Saga anti-pattern entry
7. Add `system/` to the metaengine section (deployer-driven composition root auto-wires metaengine stores)
8. Add `projectionadapter` to the metaengine section (the bridge from metaengine Store to projection.Projection)
9. Add `enginetest` to the Tooling section (exported engine test harness: RunMatrix, RunTransactionalTest)
10. Add `adttest` to the Tooling section (exported ADT test harness for cross-engine parity)

### Content Completeness
11. The Read Patterns table lists 5 patterns but the code defines 11 (ReadMembership, ReadMultiLookup, ReadLogTail, ReadVectorSearch, ReadFullTextSearch, ReadSpatialRange are missing). Add them.
12. The Cost Model section mentions `ComplexityODegree` for graph traversal but doesn't explain the cost formula `estimated_latency = (ops × nsPerOp / 1e6) + NetworkRTT`. Add it.
13. The Fold DSL section doesn't mention `AutoInsert`/`AutoUpdate` (reflection-based auto-folds that stamp Record metadata). Add them.
14. The Engine Capabilities section doesn't mention `GroupedAggregateReader` or `MultiAggregateReader` (vectorized columnar aggregation). Add them.
15. The Hot Operations section doesn't mention `MapUpdateTyped` (typed atomic read-modify-write helper). Add it.
16. The Plan Operations section mentions `ExplainPlan()` and `Doctor()` but doesn't mention `Store.Collections()`, `Store.ReplicationMode()`, `Store.Persistence()`. Add them.
17. The Fold DSL doesn't mention `Embedding` (vector fold input) or `IndexedText` (search fold input). Add them.
18. The Engines table doesn't mention Calibratable interface (engines that can be benchmark-calibrated). Add a note.
19. The metaengine section doesn't mention `metaengine/bench/` (cross-engine benchmarking module). Add to Tooling.
20. The metaengine section doesn't mention `metaengine/keycodec/` (composite key encoding for LSM backends). Add it.

### Verification & Gates
21. Run `nix run .#verify` to confirm the doc change didn't break anything downstream (doc-check is the relevant gate, but verify is the full safety net)
22. Run the api-stability golden check to see if any exported symbols referenced in the doc changed (`cd cmd/api-stability && GOWORK=off go run main.go`)
23. Run `nix run .#doc-check` (the flake version) to confirm the flake-configured path matches the manual run
24. Add a CI gate that checks DOMAIN_LANGUAGE.md line count growth (prevent stale sections — if the codebase adds modules but the doc doesn't grow, flag it)

### Broader Documentation Health
25. Check if `SKILL.md` and skill references need metaengine vocabulary updates (the skill is the AI consumer guide)
26. Check if `FEATURES.md` reflects the metaengine's current feature set
27. Check if `TODO_LIST.md` has stale metaengine tasks
28. Run the `docs-health` skill for a full documentation audit
29. Update `CHANGELOG.md` with the DOMAIN_LANGUAGE.md metaengine integration
30. Check `docs/README.md` (docs index) links to the domain language

### Metaengine Deep Cuts (for a future session)
31. Document the `ContractSuite` testing helper (exported for engine implementors)
32. Document `metaengine/irohengine/` three-tier transport testing pyramid in the domain language
33. Document the `loopback/` and `quic/` transport modules
34. Document `metaengine/projectionadapter/` EventWithID and NewTypeDecoder helpers
35. Document the `metaengine/dsl.go` DX helpers (PlanFromSQLite, PlanFromMemory, Store.LogPlan)
36. Document the SSE replay ring buffer (`sse_replay.go`, `WithSSEReplayLimit`)
37. Document cursor-encoded prefetch (`WithCursorString`, `PrefetchCache`)
38. Document the `branded_units.go` types (calibrated time-branded cost units)
39. Document the `consistency.go` module (consistency model helpers)
40. Document the `export_import.go` module (plan export/import for CI)
41. Add a "Metaengine Design Doc Vocabulary Glossary" that maps domain language terms to the design docs
42. Verify the metaengine section against `docs/planning/meta-engine-design.md` for vocabulary drift
43. Verify against `docs/planning/meta-engine-project-definition.md`
44. Verify against `docs/planning/meta-engine-assumptions-and-query-planning.md`
45. Check if the `docs/MIGRATION-kv-to-metaengine.md` doc uses consistent vocabulary

### Polish
46. Add a table of contents at the top of DOMAIN_LANGUAGE.md (the file is now 903 lines — a ToC would help navigation)
47. Add section anchors/IDs for cross-referencing from other docs
48. Consider splitting the metaengine section into its own `docs/METAENGINE_DOMAIN_LANGUAGE.md` if it grows further
49. Add "see also" cross-references between related sections (e.g., Snapshot ↔ SnapshotBackend, Checkpoint ↔ metaengine PlanResult)
50. Standardize the "Context" column format — some entries use backtick-wrapped Go symbols, others use prose. Pick one style.

---

## g) Questions I Cannot Answer Myself

1. **Should the metaengine have its own separate domain language file?** The metaengine section is now ~230 lines of a 903-line file (25%). The project's own AGENTS.md calls the metaengine "THE STRATEGIC FUTURE of this project (possibly a future dedicated project)." If it's destined to spin out, a separate `docs/METAENGINE_DOMAIN_LANGUAGE.md` might be better than embedding it in the main file. I can't decide this — it's a product/architecture direction question.

2. **Should design-stage vocabulary (ADR 0111-0117, not yet implemented) be in the domain language?** The domain language is supposed to be the "ubiquitous language" — words everyone uses. But some of the v2 vision terms (auto-projection from Record types, command lifecycle as event streams, tombstone-as-domain-event) describe a future state. Including them sets expectations; excluding them means the doc only reflects reality. This is a documentation philosophy call I shouldn't make alone.

3. **Is the Record subsection in the right place?** I put it under Core Concepts (before Storage). But Record is the structural foundation for the metaengine's ES-native vision (ADR-0111), and the metaengine section is after Read Models. An alternative placement: move the Record subsection to immediately before the Metaengine section, since that's where it's most relevant. Or keep it in Core Concepts since it's foundational to everything. I can't determine the "right" information architecture without knowing how readers navigate this file.
