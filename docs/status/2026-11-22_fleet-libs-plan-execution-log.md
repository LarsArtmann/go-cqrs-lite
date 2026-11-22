# Fleet-Libs Audit — Pareto Plan Execution Log

> Execution of `docs/planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md`
> (T01–T25). One section per wave, per-task verdicts with the plan's done-when checks.
> This file is the wave-notes home (m06 tag-prep lists live here).

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
