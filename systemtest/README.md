# systemtest

Real-engine test suites for the `system` composition root (the Feedback-#4
split: these suites moved out of `system/` so that module's consumers pull
no engine implementations).

## What lives here

- `system_sqlite_test.go`, `integration_badger_test.go`,
  `integration_postgres_test.go` — system boots against real engines
  (ephemeral; Postgres via `nix run .#integration-pg`).
- `system_projection_test.go`, `matview_config_test.go` — projection
  planning and materialized-view config through the composition root.
- `snapshot_e2e_test.go`, `checkpoint_restart_test.go`,
  `durability_test.go` — lifecycle: snapshots, projection checkpoints
  across restarts, durability tiers.
- `evolutions_options_test.go` — EvolutionSpec plumbing.
- `time_fidelity_test.go`, `adapter_*_test.go` — event-adapter ordering and
  token fidelity.

Fixtures (`fixtures_test.go`) are the task-domain twin of
`system/system_test.go`'s block — test fixtures may not cross the module
boundary.

## Relationship to systemscenario

`systemscenario/` (ADR-0153) is the CONSUMER-FACING system-level BDD harness
(Given/When/Then over a `system.New` boot). This module is the repo's own
engine-matrix suites — infrastructure tests, not a consumer import target.

## Running

```sh
GOWORK=off go test ./... -count=1        # in this directory (memory/sqlite legs)
nix run .#integration-pg                 # ephemeral Postgres leg
nix run .#test-all-backends              # full backend matrix
```
