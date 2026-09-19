# Dogfooding Self-Review - Final Status (Paused)

- **Date:** 2026-09-19 17:25 CEST
- **Session scope:** answer "How can go-cqrs-lite better use itself?" via audit + execute + verify
- **State:** PAUSED per user instruction. Awaiting direction.
- **Companion:** execution log `docs/status/2026-09-19_16-50_dogfooding-self-review-execution.md`, report `docs/reviews/2026-09-19_16-22_dogfooding-brutal-self-review.html`

---

## a) FULLY DONE

1. **Dogfooding audit** across the 94-module workspace, centered on "where does the
   library reimplement its own primitives?".
2. **Fix 1 - loopback adopts `dedup.Ring`.** `metaengine/irohengine/loopback`
   hand-rolled op-dedup as a map that reset wholesale at 10k entries (forgetting
   pre-reset op IDs and re-applying them on redelivery). The sibling `quic`
   transport already used `dedup.Ring` ("no reset gap").
   - `transport.go`: `dedupSeen map` -> `dedupRing *dedup.Ring`, new unexported
     `dedupCapacity = 10_000`.
   - `conn.go`: `markSeen` now `Has`/`Add` on the ring.
   - `dedup_internal_test.go`: rewritten to pin graceful removal of the oldest
     ID, recent-ID retention, and bounded growth.
   - `go.mod`: `dedup` promoted to a direct require (dep count 3 of budget 4).
   - **Verified:** `GOWORK=off go test ./...` green (2.0s) AND workspace-mode
     `go test ./...` green (2.1s).
3. **Fix 2 - `DeferClose` sweep, engine modules (9 sites, 7 files).**
   `metaengine/{enginetest,claimkit,pebbleengine,sqliteengine}`.
   - **Verified:** build green; `go test -short` green (claimkit 0.01s,
     pebbleengine 4.3s, sqliteengine 16.7s).
4. **Fix 3 - `DeferClose` sweep, queue engines (13 sites, 7 files).**
   `queue/{sqlite,mysql}/{facts,cancel,reads}.go`, `queue/sqlite/enqueue.go`;
   added the already-required `metaengine/v4` import.
   - **Verified:** `queue/sqlite` build + `-short` test green (545s);
     `queue/mysql` build green on re-run.
5. **Deliverable report** `docs/reviews/2026-09-19_16-22_dogfooding-brutal-self-review.html`
   built from the `html-report-kit` Bauhaus-dark template (self-contained,
   65/65 `<div>` balance).
6. **Two candidate fixes investigated and correctly REJECTED** (see c/d).

## b) PARTIALLY DONE

1. **Close-idiom unification.** 22 of 76 production sites converted (9 engine +
   13 queue). **54 remain**, concentrated where the helper is hard to import.
2. **Report accuracy.** The HTML report still cites "54 sites left" and the
   in-flight `goal-shaped-app`; it was written before the final unexport and
   before I saw the newer concurrent status reports. Content is directionally
   correct but not re-cut.
3. **Verification depth.** Per-module build/test done; the composed gates
   (`check-arch`, `check-duplication`, `api-stability TestEvery`, `#verify`) were
   NOT run.
4. **Concurrent-session coordination.** I noticed and flagged the overlap, but
   did not reconcile or hand off.

## c) NOT STARTED

1. Tier-0 `Closer`/`DeferClose` placement decision (owner/ADR).
2. `queue/postgres` close-idiom sweep.
3. Extract duplicated `scanScoredVector` / `scanJSONValues` /
   `sortAndPaginate*` into `metaengine`.
~~4. `check-arch` verification of the new `dedup` direct dependency in loopback.~~ done 2026-09-19 — arch green post-change
~~5. `check-duplication` run to confirm no new clone groups from this session.~~ done 2026-09-19 — 18:05
~~6. `api-stability` `TestEvery` (should be unaffected now the const is unexported).~~ done 2026-09-19 — zero drift
7. Quiet-window `nix run .#verify`.
8. Adding the dogfooding principle to `AGENTS.md`.

## d) TOTALLY FUCKED UP

Honest list; none caused data loss, but all are real.

1. **I edited `queue/{sqlite,mysql}/*` while another session was actively
   rewriting those same modules** (`queue/conformance/*`, `mysql/register.go`,
   `mysql/engine.go`, `open.go`, `ddl.go`). I did not check the working tree for
   active work before touching a shared area. Overlap risk was low and builds are
   green, but this was avoidable.
2. **I introduced an exported `DefaultDedupCapacity` const in a package covered
   by `cmd/api-stability`, which silently changes the API golden.** I only
   caught it while writing the caveats, then converted it to unexported and
   **shipped a transient build break** (`undefined: DefaultDedupCapacity` at
   `transport.go:129`) before fixing it. Sloppy sequencing on my own change.
3. **My first-pass agent audit contained false-premise candidates that I nearly
   acted on:**
   - "`queue/task` duplicates `id/`" - wrong; it is a purpose-built sortable ID
     with load-bearing tie-break semantics.
   - "`system.CachedEventStore` drops capabilities" - wrong; it forwards them,
     consistent with `event.DecorateStore`.
   - "319 close-idiom violations" - inflated; 255 are test files and the whole
     `defer` form is explicitly exempted by lint rule C015.
     This burned effort and could have caused churn if I had trusted the summary.
4. **Misleading early evidence gathering.** `git diff --stat` initially showed my
   changes as absent because the auto-commit daemon had already committed them;
   I momentarily misread that as "changes lost".

## e) WHAT WE SHOULD IMPROVE

1. **Verify every tool/agent claim at file:line before acting.** The three bad
   candidates all came from a summariser; a single `view` would have killed each.
2. **Check `git status` and recently-modified files before editing any shared
   module.** Concurrent sessions are the norm here; the repo even says so.
3. **Know the API-coverage surface before adding an exported symbol.** Look up
   whether `cmd/api-stability` lists the module first.
4. **Never leave a build break between edits.** Rename-const is a three-site
   operation; grep the symbol first, change all sites, then test.
5. **Run at least one gate per fix.** These were low-risk, but `check-arch` and
   `check-duplication` are cheap and directly relevant (new direct dep; dedup
   edits).
6. **For cross-module dependency changes, test BOTH `GOWORK=off` and workspace
   mode.** I did this for loopback; it should be the standing rule.
7. **Re-cut the HTML report after the final edits** so it does not cite a
   superseded state.
8. **State coordination explicitly up front**, not after the fact.

## f) Next things to get done (ranked, 40)

~~1. Reconcile the `queue/*` overlap with the session that owns the queue work.~~ done — moot: queue family shipped
2. Decide the Tier-0 `Closer`/`DeferClose` placement (ADR/owner).
3. Sweep `queue/postgres` close-idiom sites (same as sqlite/mysql).
~~4. Run `nix run .#check-arch` and confirm loopback's dep budget.~~ done 2026-09-19 — 18:05
~~5. Run `nix run .#check-duplication` (mind the dirty-baseline guard).~~ done 2026-09-19 — 18:05
~~6. Run `api-stability` `TestEvery` to confirm the const unexport made no golden change.~~ done 2026-09-19 — 18:05
7. Quiet-window `nix run .#verify`.
8. Re-cut `docs/reviews/2026-09-19_16-22_...html` with the final numbers.
9. Extract shared scan helpers into `metaengine` in a clean-tree window.
10. Extract shared pagination helpers (`SortPaginate` and siblings).
11. Kill the quic/loopback `DefaultDedupCapacity` split brain (one shared irohengine const).
12. Add a loopback-vs-quic dedup parity test.
13. Convert remaining `metaengine/enginetest` close sites if any regressions appear.
14. Sweep `storage/pebble/*` close sites once a Tier-0 helper exists.
15. Sweep `storage/snapshot_migration.go`, `storage/turso/indexing/advisor.go`.
16. Sweep `scheduling/sqlstore/*` close sites.
17. Sweep `projectionhost/sqlite_dlq.go` close site.
18. Sweep `stack/run_projections.go` close site.
19. Sweep `kv/*` and `cmd/*` close sites (5 files).
20. Add a cqrs-lint rule for hand-rolled dedup maps if F015-F017 do not cover them.
21. Add a dogfooding section to `AGENTS.md` ("use the primitive or fix its tier").
22. Add a TODO_LIST row for the Tier-0 helper decision.
~~23. Review `.art-dupl-baseline.json` after the concurrent dedup work lands.~~ done 2026-09-19 — 18:05
~~24. Check whether the concurrent `storage/pebble/command_store.go` change supersedes my finding.~~ done 2026-09-19 — moot
25. Confirm `queue/conformance/*` still passes against the swept engines.
~~26. Run `queue/postgres` conformance against in-repo PG after sweeping it.~~ done 2026-09-19 — 15:09
~~27. Make sure `example/goal-shaped-app` compiles and is added to `testModules`.~~ done 2026-09-19 — G-T23 [x]
~~28. Add `example/goal-shaped-app` to `go.work` if the concurrent session has not.~~ done 2026-09-19 — G-T23
~~29. Verify `flake.nix` testModules/lintModule reflect the new example.~~ done 2026-09-19 — G-T23
~~30. Add a recipe for "operator picks the engine" backed by the goal-shaped app.~~ done 2026-09-19 — recipes §2.39
~~31. Ensure the goal-shaped app has a `main_test.go` that runs the engine-swap demo.~~ done 2026-09-19 — G-T23
~~32. Run `#check-file-size` (my edits are small; confirm no offender growth).~~ done 2026-09-19 — G-T23
~~33. Re-run loopback tests under `-race`.~~ done 2026-09-19 — G-T23
~~34. Re-run queue/sqlite under `-race` after the sweep.~~ done 2026-09-19 — G-T23
~~35. Examine `middleware/retry.go` and `scheduling/scheduler.go` hand-rolled backoff vs `go-retry`.~~ done 2026-09-19 — G-T23
36. Evaluate `projectionhost/worker.go` backoff duplication.
37. Evaluate `metaengine/replicator.go` retry loop.
38. Evaluate `metaengine/dgraphengine/transaction.go` backoff.
39. Decide whether `any` in `metaengine.Execute` stays (rule A034 acknowledges it).
40. Add a short "dogfooding audit" cadence to the repo (e.g. quarterly review doc).

## g) Questions I cannot answer myself (3)

1. **Ownership:** should I reconcile/hand back the `queue/{sqlite,mysql}` overlap,
   or leave it entirely to the session that owns the queue work (and who is that)?
2. **Tier-0 close helper:** do you want the exported `Closer`/`DeferClose` added
   to a primitive module now, or is that gated to the v5 API window?
3. **Mode:** keep executing the remaining items (4, 5, 8-10, gates) after the
   concurrent sessions land, or stay paused and only report?

---

### Notes for the next reader

- My three fixes were absorbed into mixed `chore: auto-commit` commits
  (`02a133d7e`, `eea1c3c66`) alongside other sessions' work - there is no
  authored commit history for this session.
- Env chain required: host Go is 1.26.7 but the workspace needs 1.27.1, so every
  `go` command needs `GOTOOLCHAIN=auto`.
- `testAgents`/`nix run .#verify` were not run; this session stayed on
  per-module `GOWORK=off` verification to avoid the exclusive gate.
