# Self-Review TODO Sweep — Complete

**Date:** 2026-07-25 19:00 CEST
**Scope:** Execute the entire 50-item self-review TODO list (`docs/status/2026-07-25_17-32_brutal-self-review-and-comprehensive-status.md` §f).
**Plan:** [`docs/planning/2026-07-25_18-00_self-review-todo-sweep-plan.md`](../planning/2026-07-25_18-00_self-review-todo-sweep-plan.md)
**Result:** ✅ Every actionable item DONE or explicitly deferred-with-tracking. `#verify` GREEN (build+vet+test+race+lint+api-stability+doc-check). `#check-api-stability` GREEN (2652 exports).

## Gates (both green)

- `nix run .#verify` → exit 0. `#verify` now **includes** the api-stability step (added this session) — the "sister gate never run" gap is permanently closed.
- `nix run .#check-api-stability` → 2652 exports verified (golden regenerated for the new `testutil.RaceEnabled` export).
- `nix run .#lint` → 0 issues across all modules.

## Coverage measured

| Module | Coverage |
|--------|----------|
| idempotency (parent) | 100.0% |
| benchkit | 85.0% |
| metaengine | 84.6% |
| idempotency/kvstore | 65.1% |

## Key findings (new this session)

1. **Broken published module graph** — `codec/v4.0.4`, `decider/v4.0.3`, `listing/v4.0.3`, `storage/v4.0.3` are referenced in published `go.mod` files but were never tagged. This breaks `go mod tidy` (and GOWORK=off builds) for any module that newly pulls them. `#verify` stays green because it runs in workspace mode (go.work provides locals). Tracked as 🔥 BLOCKED in TODO_LIST.
2. **gopls phantom diagnostics root-caused** — stale in-memory snapshot after the `sink.go` file-split (gopls cited line numbers past EOF). Fix: restart gopls. Documented in AGENTS.md. Also: gopls runs without `goexperiment.jsonv2`, making its analysis unreliable.
3. **integration module is workspace-only** — its `go test` fails setup under GOWORK=off even at clean HEAD (pre-existing). The 3-way idempotency contract test therefore lives in `idempotency/kvstore` (clean GOWORK=off graph) not integration.
4. **otel_bundle.go drift** introduced by a daemon commit after the prior green verify; fixed (gci grouping) as part of the lint sweep.

## Scope-discipline notes

- The lint sweep formatted 7 files not authored this session (event/*, example/taskmanager/*, middleware/otel_bundle.go, storage/relational/schema_test.go) — all benign formatting drift that turned `#lint` red. Fixed because (a) the verify gate must be green and (b) the self-review's lint-sweep items (f13/14/17-19) explicitly call for repo-wide hygiene.
- One pre-existing gate-blocker (`middleware/otel_bundle.go` gci) was fixed in-place rather than flagged-and-skipped, because the user's mandate was "until everything works."
