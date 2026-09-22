# Status Report — v5 Train Execution: T26 Paste Items, Preflight GREEN, Verify Armed

**Date:** 2026-09-22 12:58 CEST
**Mandate:** the v5 Unification paste (sweep §4 rest · ADR-0139 · T18 tail ·
E-items · post-landing sweep · V5-MIGRATION-GUIDE · systemtest split ·
DeferClose note · cut v5.0.0) — READ/UNDERSTAND/RESEARCH/REFLECT, then
execute + verify step by step.

## a) DONE (verified green)

| #   | What                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | Evidence                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A1  | **Machine unblocked for `#verify`**: SC1091 fixed in the flake verify app (sourced verify-lock.sh tripped the strict shellcheck gate); inline `disable=SC1091` directive per repo convention; mechanism proven locally (bare → exit 1, directive → exit 0)                                                                                                                                                                                                                                                                                       | flake.nix:1686; local shellcheck A/B                                                                                                                        |
| A2  | **File-size ratchet violations resolved** (the 11:32 report's §g3): goal-closure lane growth extracted into three new files — `metaengine/planner_options.go` (planOption cluster, planner.go 579→470), `system/query_builders.go` (CRUD/counter builders, query_constructors.go 412→327), `cmd/cqrs-lint/pkg/rules/register_consumer.go` (consumer-coaching detectors, register.go 394→341)                                                                                                                                                     | `#check-file-size` ✓; module builds + tests ✓                                                                                                               |
| A3  | **Stale taskmanager lint golden repaired** — C017:1→gone (DeferClose fix landed by the prior session), C015:1 (idiomatic `r.Body.Close()`), C023 4→6, F031:1 (new rule) — failure proven PRE-EXISTING at the pre-session base commit 31e9780cc in a detached worktree before regenerating                                                                                                                                                                                                                                                        | full cqrs-lint suite green after                                                                                                                            |
| A4  | **Sweep §4 tail — pebble events migrated to stream vocabulary**: the census had a GAP (WIRE-FORMAT-KEYS.md had no pebble-events row; fresh pebble event envelopes still wrote `aggregate_id`/`aggregate_type`). Fresh rows now write `stream_id`/`stream_type` with a decode-only legacy shadow (`eventStreamKeysLegacy`, v6 marker) mirroring the pebble-command pattern; pinning test added; census gap closed in WIRE-FORMAT-KEYS.md (+ new encoding-stamp row)                                                                               | `TestDeserializeEvent_ReadsLegacyAggregateKeys` ✓; full pebble + bbolt suites ✓                                                                             |
| A5  | **E1 executed (design-corrected)**: event wire structs in pebble + bbolt stamp `codec.Encoding` (OPEN namespace — custom codecs like "protobuf" round-trip; pinned by `TestEventWireEncodingOpenNamespace`). The review's `record.Encoding` suggestion was REJECTED deliberately: its closed enum would drop custom codec stamps; documented on both wire structs. Wire bytes unchanged (goldens unaffected)                                                                                                                                     | new test ✓; bbolt/pebble suites ✓                                                                                                                           |
| A6  | **E7**: watermill `RetryConfig` → `HandlerRetryConfig` (+ `DefaultHandlerRetryConfig`); old names remain as Deprecated aliases (die at v6)                                                                                                                                                                                                                                                                                                                                                                                                       | watermill suite ✓                                                                                                                                           |
| A7  | **E6**: `BundleOption` merged into `Option` (one config struct, one option type; `NewOTelBundle` takes `...Option`); `BundleOption` survives as Deprecated alias                                                                                                                                                                                                                                                                                                                                                                                 | middleware suite ✓                                                                                                                                          |
| A8  | **E8**: typed `middleware.Kind` + `KindCommand`/`KindEvent`/`KindQuery`; `MessageAdapter.Kind` + `DeadLetterEntry.Kind` carry the enum; string-backed so log/SQL/OTel output unchanged                                                                                                                                                                                                                                                                                                                                                           | middleware suite ✓                                                                                                                                          |
| A9  | **E11**: `AdapterCore.Encode` → `func(T) (string, error)`; `ToAny` propagates; the three adapters no longer persist nil metadata on marshal errors                                                                                                                                                                                                                                                                                                                                                                                               | system suite ✓; examples build ✓                                                                                                                            |
| A10 | **E13**: SQLTimerStore phantom param documented (Go has no generic methods — the review's "or document" resolution)                                                                                                                                                                                                                                                                                                                                                                                                                              | doc comment; build ✓                                                                                                                                        |
| A11 | **E15**: `dispatcher.Middleware[H]` is THE shared shape; `command.Middleware`, `command.PublishMiddleware`, `event.Middleware`, `event.PublishMiddleware`, `query.Middleware`, `middleware.Middleware` are now aliases — one function value composes everywhere, zero consumer migration                                                                                                                                                                                                                                                         | `TestMiddlewareAliasesAreIdentical`; dispatcher/command/event/query/middleware/decider suites ✓                                                             |
| A12 | **E3/E9/E10/E14 verified already-done** in earlier waves (bbolt errorfamily wraps; turso Policy nil-write guards; ShutdownDependency validation incl. unknown-engine rejection; OwnedDBHandle vs DBHandle type split)                                                                                                                                                                                                                                                                                                                            | source inspection                                                                                                                                           |
| A13 | **T18 migration tail**: corruption / half-migrated / concurrent-init / legacy-subset tests verified present; LIVE DuckDB migration run added (`integration/snapshot_migration_duckdb_integration_test.go`, information_schema probe + RENAME on the real embedded engine) and GREEN; MariaDB leg ready, gated on the nspawn/VM infra (no local MariaDB; systemctl blocked)                                                                                                                                                                       | live run PASS                                                                                                                                               |
| A14 | **ADR-0139 owner memo**: all 4 open questions (provider semantics / reference validation timing / read-model scope / plaintext→encrypted migration) + Q5 (SQL columns 5.0-vs-5.x) consolidated into ONE reply-format memo with options + recommendations                                                                                                                                                                                                                                                                                         | `docs/reviews/2026-09-22_adr0139-encryption-and-sweep-owner-memo.md`; linked from the ADR                                                                   |
| A15 | **V5-MIGRATION-GUIDE expanded**: the missing `relational → metaengine` before/after added (real signatures — fixed after doc-check arity checks); pre-existing `NewMaterialize` arity lie fixed; systemtest replace-strip added to the cut checklist; v5 banners added to `MIGRATION_TO_STACK.md`/`MIGRATION.md`/`MIGRATION_v1.md` (historical-guide sweep)                                                                                                                                                                                      | doc-check clean on the touched docs (45 + 55 refs)                                                                                                          |
| A16 | **Feedback #4 — systemtest split (the L item)**: new Tier-7 `systemtest/` module owns system's real-engine suites (13 files git-mv'd + 4 sqlite wiring tests extracted + checkpoint restart test via the new public `system.NewEngineCheckpointStore`); `system/go.mod` drops ALL engine requires (sqlite/pebble/badger/pg GONE — consumers of system/v4 now pull zero engines); fixture twins carry `//art-dupl:accept`; registered in go.work, flake testModules, api-stability, module-layers/budget, cqrs-lint catalog; census 96→97 modules | system suite ✓ (memory-only); systemtest suite ✓ (34 PASS, 1 DSN-skip); `TestEveryModuleGoSumIsTidy` ✓; canonical-facts ✓; module-layers ✓; catalog tests ✓ |
| A17 | **DeferClose twin deprecation note** (XS): `Deprecated` doc note on `metaengine.DeferClose` (canonical home record.DeferClose; twin kept through v5 for the sibling-replace family, removed at v6)                                                                                                                                                                                                                                                                                                                                               | metaengine build ✓                                                                                                                                          |
| A18 | **API golden regenerated twice** (E-items, then systemtest) — net surface: +dispatcher.Middleware, +middleware.Kind×4, +watermill.HandlerRetryConfig (RetryConfig struct→alias), +system.NewEngineCheckpointStore; `check-changelog-symbols` ✓ (23 honest citations)                                                                                                                                                                                                                                                                             | `TestEvery` ✓                                                                                                                                               |
| A19 | **Consumer grep (pre-cut baseline)**: go-graph-rag / go-appkit / PapDashboard — NO consumer depends on the old aggregate wire spellings (PapDashboard's audit.go columns are its own local schema). Receipt recorded in WIRE-FORMAT-KEYS.md; re-run at the cut                                                                                                                                                                                                                                                                                   | grep receipt in doc                                                                                                                                         |
| A20 | **Preflight composed: GREEN** (all 6 phases) after fixing templ drift (regenerated from the right cwd) + 3 new duplication groups (annotated per convention)                                                                                                                                                                                                                                                                                                                                                                                     | preflight output                                                                                                                                            |

## b) The honest ledger (what I got wrong)

1. **Config-corruption incident #12 caught live** — the 12:04:37 daemon
   commit restored a STALE `.golangci.yml` (go 1.26.7 + jsonv2 tag resurrected
   - the entire depguard allow-list deleted, 111 lines). The hash tripwire
     did its job; per its remedy #2 (inspected first: stale overwrite, not an
     intentional edit) I restored the config from the parent commit; hash
     matches. flake.lock changes in the same commit were LEGITIMATE (newer
     lock entries) and were kept. Root cause of the stale-write process is
     still unidentified — same class as incident #11, one more datapoint for
     the pre-commit concurrency-hazard case (report §e3).
2. **Hallucinated a helper** while twinning `loadScaledDeadline` into
   systemtest: I invented a `loadavg()` signature instead of copying the real
   implementation. Caught on vet; replaced with the faithful twin. This is
   exactly the verify-external-claims failure mode — the compile gate saved me.
3. **Wrote a fake test body** in the first cut of the restart-durability test
   (guessed a `MapBackend()` interface). Caught immediately; rewrote against
   the real `metaengine.MapBackend` + the new public constructor.
4. **Extraction/fixture whack-a-mole**: the systemtest split surfaced five
   rounds of shared-fixture undefineds (mustApply, Task fixtures,
   taskDomainConfig, taskProjectionQuery, waitForProjectionProcessed,
   recordingCheckpointStore). Resolved with twin-and-accept each time, but I
   should have enumerated helper usage BEFORE the git mv.
5. **Annotated the wrong ToAny block first** (line 61 vs the reported
   74-79 region) — art-dupl reports one group per region pair; annotate ALL
   same-shape regions. Re-run caught it.
6. **Quiet-window invocation trap hit AGAIN** (missing inner `--`) — the
   exact trap the 11:32 report §d3 documented. The wrapper's usage output
   taught it on the first attempt this time; the flag grammar deserves a
   `--` passthrough fix.

## c) NOT DONE / gated (with the why)

- **Cut v5.0.0** — the paste's final item. Gated on the ADR-0123 deletion
  waves (Materialize, Relational+view, GraphProjection, Bundle+presets,
  ADR-0126 shells, BuildWhereClause, NewStreamRef validation, transport
  deletions, tombstone API) which are NOT part of the pasted mandate and are
  still open in the v5 section; cutting now would ship a v5.0.0 that
  contradicts its own contract. The cut checklist in V5-MIGRATION-GUIDE §5
  is otherwise staged.
- **Live MariaDB migration run** — test ready (`-tags integration` +
  `MYSQL_TEST_DSN`); no local MariaDB this session (no socket, systemctl
  blocked); ride the next `#integration-mysql-nspawn` quiet window.
- **ADR-0139 implementation** — owner ruling on the memo's Q1–Q4.
- **`listing.aggregate_projection` rename** — TBD collection-identity call,
  documented in the wire-key table.

## d) Verification stack (cumulative, this session)

- Module suites (GOWORK=off, -count=1): system, systemtest, middleware,
  watermill, storage/pebble, storage/bbolt, metaengine, system(+builders),
  cmd/cqrs-lint, dispatcher, command, event, query, decider — all green.
- Gates: check-file-size ✓ · check-module-layers ✓ · canonical-facts ✓ ·
  changelog-symbols ✓ · doc-check (1245 refs incl. the touched docs) ✓ ·
  api-stability golden + TestEvery ✓ · cqrs-lint analyzer pkg ✓ ·
  preflight-composed ✓ (all 6 phases).
- Workspace `go build ./...` ✓; all six examples build ✓.
- **Composed `#verify`: ARMED and RUNNING** via quiet-window-run (correct
  `--` nesting this time); log:
  `/tmp/quiet-window-run.f4y4rJ.log`. This report records its state at
  launch; the next session should read the tail and record S04.

---

_Report committed by the auto-commit daemon._
