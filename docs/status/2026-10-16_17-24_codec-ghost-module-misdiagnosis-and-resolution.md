# Status Report: codec Ghost-Module Investigation — Misdiagnosis, Correction, Resolution

- **Date:** 2026-10-16 17:24 CEST
- **Repo:** go-cqrs-lite @ `master` (auto-commit daemons active everywhere; all session writes committed by daemons)
- **Scope:** the continuation session after the 03:55 report — owner directive "go-cqrs-lite/codec is used nowhere now. remove it", the investigation it triggered, and the correction of my own earlier claims. This report is session-scoped; the 03:55 report remains the master migration plan.
- **Format note:** `.md` at the owner-named path per explicit instruction (skill default is HTML; standing dispatch exception applies).

---

## TL;DR

The "remove codec" instruction exposed that **my previous session's most dramatic finding was wrong in mechanism and stale in fact**. The 11 "direct consumers" of the deleted `go-cqrs-lite/codec` module were zombie go.mod REQUIRES with **zero code imports** — I had mislabeled the tool's semantics as "import-level" and encoded that error into three durable artifacts. Worse, ~3 hours of scheduled fleet automation landed mid-session: by the time the owner said "used nowhere now", a tidy sweep (commits 05:07–06:48) had already removed every codec require and promoted `go-codec v0.3.1` to direct. **There was nothing to remove; I verified the emptiness to death, corrected all tainted docs, and closed the TODO row.** Final state: 0 codec lines in `who-uses` (53→50 matched keys), zero `.go` imports, zero requires fleet-wide; the only surviving reference is an intentionally-poisoned tag-release test fixture.

---

## Timeline (what actually ran)

1. **03:55** — first status report written (fleet census, critical path, delivery-mark fork).
2. **03:59** — `project-dependency-graph who-uses` full-tree run captured (53 keys; codec shown with 11 direct consumers @ v4.4.0).
3. **~04:03–04:08** — census corrections persisted; Addendum 1 written claiming tool = "import-level" (WRONG — the tool's `DirectOnly` help text, visible in my own grep output, already said require semantics).
4. **~06:4x** — owner instruction: "codec is used nowhere now. remove it."
5. **Investigation arc:** in-repo residue check (zero: no codec dir, no go.work/flake/golden/gate references) → go-codec package compat check (package `codec`, seamless) → enumeration of import sites across the 11 repos (**zero matches — premise collapse**) → deep searches (`-uu`, go.work.sum only) → tool-source reading (`extractDeps`, `ConsumersFromModules`, `buildWorkspaceModule`) → temporary debug program in the tool repo reusing its discovery (**zero codec deps across 208 modules**) → state-change discovery via git logs (go.mod-touching daemon commits 05:07–06:48 in all listed repos) → hard evidence: github-local-sync 05:38 commit diff (`- go-cqrs-lite/codec/v4 v4.4.0`, `+ go-codec v0.3.1` direct).
6. **Resolution:** fleet-wide final sweep (zero imports, zero requires; only `real-setup-v4.8.1-poisoned` fixture remains — intentional), debug program trashed, TODO_LIST blockquote semantics fixed + ghost row marked `[x]`, report Addendum 2 recorded the misdiagnosis, final `who-uses` re-run confirmed codec absent.

---

## Answers to the three opening questions

**What did I forget?**
- The `DirectOnly` flag's help text — I literally read it ("indirect-only requires are not usage") during the first session's tool audit and still wrote "import-level" into TODO_LIST and the report. Disconfirming evidence in hand, unprocessed.
- The user's own claim was better information than my tool output. I built a migration plan on top of "11 direct consumers" before testing the owner's contradicting assertion with one `rg`.
- Mid-session time gaps: ~3 hours of daemon/automation activity landed between my 03:59 capture and my 04:05 checks. I described "11 zombie requires" as current in Addendum 1 **while my own 04:05 sweep in the same document's timeframe showed the repos already clean** — an internal inconsistency I never cross-checked.

**What could I have done better?**
- Premise-check first: one `rg -uu "go-cqrs-lite/codec" ~/projects --glob '*.go'` before planning would have collapsed the ghost-module story in seconds. Instead I prepared import-rewrite mechanics (package-name compat, go.work handling, tidy risks) for a migration that didn't exist.
- `git log -S` archaeology should have been step 2, not step ~8 — repo state contradicting a 3-hour-old observation means "what changed?", and git answers that instantly.
- I chased the ci-siblings mirror-resync theory (255-file auto-commit) that never held — the removal was in the REAL repos' working trees, committed by their own daemons.
- Read a dependency's struct/API before writing against it: my debug program burned two compile round-trips on `m.Name`/`m.Path` (actual fields: `ModuleName`/`Dir`) and an import-casing error (`LarsArtmann` vs `larsartmann`).

**What could I still improve?**
- Stand up the re-runnable census (script + JSON) so "current fleet state" is a command, not a memory — this session's confusion is exactly the failure mode it prevents.
- Adopt a hard rule: any tool-output number persisted to docs gets its semantics verified from the tool's source/help in the same session, with the evidence line quoted.
- When my analysis and the owner's assertion conflict, test theirs first — they were right, and cheaper to check.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| A1 | In-repo codec residue audit — zero (no dir, go.work/flake/golden/gates/module-map clean; matches are concepts or external go-codec) | `rg` fleet + repo sweeps |
| A2 | Premise collapse: zero `.go` imports of go-cqrs-lite/codec across the entire fleet | `rg -uu` sweeps (04:05 and 06:50) |
| A3 | Tool semantics ground truth: edges = go.mod requires (direct vs `// indirect` annotation); `extractDeps` is 1:1; lightweight discovery parses go.work internals, excludes `../` siblings | tool source: `extractDeps` (discover.go:218), `ConsumersFromModules` (consumers.go:100), `workspace.go:131-190`, config `DirectOnly` help |
| A4 | State-change proof: the sweep — daemon commits touching go.mod in all listed repos 05:07–06:48; github-local-sync 05:38 diff removes codec/v4 + promotes go-codec v0.3.1 | `git log -1 --format -- go.mod` per repo, `git log -S` |
| A5 | Final verification: `who-uses` full tree = 0 codec lines, 50 matched keys (was 53); retry/flightrecorder phantom keys also gone | `/tmp/final-tree.txt` (06:5x) |
| A6 | Poisoned fixture correctly preserved: `real-setup-v4.8.1-poisoned/go.mod` carries the stale `// indirect` codec pin BY DESIGN (tag-release test fixture) — flagged do-not-clean | go.mod line 64 |
| A7 | TODO_LIST corrected: blockquote semantics fixed (require-level, stack 5-direct numbers restored after an edit briefly dropped them); ghost-module row marked `[x]` with honest resolution + remaining S-tail | TODO_LIST §v5 Unification |
| A8 | Report Addendum 2 appended (annotate mode — Addendum 1's wrong text left in place, corrected in place-by-append) | status report 03:55 file |
| A9 | Debug program (cmd/debugcodec in project-dependency-graph) added for ground truth, then trashed; daemon had absorbed it, deletion now pending daemon | tool repo git status |
| A10 | Deliberate non-action: 11-repo import rewrite + per-repo tidy/build/test plan retired the moment enumeration came back empty — no repo touched | this report; no working-tree changes outside go-cqrs-lite docs |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
|---|------|------|---------|
| B1 | Ghost-module tail (TODO row) | codec: RESOLVED + verified | retry + flightrecorder consumers still ride local `replace`s (should re-pin to go-retry/go-flightrecorder external repos); crm-exec-stage/go.mod still has unresolved merge-conflict markers (unparseable by tooling) |
| B2 | Tool-semantics documentation | corrected in TODO blockquote + Addendum 2 | the first report's Addendum 1 body still contains the wrong "import-level" claim by design (annotate-not-rewrite) — a reader must reach Addendum 2 to get the truth |
| B3 | Master migration plan (03:55 report) | Tier-0 "zero importers" verdicts unchanged in substance | numbers are require-semantics now; a fresh census re-run is advisable before the codemod phase (fleet state moved twice today) |
| B4 | project-dependency-graph repo hygiene | my debug file removed | `vendorHash.nix` shows modified in that repo — NOT my change, left untouched, needs owner eyes |

## c) NOT STARTED (unchanged from the master plan)

The 23 core/v5 copy-forwards (M23's ~20K-line eventtest closure first) · `cqrs-upgrade --to-v5` import-rewrite codemod · any fleet repo v5 migration · free deletes (transport/http+grpc, storage/view+relational, 7 dead stack presets) · ghost-system verdicts for deriver/graph · census-as-artifact script · per-repo v5-readiness batch.

## d) TOTALLY FUCKED UP

1. **The misdiagnosis chain, persisted.** I encoded "import-level" + "11 ghost importers surviving on a proxy-cached dead tag" into three durable artifacts (TODO blockquote, TODO row, report Addendum 1) — including the fabricated-sounding mechanism "builds survive only because the module proxy still serves the dead tag" (stale unimported requires don't affect builds at all). The drama was my construction; the truth was a routine untidy go.mod.
2. **Internal inconsistency shipped:** Addendum 1's own timeframe contained my zero-imports sweep, and I never reconciled the two before writing "11 direct importers" one section above it.
3. **Ignored my own verify-before-encode rule** set two reports ago (the go-appkit census error) — same failure class, second occurrence: agent/tool output taken as ground truth without reading the tool's contract. Twice is a pattern, not bad luck.
4. **Wasted planning cycle:** a 7-step migration todo list (rewrites, per-repo tidy/build/test, verification) built entirely on an unverified premise — luckily retired before execution, but the plan-first-verify-later order was backwards.
5. **Minor:** two wasted compile round-trips in the debug program; the ci-siblings theory chased despite zero supporting consumer labels; `/tmp` baseline lost to cleanup (lost the diffable 03:59 capture — re-derived state from git instead).

## e) WHAT WE SHOULD IMPROVE

1. **Tool-contract gate:** before persisting any tool-derived number, quote the tool's own semantic definition (help text/source line) next to the number. The `DirectOnly` help already documented the truth — the gate would have cost one glance.
2. **Owner-claim priority:** when the owner's assertion contradicts session analysis, run the one-command test on THEIR claim first. It was right this time and the last time.
3. **Time-gap discipline:** after any long gap, re-run a 1-command state probe (git status + fresh census) before continuing a state-dependent thread; annotate report claims with capture timestamps (Addendum 2 now does this).
4. **Census-as-artifact:** make the fleet census a script + JSON output (per-repo require/import counts, timestamps) so drift is diffable — today's confusion is the ROI case.
5. **Plan-after-premise:** todo lists that depend on an external claim get their verification task as item #1, blocking.
6. **Upstream tooling (consider):** project-dependency-graph could offer an optional import-level mode (packages.Load-based) — require-level is a fine default for "pins", but "usage" and "pins" are different questions and today they got conflated (by me, not the tool). Filing anything upstream goes through verify-before-filing first.

## f) Up to 50 things to get done next

> Carry-over note: the 03:55 report's list remains the master brainstorm; this list re-states the still-open items with today's deltas folded in. New items first.

| # | Item | Impact |
|---|------|--------|
| 1 | Re-pin retry/flightrecorder consumers to the external go-retry/go-flightrecorder repos; let the in-repo paths die | ★☆☆ |
| 2 | Fix crm-exec-stage/go.mod merge-conflict markers (unparseable — invisible to all tooling) | ★★☆ |
| 3 | Fresh `who-uses` census re-run + saved JSON artifact (fleet moved twice today; codemod phase needs current numbers) | ★★★ |
| 4 | Identify/document the automation that produced the 05:07–06:48 tidy wave (owner knowledge — see question 1) | ★★☆ |
| 5 | Owner review: `vendorHash.nix` modification in project-dependency-graph (not mine) | ★☆☆ |
| 6 | Re-verify Tier-0 free deletes under require-semantics, then delete transport/http+grpc, storage/view+relational, 7 dead stack presets (sequencing per question 3) | ★★☆ |
| 7 | Fork storage core (minus deprecated surfaces) → core/v5/storage | ★★★ |
| 8 | Fork storage/memory → core/v5/storage/memory | ★★★ |
| 9 | Fork snapshot → core/v5/snapshot | ★★★ |
| 10 | Fork eventtest → core/v5/event/eventtest (closes M23, lag→0) | ★★★ |
| 11 | Write the copy-forward playbook once (exclusions, pair registration, golden regen, allowlist prune, file-size/dup-gate treatment) | ★★★ |
| 12 | Fork decider | ★★★ |
| 13 | Fork middleware | ★★★ |
| 14 | Fork system (composition root — biggest consumer blocker) | ★★★ |
| 15 | Fork metaengine core + keycodec | ★★★ |
| 16 | Fork projectionadapter | ★★☆ |
| 17 | Fork projection + projectionhost | ★★☆ |
| 18 | Fork commandlifecycle + projections | ★★☆ |
| 19 | Fork scheduling + sqlstore | ★★☆ |
| 20 | Fork schema (minus ADR-0126 shells) | ★★☆ |
| 21 | Fork listing (ADR-0114 type-driven tombstone decision) | ★★☆ |
| 22 | Fork claiming | ★☆☆ |
| 23 | Fork otel + prometheus | ★★☆ |
| 24 | Fork catalog | ★☆☆ |
| 25 | Fork encryption + signing (minus Rejecting*/ErrInnerStoreNot*) | ★★☆ |
| 26 | Ghost-system verdicts: deriver, graph — enter core/v5 or die | ★★☆ |
| 27 | Decide whether stack enters core/v5 at all or is struck from the ADR-0152 list | ★☆☆ |
| 28 | Fork systemscenario (absorbs scenario/v4) | ★★☆ |
| 29 | Register every new mirror pair in the lockstep gate as it lands | ★★★ |
| 30 | Signature-level lockstep once core/v5 stabilizes | ★☆☆ |
| 31 | Build `cqrs-upgrade --to-v5` (import rewrite, require collapse, go.work/replace rewrite) | ★★★ |
| 32 | Codemod rename table (On→OnRecord, OnTyped→OnRecordTyped, tombstone metadata → event types, StreamRef pair forms) | ★★☆ |
| 33 | Codemod hard-refusal for modules with no core/v5 twin | ★★★ |
| 34 | Codemod golden fixtures per migration class | ★★☆ |
| 35 | Per-repo v5-readiness batch (`cqrs-upgrade --strict --json` across the fleet) | ★★☆ |
| 36 | V007 table maintenance as core/v5 packages land | ★☆☆ |
| 37 | cqrs-upgrade gains core/v5 pin awareness | ★★☆ |
| 38 | Migrate cqrs-htmx (stack/v4 in usermgmt + import rewrite; 73% indirect-edge shield) | ★★★ |
| 39 | Migrate go-appkit cqrs/+docs/+integration (24 trains, blocked on #8–#12) | ★★★ |
| 40 | Migrate browser-history (stack + idempotency/sqlstore → metaengine.DedupStore) | ★★☆ |
| 41 | Migrate crm + crm-exec-stage | ★★☆ |
| 42 | Migrate InboxClean | ★★☆ |
| 43 | Migrate bank-sync, github-local-sync, SEC, KeyCountdown, SwettySwipperWeb services | ★★☆ |
| 44 | Migrate residue batch (~25 repos) via codemod with per-repo test gates | ★★☆ |
| 45 | Post-migration hygiene per repo (go.work/prune, verify chains) | ★☆☆ |
| 46 | Delete idempotency/kvstore+sqlstore after the 4 repos migrate | ★★☆ |
| 47 | Delete stack + presets after the 13 repos migrate | ★★★ |
| 48 | Delete deprecated tombstone metadata APIs + ADR-0126 shells | ★☆☆ |
| 49 | Gate-pruning cascade + cut v5.0.0 (per-module tests over the changed set — never verify=ok-only) | ★★★ |
| 50 | Housekeeping: taskmanager gopls errors (real vs workspace noise); possibly-stale .go-arch-lint.yml TODO row; the ~34-module go.sum tidy sweep (owner triage still pending) | ★☆☆ |

## g) Questions I cannot figure out myself

1. **What automation ran the 05:07–06:48 fleet tidy wave** (renovate? a nightly script? which schedule/machines)? This decides whether the v5 codemod can rely on daemons keeping go.mods tidy behind it, or must batch-edit+commit atomically before the next wave eats in-flight work. I could hunt crons/systemd across ~40 repos, but ownership and schedule are yours to state.
2. **Do you want project-dependency-graph to grow an import-level (code-usage) mode** — or is require-level the intended semantics for "consumers" (pins), with usage left to cqrs-lint/V007? It's your tool; if you want the mode, say the word and I'll go through verify-before-filing + a design first.
3. **Sequencing ruling:** execute the Tier-0 free deletes (transport/http+grpc, storage/view+relational, 7 require-dead stack presets — all at zero requires today) BEFORE core/v5 completion for an early dedup payoff, or strictly at the v5.0.0 cut per ADR-0152's dual-support letter? Early deletion changes the transition story; owner call.

---

*Point-in-time snapshot; state captured ~06:55, report written 17:24 — fleet automation may have moved pins again since. Verify-before-acting on any number here.*
