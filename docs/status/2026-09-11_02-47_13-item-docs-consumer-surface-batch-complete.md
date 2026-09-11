# Status: 13-Item Docs/Consumer-Surface Batch — Complete (6 items were stale)

**Date:** 2026-09-11 02:47 CEST
**Session scope:** execute the pasted TODO batch (docs/consumer-surface truth + testing/tooling tail), one item at a time, verified.
**Tree state at writing:** all session work absorbed by the auto-commit daemon; only `docs/api_surface.txt` dirty (concurrent session's cqrs-lint lintutil additions — not mine, left alone).

---

## Session timeline

| Time (CEST) | Work |
| --- | --- |
| 02:04–02:10 | Researched all 13 items against repo state. Found **6 of 13 already shipped** (stale TODO entries): #3 lying-log, #4 benchmark pin, #7 quick-ref rows, #10b linter-name golden, #11 templ tripwire, #12 AGENTS split. Found #5's big window already cured by the `benchkit.not_started` guard + load-scaled budgets shipped in an earlier wave. |
| 02:10–02:20 | Doc items: DOMAIN_LANGUAGE.md (+4 entries), error-taxonomy.md (watermill lie fixed + gaps), encryption README/doc.go, benchmark allowlist comment. |
| 02:16–02:19 | exhaustruct_v5 canary script + `#check-lint-config` wiring. Hit and fixed a real tooling bug (TMPDIR-inside-fixture breaks golangci typechecking). |
| 02:18–02:21 | Scoped `#doc-check` flake app; doc-check release posture decided (ship-as-is). |
| 02:21–02:26 | Skill-reference propagation: rotation write-back recipe, MySQL claiming matrix FIX, planned-tables roster. Ran into daemon mtime conflicts on recipes.md; re-read and re-applied. |
| 02:22–02:29 | Benchkit mid-run TOCTOU fix (`benchkit.expired`) + system test deadline 30s→90s×factor. Verified under 64-way CPU soaker (load 36–52) with `-race`. |
| 02:29–02:35 | Watermill: skew-recovery property test, journal-park test (FullBuffer retired as unreachable fiction), Redis Streams throughput variant. |
| 02:35–02:37 | GitHub homepage set (pkg.go.dev); social preview asset generated (SVG+PNG). API upload attempt → 404 (UI-only). |
| 02:38–02:46 | `nix fmt`, golangci ×3 modules (0 issues), doc-check (1273 refs ✓), changelog-symbols ✓, TODO_LIST evidence, CHANGELOG entry, full benchkit suite. |

---

## a) FULLY DONE

1. **DOMAIN_LANGUAGE.md entries** — Materialized-View Acceleration, IVM, View-Maintained Write (Metaengine table, ADR-0135-grounded) + At-Rest Encryption (Security table, two-model framing: payload AEAD vs `tursoengine.WithEncryption`, Cloud-BYOK boundary).
2. **error-taxonomy.md completeness** — pebble section verified accurate against source; watermill table had a **real lie** ("Metadata parse fails → Corruption"; every `watermill.parse_*` site is Rejection, verified per-site) — corrected; added missing rows: malformed-metadata Corruption, catch-up checkpoint/replay Infrastructure, subscribe/publish/lifecycle Infrastructure.
3. **encryption module docs** — README gained Loading & Validation (LoadKeyFromEnv/LoadKeyFromFile/ValidateKey/Encode/DecodeKeyBase64 + error-surface contract); doc.go gained Key Management Helpers, Envelope Format (v2), and Snapshot-State Rotation Write-Back sections. Goldens + v1↔v2 symmetry property test verified already existing and green.
4. **Benchmark auto-discovery** — answered (no auto-discovery; explicit `$`-anchored allowlist) and the decision is now a comment in `benchmark-regression.sh` so nobody "widens" it into flake territory.
5. **exhaustruct_v5 canary** — `scripts/test-exhaustruct-canary.sh`, wired into `#check-lint-config`: static layer (3 patterns present; stack + bbolt targets exist) + behavioral layer (hermetic stdlib fixture: with patterns → only control flagged; without → both flagged). Proves v5 full-name semantics under the installed golangci-lint 2.13.2.
6. **Scoped `#doc-check` flake app** — same corpus as the #verify leg, standalone runnable. Green.
7. **doc-check release posture** — DECIDED: ship the strict block-scoped resolver as-is at the next cmd/doc-check tag, no `--legacy-union` flag (internal-grade tool; stricter = fewer false passes; a transition flag is permanent maintenance for a tiny audience). Recorded in TODO_LIST.
8. **Skill-reference propagation** — recipes §2.31 now documents `RotatingSnapshotStateCodec` lazy write-back + manual rewrite path; §2.7 cross-references it. **MySQL claiming matrix corrected** — §2.26 claimed `NewClaimingMySQLStore` always fails `ErrClaimingUnsupported`; source says it works on MySQL 8.0+/MariaDB 10.6+ (SKIP LOCKED) — replaced with a real support matrix. Planned-table capability roster added (§2.27): all four engines (pg/mysql/sqlite/duckdb) implement ApplyLayoutPlan/EvolveLayoutPlan/PlannedTables; backfill pg+mysql only.
9. **Benchkit flake root cause + residual fix** — the historical nil-error window (setup burns ctx) was already fixed upstream by the `not_started` guard; the remaining TOCTOU (ctx expires after the guard, every phase silently skips, `Run` returns `(partial, nil)`) is now closed: `runPhases` returns Transient `benchkit.expired` when a caller-bound (non-Duration) ctx is expired and `TotalEvents == 0`. Duration-bounded runs keep graceful-partial semantics. CHANGELOG Fixed entry written.
10. **system.TestSystem_ResetProjection_RestartAndReplay headroom** — outer deadline 30s → 90s×load-factor (two sequential 15s inner budgets + close/reopen previously consumed the whole budget at factor 1).
11. **Watermill skew-recovery property test** — `TestCatchUpSubscriber_RestartRecoversSkewSuppressedEvent`: mints a zero-timestamp-ULID event appended AFTER the watermark, proves live suppression, then proves restart re-delivers it (journal-order ReadFrom). Pins the documented self-healing claim exactly. Green under `-race`.
12. **Watermill deterministic Close test** — `CloseWhileBlockedOnFullBuffer` RETIRED as unreachable fiction (replay forwards are serialized with awaitAck — the 256-slot buffer can never fill; the old test's tolerant assertions passed vacuously). Replaced by `TestCatchUpSubscriber_CloseWhileReplayParkedInJournal` (gating journal, deterministic park inside ReadFrom, no sleeps; also pins Subscribe-after-Close failure).
13. **Watermill Redis throughput variant** — `TestRedisStream_CatchUpReplayThroughput`: real Redis Streams live side via `ephemeral-redis.sh` pattern, 1000 events with order pinning, replay→live handoff, log-only throughput (318k events/sec locally). `#integration-redis` already runs the whole package, so no CI wiring needed.
14. **GitHub homepage** — set to `https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite` via `gh repo edit` (the item's "CLI can't set it" was stale for homepage).
15. **Social preview asset** — branded 1280×640 card at `docs/assets/social-preview.{svg,png}` (SVG source committed for regeneration).
16. **TODO_LIST.md** — all 13 batch items marked `[x]` with evidence notes.
17. **Verification sweep** — doc-check 1273 refs valid; changelog-symbols honest (20 citations); `#check-lint-config` full gate green; golangci 0 issues on watermill/benchkit/system; watermill full suite green; benchkit full suite green (see d-4 for run 1); encryption golden/symmetry green; soaker+race runs green.

---

## b) PARTIALLY DONE

1. **Social preview upload** — the undocumented GitHub API endpoint returns 404; the paste into Settings → Social preview is **manual and yours**. Asset is ready.
2. **Benchkit flake verification depth** — I verified the fix under synthetic soaker load (36–52) + `-race` on the ClosedStore/ExpiredContext and system tests, and the full benchkit suite green — but I did **not** reproduce the *actual* full 84-module `#verify` contention profile, and I did not individually re-run `TestRun_Pebble`/`TestRun_Recovery_Pebble` under load (their flake was already addressed by the earlier mustRun budget change; my change targets the ClosedStore class specifically).
3. **doc-check ship-as-is decision** — made unilaterally and recorded, but it is a release-posture call that deserves your ratification (g3 asked the question originally).
4. **error-taxonomy scope** — only pebble/watermill were depth-checked per the item's wording. The other module tables (middleware, graph, relational, projectionhost, transport/grpc) were NOT re-verified against source this session — and watermill proved those tables *can* lie.
5. **§2.28 "everything here is live on pgengine + mysqlengine"** — with sqlite/duckdb now having Apply/Evolve/PlannedTables, the D3-set framing is partially stale (backfill really is pg+mysql only, so it's not flatly wrong). I fixed the one flat contradiction (`// pg + mysql` on the evolver line → now names all four, §2.27 roster) but left the broader re-verification for a docs pass.

---

## c) NOT STARTED (noticed this session, deliberately deferred)

1. **runSubscription shutdown-noise log** — cousin of the fixed lying-log item: a Close that interrupts `awaitAck` makes replayPhase return `ctx.Err()`, and the runSubscription goroutine then logs `ERROR "catch-up replay failed: context canceled"`. Truthful but noisy on every deliberate Close. Left out of scope; **not recorded in TODO_LIST** — recording it here first.
2. **docs-health HARVEST** of this report's §f into TODO_LIST/ROADMAP — not run (you asked report-then-wait).
3. **Redis test hygiene** — `TestRedisStream_CatchUpReplayThroughput` leaves its consumer group/topic behind on the ephemeral instance (same as the existing edge tests; harmless on ephemeral, would matter on a long-lived Redis).
4. **Stale-doc epidemic process fix** — 6/13 pasted items were already done: nothing enforces "mark the checkbox when the work lands". I updated checkboxes for MY work this session, but the pattern will recur with batch-pasted lists.
5. **status-report skill HTML default** — its SKILL.md prescribes styled HTML at `docs/reviews/`; per your explicit `.md` instruction (and last session's precedent) I'm writing markdown here. The skill default remains unpropagated/unreconciled.

---

## d) TOTALLY FUCKED UP

1. **Multiedit old_string typo** (`ADT\"` transcribed instead of `ADT**`) — a pure sloppiness roundtrip lost because I didn't copy the exact text from the View output.
2. **Edit-tool mtime false alarm** — when recipes.md rejected my first multiedit ("modified since read"), I burned a git-status investigation suspecting the concurrent session of content changes; actually the tracker only refreshes on View reads (bash `sed` reads don't count), and the daemon's staging had bumped mtime. Cost: two roundtrips. Lesson internalized: after ANY failed edit, View first, diagnose the tracker, then re-apply.
3. **golangci fixture rabbit hole** — the canary failed with a typechecking error and I re-read script logic twice before bisecting env vars; the cause (`TMPDIR` pointing INTO the fixture dir breaks go/packages) took three roundtrips that a disciplined one-variable-at-a-time bisect would have taken one.
4. **Benchkit run-1 failure never diagnosed** — the first full-suite run failed with only `🚫 Container terminated` + FAIL; I re-ran blind instead of capturing the failing test name. Run 2 passed fully and run 3 confirmed PG tests skip without a container — so my "infra flake" verdict is *circumstantial, not proven*. I never identified which test failed in run 1. Honest gap in the verification log.
5. **Roster created a contradiction before catching it** — my §2.27 roster says all four engines have `LayoutPlanEvolver`; §2.28's code comment still said `// pg + mysql`. I only caught it during report writing (now fixed), but it should never have shipped in the same edit that created the roster — I had the source facts in hand and didn't re-scan the section I was editing.
6. **Redis test couples to the string literal `"event_id"`** — `metaEventID` is unexported, so the external-package test hardcodes the value. If `protocol.go` ever renames the key, the test breaks non-obviously. Small, but I introduced it knowingly for convenience.

---

## e) WHAT WE SHOULD IMPROVE

1. **Stale-TODO hygiene as a hard rule** — every completion updates the checkbox *at the moment of landing*, not in a later bookkeeping pass. 6/13 items in this batch were zombie entries; zombie lists cause re-research cost every session.
2. **Tests that cannot fail are fiction** — `CloseWhileBlockedOnFullBuffer` asserted `delivered ≤ total` and passed *vacuously* since 07-42. Tolerant assertions were praised in the 07-42 notes ("safe"), but "safe + vacuous" is worse than "flaky + meaningful". Add a review smell: any assertion that a known-good run cannot violate is not a test.
3. **error-taxonomy needs a generated-from-source gate** — the watermill table lied for weeks because docs-as-prose drift silently. A script extracting `errorfamily.*` codes per module and diffing against the doc (the `check-linter-names.sh` pattern) would make family-code drift loud.
4. **Zero-headroom deadlines are flakes-in-waiting** — the 30s system-test "fix" matched ONE inner budget and left phase 2 to starve. Rule: any outer deadline must exceed the SUM of inner budgets plus overhead, ≥2× preferred.
5. **Concurrency with the daemon** — batched multiedits against daemon-managed files should View→edit in the same breath; mtime conflicts are guaranteed otherwise. (Also: the daemon absorbing work mid-session makes `git diff` an unreliable "what changed" tool — read the diff BEFORE the daemon eats it.)
6. **Capture failures before re-running** — run 1's benchkit failure is now forever unknown because I re-ran without recording the failing test. Always `-v | grep FAIL` the FIRST time.
7. **Canary coverage is partial by design — say so loudly** — bbolt/stack ignore-patterns are verified statically (presence + target existence), only `os/exec.Cmd` is verified behaviorally. The header comment discloses this; a go.work-bridge behavioral leg could close it in a sandbox-tolerant follow-up.
8. **External tests should not hardcode unexported constants** — export the metadata keys (or accessor funcs) so cross-package tests bind to symbols, not strings.
9. **Batch reports should lead with the stale count** — "6 of your 13 items were already done" is the most valuable sentence in this session and it should be the first thing you read, not a section.

---

## f) NEXT TASKS (prioritized; ★ = carried over open from earlier sessions, unstarred = born this session)

| # | Task | Effort |
| --- | --- | --- |
| 1 | **YOU:** paste `docs/assets/social-preview.png` in GitHub Settings → Social preview (API is 404) | 1 min, manual |
| 2 | Ratify or veto the doc-check ship-as-is release posture (§g-2) | 1 min, decision |
| 3 | Fix runSubscription shutdown-noise: don't log ERROR on `context.Canceled` during deliberate Close (or record won't-fix) | XS |
| 4 | Extend error-taxonomy verification to ALL module tables vs source (middleware/graph/relational/projectionhost/grpc) | S |
| 5 | Build the error-taxonomy drift gate: extract `errorfamily.*` codes per module, diff vs doc (check-linter-names pattern) | S |
| 6 | Re-run `TestRun_Pebble`/`TestRun_Recovery_Pebble` under REAL full-`#verify` contention to confirm their flake class is dead | M |
| 7 | Diagnose this host's PG-testcontainer termination (run-1 benchkit failure): is a container runtime supposed to work here? (§g-3) | S |
| 8 | Export watermill metadata key constants (or accessors) so external tests stop hardcoding `"event_id"` | XS |
| 9 | Make `eventtest.FakeStore.ReadFrom` honor ctx (currently ignores cancellation — my gating journal had to work around it) | XS |
| 10 | Add Redis test group/topic cleanup to the edge tests (matters the day Redis isn't ephemeral) | XS |
| 11 | ★ Add `check-retracts-shipped.sh` gate (retracts are inert until a tag carries them — 2026-09-11 incident class) | S |
| 12 | ★ Add `--baseline` mode to `tag-release.sh --audit` (24 known historical violations) so it can become a CI leg | S |
| 13 | ★ Audit `scripts/batch-release.sh` for pre-hardening flow (no --smoke/--audit usage) | M |
| 14 | ★ Run `scripts/pin-sweep.sh --check` post-push (new tags may stale sibling pins) | XS |
| 15 | ★ Watch CI on the release commits + first nightly `upgrade-dogfood` run | XS |
| 16 | docs-health HARVEST: pull §f items into TODO_LIST/ROADMAP properly | M |
| 17 | Wire a "stale-checkbox killer": script comparing TODO_LIST `[ ]` items against shipped evidence (tags/code) quarterly | M |
| 18 | Tag next wave carrying this session's module changes (watermill tests, benchkit fix, encryption docs) when you say release | M |
| 19 | Behavioral exhaustruct canary leg for bbolt/stack patterns via go.work bridge (sandbox-permitting) | S |
| 20 | Canary-style behavioral checks for other semantics-sensitive linter settings (ireturn allow-list, importas) | M |
| 21 | Fixture test pinning benchmark-regression.sh's default BENCH regex (so the allowlist can't silently widen) | XS |
| 22 | recipes §2.28: re-verify the "D3 set live on pg+mysql" framing vs sqlite/duckdb parity and update | S |
| 23 | Give test-only metadata literals a doc-comment cross-link to `metaEventID` (minimum) if #8 is vetoed | XS |
| 24 | Record the runSubscription noise item in TODO_LIST if #3 is deferred | XS |
| 25 | Reconcile status-report skill's HTML default vs your .md preference (update SKILL.md or accept the standing override) | S |
| 26 | ★ Badger question (external consumers of v4.0.0–v4.1.0?) — still unanswered from last session | decision |
| 27 | ★ Dead-path example modules (taskmanager/getting-started invisible v3/v4 tags) — re-path/freeze/delete | decision |
| 28 | ★ Autonomous tag-push authorization — still unanswered | decision |
| 29 | Add a canary that `nix run .#doc-check` (new app) and the #verify inline leg stay corpus-identical (they can drift apart) | S |
| 30 | Doc-check corpus: include `docs/error-taxonomy.md` (family-code tables are consumer surface and currently unchecked) | XS |
| 31 | error-taxonomy: add the deriver/deriver-saga and commandlifecycle family sections (modules shipped, doc silent — unverified this session) | S |
| 32 | Consider `errorfamily` code-lint rule (cqrs-lint): codes must match `module.subdomain.error` shape — kills typos at write time | M |
| 33 | Watermill: `CloseWhileBlockedOnAck` + journal-park tests deserve a shared "park points" doc-comment (awaitAck vs ReadFrom are THE two states) | XS |
| 34 | Benchkit: consider surfacing `benchkit.expired`/`not_started` codes in the README error table | XS |
| 35 | SOCIAL/branding: SVG uses DejaVu — regenerate with the project's real brand font if one exists | S |
| 36 | Homepage decision follow-up: pkg.go.dev now; a docs site later (website-launch skill) would supersede it | decision, later |
| 37 | The system hardening test file has other fixed-deadline tests — audit them for the same zero-headroom pattern | S |
| 38 | warnings_test.go five fixed 60s ctxs are NOT load-scaled (found in research, not fixed — out of my items' scope) | S |
| 39 | phases_journey_test.go fixed 10s ctx — same class as #38 | XS |
| 40 | Sweep for remaining `time.Sleep`-based synchronization in watermill/system tests (FullBuffer was one of several) | M |
| 41 | CHANGELOG: decide whether test-only additions get entries (this session's watermill tests got none; the matview entry earlier cited tests) — write the rule down | decision |
| 42 | add-historical-banners.sh: check whether new docs (ADR-0137 addendum etc.) need banners per policy | XS |
| 43 | docs/status/README.md index: add this report (and check last session's is indexed) | XS |
| 44 | `#integration-redis` duration check: new throughput test adds ~1s; confirm the app's timeout budget still fits | XS |
| 45 | Consider promoting `gatingJournal` into eventtest as a reusable test double (second consumer = the watermill suite itself) | S |
| 46 | API golden: confirm no exported-symbol drift from this session (I believe none; run `cmd/api-stability` meta-test to prove) | XS |
| 47 | Run `nix run .#check-doc-links` over the edited docs (doc-check validates code refs, not markdown links) | XS |
| 48 | Load-sweep parity: `nix run .#load-sweep` covers Latency/Timer/Deadline timing tests — consider adding the ClosedStore class to its -run set | S |
| 49 | §2.27 roster + §2.28 backfill note should cross-link Doctor's `record-context` section (advanced.md §7 mentions the tooling, not these sections) | XS |
| 50 | Kill the zombie: after #17, do a one-time full TODO_LIST audit for items silently shipped by other sessions (this batch's 6 were found by luck of research depth) | M |

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Social preview style + homepage target** — I set homepage to pkg.go.dev (canonical, verifiable) and generated a GitHub-dark card with DejaVu type. Do you want this asset pasted as-is, or does the repo have a brand direction (font/palette/future docs site) the asset and homepage should target instead?
2. **doc-check release posture** — I decided ship-as-is (no `--legacy-union` transition flag) on the grounds that doc-check is internal-grade and stricter is safer. Ratify, or do you want the flag before the next `cmd/doc-check` tag?
3. **PG-testcontainer on this host** — benchkit run 1 died with `Container terminated`; runs without a container skip the PG tests cleanly. Is a container runtime (docker/podman) *supposed* to work in this environment (i.e. should I chase the run-1 failure), or is containerless the norm here (i.e. accept that `TestRun_Postgres*` only runs in CI)?

---

## Appendix: Self-review (11 questions, blunt)

1. **What did you forget?** The runSubscription shutdown-noise log (c-1) — noticed, deliberately deferred, then failed to record it anywhere until this report. The §2.28 evolver contradiction — shipped in the same edit window as the roster that contradicted it.
2. **What is something stupid that we do anyway?** Zombie checkboxes in TODO_LIST (6/13 this batch). Vacuous tolerant assertions praised as "safe". Zero-headroom deadline fixes that are flakes-in-waiting.
3. **What could you have done better?** Copy old_string from View output verbatim (edit #1 failed on a transcription typo). Bisect env vars first when the canary failed. Capture the failing test name before re-running benchkit. Re-scan the recipes section I was editing for contradictions my own roster would create.
4. **What could you still improve?** Verification depth: synthetic soaker load ≠ real `#verify` contention; error-taxonomy checked 2 of ~12 module tables; canary is behavioral for 1 of 3 patterns.
5. **Did you lie to you?** No. But run-1's "infra flake" verdict is circumstantial — stated as such in d-4 rather than dressed up as proven.
6. **How can we be less stupid?** See e-1/e-2/e-3: checkbox-at-landing, no vacuous tests, generated-vs-doc gates for taxonomy tables.
7. **Ghost systems?** None created. The `#doc-check` app overlaps the #verify inline leg by design (scoped standalone use); a drift-pinning canary for the two corpora is filed as #29. `gatingJournal` lives in watermill tests — promotion to eventtest filed as #45 rather than left as a private ghost.
8. **Scope creep?** Resisted: no exported API additions (redis variant uses the raw `message.Subscriber`; no golden regen needed), no `--legacy-union` flag, no new deps (redis already in watermill's go.mod). The two on-sight fixes (evolver comment, watermill table) were within the standing trivial-fix permission.
9. **Did we remove something useful?** `CloseWhileBlockedOnFullBuffer` — deliberately, with the reasoning preserved in the replacement test's comment (the parkable-states analysis). Its one unique property (Subscribe-after-Close fails) was carried over.
10. **Split brains?** Two created-then-caught: the §2.27/§2.28 evolver contradiction (fixed) and the `#doc-check` app vs `#verify` inline corpus (documented, canary filed). The `"event_id"` literal is a soft split brain between protocol.go and the external test — filed #8.
11. **Tests?** Added 3 meaningful tests (skew-recovery, journal-park, redis throughput); retired 1 vacuous test; ran race + soaker-load verification. Gaps: no full-#verify reproduction, PG tests unverifiable host-side, and the suite's remaining sleeps/scaled-deadline debt is catalogued (#37–#40) not fixed.

## Appendix: Verification log

| Gate | Result |
| --- | --- |
| `nix run .#doc-check` | ✓ 1273 references, 64 packages (run twice: after skill refs, after TODO_LIST) |
| `nix run .#check-lint-config` | ✓ config verify + depguard + formatters + linter-names (109) + exhaustruct canary |
| `scripts/check-changelog-symbols.sh` | ✓ 20 citations honest |
| golangci-lint (watermill, benchkit, system) | 0 issues ×3 |
| watermill suite (`GOWORK=off`, jsonv2) | ✓ 0.31s |
| watermill catchup tests `-race` | ✓ 1.26s |
| Redis throughput test (`ephemeral-redis.sh`) | ✓ PASS, 1000 events / 3ms |
| benchkit full suite | run 1 FAIL (PG container infra, test name not captured — see d-4); run 2 ✓ 98.8s; PG-only run skips gracefully |
| benchkit ClosedStore/ExpiredContext under 64 soakers, load 36–52, `-race` | ✓ 9.3s |
| system ResetProjection under soakers, `-race` | ✓ 1.4s |
| encryption golden/symmetry tests | ✓ |
| `bash -n` on new script | ✓ |
| `nix fmt` | ✓ (1 file reformatted: line-wrap in my new test) |
| `gh repo edit --homepage` | ✓ verified via `gh repo view` |
| Social preview API upload | ✗ 404 — UI-only, owner paste |
