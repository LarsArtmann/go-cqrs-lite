# Status Report — SEC Lessons Integration Session 2

> **Date**: 2026-06-08 10:02
> **Duration**: ~1 hour (09:27 → 10:02)
> **Source Plan**: `docs/planning/2026-06-08_00_08-SEC_LESSONS_INTEGRATION_PLAN.md` (90 tasks)
> **Previous Status**: `docs/status/2026-06-08_09-27_SEC_LESSONS_INTEGRATION_STATUS.md`

---

## Executive Summary

Resumed execution of the 90-task SEC lessons integration plan. Cleaned up debt from the previous session: removed the orphan `testutil/snaptest` package, fixed unused parameters in `example/user/server.go`, and verified all verification gates pass cleanly (build/test/lint/check-layers = 0 issues).

Build: ✅ | Tests: ✅ 39/39 packages | Lint: ✅ 0 issues | Module layers: ✅ Pass

|| Metric | Value |
|--------|-------|
| **Sprint 1 (Trust & Docs)** | 100% complete |
| **Sprint 2 (Operational)** | 100% complete |
| **Sprint 3 (Testing Rigor)** | ~45% complete (PBT done for 5/5 core modules; snapshot tests partial) |
| **Sprint 4 (CI & Deployment)** | ~20% (gosec + module-layers in CI; no benchmark regression or Docker CI) |
| **Sprint 5 (Consumer Experience)** | ~15% (SSE, config, sim, dual-store examples done; no Playwright, no catalog-server) |
| **Sprint 6 (Polish & Experiments)** | 0% |
| **Overall completion** | ~45% of 90 planned tasks |

---

## a) FULLY DONE

### Previous Session (carried forward)
- ✅ **FEATURES.md** — 19 planned features added
- ✅ **docs/DOMAIN_LANGUAGE.md** — ~40 CQRS domain terms
- ✅ **CONTEXT.md** — Architecture overview
- ✅ **ROADMAP.md** — 6-sprint plan + long-term vision
- ✅ **gosec in devShell + CI** — SARIF upload to GitHub
- ✅ **scripts/check-module-layers.sh** — Custom multi-module layer checker
- ✅ **Module layers CI step** — `nix run .#check-layers` wired into CI
- ✅ **Dependency budgets** — Per-module dep limits in check-module-layers.sh
- ✅ **middleware/healthcheck.go** — RFC-compliant health endpoints
- ✅ **middleware/metrics_http.go** — HTTP request metrics
- ✅ **middleware/sse.go** — SSE broker over event.Bus
- ✅ **pkg/gracefulshutdown/shutdown.go** — Signal-based shutdown
- ✅ **pkg/config/config.go** — JSON config loader with env overlays
- ✅ **decider/property_test.go** — 3 rapid PBTs
- ✅ **event/property_test.go** — 3 rapid PBTs
- ✅ **id/property_test.go** — 3 rapid PBTs
- ✅ **command/property_test.go** — 5 rapid PBTs (creation round-trip, empty type, metadata, dispatch, unregistered)
- ✅ **query/property_test.go** — 5 rapid PBTs (creation, empty type, pagination bounds, offset, dispatch)
- ✅ **integration/simulation/generator.go** — EventGenerator for stress testing
- ✅ **integration/snapshot_test.go** — Event serialization golden file test
- ✅ **example/user/Dockerfile** — Multi-stage build
- ✅ **example/user/docker-compose.yml** — Production health check
- ✅ **example/user/sse_example.go** — SSE broker demo wired into main
- ✅ **example/user/config_usage_example.go** — Config loader demo wired into main
- ✅ **example/user/dual_store_example.go** — Memory/SQL switching demo wired into main
- ✅ **benchmarks/benchmark-baseline.txt** — All benchmark results saved

### This Session
- ✅ **Removed `testutil/snaptest/` orphan** — Had no go.mod, no consumers, was dead code
- ✅ **Fixed `example/user/server.go`** — Removed 3 unused parameters (cmdDisp, qryDisp, bus), removed unused componentHealthCheck function, cleaned up unused imports (command/v2, query/v2). Still has `//go:build ignore` — it's a reference file, not compiled.
- ✅ **Verified `nix run .#check-layers`** — Passes cleanly: "Module layer check passed"
- ✅ **Verified CI wiring** — gosec + module-layers steps confirmed in `.github/workflows/ci.yml` (8 references)
- ✅ **Full verification suite** — Build ✅, Tests ✅ (39/39 packages), Lint ✅ (0 issues across all modules)

---

## b) PARTIALLY DONE

### Testing Rigor — Snapshots (Sprint 3)
- ⚠️ **integration/snapshot_test.go** — 1 test (event serialization) + golden file
  - catalog snapshot test was created then deleted in previous session (wrong API)
  - Needs rewrite using correct API: `asyncapi.NewExporter(name, ver).Export(cat)` → `doc.MarshalYAML()`
  - Other exporters: `openapi.NewExporter(name, ver).Export(cat)`, `d2.NewExporter(name, ver).Export(cat)`, `eventcatalog.NewExporter(dir).Export(cat)`
  - The catalog API is: `catalog.NewRegistry(title, ver)` → `reg.AddCommand/AddEvent/AddQuery` → `reg.Build()` → `*Catalog` → pass to exporters

---

## c) NOT STARTED

### Sprint 3 — Remaining Testing Rigor
- ❌ **catalog/snapshot_test.go** — Rewrite with correct API (was deleted)
- ❌ **Snapshot tests for projection/, signing/, memory/** — State/payload serialization verification
- ❌ **PBT for pagination edge cases** — `query.NewPagination(0, 0)` default behavior, `PaginatedResult.HasNext()/HasPrev()` boundary cases

### Sprint 4 — CI & Deployment
- ❌ **Benchmark regression CI step** — Compare against `benchmarks/benchmark-baseline.txt`
- ❌ **Docker build CI step** — Multi-platform build (linux/amd64 + arm64)
- ❌ **benchmark-baseline.txt auto-update** — Manual save only currently

### Sprint 5 — Consumer Experience
- ❌ **example/catalog-server/** — New module for embedded EventCatalog SPA
- ❌ **example/user/ SSE JavaScript client** — `web/static/sse-client.js`
- ❌ **Playwright setup** — `playwright.config.ts` + E2E tests
- ❌ **Playwright CI step** — Not added to workflow

### Sprint 6 — Polish & Experiments
- ❌ **Build tag documentation** (`jsonv2`, `arenas`, `simd`, `runtimesecret`)
- ❌ **jsonv2 codec experiment** behind build tag
- ❌ **Arena allocation experiment** in event module
- ❌ **Snapshot tests for storage/, watermill/, pebble/, turso/, codec/, otel/, schema/, snapshot/** modules

### Full Plan: ~50 of 90 tasks NOT done

---

## d) TOTALLY FUCKED UP

### Previous Session (carried forward)
- 💀 **catalog/snapshot_test.go** — Created with wrong API (`catalog.NewAsyncAPIExporter` doesn't exist), wrong `catalog.Event[T]()` signature (needs `(id, direction, ...opts)` not `(string)`). Deleted but never rewritten.
- ⚠️ **TestFoldIdempotency** — Was the wrong invariant; renamed to `TestFoldAccumulation`
- ⚠️ **SSE broker** — 3 iterations to get right: v1 used `ro.Observer` (no Unsubscribe), v2 used `ro.OnNext` (same issue), v3 used `event.Bus.SubscribeAll` ✓
- ⚠️ **testutil/snaptest** — Created as snapshot helper, never integrated into any module's go.mod, had zero consumers. NOW DELETED.

### This Session
- Nothing fucked up. Clean session — only removed dead code and fixed existing issues.

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **`example/user/server.go` still has `//go:build ignore`** — It's a reference file that doesn't compile into the example binary. The health/metrics endpoints ARE demonstrated via `demonstrateSSE()`, `demonstrateConfig()`, and `demonstrateDualStore()` in the actual running example. But `runServer()` itself is unreachable. Should either: (a) wire it into `main.go` with a flag, or (b) remove it since the demos already cover it.

2. **PBT coverage gaps** — `command/` and `query/` now have 5 PBTs each, but pagination edge cases (`NewPagination(0, 0)`, `PaginatedResult` boundary behavior) deserve dedicated tests.

3. **Catalog API documentation** — The exporter API is non-obvious: `asyncapi.NewExporter` returns an Exporter with no visible methods in the 78-line file (Export is likely defined elsewhere or embedded). The integration test at `catalog/integration_test.go:88` shows the pattern but it's not documented.

### Architecture

4. **Snapshot test infrastructure** — The previous session tried to build `testutil/snaptest` but it had no go.mod (orphan). For proper snapshot testing, each module should either: (a) inline its own `Match()` helper, or (b) a shared `testutil` package needs its own go.mod with replace directives.

5. **Example module dependency surface** — `example/user/go.mod` has 11 direct dependencies. This is fine for an example, but it means running the example pulls in a lot. The catalog demo alone pulls in go-faster/yaml + all its transitive deps.

### Testing

6. **No coverage regression gate** — Tests pass but there's no CI gate on coverage percentage. PBTs add value but we don't track coverage delta per PR.

---

## f) TOP #25 THINGS TO GET DONE NEXT

Ordered by impact × customer-value / (effort × risk).

| Rank | Task | Module | Impact | Effort | Why |
|------|------|--------|--------|--------|-----|
| 1 | **Rewrite `catalog/snapshot_test.go`** with correct API | catalog | 8 | 2 | Sprint 3 gap — was deleted, needs redo. Pattern: `asyncapi.NewExporter(name, ver).Export(cat)` |
| 2 | **Add snapshot tests for `projection/` state** | projection | 7 | 2 | Core module with zero snapshot coverage |
| 3 | **Add snapshot tests for `signing/` payloads** | signing | 6 | 1 | Security-critical module needs golden file verification |
| 4 | **Add snapshot tests for `memory/` serialization** | memory | 5 | 1 | Test-only impl but used everywhere |
| 5 | **Wire `runServer()` into example/user/main.go** or remove it | example/user | 6 | 1 | Dead code with build-ignore tag |
| 6 | **Build `example/catalog-server/`** module | example | 7 | 4 | Full new example for catalog consumers |
| 7 | **Add JavaScript SSE client** to `example/user/web/static/` | example/user | 5 | 2 | Required for SSE demo to work end-to-end |
| 8 | **Playwright setup** in `example/user/` | example/user | 7 | 4 | E2E testing for example app |
| 9 | **Add Playwright CI step** | ci.yml | 6 | 2 | Sprint 5 — gate on E2E |
| 10 | **Benchmark regression CI step** | ci.yml | 6 | 3 | Detect perf regressions against baseline |
| 11 | **Docker build CI step** (multi-platform) | ci.yml | 5 | 3 | Sprint 4 — deployment readiness |
| 12 | **Document gosec + go-arch-lint** in CONTRIBUTING.md | docs | 5 | 1 | New contributors need to know about security/layers tools |
| 13 | **Add `nix run .#check-deps`** app for dep budget visualization | flake.nix | 5 | 2 | Surfaces dependency budget state per module |
| 14 | **Document build tag experiments** in flake.nix comments | flake.nix | 4 | 1 | Sprint 6 — surfaces what's available |
| 15 | **Add PBT for `query.Pagination` edge cases** (zero inputs, overflow) | query | 4 | 1 | Boundary testing for pagination math |
| 16 | **Add snapshot tests for `storage/` SQL** | storage | 6 | 3 | SQL schema verification |
| 17 | **Add snapshot tests for `codec/` encoding** | codec | 4 | 1 | Encoding round-trip verification |
| 18 | **jsonv2 codec experiment** behind build tag | codec | 4 | 3 | Sprint 6 — experimental performance |
| 19 | **Arena allocation experiment** in event module | event | 3 | 4 | Sprint 6 — experimental performance |
| 20 | **Add snapshot tests for `watermill/` adapter** | watermill | 4 | 2 | Protocol adapter verification |
| 21 | **Add snapshot tests for `pebble/` store** | pebble | 4 | 2 | Embedded store verification |
| 22 | **Add snapshot tests for `turso/` connector** | turso | 4 | 2 | Database connector verification |
| 23 | **Add snapshot tests for `otel/` helpers** | otel | 3 | 1 | Telemetry helper verification |
| 24 | **Add snapshot tests for `schema/` upcaster** | schema | 5 | 2 | Schema evolution verification |
| 25 | **Coverage regression gate** in CI | ci.yml | 5 | 3 | Track coverage delta per PR |

---

## g) TOP #1 QUESTION

**How should snapshot/golden-file testing work in a Go multi-module workspace?**

The previous session tried to create `testutil/snaptest` as a shared helper but it had no `go.mod` (orphan). Options:

1. **Each module inlines its own `Match()` helper** (like `integration/snapshot_test.go` does with a local helper) — zero cross-module dependency, but duplicated ~50 lines per module
2. **Create a proper `testutil/` module with its own `go.mod` + replace directives** — single source of truth, but adds a module and replace directive to every consumer's go.mod
3. **Use `encoding/json` + `cmp.Diff` inline** instead of golden files — no helper needed, but no persistent baseline
4. **Use an external library** (e.g., `go-snaps`) — adds a real dependency

For a library SDK, option 1 (inline per-module) seems cleanest but tedious. Option 2 is architecturally correct but adds maintenance burden. What's the intended direction?

---

## Verification Summary

| Gate | Status | Details |
|------|--------|---------|
| `nix run .#build` | ✅ PASS | All modules compile |
| `nix run .#test` | ✅ PASS | 39/39 packages OK |
| `nix run .#lint` | ✅ PASS | 0 issues across all modules |
| `nix run .#check-layers` | ✅ PASS | Module layer check passed |
| Git working tree | ⚠️ MODIFIED | `example/user/server.go` (fixed), `testutil/snaptest/snaptest.go` (deleted) |

## Git Changes (Uncommitted)

```
 M example/user/server.go        — removed unused params, cleaned imports
 D testutil/snaptest/snaptest.go — removed orphan package
```
