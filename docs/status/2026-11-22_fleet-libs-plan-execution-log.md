# Fleet-Libs Audit — Pareto Plan Execution Log

> Execution of `docs/planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md`
> (T01–T25). One section per wave, per-task verdicts with the plan's done-when checks.
> This file is the wave-notes home (m06 tag-prep lists live here).

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
