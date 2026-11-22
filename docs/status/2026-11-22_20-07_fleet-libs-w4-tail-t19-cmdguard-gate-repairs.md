# Fleet-Libs Plan W4 Tail — T19 cmdguard Port, Gate Repairs, Closeouts

> Status report for 2026-11-22 evening session (continuation of the
> fleet-libs Pareto plan execution, `docs/planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md`).
> Prior state: W1–W3 committed (`1e6ad6fd9`, `e5276cfc4`, `665153fed`); W4 bulk
> committed by the prior session (`34fef2212`); this session took T19 + the T25
> closeouts + the wave-boundary duties. Trigger: owner "GET SHIT DONE" directive,
> then the status-report interrupt.

## What this session did (chronological, with evidence)

1. **Reconstructed state.** The W4 bulk (T16, T17+T18, T20, T21, T23, status-README
   row) was already committed as `34fef2212` by the prior session; T19, the W4
   execution-log section, and T25 were genuinely open.
2. **T19 — cqrs-upgrade moved onto the cmdguard chassis** (commit `c5a58e925`):
   - `cmd/cqrs-upgrade/v4/main.go` rewritten: `AppConfig` typed flag struct (7
     flags, help strings byte-identical to the old stdlib set), `buildCLI()` +
     `cli.RootCommand().RunE`, `ExecuteAndExit` entry; old `parseFlags` split into
     `configFromFlags` (validation) + `runPipeline` (pipeline).
   - `AppConfig` deliberately does NOT embed `cmdguard.Config` — the batteries'
     config carries its own `--strict` flag, which would collide with this tool's
     v5-readiness gate (verified in cmdguard source, config.go:16).
   - Parse contract preserved: `cobra.ArbitraryArgs` for the positional dir;
     `SetInterspersed(false)` keeps the 2026-09-11 "flags must precede the dir"
     hardening true under pflag (pflag's default would silently APPLY a
     flag-after-dir instead of ignoring it — a subtler dialect of the original bug).
   - Tests ported to the production seam (`buildCLI` + `ExecuteWithArgs`):
     flags_test.go, main_test.go, e2e_test.go, rewrite_v5_test.go, suggest_test.go
     (the last two found via vet, not upfront — see "forgot"). New
     `cli_contract_test.go` pins help/defaults parity per flag.
   - Sentinels moved to `errors.go` (repo convention; main.go was at the 350-line
     ceiling). DEP_BUDGET 3→5 (cmdguard + cobra) with inline rationale.
   - Gates: full suite green (3.8s), scoped lint clean, `#check-arch` green,
     kill-mid-write + codemod goldens unchanged (they ride the suite).
   - Golines lesson: gofmt alone is NOT this repo's canonical formatting — the
     golangci formatter pipeline (gofumpt+golines, max-len 120) also aligns struct
     tag internals; lint-scope caught it, golines canonical form applied.
3. **W4 execution-log section written** (`docs/status/2026-11-22_fleet-libs-plan-execution-log.md`):
   T15 refusal (SlugifyID semantics + renderer incompatibility → golden-unsafe,
   reconsider-when recorded), m55 refusal (event/v4 PublishRetry = v5-train work),
   T17/T18 rowed remainders with VERIFIED line citations (re-grepped all three
   `.Family() !=` sites rather than trusting the summary), T19/T20–T24 verdicts,
   T25 dispositions.
4. **T25 closeouts**:
   - m73: `level=warn` stray dispositioned without owner round-trip — origin
     identified (nats-server stdout captured by ephemeral-nats.sh 2026-09-29,
     daemon commit `3756eb4d1`, later dropped from tracking); file absent from
     disk AND index today. Nothing to trash.
   - m74: `docs/error-taxonomy.md` header v0.10.0 → v0.12.0 (matches every
     module pin — verified across event/command/storage/middleware/system).
   - m75: NOT done — needs owner Q3 answer; no policy invented.
5. **CHANGELOG rows for W2–W4** (commit `7598349c2`): SSE 400-on-malformed,
   retry-on-go-retry, docserver ETag/500, `#check-libreports`, `benchkit.FactoryFromSystem`.
   Two self-inflicted rounds: the changelog-symbols gate rejected three
   foreign-library dotted citations (etag.New, retry.Do, sse.ParseEventID) —
   reworded to prose; see near-misses for the destructive-edit round.
6. **m72 — verify-fast first run: RED, repaired at root cause** (commit `7598349c2`):
   - `cmd/cqrs-bench/v4/factory.go` 398→429 (T10's system-memory backend grew it
     past its baseline): split into `factory_system.go` (system+memory backends,
     35 lines) + `parse_helpers.go` (parseDurability/parseCodec/parsePayloadSizes/
     looksLikeFlag, 110 lines); factory.go now 300. Suite green (104s).
   - `cmd/doc-check/v4/recipes_catalog_meta3.go` 408→420 (§2.47 fence #5): the
     whole §2.47 family + `queryEndpointHelperPreamble` moved to meta4
     (meta3 284 / meta4 193); full recipes compile harness green (32s).
   - Both baseline rows dropped per the shrink-only rule (61→59);
     `#check-file-size` green.
   - The committed execution log briefly claimed "m72 verify-fast green" — false
     when written; corrected to the true sequence in the same commit that carries
     the repairs. Full re-run to green still owed (see next things #1).

## What is verified green right now

- cqrs-upgrade full suite (incl. e2e wire, kill-mid-write, goldens), scoped lint,
  `#check-arch` (budget 5 accepted).
- cqrs-bench full suite; doc-check full recipes harness.
- `#check-file-size` (post-split), `#check-md-go`, `check-changelog-symbols.sh`
  (39 honest citations).
- W1–W4 history: `1e6ad6fd9` → `e5276cfc4` → `665153fed` → `34fef2212` →
  `c5a58e925` → `7598349c2` (this session: the last two).

## What is in progress / partially complete

- **m72 completion**: the verify-fast RE-RUN after the file-size repairs has NOT
  happened (the first run is what caught them). Until it runs green, the wave
  cannot claim the full gate set.
- **Uncommitted foreign dirty set** appeared after my last commit: `TODO_LIST.md`,
  `flake.nix`, `docs/planning/2026-11-21_19-50_SUPERB-lint-campaign-closeout….md` —
  NOT touched (another session in flight; per house rules I do not revert or
  edit foreign changes).

## What I forgot (honest)

- Upfront seam inventory: I ported flags_test/main_test/e2e_test first and
  discovered rewrite_v5_test.go + suggest_test.go still called `parseFlags`/`run`
  only via `go vet` failures. One `rg "parseFlags\(|run\(ctx" *_test.go` before
  starting would have saved two build-fix cycles.
- The cli_contract_test first draft used a nonsense `cobra.EnablePrefixMatching`
  guard and wrong pflag APIs (`NFlags`, then `Len`) — three iterations that a
  30-second API check would have collapsed into one.
- I wrote an invalid struct tag (string concatenation inside a backtick tag) and
  a doubled-quote typo in the Workspace tag — both caught on build/repair, both
  avoidable by reading my own diff before saving.
- `gofmt -w` was applied where golines canonical form was required (formatter
  pipeline divergence) — lint-scope after structural edits should be reflexive
  here, not a discovery.
- W2's taskmanager hardening got no CHANGELOG row (judged example-only; arguably
  borderline since the README now documents the codegen loop).

## What was almost fucked up (caught and fixed)

- **Destructive CHANGELOG edit**: my Changed-section insert consumed the
  queue/mysql deadlock-ladder row (old_string overlapped its text). Caught by
  grepping the diff for removals, restored verbatim in the next edit; net diff
  on that row is zero.
- **False gate claim in the execution log**: "m72 verify-fast run green" was
  committed while the run had actually FAILED (that failure is what found the
  file-size violations). Corrected same-session before the report; the corrected
  text names the true sequence and the owed re-run.

## Top next things (ranked, up to 50 — honest, not padded)

1. Re-run `nix run .#verify-fast` to green (m72 closure; nothing else heavy may
   run concurrently — #verify exclusivity).
2. Then `nix run .#check-test-runtime` + `#check-untested-exports` style sweep
   gates over the W4 dirty modules if verify-fast does not already cover them.
3. Owner Q1: ratify the T15 refusal (vs. deliberate D2 golden re-baseline).
4. Owner Q2: ratify the T19 flat-flags shape (vs. subcommand restructure).
5. Owner Q3: remote-tag policy (fetch vs stay local) → AGENTS.md (m75).
6. Pin-wave: per-module tests over the m06 changed-module list, then
   `batch-release.sh` (remember: `verify=ok` builds but does NOT test — the
   90-tag red-suite lesson).
7. `git status` re-check before touching shared docs (foreign session active).
8. Consider `WithCLIVersion` for cqrs-upgrade (other cmd tools carry a version
   command; this one currently has none — deliberate surface-freeze or oversight,
   owner call).
9. Human-diff the actual `cqrs-upgrade --help` output vs. the old stdlib render
   (flag-level parity is pinned; layout will differ — confirm acceptable).
10. Extend TestCLIHelpParity to pin the `Use` line + Args validator (currently
    only flags/defaults are pinned).
11. Document the gofmt-vs-golines formatter divergence in
    `docs/agents/gotchas-tooling-build.md` (gofmt-clean ≠ lint-clean here).
12. Sweep the recipe catalog doc comments for range staleness (meta4 claimed
    §2.48–§2.50 while holding §2.51; fixed in passing, siblings unchecked).
13. FEATURES.md: update the cqrs-upgrade row (cmdguard chassis, battery commands).
14. AGENTS.md module-map/quick-ref: record that all six cmd tools are now on
    cmdguard v5.1.0 (W1 pinned five; T19 made it six).
15. CHANGELOG: decide on a W2 row for the taskmanager hardening (413/readiness/
    drain order) if the example is considered user-visible surface.
16. The 18-47 status report says "T19 in flight" — superseded by this report and
    the log; leave as write-once history (no retro-edit).
17. `#check-duplication` over the new files (factory split, errors.go move) —
    the bootstrap-guard art-dupl accept comment is present, but the gate has not
    run since the split.
18. `#check-package-surface` over cqrs-upgrade (new package files added).
19. cqrs-bench: `#test-runtime` budget rows may shift after the split (parse_helpers
    is new but tiny; factory_test unchanged).
20. Re-run `#check-libreports` after this session's line shifts (citation-sweep
    covers docs/review/libs, not status docs — should be unaffected, cheap to confirm).
21. Sweep the execution log's m06 tag-prep list: T19 changed cqrs-upgrade go.mod
    (cmdguard+cobra) — the wave's module list for batch-release must include it.
22. Ask owner whether the daemon-absorbed foreign dirty set (TODO_LIST/flake/
    planning doc) is theirs to land.
23. Consider promoting `TestCLIParse_FlagsBeforeDirReachConfig`'s errArgsAfterDir
    trick into a cmdguard testutil helper if more tools adopt it.
24. cqrs-gen README: the absolute-path quirk documented in the log deserves a
    line in `cmd/cqrs-gen/v4/README.md` (currently only in the execution log).
25. `scripts/check-module-layers.sh`: the DEP_BUDGET comment for cqrs-upgrade
    now duplicates cqrs-gen's chassis rationale — consider a shared comment
    convention row for "cmdguard chassis" tools.
26. Monitor: `go.sum` for cqrs-upgrade gained cobra's indirect graph —
    `TestEveryModuleGoSumIsTidy` should stay green (tidy ran; verify-fast confirms).
27. If the pin-wave is green-lit: tag `cmd/cqrs-upgrade` per the tag-wave ritual
    (annotated `-a -m`, four-mechanics checklist in gotchas-module-management).
28. Nightly: `#check-libreports --self-test` rides CI? (Currently flake-app only;
    decide whether it joins a nightly job.)
29. The benchkit FactoryFromSystem row mentions "presets marked legacy" —
    confirm the README/backend-list prose in cmd/cqrs-bench matches (docs drift).
30. Reflect T19's "no Config embed" decision in the cmdguard SKILL.md as a
    collision gotcha (`--strict` name clash pattern) — it bit the design phase.

(31–50: reserved for what the verify-fast re-run surfaces; entering filler would
violate the honesty bar of this report.)

## Owner questions outstanding

1. **T15 verdict**: ratify the refusal, or fund a deliberate D2 golden re-baseline?
2. **T19 shape**: flat-flags compat (shipped) vs. subcommand restructure?
3. **Q3 / m75**: remote-tag policy — fetch origin tags or stay local — recorded
   into AGENTS.md once decided; also gates the pin-wave.

— Session paused here per the interrupt directive. Working tree carries only the
foreign dirty set; my work through `7598349c2` is committed.
