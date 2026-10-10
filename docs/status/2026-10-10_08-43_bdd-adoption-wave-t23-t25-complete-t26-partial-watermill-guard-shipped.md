# BDD Adoption Wave — T23–T25 COMPLETE, T26 ~60% (watermill guard SHIPPED), T27 pending

> **Date:** 2026-10-10 08:43 CEST · **Mode:** Full Execution (owner GO at ~08:00, defaults applied)
> **Supersedes:** [`2026-10-10_07-27_bdd-adoption-wave-t20-complete-t21-t22-done-t23-research.md`](2026-10-10_07-27_bdd-adoption-wave-t20-complete-t21-t22-done-t23-research.md)
> **Plan:** [`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md)

**Progress: T01–T25 COMPLETE (handoff numbering). T26 partial — watermill leg done, example migrations + cqrs-htmx sweep + feedback memo not started. T27 not started.**

## a) FULLY DONE (this session)

### T23 — Chaos + SSE (plan F22.1–F22.4)

- **`systemscenario/chaos.go`** (343 lines): `DelayedDriver(t, base, delay)` — the journal-latency
  chaos seam. Registers a process-unique driver (sequence-numbered name; the registry has no
  unregister hook) via `metaengine.RegisterDriver` wrapping ANY registered base driver.
  **Load-bearing design correction:** the prior handoff claimed "embedding `metaengine.Engine`
  promotes ALL capability interfaces" — that is FALSE in Go (type assertions do not tunnel
  through interface embedding; `Engine` is only `Profile()+Closer`). The wrapper hand-forwards
  exactly what system wiring discovers by assertion: StreamLogBackend (5 methods), AtomicAppender,
  Transactional, SeqSeekableStreamLog (2), EventByIDBackend, StreamTemporalReader (2) — all WITH
  delay; MapBackend + MapUpdater undelayed (read models stay fast — the chaos targets the journal
  path). Unsupported bases fail loudly on first use; vector/search/spatial/snapshot/timer caps
  intentionally dropped (stream-log chaos only).
- **`systemscenario/chaos_test.go`**: (1) `ReadModelMatchesUnderLatency` — a full scenario boots
  through 2ms journal latency (boot alone proves capability forwarding: system.New's fail-closed
  atomicity gate asserts AtomicAppender/Transactional on the WRAPPER); (2)
  `OptimisticConcurrencyHolds` — two concurrent Saves at the same expected version through the
  delayed wrapper's forwarded AtomicAppender: exactly 1 of 2 wins.
- **`systemscenario/sse.go`** (189 lines): `SubscribeSSE[V](t, ctx, sc, collection)` +
  `SSEStream[V].Await/.Received` — serves the projection collection via `metaengine.ServeSSE` on
  an in-process httptest server and connects as the first client with a spec-correct SSE parser
  (data-line joining, single-space strip, heartbeat-comment skipping). An Await passing pins the
  same wire path a browser EventSource consumes.
- **`systemscenario/sse_test.go`**: `TaskViewStreamsOverHTTP` — subscribe before Given, create +
  rename, ThenQuery confirms the projection, Await confirms the streamed rename view, and both
  folds streamed (create arrived before rename).
- Gates: dep budget `systemscenario` 8→9 (chaos.go/sse.go made metaengine a PRODUCTION import —
  was test-only; rationale in check-module-layers.sh); README "Chaos and SSE" section; recipes.md
  §2.45 + 2 catalog entries; doc-check ✓ (1240 refs); recipes compile gate ✓; changelog-symbols ✓;
  api golden +21 exports; full suite race-green.

### T24 — cqrs-lint E020 advisory (plan F23)

- **E019 was TAKEN** by a concurrent wave (`data-product-without-contract`) → the rule is
  **E020 `handrolled-system-boot-in-test`** (Info, ConfidenceMedium): a test file calling
  `system.New` without importing systemscenario gets the harness suggestion.
  `cmd/cqrs-lint/pkg/rules/architecture/e020.go` + 5 fixture tests: fires on hand-rolled boot,
  silent when harness imported (Adopt consumers boot to hand over), silent in production files,
  detects aliased imports (`sys "…/system/v4"`), exact-path check keeps systemscenario and
  system/integration from tripping.
  **Implementation note:** fixture loads are syntax-only (empty TypesInfo) — the qualifier check
  needed an import-table fallback (alias or last path segment with the major-version suffix
  stripped) alongside typed exact-path resolution.
- Meta-tests: catalog entry, detector count 209→210, README headline 209→210 + architecture
  count 19→20. **Fixed on sight:** E019's README rules-table row was missing (the other wave's
  miss) — added both E019 and E020 rows.
- Api golden +`NewE020Detector`; CHANGELOG entry; symbols gate ✓; pkg/rules green (the only
  cqrs-lint red left is `pkg/rules/version` — external, see §e).

### T25 — cqrs-upgrade suggestion rule (plan F24)

- **`cmd/cqrs-upgrade/suggest.go`** (183 lines): `suggest:then-query` advisory matcher for the
  eventually idiom — a `for` loop whose condition mentions `time.Now` (deadline comparison) AND
  whose body calls `time.Sleep`. Both signals required, so retry/backoff loops stay silent.
  Message points at `systemscenario.ThenQuery`/`ThenQueryEventuallyFails`.
  **Never auto-rewrites**; `--strict` ignores suggestions entirely.
  **Key discovery:** cqrs-lint's `BuildContext` loads packages with `Tests: false` — test files
  never reach the analyzer. Widening that load would change every rule's inputs, so the matcher
  owns a syntax-only walk of `*_test.go` (vendor/testdata/.git skipped; unparseable files skipped
  silently — the deprecation scan stays the authoritative load gate).
- Wire contract: `--json` modules gain an always-present `suggestions` array; `schemaVersion`
  1→2 (bump documented in the wire comment). Pipeline: one BuildContext, two detectors
  (`scanFindingsAnalyzed`); human output renders the section always.
- `suggest_test.go` + `testdata/e2e/eventually` fixture: fires exactly once on the await loop,
  backoff loop silent, `--strict` exits clean on a suggestion-only module, wire carries the hint.
- Docs: modules.md cqrs-upgrade row extended (doc-check ✓ 1265 refs); CHANGELOG entry; symbols
  gate ✓; api golden regenerated (my module is `package main` — no export change; the +13 that
  landed are foreign, see §e).

### T26 (watermill leg only) — `ErrReentrantPublish` (plan F26.1–F26.4)

- **`event/deliveryctx.go`** NEW: `MarkInDelivery` / `ContextInDelivery` / `WithoutDeliveryMark`
  — a synchronous-bus-delivery marker carried on the handler context. **Goroutine-ID-free by
  design** (plan F26.1): it flows through synchronous handler chains and is cleared explicitly
  at sanctioned async escapes. It lives in `event/` because deriver (Tier 2) must strip it but
  cannot import watermill (Tier 4).
- **watermill EventBus AND CommandBus**: `dispatchLocal` marks the delivery context;
  `Publish` rejects a marked context with the new **`watermill.ErrReentrantPublish`**
  (Orchestration family) instead of hanging on the per-topic subscriber lock under
  `BlockPublishUntilSubscriberAck. CommandBus.Publish previously ignored ctx entirely (`_`) —
  now participates in the guard.
- **deriver.WithAsyncDispatch strips the mark** on its dispatch goroutine
  (`event.WithoutDeliveryMark(context.WithoutCancel(ctx))` — WithoutCancel keeps values, so the
  strip is explicit): the sanctioned escape publishes cleanly.
- **`watermill/reentrancy_test.go`** (race-clean triple): nested sync publish fails fast with the
  sentinel (5s watchdog so a regression hangs the TEST, not the suite); async escape with the
  mark cleared publishes cleanly; plain contexts never report in-delivery. Verified against the
  evidence-pack repro shape (`docs/evidence/2026-10-09_deriver-bus-deadlock.md` §1).
- Full suites race-green: **event, deriver, watermill, system, systemscenario**.
- **ADR-0154 addendum** written (mechanism + test pointer); CHANGELOG entry; api golden +5
  (`event.ContextInDelivery/MarkInDelivery/WithoutDeliveryMark`, `watermill.ErrReentrantPublish`,
  +1 foreign catalog export).
- Workspace note: deriver/watermill now depend on UNPUBLISHED event API → GOWORK=off standalone
  builds of those two fail until the next tag wave (expected per Q2 ruling; in-workspace green).

## b) PARTIALLY DONE

### T26 — fleet rollout (remaining ~40%)

- **Example migrations — NOT STARTED, research complete:**
  - `example/taskmanager`: `NewServer(DefaultConfig(), nil)` is the production facade wrapping
    system.New; `Server.Sys` is an exported field → `systemscenario.Adopt(t, ctx, srv.Sys)`
    composes cleanly. Surveyed flows: create → deriver auto-assign (durable queue, async by
    design) → start → complete → delete; the suite's `waitForView` polling loops are exactly
    what ThenQuery replaces. `TaskReader.Get` returns `(nil,false,nil)` on missing (not an
    error) → the delete assertion wants `ThenQueryFunc` polling for `found==false`, not
    `ThenQueryEventuallyFails`.
  - `example/goal-shaped-app`: `boot(t, ctx, configPath)` seam + `Domain()` — production-shaped.
    `main_test.go` boots `system.New` DIRECTLY (an E020 target). Migration decision made:
    Adopt over the app's own config, using ONLY systemscenario v4.0.0-tagged API (Adopt,
    Given/When/Then*) so `check-example-standalone.sh` stays green — the untagged presets
    (Memory/SQLite) would break standalone builds until the wave.
  - Both examples' go.mod need `systemscenario/v4 v4.0.0` added (published tag — safe).
- **cqrs-htmx awaitNotFound → ThenQueryEventuallyFails sweep: NOT STARTED.**
- **Feedback memo (F25.4): NOT STARTED** (friction notes are accumulating in §d instead).

## c) NOT STARTED

- **T27**: final `#verify` (itemize external reds), companion suites, plan addendum, gotchas
  retro, TODO_LIST strikes, CHANGELOG symbols + final golden, superseding report.

## d) TOTALLY FUCKED UP (nothing destructive — process misses, all recovered)

1. **Handoff design claim was wrong and I nearly built on it:** "embedding promotes ALL capability
   interfaces" would have produced a wrapper that fails system.New's atomicity gate at boot.
   Caught by verifying against Go semantics before writing code. Lesson for the retro: handoff
   "designs chosen" still get re-verified against the compiler.
2. **SSE parser off-by-one:** `line[len("data"):]` skips 4 chars, not 5 — payload arrived as
   `": {json}"`. Found via the test's decode-error diagnostics (which is exactly why Await
   surfaces stream errors), fixed to spec-correct parsing.
3. **Async-escape test subscription mismatch (TWICE):** a global python replace shadowed my
   targeted replace, so the outer publish's event type didn't match the handler subscription —
   "never completed" was a test bug, not a guard bug. Proved via a throwaway debug test that the
   machinery worked, then fixed the test (and trashed the debug file).
4. **sed mangled a var decl** (`successfulSavesatomic.Int32` — space eaten); fixed by edit.
5. **2 daemon-race edit failures** (recipes_catalog_meta2.go, check-module-layers.sh) + 1 partial
   multiedit — all recovered with asserted python replaces / single-edit retries. The
   edit-then-verify habit (immediate rg/build after every edit) caught each within one step.
6. **`.When().When()` chain doesn't exist** — second act is `.Command(...)`; caught by compile.

## e) EXTERNAL STATE (concurrent agents — not mine, itemized for T27)

- **File-size gate reds (4, all foreign):** metaengine/engine.go grew 700→712, reflect.go
  352→359, typed_reader_scan.go 367 (NEW offender), projectionhost/host.go 382→383 — the
  graph-native wave's in-flight growth.
- **cqrs-lint `pkg/rules/version` red:** TestV007_TablesCoverAllV5DeprecationMarkers fails on
  core/v5 markers (tombstone/metadata/query symbols) — the core/v5 wave's drift, untouched.
- **E019 README row was missing** — foreign miss, fixed on sight (both E019+E020 rows added).
- **Api golden absorbed +13 foreign exports** (schema EventOf/TypedEventSchema etc., system
  SchemaSet/Schemas — the schema-declaration wave) alongside my +26 across the session.
- Another wave's status report appeared at 07:37 (`graph-native-wave-execution-status.md`).
- Tree was CLEAN at session start except that foreign untracked report; all my work is
  daemon-absorbed (no authored commits), consistent with the standing mode.

## f) NEXT — up to 50 items (ordered)

1. **T26 example migration — taskmanager** (F25.1): add `systemscenario_test.go` adopting
   `NewServer(...).Sys`; go.mod += `systemscenario/v4 v4.0.0`; convert create→assign→start→
   complete→delete flows to Given/When/ThenQuery (+ThenQueryFunc for the delete-vanish);
   suite green + `check-example-standalone.sh --build`.
2. **T26 example migration — goal-shaped-app** (F25.2): same shape over `boot()`; also flip
   main_test's raw boot comments where Adopt fits; green + standalone gate.
3. **T26 cqrs-htmx sweep** (F25.3 sibling repo): `awaitNotFound` blocks →
   `ThenQueryEventuallyFails`; run its suite.
4. **T26 feedback memo** (F25.4): write `docs/planning/...` or skill-reference addendum with the
   friction found (sse parser need, Adopt-over-facade ergonomics, TypedReader found=false shape,
   waitForView→ThenQuery conversion notes).
5. **T27.1** `nix run .#verify` — expect my-green/externals-red; itemize: core/v5 version rule,
   4 file-size offenders, README-link gate if foreign edits broke it.
6. **T27.2** companion suites: cqrs-htmx + go-appkit full tests (their trees were mid-flight by
   other agents at last look — check freshness first).
7. **T27.3** `nix run .#check-md-go` + gotchas retro: edit-then-verify under daemon races,
   handoff-claims-need-compiler-verification, fixture TypesInfo emptiness, BuildContext
   Tests:false, sed/python-replace shadowing, watcher.Close no-value, .When chain shape.
8. **T27.4** plan addendum: DONE/PARTIAL/NOT SHIPPED per task row; restate the 5-module wave
   (systemscenario + event + deriver + watermill + cqrs-lint/cqrs-upgrade tool modules);
   F20.4 partial-with-rationale note.
9. **T27.5** TODO_LIST strikes (deriver deadlock entry — now loud-fail + async cure shipped),
   CHANGELOG symbols gate, final api golden regen, final superseding status report.
10. (pending owner) Q1/Q2 below; ADR-0152 addendum only on approval (carried).

## g) QUESTIONS (cannot figure out myself)

1. **Example-migration shape (F25.1/F25.2):** I plan to ADD a systemscenario suite adopting each
   app's production boot (Server.Sys / boot()), leaving the HTTP integration test and the
   decider-pure scenario/v4 suites untouched — three testing tiers stay distinct, and using only
   v4.0.0-tagged harness API keeps `check-example-standalone.sh` green before the wave tags the
   new presets. Alternative: also REWRITE the existing suites' waitForView polling into harness
   assertions (bigger diff, one less idiom in the codebase). Proceed with ADD-only?
2. **cqrs-htmx sweep timing:** ThenQueryEventuallyFails is tagged (v4.0.0) so there is no wave
   dependency — but cqrs-htmx's tree was mid-flight with other agents during T20. Migrate now on
   top of the current tree, or sequence after its next tag-wave lands?
3. **Watermill guard strictness ruling (behavior change):** the guard rejects a nested publish on
   a marked context even from an EXTERNAL `go func(){ bus.Publish(handlerCtx, ...) }()` that
   inherited the delivery ctx without clearing the mark — previously that succeeded (different
   goroutine, no deadlock); it now fails fast with ErrReentrantPublish telling the author to
   detach. Loud-over-silent is the plan's spirit, but it converts working (if fragile) consumer
   code into an error. Ship as-is (documented), or restrict detection to same-topic nesting?

---

**Stop point:** waiting on owner. Defaults if told "continue": (1) ADD-only example suites,
(2) migrate cqrs-htmx now, (3) ship the guard as-is. Then T27.
