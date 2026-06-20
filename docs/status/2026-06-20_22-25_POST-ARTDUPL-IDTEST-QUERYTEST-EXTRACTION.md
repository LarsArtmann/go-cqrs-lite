# Status Report — Post `art-dupl` Deduplication (idtest / querytest extraction)

**Date:** 2026-06-20 22:25 CEST
**Scope:** `art-dupl --semantic --sort total-tokens -t 30 --html` + execution
**Branch:** `master` (4 commits ahead of origin)
**Working tree:** clean

---

## TL;DR

Ran the clone analysis, triaged all 73 groups, and eliminated the only **harmful**
duplication: the panic-on-error ID-parse helper family (`parseAggID` ×17 defs / ~136
call sites across ~20 files) plus `mustNewQuery`. Extracted two new shared test
packages — `id/idtest` and `query/querytest` — mirroring the established
`event/eventtest` pattern. **Zero harmful duplication remains.** Build, vet, tests
(all 37 modules), api-stability, and lint (on touched files) are green.

| Metric | Before | After | Δ |
| --- | --- | --- | --- |
| Clone groups | 73 | 64 | **−9** |
| Total clones | 186 | 140 | **−46** |
| Complexity score | 2.51 | 2.15 | **−14%** |
| Parse-helper groups | 9 | 0 | **gone** |
| `mustNewQuery` groups | 1 | 0 | **gone** |
| API exports | 1598 | 1605 | +7 (idtest×6, querytest×1) |

---

## (a) FULLY DONE ✅

1. **`art-dupl` analysis** — semantic scan at `-t 30`, HTML + JSON reports in
   `reports/art-dupl/` (`report.html`, `report.json`, `report-after.*`).
2. **Triage of all 73 clone groups** — every group classified extract/exclude/accept
   with rationale, documented in [`docs/planning/dedup-plan.md`](../planning/dedup-plan.md).
3. **`id/idtest` package** (NEW) — 6 typed `MustParse*` helpers
   (`AggregateID`, `EventID`, `CorrelationID`, `CausationID`, `UserID`, `RequestID`)
   backed by a private generic `must[T]`. Table-driven tests (happy path + panic cases,
   incl. the AggregateID string-backed quirk). Lint-clean.
4. **`query/querytest` package** (NEW) — `MustNew(query.Type)` + tests. Lint-clean.
5. **Migration of ~20 files** to `idtest.MustParse*`: `event`, `command`, `schema`,
   `signing` (incl. `internal/testutil/testutil.go` + `multisig/`), `snapshot`,
   `storage` (+ `memory/`, `pebble/`), `listing`, `watermill`, `integration`.
6. **Migration of 4 files** to `querytest.MustNew`: `query`, `integration` (×2),
   `middleware`.
7. **api-stability gate** — registered `id/idtest` + `query/querytest` in
   `cmd/api-stability/main.go`, regenerated `docs/api_surface.txt` (1605 exports).
8. **AGENTS.md** — module tree + Quick Reference updated with the two new packages.
9. **Deleted `event/test_helpers_test.go`** — it only contained the two helpers.
10. **Full suite green**: `go build ./...`, `go vet ./...`, `go test ./...` (0 FAIL),
    `api-stability` (OK 1605).

### Why `id/idtest` and not `testutil`?

`testutil` depends on `event` (Layer 1), so `event` cannot import it (cycle).
`id` is Layer 0 — every consumer already depends on it, and `id/idtest` is a
subpackage of the same module, so **no go.mod changes** were needed anywhere.
This mirrors the established `event/eventtest` shared-test pattern.

---

## (b) PARTIALLY DONE 🟡

- **Per-clone-group rationale**: the accept/extract decisions live in
  `docs/planning/dedup-plan.md`, not as inline code comments or an `.art-dupl.json`
  exclusion config. The 64 remaining groups are all accepted idioms, but a future
  `art-dupl` run will re-surface them as noise unless codified.

---

## (c) NOT STARTED ⬜

- **`.art-dupl.json` config file** — to permanently exclude the accepted idiom
  patterns (same-file self-matches, `if err != nil` blocks, struct literals) so
  future runs report only net-new duplication.
- **Duplicate-code ADR** (`docs/adr/`) — formalising the accept/extract policy so
  the "zero *harmful* duplication, not zero report lines" principle is durable.
- **`mustNewCmd` consolidation** — duplicated between `command/test_helpers_test.go`
  and `testutil/command.go`. Layering-constrained (command → ✗ → testutil → command).
  See open question (g).

---

## (d) TOTALLY FUCKED UP 💥

**Nothing.** Repo is clean and green.

One process note: the repo was **not** actually clean at session start (despite the
snapshot saying so) — a parallel session was mid-flight on a `reactive`-subsystem
removal and a `Decider.Fold → Apply` rename. Those changes (commits `1a851c6a`,
`ca29a723`) intermingled with this dedup work on disk. They were resolved coherently
by that session and committed alongside my dedup commits (`447b3ce9`, `3b59c128`).
No revert was needed; I left the foreign changes untouched per project policy.

---

## (e) WHAT WE SHOULD IMPROVE 📈

1. **Codify exclusions** — a `.art-dupl.json` would make `-t 30` runs signal-only.
2. **Surface `idtest`/`querytest` in `SKILL.md`** — the consumer guide doesn't yet
   mention them; consumers writing tests would benefit from knowing they exist.
3. **Generalise the `must[T]` primitive** — both `idtest` and `querytest` define a
   private `must[T any](v T, err error) T`. If a 3rd test-helper package appears,
   consider promoting it (though 2 copies of a 5-line private fn is fine).
4. **`nix fmt` runs emit collateral** — running the formatter touched 17 files with
   pre-existing formatting debt unrelated to the task. A pre-commit `nix fmt` check
  would keep this from accumulating.

---

## (f) TOP 25 — WHAT TO GET DONE NEXT (impact ↓)

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Resolve `mustNewCmd` duplication (see question g) | high | low |
| 2 | Add `.art-dupl.json` exclusion config for accepted idioms | high | low |
| 3 | Write duplicate-code ADR (`docs/adr/`) | med | low |
| 4 | Mention `idtest`/`querytest` in `SKILL.md` consumer guide | med | low |
| 5 | Raise art-dupl threshold guidance in CONTRIBUTING | med | low |
| 6 | Add `idtest` to the `testutil` README "see also" | low | low |
| 7 | Consolidate `signing/internal/testutil` to use `idtest` fully | low | low |
| 8 | Audit `example/*` for shared test helpers (currently self-contained) | low | med |
| 9 | Group 48 `startAggregateSpan` (pebble/otel.go vs sql/otel.go) — investigate | low | med |
| 10 | Promote shared `must[T]` if a 3rd consumer appears | low | low |
| 11 | Wire `nix fmt` into a pre-commit hook | med | med |
| 12 | Add `//nolint` audit pass for positioning after fmt | low | med |
| 13 | Coverage check on `idtest`/`querytest` (currently ~100%) | low | low |
| 14 | Document the Layer-0 test-helper pattern in AGENTS.md conventions | med | low |
| 15 | Push the 4 ahead commits to origin (after review) | med | low |
| 16 | Re-baseline `benchmark-baseline.txt` if allocs changed | low | med |
| 17 | Add a `make dedup` / `nix run .#dedup` wrapper for art-dupl | low | low |
| 18 | Review `catalog/internal/cattest` for idtest reuse | low | low |
| 19 | Check `decider` tests for parse-helper usage (none found) | low | low |
| 20 | Consider `eventtest.MustParseAggregateID` re-export for convenience | low | low |
| 21 | Add CI gate running `art-dupl -t 30` with fail-on-new-groups | med | med |
| 22 | Sweep for remaining per-module `parseXxxID` in examples | low | low |
| 23 | Document `didPanic` test helper idiom (inner-closure pattern) | low | low |
| 24 | Align `querytest` naming with any future `commandtest` | low | low |
| 25 | Update FEATURES.md with the two new test-helper packages | low | low |

---

## (g) MY TOP #1 QUESTION I CAN'T RESOLVE MYSELF 🤔

**Where should the canonical `mustNewCmd` live?**

It's currently duplicated: `command/test_helpers_test.go` (local, unexported) and
`testutil/command.go` (exported `MustNewCmd`). The natural fix mirrors what I just
did — a `command/commandtest` subpackage. But:

- `testutil` *already* exports `MustNewCmd` and several modules import `testutil`
  for it.
- `command` itself can't import `testutil` (cycle: testutil → command).
- Moving it to `command/commandtest` would deprecate the `testutil` copy and force
  consumers to migrate imports.

**I can't decide**: keep the `testutil.MustNewCmd` as the canonical home (and accept
that `command`'s own tests keep a local copy due to the cycle), OR establish
`command/commandtest` as canonical (consistent with the new `idtest`/`querytest`
precedent, but breaks the existing `testutil` API). This is a cross-cutting
test-helper-placement policy call that affects the public API and needs your steer.

---

## Verification Commands

```bash
nix run .#build                              # build OK
go vet ./...                                 # vet OK
go test ./... -count=1                       # 0 FAIL
cd cmd/api-stability && go run .             # OK 1605 exports
art-dupl --semantic -t 30 -j > /tmp/a.json   # 64 groups, 140 clones
```

_Arte in Aeternum_
