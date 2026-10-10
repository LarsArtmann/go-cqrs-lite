# Tag-Wave Manifest: BDD Harness Adoption Wave (draft v1, 2026-10-09)

> Companion to [`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md) T02/T14.
> Status: DRAFT — re-verify the changed sets immediately before tagging (T14 pre-tag gate).

## Wave contents (tag order matters — dependencies below)

| Order | Module              | Version | Carries                                                                                                  | Gate evidence (2026-10-09)                                                                                                                    |
| ----- | ------------------- | ------- | -------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| 1     | `schema/v4`         | v4.6.0* | `schema.Event` + `schema.EventSchema` declaration API (concurrent session's work; system consumes it)    | standalone suite green GOWORK=off (0.016s); **untagged API consumed by system/schema_test.go — system cannot tag without this**               |
| 2     | `deriver/v4`        | v4.4.0* | `WithAsyncDispatch` (T08, additive; ADR-0154 decision b)                                                 | added at T08; systemscenario fixture (test dep) consumes it                                                                                   |
| 3     | `system/v4`         | v4.12.0 | `Clock`/`WithClock`/`ManualClock`/`RealClock` seam + schema wiring + `adapter_event_serial` (concurrent) | in-workspace suite green; GOWORK=off RED until schema tags (verified: published v4.5.2 lacks `schema.Event`)                                  |
| 4     | `scheduling/v4`     | v4.7.0* | `WithClock` (additive, self-contained `func() time.Time`)                                                | standalone + engine + sqlstore suites green GOWORK=off                                                                                        |
| 5     | `systemscenario/v4` | v4.0.0  | FIRST tag — the whole harness                                                                            | in-workspace race suite green; GOWORK=off RED until system+deriver tag (consumes `system.Clock`, `deriver.WithAsyncDispatch` in test fixture) |

\* version numbers to confirm against semver drift at tag time (schema/deiver/scheduling bumps are minor-worthy: new exported API).

## Dependency edges (why the order)

```
schema ──► system ──► systemscenario
deriver ─(test dep)──► systemscenario
scheduling (independent)
```

- `system/schema_test.go` references `schema.Event`/`schema.EventSchema` — published
  v4.5.2 lacks them (build failure verified GOWORK=off 2026-10-09).
- `systemscenario/scenario.go` references `system.Clock`/`WithClock`/`NewManualClock` —
  published v4.11.0 lacks them (build failure verified GOWORK=off).
- `systemscenario` fixture flips to `deriver.WithAsyncDispatch` at T09 → test-scoped
  require, must resolve at published tag.

## go.mod pinning at tag time

- `systemscenario/go.mod` must require `system/v4 v4.12.0` and the new `deriver/v4` tag
  (batch-release self-sources `scripts/go-env.sh`; verify pins post-tag with a scratch
  `go get module@tag` probe — F14.4).
- Companions' pre-tag replaces (4 files) drop AFTER the wave (T15).

## Full-suite evidence (F02.2, 2026-10-09)

| Module                | Mode                              | Result                                                        |
| --------------------- | --------------------------------- | ------------------------------------------------------------- |
| system                | in-workspace                      | ok 0.739s                                                     |
| system                | GOWORK=off                        | BUILD FAIL (schema API untagged — expected pre-wave)          |
| scheduling            | GOWORK=off                        | ok 1.008s                                                     |
| scheduling/engine     | GOWORK=off                        | ok 0.017s                                                     |
| scheduling/sqlstore   | GOWORK=off                        | ok 5.381s                                                     |
| scenario              | GOWORK=off                        | ok 0.150s (go.mod-only drift since v4.4.3 — no re-tag needed) |
| systemscenario        | in-workspace -race                | ok 2.044s                                                     |
| systemscenario        | GOWORK=off                        | BUILD FAIL (system Clock untagged — expected pre-wave)        |
| schema                | GOWORK=off                        | ok 0.016s                                                     |
| release-scripts smoke | `nix run .#check-release-scripts` | green                                                         |

## CHANGELOG section mapping (F14.5)

- `systemscenario/v4 v4.0.0` — Added: the system-level BDD harness (Given/When/Then,
  manual clock, journal/golden/equivalence assertions).
- `system/v4 v4.12.0` — Added: Clock seam (`WithClock`, `ManualClock`); schema
  declarations on every read path (concurrent work — coordinate wording).
- `scheduling/v4 v4.7.0` — Added: `WithClock`.
- `deriver/v4 v4.4.0` — Added: `WithAsyncDispatch` (ADR-0154).
- `schema/v4 v4.6.0` — Added: `Event`/`EventSchema` declaration API (concurrent work).

## Companion baselines (T03, 2026-10-09)

**cqrs-htmx** (26 modules): ALL GREEN. Pre-existing gap found and fixed during the
baseline: go.work lacked a `schema/v4` local replace while the local `system/`
replace consumes the untagged schema API — `dashboardui/systembridge`,
`examples/system-demo`, `systemadapter` were build-broken; replace added (same
local-dev pattern as the existing block), all three green after.

**go-appkit** (11 workspace members): 10 GREEN incl. `cqrs` (the harness pilot
module, 11.2s). `integration` RED on 2 tests — BOTH pre-existing and unrelated to
this wave: `TestGoModPinsMatchDocumentedPins` (undocumented `go-appkit/docs` family
module) and `TestDocsCompositionThroughAppkitService` ("SUPERB Docs E2E" content
mismatch). External owners; not touched. (go-appkit go.work also needed the
`schema/v4` local replace for the same system-consumes-schema reason; added with a
drop-with-the-pair note.)
