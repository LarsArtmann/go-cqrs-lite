# Queue arc — T24 verify-ci triage: three fixes landed, NEW Reads flake in evidence re-run

**Date:** 2026-09-14 14:46 CEST
**Session:** ~14:15–14:46, continuation of the 12:50–14:14 queue arc
(SUPERB plan `docs/planning/2026-09-14_12-45_SUPERB-cqrs-to-the-max.md`,
prior report `2026-09-14_14-14_queue-arc-m1-m2-green-m3-tag-gate.md`).
**Scope this session:** T24 only — frozen-tree `nix run .#verify-ci`,
triage, fixes. T12–T23 untouched (owner's standing WAIT from 14:14 was
superseded mid-session by an explicit "execute everything, keep going"
directive; this report is the ordered stop).

## a) FULLY DONE (this session)

1. **Session-start ritual** both repos (git log/status/stash, tags) +
   re-read of the SUPERB plan (full task tables, milestones M1–M5,
   guardrails G1–G7) and the 14-14 report's §f/§g. Todos seeded with the
   corrected true statuses (T1–T11 DONE).
2. **T24.1 — frozen-tree matrix run #1** (tree clean, daemon swept):
   full log `/tmp/verify-ci-2026-09-14-full.log`. Exactly **three
   failures, all in this arc's new surface**; every other module green:
   - `queue/sqlite` `TestConformance/Lifecycle/heartbeat_extends_the_lease`
     — "lease not extended: 14:28:18.087 <= 14:28:18.087"
   - `cmd/api-stability` `TestMultiPackageModulesHaveArchLintConfig` —
     "module queue has 4 production packages but no .go-arch-lint.yml"
   - `cmd/cqrs-lint` `TestCatalogEveryGoWorkModuleCovered` — queue,
     queue/sqlite, queue/postgres neither catalogued nor excluded.
3. **Fix 1 — heartbeat pin ms-race** (`queue/conformance/lifecycle.go`
   `pinHeartbeat`): claim and heartbeat both used 1-minute leases;
   executed within the SAME millisecond, the ms-truncated stamps are
   identical and the strict `After` fails. Now sleeps 2ms past the
   boundary (comment explains why). Root cause verified against
   `harness.go`'s `claim()` (lease = time.Minute).
4. **Fix 2 — `queue/.go-arch-lint.yml` created**: components
   core/task/facts/conformance with the real intra-module graph
   (core → task+facts; conformance → core+task+facts; task/facts
   stdlib-only), nested engine modules excluded (metaengine pattern).
   `scripts/check-arch.sh` green for queue (and repo-wide unchanged).
5. **Fix 3 — cqrs-lint catalog**: `queue` added to
   `buildDefaultCatalog()` (CategoryWorkflow, beside scheduling);
   test count pins updated 32→33 scored / 38→39 total;
   `queue/postgres`+`queue/sqlite` added to the drift test's exclusion
   map ("sub-engine (covered by queue)").
6. **Per-module gates re-run green** after the fixes: full cqrs-lint
   module, api-stability (targeted), queue/sqlite full suite.

## b) PARTIALLY DONE

1. **T24 — evidence re-run (matrix run #2) launched at ~14:44**
   (`/tmp/verify-ci-2026-09-14-rerun.log`, background job) and **still
   in flight at report time** — with a **NEW failure already captured**:
   `queue/sqlite` `TestConformance/Reads/status_counts` —
   `reads.go:145: queue: invalid status transition: running -> cancelled`.
   NOT reproduced by the local full-suite run minutes earlier (ok,
   3.5s). Suspects: (i) another latent timing/ordering nondeterminism
   in a Reads pin (same class as the heartbeat ms-race — a
   pending-only Cancel hitting a task that a claim already took), or
   (ii) concurrent-arc edits to shared files between runs (`git log --
   queue/` not yet checked). T24 stays RED until run #2 completes and
   this flake is root-caused.
2. **T12/T13 — prepared, not executed**: tag-release.sh semantics
   researched from the plan/§f; dry-runs were sequenced strictly after
   T24 evidence and were not reached.

## c) NOT STARTED (unchanged from 14-14 report)

T12 dry-run; T13 (queue trio tags); T14 deps validation + cycle
rejection; T15 ADR-0134 tokens; T16 FactSink-in-tx; T17 MySQL engine;
T18 metaengine read-adapter; T19 taskmanager example; T20 PapDashboard
evaluation; T21 tq re-open ADR; T22 doc tail; T23 rulings batch.

## d) TOTALLY FUCKED UP

1. **The prior session's "G3 full ceremony complete" was overstated**:
   the matrix proves two ceremony gaps (arch-lint config missing,
   cqrs-lint catalog missing). Ceremony was claimed from a checklist,
   not from the gates that enforce it — exactly the class of
   uncited-claim the repo's own rules ban.
2. **The conformance suite shipped with ≥2 latent nondeterministic
   pins**: the heartbeat ms-race (now fixed) and the Reads/status_counts
   ordering race (new, undiagnosed). Earlier "green plain + -race
   -count=2" evidence passed by timing luck, not determinism.
3. **This session repeated the claim-pattern it was fixing**: I declared
   all three fixes "verified" on per-module gates; the very next matrix
   run produced a fourth failure. Matrix-level green ≠ sum of
   per-module greens (test ordering/flags differ).
4. Small but real: my first `.go-arch-lint.yml` had parse errors
   (false/false flags where the validator demands a `mayDependOn` ref or
   a true flag) and excluded `conformance` instead of declaring it a
   component — two avoidable fix cycles on a 25-line file because I
   wrote the config before reading the validator's rules or
   metaengine's multi-package example.

## e) WHAT WE SHOULD IMPROVE

1. **Never claim a gate/ceremony complete without citing the enforcing
   gate's green output** — "checklist done" is a hypothesis, the gate
   run is the evidence.
2. **Audit ALL conformance pins for ms-boundary/ordering
   nondeterminism** — grep every time comparison in `queue/conformance`;
   each strict `After` needs a forced clock gap (or `>=` where equality
   is legal). The suite is the parity bar for three engines; it must be
   deterministic first.
3. **The matrix is the only "green" that counts.** Per-module gates are
   progress signals; evidence claims must reference `#verify-ci` runs.
4. **Read the validator schema + one passing complex example before
   writing any config file** (arch-lint cost 3 rounds this session).
5. **Root-cause flakes before re-running**: each matrix run costs ~7
   minutes; the flake fix loop is read-pin → fix → local `-count=5` →
   matrix, never "re-run and hope".

## f) NEXT (up to 50, ordered)

1. Poll matrix run #2 to completion; read its full log tail.
2. Triage `Reads/status_counts`: read `queue/conformance/reads.go`
   :100–160; reproduce locally with `-count=5 -shuffle=on`; check
   `git log --oneline -- queue/` for concurrent-arc edits.
3. Fix the flake (forced clock gap or corrected claim/cancel
   sequencing); local `-count=5` green.
4. Matrix run #3 = T24 evidence GREEN; record log path as tag-wave
   insurance (T24.3).
5. `git log -- queue/ cmd/cqrs-lint/` sanity: confirm no foreign edits
   rode into the daemon commits carrying this session's fixes.
6. CHANGELOG `[Unreleased]`: cqrs-lint catalog queue entry (user-visible
   suggestion surface) — check changelog-symbols gate.
7. T23 rulings batch doc (see g3): recommendations + decision table,
   ready for owner ruling without further research.
8. T12.1: CHANGELOG/release-note check for claiming entry.
9. T12.2-prep: `tag-release.sh claiming v4.0.0 "<desc>" --dry-run` on
   clean tree; capture output.
10. T12.3 (OWNER-GATED): push tag; bump sqlstore/example pins.
11. T12.4 (after push): strip sibling replaces; standalone GOWORK=off
    builds green.
12. T12.5 (after push): proxy probe (`go get` claiming@v4.0.0 scratch
    module).
13. T13.1–T13.4: golden+TestEvery, doc rows, module-map/features check
    for the queue trio (mostly done 13:xx — verify, don't redo).
14. T13.5 (OWNER-GATED): batch-release wave `queue/v4/v4.0.0`,
    `queue/sqlite/v4/v4.0.0`, `queue/postgres/v4/v4.0.0`.
15. Post-wave: `nix flake check` (vendorHash drift after go.mod edits).
16. Post-wave: `nix run .#check-duplication` — expect art-dupl accepts
    for the engine mirrors.
17. Post-wave: `nix run .#verify` (race/coverage/doc gates).
18. Post-wave: `nix run .#check-coverage` — queue modules coverage floor.
19. Post-wave: `nix run .#vulncheck`.
20. Advisory lint count check for queue/* vs baseline policy.
21. T14.1: deps enqueue-time validation design note (see g2).
22. T14.6: cycle-rejection policy + test (new surface; design note
    first).
23. T14: unblock-bump (ADR-0015 donor concept) evaluation for queue/.
24. T14.5: conformance pin blocked-until-parent-done (verify existing
    deps pins cover it; extend if not).
25. T15.1: ADR-0134 adoption note (tokens day one; Claim struct is the
    seam; unreleased module ⇒ non-additive signature change is free).
26. T15.2: lease_token column + crypto/rand mint in both engines.
27. T15.3: token predicates in Heartbeat/Complete/Fail/FailPermanent/
    Requeue/CancelOwned.
28. T15.4: theft-detection error semantics (ErrLeaseNotHeld path).
29. T15.5: conformance pins (holder-only ops).
30. T15.6: docs + api golden regen (signature changes!).
31. T16.1: FactSink-in-tx capability interface design.
32. T16.2/T16.3: sqlite + PG in-tx external fact append.
33. T16.4: watermark operator surfaces (List/Set) evaluation.
34. T16.5: conformance pin "no state change without fact" via sink.
35. T16.6: ADR-0001-lineage docs.
36. T17.1–T17.3: MySQL engine (DATETIME(3), claiming.MySQLClaimSelect
    two-statement claims).
37. T17.4: mysql testcontainer integration harness (docker available).
38. T17.5: MySQL conformance green + ceremony (go.work/flake/golden/
    module-map/layers budgets/CHANGELOG).
39. T19.1–T19.4: taskmanager example on queue/sqlite; README; tests.
40. T19.5: cqrs-lint V006 golden refresh after example dep changes.
41. T18: metaengine read-adapter design note (read side ONLY).
42. T18.2–T18.5: projection folds, FilterSpec demo, bench, docs.
43. T20: PapDashboard usage read + mapping doc + gap list + verdict.
44. T21.1–T21.5: tq parity checklist, facade re-point spike, tq
    conformance against upstream, journal migration sketch, ADR draft.
45. T22.1: proposal doc P0 phrasing fix (claiming DONE + trim note).
46. T22.2: cqrs AGENTS go.work use-block drift (go-idempotency).
47. T22.3: rejection-propagation rule encoded in tq AGENTS.
48. T22.4: SKILL.md queue section + references/modules.md row checks.
49. T22.5 + plan annotation: SUPERB plan milestone banners (ANNOTATE,
    never rewrite); AGENTS module count 88→91.
50. T22.5: final arc status report (+ archive-counter updates).

## g) Questions for the owner (cannot figure out myself)

1. **Tag + push authority for the v4.0.0 wave** (T12/T13; same as 14-14
   Q1, still the only blocker for M3 completion): dry-runs only until
   you rule — cut+push `claiming`, then `queue`/`queue/sqlite`/
   `queue/postgres` autonomously, or you run the wave yourself after
   reviewing the dry-run output? (Tags are immutable once pushed; G7
   keeps the owner in the loop; proxy resolution order matters:
   claiming must land before the queue trio.)
2. **T14 dep-validation scope** (14-14 Q3, still open): standardize
   "all deps must exist at enqueue time, on every engine" (+ cycle
   rejection) — a small deliberate semantic change vs the donor (which
   inserts blindly; FKs reject dangling deps on SQLite only) — or stay
   donor-faithful and add ONLY cycle rejection? This is a contract
   semantics choice, not a detail I can infer.
3. **gci-vs-treefmt repo-wide lint conflict** (pre-existing,
   owner-escalated, queued as T23): do you want to rule NOW (which tool
   wins) so the T23 batch ships with decisions instead of
   recommendations, or should T23 present a recommendation table and
   wait?

---

_Evidence anchors: matrix run #1 full log
`/tmp/verify-ci-2026-09-14-full.log` (3 failures, rest green);
post-fix per-module gates green (cqrs-lint full module, api-stability
targeted, queue/sqlite suite ok 3.5s); matrix run #2
`/tmp/verify-ci-2026-09-14-rerun.log` IN FLIGHT at 14:46 with the new
`Reads/status_counts` failure captured above; check-arch.sh green incl.
queue; fixes uncommitted at report time (daemon will sweep — expected)._
