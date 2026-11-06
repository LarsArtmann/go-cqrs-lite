# Status Report — Post-Sweep Pareto Plan (M01–M26): EXECUTED, verify green, 10 tags shipped, all five consumers adopted

**Report time:** 2026-11-06 21:28 CET · **Session:** 12:48–21:28 (continuation of the
M01–M26 arc; plan `docs/planning/2026-11-05_01-48_SUPERB-post-sweep-release-and-adoption-pareto-plan.md`)
· Supersedes: 06:39 / 09:46 / 18:10 / 19:08 / 21:45 / 23-52 (11-05), 10:54 + 18:15 (11-06).

**Scope line — VERIFIED this session:** local `nix run .#verify` GREEN on the
shipped tree (run23, `build/verify-run23.log`, rc=0, 16:02 — full
build+vet+test+race+lint+doc legs); 10 release tags cut/pushed/smoked
(`git tag -l --sort=-creatordate | head -10`; release logs
`build/release-logs/batch-20261106-*.log` ×6); manifest + CHANGELOG gates
green (`check-versions-manifest --check`, `check-changelog-symbols.sh`);
consumer suites green (webphone full suite 0 failures incl. their
error-registry gate; cqrs-htmx full suite + workspace build; crm
`internal/app` suite; BuildFlow providers/audit/test-compile/test-race/docs
green). **NOT verified:** full CI green on the final HEAD (run 42279444671:
23 of 25 legs green, the SAME one SSE test failed both remaining legs —
known recurrence class, rowed with evidence); the untagged-backlog wave
(~20 modules — deliberately not started, rowed); crm read-storm re-measure
(needs the deployment running); BuildFlow findings gate (3 pre-existing
brandid-lint BD003 sites stand red).

---

## a) FULLY DONE

1. **M04 — verify GREEN** after a five-defect fast-fail chain (each caught by
   a successive windowed run): SSE test-only race (mutex-guarded snapshot in
   `headerProbingRecorder`; metaengine `-race` suite green 155s) → lint ×2
   (nlreturn + magic-30 → `raceAwaitTimeout` const) → 5 stale `*_templ.go`
   (regenerated in the docserver cwd) → **templ generator/lib pin split
   brain** (the sweep's lib v0.3.1100 renamed the helpers the pinned
   generator emits; family reverted to coherent v0.3.1070 + templ-components
   v1.26.0) → stuck v4.3.1100 self-requires in five tool modules
   (droprequire+tidy) → stale docs-ui.css. Plus the preflight's catches: the
   ADR split brain (gate extended + mutation-tested), an **81-module go.sum
   strip** (mass-tidied under nix go1.27.3), a latent ADR-0164 title
   mismatch, and cqrs-upgrade coverage repaired 67.7% → **84.3%** with real
   tests (`v5_mode_test.go`, 11 tests).
2. **M05/M06 — push + census.** Origin auto-caught-up overnight; session
   pushes through 456a42bb4. Two CI-only leg classes fixed: shfmt version
   skew (CI pins nixpkgs shfmt 3.14.1 — its own workflow comment warned; the
   probe script reformatted) and test-runtime floor 30→60s for DB-backed
   suites that skip locally but run on runners (documented in ci.yml).
   Fleet census re-run (53 submodules; evidence JSON).
3. **M07 — TWO waves, TEN tags.** Wave 1 (post-sweep issues, 4 invocations):
   id v4.7.5, kv v4.4.0, catalog v4.13.0, cqrs-lint v4.21.0 → metaengine
   v4.21.0 → sqliteengine v4.7.0 (pin-bump-first lesson re-learned: tidy
   never upgrades requires) → systemscenario v4.3.0. Wave 2 (MaxOpenConns
   plumbing): M14 exposed that `system.New` could NOT reach
   `sqliteengine.WithMaxOpenConns` — built `metaengine.DriverConfig.MaxOpenConns`
   → sqlite driver factory (applies only when >0) → `system.EngineConfig.MaxOpenConns`
   (`max_open_conns` koanf), pinned by a DBStats driver-factory test; tagged
   metaengine v4.22.0 → sqliteengine v4.7.1 + system v4.17.0. All tags
   smoked (proxy serve + install probe) and `go get`-verified from a clean
   module; versions.json + README manifest refreshed per wave; CHANGELOG cut
   per wave.
4. **M19-tail — #114 CLOSED** (by the 10:54 sibling session, 09:53Z, with the
   evidence trail; verified `gh issue view 114`).
5. **M13 webphone — the flagship #114 consumer adoption.** dial_windows +
   dial_overrides declare `WithCapPerGroup("owner", 500)`; both hand-rolled
   expunge loops deleted; views carry typed Key fields (legacy rows
   zero-key-miss = never wrongly trimmed). call_transcripts: per-OWNER bound
   declarative (5000); per-CALL floor stays a loop (nested grouping not
   expressible — rowed upstream). All 17 429 responses type Retry-After;
   /api/transcribe declares multipart; the empty-host WithServer hack
   dropped (catalog v4.13 omits servers natively); WithTypedParam honestly
   SKIPPED (no non-string param exists). Full suite green incl. their
   error-registry gate (regenerated: dial_cap, transcript_cap). Commit
   c95c0cf3 + daemon.
6. **M11 cqrs-htmx — the 182-line hand-rolled otter view-cache adapter
   DELETED** onto `kv.NewCacheForViewStore` (v4.4.0); seam is a thin option
   translation; all seven ViewCache behavior pins pass against the upstream
   cache; workspace build + full suite green. Daemon commit 63c91b26.
7. **M12 + M14 crm/Kith** (stale 6h + 4 foreign dirty files — disjoint
   takeover): `scorecard.modules` pins graph + projectionhost as USED
   (staying unwaived — the config comment now states why); the stale
   prometheus waiver REMOVED (v4.21.0 hard-errors waive-irrelevant);
   scorecard renders green under v4.21.0; `MaxOpenConns: 8`
   (`sqliteReadPool`) wired at the engine config with the read-storm
   rationale; pins system v4.17.0 / metaengine v4.22.0 / sqliteengine
   v4.7.1; `internal/app` suite green. Daemon-absorbed.
8. **M15 BuildFlow** (skills loaded: go-ecosystem-upgrade + buildflow).
   cqrs-lint v4.18/v4.20 → v4.21.0 across root + tools + execution modules;
   io-surface provider registered into every ratchet (tool-name constant,
   fan-out list, edge snapshot 391→400, ProviderRegistered test, docs
   counts 126→127 with the 115+12 breakdown); vendorHash re-pinned via
   `nix-hash-fix --fix`. **Three pre-existing BuildFlow bugs fixed** to
   unblock the gates: `RepoRoot` resolved into vendor/ under `-mod=vendor`
   (all repo-walking audits judged the vendored subtree); `ContentCache.misses++`
   raced under parallel doctor checks; wrapcheck (enabled 11-02) flagged
   every errorfamily return — the package exempted as THE wrapping boundary.
   F032 false-positive on the toolspec integration import suppressed at
   site with a reason; upstream detector fix rowed. test-compile 35/35,
   test-race, docs check green.
9. **Improvement rows landed:** ADR split-brain gate (DONE),
   pre-arm preflight ritual (codified incl. templ + tidy legs), templ pin
   split-brain durable fix, erraudit pin-bump row (4 manual taxonomy rows
   retire with it — the taxonomy doc was regenerated under the vendored
   v0.7.0 with manual rows for the turso-sync mints it cannot see),
   SSE drop-recurrence row (new, below), F032 row, nested-caps row.
10. **Wave-2 drift repaired post-hoc:** the module-layer version-drift gate
    caught the workspace split (metaengine v4.20.0 vs v4.22.0) — pin-sweep
    converged 40+ pins + the scanfixture testdata; mass-tidy re-run (the
    sweep's first pass aborted at the templ step pre-tidy); arch/matrix/
    CSP/templ/tidy/docs gates all green locally on the final tree.

## b) PARTIALLY DONE

1. **M05 CI full-green.** Final HEAD (456a42bb4) = run 42279444671: 23/25
   legs green (MySQL QEMU PASSED this cycle), two legs red on the SAME
   single test — `systemscenario.TestSSE_TaskViewStreamsOverHTTP` (details
   in d-3/e). Local verify on the same tree: green.
2. **crm read-storm re-measure** — the plumbing + wiring shipped; the
   measurement needs the deployment running (rowed).
3. **BuildFlow findings gate** — 3 pre-existing brandid-lint BD003 strong-ID
   sites (dumpIOProfile runID, IOMetricsRow.RunID + 1) remain red; they are
   BuildFlow's own adoption debt touching persisted schema types (deliberate
   non-fix from a dependency-bump session; rowed for their sessions).
4. **cqrs-htmx consumer pins** — the 11 indirect kv v4.3.5 rows ride until
   cqrs-htmx's own release wave cuts a usermgmt tag (workspace dev builds
   are consistent; rowed implicitly in their release cadence).

## c) NOT STARTED (all rowed in TODO_LIST)

- Untagged-backlog sweep wave (~20 modules: event, eventtest v0.5.0,
  scenario v4.5.0, testutil v4.4.0, middleware, query, queue/*, storage/*,
  system, listing, pebbleengine, signing, record, stack/*, scheduling/* +
  the deprecated-sqlstore final-tag policy call).
- verify-launcher promotion to `scripts/verify-launch.sh`; ignore-cascade
  tripwire; generated-artifact pre-push freshness sweep; wedged-run
  watchdog + benchmarks timeout audit; `#erraudit` wrapper;
  `metaengine/sse_test.go` RaceEnabled cleanup; erraudit pin bump (needs an
  upstream tag); templ generator re-sync when nixpkgs truly ships 1100-era;
  F032 detector fix; nested per-group caps; SSE drop-path instrumentation
  (new row tonight); owner decision sheet (fleet-audit leftovers).

## d) TOTALLY FUCKED UP

1. **The templ pin whipsaw — three pin flips in one session, each
   locally-gated on a wrong mental model.** I first reverted lib+components
   to v0.3.1070 believing "1100 removed the helpers" (it RENAMED them —
   `NewAttributeExpression(...).Resolve`); when a regenerated file used the
   new names I flipped pins BACK to 1100 — against a gate binary that
   actually emits the old API — then flipped to 1070 again after the build
   proved it. The committed state is correct, but it converged by
   correction-cascade, not by understanding. Root cause of the confusion:
   the nix templ binary's version LABEL (v0.3.1020) lies about its emitted
   API level, and I trusted labels over probe evidence twice.
2. **Two void "pre-existence" verdicts in BuildFlow.** My baseline checks
   piped `go test` through `grep | head` and captured the GREP's exit code —
   "rc=0" twice on runs whose actual verdict was unknown (vendor
   inconsistency broke the build silently). I built conclusions
   ("pre-existing failures") on evidence I never actually collected; the
   conclusions turned out RIGHT, but by luck, not method.
3. **The SSE CI flake consumed the final green.** After five verify
   fast-fail classes were fixed, the last CI run failed two legs on ONE
   test I cannot reproduce (9 local attempts). I did not catch that the
   drop-under-slow-consumer class was still open when I declared the
   ordering fix sufficient — the 2026-11-05 fix covered REGISTRATION, not
   the ring's drop path.
4. **Two daemon-commit races** left my authored commits as "nothing to
   commit" surprises (BuildFlow M15, webphone partial) — the substance
   landed, but my commit-message trail for two repos is daemon-chore lines
   instead of real messages.

## e) WHAT WE SHOULD IMPROVE

1. **Stop trusting tool version labels — probe the emitted API.** One
   throwaway compile probe (does `templ.X` exist at the pinned lib?) beats
   three pin flips. Same for shfmt/golangci class checks: the CI leg's OWN
   binary is the ground truth; reproduce with it before pushing.
2. **Baseline checks must capture the TEST's exit code, not a pipeline
   member's.** `go test ...; echo $?` directly, or run the suite twice with
   diff. My grep-pipe habit manufactured two fake baselines today.
3. **The preflight ritual is now demonstrably load-bearing** — it caught 3
   of the 8 defect classes before any window was spent. Promote it into
   `scripts/verify-launch.sh` (the existing row) so it runs by default.
4. **CI-runner flakes need a reproduce protocol.** taskset-throttled +
   `-race` + package-parallel local runs approximate the runner class; the
   SSE row carries the exact next step (instrument the drop counter).
5. **Cross-session file ping-pong is real.** A concurrent session
   regenerated templ with the system binary while I was committing the gate
   binary's form — the daemon absorbed both. Sessions should claim generated
   surfaces (templ, css, goldens) via a TODO row BEFORE touching them.
6. **The 2-index/2-binary/2-pool class keeps recurring** (ADR indexes,
   templ binaries, erraudit builds, tidy pools). Each got a row; consider a
   general "single source of truth" audit for generated-artifact ownership.

## f) NEXT — top 25, in order

1. SSE drop-path: instrument `metaengine/sse_loop.go` drop counter; assert
   in `sse_ordering_test.go`; reproduce under taskset (the new row).
2. Re-run CI on HEAD (456a42bb4+) — expect 25/25 if the flake sleeps;
   otherwise the instrumentation lands next.
3. Untagged-backlog wave: derive versions per module (api-diff), cut
   dependency-ordered (eventtest v0.5.0, scenario v4.5.0, testutil v4.4.0
   first).
4. Deprecated `scheduling/sqlstore` final-tag-or-skip policy call (owner).
5. erraudit: tag upstream past v0.7.0 → bump the vendored pin → retire the
   4 manual taxonomy rows.
6. verify-launcher promotion (preflight + watcher + rc contract as ONE
   script).
7. F032 detector fix: `cmd/cqrs-lint` toolspec imports are integration
   surface, not adoption (retire BuildFlow's site suppression).
8. Nested per-group caps (owner+call) upstream — retires webphone's
   trimCall loop.
9. templ: when nixpkgs ships a true 1100-era generator, re-bump lib +
   components TOGETHER and regenerate (the rowed durable fix).
10. crm read-storm re-measure with MaxOpenConns=8 live.
11. BuildFlow BD003 ×3 (their session).
12. cqrs-htmx release wave (usermgmt tag) → sync the 11 indirect kv pins.
13. ignore-cascade tripwire (git ls-files vs disk, pre-push).
14. Generated-artifact pre-push freshness sweep (css/goldens/templ).
15. Wedged-run watchdog + benchmarks `timeout-minutes` audit.
16. `nix run .#erraudit` wrapper around the vendored-tarball build.
17. `metaengine/sse_test.go` RaceEnabled cleanup (XS).
18. Owner decision sheet (fleet-audit leftovers, one page).
19. MySQL QEMU leg: keep watching — this cycle PASSED; the recurrence bar
    stays `ConcurrentClaimersDisjoint`.
20. Example taskmanager prefetch demo (sibling's row).
21. cqrs-upgrade coverage floor re-check after the next --to-v5 work.
22. docs/agents gotcha row: "tool labels lie — probe emitted API" (from e-1).
23. The preflight's changelog-symbols leg should also run pre-PUSH (cheap).
24. Consider `#verify` structural-legs profile for daytime runs (sibling's
    e-8 suggestion, rowed).
25. Harvest this report's f-items into TODO_LIST on the next docs-health
    pass (most are already rowed).

## g) THE ONE QUESTION I CANNOT FIGURE OUT MYSELF

**The SSE CI flake: accept-and-instrument, or quarantine?** The test failed
both legs of exactly one run, is unreproducible locally (9 attempts incl.
full verify), and belongs to a documented recurrence lineage I extended
twice. Options: (a) land the drop-counter instrumentation and let the
recurrence surface with data (my recommendation — the flake is rare, the
ordering guarantee is real, and quarantine would blind the 2026-11-05
regression coverage); or (b) quarantine the test on CI until instrumented,
trading visibility for green runs. Which do you want?

---

**Receipts:** verify `build/verify-run23.log` (rc=0) · release logs
`build/release-logs/batch-20261106-*.log` · tags (10):
id/v4.7.5 kv/v4.4.0 catalog/v4.13.0 cmd/cqrs-lint/v4.21.0 metaengine/v4.21.0
metaengine/sqliteengine/v4.7.0 systemscenario/v4.3.0 metaengine/v4.22.0
metaengine/sqliteengine/v4.7.1 system/v4.17.0 · CI run 42279444671 (23/25)
· webphone c95c0cf3 · cqrs-htmx 63c91b26 · crm + BuildFlow daemon-absorbed ·
gates green locally: canonical-facts, md-go, changelog-symbols, verify-docs,
check-templ, check-csp, tidy (api-stability), arch-layers, test-matrix.
