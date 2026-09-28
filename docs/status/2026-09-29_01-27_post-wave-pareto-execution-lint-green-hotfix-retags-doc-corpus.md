# Post-Wave Pareto Execution: Lint Green, watermill/tursoengine Hotfix Re-tags, README Doc-Corpus

**Written:** 2026-09-29 01:27 CEST
**Session start:** ~00:30 CEST (continuation of the 2026-09-28 TODO-execution day)
**Executed plan:** [`docs/planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md`](../planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md) (ready-queue order: M01 → M02 ∥ M06 ∥ M08 ∥ M03 → M07 → M10 ∥ M12 ∥ M13 → receipts)
**Machine:** load1 10–12.3 throughout (never <5 — all quiet-window gates stayed closed, by rule).

---

## a) FULLY DONE (verified green unless noted)

### M01 — Lint → green (the gate that guards everything)
- G703 (gosec path-traversal taint) nolints moved to the SINK lines in
  `catalog/cmd/ec-fixture/main.go` — with a fresh trap lesson: statement +
  trailing nolint >120 chars makes golines WRAP the call and relocate the
  comment to the last line (silently un-suppressing). Fix: pre-wrap the call
  so its opening line has room for the directive.
- dupl accept on the declarative rule tables: the finding FLIPS sides
  (adoption → boilerplate) after fixing one file — both `catalog_adoption.go`
  and `catalog_boilerplate.go` line-1 `//nolint:dupl` needed.
- `nix fmt` + module lints green + **full `nix run .#lint`: 88/88 modules clean**.
- Quick tests on the 9 lint-touched modules green (scheduling/sqlstore first
  run rc=1 was my /tmp-path bug in the loop; clean on rerun, 7.3s).

### M06 — config-war tripwire trio: ALREADY STRUCTURAL, now PROVEN
- Discovery: `#check-lint-config` (config-verify + hash-golden + depguard +
  formatters pin) has run inside BOTH `#verify` and `#verify-fast` since
  2026-09-06 — the plan's "wire it" step was already done. What was missing
  was proof.
- **Mutation-verified 2026-09-29:** planted `gci` back into
  `formatters.enable` → gate fails rc=1 with "content hash MISMATCH
  (corruption tripwire)" → restored → rc=0. The 2-day-undetected
  incident-12 class (09-26 daemon rewrite) is now mechanically caught by a
  ~2-minute `#verify-fast` probe.

### M08 — named-byte payload hazard: swept, documented, rule deferred
- Repo-wide sweep clean: gRPC getters return plain `[]byte`; the watermill
  command bridge carries no payload through constructors; no
  `json.RawMessage` at any constructor; the three in-repo named `[]byte`
  types (`Signature`, `Ciphertext`, `JSONValue`) never reach `event.New`.
- FAQ entry added ("never pass a named `[]byte` type into `event.New`") —
  doc-check green (1246 refs at that point).
- F032 lint-rule candidate: DEFERRED until the FP-sweep harness is
  trustworthy (M10, now done — rule build is unblocked and specced in
  TODO).

### M03 — watermill/v4.6.2 hotfix published (ACTIVE CONSUMER HARM STOPPED)
- Pre-cut: tag-content audit (`git log` since v4.6.1 = the fix + pin bumps),
  module tests green.
- First `batch-release.sh` attempt FAILED on the ambient-env trap
  (`GOTOOLCHAIN=local`, go 1.26.7 < 1.27.1) — the new run log (M15/e1)
  diagnosed it in one read, exactly as designed.
- Second attempt with `scripts/go-env.sh` sourced: **tagged, pushed,
  proxy.golang.org serves `watermill/v4@v4.6.2`, smoke green** (attempt 1).
- Post-tag pin-sweep bumped stack presets + integration to v4.6.2
  (18 files, committed).

### M07 — tag-wave mechanics
- **CHANGELOG sections cut** (committed authored `1962b83fd`):
  `## [watermill/v4.6.2] — 2026-09-29` and the missing
  `## [7-tag wave] — 2026-09-28` train section (8 bullets moved; dedup
  bullet annotated that benchkit's export rides its next tag).
  `check-changelog-symbols.sh` 40/40 honest; `TestTagContentMatchesChangelog`
  green.
- **tursoengine/v4.2.1 re-cut** (the sanctioned poisoned-tag surgery):
  - Cut 1 REFUSED by the zip-content guard: two daemon-committed ~8MiB
    build binaries (`catalog/cmd/{catalog-export,ec-fixture}/…`) sat in the
    tree. Removed via `git rm`, both paths gitignored (the old ignore entry
    covered a pre-move path only).
  - Cut 2 lost a HEAD race against the auto-commit daemon (my commit's
    lock check failed; the changes rode the daemon commit instead).
  - Cut 3 on the clean tree: **tagged, pushed, proxy serves v4.2.1, and
    `@latest` resolves v4.2.1** — the poisoned-@latest residual defect from
    the data-mesh arc is CLOSED.

### M12 — benchkit polish-tail: ALL FIVE debts closed
- (b) `<metric>_cov%` verified through REAL benchstat-format output:
  new subprocess test runs `go test -bench` and parses the actual
  tab-separated `<name> <iters> <value> <unit>` columns (green, 7.3s).
- (c) `startProfiling` teardown-order test — after finding the function
  (it lives in `cmd/cqrs-bench`, NOT benchkit; see §d-5): pins
  flush-before-close via "valid gzip pprof stream" + empty-flags no-op.
- (g) benchkit README + doc.go API tours synced (RunSuiteRepeated,
  HeadlineMetricNames, Config.ReservoirSize).
- (h) `noise_target_guard` identifier-grade: `--exclude='*_test.go'` on all
  three greps (a rename surviving only in tests now fails the guard);
  bash -n + live-target verification green.
- (i) testcontainers teardown noise: EXPECTED cleanup — orderly
  Stop→✅stopped→Terminating→🚫terminated sequence; `docker ps` shows ZERO
  residual testcontainers after the run. Not a leak.

### M13 — README/doc gates: module READMEs are now gated surface
- doc-check corpus extended: auto-discovery globs depth 1–3 (97 module
  READMEs), flake `#doc-check` + `#verify` inline corpus updated; the CI
  doc-check leg (zero-arg auto-discovery) inherits everything.
- Fixed everything the corpus surfaced:
  - 1 broken ref (irohengine `engine.MapSet` — false positive on a
    fence-local variable → doc-check now skips block-local declared idents);
  - 6 ARITY LIES: eventtest conformance calls ×3 (dropped removed aggID
    arg), `listing.NewInMemoryAggregateReader` → real constructor is
    `NewInMemoryStreamReader(store, opts...)` ×2, `metaengine.ServeSSE`
    missing (w, r) ×1;
  - 4 ambiguity warnings: doc-check gained file-level import carryover
    (continuation fences inherit prior blocks' imports); readme-quickstart
    + queue README quick-starts got scoping imports.
  - nav-checker `loadDocName` gained the same-skill `../SKILL.md`
    candidate (watermill advanced.md's "(SKILL.md §1)" refs were resolving
    to the WRONG skill's SKILL.md); core.md §ref legend now names docs.
- **Full auto-discovery run: 2420 references valid across 86 packages,
  zero warnings, zero broken nav.** (Was 1246/54 before READMEs.)

### M10 — cqrs-lint FP-sweep harness refresh
- Harness: stderr captured separately; zero-findings + stderr → `STDERR:`
  tail; EMPTY stdout → `NO JSON OUTPUT (linter skipped repo)` (a skip is
  never read as clean); arithmetic hardened (`${n:-0}`).
- Corrected 12-repo baseline re-run: **426 findings, 41 low-confidence**
  (old: 357/36 with five SILENT rows). The five previously-silent repos
  now report (bank-sync 6, browser-history 4, github-local-sync 14,
  go-localsync 15, dnsblockd 17). `overview` = transitive-only consumer
  (7 indirect pins, 0 direct) → labeled skip.
- Outlier verdict (M10.4): crush-daily 38/14 is STABLE vs 39/15 —
  consistent adoption-debt signal, not a flake;
  standard-bug-tracking-schema 194/9 also stable.

### M02 — receipts sync
- Dedup-campaign row rewritten: (b)(c)(d-pg/dgraph/redis)(e)(f) struck
  with dated receipts; (a) composed-verify + (d)-mysql remain, gates named.
- CI "Composed #verify re-record" row: STATE 2026-09-29 (pre-run
  obligations green; only the quiet window missing).
- Status-index integrity checked (live rows vs disk, no drift).
- Three AGENTS lessons written into the gotcha files: G703-sink +
  golines-relocation nolint refinements; incident-12 + "trust the
  tripwire, #verify-fast is the cheap post-daemon-wave probe"; named-byte
  constructor hazard (full lesson + watermill/v4.6.1 case).

---

## b) PARTIALLY DONE

1. **M10.5 baseline doc rewrite** — content drafted; the write hit doc
   tooling's stale-read guard (script-vs-View mtime). File re-Viewed;
   ready to land in one step.
2. **M07.5 receipts** — tursoengine/v4.2.1 CHANGELOG section entry +
   TODO poisoned-tag row strike not yet written (tag + proxy + @latest
   receipts are in §a; only the paperwork remains).
3. **TODO_LIST/CHANGELOG receipts for today's batch** — M03/M07/M10/M12/M13
   rows + [Unreleased] entries for the doc-check corpus, FP-sweep harness,
   benchkit tests, binary-drop fix, M06 mutation proof: not yet struck.
4. **Post-change gate sweep for this session's code edits** —
   cmd/doc-check (scan.go, navrefs.go, main.go), cmd/cqrs-bench
   (profiling_test.go), benchkit (polish_tail_test.go, doc.go, README),
   flake.nix, scripts/{fp-sweep,benchmark-regression}.sh: module lints +
   api-stability golden regen + `nix flake check` not yet run as a batch
   (doc-check's own tests ARE green; benchkit targeted tests green;
   cqrs-bench targeted tests green).

## c) NOT STARTED (plan order)

- **M04 composed `#verify`** — quiet-gated; load never dropped below ~10.
- **M05 MySQL quiet legs** (vm + shuffled seeds) — quiet-gated.
- **M11 calibration campaign** (SearchQuery + dgraph constants) —
  quiet-gated.
- **M09 catalog/v4.6+ wave + mesh-demo strip + pin sweep** — owner-gated
  (BLOCKED row).
- **M14** canonical-facts derivation · **M15** owner decision bundles ·
  **M16** queue M4 tail · **M17** system test-mass · **M18** temporal
  property tests · **M19** NATS JetStream leg · **M21** quality long
  tail · **M20/M22/M23/M24/M25/M26/M27** (gated/watch tier).

## d) TOTALLY FUCKED UP (all caught + recovered; lessons recorded)

1. **rc-capture bug, AGAIN** (`cmd | tail; echo $?` prints tail's rc) in
   the M06.2 mutation leg — the exact repo-documented trap. Redone with
   `$(...)` capture; real MUTATED_RC=1 verified.
2. **First batch-release ran without `go-env.sh`** → verify leg died on
   GOTOOLCHAIN=local/go 1.26.7. The documented one-line rule; the new run
   log turned it into a 5-minute diagnosis (silver lining: M15/e1 proving
   itself in production on day one).
3. **CHANGELOG Python surgery left the watermill bullet DUPLICATED**
   (extracted for the new section, forgot to delete the [Unreleased]
   copy) — caught in diff review; also tripped the mtime guard once
   (script-edited file needs re-View).
4. **flake.nix glob-base error**: wrote `../*/README.md` (cwd =
   cmd/doc-check → cmd/*/README.md) instead of `../../*/README.md`; my
   first "fix" ADDED a nonsense anchor line before correcting. Final form
   verified by running the actual corpus green (2420 refs).
5. **Declared M12(c) "reference rot — startProfiling never existed"** from
   a benchkit-scoped search. Wrong: it lives in `cmd/cqrs-bench`. A wider
   workspace grep corrected it before the receipt was written. (Root
   cause: trusting the TODO's module attribution over searching the
   workspace.)
6. **Daemon commit race**: my binary-drop commit failed on HEAD lock
   (`cannot lock ref`); the staged changes rode the daemon's commit
   instead. Outcome fine (changes landed), authored history lost.
7. **fp-sweep arithmetic hole** (empty-string findings cell for overview —
   jq on empty input yields "" not 0) — hardened in the same session.

## e) WHAT WE SHOULD IMPROVE

1. **`tag-release.sh`/`batch-release.sh` should source `go-env.sh`
   themselves** (or re-exec under it) — my failed cut is the Nth ambient-env
   victim; scripts that run `go build` in their verify legs should own
   their toolchain env.
2. **Clean-tree gate vs daemon**: release scripts could wait-for-clean
   (bounded) or auto-stash instead of aborting; tonight's cut needed three
   attempts across two daemon races.
3. **Gitignore hygiene is reactive**: two 8MiB binaries rode the tree for
   hours because the ignore entry predated a directory move. A tiny
   `check-no-tracked-binaries` gate (or extending the zip-content guard to
   a daily check over HEAD, not just at cut time) would catch this class
   pre-tag.
4. **My search scope discipline**: one workspace-wide grep beats three
   module-scoped greps + a wrong conclusion (§d-5).
5. **doc-check corpus grew 1246→2420 refs**: watch the CI leg's runtime;
   if it bothers CI, split READMEs into their own leg.
6. **rc hygiene in verification one-liners**: use `out=$(cmd); rc=$?` form
   everywhere; the pipe-tail form has now bitten this repo twice in a week.

## f) NEXT (ready queue; receipts first, then the plan's gates)

1. Land the M10.5 baseline doc (unblocked, content drafted).
2. CHANGELOG [Unreleased] entries for today's batch (doc-check corpus +
   fixes, FP-sweep harness, benchkit (b)(c)(g)(h)(i), binary-drop +
   gitignore, M06 mutation receipt, tursoengine section entry).
3. TODO_LIST strikes: watermill re-tag row, poisoned-tag row (v4.2.1
   published, @latest fixed), benchkit polish-tail row, README-tail rows,
   fp-sweep rows, lint row.
4. AGENTS gotcha: batch-release needs go-env.sh (or the script self-source).
5. Post-change gate batch: lint cmd/doc-check + cmd/cqrs-bench + benchkit;
   `cd cmd/api-stability && GOWORK=off go run . --update` (code changed);
   `nix flake check` (flake.nix touched); `nix run .#check-md-go` (docs
   touched); `nix run .#check-duplication` (doc-check/cqrs-bench code).
6. M04 composed `#verify` (quiet window; preflight-composed first).
7. M05 MySQL legs (same window if it holds) + F52 receipts.
8. M11 calibration campaign (same window).
9. M09 catalog wave (owner go-ahead) → mesh-demo strip → pin sweep → M27.
10. M14 canonical-facts derivation into FEATURES.
11. M15 owner bundles (quick rulings one-pager + ADR pack).
12. M16 queue M4 tail (clock-seam design + parallel-migrate sweep).
13. M17 system test-mass (config-loader fuzz, shutdown stress,
    determinism).
14. M18 temporal rapid properties (out-of-order stamps, retention,
    tombstone-as-of).
15. M19 NATS JetStream roundtrip leg (`ephemeral-nats.sh` + flake app).
16. M21 quality long tail (GraphBFS unify, passthrough conventions,
    contention-retry backport).
17. F032 named-byte lint rule build (now unblocked post-M10).
18. FP-sweep: rule-tuning pass targeting crush-daily's 14 + cqrs-htmx's 4
    low-confidence findings.
19. M20 matview triage / M22 CI watch / M23 upstream filings / M24
    enricher / M25 mesh-demo variant / M26 v5 inventory (gated tier).
20. Push master branch to origin (tags pushed; branch is ahead).
21. Verify the gitignore entries hold after the next catalog build (daemon
    re-absorb watch).

## g) QUESTIONS (cannot resolve myself)

1. **Branch push:** both re-tags are on origin, but `master` itself is
   several commits ahead (my authored commits + daemon commits). Push
   master now, or do you want to review first?
2. **M09 catalog/v4.6+ wave:** your directive said "the WHOLE TODO LIST",
   but the row is marked owner-gated (13–32 modules, big blast radius,
   unblocks mesh-demo strip + consumer adoption). Cut it this session or
   wait for a separate explicit go-ahead?
3. **Quiet-window strategy for M04/M05/M11:** load never dropped below ~10
   all session. Keep a background poller waiting for load1<5 (could be
   hours), or schedule a dedicated quiet-window session and close this one
   with the receipts batch?

---

**Session verification summary:** `#lint` 88/88 · check-lint-config
mutation-proven · doc-check 2420 refs / 86 pkgs · changelog symbols 40/40 ·
tag-content gate green · watermill/v4.6.2 + tursoengine/v4.2.1 proxy-served
· benchkit/cqrs-bench targeted tests green · 9-module quick tests green.
