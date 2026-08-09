# Status Report: Metaengine Domain Language Split

**Date:** 2026-08-09 02:43
**Session goal:** Split metaengine content out of `docs/DOMAIN_LANGUAGE.md` into a dedicated `docs/METAENGINE_DOMAIN_LANGUAGE.md`, hyperlinked bidirectionally.

---

## a) FULLY DONE

1. **Created `docs/METAENGINE_DOMAIN_LANGUAGE.md`** (540 lines) — comprehensive metaengine vocabulary with:
   - Table of contents (21 entries)
   - All 16 content subsections migrated from root: Core Concepts, ADTs, Fold DSL, Cost Model, Storage Layouts, Read Patterns, Filter/Sort/Pagination, PlanRule Pipeline, Persistence, Replication, Materialize vs Replay, Temporal Reads, Plan Operations, Engine Capabilities, Hot Operations, Engines, Projection Adapter
   - **New: Key Codec section** — documents `keycodec` package (MapKey, CollectionPrefix, CounterKey, StreamKey, MultimapKey) — was entirely absent from domain language before
   - **New: Testing & Benchmarking section** — documents `enginetest`, `adttest`, and `metaengine/bench` (bench was entirely absent before)
   - **New: Shared Terms table** — 6 cross-references to root DOMAIN_LANGUAGE.md (Record, Stream, Decider, Projection, Tombstone, Event Store)
   - **New: Metaengine-specific Anti-Patterns** — 4 entries (Manual query routing, Hand-written DDL, God Store, Theoretical cost)
   - Engine interface hierarchy code block (full, with all optional backends + capabilities + lifecycle)
   - **Cost Model ↔ Replication cross-reference** — cost formula now links to Replication section; Replication section links back to Cost Model formula
   - **Own verification block** — 76 references / 13 packages, covering all 9 engines
   - **All 9 engine constructors in verification** — added the 5 that were previously missing (duckdbengine.New, badgerengine.NewBadgerEngine, dgraphengine.New, irohengine.Replicated, graphadapter.New) plus their `From*` variants

2. **Updated `docs/DOMAIN_LANGUAGE.md`** (1019 → 640 lines, -37%):
   - Metaengine section replaced with 8-line stub + link to new file
   - Interface Hierarchy: 30-line metaengine.Engine block → one-line reference link
   - Verification block: removed all metaengine imports + symbols (now in new file)
   - Cross-Cutting: Deriver term added (was only referenced by name, never defined inline)
   - Saga entry now hyperlinks to `#deriver` (was bare `deriver.Deriver`)
   - Read Models section: added note about the metaengine as the automated fourth tier
   - Cross-Cutting metaengine pointer updated from `#metaengine` to `METAENGINE_DOMAIN_LANGUAGE.md`

3. **Updated `cmd/doc-check/main.go`** — added `METAENGINE_DOMAIN_LANGUAGE.md` to auto-discovery list

4. **Updated `flake.nix`** — added BOTH `docs/DOMAIN_LANGUAGE.md` and `docs/METAENGINE_DOMAIN_LANGUAGE.md` to the verify gate (both were previously MISSING from the verify gate entirely — a pre-existing gap)

5. **Verification passed in all modes:**
   - Auto-discovery (CI path): 1421 references / 61 packages ✓
   - Explicit args (flake verify gate): 1447 references / 62 packages ✓
   - Both files together only: 184 references / 56 packages ✓

---

## b) PARTIALLY DONE

1. **`(planned)` inline annotations** — the implementation-status note is block-level ("ADRs 0111-0117 describe the v2 vision... Terms from unimplemented ADRs are marked **(planned)** below"), but no individual terms actually carry the `(planned)` annotation inline. A reader scanning a specific table row (e.g. `OnRecord` which is partially implemented) can't tell from the row itself — they have to read the intro note. This was a known tradeoff from the prior session and was not addressed in this session.

2. **`keycodec` in verification block** — I added `keycodec.MapKey` as a single representative symbol, but the package has 20+ exported functions. Only 1 is verified. The others (CollectionPrefix, CounterKey, StreamKey, MultimapKey, LogPrefix, EncodeJSON, etc.) are in prose but not in the verification block.

3. **`metaengine/bench`** — documented in the Testing & Benchmarking table but has NO symbols in the verification block (it's test-only: all exports are `Benchmark*` functions in `_test.go` files, which doc-check cannot verify since it parses non-test `.go` files). The table entry notes "Test-only (no non-test exports)" which is honest, but the doc-check coverage is zero for this module.

4. **`projectionadapter` partial verification** — only `projectionadapter.New` is in the new verification block. The prose documents `WithEventDecoder`, `EventWithID[P]`, `NewTypeDecoder` — none of these are verified. They were verified in the OLD root file's block either (only `projectionadapter.New` was there), so this is a carried-forward gap, not a regression.

---

## c) NOT STARTED

1. **AGENTS.md update** — The root AGENTS.md references `docs/DOMAIN_LANGUAGE.md` in the project documentation table but does NOT mention `docs/METAENGINE_DOMAIN_LANGUAGE.md`. Anyone reading AGENTS.md to understand doc structure won't know the metaengine split exists.

2. **CONTRIBUTING.md / docs-health** — No update to CONTRIBUTING.md to document the two-file domain language convention (when to add terms to which file).

3. **Stale status report cleanup** — The prior session's status report (`docs/status/2026-08-09_02-07_domain-language-metaengine-round2.md`) references the old single-file structure. Its "50 next steps" list is partially outdated (some items are now done by this split, others remain). Not updated.

4. **CHANGELOG.md** — No entry for the doc restructure.

5. **CI workflow check** — `.github/workflows/ci.yml` line 69 runs doc-check with no explicit args (auto-discovery), which now picks up the new file automatically. But this was NOT tested via CI — only locally. The CI YAML itself needs no change, but this should be verified.

6. **GitHub anchor link verification** — The cross-file anchor links (e.g. `DOMAIN_LANGUAGE.md#read-models`, `DOMAIN_LANGUAGE.md#event-sourcing`, `METAENGINE_DOMAIN_LANGUAGE.md#interface-hierarchy--engine`) are valid GitHub-flavored markdown anchors, but I did NOT verify them with a renderer. They could have edge-case formatting issues on GitHub (spaces in headers → dashes, special chars).

---

## d) TOTALLY FUCKED UP

1. **Em dashes in the new file** — The AGENTS.md says "Never use em dashes in source code; use commas, periods, parentheses, or semicolons instead." I used `—` (em dash) extensively in the new `METAENGINE_DOMAIN_LANGUAGE.md` throughout tables and prose (e.g. "80% auto-generated, 100% auto-routed"). However, this was ALSO present in the original content I migrated from the root file — I preserved the existing style rather than introducing a new violation. The root file already used em dashes in the same content. This is a carried-forward style issue, not a new one, but I should have cleaned it up during the migration since I was rewriting the content anyway. The AGENTS.md rule is about source code specifically ("in source code"), and markdown docs may be a gray area, but the spirit of the rule is clear.

2. **No table of contents in root file** — The root DOMAIN_LANGUAGE.md is now 640 lines with 23 headings and still has NO table of contents. I added a ToC to the new file but did not add one to the root file. The whole point of the split was navigation improvement, and I only improved navigation in half the files.

3. **Did not update the prior status report** — `docs/status/2026-08-09_02-07_domain-language-metaengine-round2.md` has a "3 open questions" section asking exactly the questions this session answered (should we split? should 5 engine submodules be in verification? etc.). I should have updated that report to mark those questions resolved, but I didn't touch it.

---

## e) WHAT WE SHOULD IMPROVE

1. **Missing Deriver row in CQRS section** — The Deriver term is now in Cross-Cutting (added this session) but it's really a CQRS concept (event→command derivation, the reverse of normal flow). It might belong under CQRS or as its own subsection. Currently it's buried at the bottom of Cross-Cutting between High-Water Mark and the metaengine pointer.

2. **No "when to use this file" guidance in root** — The root file's metaengine stub says "Full vocabulary: [link]" but doesn't tell a reader WHEN they need the metaengine vs when the root file suffices. A one-sentence "you need the metaengine file if you're building read models with the cost-based planner" would help.

3. **The root file's verification block is now lighter** — It went from verifying 174 symbols to ~98 (lost the 76 metaengine ones). This is correct behavior (they moved to the new file), but the root file no longer provides ANY verification that metaengine symbols exist. If someone deletes the new file, the root file won't catch it — doc-check's auto-discovery catches it, but the root file is no longer self-contained for metaengine references in its stub text (e.g. `metaengine.Plan` mentioned in the stub is not verified by the root file's block).

4. **Engines table in new file could cross-reference stack presets** — The engines table lists `metaengine/sqliteengine/` etc. but doesn't mention that `stack/sqlite`, `stack/duckdb`, etc. are the consumer-facing presets. A reader might wonder which to import.

5. **No bidirectional link from Anti-Patterns** — The root file's Anti-Patterns section has two metaengine entries ("Manual query routing", "Hand-written DDL"). These should link to the new file's "Terms We Avoid" section for the expanded metaengine-specific anti-patterns. Currently they're duplicated without cross-linking.

6. **Replication section prose mentions NetworkRTT formula but Cost Model section is where the formula lives** — I added a cross-reference, but the Replication section repeats the concept ("additive fixed latency") in its own words. Could DRY this.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (correctness / completeness)

1. Add table of contents to root `DOMAIN_LANGUAGE.md` (640 lines, 23 headings, no ToC)
2. Add `METAENGINE_DOMAIN_LANGUAGE.md` to AGENTS.md project documentation table
3. Update the prior status report (`2026-08-09_02-07`) to mark its 3 questions as resolved
4. Add more `keycodec` symbols to the new verification block (only 1 of 20+ verified)
5. Add `projectionadapter.WithEventDecoder`, `EventWithID`, `NewTypeDecoder` to verification block
6. Add `(planned)` inline annotations to specific v2-vision terms (OnRecord is partial, tombstone-as-event is planned, command-lifecycle-as-events is planned)
7. Add CHANGELOG.md entry for the doc restructure
8. Verify cross-file anchor links actually render on GitHub (not just locally valid)

### Medium Priority (quality / navigation)

9. Cross-link root Anti-Patterns metaengine rows to new file's "Terms We Avoid" section
10. Add a "when to read this file" sentence to root metaengine stub
11. Cross-reference stack presets in the engines table (sqliteengine ↔ stack/sqlite)
12. Add Iroh transport sub-modules (`loopback`, `quic`) to the new file's verification block
13. Add `metaengine/bench` symbols to verification if any non-test exports exist (verify: none found this session, but double-check)
14. DRY the NetworkRTT description between Cost Model and Replication sections
15. Consider moving Deriver from Cross-Cutting to its own subsection or under CQRS
16. Add `keycodec` module to AGENTS.md module list (it's a sub-package, but it's documented now)
17. Add a "How to maintain this file" note to both domain language files (when to add terms to which file)
18. Audit the root file for any remaining metaengine symbols in prose that no longer have a verification backstop

### Lower Priority (polish)

19. Replace em dashes with commas/periods/parentheses throughout both domain language files (AGENTS.md style rule)
20. Add `WithEventDecoder` to the Projection Adapter table's verification
21. Add the `CalibrationCosts` type to the verification block (referenced in Calibratable description)
22. Add `PlanContext` type to verification block (referenced in interface hierarchy)
23. Add `RulePipeline.Run` method to verification block
24. Add `QueryAssignment` struct fields to verification block
25. Add `CollectionInfo` struct to verification block
26. Add `QueryDecl` and `QueryConfig` to verification block
27. Add `FoldKind` constants (`FoldInsert`, `FoldUpdate`, etc.) to verification block
28. Document the `loopback` and `quic` transport sub-modules in the new file's prose (they're mentioned in a blockquote but not in a table)
29. Add the `MemoryDriver` graph adapter to the Engines table description
30. Add `metaengine.Store.Explain(ctx)` method reference (currently only `ExplainPlan()` is documented)
31. Add `SerializablePlan` serialization format details (JSON structure)
32. Document `WithReplication` and `WithNetworkRTT` plan options (mentioned in AGENTS.md but not in domain language)
33. Add `WithStats` / workload stats option (future API mentioned in prose)
34. Consider whether `record.Record` should be in the new file's verification block (it's imported by OnRecord but currently only verified in root)

### Documentation infrastructure

35. Add a doc-check test that asserts both domain language files are in the auto-discovery list
36. Add a CI assertion that `docs/DOMAIN_LANGUAGE.md` and `docs/METAENGINE_DOMAIN_LANGUAGE.md` both exist
37. Consider a doc-check feature: warn when a markdown file references symbols but has no verification block
38. Consider adding `docs/DOMAIN_LANGUAGE.md` and `docs/METAENGINE_DOMAIN_LANGUAGE.md` to the CI workflow's explicit doc-check call (line 69 uses auto-discovery, which works but is implicit)
39. Add the new file to the SKILL.md or skill references if AI consumers should know about metaengine vocabulary
40. Audit all ADRs that reference `DOMAIN_LANGUAGE.md#metaengine` (the old anchor) — they now need to point to `METAENGINE_DOMAIN_LANGUAGE.md`

### Content gaps (terms missing from both files)

41. Add `metaengine.Store.LogPlan(logger)` to the domain language (DX helper, mentioned in AGENTS.md)
42. Add `metaengine.PlanFromMemory(queries...)` and `metaengine.PlanFromSQLite(dsn, queries...)` one-shot helpers
43. Add `metaengine.NewSQLiteEngineFromDSN(dsn)` DX helper
44. Add the SSE/Watcher integration terms (`WatchTyped`, `WatchTypedWithSeq`, `WithReplay`, `SSEReplay`)
45. Add `metaengine.DeferClose(c Closer)` helper (used across engine modules)
46. Document the `PrefetchCache` key matching logic (cursor-encoded, thread-safe via RWMutex)
47. Add `WithCursorString(s)` / `Cursor.Encode()` round-trip documentation
48. Document `metaengine.NewTypeDecoder(Register(...))` pattern
49. Add `enginetest.RunMatrix` — wait, this lives in `adttest`, not `enginetest`. Ensure the domain language reflects this correctly (it does in the Testing section, but verify no stale references exist)
50. Review whether the `graphadapter` engine should mention it wraps `graph.MemoryDriver` (ADR-0113)

---

## g) Questions (3)

### Q1: Should the root DOMAIN_LANGUAGE.md keep a minimal metaengine verification block?

The root file's stub mentions `metaengine.Plan` and `metaengine.Store` in prose, but the verification block no longer imports the metaengine package. If the new file is deleted, these references silently stop being checked (doc-check treats unimported package aliases as "external" and skips them). Options:
- **A)** Add back 2-3 metaengine symbols to the root verification block (redundant with new file but self-contained)
- **B)** Accept the coupling — the new file must exist for the root file's metaengine prose to be verified
- **C)** Remove metaengine symbol names from the root stub prose (use generic language only)

### Q2: Should em dashes be cleaned up in the domain language files?

The AGENTS.md rule says "Never use em dashes in source code." Both domain language files use `—` extensively. This was inherited from the original content. Should I do a cleanup pass replacing all em dashes with commas/periods/parentheses? It's a large find-replace across both files (100+ occurrences) and will produce a noisy diff, but it aligns with the stated style rule. Or is the rule intended for `.go` files only, with markdown docs being exempt?

### Q3: Should the metaengine vocabulary be split further?

The new file is 540 lines. The engines section alone documents 9 engines across 3 storage types. Should the engines get their own `docs/METAENGINE_ENGINES.md`? Or is 540 lines the right size — comprehensive but not overwhelming? My recommendation is to keep it as one file (the ToC makes navigation easy), but I want to confirm before the file grows further.
