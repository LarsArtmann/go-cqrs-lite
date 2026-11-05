# Status Report — 2026-11-05 18:10 CET: SSE root cause landed, CI triaged, verify externally interrupted

Session context: resumed the parked session (park report 09:46) under the standing
"finish everything" directive. This report covers 09:46 → 18:10 CET only.

---

## a) FULLY DONE (this session)

1. **ESCALATED SSE bug hunt — root-caused and FIXED (M04's blocker).**
   - The dead-letter WARN (`cbor: cannot unmarshal positive integer into TaskUpdated.Title`)
     that drove the escalation was a **red herring**: `TestDepth_PoisonEventDeadLetters`
     deliberately journals a poison `task.updated` with `"Title": 42` and its WARN
     interleaves with parallel tests in the package log. No decode bug exists.
   - **Real bug 1 (notification loss):** `metaengine.ServeSSE` committed+flushed the 200
     header BEFORE `watcher.Watch()` registered with the store hub — a client unblocked
     at the flush could fold projections (memory engine = µs) before the handler
     goroutine registered; every notification in that window was lost. Both serve paths
     (plain + Last-Event-ID replay) now subscribe before header commit
    (`subscribeSSEHeaders` helper, sse.go).
   - **Real bug 2 (drop-newest):** the subscriber hub delivered into a cap-1 entry
     channel with drop-NEWEST semantics — a slow consumer's newest projection state was
     discarded (attempt-8 signature: `ship it` streamed, `renamed` vanished), inconsistent
     with `forwardWithDropOld` downstream. Now latest-wins (`sendLatest` evicts oldest).
   - **Deterministic regression tests:** `TestServeSSE_SubscribesBeforeHeaders`
     (header-commit probe writes through the store at the exact commit moment),
     `TestSubscriberHub_NotifyLatestWins`. Repro evidence: pre-fix 3/20 failures,
     post-fix 30/30 green; metaengine full package green.
   - **Attempt-8's third shape** (`TestProperty_ReadModelMatchesFold` one event behind at
     the 5s deadline under -race load): harness await budget relaxed to 30s under
     `RaceEnabled` (scenario.go `defaultConfig`), mirroring the 2026-10-27 SSE await
     precedent. `WithAwaitTimeout` stays authoritative. systemscenario suite green.
   - TODO_LIST recurrence row (2593) closed with the full root cause.
2. **CI run 41978263864 — all 4 failure classes triaged, 3 fixed locally (M05 work).**
   - **Verify-fast gate (tursogo pins ahead of constant):** ran the runbook step-1
     ivmrepro suite on the v0.8.3 pin — all three grouped-view defects still reproduce
     (~173s, defect C at the full 27k wall) — advanced `metaengine.TursoGoIVMVerifiedThrough`
     v0.8.2→v0.8.3, `TursoGoIVMLastVerified`=2026-11-05, swept all 6 live citation files,
     `#check-turso-version` green.
   - **actionlint+shellcheck (SC2012):** `erraudit-sweep.sh` `ls|sort|tail` → `find`;
     shellcheck style-level clean via nix shell.
   - **Nix Flake Check (SQLITE_BUSY under shuffle seed 119160053384665):** root-caused as
     the BUSY_SNAPSHOT class — a DEFERRED tx that reads before writing gets an immediate,
     unretryable-by-busy_timeout BUSY when another connection commits between read and
     write upgrade. Fixed: `runImmediateTx` (BEGIN IMMEDIATE on a dedicated pooled conn,
     cancel-stripped COMMIT/ROLLBACK) wired into `RunInTx` and the planned-table
     read-modify-write path (`txExecutor.tx` widened to `metaengine.SQLExec`). Not
     locally reproducible (50 shuffled runs green), so the fix is class-elimination, not
     repro-verified. sqliteengine full suite green + 3x full suite under -race green.
   - **MySQL QEMU (`ConcurrentClaimersDisjoint`):** classified INFRA flake — 5 pooled
     connections "reset by peer" at failure time, error `claimkit.ClaimDue: begin:
     invalid connection`; the same subtest passed in another group in 0.03s. Nothing to
     fix locally; next CI run is the receipt.
3. **CHANGELOG:** 4 Fixed entries + 1 Changed entry (turso range extension);
   `check-changelog-symbols.sh` green (40 citations).
4. **Gates re-run green after edits:** api-stability golden regen (6,440 exports),
   `#check-turso-version`, `#check-adr-numbering` (161 ADRs, next=0164),
   `#check-lint-config`, changelog-symbols, metaengine/systemscenario/sqliteengine suites.
5. **Lost drafts rebuilt IN-REPO:** `/tmp` was wiped externally mid-session (second draft
   loss); both ADR-0164 and M23 drafts reconstructed into `build/adr-0164-draft.md` and
   `build/m23-draft.md` (gitignored, tree-safe).
6. **M06/M07 prep:** wave versions re-derived (catalog v4.12.0+16 commits, kv v4.3.5+22,
   id v4.7.4+9, metaengine v4.20.0+90, sqliteengine v4.6.0+31, cmd/cqrs-lint v4.20.0+10);
   adttest/cap.go/query_config.go/buildKeyExtractor surface studied for M16–M19.

## b) PARTIALLY DONE

1. **M04 (verify green):** every known blocker is fixed and committed (daemon absorbed:
   9a91e323d, 63750e705, …); file-size ratchet miss (dx.go 443→444) caught by attempt-13a
   and fixed (condensed comment, back to baseline 443). **Verify attempt 13 is NOT
   concluded:** launch A lost its /tmp log to the wipe; launch B refused by the load
   guard (load spiked 5→40 in seconds); launch C fired at 17:58 CET in a quiet window
   and was **externally SIGINT'd ~18:08** mid-test-leg — 88 packages processed, ZERO
   real test failures observed before the interrupt (benchkit/api-stability FAIL lines
   ARE the interrupt). `go.work.sum` left with a benign +1 hash (not truncation).
   Needs one clean relaunch.
2. **M05 (CI green on master):** all local fixes committed; the receipt requires a push
   (new run supersedes the failed one via the concurrency group). Blocked on M04.

## c) NOT STARTED (blocked on M04→M05→M07 chain)

M06 pre-release ritual (changed-set per-module tests + fleet census) · M07 tag wave
(6 modules) · M11 cqrs-htmx view_cache deletion · M12 crm scorecard pin + WithMaxOpenConns
· M13 webphone lint hardening · M14 BuildFlow/Kith MaxOpenConns + read-storm re-measure ·
M15 remaining consumers · M16 ADR-0164 · M17 WithCapPerGroup/EnforceCapGrouped · M18
sqliteengine TrimGroupedCap · M19 adttest pin + capabilities regen · M21 GT_LINT_MODULES
app · M23 README SeqSeekable section (draft ready in build/) · M24 baseline.go
de-explosion · M26 pragma golden test · M22 final receipts/errata.

## d) TOTALLY FUCKED UP (own-process failures)

1. **Drafts lost to /tmp TWICE:** the first loss was already flagged as "session drafts
   belong in the repo", and I still wrote round 2 to /tmp. Only the second wipe moved
   them into `build/`. Should have been repo-side after loss #1.
2. **Verify-13a log written to /tmp** (wiped): a possibly-40-minute burn whose outcome is
   unknowable. build/-local logs are now the rule (practiced from 13b on).
3. **File-size ratchet miss (dx.go +1 line):** I edited a baselined 443-line file without
   checking the baseline — cost one verify launch. wc -l against the baseline belongs in
   my edit ritual for any file near 350.
4. **Changelog edit needed 3 attempts** (`cat -A` artifacts `$` pasted into old_string —
   copy discipline slip).

## e) WHAT WE SHOULD IMPROVE (structural)

1. **External interference hardening:** THREE independent external events hit this
   session — (i) /tmp wipe destroying logs+drafts, (ii) the CI benchmarks job cancelled
   externally at 10:09Z, (iii) SIGINT to my in-flight verify at ~18:08. Verify burns are
   expensive; long runs should detach (setsid/nohup) and log repo-side.
2. **CI benchmarks job needs `timeout-minutes`:** a 2h+ informational job held the run
   conclusion hostage until externally cancelled.
3. **Verify-launcher should be a script, not a hand-rolled watcher:** the counter-loop
   watcher + margin arithmetic + log location were re-typed four times this session.
   Rowed before; now with evidence it should ship (scripts/verify-launch.sh).
4. **Workstation service floor vs load ritual:** the host floor oscillates 8–45 from
   persistent services (crm-server peaked at 78% CPU — plausibly the M14 read-storm
   itself). The 15.5 margin ritual burned ~2h today. Needs a policy (justified ceiling
   burns vs quiet-window waiting) — question g1.
5. **SSE race relaxation is now harness-wide:** good, but the per-test RaceEnabled blocks
   in sse_test.go are now redundant — cosmetic cleanup candidate.

## f) NEXT (ordered; ~30 items)

1. Relaunch verify attempt 13 (quiet window; build/-local log; detached).
2. If green: push master → new CI run → M05 receipt (watch MySQL leg for the flake,
   Dgraph leg for tx-aborts, benchmarks for duration).
3. M06: changed-set per-module `GOWORK=off go test -count=1` over catalog, kv, id,
   metaengine, sqliteengine, cqrs-lint (+ dependents: system, systemscenario, stack).
4. M06: `bash scripts/census-fleet.sh`.
5. M07: re-derive semver per module from `git log <tag>..HEAD` (draft: catalog v4.13.0,
   kv v4.4.0, id v4.7.5, metaengine v4.20.1, sqliteengine v4.7.0, cqrs-lint v4.21.0 —
   metaengine may deserve minor for 90 commits; inspect before tagging).
6. M07: `batch-release.sh` wave; versions-manifest + README manifest; push tags;
   `go get` smoke per tag from /tmp; `check-versions-manifest`.
7. M11: cqrs-htmx — delete `usermgmt/view_cache.go` → `kv.NewCacheForViewStore`
   (+WithCacheCopyFunc), pin bump, run its suite.
8. M12: crm — `scorecard.modules` projectionhost pin, drop 7 waiver-dup baseline rows,
   re-measure.
9. M13: webphone — WithResponseHeader/WithTypedParam/multipart, drop WithServer
   suppression, catalog bump, adopt WithCapPerGroup after M17–M19 (or hand-trim stays).
10. M14: Kith CRM — MaxOpenConns raise + read-storm re-measure (also expected to lower
    the workstation load floor — see e2).
11. M15: BuildFlow cqrs-lint pin → wave tag + vendorHash.nix re-pin.
12. M16: land ADR-0164 from build/adr-0164-draft.md (+docs/README.md index row;
    check-adr-numbering before+after).
13. M17: metaengine cap_grouped.go — WithCapPerGroup, validateCapConfig ext,
    EnforceCapGrouped, CapGroupTrimmer interface, EnforceCaps dispatch.
14. M18: sqliteengine TrimGroupedCap (meta_map json_extract + planned-table column
    window-function SQL).
15. M19: adttest/grouped_cap.go pin; capabilities regen; recipes fence; api-stability;
    untested-exports; CHANGELOG/FEATURES; comment+close #114.
16. M21: GT_LINT_MODULES flake app.
17. M23: README SeqSeekableStreamLog section from build/m23-draft.md (+readme fence
    ratchet classification).
18. M24: baseline.go warn() de-explosion.
19. M26: NewSQLiteEngineFromDSNWith pragma-override golden test.
20. M22: final receipts + errata (go-jose no-op, crm baseline already empty).
21. Improvement rows: verify-launcher script; ignore-cascade tripwire;
    generated-artifact pre-push sweep; wedged-run watchdog; `nix run .#erraudit` wrapper;
    CI benchmarks timeout-minutes; sse_test.go redundant RaceEnabled cleanup.
22. Final full verify post-#114; completion report superseding 06:39/09:46.

## g) QUESTIONS (cannot resolve myself)

1. **External interference policy:** three events (host /tmp wipe, external cancel of
   the CI benchmarks job, SIGINT into my running verify) suggest a second agent session
   or owner action on this host/repo. Is concurrent-session operation expected, and may
   I harden long runs (detached setsid verify + repo-side logs) as the default?
2. **Load policy for verify burns:** the workstation service floor oscillates load 8–45
   (crm-server alone hit 78% CPU — plausibly the M14 read-storm). When the quiet-window
   watcher misses, do you want justified-ceiling burns (e.g. VERIFY_MAX_LOAD=40,
   documented per-run) or strict quiet-window patience even if it costs hours?
3. **MySQL QEMU recurrence bar:** if `ConcurrentClaimersDisjoint` fails again on the
   post-push CI run, do you want connection-invalid retry robustness in queue/mysql
   (claimkit begin path), or is QEMU-leg flakiness to be tolerated/re-tried as
   environment?

---

**Parked awaiting instructions.** Verify-13 requires one relaunch; everything up to the
tag wave is ready to execute the moment M04's green receipt exists.
