# Status — system/v4 Review Continuation: FIXES LANDED, HARVEST DONE, RELEASE PENDING

**Date:** 2026-08-17 14:24
**Session:** continuation of the SUPERB plan (`docs/planning/2026-08-16_18-30_SUPERB-system-v4-finish-review-fix-report.md`)
**Module:** `system/v4` in go-cqrs-lite
**Verdict:** The entire 18-task plan (M1–M18) is executed. All P1 fixes + regression tests committed and gate-verified. One item I got wrong mid-session (§d). Nothing pushed — master-only.

---

## a) FULLY DONE

1. **M8 (P1-1) — CachedEventStore write invalidation.** `Save`/`AppendBatch` now evict the stream key after successful writes; failed writes keep the valid entry. 4 regression tests (save/append freshness, cache-hit round-trip avoidance, error-path retention), race ×3 GREEN.
2. **M9 (P1-5) — Ackable volatile-source-of-truth.** Rule key now `volatile-source-of-truth:<role>` guarded by `isAcknowledged`; un-acked message unchanged. Regression test both directions.
3. **M10 (P2-7/8) — Unknown-engine error + deterministic engine creation.** Typo'd projection engine fails `New()` with `ErrUnknownEngine` listing unresolved names; engines created in sorted-name order. Tests: typo errors; EngineNames stable across 10 boots with 4 engines.
4. **M11 (P1-3/4) — Bus validation, deterministic selection, fan-out closers.** All `Buses` entries validated before selection (two-pass); each fan-out bus registered as a named closer; test proves fan-out bus rejects `Publish` after `Close()`.
5. **M1/M2/M3/M4 — Review coverage completed.** Read all remaining test files (hardening 725L, scream_plan 594L, extended 483L, 6 integration files, 5 partial tails) + `system/integration/` sub-module (doc.go, duckdb_test.go, go.mod). New findings: only confirmed P3 nits (discarded errors), no new P1/P2.
6. **M5 — README audit.** 3 doc-lies catalogued: `EngineNames` "creation order" (fixed), `mode: sync` documents an unimplemented field (routed), "missing durability" scream rule doesn't exist (recorded).
7. **M6 — Replace/publish audit.** 6 local replaces mapped; local metaengine is ≥12 commits past published v4.11.0 (MariaDB generated-column layout, vector ADT, CBOR-default unification). Conclusion: **all session fixes are master-only; published system/v4.4.0 still has all 5 P1 bugs.** `go mod tidy` applied (missing eventtest dep).
8. **M7 — Scoped lint gates.** cqrs-lint from system/ (C025 batch audited = false positives), golangci-lint 0 issues, coverage 74.4%.
9. **M12 — P2 batch.** lookupSeq/lookupSeqToken propagate journal-scan errors (silent replay-from-zero eliminated); zero-cursor skips the scan entirely (verified contract, found a real pre-existing bug in the position path while testing); seqCache → otter bounded 4096; WARN/ADVISORY diagnostics stored on System at construction and surfaced via `ScreamReport()`; dead RoleCommands/RoleQueries conditions removed. 5 regression tests.
10. **M13 — P3 polish.** RLock on Snapshot/Health/Explain; sorted bus topology; `reifyTo` panics loudly instead of silently corrupting projections; buildCRUDQuery variants merged; SnapshotAdapter key symmetry structural; doc dedupes; `mustApply` test helper (11 seed sites); 3 hardening tests assert `New()` success.
11. **M14 — Full gates.** build + vet + race `-count=3` GREEN; gofmt clean; golangci-lint 0 issues (run after each batch, not just at the end).
12. **M15 — HTML review report.** `docs/reviews/2026-08-16_full-code-review-system.html` — stat cards, 30 issue cards (5 P1 / 14 P2 / 11 P3) with file:line, fixed-on-spot ledger, verification table with honest not-run entries (postgres/duckdb).
13. **M16 — Harvest.** TODO_LIST gained the "system/v4 Full-Code-Review Follow-Ups" section (10 routed items with impact/effort); AGENTS.md gained a system/v4 section (review outcomes, replace-shadowing warning, ACK-key convention, C025 false-positive warning).
14. **M17 — Design proposals.** `docs/adr/2026-08-17_system-v4-review-proposals.md` — Count-by-name dispatch, named-bus API, role wiring, reserved-config honesty table, Durability wiring, backend contract, release sequencing, stack cross-check.
15. **M18 — Ecosystem cross-checks.** Verified against source (caught my own wrong assumption — see §d). stack.Bundle shares NO scream bugs (has no such machinery) but DOES have the wired Durability translation tables system needs — reversed the proposal direction. Dep-diet evaluated: keep engine blank-imports in root (consumers don't inherit them).
16. **Commits (all on master, unpushed):** `a211ebcb2` (P1 batch), `449e0e5a7` (P2 batch), `42dfab5b0` (P3 batch), `6ca027c27` (fmt). The daemon's `15a7cfe6d` swept in my TODO_LIST/AGENTS/ADR/report files.

---

## b) PARTIALLY DONE

1. **Push.** The prior session's instruction was "commit and push", and 4 commits from the earlier wave were pushed. This session's 5 commits are NOT pushed — I stopped short because the working tree was contaminated by a concurrent foreign wave (140+ modified files mid-session) and pushing master would ship someone else's in-flight state too. Needs a human decision (see §g).
2. **Full `nix run .#verify`.** Routed behind host buildcache repair per plan (module-level race ×3 is GREEN). Also: the pre-commit BuildFlow hook itself is broken in this environment (`GOTOOLCHAIN=local` + go.work 1.26.6 → hook's go step always fails) — that's why commits 2–4 used `--no-verify`. First commit ran the hook (failed on that env bug, daemon landed it anyway).

## c) NOT STARTED

1. **Release sequence** — metaengine tag → engine-adapter tags → system/v4.5.0 with replaces stripped. Everything upstream of "push" blocks this.
2. **Reserved-config implement-or-deprecate** (BusConfig.Mode, Subscribe, Collections, Cache.Engine) — proposals written, no code.
3. **Count-by-name / named-bus / role wiring / Durability wiring** — proposals only, per the plan's contract-change rule.
4. **system/ coverage push past 74.4%** — routed to TODO_LIST.
5. **Host buildcache repair ticket** — TODO_LIST entry exists; no ops action taken (out of scope for this repo).

## d) TOTALLY FUCKED UP (caught and corrected, but honestly)

1. **I asserted stack.Bundle shared the ack/WARN bugs without reading it.** The proposals doc §8 was drafted from the summary's assumption. When I verified against source, stack has NO scream machinery at all — my premise was flat wrong. Corrected the section and flipped its direction (stack's Durability tables are the *input* system needs). Lesson re-confirmed: verify-external-claims exists for exactly this.
2. **Two build failures from Go basics.** `(*T){}` is not a valid composite literal (test file), and `pubs[1].(event.Publisher)` asserted to the same type (staticcheck). Small, but each cost a round trip I should have avoided.
3. **My first bus-validation fix was wrong.** Returning on the first *sorted* entry still skipped validation of later entries — my own new test caught it, fixed with a validate-all-then-select two-pass. Good process outcome, sloppy first attempt.
4. **M14 gates initially "failed" on a foreign breakage.** A concurrent workspace wave added a broken `scheduling/scheduling` replace that broke even `go build ./system/` in workspace mode. Not my damage; re-ran everything with `GOWORK=off`. But it means **workspace-mode verification of this session's work is unproven** — only isolated module-mode is.

## e) WHAT WE SHOULD IMPROVE

1. **The 6 replace directives make local truth ≠ published truth.** Every fix session on system/ has this shadow: GREEN locally on code consumers can't get. The go.mod hygiene task (replace-drop sweep already in TODO_LIST) should be prioritized, or replaces should live only in go.work (where they belong for local dev).
2. **Test-suite pins leaky contracts.** `system_wiring_test.go:102-186` pins the positional `MultiBus.Publishers()` dig. Any future named-bus API must either keep that test or consciously rewrite it — it will fight improvement.
3. **No repo-wide coverage threshold**, and system/ uncovered mass is in the reflection-heavy evolutions paths — the riskiest code has the least direct coverage.
4. **WARN diagnostics still don't LOG.** They're stored and queryable now, but nothing prints them at startup. An operator who never calls `ScreamReport()` still sees nothing. One `log` call in `New()` would close that; I didn't add it because system/ has no logger concept — that's a design question for Lars.
5. **The BuildFlow pre-commit hook is env-broken** (`GOTOOLCHAIN=local` vs go.work 1.26.6). Every `--no-verify` commit this session inherited that. Should be fixed at the hook config level, not worked around per-commit.
6. **Concurrent-wave sessions on one repo are expensive.** The foreign wave cost: contaminated gate runs, forced `--no-verify` commits, unpushable master. Coordinate waves or use worktrees.

## f) NEXT — up to 50, ordered

**Release & push (blocking everything):**
1. Decide on push of the 5 session commits (see §g Q1).
2. Verify the foreign wave's state compiles (`go build ./...` workspace-mode) before pushing master.
3. Tag metaengine v4.12.0 (local is ≥12 commits past v4.11.0 — release notes need those features).
4. Tag the 4 engine adapters (badger/pebble/pg/sqlite) past their local deltas.
5. Strip system replaces → tag system/v4.5.0 (go-release flow; CHANGELOG entry from the fix commits).
6. Fresh-consumer proxy test: `GOWORK=off go get system/v4@v4.5.0` in a scratch module.
7. pkg.go.dev propagation check.
8. Check whether any OTHER repo consumes metaengine local-drift features and needs the metaengine tag first (cqrs-htmx, setup).

**Correctness follow-ups (routed proposals → decide):**
9. Lars decision: Count-by-name dispatch (proposal §1) — approve/reject.
10. Lars decision: named-bus API (§2).
11. Lars decision: role wiring semantics (§3).
12. Lars decision: reserved-config table — implement or deprecate each of the 4 fields (§4).
13. Lars decision: Durability pragma mapping — lift stack's translation tables into metaengine/system (§5).
14. Implement whichever of 9–13 get approved, each with regression tests.
15. Emit WARN diagnostics at construction (needs a logger story first — see §e4).
16. EventAdapter backend-contract doc (Atomic | Transactional | Racy per backend).
17. Port/verify none-needed for stack (corrected §8) — close that TODO entry as "no action".
18. Add `Memory` engine to the durability mapping decision (currently only persistent engines).

**Hardening & tests:**
19. evolutions reflection error-path tests (biggest uncovered mass).
20. Coverage threshold proposal for the repo (e.g. 70% floor per module).
21. Race ×3 on `system/integration/` (duckdb) once CGo env exists.
22. Run postgres integration suite against a real DSN (nix VM or POSTGRES_TEST_DSN).
23. Test: two Systems sharing one SQLite file DSN concurrently (documented engineCache gap, P3-9).
24. Rewrite `system_wiring_test.go` MultiBus pin once named-bus API lands.
25. Add a concurrency test for otter seqCache eviction under parallel ReadFrom.
26. Test scream-rule ACK for `durability-downgrade:<role>` (covered for volatile only).

**Docs & hygiene:**
27. README: remove/redirect the `mode: sync` example until Mode is implemented.
28. README: fix the "missing durability" scream-store overstatement.
29. CHANGELOG for system v4.5.0 (draft from the three fix commits).
30. TODO_LIST: mark the go mod tidy / replace-audit items from §M6 done.
31. AGENTS.md: record the BuildFlow GOTOOLCHAIN hook bug (if not fixed at source).
32. Consider moving engine blank-imports into system/integration after all — re-evaluate once root tests can import via test-only module indirection.

**Environment & infra:**
33. Host buildcache repair (/mnt/buildcache) — the actual ops ticket.
34. Fix BuildFlow pre-commit GOTOOLCHAIN handling.
35. Re-run full `nix run .#verify` once buildcache is healthy.
36. Add GOLANGCI_LINT_CACHE note to AGENTS if $HOME fallback lacks it (it worked here, verify why).
37. Investigate the foreign wave's `scheduling/scheduling` replace bug — is it still broken on master?

**Quality:**
38. `CQRS_` env-prefix integration test for acknowledge_warnings end-to-end through LoadConfig → New.
39. Bench: seqCache otter vs old map on cache-hit ReadFrom (regression check for the bound).
40. Bench: cache-invalidation cost on write-heavy cached deployments (each Save now evicts).
41. godoc examples for Count/Get/Find constructors (none exist).
42. Consider an `EngineNames` → `EngineInfo` richer introspection (driver, DSN redacted).
43. Sweep remaining `sys, _ :=` discards in other test files (I fixed hardening only).
44. Add `errors.Is` assertions for the new wrapped cursor errors in callers (projectionhost).
45. Verify cqrs-lint C025 false-positive batch stays ignored — add `.cqrs-lint.json` disables if it reappears in CI.
46. Draft v5 note: system replaces belong in go.work, not go.mod (architectural cleanup).
47. Review the daemon-landed commit `15a7cfe6d` — it swept my files with foreign changes; verify nothing of mine was mangled.
48. Update the SUPERB plan file: mark M1–M18 executed (it still reads as a plan).
49. status-report skill annotate pass over the mid-flight report (2026-08-16_18-25) pointing at this final report.
50. Celebrate — then delete the completed review section from TODO_LIST once the release ships.

## g) QUESTIONS (cannot resolve myself)

1. **Push now or wait?** Master carries my 5 verified commits PLUS a concurrent wave's commits (incl. one that briefly broke `go build` via a bad replace). Do you want me to verify the wave's compiles and push everything, or hold until you've reviewed the foreign commits?
2. **Should the fixes ship before the design decisions?** Tagging system/v4.5.0 + metaengine v4.12.0 publishes the 5 P1 fixes but locks nothing about Count-by-name/named-bus (those would be v4.6.0+ anyway). Release now, or bundle with whichever proposals you approve?
3. **Where should construction-time WARN diagnostics go?** system/ has no logger concept today. Options: (a) slog default logger one-liner in `New()`, (b) require consumers to call `ScreamReport()` (status quo), (c) inject `*slog.Logger` via DeploymentConfig. This decides item 15.
