# Queue/ Arc Execution — Mid-Flight Status (M1+M2 green, M3 blocked at tag gate)

**Date:** 2026-09-14 14:14 CEST

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (queue M4 — ADR-0134 tokens, T14 dep validation, FactTx/Watermarks, queue/mysql live-green; CHANGELOG 2026-09-19). Open remainder tracked in TODO_LIST "Durable Work Queue module": family tag wave (0/90 cut), README quickstart + conformance doc polish, PG `-race -count=2` symmetric leg. ARCHIVED.
> **Session:** executing the SUPERB plan
> `docs/planning/2026-09-14_12-45_SUPERB-cqrs-to-the-max.md` (T1–T24) under
> the owner's "GET SHIT DONE! The WHOLE TODO LIST!" directive.
> **Repos touched:** go-cqrs-lite (`/home/lars/projects/go-cqrs-lite`) only.
> **Overall:** M1 (contract + suite) and M2 (SQLite engine suite-green) are
> DONE and gated. The PG engine (T8/T9) also went suite-green on its first
> run. M3's tag wave is blocked on the clean-tree requirement mid-daemon
> sweep, and the verify-ci matrix re-run FAILED somewhere above the
> captured tail — not yet diagnosed (details in §d).

---

## a) FULLY DONE (verified this session)

1. **T1 — `queue/` contract module** (`github.com/larsartmann/go-cqrs-lite/queue/v4`):
   - `store.go`: `Store[T]` (25 methods, tq-faithful: Enqueue/ClaimDue/
     Complete/Fail/FailPermanent/Requeue/Heartbeat/Cancel family/
     MarkOrphaned/RescueDead/DismissDead/UpdatePendingPriority + reads +
     journal + watermarks), `Filter`, `Codec[T]`/`JSONCodec[T]`.
   - `task/` subpackage: `Task[T]`, `New[T]`+`Normalize`, `ID`/`NewID`
     (transcribed), `Status` state machine with `CanTransitionTo`.
   - `facts/` subpackage: `Fact`, `FactType` (donor-identical `task.*`
     wire names), `RequeueEvidence`, `ReprioritizeEvidence`.
   - `claim.go`: `Claim[T]` (task + LeaseUntil; ADR-0134 token seam
     documented). `errors.go`: sentinels incl. errorfamily-classified
     `ErrLeaseNotHeld` (`queue.lease_not_held`). `priority.go`: aging
     constants (3d/pt, cap 10).
   - Full G3 ceremony: go.work, flake testModules, api-stability slice
     - golden regen (7025 exports) + TestEvery, LAYER[queue]=3,
       DEP_BUDGET=1, module-map row, references/modules.md row, CHANGELOG
       part-1 entry, doc-check 1132 refs valid.
2. **T2–T4 — `queue/conformance` shared suite** (1,626 lines, 10 files,
   all under the 350-line gate): Harness{NewStore, Backdate} + Run;
   pins: transition matrix, roundtrips with fact trails, lease guards,
   heartbeat, cooperative cancel incl. reclaim-finalize, fencing stress
   (8 workers × 24 tasks), expiry reclaim + Released forensics,
   priority/delay/deps/aging order (incl. cap + stored-priority
   immutability), backoff ladder, DLQ exhausted/permanent, verbatim
   evidence, requeue no-burn, rescue/dismiss, dedup convergence
   (terminal keys forever), journal seq/tail/cursor, watermark
   monotonicity, filter/list/count/status reads.
3. **T5–T7 — `queue/sqlite` engine: suite GREEN** (M2): donor SQL
   transcribed (single-writer MaxOpenConns(1)+WAL, two-step claim with
   RowsAffected fence, nil-detail normalization, dedup re-check in tx).
   `go test` green incl. `-race -count=2`. Ceremony complete
   (LAYER=5, budget 2, rows, CHANGELOG part-2).
4. **T8–T9 — `queue/postgres` engine: suite GREEN on FIRST run**:
   SKIP LOCKED candidate select, FOR UPDATE attempt reads, pgx v5,
   `Open`/`OpenWithPool` (caller-owned pools NOT closed — tq lesson),
   storage mapping mirrors SQLite. Green via pgtestcontainer
   (`-tags integration`) incl. `-race`. Ceremony complete (golden 7025,
   layers, rows, CHANGELOG part-3).
5. **T10 — dedup seam DECISION (not code, by design):**
   `docs/planning/2026-09-14_queue-dedup-seam-decision.md` — the seam
   IS the native `DedupKey` contract (partial unique index, both
   engines, forever-suppression pinned); go-idempotency adapter
   REJECTED (TTL command-dedup ≠ forever entity convergence).
6. **T11 — priorities + aging: DONE as part of T1–T9** (constants in
   contract, order expression in both engines, ordering pins in suite);
   bands/clamping documented as app-layer omission in queue/doc.go.
7. Engine-owns-claim-SQL decision documented (queue predicate richer
   than `claiming.Spec`; growing Spec = the speculative-knob path G2
   forbids) in queue/sqlite/doc.go + CHANGELOG.

## b) PARTIALLY DONE

1. **T24 — verify-ci matrix: FAILED, undiagnosed.** The run (13:33→
   ~14:05) raced my live edits (doc.go/CHANGELOG mid-run). Tail shows
   everything from idempotency/kvstore down green; the failing module
   is above the captured output window. Needs a clean re-run + read the
   full log (it prints to the flake checkPhase).
2. **T12 — tag claiming v4.0.0: dry-run READY, blocked.** Dry-run
   attempt refused: working tree had uncommitted changes (daemon had
   not swept yet). Re-run `bash scripts/tag-release.sh claiming v4.0.0
   "..." --dry-run` on a clean tree, then the real tag. Note: claiming
   tag also wants sqlstore's replace pinned (T12.3) + standalone build
   gate (T12.4) + proxy probe (T12.5) — none started.
3. **T13 — cut queue v4.0.0: not started** (depends on T12; golden is
   current; docs rows done; CHANGELOG entries written; tag not cut).

## c) NOT STARTED

- **T14** DAG dep-gating ADDITIONS (deps table + NOT EXISTS already
  transcribed + pinned in both engines; remaining: enqueue dep
  validation, cycle rejection, unblock-bump design).
- **T15** ADR-0134 claim tokens (mint/renew/finalize predicates,
  theft detection, conformance pins).
- **T16** FactSink-in-tx capability + watermark API completion.
- **T17** MySQL dialect engine.
- **T19** example/taskmanager upgrade. **T18** metaengine read-adapter.
- **T20** PapDashboard evaluation. **T21** tq re-open ADR.
- **T22** doc/process tail (proposal P0 phrasing, cqrs AGENTS go.work
  drift, rejection-rule encode, plan annotation, final report+index).
- **T23** owner rulings batch (lint gci-vs-treefmt, OrderBy knob,
  rejection policy, wave timing).

## d) TOTALLY FUCKED UP (this session's own errors, all caught+fixed)

1. **The `journal` alias collision** (my design miss): I first shipped
   `queue/journal`, which claimed the repo-wide `journal` doc alias and
   broke doc-check (`journal.ReadAll` in core.md's cheat sheet resolved
   against MY package). Fixed by renaming to `queue/facts` and
   reverting my premature core.md edit (which the daemon had already
   committed — reverted via follow-up edit, not history rewrite).
2. **First conformance registration was garbage**: invented
   `dbOf`/`errLeaseNotHeld`/`facts_Reprioritized` phantom helpers and an
   `errSuppressed` tx-abort flow I then abandoned mid-file. Rewrote
   enqueue.go and conformance_test.go cleanly (internal test package
   reaching `st.db`).
3. **4 rounds of suite bugs** before green: forgot claims before
   Fail/Complete (3 subtests), dangling dep tripped FK NOT NULL (suite
   invented a dep the donor would also reject), parked-task re-claim,
   completing an unclaimed task, unscoped band-filter expectations.
   The ENGINE was right each time; the SUITE was wrong — the suite is
   new code, the engine is a transcription, and it showed.
4. **Engine bugs found by round 1–2**: nil `[]byte` detail bound as
   NULL against the NOT NULL column (fixed: empty-bytes
   normalization); `Claim.LeaseUntil` sub-ms precision mismatch with
   the persisted ms (fixed: truncate to stored ms).
5. **CHANGELOG header mangling**: my part-3 insertion replaced the
   part-1 header, briefly orphaning part-1's body under part-2's
   header and LOSING part-2's body. Restored by follow-up edits;
   final structure verified (part3/part1/part2 order, gates green).
6. **First PG-store file split** left a 390-line cancel.go (over the
   350 gate) — caught by the ratchet, split into cancel.go+details.go.
7. **verify-ci launched while still editing** — invalidated the run
   (see §b1). Should have frozen edits or waited.

## e) WHAT WE SHOULD IMPROVE

1. **Run the big matrix gates on a frozen tree** — I started verify-ci
   mid-edit; the failure may be pure contamination. Freeze → run →
   read.
2. **Write suite pins against the DONOR's proven test shapes first** —
   half my round-2/3 failures were pins the donor suite had already
   solved (cleanup-by-cancel, no-claim-no-finalize). Read the donor's
   test bodies before inventing pin sequencing.
3. **Check package-name alias collisions before naming subpackages**
   (`journal` was claimed by doc idiom; a 2-minute grep of doc-check's
   alias model would have caught it).
4. **The 30-line function gate is advisory-funlen only** — several of
   my functions (scanTaskRow, pins) likely add advisory lint findings
   in a repo whose lint is already owner-escalated; do not make it
   worse, and the baseline-growth question belongs in T23.
5. **Dedup/deps duplication between engines** is intentional (ADR-0007
   mirror pattern) but the art-dupl gate has not been run against the
   new modules — expect `//art-dupl:accept` annotations to be needed
   (queue/sqlite vs queue/postgres vs donor).

## f) NEXT (up to 50, ordered)

~~1. Re-run `nix run .#verify-ci` on a FROZEN tree; capture full log;~~
~~ triage any red module (T24.2).~~ done 2026-09-14 — 14-46 continuation; matrix evidence
~~2. Record T24 evidence (matrix output) into this arc's final report.~~ done 2026-09-14 — 14-46 report
3. `tag-release.sh claiming v4.0.0 --dry-run` on clean tree (T12.1).
4. T12.2: real tag claiming v4.0.0 via script (owner in loop — see Q1).
5. T12.3: push tag; bump sqlstore + example pins off the replace.
6. T12.4: strip sibling replaces in sqlstore/example; standalone
`GOWORK=off` builds green.
7. T12.5: proxy probe (`go get` claiming@v4.0.0 in a scratch module).
~~8. T13.1–T13.4: golden+TestEvery, doc rows (done), CHANGELOG (done),~~
~~ module-map/features check for queue trio.~~ done — verified at creation
9. T13.5: tag `queue/v4/v4.0.0`, `queue/sqlite/v4/v4.0.0`,
`queue/postgres/v4/v4.0.0` (batch-release.sh wave; owner in loop).
10. Push wave + `batch-release.sh --smoke-all` proxy checks.
11. Re-run `nix flake check` (vendorHash drift after go.mod changes).
~~12. Run `nix run .#check-duplication` over the new modules; add~~
~~ `//art-dupl:accept` notes where the mirror pattern is intentional.~~ done 2026-09-16 — 42 accepts, gate green (09-35 §a1)
13. Run `nix run .#verify` (full: race/coverage/doc gates).
14. Run `nix run .#check-coverage` — new modules need coverage floor.
~~15. Check golangci advisory count for queue/* modules vs baseline~~
~~ growth policy (lint-baseline equivalent — cqrs side).~~ done 2026-09-16 — queue family 241→0
16. T23: assemble the rulings batch (gci-vs-treefmt lint, OrderBy
knob, rejection-propagation policy, wave timing) for the owner.
17. T22.1: fix proposal doc P0 phrasing (claiming DONE + trim note).
~~18. T22.2: fix cqrs AGENTS go.work use-block drift (go-idempotency).~~ done 2026-09-16 — 7th-pass AGENTS truth pass
19. T22.3: encode the rejection-propagation rule (tq AGENTS).
~~20. T14.1: deps enqueue-time validation design (exists / self-dep).~~
~~21. T14.6: cycle-rejection policy + test (tq has none — new surface,~~ done 2026-09-19 — M4: ErrDanglingDep, cycles unrepresentable
~~ needs a small design note first).~~ done 2026-09-19 — declined with rationale (claim-time gating covers it)
~~22. T14: unblock-bump (ADR-0015 donor concept) evaluation for queue/.~~
~~23. T15.1: ADR-0134 adoption note (tokens day one — supersede owner~~ done 2026-09-19 — ADR-0134 Accepted
~~ string; Claim struct is the seam).~~
~~24. T15.2–T15.5: token column, mint (crypto/rand), token predicates in~~
~~ renew/finalize, theft error semantics, conformance pins.~~ done 2026-09-19 — M4 tokens shipped
~~25. T15.6: docs + golden regen.~~ done 2026-09-19 — M4
~~26. T16.1: FactSink-in-tx capability interface design.~~ done 2026-09-19 — queue.FactTx/FactSink
~~27. T16.2/T16.3: sqlite + PG in-tx external fact append.~~ done 2026-09-19 — M4
~~28. T16.4: watermark API completion (List/Set operator surfaces?).~~ done 2026-09-19 — Store.Watermarks
~~29. T16.5: conformance pin "no state change without fact" via sink.~~ done 2026-09-19 — M4 conformance + AssertFactSink
~~30. T16.6: ADR-0001-lineage docs.~~ done 2026-09-19 — M4 CHANGELOG/README
~~31. T17.1–T17.3: MySQL engine (DATETIME(3), two-statement claims via~~
~~ claiming.MySQLClaimSelect+StampLeaseMySQL — the ONE dialect where~~
~~ claiming/ statements may fit directly).~~ done 2026-09-19 — queue/mysql live-green (M4)
~~32. T17.4: nspawn/mysql integration run.~~ done 2026-09-19 — wired into VM/nspawn legs
~~33. T17.5: MySQL conformance green.~~ done 2026-09-19 — M4
~~34. T19: taskmanager example on queue/sqlite (the on-ramp).~~ done 2026-09-19 — substrate T22
~~35. T19.5: cqrs-lint V006 golden refresh after example dep changes.~~ done 2026-09-18 — goldens re-pinned
36. T18: metaengine read-adapter design note (read side ONLY).
37. T18.2–T18.5: projection folds, FilterSpec demo, bench, docs.
38. T20: PapDashboard usage read + mapping doc + gap list + verdict.
~~39. T21.1: parity checklist vs conformance suite coverage table.~~ done 2026-09-19 — semantic-diff: 1:1 core
40. T21.2: facade re-point spike in a tq branch.
41. T21.3: run tq's own conformance batteries against upstream queue.
42. T21.4: journal migration sketch (facts schema compat).
43. T21.5: tq re-open ADR draft + verdict criteria.
44. T22.4: per-landing skill/doc row checks (SKILL.md queue section).
~~45. T22.5: final arc status report + index row (this file's successor).~~ done 2026-09-19 — M4 report
46. Annotate the SUPERB plan file with milestone results (ANNOTATE
mode — never rewrite; plans are point-in-time).
47. Add queue/ sections to SKILL.md / references (consumer recipes).
48. tq-side: TODO_LIST row for the re-open ADR once parity evidence
exists (pool food, tq repo).
~~49. Update go-cqrs-lite AGENTS module count (88→91 go.mods).~~ done 2026-09-16 — 7th pass §a6
50. Post-wave: `nix run .#vulncheck` (per-module standalone builds).

## g) Questions for the owner

1. **Tag + push authority for the wave (T12/T13):** the standing rule
   is tags only via `scripts/tag-release.sh` with the owner in the
   loop, and no pushes by default. "GET SHIT DONE" green-lit building
   — does it also green-lit me CUTTING and PUSHING the v4.0.0 wave
   (claiming + queue + queue/sqlite + queue/postgres) autonomously, or
   do you want to run the tag/push step yourself after reviewing the
   dry-run?
2. **verify-ci red module:** the matrix run raced my edits and failed
   somewhere outside the captured output window. Do you want me to
   re-run and self-triage on a frozen tree (my default next step), or
   is there a CI/nix constraint (disk, cache) I should respect first?
3. **Dep-validation scope for T14:** the donor blindly inserts deps
   rows (FKs reject dangling ones on SQLite only). Should queue/
   standardize validation (all deps must exist at enqueue time, on
   every engine) — a small semantic CHANGE vs the donor — or keep
   donor-faithful blindness and only add cycle rejection?

---

_Evidence anchors: suite green `queue/sqlite` (plain + `-race -count=2`,
2026-09-14 ~13:20), `queue/postgres` (integration + `-race`, ~13:29);
api golden 7025 exports + TestEvery ok; doc-check 1132 refs valid;
layers + file-size + changelog-symbols + error-taxonomy gates green;
verify-ci ❌ (contaminated run, §d7)._
