> **RESOLVED-BY-ROUTING — docs-health 10th pass (2026-09-21):** both Deferred items landed — the bench baseline was re-pinned under go1.27.1 the same evening (T18b: `a91e7cd90`, then the widened-gate closure chain; state in [TODO_LIST.md](../../TODO_LIST.md), Metaengine Universal Storage Substrate rows) and the stale taskmanager v4 tags were DELETED 2026-09-20 per owner ruling. The §Process launcher lessons shipped as M03–M05 (`--wait-loop`, `preflight-composed.sh`, verify load guard). Archived.

# Status Report 2026-09-20 16:39 — COMPOSED VERIFY GREEN (S03); W2 Tails Cleared; taskmanager v0.2.1 Shipped

> Continuation of the 13:21 report. The critical-path goal — a quiet-window
> composed `#verify` GREEN with the S03 record — is DONE, four verify
> attempts later. All numbers below verified from logs this session.

## The verify arc (13:21 → 15:04)

| Attempt | Window               | Result         | Root cause                                                                                                                                        |
| ------- | -------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 7       | 13:27–13:29 (killed) | Test FAIL      | api golden drift — the new `adttest.AssertTxIsolationFromForeignContext` export; I skipped the same-edit regen rule                               |
| 8       | 13:33–13:35          | Test FAIL      | `TestSystem_GracefulClose_ContextExpired`: genuine Go select race in `GracefulClose` (pre-cancelled ctx vs instant Close — uniform select choice) |
| 9       | 14:25–14:39          | Templ FAIL     | 5 `catalog/docserver/*_templ.go` drifted — W2-wave tail never verified (templ phase dark since before the wave)                                   |
| **10**  | **14:53–15:04**      | **GREEN rc=0** | **all 19 phases, 0 FAILs, ~11 min**                                                                                                               |

## What landed since 13:21

1. **W2 clone debt cleared** — tx-isolation scenario CONSOLIDATED into
   `metaengine/adttest.AssertTxIsolationFromForeignContext` (3×48 identical
   lines removed; the `AssertVectorDimensionGuard` conformance pattern),
   5 per-engine graph-SQL regions annotated `//art-dupl:accept` (dialect
   placeholders differ by design). Duplication gate: **0 new, baseline 60**.
   Verified green inside verify 10.
2. **`system.GracefulClose` determinism fix** (real product bug): a done-win
   now re-checks `ctx.Err()` so a pre-expired context always wins over an
   instantaneous Close. System suite green + 20× `-race` stress green.
   CHANGELOG Fixed entry.
3. **Templ regen** with the pinned CLI (v0.3.1020, via `nix develop`) —
   check-templ green.
4. **S03 recorded** in TODO_LIST (attempt 10: 14:53:28–15:04:25, all phase
   names, log pointer, fix trail). Authored commit `docs: record S03...`.
5. **`example/taskmanager v0.2.1` shipped** — the tag guard correctly
   refused a v4 tag (suffix-less module path; the existing v4.x git tags are
   proxy-invisible; the v0 line is the examples' convention — v0.1.0/v0.2.0
   served). Tag pushed, **proxy serves it**, clean-dir `go install` works,
   **`--help` exits 0** — the original bug fixed end-to-end.
6. **T13 load-sweep GREEN** ("all timing-assertion tests survived CPU
   load", run under fleet load 50).
7. **T15 verify-ci GREEN** (GOWORK=off per-module build+test, all modules).
8. CHANGELOG Added/Fixed entries for the adttest helper + GracefulClose fix;
   changelog-symbols gate ✓.

## Deferred (with reasons)

- **T14 bench-baseline supersede** — protocol requires calibration-gate PASS
  on a quiet host; the fleet ran 20–139 load all afternoon (wait-for-quiet
  timed out after 1h at load 29/40). Capturing under load would recreate the
  documented provenance gap. bench-gate is GREEN against the existing
  2026-09-11 baseline (verify 10 proof) — only its documented provenance gap
  remains. **Supersede at the next genuinely quiet window** (the 00:30–05:00
  band is historically reliable): `scripts/calibration-gate.sh` (must PASS,
  it's a script not a flake app) then
  `./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt`
  (writes the provenance header automatically).
- ~~**Stale `example/taskmanager/v4.*` git tags** (proxy-invisible): deleting
  remote tags is an owner call — added to the owner bundle questions.~~ **DONE 2026-09-20:** v4.0.0/v4.0.1/v4.1.0 deleted from remote+local per the W3 ruling.

## Corrections of my own earlier claims

- "bbolt is the one preset missing its doc.go deprecation" — WRONG: bbolt's
  package godoc (with the full v5 deprecation paragraph) lives in
  `preset.go`, not a separate `doc.go`. Semantically complete; no gap.
- The 13:21 report's "next: annotate the 2 W2 clone groups" — the tx group
  deserved (and got) consolidation, not annotation: `adttest` existed and
  the engines already import it. Annotation would have cemented a wrong
  structure.

## Process notes

- The canonical gate pair (`wait-for-quiet.sh` + `can-run-composed-gate.sh`)
  is two ONE-SHOT scripts; a load rebound between them aborts the chain.
  Retry loop supervisor is the working composition (used for attempts 9-10).
- Pre-flighting every dark phase before a verify run (bench-gate,
  api-stability, coverage, templ after attempt 9) is what made attempt 10
  the last one.

## Handoff state

- Tree clean, daemon absorbed everything; **9 commits unpushed** (pushed at
  close: see ledger). Verify 10 log: `/tmp/verify-attempt10.log`.
- Owner bundle: `docs/status/2026-09-20_11-36_owner-bundle-w3.md` (+ the
  stale-v4-tags question).
