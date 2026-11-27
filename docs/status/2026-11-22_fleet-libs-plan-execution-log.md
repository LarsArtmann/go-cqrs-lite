# Fleet-Libs Audit — Pareto Plan Execution Log

> Execution of `docs/planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md`
> (T01–T25). One section per wave, per-task verdicts with the plan's done-when checks.
> This file is the wave-notes home (m06 tag-prep lists live here).

## W4 — Consolidation + closeouts (T15–T25) — DONE 2026-11-22

> Bulk landed in commit `34fef2212`; the T19 cmdguard port + T25 closeouts + this
> log section are the same-session tail. Owner questions outstanding: T15 verdict
> ratification, T19 shape (flat-flags compat chosen), pin-wave + remote-tag policy.

### T15 catalog D2 → go-output/d2 — REFUSED (plan's own gate)

- m52's done-when ("D2 golden unchanged") is unreachable: `escape.SlugifyID` is NOT a
  drop-in for hand-rolled `sanitizeID` (catalog/v4/docserver/connections.go:221) — it
  lacks ToLower and adds asterisk/bracket handling, so connection IDs change bytes; and
  go-output/d2's `Diagram` renderer does not reproduce the golden-pinned output format.
- Swapping would force a D2 golden re-baseline = user-visible diff churn — exactly the
  verschlimmbesser the plan's §0 blesses refusing. m51/m53 moot (no swap, no budget bump).
- **Reconsider when:** go-output grows a byte-compatible renderer, or the v5 train
  re-baselines the D2 goldens anyway.

### T16 retry consolidation — DONE (m54); m55 REFUSED

- middleware/v4/retry.go: hand-rolled loop → `retry.Do` (go-retry). Config parity:
  OnRetry logging, OnExhausted → DeadLetter entry, per-attempt OTel spans inside
  AttemptFunc. Sentinels `ErrRetryExhausted`/`ErrRetryCanceled` preserved by wrapping
  go-retry's errors (cause chains keep ErrCanceled/ErrDeadlineExceeded distinctions);
  dead `backoff()` helper deleted. Full middleware suite green.
- **m55 (event/v4 PublishRetry) REFUSED on the v4 byte-freeze**: event/ is a frozen
  train — a behavioral consolidation there is a v5-train change, not an audit follow-up.
  No ADR cut; this row is the decision record.

### T17+T18 errorfamilytest sweep — DONE (19 sites, 6 rowed)

- 19 mechanical `errorfamily.Classify(err) != family` sites converted to
  `errorfamilytest.AssertFamily` across 12 files (schema ×2, command, id, storage ×2,
  event, storage/turso ×3, integration, benchkit). errorfamilytest is a subpackage of
  the go-error-family root module — imports only, zero new deps. All 8 module suites green.
- Plan's ~98 estimate was wrong: only ~25 real sites existed. 6 non-mechanical shapes
  ROWED, not converted: `.Family() !=` forms in event/v4/errors_taxonomy_test.go:80,
  event/v4/coverage_test.go:159, stack/pebble/v4/preset_test.go:328 (foreign-dirty file —
  untouched by design), plus three assertion-shape variants that assert on family
  extraction rather than classification.

### T19 cqrs-upgrade → cmdguard — DONE

- Sixth CLI on the fleet chassis: `AppConfig` typed flag struct (7 flags, help strings
  byte-identical to the stdlib set — pinned by `TestCLIHelpParity`), `buildCLI()` +
  `cli.RootCommand().RunE`, `ExecuteAndExit` entry. `AppConfig` deliberately does NOT
  embed `cmdguard.Config`: the batteries' config carries its own `--strict` flag, which
  would collide with the v5-readiness gate.
- Shape: **flat single-command** (owner Q2 — compat-first; subcommand restructure would
  break every scripted `cqrs-upgrade --json --strict <dir>` consumer for zero gain).
- Parse contract preserved: `cobra.ArbitraryArgs` for the positional dir;
  `SetInterspersed(false)` keeps the 2026-09-11 hardening true (flags-after-dir is a
  LOUD `errArgsAfterDir`, not pflag's silent interspersed apply — pinned by
  `TestCLIParse_RejectsArgsAfterDir`/`TestCLIParse_FlagsBeforeDirReachConfig`).
- Exit-code contract: RunE errors propagate through `ExecuteWithArgs`
  (`errors.Is(errStrictViolations)` still holds in the e2e ports); `--json` stdout wire
  clean (cmdguard prints errors to stderr).
- Codemod golden suite unchanged (green); kill-mid-write test green; DEP_BUDGET 3→5
  (cmdguard + cobra, rationale inline).

### T20 signing determinism test — DONE

- signing/v4/payload_determinism_test.go (internal): canonicalPayload idempotency +
  DefaultCodec/JSONCodec/CBORCompactCodec all satisfy `codec.DeterministicCodec`.

### T21 cqrs-gen dogfood — DONE

- ~~`//cqrs:command task.create` marker on CreateTaskCmd (example/taskmanager/handlers.go);~~
  ~~handlers_gen.go committed beside it; README regeneration-loop section. Generator quirk~~
  ~~recorded: needs an ABSOLUTE path arg (`cqrs-gen --type=command --output=<abs> <abs-dir>`).~~
  **Superseded — the loop never ran: `RegisterCreateTaskCmdHandler` had zero
  callers and no test compiled or executed generated code. Retired in `0b4d58409`;
  `example/goal-shaped-app` is the live dogfood (ADR-0168).**

### T22 go-sse CHANGELOG — DONE (other repo, commit 1b7d168)

- [0.7.0] section cut from Unreleased, dated to the tag date 2026-11-04.

### T23 audit artifact gates — DONE

- `scripts/check-libreports.sh` + flake app `#check-libreports`: citation sweep (900 ok /
  0 bad / 23 reports) + index↔report↔asset link bijection; `--self-test` fault-injects.
  First live run caught a stale citation the T05 edit caused → go-atomic-write report
  re-derived (UPDATE header, not retro-edited).

### T24 session infra — DONE

- m69: status row for the 16-30 audit report. m70: border-before-trim screenshot lesson →
  crush-config lessons.md (bc637bf). m71: sequential-dispatch+resume pattern →
  SKILLS/library-deep-dive runbook (f4cd4a8).

### T25 closeouts — DONE

- m72: `#verify-fast` first run caught TWO file-size ratchet violations the wave
  commits smuggled past per-module testing: `cmd/cqrs-bench/v4/factory.go`
  (398→429, grown by the T10 system-memory backend) and
  `cmd/doc-check/v4/recipes_catalog_meta3.go` (408→420, the §2.47 fence-#5 entry).
  Root-cause fix, not re-baseline: factory.go split into `factory_system.go`
  (the system/memory backends) + `parse_helpers.go` (CLI parse helpers) → 300
  lines; the §2.47 recipe family + its helper const moved from meta3 to meta4 →
  meta3 284 / meta4 193. Both baselined rows dropped (the shrink-only ratchet,
  61→59 entries), both module suites green, file-size gate green. The full
  `#verify-fast` re-run to green is still owed before m72 closes.
- m73: `level=warn` stray dispositioned — origin identified (nats-server stdout captured
  by scripts/ephemeral-nats.sh on 2026-09-29, absorbed by daemon commit 3756eb4d1,
  later dropped from tracking); file no longer exists on disk or in the index. Nothing
  to trash.
- m74: error-taxonomy doc header v0.10.0 → v0.12.0 (matches every module pin).
- m75: **pending owner Q3** (fetch remote tags vs stay local) — no policy invented.

## W3 — Bridges + seams (T08–T14) — DONE 2026-11-22

### T08 SSE correctness — DONE

- `parseLastEventID` (metaengine/v4/sse.go): wire-valid via `sse.ParseEventID` + numeric seq;
  validated BEFORE header commit; present-but-unusable → **400** (was: silent full-journal replay).
- `SSEConfig.Retry` + `WithSSEReconnectAfter` → `sse.WriteRetry` behind committed headers on BOTH
  paths (subscribeSSEHeaders), default off.
- Drop-old-vs-Broadcaster rationale paragraph in ServeSSE doc (watcher already IS the fanout).
- Tests: `TestSSE_MalformedLastEventID_Returns400` (banana/3.5/-1), `TestParseLastEventID`
  (incl. newline-corrupt via constructed request), `TestSSE_ReconnectAfter_Advertised`
  (sticky Retry=2500ms on first event).

### T09 ssetest/sseparse adoption — DONE

- metaengine raw-TCP legs (LastEventID_Reconnect, ReplayLimit) rewritten on
  `ssetest.NewStreamReader` + `RequireEventID`/`RequireDataContains` typed asserts; sleep-free
  (the header-commit ordering guarantee replaces the timing hacks); reading-exactly-N is the
  absence proof. ssetest v0.5.0 test-dep.
- systemscenario/v4/sse.go: hand-rolled bufio parser → `sseparse.NewStreamReader`; `id:` fields
  recorded (`ReceivedIDs()`) + surfaced in Await failure diagnostics. sseparse v0.2.1 dep (budget OK).

### T10 benchkit bridge — DONE

- cqrs-bench `system-memory` backend via `benchkit.FactoryFromSystem` + system.New deployment;
  presets marked legacy-path (memory stays preset for benchmark comparability — deliberate
  deviation from "flip the default", recorded here; the system adapter honestly skips kv
  read-model phases so `memory` numbers would silently change meaning).
- Smoke run green (`run --backend system-memory --profile dev`); suite green 77s; budget 19→20.

### T11 go-health seam — DONE

- example/taskmanager: `/healthz` = `health.NewChecks{"engines": Sys.HealthCheck}` probe
  ReadinessHandler (the audit's free seam — CheckFunc ≡ HealthCheck); pinned by
  `TestHardening_GoHealthProbeRendersEngineDetail`. go-health v0.10.0 dep.
- Recipe §2.47 fence #5 (cataloged + pinned) with the dashboard flip-trigger note.

### T12 docserver polish — DONE

- `etag.New` wrap on the six deterministic spec/data routes (304 test: `docserver_etag_test.go`).
  HTML pages deliberately unwrapped: per-request CSP nonce legitimately changes bytes each render
  (decision recorded in RegisterRoutes + csp.go). DEP_BUDGET catalog 9→10.
- `renderComponent` buffers the templ render: render errors become clean 500s (was: best-effort
  after first byte) and single-write pages gain Content-Length.
- d2view.templ: two hand-rolled pre/code + CopyButton pairs → one `display.CodeBlock` (templ
  regenerated with the pinned v0.3.1100 CLI; `#check-templ` green both legs; `#check-csp` PASS).

### T13 go-appkit refresh — DONE (other repo, commit 9a03806)

- cqrs: metaengine v4.26.0, sqliteengine v4.9.1 (EnforceCap deadlock fix), system v4.19.0,
  systemscenario v4.3.0; docs: catalog v4.15.0; integration re-pinned same wave.
- Module tests green (cqrs 22.5s, docs, integration via published-tag resolution); CHANGELOGs cut;
  **tagged cqrs/v0.10.0 + docs/v0.3.4 and pushed** (plan m47 "+ release").

### T14 scheduler engine-path example — DONE

- New `example/scheduler-engine-status`: `scheduling/engine.NewTimerStore[reminder]` over the
  memory DueClaimer engine, OTel fired counter on /metrics, JSON /status with rate anchor;
  `TestScheduler_FiresDueTimer` pins schedule→claim→dispatch→retire.
- Full registration: go.work, flake exampleModules, LAYER 7 + DEP_BUDGET 5, api-stability
  exclusion maps (×3), cqrs-lint catalog known-absent map, root README row.
- Legacy scheduler-otel-status banner + README point at the new path (deprecated-path framing).

### Gate repairs ridden in W3 (pre-existing debt, not plan tasks)

- `TestEveryModuleGoSumIsTidy` was red repo-wide (60 modules missing /go.mod checksum rows — the
  documented drift-debt class): mechanical `go mod tidy` sweep, gate green.
- DEP_BUDGET raises with rationale: cqrs-lint 9→10 (atomic-write), cqrs-bench 19→20 (system),
  catalog 9→10 (etag).

## W2 — File safety + golden path (T05–T07) — DONE 2026-11-22

### T05 go-atomic-write in the write-path tools — DONE

- Direct pins v0.6.2 in cmd/cqrs-upgrade (new dep) + cmd/cqrs-lint (indirect→direct).
- cqrs-upgrade: `osWriteFile` (util.go) now `atomicwrite.Write` (staged temp + fsync + rename;
  preserves the target's existing mode — also fixes the old forced-0600 rewrite of go.mod);
  `rewrite_v5.go` + `rewrite_v5_surface.go` write sites route through it (dropping their
  stat-then-write perm dance).
- cqrs-lint: `writeBaseline` DIY temp+rename (no fsync!) replaced by `atomicwrite.Write`.
- Kill-mid-write self-test (`util_kill_test.go`, re-exec child + SIGKILL): victim file is
  old-or-new across rounds, never truncated — GREEN. Full suites green (cqrs-upgrade 7.7s
  incl. codemod goldens unchanged; cqrs-lint 39.6s).

### T06 golden-path hardening (example/taskmanager) — DONE

- `wrapHandler` chain (setup.go): `httputil.Recovery` outermost → `httputil.MaxBodySize(1 MiB)`
  → `servertiming.ServerTimingMiddlewareWhen(timingEnabled)` (Config.ServerTiming + `-server-timing`
  flag; default OFF).
- Health: hand-rolled `/health` → `httputil.HealthHandler` + `/health/live` (LiveHandler) +
  `/health/ready` (ReadyHandlerWithProbe over `system.HealthCheck`).
- 413 mapping: `writeDecodeError` (`*http.MaxBytesError` → 413, else 400) on all four decodeJSON sites.
- Drain order: HTTP `Shutdown` BEFORE `ProjHost.Stop` (in-flight requests' tail events consumed).
- `httpx.go` timing phase → `MeasureWithDesc("query", "typed dispatch + deterministic encode")`.
- New `hardening_test.go`: panic→500, 2 MiB body→413, timing gated off/on, readiness 200 +
  `ready()==true` — all GREEN; full example suite + `-race` GREEN; example-standalone 0 findings.

### T07 recipe §2.47 mirror — DONE

- Job table +3 rows (panic containment, body ceiling, liveness/readiness).
- Helper fence: `MeasureWithDesc` via `ServerTimingFromContext` (matches httpx.go).
- Assemble fence: single `httputil.Chain`, Recovery outermost, `MaxBodySize`,
  `ServerTimingMiddlewareWhen` behind a deployment predicate (was: unconditional outermost timing).
- Prod-posture paragraph names example/taskmanager as the runnable twin incl. drain order +
  hardening_test.go pin.
- Gates: recipes compile harness + doc-check green (see W2 commit), `#check-md-go` green.

## W1 — Version-truth sweep (T01–T04) — DONE 2026-11-22

### T01 go-sse → v0.7.0 — DONE

- metaengine direct pin bumped + 41 modules swept (`go get go-sse@v0.7.0 && go mod tidy` each);
  `rg 'go-sse v0.6' -g go.mod` → **0 hits**.
- `metaengine/v4/sse.go`: `SSEConfig{MaxBuffer: 64}` → `sse.DefaultSubscriberBuffer` (fanout.go:14
  constant, same value); both doc comments now reference the constant.
- metaengine tests green (38.8s, incl. sse_replay legs).
- CHANGELOG `[Unreleased]` row landed; changelog-symbols gate green (36 citations verified).

### T02 go-idempotency → v0.6.0 — DONE

- Bumped in system, integration, idempotency/sqlstore, commandlifecycle, example/taskmanager;
  `rg 'go-idempotency v0.3' -g go.mod` → **0 hits**.
- Stale pebble comment fixed (adapter_test.go: no longer credits the deleted
  `idempotency/kvstore`; now names the atomic set-if-absent seam).
- Tests green: system 4.8s, commandlifecycle 0.02s, sqlstore 25.4s, middleware 3.4s
  (+ integration/-short and example/taskmanager legs recorded in session).

### T03 tool pins + fences — DONE

- httputil v1.8.0→v1.9.0 in example/taskmanager (server_timing already v1.0.3).
- cmdguard v5.0.0→v5.1.0 in api-stability, cqrs-bench, cqrs-gen, cqrs-lint, doc-check; all 5 build.
- `cmd/doc-check/v4/recipes_compile_test.go` externalRequireVersions: go-etag/server v0.6.2,
  httputil v1.9.0, server_timing v1.0.3; `TestRecipes` green (13.0s).

### T04 templ-components v1.27.2 — DONE

- catalog/v4 (root + icons + utils + htmx indirect) bumped; module + docserver tests green.
- `templ generate` re-run from canonical `catalog/v4/docserver` cwd with the pinned v0.3.1100 CLI
  (devshell); first pass with system templ v0.3.1123 was DISCARDED — generator newer than the
  go.mod runtime is exactly the drift the flake pin exists to prevent.
- `#check-templ` green (drift + FileName legs); docs-ui.css regenerated (v1.27.2 rendered-surface
  hardening); `#check-csp` PASS.

### m06 — Tag-prep: modules changed by W1 (for batch-release.sh)

go-sse-swept (pin-only): metaengine/v4, all metaengine/*engine+adapter+observer+bench,
benchkit/v4, scheduling/engine/v4, queue/{sqlite,postgres,mysql}/v4, system/v4,
systemtest/v4, systemscenario/v4, systemcatalog/v4, system/integration/v4,
integration/v4, stack/metaengine/v4, commandlifecycle/projections/v4,
example/{getting-started,goal-shaped-app,graph-native,mesh-demo,metaengine-quickstart,standard-topology,taskmanager},
cmd/cqrs-bench/v4.
go-idempotency-swept (pin-only): system/v4, integration/v4, idempotency/sqlstore/v4,
commandlifecycle/v4, example/taskmanager.
Content-bearing: metaengine/v4 (+DefaultSubscriberBuffer reference), catalog/v4
(templ-components v1.27.2 + regenerated css), cmd/doc-check/v4 (fence pins),
cmd/{api-stability,cqrs-bench,cqrs-gen,cqrs-lint}/v4 (cmdguard v5.1.0), storage/pebble/v4
(comment-only test edit).
Not tagged (test/scratch): cmd/cqrs-lint/v4/testdata/scanfixture, scratch/counterkey.
Release note: run per-module tests before tagging (batch-release verify=ok builds only —
AGENTS "Verify Before Release"). All legs listed above were green at execution time.
