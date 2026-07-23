# Status Update — ADR Review & Cleanup

**Date:** 2026-07-23 18:21
**Session scope:** Read all 57 ADR files (0001–0057 + README), identified inconsistencies/bad decisions, wrote findings to review file, then executed full cleanup per user decisions.

---

## a) FULLY DONE

### Review phase
- ✅ Read all 58 ADR `.md` files (0001–0057 + README + 0058 that appeared mid-session)
- ✅ Verified key claims against actual Go code (pebble Journal impl, event production imports, metadata aliases, event.Bus existence, bus implementations, module count, codec time encoding)
- ✅ Wrote comprehensive findings to `docs/reviews/2026-07-23_adr-review.md`

### Cleanup phase — Section A (README index)
- ✅ Fixed 8 status mismatches (0004 Superseded, 0011 Proposed, 0012 Proposed, 0017 Accepted, 0018 Accepted, 0031 Implemented, 0027 Deprecated, 0028 status note)
- ✅ Added ADRs 0055, 0056, 0057, 0058, 0059 to the index (were missing entirely)
- ✅ Filled in dates for 20+ ADRs that showed "—"
- ✅ Fixed ADR-0001 date (2026-04-29 → 2026-05-03)
- ✅ Marked ADR-0003 module count as historical (9 → 55)

### Cleanup phase — Section B (Contradictions)
- ✅ **B1:** Fixed ADR-0001 false claim that `aggregate/` package still exists (it was removed in Session 99); added cross-ref to ADR-0058
- ✅ **B2:** Added supersede banner to ADR-0009 (pebble "never" Journal — overturned by ADR-0019)
- ✅ **B3+B4:** Added supersede banner to ADR-0014 (eventtest extraction deferred → done by ADR-0045); removed stale `samber/ro` from dep list
- ✅ **B5:** Added supersede banner to ADR-0015 (JSON default → flipped to CBOR by ADR-0051/0053)
- ✅ **B6:** Added cross-references between ADR-0020 (pre-compute chains) and ADR-0049 (rebuild per dispatch) documenting why they differ
- ✅ **B7:** (Covered by E1 — ADR-0027 loud deprecation)
- ✅ **B8:** Added status note to ADR-0028 — v3 ghost bus removal never executed
- ✅ **B9:** Added cross-reference to ADR-0030 — projection/ name reused by ADR-0037
- ✅ **B10:** Added status note to ADR-0031 — aliases still exist, repointed to `metadata.CustomData`
- ✅ **B11:** Fixed ADR-0056 `TimeUnix` misstatement → corrected to `TimeUnixDynamic`

### Cleanup phase — Section C (Stale references)
- ✅ Fixed all `example/todo/` → `example/taskmanager/` references (ADRs 0004, 0009, 0016)
- ✅ Fixed ADR-0018 `projection.Runner` → `projectionhost.Host`
- ✅ Removed arena allocation from ADR-0026 (confirmed removed in AGENTS.md)
- ✅ Fixed ADR-0026 WASM module count (7 → 6, added honesty note)
- ✅ Updated ADR-0040 "Implementation plan (deferred)" → "Implementation" (module is built)

### Cleanup phase — Section D (Broken links)
- ✅ Fixed ADR-0052 broken link: `0044-self-describing-blind-stores.md` → `0044-blind-store-encoding-stamps.md`

### Cleanup phase — Section E (Questionable decisions)
- ✅ **E1:** Added loud deprecation banner to ADR-0027 (PG LISTEN/NOTIFY bus)
- ✅ **E2:** Renamed ADR-0046 "Four-Tier" → "Seven-Tier" (file rename via `git mv`, title update, all reference updates across AGENTS.md, CONTRIBUTING.md, docs/README.md, docs/planning/, docs/architecture-understanding/)
- ✅ **E4:** Rewrote ADR-0010 Decision section to match reality (io.Closer retained, no Lifecycle type), preserved original proposal in "Original proposal (superseded)" subsection
- ✅ **E5:** Trimmed TypeDB framing from ADR-0040 — reduced from a full comparison table + 3 sections to a one-paragraph "Design inspiration" reference
- ✅ **E6:** Made ADR-0026 honest about WASM (6 modules not 7, core-only not full-stack, "best-effort target")
- ✅ **E7:** Added "Consumer Burden" section to ADR-0043 documenting the two-DLQ-API pain; drafted ADR-0059 (Proposed) with a narrow unification path (Option D bridge)

### Cleanup phase — Section F (Cosmetic)
- ✅ Fixed ADR-0002 sentinel error count ("~20" → "hundreds")
- ✅ Fixed ADR-0003 module count (9 → 55)
- ✅ Fixed ADR-0046 codec dependency count ("38 of 48" annotated with current count note)

### Meta
- ✅ Updated `docs/reviews/2026-07-23_adr-review.md` priorities table — all 11 items marked Done

---

## b) PARTIALLY DONE

### ADR-0046 reference sweep — incomplete
The filename rename (`0046-four-tier-model.md` → `0046-seven-tier-model.md`) was propagated to: AGENTS.md, CONTRIBUTING.md, docs/README.md, docs/planning/storage-domain-separation.md. However, several historical/status files still reference the old name:
- `CHANGELOG.md` (2 references to "Four-Tier Model")
- `docs/status/2026-07-12_16-25_post-v4-comprehensive-cleanup.md`
- `docs/feedback/2026-07-10_swettyswipper_deep-adoption-review.md` (2 references)
- `docs/planning/2026-07-12_14-18_POST-V4-COMPREHENSIVE-PLAN.md`
- `docs/status/archive/2026-07-09_07-41_PARETO-EXECUTION-COMPLETE.md`

These are **historical point-in-time documents** that accurately described the ADR as "Four-Tier" at the time of writing. Rewriting them would falsify history. A judgment call is needed (see Section g).

### `FOUR-TIER-MODEL.md` filename not renamed
The H1 title was updated to "Seven-Tier Model" but the file remains at `docs/architecture-understanding/FOUR-TIER-MODEL.md` (not renamed to `SEVEN-TIER-MODEL.md`). This was a deliberate tradeoff to avoid breaking existing links, but it's a name mismatch.

---

## c) NOT STARTED

1. **ADR-0045 line 55 inaccuracy:** Claims "All 53 modules build cleanly" — actual count is 55. Not touched (cosmetic, in an Accepted ADR).
2. **ADR-0012 + ADR-0011 status:** Both are "Proposed" in their files and now correctly marked Proposed in the README. But neither has been implemented or formally accepted/rejected. They're in limbo. No action taken — needs user decision.
3. **ADR-0006 README mini-ADR vs file duplication:** The README contains inline mini-ADRs for 0001–0006 that partially duplicate (and sometimes disagree with) the individual files. This duplication is a structural issue that wasn't addressed.
4. **AGENTS.md references to "AggregateID"** throughout the Key Patterns section — these are still `Aggregate*` not `Stream*`, but ADR-0058 is only Proposed, so this is correct until 0058 is accepted.

---

## d) TOTALLY FUCKED UP

Nothing. All edits verified present in the working tree. No data loss, no broken files, no incorrect content introduced.

**One process note:** During the session, concurrent commits by the user (commits `949b8323`, `59d7df14`, `adfdefb4`, `c12a251e`, etc.) captured some of my doc edits as part of larger refactoring commits. This means my ADR edits are interleaved with unrelated code changes (aggregate→stream rename, OTel integration, listing improvements) in the git history. The edits are all present and correct, but the commit attribution is mixed.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements
1. **ADR lifecycle discipline:** The root cause of most inconsistencies is that ADRs are written once and never updated when superseded. A CI check (like `doc-check`) that verifies supersession chains would prevent this.
2. **README index automation:** The index table is manually maintained and drifted significantly. It should be auto-generated from the individual ADR files' Status/Date headers.
3. **"Never" claims in ADRs:** ADR-0009's "will never implement Journal" was overturned in 3 weeks. ADR style guide should require hedged language ("deferred unless consumer need emerges").
4. **ADR review cadence:** This review found 11+ stale ADRs accumulated over ~3 months. Quarterly ADR freshness reviews should be a documented practice.

### Content improvements
5. **ADR-0028 ghost code:** The v3 bus removal is documented as done but never happened. This is the largest gap between docs and reality. Either execute the removal or update the ADR to reflect the deferral decision.
6. **ADR-0031 metadata aliases:** The alias-to-struct migration was declared done but only half-done (repointed, not converted). This needs either completion or honest documentation of the deferral.
7. **DLQ split (ADR-0043/0059):** ADR-0059 is a Proposed path forward. It needs a decision — adopt, reject, or defer.
8. **ADR-0011/0012 in limbo:** Two ADRs have been "Proposed" for 6+ weeks with no resolution. They should be accepted, declined, or withdrawn.

---

## f) Up to 50 Things to Get Done Next

### ADR cleanup (follow-up from this session)
1. **Decide on historical file references to "Four-Tier Model"** — annotate vs leave as historical
2. **Rename `FOUR-TIER-MODEL.md` filename to match new title** (or accept the mismatch)
3. **Resolve ADR-0011 (ErrDispatcherClosed)** — accept, decline, or withdraw
4. **Resolve ADR-0012 (Split catalog)** — accept, decline, or withdraw
5. **ADR-0058 (Aggregate→Stream rename)** — accept or reject; currently Proposed
6. **ADR-0059 (DLQ unification)** — accept, reject, or defer
7. **Add CI check for ADR supersession chains** — auto-detect when ADR-X supersedes ADR-Y but ADR-Y has no supersede note
8. **Auto-generate ADR README index** from individual file headers
9. **Fix ADR-0045 module count** ("53" → "55")

### ADR-0028 ghost code (major)
10. **Execute v3 bus removal** OR formally document why it's deferred
11. **Remove `event.Bus`, `event.Subscriber`, `event.Middleware`** if Watermill is the canonical path
12. **Remove `storage/pg_bus.go`** (deprecated by ADR-0027/0028)
13. **Remove `command/memory_bus.go`** if superseded by Watermill command bus
14. **Audit all consumers for bus usage** before removal

### ADR-0031 metadata aliases (moderate)
15. **Convert `command.Metadata` from alias to standalone struct** (finish the migration)
16. **Convert `query.Metadata` from alias to standalone struct**
17. **Update AGENTS.md metadata section** to match the actual shape
18. **Add deprecation notice** on `event.Metadata` usage in command/query contexts

### DLQ improvements (if ADR-0059 accepted)
19. **Add `Event event.Event` to `middleware.DeadLetterEntry`** (optional field)
20. **Add `Replay(handler)` to `middleware.MemoryDeadLetterStore`**
21. **Create `dlq.Summary` adapter type** for unified dashboarding
22. **Write integration test** proving both DLQ types work together

### Documentation health
23. **Sweep AGENTS.md** for any remaining stale references (module counts, API examples)
24. **Update `docs/DOMAIN_LANGUAGE.md`** if ADR-0058 is accepted (Aggregate→Stream vocabulary)
25. **Audit all `docs/status/` files** for references to ADR decisions that have since changed
26. **Consolidate README inline mini-ADRs** or remove them (they duplicate and drift from the individual files)
27. **Update CHANGELOG.md** with ADR-0046 rename note

### ADR quality (ongoing)
28. **Add "Last reviewed" date field** to ADR template
29. **Create ADR template** with required sections (Status, Date, Superseded-by, Context, Decision, Consequences)
30. **Audit ADR-0006's inline tombstone section** — it duplicates ADR-0005 content
31. **Verify ADR-0019 key layout claims** against actual pebble key prefixes (done partially in this session)
32. **Check ADR-0033 multi-DB routing claims** against current stack preset code
33. **Verify ADR-0044 envelope format** matches actual `WrapEncode`/`UnwrapDecode` implementation

### Broader codebase (noticed during ADR review)
34. **`samber/ro` dependency** — ADR-0014 listed it as an event production dep but it's gone. Verify it wasn't supposed to be there (or was removed intentionally).
35. **Pebble `cqrs_journal:` index** — ADR-0009 said it would never exist, ADR-0019 added it. Verify the performance tradeoff (double writes) was actually measured.
36. **Module count tracking** — 55 modules and growing. Consider a `MODULES.md` auto-generated from `go.mod` files.
37. **Event `Bus` interface fate** — if it stays, document why Watermill didn't fully replace it. If it goes, create a removal plan.

---

## g) Questions I CANNOT Figure Out Myself

### 1. Historical documents referencing "Four-Tier Model" — annotate or leave?

Several point-in-time status reports, feedback reviews, and planning docs still say "Four-Tier Model." These were accurate at time of writing. Per the `update-old-docs` skill philosophy, should I:
- (a) Leave them untouched (they're historical snapshots, accuracy is temporal)
- (b) Add a brief inline annotation like `*(renamed to Seven-Tier Model in ADR-0046, 2026-07-23)*`
- (c) Do nothing — the ADR itself documents the rename

### 2. Should ADR-0011 and ADR-0012 be resolved?

Both have been "Proposed" for 6+ weeks:
- **ADR-0011** (Unify `ErrDispatcherClosed`) — proposes re-exporting a single sentinel from `dispatcher/` to `command/` and `query/`
- **ADR-0012** (Split catalog into sub-modules) — proposes breaking the 9K-line catalog module into 5 modules

These need a business decision: is there intent to implement them, or should they be formally declined?

### 3. The `FOUR-TIER-MODEL.md` filename — rename or keep?

The H1 title now says "Seven-Tier Model" but the file is still `docs/architecture-understanding/FOUR-TIER-MODEL.md`. Renaming would break any external links to this file. Keeping it creates a title/filename mismatch. What's your preference?
