> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** Superseded: the verify arc ended GREEN (S03 — 16-39 report); T36e was validated + mutation-tested + nightly-wired (13-21 §a); the taskmanager `--help` fix shipped end-to-end as v0.2.1 (16-39 §a5); the ADR-0123 README banner sweep eliminated the citation debt 37→0 (13-21 §a3). The 10:31 repair ratification rides the W3 owner bundle (§3).

# Status Report — Verify-5 In Flight; Concurrent-Corruption Repaired; README Gates Landed (2026-09-20 10:56)

> Continuation of the publish-and-prove execution (compaction handoff at ~10:30).
> Snapshot at 10:56:54 — **verify attempt 5 is LIVE in the Race phase (0 FAIL lines)**,
> Test phase fully green. This report pauses work per owner instruction.

## 1) Headline: verify attempt 4 was killed by a concurrent edit, not lint debt

- Attempt 4 (supervisor-launched 09:51) exited `VERIFY4-EXIT=1` at the lint phase at 10:32.
- **Forensics**: the first linted modules (event, command, commandlifecycle) were GREEN.
  Every module after line 888 of `/tmp/verify-attempt4.log` failed with
  `go.work lists go 1.27; module ../.. requires go >= 1.27.1`. File mtimes show an
  unknown concurrent editor touched `go.work` + 94 module `go.mod`s at **10:31:40**
  and `.golangci.yml` at **10:31:53** — mid-verify. The daemon absorbed it as
  `4a2a71ba1` (162 files) at 10:32:41.
- The edit: downgraded `go 1.27.1` → `go 1.27` in go.work + all module go.mods
  (**but not root go.mod** — leaving an inconsistent workspace that fails resolution
  entirely), set golangci `run.go: 1.26.7`, re-added the removed
  `goexperiment.jsonv2` build tag, and **deleted the depguard allow-list**. Markdown
  table reformats rode along (formatter signature — those were KEPT).
- Perpetrator: unidentified (7 parallel crush sessions run on this host across other
  repos; the crush-daily scheduler is innocent — its windows are 00:30–04:45).
  Possibility remains that a fleet session reached into this repo reacting to the
  live lint errors it saw in the streaming log.

## 2) Repair (committed, pushed)

- Reverse-applied exactly the go.work/go.mod/.golangci.yml hunks of `4a2a71ba1`
  (kept the markdown reformats): all 95 go.mod files + go.work back on
  **go 1.27.1**, `.golangci.yml` restored (`run.go: 1.27.1`, no jsonv2 tag,
  depguard allow-list back). Daemon absorbed as `9212c8408`; **pushed**.
- Gates re-validated: `nix run .#check-lint-config` ✓ (incl. exhaustruct canary),
  `GOTOOLCHAIN=auto go build ./...` ✓, `check-changelog-symbols.sh` ✓ (6 citations).
- Latent issue NOT yet fixed (documented here for the owner): flake.nix:95 assumes
  `pkgs.go_1_27` satisfies the 1.27.1 floor, but this host's nixpkgs resolves it to
  **go1.27.0**. Everything works because verify/agents run with `GOTOOLCHAIN=auto`
  (runtime toolchain fetch). A nixpkgs bump or explicit pin closes the gap.
- Pathspec lesson: `git diff -- '*.go.mod'` does NOT match nested `dir/go.mod`;
  `':(glob)**/go.mod'` does. First reverse-apply silently missed 94 files — caught
  by counting `go 1.27` occurrences before/after.

## 3) Verify attempt 5 — LIVE

- Supervisor re-armed at 10:44 with **`GOTOOLCHAIN=auto` exported** (attempt 4's env
  was fine for build/test/race but this removes the entire class).
- Quiet-gate refused once (load rebound 9.66 → >10 race between wait-for-quiet and
  the gate's own check), requeued, **passed 10:52**.
- Progress at 10:56: verify-docs ✓ (license/ADR/error-family checks), Module
  Coverage ✓, Build ✓, Vet ✓, **Test ✓, Race in progress, 0 FAIL lines**.
- Known supervisor cosmetic bug: the rc echo after `VERIFY5-EXIT=$?` consumes `$?`
  (supervisor log will say rc=0 always) — the authoritative marker is
  `VERIFY5-EXIT=` in `/tmp/verify-attempt5.log`. Fix if re-arming.
- Expected remaining: Race (~15-30m under load) → Lint (~10m, the phase attempt 4
  died in — now on a consistent tree) → arch/modsums/lint-config/css/duplication/
  turso/templ/bench/coverage/api-stability/error-taxonomy/doc-check (~10-15m).

## 4) Done this session (besides the repair)

- **taskmanager `--help` bug (real)**: the demo binary ignored args and BOOTED the
  server — the two "hung" processes from the 08:41 smoke were orphaned servers
  fighting over :8080 (killed). Fixed: `-help`/`-addr`/`-db` flags, honest usage
  text (routes verified against http.go), `Run(cfg Config)` signature.
  Module tests green; CHANGELOG Fixed entry; commits `9c9d75430` (daemon) +
  `a978bb8a6` (authored). **Follow-up**: cut `example/taskmanager v4.1.1` + proxy
  smoke in the same quiet window after verify (the v4.1.0 tag predates the fix).
- **T37 (deep-read the big READMEs): COMPLETE, zero drift.** All of catalog, graph,
  stack, storage, storage/view, watermill, otel, prometheus verified symbol-by-symbol
  against source (every cited func/type/option/ADR/script exists; CLI path
  `catalog/cmd/go-cqrs-lite-catalog` exists; `SubscribeAll`, `AutoMapperWithTombstone`,
  view aliases all real — three initial "missing symbol" alarms were my own
  rg-alternation shadowing, not drift).
- **T36f (readme link gate): SHIPPED.** `scripts/check-readme-links.sh` (wrapper
  feeding all 95 READMEs through check-doc-links.sh; `CHECK_READMES_ROOT`/`_FILES`
  hooks; 2-leg `--self-test` verified non-vacuous). Real run: **673 relative link
  targets, 0 broken** after fixing one false positive (backticked
  `id.Parse[T](x.String())` generics prose in cqrs-lint README). Daemon commit
  `5c035520f` + `9bf2df4c2`.
- **T36e (deprecated-symbol gate): WRITTEN, NOT YET VALIDATED.**
  `scripts/check-readme-deprecated.sh`: source-derived deprecated symbol discovery
  (per-file awk over doc comments — **98 deprecated exported symbols exist**, the
  ADR-0123 v5-removal surface: Bundle, Materialize, NewGraphProjection, AutoMapper*,
  view/relational aliases, DetectTombstone/MarkTombstone/TombstoneMark, ...),
  backtick-only citation matching (prose like "handler" can't trip it), banner
  escape hatch (README top-banner ⇒ skip), grandfathering baseline
  `scripts/readme-deprecated-baseline.txt` (not yet written).
  **PENDING: `--self-test` run, `--write-baseline`, real run, nightly-gates.yml
  wiring.** Interrupted here for this report.

## 5) Honest residue / what I'd criticize

1. `check-readme-deprecated.sh` violates my own goldens rule — written but its
   3-leg self-test has not been executed yet. Do that first on resume.
2. The ADR-0123 finding deserves more than a gate: ~15-20 module READMEs
   (stack/*, storage/view, graph, listing, projection…) currently teach
   deprecated-at-v5 APIs as their primary quick-start with no banner. The
   package-banner sweep (add `> **Deprecated:** removed in v5 … see X` to those
   READMEs, then shrink the baseline) is the real fix; the gate only stops NEW
   debt.
3. pgrep false positive: `pgrep -f 'nix run .#verify'` matched the supervisor's
   own cmdline — cost one confused cycle before reading the supervisor log.
4. The supervisor rc bug (§3) — cosmetic, known.
5. Not identified: the 10:31 concurrent editor. If verify 5 dies the same way,
   the next step is a `flock`-based edit guard on the repo during verify windows.

## 6) Next steps (in order, after owner instruction)

1. Poll `/tmp/verify-attempt5.log` for `VERIFY5-EXIT=`; on 0 → record **S03** in
   TODO_LIST's composed-verify row (date, commit, per-phase durations); on failure →
   triage from log, fix-forward, re-arm (with the rc-echo fixed).
2. Same quiet window: `check-readme-deprecated.sh --self-test` → real run →
   `--write-baseline` → wire both new gates into `nightly-gates.yml` (T19/T20
   step pattern).
3. `example/taskmanager` v4.1.1 cut + `--smoke-all` (validates the --help fix
   through the proxy).
4. T13 load-sweep → T14 benchmark-baseline re-pin (provenance header) → T15
   verify-ci + verify-docs e2e — same window, sequential.
5. ADR-0123 README banner sweep (from §5.2) + baseline shrink.
6. W3 owner bundle: the 2 remaining standing questions (readme_claims_test.go
   ownership; iroh P99 50→150ms keep-or-revisit) **plus new ones from this
   session**: (a) ratify the 10:31 repair ruling (restored 1.27.1 contract vs the
   unknown editor's downgrade), (b) flake.nix go pin vs nixpkgs 1.27.0 lag (§2),
   (c) whether to add a repo edit-lock during verify windows.
7. Closing status report + ledger update + push.

— Tree: uncommitted nothing (daemon absorbed everything; authored history where
the race allowed). Master pushed through `9212c8408`; local `5c035520f`..`9bf2df4c2`
(gates + README fix) not yet pushed — push on resume. Load 30.69 (verify race +
fleet).
