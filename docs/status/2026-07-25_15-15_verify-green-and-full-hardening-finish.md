# Session Report — Verify Gate to Green + Full Hardening Finish

**Date:** 2026-07-25
**Scope:** Finish every open item from the 72h diff review + metaengine hardening session.
**Outcome:** `nix run .#verify` is GREEN (build + vet + test + race + lint + doc-check, 58 modules).

---

## What changed this session

### 1. Unblocked the RED verify gate (race flakes)

Two benchkit timing tests were flaky under the `-race` detector (5-10x slowdown):

- `TestRun_SQLite_DurationAborts` — 5s hang threshold exceeded (5.066s).
- `TestRunSoak_TrendsPopulated` — 1MB/iter heap-growth threshold exceeded.

Added a build-tag-gated `raceEnabled` constant (`benchkit/race_off.go` + `race_on.go`)
and made both thresholds race-aware. The tight 5s / 1MB guards still apply to normal
runs; under `-race` they relax to 30s / 16MB so they catch genuine hangs/leaks without
flaking. Both tests verified stable across 3 consecutive `-race` runs.

### 2. Fixed the idempotency `Record` split-brain (public API bug)

`idempotency/kvstore.Record` overwrote on every call (extending the TTL), violating the
documented `Store.Record` contract that `MemoryStore` and `sqlstore` already honor.
Changed `backend.Set` → `backend.SetIfAbsent` (the backend already required
`kv.ConditionalWriter`, which `CheckAndRecord` already used). This matches
`MemoryStore.Record` exactly (record-if-absent, no-op on existing, TTL not extended).

### 3. Regression tests for the hardening fixes (8 new tests)

| Test                                                              | Guards                                                  |
| ----------------------------------------------------------------- | ------------------------------------------------------- |
| `benchkit.TestScalingSweep_SynthesizesFailedWhenRunReturnsNil`    | sweep.go NPE: nil `*Result` → synthesized FAILED row    |
| `metaengine` `Cursor.Encode` specs (4)                            | error path String() swallowed; Encode() surfaces it     |
| `metaengine` `MapUpdate atomicity`                                | tx-wrapped update loses no concurrent increments        |
| `metaengine` `Multimap restart safety`                            | reopen DB → no (collection,key,seq) PK collision        |
| `metaengine` `Cross-engine reification`                           | SQLite `map[string]any` → typed struct via ExecuteTyped |
| `idempotency/kvstore.TestStore_Record_DoesNotExtendTTL`           | Record no-op on existing                                |
| `idempotency/kvstore.TestStore_Record_MatchesMemoryStoreContract` | Memory + KV behave identically                          |

### 4. Lint cleanup (7 issues → 0)

The race-flake fix let the gate finally reach the lint stage, which surfaced 7 issues:

- **metaengine (5)** — from the prior session's hardening: `reify.go` wrapcheck (wrapped
  json errors + added `fmt` import), `cursor.go` nlreturn (blank line before return),
  `cost.go` package godoc prefix, `sqlite_backends.go` unused `//nolint:wrapcheck`.
- **otel/setup.go (1)** — pre-existing `varnamelen` (`mp` → `meterProvider`).
- **stack/pebble/preset.go (1)** — 72h-window `gosec` G115 (added `//nolint:gosec` on
  the already-isolated `safeInt64` helper, per AGENTS.md convention).

### 5. Documentation

- `AGENTS.md` — recorded the unified idempotency `Record` contract and metaengine SQLite
  hardening facts (tx-atomic MapUpdate, restart-safe multimap seq, cross-engine
  reification, caller-owns-DB Close).
- `docs/planning/...idempotency-record-contract-design.md` — marked **Decided: Option A
  implemented**, resolving the open question (the backend already supports SetIfAbsent).

---

## Open questions resolved

1. **Verify gate RED under `-race`** → FIXED (race-aware thresholds).
2. **`metaengine sqliteEngine.Close()` no-op** → INTENTIONAL & documented ("caller owns
   the `*sql.DB`"); not a leak.
3. **idempotency contract decision** → Option A implemented (no-op-on-existing).

## Left intentionally untouched

- **Auto-commit daemon history** — per prior decision, left as-is (garbled messages,
  user's WIP mixed in). All WIP files compile; the full test suite covers them.
- **The idempotency `RefreshTTL` optional capability** (design note item 3) — not needed
  for the contract fix; remains a future enhancement if long-running-op lease extension
  is ever required.
