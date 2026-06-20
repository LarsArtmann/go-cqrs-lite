# SEC Lessons Integration Plan

> Comparing `~/projects/SEC` (production consumer) against `go-cqrs-lite` (library) to identify what the library can learn from its own consumer.
> **Goal:** Raise the bar on trust, operational readiness, testing rigor, and consumer onboarding.

---

## Pareto Breakdown

| Tier          | % of Tasks | Cumulative Value | Focus                                                                                                                                                      |
| ------------- | ---------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1%**        | 9 tasks    | **~51%**         | Documentation (FEATURES.md, DOMAIN_LANGUAGE.md, CONTEXT.md, ROADMAP.md), gosec security scanning, go-arch-lint boundary enforcement                        |
| **4%**        | 18 tasks   | **~64%**         | Health/metrics middleware, graceful shutdown, property-based testing (rapid), snapshot testing (go-snaps), benchmark regression CI, Docker packaging       |
| **20%**       | 55 tasks   | **~80%**         | Embedded EventCatalog server, SSE broker, pkg/config module, simulation framework, Playwright E2E, dual-store example, extended snapshot/property coverage |
| **Remaining** | —          | **~100%**        | Build tag experiments, additional module snapshot tests, documentation polish                                                                              |

---

## Comprehensive Task Table

**Sorted by:** `Impact × CustomerValue / (Effort × Risk)` — highest first.
**Max task size:** ~12 minutes each.

| #   | Task                                                                         | Module / Area | Impact | Effort | Risk | Cust.Val | Score    | Tier      |
| --- | ---------------------------------------------------------------------------- | ------------- | ------ | ------ | ---- | -------- | -------- | --------- |
| 1   | Scan all modules → create FEATURES.md skeleton with module inventory         | docs          | 9      | 2      | 1    | 9        | **40.5** | 1%        |
| 2   | Write FEATURES.md core library modules section (22 modules, honest status)   | docs          | 9      | 2      | 1    | 8        | **36.0** | 1%        |
| 3   | Write FEATURES.md examples, cmd tools & integration section                  | docs          | 8      | 2      | 1    | 8        | **32.0** | 1%        |
| 4   | Write FEATURES.md status table (DONE / PARTIALLY / PLANNED)                  | docs          | 8      | 1      | 1    | 7        | **56.0** | 1%        |
| 5   | Create docs/DOMAIN_LANGUAGE.md — CQRS & Event Sourcing glossary              | docs          | 9      | 2      | 1    | 8        | **36.0** | 1%        |
| 6   | Create docs/DOMAIN_LANGUAGE.md — branded IDs & error taxonomy terms          | docs          | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 7   | Create CONTEXT.md at root — architecture overview & consumer patterns        | docs          | 8      | 2      | 1    | 6        | **24.0** | 1%        |
| 8   | Create ROADMAP.md — short-term actionable tasks (next 90 days)               | docs          | 7      | 1      | 1    | 6        | **42.0** | 1%        |
| 9   | Create ROADMAP.md — long-term vision & raw ideas                             | docs          | 6      | 1      | 1    | 5        | **30.0** | 1%        |
| 10  | Add gosec to flake.nix devShell                                              | build         | 7      | 1      | 1    | 7        | **49.0** | 1%        |
| 11  | Run gosec on event/ module → fix any findings                                | security      | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 12  | Run gosec on command/ module → fix any findings                              | security      | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 13  | Run gosec on decider/ module → fix any findings                              | security      | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 14  | Run gosec on storage/ module → fix any findings                              | security      | 8      | 2      | 1    | 6        | **24.0** | 1%        |
| 15  | Add gosec CI step with SARIF upload to GitHub                                | ci            | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 16  | Create .go-arch-lint.yml with module dependency layer rules                  | arch          | 9      | 2      | 2    | 8        | **18.0** | 1%        |
| 17  | Add go-arch-lint to flake.nix devShell                                       | build         | 6      | 1      | 1    | 5        | **30.0** | 1%        |
| 18  | Run go-arch-lint on event/ → fix violations                                  | arch          | 7      | 2      | 2    | 5        | **8.75** | 1%        |
| 19  | Run go-arch-lint on command/ → fix violations                                | arch          | 7      | 2      | 2    | 5        | **8.75** | 1%        |
| 20  | Run go-arch-lint on decider/ → fix violations                                | arch          | 7      | 2      | 2    | 5        | **8.75** | 1%        |
| 21  | Add go-arch-lint CI step                                                     | ci            | 8      | 1      | 1    | 6        | **48.0** | 1%        |
| 22  | Create middleware/healthcheck.go — HealthCheck type + handler                | ops           | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 23  | Add health check middleware unit tests                                       | ops           | 7      | 2      | 1    | 6        | **21.0** | 1%        |
| 24  | Create /health/live endpoint helper                                          | ops           | 6      | 1      | 1    | 5        | **30.0** | 1%        |
| 25  | Create /health/ready endpoint helper                                         | ops           | 6      | 1      | 1    | 5        | **30.0** | 1%        |
| 26  | Create middleware/metrics.go — HTTP metrics handler                          | ops           | 7      | 2      | 1    | 6        | **21.0** | 1%        |
| 27  | Add metrics handler tests                                                    | ops           | 6      | 2      | 1    | 5        | **15.0** | 1%        |
| 28  | Create pkg/gracefulshutdown with shutdown helper                             | ops           | 7      | 1      | 1    | 6        | **42.0** | 1%        |
| 29  | Add graceful shutdown tests                                                  | ops           | 6      | 2      | 1    | 5        | **15.0** | 1%        |
| 30  | Add operational endpoints example in example/user/                           | examples      | 6      | 2      | 1    | 5        | **15.0** | 1%        |
| 31  | Add rapid to decider/ go.mod + create property_test.go                       | testing       | 8      | 2      | 1    | 7        | **28.0** | 1%        |
| 32  | Add decider rapid invariant: deterministic decide (same input → same events) | testing       | 7      | 2      | 1    | 6        | **21.0** | 1%        |
| 33  | Add decider rapid invariant: version monotonicity                            | testing       | 7      | 2      | 1    | 6        | **21.0** | 1%        |
| 34  | Add rapid to event/ go.mod + create property_test.go                         | testing       | 7      | 2      | 1    | 6        | **21.0** | 4%        |
| 35  | Add event rapid invariant: event immutability (clone ≠ mutate original)      | testing       | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 36  | Add rapid to id/ go.mod + create property_test.go                            | testing       | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 37  | Add id rapid invariant: ULID validity & prefix correctness                   | testing       | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 38  | Add go-snaps to integration/ go.mod                                          | testing       | 7      | 1      | 1    | 6        | **42.0** | 4%        |
| 39  | Create integration snapshot test for event JSON serialization                | testing       | 7      | 2      | 1    | 6        | **21.0** | 4%        |
| 40  | Create integration snapshot test for catalog AsyncAPI 3.0 export             | testing       | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 41  | Create integration snapshot test for catalog OpenAPI export                  | testing       | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 42  | Add go-snaps to catalog/ go.mod                                              | testing       | 6      | 1      | 1    | 5        | **30.0** | 4%        |
| 43  | Create catalog snapshot test for D2 diagram export                           | testing       | 5      | 2      | 1    | 5        | **12.5** | 4%        |
| 44  | Create catalog snapshot test for EventCatalog JSON export                    | testing       | 5      | 2      | 1    | 5        | **12.5** | 4%        |
| 45  | Add go-snaps to projection/ go.mod                                           | testing       | 5      | 1      | 1    | 4        | **20.0** | 4%        |
| 46  | Create projection snapshot test for state rendering                          | testing       | 5      | 2      | 1    | 4        | **10.0** | 4%        |
| 47  | Run all benchmarks → save benchmark-baseline.txt                             | ci            | 7      | 1      | 1    | 5        | **35.0** | 4%        |
| 48  | Add CI step: fail if any benchmark >2× slower than baseline                  | ci            | 7      | 2      | 1    | 5        | **17.5** | 4%        |
| 49  | Add benchmark comparison script (current vs baseline)                        | tooling       | 5      | 2      | 1    | 4        | **10.0** | 4%        |
| 50  | Add Dockerfile for example/user/ (multi-stage: builder → scratch → alpine)   | deploy        | 6      | 2      | 1    | 5        | **15.0** | 4%        |
| 51  | Add docker-compose.yml for example stack (user + memory store)               | deploy        | 5      | 1      | 1    | 4        | **20.0** | 4%        |
| 52  | Add Docker build CI step (linux amd64 + arm64)                               | ci            | 5      | 2      | 1    | 4        | **10.0** | 4%        |
| 53  | Add EventCatalog generation script to example/user/                          | docs          | 6      | 2      | 1    | 5        | **15.0** | 20%       |
| 54  | Create example/catalog-server/ with embed pattern                            | docs          | 6      | 2      | 1    | 5        | **15.0** | 20%       |
| 55  | Add HTTP handler for serving embedded EventCatalog SPA                       | docs          | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 56  | Add catalog server tests                                                     | docs          | 4      | 2      | 1    | 4        | **8.0**  | 20%       |
| 57  | Create middleware/sse.go — SSEBroker over event bus                          | examples      | 6      | 3      | 2    | 5        | **5.0**  | 20%       |
| 58  | Add SSE broker unit tests                                                    | examples      | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 59  | Add SSE handler to example/user/                                             | examples      | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 60  | Add SSE JavaScript client to example/user/ static files                      | examples      | 4      | 2      | 1    | 3        | **6.0**  | 20%       |
| 61  | Create pkg/config/ module — Config struct + validation                       | tooling       | 6      | 3      | 2    | 5        | **5.0**  | 20%       |
| 62  | Add YAML config loader with env-specific overlays                            | tooling       | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 63  | Add config module tests                                                      | tooling       | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 64  | Add config usage example in example/user/                                    | examples      | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 65  | Create integration/simulation/ — event sequence generator                    | testing       | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 66  | Add decider simulation benchmark (bulk aggregates, 1K→10K)                   | testing       | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 67  | Add event store throughput simulation benchmark                              | testing       | 4      | 2      | 1    | 3        | **6.0**  | 20%       |
| 68  | Add Playwright setup to example/user/ (package.json, config)                 | testing       | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 69  | Add Playwright E2E test: health endpoint reachable                           | testing       | 5      | 2      | 2    | 4        | **5.0**  | 20%       |
| 70  | Add Playwright E2E test: example/user/ core command→query flow               | testing       | 6      | 3      | 2    | 5        | **5.0**  | 20%       |
| 71  | Add Playwright CI step                                                       | ci            | 5      | 2      | 2    | 4        | **5.0**  | 20%       |
| 72  | Add runtime store factory to example/user/ (memory vs SQL)                   | examples      | 5      | 2      | 1    | 4        | **10.0** | 20%       |
| 73  | Add config-driven store selection in example/user/                           | examples      | 4      | 2      | 1    | 3        | **6.0**  | 20%       |
| 74  | Document experimental build tags (jsonv2, arenas, simd, runtimesecret)       | docs          | 4      | 1      | 1    | 3        | **12.0** | Remainder |
| 75  | Add jsonv2 codec experiment behind build tag                                 | codec         | 3      | 2      | 2    | 3        | **2.25** | Remainder |
| 76  | Add arena allocation experiment in event module                              | event         | 3      | 2      | 2    | 2        | **1.5**  | Remainder |
| 77  | Add rapid to command/ go.mod + create property_test.go                       | testing       | 4      | 2      | 1    | 3        | **6.0**  | Remainder |
| 78  | Add rapid to query/ go.mod + create property_test.go                         | testing       | 4      | 2      | 1    | 3        | **6.0**  | Remainder |
| 79  | Add go-snaps to signing/ module + snapshot tests                             | testing       | 4      | 2      | 1    | 4        | **8.0**  | Remainder |
| 80  | Add go-snaps to middleware/ module + snapshot tests                          | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 81  | Add go-snaps to storage/ module + snapshot tests                             | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 82  | Add go-snaps to snapshot/ module + snapshot tests                            | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 83  | Add go-snaps to listing/ module + snapshot tests                             | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 84  | Add go-snaps to schema/ module + snapshot tests                              | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 85  | Add go-snaps to memory/ module + snapshot tests                              | testing       | 2      | 2      | 1    | 2        | **2.0**  | Remainder |
| 86  | Add go-snaps to watermill/ module + snapshot tests                           | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 87  | Add go-snaps to pebble/ module + snapshot tests                              | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 88  | Add go-snaps to turso/ module + snapshot tests                               | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |
| 89  | Add go-snaps to codec/ module + snapshot tests                               | testing       | 2      | 2      | 1    | 2        | **2.0**  | Remainder |
| 90  | Add go-snaps to otel/ module + snapshot tests                                | testing       | 3      | 2      | 1    | 3        | **4.5**  | Remainder |

**Legend:**

- **Impact:** 1–10 (how much does this improve the library for consumers)
- **Effort:** 1–5 (1 = trivial, 5 = complex)
- **Risk:** 1–5 (1 = safe, 5 = may break existing code)
- **Cust.Val:** 1–10 (direct consumer-facing value)
- **Score:** `Impact × Cust.Val / (Effort × Risk)`

---

## D2 Execution Graph

```d2
direction: down

# Tier layers
tier1: "Tier 1% (51% value)" {
  style.fill: "#ffcccc"
}
tier4: "Tier 4% (64% value)" {
  style.fill: "#ccffcc"
}
tier20: "Tier 20% (80% value)" {
  style.fill: "#ccccff"
}
remainder: "Remainder (100% value)" {
  style.fill: "#ffffcc"
}

# Tier 1 nodes
docs_features: "FEATURES.md\n+ DOMAIN_LANGUAGE.md"
docs_context: "CONTEXT.md\n+ ROADMAP.md"
sec_gosec: "gosec scanning\n+ SARIF CI"
sec_arch: "go-arch-lint.yml\n+ CI step"
ops_health: "HealthCheck\n+ /live + /ready"
ops_metrics: "Metrics HTTP\nhandler"
ops_shutdown: "Graceful\nshutdown helper"
test_rapid_decider: "rapid PBT\non decider"
test_snap_integration: "go-snaps on\nintegration"

tier1 -> docs_features
tier1 -> docs_context
tier1 -> sec_gosec
tier1 -> sec_arch
tier1 -> ops_health
tier1 -> ops_metrics
tier1 -> ops_shutdown
tier1 -> test_rapid_decider
tier1 -> test_snap_integration

# Tier 4 nodes
test_rapid_event: "rapid PBT\non event + id"
test_snap_catalog: "go-snaps on\ncatalog exports"
test_snap_projection: "go-snaps on\nprojection"
ci_bench: "Benchmark\nbaseline + regression"
deploy_docker: "Dockerfile +\ncompose + CI"

tier4 -> test_rapid_event
tier4 -> test_snap_catalog
tier4 -> test_snap_projection
tier4 -> ci_bench
tier4 -> deploy_docker

# Tier 20 nodes
docs_embed: "Embedded\nEventCatalog server"
examples_sse: "SSE broker\n+ JS client"
tooling_config: "pkg/config\nmodule"
test_sim: "Simulation\nframework"
test_e2e: "Playwright\nE2E tests"
examples_dualstore: "Dual store\nruntime example"

tier20 -> docs_embed
tier20 -> examples_sse
tier20 -> tooling_config
tier20 -> test_sim
tier20 -> test_e2e
tier20 -> examples_dualstore

# Remainder
remainder_tags: "Build tag\nexperiments"
remainder_snap: "Extended\ngo-snaps coverage"
remainder_rapid: "Extended\nrapid coverage"

remainder -> remainder_tags
remainder -> remainder_snap
remainder -> remainder_rapid

# Dependencies (logical flow)
docs_features -> docs_context -> docs_embed
sec_arch -> sec_gosec -> ci_bench
test_rapid_decider -> test_rapid_event -> remainder_rapid
test_snap_integration -> test_snap_catalog -> test_snap_projection -> remainder_snap
ops_health -> ops_metrics -> ops_shutdown -> examples_sse -> examples_dualstore
tooling_config -> examples_dualstore
test_sim -> ci_bench
deploy_docker -> test_e2e
```

---

## Execution Order (Grouped by Sprint)

### Sprint 1: Trust & Documentation (Tasks 1–9, 10–15, 16–21)

**Goal:** Establish documentation discipline, security scanning, and architecture enforcement.

- All documentation files (FEATURES.md, DOMAIN_LANGUAGE.md, CONTEXT.md, ROADMAP.md)
- gosec integration + CI
- go-arch-lint configuration + CI

### Sprint 2: Operational Readiness (Tasks 22–30)

**Goal:** Give consumers production-grade operational middleware.

- Health check middleware + endpoints
- Metrics HTTP handler
- Graceful shutdown helper
- Operational endpoints in example/user/

### Sprint 3: Testing Rigor — PBT & Snapshots (Tasks 31–46)

**Goal:** Catch regressions and edge cases that unit tests miss.

- rapid property tests on decider, event, id
- go-snaps snapshot tests on integration, catalog, projection

### Sprint 4: CI & Deployment (Tasks 47–52)

**Goal:** Prevent performance regressions and provide container packaging.

- Benchmark baseline + regression CI
- Dockerfile + docker-compose + CI build

### Sprint 5: Consumer Experience (Tasks 53–73)

**Goal:** Show consumers how to use the library in real applications.

- Embedded EventCatalog server example
- SSE broker + example
- pkg/config module
- Simulation framework
- Playwright E2E tests
- Dual store runtime example

### Sprint 6: Polish & Experiments (Tasks 74–90)

**Goal:** Extended coverage and forward-looking experiments.

- Build tag documentation + experiments
- Extended go-snaps across all modules
- Extended rapid PBT across command/query

---

## Key Decisions

1. **No new modules for ops/config** — `middleware/healthcheck.go`, `middleware/metrics.go`, and `pkg/config/` live in existing modules. This avoids module proliferation while adding value.
2. **Examples drive features** — Every new feature (SSE, config, dual store, embedded docs) must have a working example in `example/user/`. If it can't be demonstrated, it doesn't ship.
3. **Property tests are additive** — `rapid` tests live alongside existing tests, never replacing them. They catch edge cases; table-driven tests catch expected behavior.
4. **Snapshot tests only for stable outputs** — AsyncAPI, OpenAPI, D2, and EventCatalog JSON exports are stable enough for snapshots. Event payloads with timestamps are NOT snapshotted.
5. **Security is non-negotiable** — gosec findings in core modules (event, command, decider, storage) must be resolved before any other Tier 2 work begins.
6. **go-arch-lint is advisory at first** — Run it, fix violations, then make it a CI gate. If it produces false positives, document them with `// nolint` annotations.

---

## Success Criteria

- [ ] `FEATURES.md` exists and lists all 22+ library modules with honest status
- [ ] `docs/DOMAIN_LANGUAGE.md` defines ≥20 domain terms
- [ ] `gosec` passes on event, command, decider, storage with zero findings
- [ ] `go-arch-lint` passes in CI with zero violations
- [ ] Health check middleware has tests and is used in `example/user/`
- [ ] Metrics handler has tests and is used in `example/user/`
- [ ] Graceful shutdown helper has tests and example
- [ ] `decider/` has `rapid` property tests with ≥2 invariants
- [ ] `integration/` has `go-snaps` snapshot tests for event serialization and catalog exports
- [ ] Benchmark baseline exists and CI fails on >2× regression
- [ ] `example/user/` has a working Dockerfile
- [ ] `example/catalog-server/` serves embedded EventCatalog SPA
- [ ] `middleware/sse.go` exists with tests and `example/user/` demo
- [ ] `pkg/config/` module exists with YAML loader and tests
- [ ] `integration/simulation/` exists with event generator and benchmarks
- [ ] Playwright E2E tests pass for `example/user/` core flow
- [ ] `example/user/` demonstrates runtime store switching (memory vs SQL)

---

_Plan created: 2026-06-08 00:08_
_Source: Comparative analysis of `~/projects/SEC` (production consumer) vs `go-cqrs-lite` (library)_
