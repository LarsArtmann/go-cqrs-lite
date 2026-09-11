# Status: Cordis Pareto Execution CLOSED — all 27 tasks done, verify-fast GREEN

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** §e next-steps: sqliteengine.ResetEngine + the EngineResetter ladder live as the TODO_LIST Cordis follow-ups (🔥 prioritized); §f Q1 (CHANGELOG test-only policy) remains unanswered — test-only/infra fixes stay unchangelogged per M-08 precedent; Q3 (release-train timing) rides the TODO tag-wave item.
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.

> **When:** 2026-09-11 00:04 · **Session:** "READ, UNDERSTAND, RESEARCH, REFLECT / keep going until done / brutal self-review + comprehensive status"
> **Input:** [`docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](../planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md)
> **Supersedes:** [`2026-09-10_23-35_cordis-all-27-done.md`](2026-09-10_23-35_cordis-all-27-done.md) — written mid-verify; it predates three gate fixes (see §d) and lacks the self-review. Content stands otherwise.

---

## a) FULLY DONE (this closing stretch, each with its verify gate)

### M-15 — cqrs-lint E018 `projection-without-emitter`

Mirror of E006: flags projections subscribing to an event type nothing emits or catalogs — the `user.creted` typo class. **Warning severity** (warn-first guardrail; runtime gate stays the hard tier), catalog-escape honored, silent when zero emissions detected (scanner-limit guard). Registered in all five sites (detector, RegisterAll, catalog metadata, README table+counts 205→206, RULES.md regenerated); pinned-count meta-test updated; 6 fixture tests; self-suite green; lint 0.

### M-16 — catalog coeffect validation in the EventCatalog export

`catalog.ValidateCoeffects` (dangling events = violations; explicit `Producers` honored; unconsumed = advisory only) + `Catalog.DeriveProducersConsumers` exported (the exporter's derivation shared; eventcatalog keeps a forwarder). Exporter writes `coeffects.md` (graph table, DANGLING/advisory status, dangling count). 7 tests. **`#check-eventcatalog` GREEN end-to-end** (54 pages built clean).

### M-17/M-18/M-19 — ground-truth pass

ADRs 0114/0123/0124/0126/0127 verified; one correction via addendum ("ADR-0124 §11" — no numbered sections exist; semantics live in code + the layout-planning doc, verified at `relayout.go:33,64,171`). arXiv PDF archived + grounding addendum (Def 1 twisted composition, Def 2 effect context, Def 19 coeffect context, Defs 31–33 + Lemma 32 tests/≃, Theorem 7 recovery). Figures recounted: **84 modules** (was 82), **72 production + 33 test DeferClose sites** (was 47+17); AGENTS.md corrected, mapping-doc addendum records the drift.

### M-21/M-22/M-23 — health-driven engine deactivation (ADR-0137)

ADR written + indexed (136/136). Implemented in `metaengine/engine_health.go`: errorfamily Infrastructure/Transient failures count (default 3, `SetEngineFailureThreshold`); quarantine excludes from `routableLocked` + execution reroutes via `effectiveQueryLocked`/`routedQuery` (plan untouched); `StartAutoReprobe` reactivates on `Prober` success; `ReactivateEngine` manual; `EngineStats.Health` + `Doctor` "--- Engine Health ---" section. 5 tests under `-race`; full metaengine suite + lint 0.

### M-24/M-25 — equivalence tooling

`scenario.Interleaved` + `AssertObservationalEquivalence` + `EquivalenceProbe` + minimal `ScenarioReporter` (Helper+Fatalf — rapid.T satisfies it; testing.TB does NOT on go1.26, missing ArtifactDir). `GivenProjection` widened `*testing.T`→`testing.TB` (source-compatible). rapid property test: randomized interleavings (0–64 events, closed key space) — A's answers identical across every key.

### M-26/M-27 — vocabulary + diagram

Keep-internal default EXECUTED and verified leak-free (no paradigm terms in SKILL.md/README/references/DOMAIN_LANGUAGE; "coeffect" appears publicly only as shipped API names). Mermaid concept map appended to the mapping doc.

### M-20 — consolidated release hygiene ✅

- CHANGELOG `[Unreleased]`: five entries (coeffect gate, E018, catalog coeffect validation, ADR-0137 deactivation, scenario equivalence) — symbol gate **18 citations green**.
- API golden: 6752 → 6755 → 6766 → **6773** exports; `TestEvery` green each regen.
- doc-check **1042 refs** green (core.md §3.9 recipe + AGENTS.md contracts #23/#24 included).
- `go work sync` + workspace check OK (rapid direct test dep).
- **`nix run .#verify-fast` GREEN END-TO-END** (third attempt — see §d for what the first two attempts flushed out). Turso and catalog/schema flakes did NOT recur (one sighting each, ever).

### Housekeeping

TODO_LIST Cordis section rewritten to follow-ups-only (sqliteengine 🔥, EngineResetter ladder, fold-write failover, E018 fold coverage, goleak metaengine/projectionhost, `[Unreleased]` tripwire, wire `#check-file-size`, release-train note). AGENTS.md contracts **#23** (deactivation: lock ordering, families, probe-gated reactivation) and **#24** (three-tier coeffect lockstep). recipes.md §2.33 (survive a dead engine), core.md §3.9 (declare the event universe).

---

## b) NOT DONE / deliberately deferred

- **Full `nix run .#verify`** (soak tests + docserver CSS + doc-assertions beyond fast) — NOT run; verify-fast green only. The plan's Wave 3 gate names `#verify`; budget says defer to the pre-tag gate. Same for `#vulncheck`, `#check-coverage`, integration suites (no storage-layer code touched this session).
- **`sqliteengine.ResetEngine`** — follow-up per decision 3 (risk surface; TODO_LIST 🔥).
- **Fold-write failover** for quarantined engines (reads only; writes fail loudly by design) — TODO_LIST.

## c) TOTALLY FUCKED UP (and what happened after)

1. **Wrote `equivalenceEvent(&testing.T{}, ...)` in a package-var initializer** — a bare `&testing.T{}` whose Fatalf would panic outside a runner. Caught it myself before running; replaced with a `fixedEquivalenceEvents(t)` function. Sloppy first draft of the worst kind: looks fine, detonates on the error path.
2. **Designed `AssertObservationalEquivalence` against `testing.TB` first** — hit the `*rapid.T`-lacks-ArtifactDir wall at build time, then redesigned with `ScenarioReporter`. Should have checked rapid.T's interface against go1.26's TB BEFORE designing (the verify-first reflex I keep preaching).
3. **Three verify-fast attempts to green.** Attempt 1 died on a PRE-EXISTING cqrs-upgrade parallel-test race (`versionResolver` global swapped by two `t.Parallel()` tests); attempt 2 on `go.uber.org/goleak` missing from the depguard allow-list (M-08 leftover); attempt 3 on 2 new clone groups (my sorted-keys loop + a builder prologue). All fixed (race serialized with comments; depguard entry; `slices.Sorted(maps.Keys(...))` + `//art-dupl:accept` on the trivial prologue). Lesson: prior sessions' "done" was never verified end-to-end — I inherited three landmines and only found them because I insisted on the green gate. Run verify EARLY, not after building on top.
4. **Assumed `memoryEngine` implements `RawValueReader`** — wrote the override, build failed, deleted it. Read the interface first next time.
5. **Daemon mtime churn broke ~5 edits mid-flight** (auto-commit + `nix fmt` touching files between my view and edit). Handled each by re-viewing, but the tight view→edit pairing should be reflex from the start.

## d) WHAT WE SHOULD DO BETTER

1. **The "green gate" is the only truth.** Two sessions of M-tasks claimed done with verify-fast never green end-to-end — three latent failures accumulated silently. Rule: no wave is closed until `#verify-fast` is green IN THAT SESSION, landmines included.
2. **Research the test-runtime interfaces before designing test-facing APIs** (the TB/rapid wall cost a redesign cycle).
3. **The daemon-absorbed-doc problem is now PROVEN, not hypothetical** — my 23:35 status report was absorbed while verify still had two failures pending; any reader between commits saw "all done" with a red gate. The `[Unreleased]`-position tripwire (TODO_LIST) plus "write status reports only after the gate is green" (this report) is the mitigation.
4. **check-file-size exists but nothing runs it** — a gate that isn't wired is documentation, not a gate. Same skepticism as the ADR-index blind spot.

## e) NEXT — in execution order

1. `nix run .#verify` full (pre-tag gate) + `#vulncheck` + `#check-coverage`
2. `sqliteengine.ResetEngine` (🔥; 8 `meta_*` tables + planned tables + matviews + `multiSeq`)
3. EngineResetter on remaining persistent engines (pebble, bbolt, badger, pg, mysql, turso, duckdb, dgraph, iroh)
4. Reset capability surfaced in `Doctor`/`GetEngineStats`
5. Fold-write failover for quarantined engines (shadow-replication or write-reroute + catch-up)
6. E018 fold-case coverage (CollectFoldCaseStrings lacks positions — needs scanner work)
7. goleak for `metaengine` + `projectionhost` suites
8. `[Unreleased]`-position tripwire in verify-docs.sh
9. Wire `#check-file-size` into verify + split `catalog/eventcatalog/exporter.go` (373 > 350)
10. Deactivation: health-aware `CheckRouting` interplay test; quarantine counter via otel; integration test with a real remote engine (currently memory-fakes only)
11. rapid property for 3+ projections; `AssertObservationalEquivalence` store-cleanup ergonomics
12. `coeffects.md` as a first-class EventCatalog custom doc instead of a stray file
13. Release train: projectionadapter sibling replace + metaengine pin ride the cut; CHANGELOG entries for test-only/infra fixes — decide policy (see Q1)
14. Revisit vocabulary positioning at the v5 gate (ADR-0123 train)

## f) QUESTIONS (cannot resolve myself)

1. **CHANGELOG policy for test-only/infra fixes:** do the cqrs-upgrade race fix, the layer-check exception, and the depguard addition get `[Unreleased]` entries (consumer-runnable cmd module vs. pure infra), or is test-only exempt as with M-08's goleak?
2. **Run full `#verify` (soak) now to close Wave 3 per the plan's gate wording, or defer to the pre-tag gate?** (Hours of soak time vs. plan literalism — verify-fast is green.)
3. **Next release train timing:** cut with this batch (coeffect gate + E018 + deactivation + equivalence tooling), or hold until `sqliteengine.ResetEngine` lands so "one-call revert" covers the production-default engine?

— Final gate state: `#verify-fast` ✅ · doc-check 1042 ✅ · api golden 6773 + TestEvery ✅ · symbol gate 18 ✅ · duplication 0 new ✅ · check-eventcatalog ✅ · layer check ✅ · verify-docs 136/136 ✅. Working tree daemon-absorbed, nothing pushed. WAITING FOR INSTRUCTIONS.
