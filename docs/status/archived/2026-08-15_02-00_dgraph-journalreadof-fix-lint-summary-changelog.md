# Session Report — Dgraph JournalReadFrom fix, lint summary line, ADR-0128 changelog, meta-test — 2026-08-15 02:00

> Continuation of the standing instruction (READ → UNDERSTAND → RESEARCH → REFLECT → execute
> one step at a time → verify). Prior session delivered the first fully green verify gate since
> ADR-0128; this session worked the no-approval-needed follow-ups from that report.

## What was done (a–f)

### a. Doc-check warnings eliminated (root cause, not symptom)

The prior session planned to "remove stale probe paths from flake.nix" — recon showed the flake
invocation was already clean; the warnings (`cannot read ../../flightrecorder|retry`,
`no exports found in idempotency`) came from **stale import paths in living docs** pointing at
the deleted shim modules:

- `FEATURES.md`: idempotency + flightrecorder import lines → external paths (`go-idempotency`,
  `go-flightrecorder`, ADR-0128 note)
- `docs/DOMAIN_LANGUAGE.md`: dropped `flightrecorder/v4`, `idempotency/v4`, `retry/v4` from the
  module-import overview (kept `idempotency/sqlstore/v4` — that module still exists)

Result: `cmd/doc-check` now exits 0 with **0 warnings** (was 5), all 1020 references valid.
(`recipes.md` references were already correct external paths — untouched.)

### b. `#lint` summary line (flake.nix)

The lint app looped 76 modules and exited 1/0 with no summary. Now tracks per-module failures
and prints `✅ Lint: 76/76 modules clean` or `❌ Lint: findings in: <modules>`. Verified both
paths: the failure path surfaced a real (transient, cache-related) `stack/bench` typecheck
finding, and the success path after unblock.

### c. check-module-layers.sh meta-test (+ 3 stale keys removed)

New `TestLayerScriptKeysMapToModules` in `cmd/api-stability/main_test.go`: every `LAYER`,
`DEP_BUDGET`, `EXCEPTIONS` (key AND dep value), and `TEST_INFRA_MODULES` entry must point at a
directory with a `go.mod`. The script silently `continue`s on missing go.mod — deleted modules
quietly drop out of layer/budget enforcement.

**It found 3 dead keys on its first run**: `LAYER[metaengine/{adttest,enginetest,keycodec}]` —
those are packages inside the metaengine module (`metaengine/v4/adttest`), not modules.
Removed (plus a duplicate `LAYER[metaengine/projectionadapter]`), with an explanatory comment.
Also fixed the AGENTS.md module map that implied they were modules.
`bash scripts/check-module-layers.sh` + the meta-test + `TestExceptionsAreMinimal`: green.

### d. CHANGELOG [Unreleased] now describes the ADR-0128 extraction

Confirmed missing (the extraction commit `5127039da` had no entry). Added a `### Changed`
section: deleted shim modules + external versions, internal consumer migration, registry sweep,
Dgraph counter observability, idempotency cache tuning, and the LAYER key mis-spacing fix.
`scripts/verify-docs.sh` (single `[Unreleased]` check etc.): green.

### e. Dgraph `JournalReadFrom` off-by-one — FIXED, verified against live Dgraph

**Root cause is deeper than the TODO described.** The only production consumer,
`EventAdapter.lookupSeq`/`ReadFrom` (system/adapter_event.go), derives `afterSeq` from entry
**indexes** (`index+1`, `afterSeq+i+1`) — positional arithmetic that is only exactly right for
engines with dense per-collection seqs. Dgraph seqs are sparse UnixNano timestamps, so the old
`@filter(gt(seq, afterSeq))` re-delivered the **entire journal** on every resume (off by
~1.7e18, not 1).

**Fix** (metaengine/dgraphengine/stream_log.go): `JournalReadFrom` now skips `afterSeq` leading
entries — `first: afterSeq+limit` server-side, slice off `afterSeq` client-side (one round-trip,
no reliance on DQL `offset` semantics). Positional semantics match every dense-seq engine where
position == seq.

**Test hardening:**

- `TestStreamLog_JournalReadFrom`: exact counts now (skip-1 → 2 entries starting "v1";
  limit 1 → first entry; resume past end → empty)
- New `TestStreamLog_HarnessParity` wires `enginetest.RunStreamLogBackendTest` — dgraph was the
  only StreamLog engine NOT running the shared contract suite
- **Collision found and fixed en route**: the shared contract harness hardcodes collection
  "events"; on Dgraph the server persists across tests, so my parity test + the ADT matrix both
  wrote "events" → cross-engine divergence (6 entries vs 3). Added
  `enginetest.RunStreamLogBackendTestIn(t, eng, col)` (wrapper keeps old signature for the
  isolated-DB engines); dgraph parity uses `events_parity`.

**Live verification** (`nix run .#integration-dgraph`, ephemeral Zero+Alpha): first run failed
exactly as predicted (old code re-delivered everything; matrix divergence), final run
**24/24 PASS, 0 FAIL, exit 0** — including all StreamLog tests.

### f. Re-verified upstream claims (status reports are point-in-time)

- `TestRedisStreamRoundtrip` (claimed done in CHANGELOG): re-run against ephemeral Redis —
  **PASS in 0.51s**. The NATS corpse stub is indeed deleted.
- TODO_LIST.md updated: Dgraph item marked done; **new M-effort item added** — seq-carrying
  journal reads. The positional `index+1` seq arithmetic in `EventAdapter` also silently
  duplicates entries on sqlite when collections interleave writes (global AUTOINCREMENT seq ≠
  position). Proper fix = seq-carrying read API (`StreamLogEntry{Seq, Value}`).

### g. go.work Go 1.26.6 unblock (external, in-flight)

Mid-session, a **parallel session in `../go-codec`** (uncommitted working tree, 13 dirty files,
new `scripts/check-go-version.sh`) bumped that repo to `go 1.26.6`. Because go.work `use`s the
sibling, EVERY workspace command in this repo failed: `module ../go-codec requires go >= 1.26.6,
but go.work lists go 1.26.5`. All workspace gates (build/test/race/lint) were blocked.

Resolution: `GOTOOLCHAIN=auto` + proxy makes `go1.26.6` available on demand (verified), so
**go.work's `go` directive → 1.26.6** (one line, dev-only; per-module go.mod files stay 1.26.5,
zero consumer impact, CI's GOWORK=off per-module path unaffected). Workspace build green again.
The sibling's bump was NOT touched (not my change to revert).

## Honest failures / notes

1. First multiedit on stream_log.go left a duplicated function tail (old_string ended at the
   loop header) — caught by immediate review of the edited region, removed, build+vet clean.
2. First multiedit on stream_log_test.go assumed a `metaengine` import alias that didn't exist
   (1 of 2 edits failed; the tool reported it, re-applied correctly).
3. My parity test itself introduced the "events" collection collision on the shared Dgraph
   server — found by running the FULL dgraph suite rather than only my new tests, which is why
   full-suite verification after each change stays mandatory.
4. Ran two background gates concurrently (lint + dgraph suite) sharing /mnt/buildcache — the
   first lint run showed a transient `stack/bench` typecheck-only failure that vanished on
   re-run. Treat single-module typecheck noise from parallel cache access as suspect; re-run
   before investigating code.

## Gate results

- `cmd/doc-check`: exit 0, 0 warnings, 1020 refs valid
- `bash scripts/check-module-layers.sh`: exit 0
- `bash scripts/verify-docs.sh`: exit 0
- api-stability meta-tests (`TestEvery*`, `TestLayerScriptKeysMapToModules`,
  `TestExceptionsAreMinimal`): ok
- metaengine (GOWORK=off, root pkg): ok
- dgraphengine full suite vs live Dgraph: **24/24 PASS, exit 0**
- watermill `TestRedisStreamRoundtrip` vs live Redis: PASS
- api_surface.txt: no diff (golden tracks root packages only; enginetest sub-package symbols
  are not part of the tracked surface)
- Full `nix run .#verify`: **EXIT=0, all 18 phases, `✅ All verification checks passed`,
  0 test failures, `✅ Lint: 76/76 modules clean`** (log: /tmp/verify-final2.log).
  First run failed at the Test phase (benchkit ×3 + duckdbengine at exactly the 5m timeout) —
  traced to load contention from this session's own concurrent Dgraph soak; both modules pass
  in isolation (duckdb 150s vs >300s, benchkit 43.9s vs 133s). Exclusive re-run: green.
  **Lesson: never run integration suites concurrently with the verify gate — the gate's 5m
  per-package timeout and benchkit's timing-sensitive assertions (Duration=10ms abort) are
  load-sensitive.**

## Open items for the user

1. **Tagging** (still awaiting approval, unchanged): engine v4.0.2 (×4) + watermill/v4.5.0, <- OPEN. TODO_LIST 'Release / Tagging' + ROADMAP 'Open Questions' #1
   then remove the 5 temporary replaces in system/go.mod and tidy the ~49 stale indirect refs.
   Never tag/push without explicit instruction.
2. **Go 1.26.6 direction**: the sibling go-codec is mid-upgrade (uncommitted). Decide whether <- OPEN. ROADMAP 'Open Questions' #2
   go-cqrs-lite formally moves to Go 1.26.6 (per-module go.mod + CI + nixpkgs go pin) or the
   sibling bump lands back at 1.26.5. The go.work-only bump unblocks dev but is a half-state.
3. SA1019 exclusion permanence (from prior session, unchanged). <- OPEN. TODO_LIST 'v5 Unification Phase 8' (kvstore SA1019 decision) + ROADMAP 'Open Questions' #3

---

## Resolution (2026-08-15)

All 3 user-items routed: tagging -> TODO_LIST "Release / Tagging" + ROADMAP
Open Questions #1; Go 1.26.6 -> Open Questions #2; SA1019 -> TODO_LIST v5 +
Open Questions #3. Everything else in this report was verified done at the
time (committed across `7c0a62c98` + `2e9a2fc28`). Archived by the
docs-health annotation pass.
