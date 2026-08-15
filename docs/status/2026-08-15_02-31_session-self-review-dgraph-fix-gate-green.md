# Session Self-Review + Status — Dgraph fix, gate GREEN, and what I forgot — 2026-08-15 02:31

> Brutal self-review of THIS session only (continuation of the standing
> READ→UNDERSTAND→RESEARCH→REFLECT→execute loop). Predecessor report:
> `2026-08-15_02-00_dgraph-journalreadof-fix-lint-summary-changelog.md` (same night, same
> session's work — this report adds the review layer the user asked for).

## a) FULLY DONE (verified, gates green)

1. **Dgraph `JournalReadFrom` fix** — positional skip semantics replacing `gt(seq)` filtering
   (sparse UnixNano seqs made every resume re-deliver the whole journal). Exact-count tests +
   `enginetest.RunStreamLogBackendTest` parity (Dgraph was the only StreamLog engine not
   running the shared suite). **24/24 PASS against live ephemeral Dgraph.**
2. **Shared-server test collision fix** — `RunStreamLogBackendTestIn(t, eng, col)` added;
   parity test uses `events_parity` because the ADT matrix also writes collection `events`
   on the same persistent Dgraph server.
3. **Doc-check 0 warnings** — stale shim import paths fixed in FEATURES.md +
   DOMAIN_LANGUAGE.md (root cause, not the flake invocation the prior session suspected).
4. **`#lint` summary line** — `✅ Lint: 76/76 modules clean` / `❌ findings in: <mods>`.
5. **`TestLayerScriptKeysMapToModules` meta-test** — found 3 dead LAYER keys
   (`metaengine/{adttest,enginetest,keycodec}` are packages, not modules) + a duplicate;
   removed; AGENTS.md module map corrected.
6. **CHANGELOG**: ADR-0128 extraction entry (was missing) + Dgraph fix entry (added during
   THIS self-review — see d5).
7. **`TestRedisStreamRoundtrip` re-verified live** (ephemeral Redis): PASS 0.51s.
8. **go.work → Go 1.26.6 unblock** — sibling `../go-codec` (parallel session, uncommitted)
   bumped to `go 1.26.6`, which broke every workspace command here; one-line go.work bump
   restored workspace builds (dev-only; module go.mod files untouched).
9. **Full verify gate GREEN**: `nix run .#verify` EXIT=0, 18/18 phases, 0 test failures,
   lint 76/76 (log: /tmp/verify-final2.log).
10. **TODO_LIST.md** — Dgraph item closed with DONE note; new M-item added (seq-carrying
    journal reads — see f3).
11. Session status report written (02:00 doc), now amended + superseded by this review.

## b) PARTIALLY DONE

1. **Go 1.26.6 story** — unblocked dev via go.work only. Per-module go.mod (1.26.5), CI
   toolchain pin, nixpkgs go version, `.go-version` files: NOT touched. Half-state, needs a
   decision (g2).
2. **Verify-gate exclusivity rule** — root-caused and recorded in the 02:00 status doc, but
   NOT yet an AGENTS.md gotcha (the place future sessions actually look). Listed in f11.
3. **Gopls diagnostics noise** — after the go.work bump, gopls floods `go-flightrecorder is
   not in your go.mod` (30 errors) across decider/projectionadapter etc. Builds and gates are
   green, so it's snapshot staleness (known class); not restarted, not fixed.

## c) NOT STARTED (deliberately deferred, no approval or out of scope)

1. Tagging engines v4.0.2 (×4) + watermill/v4.5.0 — needs explicit user instruction.
2. Removing the 5 temporary `replace` directives in system/go.mod + the ~49 stale indirect
   refs tidy sweep — gated on tags.
3. `nix run .#vulncheck` pre-tag checklist run.
4. v5 Phase 8 deletions + migration guide.
5. DuckDB/Row calibration benches; cqrs-lint per-module regression tests; `.golangci.yml`
   exclusion audit; go-codec repo scaffolding (sibling session's lane).
6. SA1019 → go-idempotency contract-suite migration question (still open from prior session).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Self-inflicted red gate**: I ran the 60–115s Dgraph soak suite CONCURRENTLY with the
   full verify gate. Result: benchkit timing tests failed (6.0s vs 5s bound; Duration=10ms
   abort test), Postgres tx timeouts, duckdbengine hit exactly the 5m timeout. Cost: one
   full wasted gate cycle (~25 min) + isolation re-runs to prove contention. I document
   load-sensitivity rules in AGENTS.md; I failed to apply them to myself.
2. **Multiedit orphan tail**: my stream_log.go edit's old_string ended at the loop header,
   leaving a duplicated function tail. Caught by immediate post-edit read, but it's exactly
   the class of bug `lsp_replace_symbol` (whole-function replace) exists to prevent.
3. **Multiedit partial failure**: assumed a `metaengine` import alias in the test file that
   didn't exist (1 of 2 edits failed). Batched edits without re-checking imports I'd just read.
4. **Wrote the status report BEFORE the gate finished** ("run in progress at report time"),
   then had to amend it with the real result. Premature documentation = premature green's
   doc-writing cousin.
5. **Forgot the CHANGELOG entry for the fix I shipped**: I added the ADR-0128 entry, claimed
   "CHANGELOG done" in the todo list — and only noticed during THIS self-review that the
   Dgraph JournalReadFrom fix (the session's headline deliverable!) had no entry. Fixed now.
6. **Unproven dismissal**: first `stack/bench` lint failure was waved off as "transient cache
   noise" after a clean re-run; plausible (parallel jobs shared /mnt/buildcache) but never
   root-caused. If it recurs, investigate, don't shrug.
7. **Burned a cycle on script invocation**: ran `bash scripts/ephemeral-dgraph.sh` directly →
   `dgraph: command not found` (binary only exists inside the nix app). Should have grepped
   flake.nix for the wrapper app first (it was already listed in the AGENTS.md quick ref).

## e) WHAT WE SHOULD IMPROVE (process, from this session's scars)

1. **Gate exclusivity**: no integration suites while `#verify` runs. Add to AGENTS.md.
2. **Reports after gates, never during** — a status doc is a point-in-time capture of
   FINISHED evidence.
3. **CHANGELOG discipline**: every user-visible fix gets its entry in the SAME change, not
   discovered in review. The verify gate can't catch missing prose.
4. **Prefer `lsp_replace_symbol` for whole-function edits** — the orphan-tail and
   partial-failure multiedits both vanish as failure classes.
5. **Reverse coverage for the LAYER map**: my new meta-test checks keys → go.mod, but
   nothing checks every go.mod dir HAS a LAYER entry (a new module without a LAYER key
   silently skips layer enforcement). Cheap to add, symmetric protection.
6. **Harness hardcoded collections**: the ADT matrix hardcodes `events` (and others) —
   fine for isolated-DB engines, a landmine for shared-server engines. Audit adttest
   scenario collections, or namespace them per factory.

## f) NEXT — up to 50, ordered by leverage

**Release (blocked on g1):**
1. Tag engines v4.0.2 (sqlite/badger/pebble/pg) + watermill/v4.5.0 (annotated, via scripts/tag-release.sh).
2. Drop the 5 temporary replaces in system/go.mod; re-verify system standalone (GOWORK=off).
3. `go mod tidy` sweep of the ~49 stale indirect shim refs; verify no-diff in go.sum noise.
4. Run `nix run .#vulncheck` + `#check-arch` as the pre-tag checklist.

**Toolchain:**
5. Decide Go 1.26.6: repo-wide adoption (go.mod sweep, CI matrix, nix go pin, .go-version) or sibling revert; then align.
6. Restart gopls / refresh snapshot after go.work changes (30-error flood is noise).
7. Add `nix fmt` to my end-of-session checklist (gofmt -l alone is weaker than gofumpt+golines).

**Metaengine correctness/depth:**
8. Seq-carrying journal reads — `JournalReadAllWithSeq` or `StreamLogEntry{Seq,Value}`; adapters resume on true engine seqs (sqlite global AUTOINCREMENT + positional `index+1` cache in `EventAdapter.lookupSeq` can duplicate entries when collections interleave). NEW TODO item from this session.
9. Dgraph `JournalReadFrom` deep-resume over-fetch (fetches afterSeq+limit then slices; fine now, cursor/offset later if journals grow).
10. Audit adttest harness hardcoded collections for shared-server engines.
11. Two-live-engine integration test (AddEngine + Backfill correctness).
12. Brute-force vector search on Pebble/bbolt (Vector ADT memory-only today).
13. Recursive CTE graph dispatch for PG/MySQL.
14. Recursive CTE optimization for deep SQLite traversals.
15. DuckDB (Columnar) 60s disk calibration — the exact-tie cell (2.65 vs 2.65).
16. SQLite/PG/MySQL Row-layout calibration (still analytical estimates).
17. Calibration-baseline CI regression check.
18. Layout long-horizon: fold-pipeline sync (Active+DualUse), async replication (Backup), role transitions, workload trace format, aggregate boundary config, per-fold mutex, multi-collection batch atomicity.

**Hardening / tests:**
19. benchkit timing tests: mark load-sensitive or compute bounds relative (TestRun_SQLite_DurationAborts failed 6.0s vs 5s under contention).
20. duckdbengine: 150s clean vs 300s timeout — split the soak or raise its per-package budget.
21. Reverse LAYER coverage meta-test (every go.mod dir has a LAYER entry).
22. cqrs-lint per-module regression tests (F004, F007, F009, F012, F017, F023–F029, B030).
23. `.golangci.yml` exclusion audit (system/ 20 linters, cmd/cqrs-lint/ 17, metaengine/ 24).
24. Real broker edges on watermill-redisstream: redelivery duplicates, consumer-group rebalance, message size limits (gochannel tests can't catch these).
25. macOS verification of scripts/ephemeral-pg.sh.

**Infrastructure polish:**
26. `#check-lint-config` + `#verify-ci` nix apps (mirror GH Actions GOWORK=off per-module).
27. Wire `#sweep` to pre-commit/cron.
28. Consolidate engine `register.go` boilerplate (7 modules).
29. ephemeral-dgraph.sh: document that direct `bash` invocation needs dgraph on PATH (or self-exec via nix), so the header's usage example stops lying.
30. AGENTS.md: add the gate-exclusivity gotcha (see e1).

**v5 (Phase 8):**
31. Delete stack.Materialize; 32. Delete storage.RelationalProjection + storage/view; 33. Delete graph.GraphProjection; 34. Delete stack.Bundle + 8 presets; 35. Delete stack.RunProjections; 36. Delete ADR-0126 compat shells; 37. Final v4.x patches + drop transport/http+grpc from registries; 38. v5 migration guide; 39. Cut v5.0.0.

**Docs/debt:**
40. Harvest remaining open items from 2026-08-14/15 status reports into TODO_LIST.md (docs-health HARVEST pass).
41. go-codec repo scaffolding (sibling lane; FEATURES/ROADMAP/SECURITY/CI).
42. Re-check `cmd/doc-check` warning count stays 0 in CI (regression tripwire?).

## g) QUESTIONS (cannot figure these out myself)

1. **Tag now or batch?** Engine v4.0.2 (×4) + watermill/v4.5.0 as a standalone release pass
   now, or batched with final transport/http+grpc v4.x patches? And if tagging: **is pushing
   tags + master to the remote authorized?** (Never tag/push without explicit instruction.)
2. **Go 1.26.6 direction**: adopt repo-wide in go-cqrs-lite (I bump go.work already; full
   sweep = go.mod files + CI + nix go pin), or park at 1.26.5 and ask the sibling go-codec
   session to hold its uncommitted bump?
3. **SA1019 exclusion permanence** (standing from prior session): keep the scoped
   `(middleware|idempotency)/.*_test\.go$` exclusion permanently, or migrate kvstore test
   matrices onto the go-idempotency contract suite before v5?

## Gate evidence (final)

- `nix run .#verify`: **EXIT=0, 18/18 phases, `✅ All verification checks passed`,
  0 FAIL, `✅ Lint: 76/76 modules clean`** (/tmp/verify-final2.log)
- dgraphengine vs live Dgraph: 24/24 PASS (/tmp/dgraph-fixed2.log)
- watermill Redis roundtrip vs live Redis: PASS (/tmp/redis-verify.log)
- `verify-docs.sh`: PASS (re-run after the CHANGELOG addition in this review)
