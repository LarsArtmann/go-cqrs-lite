# Work Record — reconstructed: orphaned daemon commit `cec9248da`

> **Reconstruction, not authoring (2026-09-13 08:42 CEST):** commit
> `cec9248dafdd63e711e21bfc15570c586922a0b7` (2026-09-09 02:19 +0200,
> "chore: auto-commit 3 changed file(s) (heuristic)") absorbed work from a
> session that never wrote its closing report — the provenance gap was filed
> from 04-35 §c1 / 01-38 §f8 and is closed here. Every claim below was
> re-verified against the 2026-09-13 tree, not taken from any lost session
> notes (there are none).

---

## What the commit contains (what / where / why)

| #  | Work                                                                                                                                                                     | Where                                                          | Why it exists                                                                                                                                                              |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Renamed-aggregate-code tripwire** — `TestNoRenamedAggregateFamilyCodeReappears` walks repo Go source and fails on any of 17 exact error-family code strings from the 2026-09-08 stream-vocabulary rename (E4). Exact-string table, deliberately NOT a substring sweep, so legitimate identifiers like the `listing.aggregate_projection` projection name do not false-fire. | `cmd/api-stability/aggregate_code_tripwire_test.go` (new, 96 lines) | The 17-code rename (17 codes, 9 modules) must not silently rot back; the api-stability golden alone cannot see string literals reappearing in test files.                 |
| 2  | **fix.go dedup** — `RemoveStaleInlineSuppressions` and `PlanStaleInlineSuppressions` shared identical group-then-order boilerplate; extracted `staleByFile()` (group stale entries by file) + `finalizeFixResult()` (deterministic Removed/Skipped/Files ordering). No behavior change. | `cmd/cqrs-lint/pkg/suppression/fix.go`                          | The remove/plan pair had drifted into a copy-paste clone; the extraction makes the shared pipeline visible (ADR-0069 helper spirit, at function granularity).               |
| 3  | **PG claiming test hardening** — `TestClaimingPostgres_RenewVsClaimRace` inlined goroutine bodies extracted into `pollAssertingLeaseHeld()` + `reclaimOnce()`; the `reclaims` counter changed from a plain `int` incremented from two goroutines to `atomic.Int32` — a REAL data race fixed, not cosmetics. | `scheduling/sqlstore/pg_integration_test.go`                    | The race test's reclaimer loop raced on the shared counter under `-race`; helpers make the poll-vs-reclaim phases readable.                                                  |

## Verified how (2026-09-13 re-verification, this session)

1. Tripwire: `cd cmd/api-stability && GOWORK=off GOEXPERIMENT=jsonv2 go test
   -tags "goexperiment.jsonv2" -run TestNoRenamedAggregateFamilyCodeReappears
   . -count=1` → **PASS** (0.09s).
2. fix.go dedup: `cd cmd/cqrs-lint && GOWORK=off GOEXPERIMENT=jsonv2 go test
   ./pkg/suppression/... -count=1` → **PASS** (package green after the
   extraction).
3. PG test helpers: `cd scheduling/sqlstore && GOWORK=off GOEXPERIMENT=jsonv2
   go vet -tags "integration goexperiment.jsonv2" ./...` → **compiles clean**
   with the integration tag; `pollAssertingLeaseHeld` (pg_integration_test.go:620)
   and `reclaimOnce` (pg_integration_test.go:649) present. Canonical
   behavioral verification = live-PG run (`nix run .#integration-pg`), which
   this reconstruction session did not re-run — compile + presence is the
   honest ceiling for a provenance record; the test is part of the standard
   PG leg and ran green in the 2026-09-11 CI triage sessions.

## Status

All three pieces are DONE and still in the tree unchanged in intent (fix.go
has evolved since — the extraction survived). No forward-looking items were
lost with the missing report: none of the three files carries a TODO, and
nothing in TODO_LIST traces to this commit's session. Provenance gap closed.
