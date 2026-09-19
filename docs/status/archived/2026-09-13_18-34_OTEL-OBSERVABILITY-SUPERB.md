# Status Report — OTEL-OBSERVABILITY SUPERB (2026-09-13 18:34 CEST)

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped via later sessions (TODO_LIST `[x]` rows + CHANGELOG `[Unreleased]` dated entries). Unstruck items remain OPEN, tracked in TODO_LIST/ROADMAP where actionable (tag waves, quiet-window `#verify`, billing-gated CI, owner [BLOCKED] rulings); XS polish wishes not yet harvested stay here as the historical record. ARCHIVED.

> Scope: this session's OTel observability wave (plan:
> `docs/planning/2026-09-13_14-31_SUPERB-OTEL-OBSERVABILITY.md`).
> Everything below is this session's work unless marked otherwise.

## a) FULLY DONE ✅

| #               | Deliverable                                                                                                                                                                                                                                                                                                    | Evidence                                                                                                                                                                                                                                           |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1              | **metaengine health hooks** — `Hooks.OnQuarantined/OnReactivated/OnProbe/OnCatchUp` (ADR-0137 transitions as typed callbacks), emitted **outside the health mutex**; re-entrancy proven by test (callback → `HealthSnapshot()` cannot deadlock); `Hooks.Merge` + `Store.CurrentHooks` for composable hook sets | `metaengine/health_observer.go`, `metaengine/hooks.go`, `metaengine/health_observer_test.go`, `metaengine/hooks_test.go` — full module suite green incl. `-race` (35.7s); `engine_health.go` SHRANK 393→385 lines (ratchet-safe)                   |
| M2+M3           | **`metaengine/otelobserver` (new module)** — `Attach(store, meter)` merges into existing hooks and records `cqrs.metaengine.{quarantine,reactivate,probe,catchup}.total{engine,reason,outcome}` + `catchup.replayed{engine}`                                                                                   | End-to-end external test with manual metric reader drives a real flaky-engine quarantine → catch-up → reprobe lifecycle and asserts every counter; module green with `-race`                                                                       |
| M4              | **`ClaimMetricsSnapshot.StartedAt`** — process-start anchor for cross-restart claim rates; JSON wire shape re-pinned                                                                                                                                                                                           | `scheduling/sqlstore/claim_metrics.go`; module suite green. Recorder deliberately NOT in-module (documented lean-budget design) — moved to the example                                                                                             |
| M5              | **`otel/otlp` (new module)** — `SetupOTLP(ctx, OTLPConfig)`: one-call OTLP/HTTP trace+metric export layered on `otel.Setup`; no gRPC dep; trailing options override                                                                                                                                            | Tests: httptest collector receives `/v1/traces` + `/v1/metrics` with auth headers; reader-override test proves trailing options win. Green with `-race`                                                                                            |
| M6              | **`example/scheduler-otel-status` (new runnable example)** — ClaimMetrics hooks → OTel counters on `/metrics` (Prom bridge), snapshot + live `claimedPerMinute` on `/status`                                                                                                                                   | **Smoke-verified live**: status JSON + `cqrs_scheduler_claim_*_total` scraped from a running server. (Fun fact: :8080 was occupied by a local SigNoz — the observability wave was blocked BY an observability platform; example now honors `ADDR`) |
| M7              | **`otel.DBSystem`** — `db.system` OTel semconv on all pebble + bbolt span helpers                                                                                                                                                                                                                              | Both modules green; sibling replaces added (documented unpublished-symbol pattern)                                                                                                                                                                 |
| M8              | **Exemplars verified ON by default** — SDK 1.46 ships `TraceBasedFilter`; my review claim "exemplars off" was WRONG. Proof test pins exemplars carrying trace/span IDs                                                                                                                                         | `otel/exemplar_test.go` green; no API change needed                                                                                                                                                                                                |
| M9              | **Docs** — recipes.md §2.8 (OTLP+exemplars) + §2.35 (health hooks + otelobserver), modules.md rows, SPAN_NAMING.md db.system note, 3 new READMEs, otel/README exemplars section                                                                                                                                | `doc-check`: 1127 references valid, exit 0                                                                                                                                                                                                         |
| M10             | **Meta docs** — CHANGELOG `[Unreleased]` (3 sections, 56 symbol citations verified), FEATURES.md (3 new feature tables + module rows), TODO_LIST.md (closed the claim-metrics example items, opened SQL db.system follow-up), module-map, AGENTS.md (88 go.mod, tier tree)                                     | `check-changelog-symbols`: 56/56 verified, 0 fiction                                                                                                                                                                                               |
| Gates (partial) | `nix fmt` clean · api golden regen'd (6856 exports) · all api-stability meta-tests green · `check-arch` green · layer smoke tests green · **`verify-ci` FULLY GREEN** (GOWORK=off per-module build+test across the whole matrix, incl. system)                                                                 | see M11 for the one gate that isn't                                                                                                                                                                                                                |

## b) PARTIALLY DONE ⚠️

1. **M11 — `#verify` (full gate): blocked by the MACHINE, not the code.** The full
   suite run failed on `TestCLI_WarmupFlag` / `TestCLI_Compare` /
   `TestLayoutCommand_AllLayouts` — all three with
   `ld.bfd: final link failed: No space left on device` while linking the
   duckdb CGo static binary (multi-GB link on a 90%-full tmpfs `/tmp`), plus
   `TestSystem_ResetProjection_RestartAndReplay` ("processed=0") which
   **passes solo (0.126s)** — disk-pressure contamination of the same run.
   `verify-ci` (the same modules, GOWORK=off) is fully green, and every
   touched module passed targeted `-race` runs. `/tmp` sits at 43G/48G from
   accumulated cross-tool artifacts (gexec_artifacts ≈ 5×400MB, playwright,
   bunx/pnpm caches, nix-shell dirs) that are not mine to delete.
2. **M12 — commits + push: commits absorbed by the auto-commit daemon**
   (content is all committed); **push NOT done** — user instruction switched
   to "write status, then WAIT" before the push step.

## c) NOT STARTED ⏸️

- **storage/sql dialect-aware `db.system`** — needs Dialect threaded into
  package-level span helpers (~10 call sites); routed to TODO_LIST (S/M).
- **scheduling hardening tail remainder** — race-stress, counter-scope pin,
  property test, `decodeDueTimer` fuzz, RenewLease tokens (TODO_LIST).
- **Per-view IVM write-amp otel counter** — matview v2 territory (TODO_LIST).
- **v4 tag wave** including `otel/v4.5.0`, `otel/otlp/v4.0.0`,
  `metaengine/v4.14.0`, `metaengine/otelobserver/v4.0.0`,
  `scheduling/sqlstore/v4.5.0`, `storage/pebble`+`bbolt` re-tags (they
  depend on the unpublished `otel.DBSystem` until then) — the sibling
  replaces are stripped by `tag-release.sh` at cut time.

## d) TOTALLY FUCKED UP! 💥

- **Nothing in the shipped code.** Every gate that could run, passed.
- **Environment: `/tmp` chronically at 90%** — full `#verify` cannot pass on
  this machine while duckdb links + other tools' caches compete for tmpfs.
  This will bite every future full verify until /tmp is relieved or TMPDIR
  is moved to `/` (270G free).
- Process fumbles this session (both caught by gates, both fixed):
  inserted a `DEP_BUDGET` line mid-`LAYER` block in
  `check-module-layers.sh` (broke bash parsing → caught by `#check-arch`),
  and briefly clobbered the `otel/otlp` budget line while repairing it.
- Note: a parallel session's docs work (`event-query-model.md`, metaengine
  README rewrite) landed interleaved with mine via the daemon — inspected,
  not mine, untouched.

## e) WHAT WE SHOULD IMPROVE 🔧

1. **Module-add procedure is under-documented by 3 wiring points.** AGENTS
   lists 5 steps; reality needed 8: go.work, flake testModules, api-stability
   modules slice, layer script `LAYER`+`DEP_BUDGET`, cqrs-lint module-catalog
   exclusion, api-stability test exclusions (examples), examplePaths (for
   examples), golden regen. Three separate meta-tests each discover one
   missing piece — a single `NewModule` checklist (or one aggregated
   meta-test) would kill a full fix-cycle.
2. **My exemplars review claim was wrong** — I reported "exemplars off"
   without reading the SDK default (`TraceBasedFilter`). The verify-first
   task caught it; lesson encoded as a proof test. Reviews should verify
   SDK defaults from source, not from memory.
3. **Example go.mod brittleness**: `claiming/` has no published tag, so any
   consumer of `scheduling/sqlstore` needs the unpublished-sibling replace
   dance. Tagging `claiming/v4.0.0` would remove a whole class of friction.
4. **`/tmp` hygiene**: `#verify` should set `TMPDIR` to a location with real
   disk (flake env change), or the box needs a tmpfiles.d cleanup for
   `gexec_artifacts*`/`go-build*`.
5. **`Hooks.Merge` exists now, but `WithMetrics`/`WithTracing` still clobber
   existing hooks** (pre-existing wart, visible now that composition
   matters) — migrate them to Merge internally at v4.x.

## f) TOP #25 NEXT (by impact)

| #  | Task                                                                                                                      | Impact                          | Effort           |
| -- | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ---------------- |
| 1  | Relieve `/tmp` (clear gexec/go-build artifacts or point TMPDIR at `/`) and re-run `#verify` to green                      | Unblocks the only red gate      | XS/S             |
|~~ 2  ~~|~~ `git push` master (user go/no-go)                                                                                         ~~ done 2026-09-13 — 18-35 §a26 (pushed, in sync)  |~~ Ships the wave                  ~~|~~ XS               ~~|
| 3  | Cut the v4 tag wave (6 modules above) via `tag-release.sh` flow                                                           | Consumers get the features      | M                |
| 4  | Tag `claiming/v4.0.0` (unblocks example/sqlstore consumers)                                                               | Removes replace friction        | S                |
|~~ 5  ~~|~~ storage/sql dialect-aware `db.system` spans                                                                               ~~ done 2026-09-15 — CHANGELOG storage entry  |~~ Semconv completeness            ~~|~~ S/M              ~~|
| 6  | Migrate `WithMetrics`/`WithTracing` onto `Hooks.Merge` (stop clobbering)                                                  | Composability correctness       | S                |
| 7  | otelobserver: expose gauge-style "currently quarantined" (up-down counter) derived from hooks                             | Operator dashboards love gauges | S                |
|~~ 8  ~~|~~ Add `metaengine` health hooks to `system.New` wiring docs/recipes (operator quickstart)                                   ~~ done 2026-09-13 — recipes §2.35 (M9)  |~~ Adoption                        ~~|~~ S                ~~|
| 9  | `example/scheduler-otel-status`: add a smoke test hitting /status + /metrics via httptest                                 | Example stays runnable          | S                |
| 10 | cqrs-lint rule F030+: suggest `otelobserver.Attach` when metaengine + otel are both imported                              | Adoption coaching               | M                |
| 11 | Aggregated module-add meta-test (single source of the 8 wiring points)                                                    | Dev-velocity, kills fix-cycles  | M                |
|~~ 12 ~~|~~ Scheduling hardening tail: race-stress + counter-scope pin tests                                                          ~~ done 2026-09-13 — CHANGELOG tests entry  |~~ Reliability                     ~~|~~ S                ~~|
|~~ 13 ~~|~~ `decodeDueTimer` fuzz                                                                                                     ~~ done 2026-09-13/16 — TODO_LIST row (shipped 2026-09-13..16)  |~~ Robustness                      ~~|~~ S                ~~|
| 14 | RenewLease ownership/claim tokens                                                                                         | Correctness                     | M                |
| 15 | Matview v2 per-view IVM write-amp otel counter                                                                            | Operator insight                | M                |
| 16 | Turso grouped-view upstream issue (blocked on user approval — pre-existing)                                               | Upstream fix                    | XS once approved |
| 17 | `system`/`stack` layers: consider span coverage at composition layer (currently 0 spans)                                  | Trace completeness              | M (decide first) |
| 18 | otel/otlp: add gRPC variant module (`otel/otlpgrpc`)? only if asked — dep-heavy                                           | Convenience                     | M                |
| 19 | Doc: AGENTS "Add a New Module" procedure update (8 steps)                                                                 | Docs truth                      | XS               |
| 20 | api golden: add meta-test that CHANGELOG-cited symbols exist (currently only script)                                      | Gate parity                     | XS               |
| 21 | pebble/bbolt: drop sibling replaces after otel/v4.5.0 tag lands                                                           | Hygiene                         | XS               |
| 22 | `Hooks` doc: cross-link SPAN_NAMING.md from otelobserver README                                                           | Discoverability                 | XS               |
| 23 | Coverage drift check after wave (`#check-coverage`)                                                                       | Gate                            | S                |
| 24 | `#vulncheck` per-module standalone build (new modules enter the graph)                                                    | Release gate                    | S                |
| 25 | Bench-regression gate (`benchmark-regression.sh`) — hooks added to hot-adjacent paths (execute records failures; measure) | Perf proof                      | S                |

## g) MY TOP QUESTION 🤔

**Should `/tmp` relief be a machine-level fix (clear ~15G of other tools'
artifacts — your call what's safe) or a repo-level fix (point `#verify`'s
TMPDIR at `/` in flake.nix, which I can do autonomaneously but changes build
behavior for everyone)?** This is the only thing between the wave and a fully
green `#verify`, and I cannot judge which artifacts on /tmp are yours vs
stale. (Secondary, simpler: say "push" and M12 completes.)
