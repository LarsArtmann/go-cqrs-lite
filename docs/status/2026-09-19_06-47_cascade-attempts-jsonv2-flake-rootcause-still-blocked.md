# Status Report — Cascade Attempts, json/v2 Flake Root-Cause, Still Environmentally Blocked

**Report time:** 2026-09-19 06:47 (session window: 2026-09-18 ~18:45 → 23:30; morning state re-checked at report time)
**Session scope:** Continuation of the ADR-0141 follow-up backlog (verify cascade + CHANGELOG + post-split gates). This report covers only what THIS session did and noticed. Predecessor artifacts: `2026-09-18_18-11_session-self-review-temporal-followups-cascade-blocked.md`.

---

## Self-review answers (asked first, answered first)

**What did you forget?**

1. To institute a **tree-stability check before the first cascade attempt** — I gated on machine load only. The other session's commits landed during my "quiet" windows (18:54 during post-split work, 20:19–20:23 during cascade run 3). The 5-min-stable watcher I eventually built should have existed before attempt 1.
2. The **file-size baseline pre-check** before editing `exporter.go` (a baselined 382-line file): my +8-char nolint pushed the line past the formatter's wrap threshold → 384 lines → baseline growth. Caught by the fmt gate after the fact, not before the edit.
3. That `/tmp` logs evaporate overnight — my verify-fast logs from last night are gone; the evidence below is what I captured in the final report at 23:30. Log copies should have gone to a durable location.

**What could you have done better?**

1. The docserver flaky-test reproduction: ~10 CLI variant runs (toolchain store binaries, GOEXPERIMENT on/off/none, tag matrix, GOWORK on/off) before I read the test file's **imports** — the answer was on line 5 (`encoding/json/v2`, whose Marshal does not sort map keys). READ THE CODE FIRST; theorize second.
2. My last-night closing headline said "Session complete" while the primary deliverable (a green cascade) was unverified. The body was accurate; the headline overstated.
3. Attempt accounting discipline: runs 3–4 of verify-fast (~40 min machine time) were partially invalidated by mid-flight tree churn. After run 2's clean flake-only failure, the correct next move was the stability watcher, not another immediate retry.

**What could you still improve?**

1. Convert the stability watcher into `scripts/wait-for-quiet.sh` (load + no-commits + no-edits, N consecutive minutes) so future sessions gate composed runs on it mechanically.
2. Push the json/v2 byte-comparison audit (see e/2) — the docserver test was one member of a _class_; there may be more.
3. Fix-or-deprioritize the two timing-flaky tests (system starvation flake: 3 strikes in one day; queue/sqlite claim-expiry) — they are now the highest-probability blockers of every composed gate on this repo.

---

## a) FULLY DONE (verified green by a command run this session)

1. **Todo-list recreation + §g check** — the 18:11 report's three questions remain unanswered by any file/chat evidence; defaults applied (strict cascade exclusivity).
2. **CHANGELOG `[Unreleased]` Added** — cited `analyzer.StoreBigTable` (bullet in the ADR-0141 section) + new section for `catalog.DeliveryExactlyOnce`/`catalog.DeliveryAtLeastOnce`. Gate: `check-changelog-symbols.sh` → **118 citations, exit 0**. (This was the predecessor's "real miss".)
3. **`#check-duplication`** after the parser 3-way split — **0 new clone groups** (baseline 60), exit 0.
4. **Lint `cmd/cqrs-lint` → green** by fixing **14 findings**:
   - my regression: `StoreBigTable` missing from the exhaustive switch in `rules/api/a009_a013.go` (added to the keep-generic group — no stack preset exists for bigtable);
   - `toolspec.go` unconvert (`Iteration.Applied` is already `int`);
   - 12 from the 09-17 session: two `.golangci.yml` exclusions (`rules/*/template.go` → gochecknoglobals; `toolspec/toolspec.go` → gochecknoinits) mirroring the `metaengine register.go` precedent, with a comment tying them together.
   - Full cqrs-lint suite re-run after fixes: **19/19 packages ok**.
5. **Lint `catalog` → green** — exporter.go gocyclo(27) suppressed by extending the existing `//nolint:cyclop` directive to `gocyclo` (author's "straight-line pipeline" intent already declared); file kept at exactly the baselined **382 lines**. The other session's 8 docserver findings were **deferred to them (files edited as recently as 5 min prior)** and they fixed them within the hour — the de-confliction policy worked.
6. **Scoped `nix fmt --fail-on-change` → green** over every file I touched (the exporter wrap near-miss caught and corrected here).
7. **`#check-lint-config` → green** after my `.golangci.yml` exclusions.
8. **docserver suite green ×3** (18:52 state, 19:58 state, 20:26 state — tracking their churn), **eventcatalog tests green**, **queue/sqlite conformance green standalone** (4.2s vs 18.4s in-suite).
9. **Cascade run 1's failure root-caused and fixed**: ADR index drift (140 files vs 139 indexed — their new ADR-0142 landed 20:04 unindexed) → added the row to `docs/README.md`.
10. **Cascade run 4's docserver failure root-caused AND fixed** — the session's key find:
    - Their new `TestDocsServer_OpenAPISpecRequestScopedServers` byte-compared `json.Marshal(decoded any)` (map) against `json.Marshal(typed struct)`. **`encoding/json/v2` does not sort map keys** (v1 did), so the comparison flips on map-iteration luck — evidenced by 2/2 suite failures with byte-identical messages vs ~8/8 standalone passes, reproducible under no single toolchain/env combination (all combinations pass standalone).
    - Fix: decode BOTH sides and `reflect.DeepEqual` (order-insensitive). Verified **×5 (`-count=5`) green**.
11. **Footgun documented** — `docs/agents/gotchas-language-footguns.md`: json/v2 map-order entry with the fix pattern ("never compare raw bytes across a decode boundary").

## b) PARTIALLY DONE

1. **Verify cascade — 4 attempts, every failure dispositioned, zero green runs:**
   - Run 1 (21:0x): doc-assertions — ADR index drift → fixed (a/9).
   - Run 2 (stable tree): failed ONLY on the **filed** `TestSystem_ResetProjection_RestartAndReplay` starvation flake (46.27s in-suite / 0.18s standalone). One retry taken per policy.
   - Run 3: invalidated by mid-flight tree churn (their commits 20:19–20:23 landed during the run; docserver test executed mid-edit).
   - Run 4 (5-min-stable tree): docserver order-flaky test (→ fixed, a/10) + `TestConformance/Claims/expiry_reclaim` in queue/sqlite (timing-marginal: green in runs 2–3 and standalone; claim-expiry is wall-clock sensitive, 4× slower in-suite).
   - Run 5: **never started** — 3.5h of bounded waits (30+120+30 min) with the co-tenant session running gates continuously 16:11→23:22 (load 49–125 via govalid ×3, nix flake check, golangci-lint, LTO builds).
2. **Repo-wide lint** — my two modules green, but the cascade's lint phase was never reached (all runs died in the test phase), so the full-repo lint gate is unverified this session.

## c) NOT STARTED

1. `#verify-fast` **green** run (the standing blocker; everything upstream of it is green).
2. `#verify` (full, incl. soaks) — gated behind 1.
3. `#verify-ci` (per-module GOWORK=off matrix) — gated behind 1.
4. Morning re-assessment (06:47): still blocked — `vulnix` + `golangci-lint fmt --diff` running now, tree dirty with the overnight session's uncommitted `encryption/`+`deriver/` edits. The co-tenancy is now 15h continuous (16:11 → 06:47).

## d) TOTALLY FUCKED UP (honest)

1. **Retry sequencing**: after run 2's clean flake-only failure I re-ran immediately instead of first building the tree-stability watcher — run 3 burned ~20 min on a churning tree and produced two artifacts I then had to root-cause separately.
2. **Reproduction theater**: ~10 standalone variant runs chasing toolchain/env theories (including an arm64 store binary and GOEXPERIMENT=none) before reading the failing test's imports. The mechanism was visible in the source in 30 seconds once I looked.
3. **Naive quiet criterion**: my first watcher defined quiet as load<6 — it declared QUIET while the other session was mid-edit (commits 4 min later). Quiet must mean load AND tree-stable.
4. **Baseline-blind edit**: touched a baselined 382-line file without checking the file-size gate's constraints first; the formatter wrap would have grown it to 384 (gate failure) — caught only because I ran the scoped fmt gate immediately.
5. **Headline inflation**: last night's closing message said "Session complete" — accurate in its table, overstated in its title (primary deliverable unverified).

## e) WHAT WE SHOULD IMPROVE (systemic, from this session's evidence)

1. **Tree-stability gate as a script** — `scripts/wait-for-quiet.sh` (load < N AND no new commits AND no tracked-file edits for M consecutive minutes). This session hand-rolled it three times.
2. **json/v2 byte-comparison audit** — grep `_test.go` for `json.Marshal` comparisons across decode boundaries; the docserver test was one member of a class that v1's sorted-map-keys behavior used to hide.
3. **The system starvation flake is now the #1 gate blocker** — 3 strikes in one day (45.7/46.3/66.9s). Filed 2026-09-16 but unfixed; every composed gate on this repo is probabilistic until it is.
4. **queue/sqlite claim-expiry test needs a virtual clock or generous margin** — wall-clock-sensitive claim expiry under 4× contention slowdown.
5. **verify-fast test phase runs ALL modules in one `go test` at default parallelism** — this _amplifies_ timing-flake probability (same binary set, max contention). A `-p` cap or per-module sequencing would trade minutes for determinism.
6. **Env pinning inconsistency in flake apps** — `doc-check` and `integration-pg` export `GOEXPERIMENT=jsonv2` explicitly; `verify`/`verify-fast` do NOT (they inherit the caller's env). A bare `nix run .#verify-fast` from a clean shell runs stdlib json v1 with the `goexperiment.jsonv2` tag forced — exactly the split that made the docserver failure mysterious for an hour. Pin it in the app.
7. **Cascade aborts at first failed phase** — when tests flake, lint/arch/coverage/api-stability phases never run, losing their signal per attempt. Consider running cheap gates before the (probabilistic) test phase, or continuing past a failed phase with a summary.
8. **golangci cache mount is dead** — buildflow env-guard reported `GOLANGCI_LINT_CACHE /home/lars/projects/.golangci-disk` missing and rewrote to the default; caches run cold (fleet infra).
9. **Clock-skew anomaly noticed, unexplained**: `catalog/docserver/render.go` had mtime 19:44 when wall time was 18:58 (46 min in the future) during the other session's templ regeneration. One look with `stat` next time it appears; harmless so far but weird.
10. **Durable run logs** — verify logs went to /tmp and evaporated; evidence should live somewhere the next session can cite (e.g. `docs/status/` companions or `/home/lars/projects/.gotmp`).

## f) Things to get done next (prioritized; this session's observations only)

1. **Run the cascade when stable-quiet**: `#verify-fast` (one retry iff ONLY the system flake) → `#verify` → `#verify-ci`. Everything upstream is green; this is the only unverified delta.
2. **Fix `TestSystem_ResetProjection_RestartAndReplay` starvation** (filed 2026-09-16; 3 strikes 09-18) — or temporarily split it out of the parallel suite (`-short` skip + dedicated sequential run) until fixed.
3. **Make the queue/sqlite claim-expiry test clock-robust** (virtual clock or margin ≥ observed 4× contention slowdown).
4. **json/v2 byte-comparison audit** across all `_test.go` (grep `string(got) != string(base)`-shaped patterns); fix any further decode-boundary byte comparisons.
5. **Pin `GOEXPERIMENT=jsonv2` in the `verify`/`verify-fast`/`verify-ci` flake apps** (one line each; consistency with `doc-check`) — e/6.
6. **`scripts/wait-for-quiet.sh`** with `--stable-minutes` (+ optional `--modules` scope) and a self-test per the gate-script convention; wire into the cascade resume runbook.
7. **Cap test-phase parallelism** in verify apps (`-p` or per-module loop) — determinism over raw speed for the canonical gate.
8. **Restore the golangci-lint cache mount** (`.golangci-disk`) — buildflow env-guard says it's dead.
9. **Investigate the render.go future-stamped mtime** (clock skew during templ regen) — one `stat` + correlation check next occurrence.
10. **HARVEST check**: items 2–7 above are TODO_LIST candidates (the 13 ADR-0141 items were already harvested by the predecessor — do not duplicate).
11. _(observed, not mine to fix)_ The overnight session's uncommitted `encryption/` + `deriver/` working tree (00:58 mtimes) needs an owner-state confirmation before anyone runs tree-wide gates.
12. _(observed)_ Overnight daemon commits 00:54–00:58 swept 526 files (442+64+20; net −421 lines, mostly single-line removals) — unreviewed by me; worth a skim by its owning session.

## g) Questions I cannot answer myself

1. **Cascade policy on this machine (asked 18:11 §g/2, now with 5 more data points):** the co-tenant sessions have now run gates continuously for 15h (16:11 → 06:47). Strict exclusivity means the cascade waits indefinitely. Do you want (a) strict exclusivity as-is, (b) a scheduled quiet window (e.g. run the cascade nightly at a fixed hour), or (c) fix-first (items f/2–f/4) so the gate stops being probabilistic, then relax?
2. **Env pinning policy (e/6):** should the verify-family flake apps pin `GOEXPERIMENT=jsonv2` explicitly (making a clean-shell `nix run .#verify` behave identically to the documented env chain), or is inheriting the caller's env deliberate (CI may already set it — I did not audit ci.yml, per your no-unrelated-research instruction)?
3. **Who owns the current working tree?** `encryption/*.go` + `deriver/deriver.go` are modified-uncommitted right now (00:58 mtimes) and another session ran `golangci-lint fmt`/`vulnix` at 06:47. Is that session still active (→ I keep deferring tree-wide gates), or did it finish (→ the tree needs its state confirmed/absorbed before the cascade)?

---

_Point-in-time report written 2026-09-19 06:47. Claims trace to command runs captured in-session (verify-fast logs cited in the 23:30 handoff were lost with /tmp overnight; exit codes and messages were recorded contemporaneously). Working tree at report time: dirty with non-session changes; HEAD `36d5fee80` (00:58)._
