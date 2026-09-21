# Survey: the Scan/Find 100-row default — consumer census + v5 options memo

> **Status:** SURVEY DELIVERED — decision RULLED 2026-09-21 (owner, in-session):
> **Option C** — flip to unbounded at the v5 cut + cqrs-lint nudge + optional
> operator ceiling. The two v4-safe add-ons LANDED 2026-09-21
> (`metaengine.WithDefaultLimit` store option — `WithDefaultLimit` plan option,
> and cqrs-lint **F031** `scan-without-limit`); the default flip itself is
> staged for the v5 branch (TODO G-T14 remains open for that step only).
> **Date:** 2026-09-21 · **Origin:** CV reflection doc §4.1 + goal-closure G-T14
> (F111/M20 of the
> [owner-unblock-trust plan](2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md))
> **Routing:** decision lands in the v5 ADR-0123 cut plan, not a standalone ADR.

## State of the default (verified 2026-09-21)

`TypedReader.Scan`/`ScanPage` cap at **100 rows** unless `WithLimit` is passed;
`WithLimit(0)` = unbounded. Set at `metaengine/typed_reader_scan.go:19` and `:270`
(`scanConfig{limit: 100}`); documented loudly since 2026-09-17
(`metaengine/scan_options.go:94-97`, the WARNING at `typed_reader_scan.go:11-13`,
FAQ, `readmodels.md`, `modules.md`). `system.Find` forwards no limit by default
(`system/runtime.go:131` appends `WithLimit` only when > 0), so it **inherits the 100
cap silently**.

## Consumer census

| Consumer                                    | Limit behavior                                          |
| ------------------------------------------- | ------------------------------------------------------- |
| `example/goal-shaped-app` (`app.go:209`)    | `Scan(ctx, WithFilter(...))` — no limit → capped at 100 |
| `benchkit` (`phases_metaengine_map.go:103`) | explicit `WithLimit(100)` — aware                       |
| `system.Find` consumers                     | inherit 100 unless they pass `WithLimit`                |
| CV (external)                               | hit the silent truncation; the finding behind this memo |

No in-repo consumer passes an explicit large limit; everyone either is capped
unknowingly or asks for exactly the default.

## Options for the v5 cut

1. **Keep documented-100.** Safest for memory (meta_map rows are decoded JSON);
   truncation stays silent-but-documented.
2. **Flip to unbounded (recommended).** Rationale: silent truncation is a
   **correctness** hazard (wrong answers, silently); an unbounded scan is an
   **operations** hazard (slow, big) — and the escape hatches for the latter
   (`WithLimit`, `ScanPage` + `WithCursor` keyset pagination) already exist and are
   the documented path for large collections. A v5 breaking cut is the one window
   where the flip is honest.
3. **Unbounded + operator ceiling:** add `metaengine.WithDefaultLimit(n)` at Store
   construction so an operator can cap worst-case memory deployment-side without
   code changes.
4. **Lint nudge either way:** a cqrs-lint rule flagging `Scan(...)` calls with no
   `WithLimit` in non-test code makes the choice visible at author time (pairs with
   option 2 or 3).

## Recommendation

Option 2 + 4 at the v5 cut; option 3 as a fast-follow if operators ask. The survey
census above is the evidence pack G-T14 asked for.
