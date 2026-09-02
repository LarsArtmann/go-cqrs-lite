# Status — Plan V3 Execution: Trust Floor (T02–T07), P03 Race (T24), Docs Debt (T29–T35, T40, T49, T50.2)

> **Date:** 2026-08-29 14:50 CEST · **Session:** "Get shit done — the whole
> TODO list" · **Plan:** [`2026-08-29_12-14_SUPERB-ALL-TODOS-PARETO-PLAN-V3.md`](../planning/2026-08-29_12-14_SUPERB-ALL-TODOS-PARETO-PLAN-V3.md)
> **Boundary honored:** T08–T19 (tag waves, v5 spine) and T42 stayed
> `[user]`-gated; the parallel tag-wave session executed B1–B3 and completed
> the wave during this session.

## a) DONE (every item re-verified)

1. **T04 archive spot-verify** — `shuf -n 20` over the 1,041 archived docs;
   20/20 verdicts upheld, 0 files moved back. Two load-bearing open items
   confirmed live-tracked (TODO_LIST:240, :29); CatalogMeta removal and
   benchkit `contextcheck` fixes verified against current code. Addendum in
   the audit report.
2. **T07 backuptest standalone** — the wave's B3 cut `backuptest/v4.1.0`
   (first fetchable tag); I bumped the bbolt/pebble pins and dropped both
   `=> ../backuptest` replaces. GOWORK=off build+test green.
3. **T05 metaJSON discards** — honest re-scoping: `Metadata[K].Custom` is
   `map[K]string`, so the marshal error path is UNREACHABLE today — the
   harvest's "silent corruption" premise was wrong, and the fix is hardening
   (nil-on-failure + deterministic marshal, mirroring the event adapter)
   plus the FIRST metadata roundtrip tests through the envelope path.
4. **T06 ScanSlice pre-size** — `RowCount()` (a SELECT COUNT) would double
   remote RTT and `RowsAffected` is unusable pre-iteration; instead
   `ScanSlice` takes an optional capacity hint and `JournalReader` threads
   its bounded limit (capped 4096) into the drain-path scans. Benchmark
   added; 10K-row run shows the hinted path at ~264KB/12K allocs vs default
   growth.
5. **T02 full verify floor** — run 1: RED (10 lint findings + 1 load-flaky
   probe test). Fixes: my storage test godot/gofumpt; the wave session's
   watermill/listing/pgtestcontainer findings (prealloc, unconvert,
   exhaustive→documented nolint, golines, wsl); the probe test now polls
   until BOTH live-RTT and read-tracker signals are visible (timing-proof by
   construction). **Run 3: full `#verify` GREEN** (build, vet, test, race,
   lint 76/76, check-arch, depguard, doc-check 1154 refs) at `50a9a212d`;
   recorded in TODO_LIST in place of the 2026-08-16 claim.
6. **T03 duplication + arch + link checker** — both gates green; new
   `scripts/check-doc-links.sh` (code-fence aware, symlink-resolving,
   archived-history-exempt). First pass found **33 real broken links** —
   mostly audit-reorg casualties; all repointed. ~280 initial hits were
   parser noise (generics in code fences) or frozen history.
7. **T24 / deep-review P03 (🔥)** — confirmed real: `Plan` shares fold
   instances across Stores, so live `Apply` + `Verify` replay raced the
   `recHolder` cell (per-Store fold locks don't serialize each other).
   Regression race test written FIRST — DATA RACE reproduced pre-fix, green
   ×3 post-fix. Fix: Record flows through the invoke closure as a value for
   all 11 fold kinds; recHolder/recordSetter/internal SetCurrentRecord
   deleted; `RecordAwareFold` kept as a Deprecated compat interface.
   **Bonus:** OnRecord Append/MultiEntry/Embedding/IndexedText/Point handlers
   had always received an EMPTY record (setter never wired) — fixed by the
   same threading. Plus T24.3/T24.4: `EventInput.Record` (additive) is stored
   by `EventLog.RecordEvent`, and Backfill / Demote catch-up / Verify replay
   now carry the original record instead of a synthesized one.
8. **T29 CHANGELOG fold** — the 2026-08-16-era unreleased block (2,005
   lines) folded under the single `[Unreleased]` window. The symbol gate
   then scanned the whole window and flagged 18 citations: external-package
   false positives (gate SKIP_ALIASES extended per its own policy), wildcard
   sentinel mentions, honest negations, and four APIs renamed/deleted after
   the fact — all reworded to current truth with supersession notes. Gate
   green: 112 citations verified.
9. **T30/T31/T32 docs truth-pass** — faq.md gained a v5-deletion overview
   (it had zero); AGENTS codec-defaults carries the v5 note (372/377 lines);
   method-level `Deprecated:` convention recorded as an ADR-0123 addendum;
   **storage/pebble verified to have NO stack import** (the "durability
   re-home BLOCKS T15" claim was already stale — only a doc comment, fixed);
   stack/bench decision: delete with stack at v5. SEVEN-TIER-MODEL now says
   Tier-3-with-Tier-0-core (deps verified from go.mod: dedup + record + id)
   and dropped codec/flightrecorder/retry rows. The ~41-byte figure →
   43–46 bytes in all 3 active docs (+1 straggler in the bench code comment).
   AGENTS gotcha added (pebble close-without-flush, bbolt mmap quantization,
   MySQL-VM trio) within budget.
10. **T33/T34.2 engine docs** — READMEs written for the four undocumented
    engines (mysql/sqlite/turso/badger), each capability list verified
    against the engine's compile-time interface assertions; metaengine
    capability table gained MySQL/Turso/Badger health rows and the engine
    index lists all nine; modules.md gained the missing bboltengine row;
    pebble's package doc no longer cites the superseded zero-dep-core
    framing.
11. **T34.1/T35 benchmarks + release docs** — pebble durability benches run
    in a genuine calm window (load ≈1.5): sync ≈2.45ms/op, async ≈2.18ms/op
    median of 5×200 — async saves only ~11% on this device; cell filled.
    CONTRIBUTING gained the pin-bump-before-tag recipe + GOPRIVATE
    clean-room verification commands. ADR-0130 documents the full
    engine×tier durability mapping (from each engine's code, including the
    rejectors and the two-sources-of-truth rule). Doctor gained a
    `--- Durability ---` section via the new optional `DurabilityReporter`
    capability (per-engine adoption lands with engine tags).
12. **T40 design decisions** — ULID entropy: question already answered IN
    CODE (`id/entropy.go` lock-free epochs); documented as accepted in
    ADR-0131. Pebble calibration basis: post-Flush/pre-Compact fixed as the
    basis in ADR-0132. command.Bus/MemoryBus removal: DECLINED with
    rationale (47 lines, saga-example load-bearing, complements watermill).
13. **T49 skill-refs freshness** — the five references swept for deleted/
    renamed/deprecated tokens: external-module paths correct post-ADR-0128,
    deprecations marked; one residual fixed (metaengine row's un-caveated
    `stack.Bundle` integration). doc-check 931 refs green.
14. **T50.2 feedback lane** — `docs/feedback/archive` → `archived`; lane
    naming now uniform across status/planning/feedback; no links referenced
    the old name.
15. **Concurrent-session collision repairs** — go.work aligned to the root
    module's go 1.26.7 (their bump broke every workspace build); the
    config reformat's re-enablement of `gci` reverted (it deleted the
    config's own rationale; AGENTS rule 18 says treefmt owns grouping).
16. **Pushes** — master pushed to origin (no tags); synced with the wave
    session's state.

## b) NOT DONE (honest)

1. **T36 docserver set** (css GET test, cId note, deps table, drift gate,
   CSP nonce, EventCatalog CLI validation) — untouched.
2. **T37 benchmark-regression hardening** (fixture test, threshold re-tune,
   runbook, actionlint, `verify --module`) — untouched.
3. **T38 cqrs-lint C040 + doctor/audit polish** — untouched.
4. **T39 consumer asks** — snapshot encryption (design+impl), go-retry
   `DoWithValue[T]` (external repo), OTel exporter-lifecycle docs.
5. **T41 July archive pass** (174 status + ~40 planning files) — untouched;
   needs its own session.
6. **T42** — needs the user's annotation-depth policy answer (standing
   question from the audit).
7. **T08–T19/T43–T48** — `[user]`-gated / multi-day; the parallel session
   completed the tag waves (B1–B7) per its final planning doc.

## c) Gates / verification

- Full `#verify` GREEN (run 3) at `50a9a212d`; later runs caught the golden
  slip (DurabilityReporter) and the gci regression — both fixed.
- Final state: build/vet/test green through run 7 except ONE ambient-load
  flake (`metaengine/bench` cutover test, "context canceled" at box load 10+
  from the parallel session's own gate); passes ×2 isolated. Lint/doc
  phases were then re-run and are green EXCEPT findings introduced by the
  parallel session's in-flight tombstone-deprecation sweep (~20 modules,
  their un-scoped SA1019 uses and pending nolint scoping) — left for them
  to finish; touching it mid-flight would collide.
- check-duplication: 0 new clones (baseline 111). check-arch green.
  changelog-symbol gate green (112 citations). doc-check green (931 refs).
  link checker green (742 targets). api-stability golden current (4,289).

## d) Failures / self-review

1. **`git add -A` swept the parallel session's unstaged work** into the T40
   commit (62 files). Not reversible without forbidden history surgery; the
   commit message now documents exactly what was swept. All later commits
   used explicit paths.
2. **Golden regen slipped the same-edit rule** once (DurabilityReporter) —
   caught by the full gate, not by my discipline.
3. **Two verify runs spent on avoidable REDs** (lint findings from the wave
   session's committed code; the probe-test race with the read tracker) —
   the probe fix is structural; the lint findings were committed by the
   wave session without a lint pass.
4. **T05/T06 premises partially wrong** — the harvest clusters survived
   re-verification only in weakened form (unreachable error path; COUNT
   would have been a pessimization). Both were re-scoped honestly rather
   than implemented as specified.
5. Wave-session collisions (3): backuptest tag raced (resolved — they cut
   it, I dropped the replaces), go.work breakage (fixed), gci reintroduction
   (fixed, with the config's own note restored).

## e) Commits (all on master, pushed)

`bea0c21cb` T04 addendum · `5fb01f764` T07 · `0598219e2` T05 · `c2c1962c7`
T06 · `bc1423761` lint fixes (wave session's findings) · `50a9a212d` probe
flake fix · `574f68778` T02+T03 · `0598219e2`→`c2c1962c7` scoped tests ·
`ab1795544` T29 · `ddc69624e` T30–T32 · `3051a7d7a` T33/T34.2 ·
`ffb1ae35f` T34.1+T35 · `38afb6d2e` T40 (+documented sweep) · `8010cd267`
go.work 1.26.7 · `54e697c3a` golden regen · `4ec1fd594` gci restore ·
`e13e9be43` T50.2.

## f) Next steps (ordered)

1. Parallel session finishes the tombstone-deprecation lint scoping (their
   in-flight changes are the only current RED).
2. T36–T39 code tasks (each 1–3h, precise starting points in plan V3 §3).
3. T41 July archive pass (same protocol as the 08-29 audit).
4. T50.1 GitHub Releases — still blocked on the billing fix.
5. T42 annotation-depth — still needs the user's policy answer.
6. T43–T48 feature waves — `[user]` sequencing per plan.

## g) Questions

1. The wave session's `.golangci.yml` reformat also re-introduced `gci`;
   I reverted that per AGENTS rule 18 — confirm that config change was
   unintended.
2. T39.1 snapshot encryption is a feature-sized ask (design + implementation
   - rotation test). Want it next session, or design-note first?
3. T41 July pass: same archive rule as August, or leave July frozen?

---

_Point-in-time at `4ec1fd594` (my last commit) / `5e6dfffb2` (wave session's
latest). 2026-08-29 14:42 CEST._
