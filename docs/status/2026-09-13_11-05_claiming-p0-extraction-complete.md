# Status Report: claiming/ P0 Extraction — Complete (with parallel-session interference)

**Date:** 2026-09-13 11:05 CEST
**Session:** Complete the durable-work-queue P0 (extract lease-claim SQL core from `scheduling/sqlstore` into `claiming/`, wire it, delegate the timer store to it)
**Format note:** written as `.md` per explicit user instruction (repo default for status reports is styled HTML).

---

## Session premise — and the first surprise

The handoff described `claiming/` as _untracked work-in-progress_. Reality at session start: **the module had been deleted from the working tree** — auto-committed at 08:52 (`cbd011503`), then deleted and auto-committed again at 08:55 (`d9045b976`), with no documented rationale anywhere. No branch contained a newer version; the only version ever to exist was the buggy WIP. Since the tree was clean (recovery = purely additive), TODO_LIST and the queue planning doc still actively called for exactly this work, and the user's instruction was explicit, I recovered all 7 files from `cbd011503` and completed them.

**Lesson:** handoffs on this repo age in minutes. Re-verify the premise before anything else. (I did — this was the session's most important decision.)

---

## a) FULLY DONE

1. **Recovered `claiming/` from git history** (`cbd011503`) — 7 files, exactly as the handoff described (bug included).
2. **Fixed the compile bug** — `andSuffix(s.And)` → `andSuffix(s)` at `stmt.go:15/38/56` (signature takes a `Spec`).
3. **Fixed 3 latent test bugs** the never-compiled suite was hiding:
   - second claim ran at `now == leaseUntil` — the `lease <= now` predicate re-opens the row at equality, so the "fenced" assertion could never hold; now claims strictly inside the window with a comment explaining the boundary;
   - final round-trip assertion expected 1 row where 0 is provable (fresh lease fences "due", "future" never due);
   - hand-rolled `contains`/`indexOf` replaced with `strings.Contains`.
4. **`go mod tidy`** — go.sum generated; `modernc.org/sqlite v1.58.0` added as test dep (same version sqlstore pins).
5. **claiming module green standalone** — build, vet (incl. test files, integration tag), all tests, repeatedly.
6. **Workspace wiring** — `go.work` use-block, `flake.nix` `testModules`, `cmd/api-stability` modules slice (Layer 4 group).
7. **api-stability golden regenerated** — +14 claiming exports (6825 total), diff verified claiming-only; `TestEvery` meta-tests green.
8. **check-arch maps** — `LAYER[claiming]=4`, `DEP_BUDGET[claiming]=1` (with rationale comments); layer + budget checks pass.
9. **cqrs-lint module-catalog exclusion** — `claiming` added to `excludedModules` (found only because verify-fast failed on this meta-test — the fifth wiring point nobody's checklist mentions).
10. **sqlstore delegation** — all claim SQL now built via `claiming.Spec` + builders:
    - `claimStmt` → `PostgresClaimStmt`/`SQLiteClaimStmt`;
    - MySQL path → `MySQLClaimSelect` + `StampLeaseMySQL`;
    - `RenewLease` → `RenewStmt`;
    - `ensureLeaseColumn` → `claiming.EnsureLeaseColumn` (local copies deleted);
    - `timersSpec()` added as the single table-shape source.
11. **`sqlstore.Dialect` and `DefaultClaimLease` are now aliases** of the claiming definitions — the "values MUST match" duplicate and its `art-dupl:accept` directive deleted. Public API, `ClaimMetrics`, and error identity (`ErrClaimingUnsupported`, `ErrLeaseNotHeld`) unchanged.
12. **sqlstore go.mod** — `require claiming/v4 v4.0.0` + sibling `replace … => ../../claiming` with the standard tag-release-strip comment (same pattern as system/middleware).
13. **sqlstore suite green** — behavioral tests (fencing, expiry reclaim, renewal, restart, two-claimer contention on SQLite) pass repeatedly under GOWORK=off.
14. **Lint findings fixed (all real ones):**
    - claiming: `exhaustive` + QF1003 (tagged switch with all cases), `exhaustruct` (via `And: ""` in sqlstore's Spec literal), `stringsbuilder` ×2 (`strings.Join`), `sqlclosecheck` (repo's IIFE-defer pattern), 6 stale `//nolint:gosec` directives removed and 2 re-added exactly where gosec actually fires;
    - sqlstore: `wrapcheck` ×2 fixed with `%w` context wraps (errorfamily codes survive).
15. **migrate.go intra-file clone deduplicated** by extracting `addLeaseColumnIfMissing` — my art-dupl clone group eliminated (real extraction, not an annotation).
16. **Docs updated end-to-end:** skill `references/modules.md` row, `docs/agents/module-map.md` row, AGENTS.md (Tier 4 list + module count 84→85, verified by the doc-assertion gate), FEATURES.md module table row, TODO_LIST queue entry marked "P0 done", planning doc P0-landed note, `docs/error-taxonomy.md` claiming section + `GATED_MODULES` gate entry (163 codes across 6 modules green), CHANGELOG `[Unreleased]` Added section (37 symbol citations verified honest).
17. **Gates green for my slice:** doc-check (1055 references), arch/layers/budgets, changelog-symbols, error-taxonomy, file-size ratchet (green after the parallel session shrank store.go back), duplication for my groups, root workspace build (multiple runs), both module test suites, focused golangci per module.
18. **Everything committed** via the auto-commit daemon; `claiming/` fully tracked (8 files incl. go.sum).

---

## b) PARTIALLY DONE

1. **Composed `verify-fast`: never witnessed a full green run.** Every phase covering my work passed, but the run as a whole died three times on the _parallel session's_ live churn (system reset/replay flake under load; then `event/errors.go` mid-edit syntax error; finally `metaengine/catchup_state.go` referencing a not-yet-existing `s.checkUps` field at the vet phase). Needs one clean re-run once metaengine quiesces.
2. **Repo-wide `#lint`: red, and not by me.** gci was re-added to `.golangci.yml` formatters (by the parallel session, config churn visible in git log) and disagrees with treefmt's 3-group import layout **repo-wide** — 433 findings including files untouched for weeks. My files match the treefmt convention (`nix fmt` gate enforces it); the two formatters are irreconcilable at file level until the config gains gci `custom-sections` with the local prefix (or gci is removed again).
3. **MySQL claim delegation: byte-pinned, not live-verified.** Statement bytes are pinned by claiming's byte-exact tests and the wiring is compile-checked, but no live MariaDB run this session (`#integration-mysql-vm` not executed — heavy, needs QEMU).
4. **PG claim delegation: unit-verified only.** The pgtestcontainer suite in sqlstore didn't spin containers in my runs (suite finished in ~3–5 s); the PG statement is proven byte-identical, not live-executed.
5. **api-stability golden after final code shape: reasoned, not re-proven.** The post-lint refactors (RenewStmt switch↔if-chain↔switch, migrate.go extraction) changed no exported symbols, so the golden should be byte-stable — but I did not re-run `--update` after the last edits to _prove_ the empty diff.
6. **`go test -race` on claiming/sqlstore: not run** (part of full `#verify`, which the parallel churn kept killing).

---

## c) NOT STARTED

1. **Tag `claiming/v4.0.0`** — release flow (tag-release.sh, proxy check, smoke), then drop sqlstore's sibling replace and bump the require to the real version (per-module tidy — the missing `/go.mod` go.sum hash gotcha applies).
2. **queue/ module P1–P5** — lifecycle+backoff+DLQ+dedup, conformance suite, priorities/aging, DAG gating, same-tx journal, go-taskqueue adoption decision (TODO_LIST carries the entry).
3. **gci-vs-treefmt reconciliation** + updating the now-false AGENTS gotcha note that says gci was removed.
4. **Parallel session's cleanup** (not mine to start): 4 new art-dupl clone groups in `metaengine/*engine/reset*.go`, unfinished `catchup_state.go`.
5. **ADR for the extraction** — precedents (ADR-0065, ADR-0128) got ADRs; claiming has planning-doc + CHANGELOG coverage but no ADR. Possibly fine; owner's call.
6. **`nix run .#vulncheck` / `#check-coverage`** for the new module's dep tree and coverage drift baseline.
7. **TODO_LIST entry for the tagging step** — the queue entry says "P0 done" but nowhere is "tag claiming + strip replace" written down as its own task. The release is currently only in my head and this report.

---

## d) TOTALLY FUCKED UP (honest)

1. **I shipped a real double-fire bug for ~90 seconds.** When delegating, I passed `(leaseUntil, now)` to `SQLiteClaimStmt(s, now, leaseUntil)` — mirroring the _old internal arg-slice order_ instead of reading the new builder's signature. The builder reorders internally for `?1/?2`, so the lease got stamped with `now` (instantly expired) → rows immediately re-claimable → double-processing. sqlstore's behavioral tests caught it on the first run ("second claimer got 2 timers while leases fresh"). **Root cause: transcribing old call shapes instead of reading the new signature.** This is exactly the bug class the byte-exact pins can't catch — only behavioral tests can.
2. **I repeatedly piped test output through `grep | tail` and got burned** — masked exit codes, a "no output" confusion on the metaengine run, a false `dupl-exit:` empty. This is a _documented_ repo gotcha (pipeline masking). I violated it 2–3 times before switching to raw logs captured to files. Inexcusable; the rule exists in writing.
3. **False-confidence moment on the cqrs-lint meta-test:** I guessed the test name, got `ok … 0.004s [no tests to run]`, and briefly treated it as fixed. The output itself saved me. Rule: a passing test run must show it matched >0 tests.
4. **Whitespace churn on edits:** 2 edit-tool failures from transcribing comment text/diff context from memory rather than from the file (test-file comment; go.mod tail). Recovered each time by re-reading, but each was a wasted round trip that the workflow rules explicitly warn about.
5. **Never restarted the LSP** after adding claiming to the workspace — stale gopls diagnostics (BrokenImport, long-fixed andSuffix errors) polluted every single tool response all session. `lsp_restart` exists. I ignored the noise instead of eliminating it.
6. **Assumed golden stability instead of proving it** after the final refactors (see b.5). Cheap to verify, not done.

**Did I lie to you?** Not intentionally; two claims in my final summary deserve precision: "lint clean except systemic gci" is true _at module level for the two modules I focused on_ — repo-wide lint is red; and "everything committed" is the daemon's doing, not a reviewed commit series.

---

## e) WHAT WE SHOULD IMPROVE (process, from this session)

1. **Read the new signature before delegating a call site.** Parameter-order conventions differ between "returns args" builders and raw queries; never mirror the old call shape.
2. **Raw logs to files, always** — `go test … > /tmp/x.log 2>&1; echo exit=$?` — never `| grep | tail` for anything that gates a decision. (Documented gotcha; enforce on self.)
3. **Assert tests ran** (`[no tests to run]` = failure of verification, not success of code).
4. **New-module wiring has 6 touchpoints and only 4 have meta-tests** (flake testModules ✓, api-stability slice ✓, cqrs-lint catalog ✓, module count ✓; go.work-vs-everything partially; check-arch maps = none). One completeness meta-test over all six would have caught the catalog gap without a 5-minute verify-fast cycle.
5. **Restart the LSP when the workspace shape changes** (new module in go.work).
6. **Parallel sessions need a gate protocol:** my composed-gate attempts burned ~15 minutes racing another session's mid-flight edits (event syntax error, catchup_state.go). Either a shared gate lock, or the discipline to snapshot "my-slice green + interference documented" and re-run the composed gate once the other session quiesces.
7. **Docs/config split brain exists right now:** AGENTS (gotchas) says gci was removed from formatters; `.golangci.yml` has it back with 433 findings. Whichever way the reconciliation goes, the doc and the config must land together.
8. **The handoff's step 7 ("git add claiming/ — it will otherwise be lost") almost came true in the worst way** — the module was deleted _after_ being committed, and only git history saved it. The real fix is what finally happened: finish the work so the module has tests, wiring, and a changelog entry that would make silent deletion loud.

---

## f) Up to 50 things to do next

_My slice (release + proof):_

1. Re-run `nix run .#verify` (or verify-fast) once the metaengine session quiesces; record the first full green.
~~2. Re-run api-stability `--update`; confirm empty diff (proves b.5).~~ done — golden regenerated + TestEvery green repeatedly (7,092+ exports)
~~3. `go test -race` on claiming + scheduling/sqlstore.~~ done 2026-09-13/14 — sqlstore -race green; claiming via queue family -race
~~4. Run sqlstore's pgtestcontainer suite (Docker) — live PG claim proof post-delegation.~~ done 2026-09-16 — PG half (TODO_LIST [x])
5. Run `nix run .#integration-mysql-vm` — live MariaDB SKIP LOCKED proof post-delegation.
~~6. **Add a TODO_LIST item: "tag claiming/v4.0.0 + strip sqlstore replace + pin bump + per-module tidy"** (currently only in this report).~~ done — TODO_LIST row exists (tag-wave row)
7. Tag claiming/v4.0.0 via tag-release.sh (audit + proxy + smoke).
8. Drop sqlstore's sibling replace; require the real tag; per-module tidy (go.sum `/go.mod` hashes).
9. `nix run .#vulncheck` for claiming's dep tree.
10. `nix run .#check-coverage` — watch claiming's baseline (PG/MySQL paths untestable in unit tests).

_claiming module hardening (small, high-value):_
11. Validate Specs at builder entry (empty Table/IDColumn/… → `ErrInvalidSpec`); today a malformed Spec emits broken SQL silently.
12. Cheap guard: reject `Spec.And` containing `$`/`?` (documented prohibition, currently unenforced).
13. Pre-rendered statement cache (Spec+dialect → stmt string; only args vary) — `Due()` currently re-concatenates SQL every poll (parity with before, but a free win).
14. Optional `Spec.Limit` knob (batch claiming) — queue P1 will want it (go-taskqueue semantics).
15. Example test for SQLite client-side ordered-claim ranking (documented workaround, no example).
16. claiming README.md (check sibling-module convention).

_Docs truth:_
17. FEATURES.md `ClaimingTimerStore` feature row (Scheduling section) still narrates the SQL as sqlstore-owned — add "delegates to claiming/".
18. Update idempotency/sqlstore's Dialect duplicate comment to name claiming as the canonical definition.
19. Reconcile gci config (custom-sections with local prefix, or remove gci) and fix the AGENTS gci-removal gotcha note in the same change.
20. Decide on an ADR for the claiming extraction (precedent: ADR-0065/0128).
21. After config reconciliation, re-run focused lint on claiming/sqlstore — expect zero.

_Queue plan (the actual point of P0):_
~~22. queue/ P1: task lifecycle (pending→running→completed/dead) + attempts/backoff/DLQ + dedup'd enqueue (SQLite+PG).~~ done 2026-09-14 — queue contract + engines (CHANGELOG)
~~23. queue/ P1: mirrored conformance suite across dialects.~~ done 2026-09-14 — queue/conformance suite
~~24. queue/ P2: priorities + aging in claim order; MySQL dialect.~~ done — priorities/aging M1–M3; queue/mysql live-green 2026-09-19
~~25. queue/ P3: DAG dep gating (composite-key NOT EXISTS).~~ done 2026-09-19 — M4 T14 ErrDanglingDep
~~26. queue/ P4: same-tx journal option + watermark/cursor API.~~ done 2026-09-14/19 — journal + FactTx/Watermarks
~~27. queue/ P5: go-taskqueue adoption ADR.~~ done 2026-09-19 — T23 semantic-diff memo [x]
~~28. Owner-bearing claims / claim tokens (RenewLease ownership note in sqlstore).~~ done 2026-09-19 — M4 T15 ADR-0134 claim tokens
~~29. example/taskmanager upgrade to real queue consumer.~~ done 2026-09-19 — TODO_LIST [x] T22

_Meta / tooling:_
30. New-module wiring completeness meta-test (6 touchpoints, one check).
31. go-taskqueue consumer pin evaluation after queue lands.
~~32. cqrs-lint taskmanager version golden refresh at the tag wave.~~ done 2026-09-13/18 — goldens re-pinned

_Parallel session's outstanding items (observed, not mine):_
~~33. Finish/fix `metaengine/catchup_state.go` (breaks workspace vet as of 11:00).~~ done 2026-09-13 — catch-up observability (CHANGELOG)
~~34. Annotate or baseline-regen the 4 new art-dupl groups in `metaengine/*engine/reset*.go` (baseline regen needs a committed-clean baseline).~~ done 2026-09-13 — pareto 11-28 §a21
~~35. The `TestSystem_ResetProjection_RestartAndReplay` + metaengine root failures under parallel load — flake-class or real; passes standalone; needs a quiet-machine rerun to classify.~~ done 2026-09-19 — TODO_LIST [x] RESOLVED (ADR-0143)
~~36. Verify the staged-`.go` syntax pre-commit gate is installed in whatever flow produced the broken `event/errors.go` commit (the exact class it exists for).~~ done 2026-09-18 — TODO_LIST [x] pre-commit hardening
~~37. Re-run `nix run .#lint` after 17/33 land; expect only real findings.~~ done 2026-09-19 — TODO_LIST [x] lint zero

_Smaller observations from this session:_
38. Root-level `t/` and `result/` directories (taskd buffer, venv-looking tree) — confirm they are intentional/ignored, not junk accumulating at repo root.
39. Consider a `queue/` planning-doc pointer from claiming's package doc (currently only the reverse).
40. Document the claiming↔idempotency/sqlstore "intentional duplicate Dialect" relationship in one place (claiming.go mentions it; module-map could).
41. After tagging, run the GOWORK=off build matrix over swept modules (tag-wave mechanic 3).
42. Consider `RenewStmt` unknown-dialect behavior contract test (currently only commented).
43. claiming_test.go: pin the SQLite `?1/?2` arg ORDER explicitly in a comment next to the byte-exact test (the swap bug's natural antidote for future readers).
44. sqlstore: the two `%w` wraps I added ("sqlstore: ensure lease column: claiming: …") — decide whether double-prefixing reads fine or whether claiming should drop its own prefix when wrapped (cosmetic).
~~45. Watch file-size ratchet: claiming files all ≪350 today; queue code should start in new files, not grow claiming.go.~~ done — ratchet green (CHANGELOG 2026-09-15)
46. SKILL.md quick-reference: verify whether claiming belongs in the cheat-sheet (modules.md covered; cheat-sheet unchecked).
~~47. Re-verify TODO_LIST wording after queue P1 starts (the P0-done phrasing will age).~~ done — TODO_LIST queue section current (M4 state)
48. Consider running the docserver/EventCatalog checks if claiming ever appears in docs served to consumers (not yet).
49. Confirm go.work.sum committed entries for claiming are complete across all consumer modules (builds passed; one `go work sync` audit at tag time).
50. Post-tag: `go install github.com/larsartmann/go-cqrs-lite/claiming/v4@v4.0.0` clean-dir smoke (the v4.8.0 poisoned-tag lesson).

---

## g) Questions I can NOT figure out myself

1. **Who deleted `claiming/` at 08:55 and why?** Auto-commits show the files vanishing 3 minutes after first commit — user `trash`, or a parallel session's cleanup. My recovery assumed the instruction to finish P0 stood. If the deletion was a deliberate cancellation, say so and I'll unwind differently (though the completed, green, fully-wired state is now strictly better than what was deleted).
2. **Tag `claiming/v4.0.0` now, or hold until queue P1 exists?** Now unblocks dropping sqlstore's sibling replace and makes the module proxy-visible for go-taskqueue; holding means one tag wave covering both consumers. Owner's timing call.
3. **The gci-vs-treefmt conflict — what's the intended end state?** gci is back in `.golangci.yml` formatters (parallel session's churn) while AGENTS documents its removal, and the two tools disagree on import grouping for effectively every file in the repo (433 findings). Configure gci `custom-sections` with the local prefix, or remove gci again? And relatedly: **is the metaengine session still active right now** — should shared-gate work wait for it to finish?

---

_Point-in-time snapshot; expect the parallel-session items (33–37, 19/21) to change underneath this file. Next session should re-verify before acting on them._
