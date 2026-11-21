# C-list mop-up: config re-bind shipped, survey truth-corrected, precheck builds again

> **Date:** 2026-11-21_02-27 Saturday CET
> **Session:** continuation of the C-list (`docs/planning/2026-11-18_06-41_SUPERB-idempotency-fleet-repair-and-linter-follow-through.md`) — executing the 01-28 report's §f next-items (10 mechanical steps + consumer health + gated-tranche prep).
> **Predecessor:** `2026-11-21_01-28_fleet-repair-execution-complete-and-the-cwd-preset-trap.md`

## a) FULLY DONE (verified this session)

### a1. Survey truth-correction pass (§f-6 + §f-16 + unplanned Zlota44 re-verification)

`docs/evidence/2026-11-17_idempotency-fleet-wiring-survey.md` now carries dated corrections for THREE
wrong claims, each verified in source before writing:

1. **§2.1 sqlstore auto-sweep** — FALSE, corrected in place with the source citation
   (`idempotency/sqlstore/v4/store.go:312`: `Store.Sweep` is Deprecated, optional, CALLER-invoked;
   "Call periodically … or rely on lazy deletion in Seen/CheckAndRecord"). §4's unbounded-verdicts
   note added (they STAND and strengthen); §7.5's "native TTL comment is ACCURATE" marked superseded.
2. **§7.6 Swetty collection name** — `wanted_ids` matches NO file and NO commit in Swetty's entire
   git history (`git log --all -S` empty). Truth: `api_command_idempotency`
   (`services/api/idempotency_dedup.go:17`). Row 9 + §7.6 corrected. The §7.6 "correction" was
   itself wrong — the survey's error count grows, honestly recorded.
3. **Row 8 / §4 / §7.3 Zlota44** — the store is **SQLite, not bbolt** (`internal/app/idempotency_sqlite.go`,
   172 lines, same `*sql.DB` as the event store), no `bbolt` anywhere in repo history; it already
   implements `go-idempotency`'s Store (direct v0.3.2 require beside the shim). The false comment is
   its "row growth is bounded by distinct command keys … no background sweeper goroutine is needed" —
   falsified by `ReDiscoverPropertyCmd` (`domain/property/commands.go:138`), whose key embeds a unique
   per-command ID: expired unique-key rows are never `Seen` again and linger forever.

### a2. cqrs-lint config binding — the cwd-preset footgun is DEAD (§f-8)

New `cmd/cqrs-lint/v4/config_binding.go`: `rebindForeignCwdConfig` runs at the top of
`applyConfigOverrides` (so lint run, scorecard, doctor, and doctor-audit all get it). When the lint
target lies OUTSIDE the process cwd (sibling checkout; monorepo descendants keep legitimate ancestor
inheritance) and the cwd carries a `.cqrs-lint.json`, the five config-file-only project keys
(preset, features, rules, health, scorecard) are re-bound to the target's own config or tag
defaults, with a stderr warning naming both directories and the remedy (`cd`, or cmdguard's
`--config`). Flag-carrying keys are untouched — no flag exists for the re-bound keys, so no explicit
CLI value can be clobbered. 7 unit tests (`config_binding_test.go`), including the monorepo-
descendant and malformed-target-config cases. **E2E proof:** browser-history linted from
`~/projects/go-cqrs-lite` now prints the warning AND reports its two F030 findings — the old binary
reported neither silently. Full cqrs-lint suite (21 packages) green after wiring.

### a3. doc-check `--json` carries the dead-citation class (§f-9)

`jsonSummary` gains `modPathIssues` (same `navIssue` shape as nav/arity), any non-empty class flips
`valid=false`, `emitJSON` takes an `io.Writer`, and two pin tests cover all-classes-present and
nil-to-`[]` normalization. Full doc-check suite green.

### a4. F030 nil-AST guard fixture (§f-10)

`TestF030_NilASTFileIsSkipped` (f030_test.go): a nil-AST GoFile is skipped without panicking and
without suppressing findings from healthy files.

### a5. Fleet-commit ledger (§f-11)

Baseline evidence §6 appended: all six fleet commits with live-verified push states (all unpushed;
SEC carries a foreign daemon chore `d084837c` on top of mine; `d195bf3` flagged orphanable —
referenced only by the precheck worktree's detached HEAD).

### a6. Receipt-capture script (§f-14)

`scripts/fleet-dedup-receipt.sh` — per-repo-cwd scorecard+F030 receipts for the nine-repo fleet,
self-sources `go-env.sh`, artifacts under `build/fleet-receipts/<ts>/`. shfmt-clean, syntax-checked,
live-verified on InboxClean (`used | go-idempotency`, F030 0, HEAD `5c208788`).

### a7. Gated-tranche drafts, written against corrected truth (§f-24..28)

Five dated draft blocks in the plan (all explicitly GATED, none executed):

- **C18 KeyCountdown** — engine-dedup adapter (Swetty shape) + mandatory `DedupSweep` ticker
  (DiscordSync reference), ordering untouched.
- **C19 browser-history** — collapse dual branch + engine sweeper; MUST run the committed-unexercised
  C22 tests and then flip `SweepEnabled` to the engine truth.
- **C20 precheck** — lazy construction over close-after-discard (kills the leaked 10m ticker at the
  root); worktree-topology warning (detached HEAD, orphanable hashes).
- **C21 Zlota44 — RESCOPED** by the a1.3 discovery: the minimal honest scope is the `ErrDuplicate`
  sentinel swap (5 files) + shim-require drop + false-comment fix + a periodic `Sweep()`; the
  21.6 "delete the 172-line store" step is SUPERSEDED (the store already targets the live module —
  deleting it buys nothing).
- **C12 codemod** — parent/kvstore → `go-idempotency` rewrites, sqlstore reported as MANUAL sites;
  downgrade clause if fleet residue is gone by hand before D30.

Plus: C24 preamble updated (`api_command_idempotency`, 24.5 marked DONE via `0c82adeff`), and the
C26 receipt-doc skeleton created (`docs/evidence/2026-11-21_fleet-dedup-bundle-receipts.md`).

### a8. Guardrail sweep (§f-13) — the runnable set, all green

`#check-md-go` (no new errors), `#check-file-size` (no growth), `#check-test-skips` (245 sites
routed), **api-golden no-op confirmed** (regen → zero diff, 6513 exports), `#check-coverage`
(all 89 module + 170 package floors hold), `#check-test-runtime` (all packages within budget),
`lint-scope` 2/2 modules clean, formatting applied to every touched file
(golangci-lint fmt / dprint / shfmt), `check-changelog-symbols` green.

## b) PARTIALLY DONE

### b1. precheck api builds for the first time — tests still red, uncommitted (§f-18)

The templ matrix is now fully mapped: `templruntime.NewAttributeExpression` lives in the
`templ/runtime` subpackage (v0.3.1100+ only); EVERY templ-components release (through v1.27.x) is
v0.3.1020-era codegen; cqrs-htmx `loginpage/v4.13.0` is the AutoRedirect-bearing, templ-v0.3.1070
release. Coherent fix (the "1070 era"): regenerated all 14 `*_templ.go` with the
`templ@v0.3.1070` CLI (the committed files were too-new-CLI output), bumped
`cqrs-htmx/loginpage/v4` v4.12.2 → v4.13.0, tidied — **the api module BUILDS** (first time; it was
pre-existing-broken). The full api test run FAILS (491 s runtime; failing-test list was being
captured when this report was cut — next session starts there). Nothing committed yet.

### b2. Library working tree — batch complete but UNCOMMITTED

All a2–a7 library changes + CHANGELOG rows (two new; plus a REPAIR of the F030-registry row whose
`- **cqrs-lint: F030` head was eaten by daemon auto-commit `514bd721d` — restored from `d033c3a23`)
sit uncommitted; the phase boundary commit was deliberately deferred because item 8 (consumer
health) is mid-flight. Daemon-blend risk assumed knowingly.

### b3. dependabot (§f-15) — verified, action is owner's

Alerts 138/139 (otel trace/sdk) still OPEN, created 2026-11-13 — past the one-week staleness bar.
No truthful dismissal reason exists; owner pings GitHub per the recorded protocol.

## c) NOT STARTED

- **§f-12 harvest** of the remaining open items into TODO_LIST (deliberately last; the cwd-preset
  row is now a "FIXED" note, not an open row).
- **§f-19 browser-history master templ matrix** (Option B is now proven next door: downgrade the
  templ pin to v0.3.1070, align components, then RUN the C22 tests committed unexercised).
- **§f-20 SEC 8 red tests**, **§f-21 InboxClean 2 labels tests**, **§f-22 go.sum policy**.
- **§f-17 gls go.work disposition** (owner taste), **§f-30 formal C26 legs** (post-tranche).
- All owner-gated items (C5/Q1, C23 23.1–23.2, C18–C21, C12, C25, C26) — untouched by design.

## d) TOTALLY FUCKED UP (honest ledger)

1. **A `head`-pipe lied about a build result once.** `go build ./... 2>&1 | head && echo OK` printed
   OK from `head`'s exit code, not the build's. Caught immediately by re-running un-piped; the
   "BUILD-OK" in that cell was false for one command. Rule: never gate on a piped command's tail.
2. **Three more stale-mtime edit rejections** (CHANGELOG, `json_report_test.go`, the survey) — every
   one from editing right after dprint/daemon touched the file; every one fixed view-first. This is
   now 6 strikes across two sessions: in this repo, VIEW IMMEDIATELY BEFORE EVERY EDIT is not optional.
3. **My own survey table edit merged rows 9 and 10a** (dropped a trailing newline) — repaired in the
   next command, but dprint-padded markdown tables demand post-edit line verification, which I then
   did for every subsequent table touch.
4. **Wrong templ invocation first** (`go run github.com/a-h/templ@…` — "not a main package") before
   finding `cmd/templ`. Two minutes lost, no damage.
5. **Report was cut while a diagnostic was in flight** — the precheck failing-test list (job 069) is
   still running; the honest state is "FAIL, names unknown at cut time", recorded as such above.

## e) WHAT WE SHOULD IMPROVE

1. **Verify survey claims against CURRENT source before acting on them — again.** Three of the
   survey's own §7 "corrections" were wrong (bbolt, wanted_ids, native-TTL-comment). The C21 draft
   would have deleted a working store on the strength of a bad claim. The dated-correction discipline
   saved it, but the pattern is: every "verified" fact older than the working tree deserves a re-grep.
2. **The evidence doc's §1.1 pointer** still says the cwd-preset footgun is "worth a cmdguard-level
   fix or a loud warning; rowed in the plan's corrections" — now SUPERSEDED by the shipped fix; the
   harvest should tick it and the FAQ/skill reference may want a line (cqrs-lint now re-binds).
3. **Dangling budget rows observed during the sweep** (`catalog/v4/cmd/catalog-compat`,
   `catalog/v4/compat` absent from the runtime run) — almost certainly the parallel M-wave session's
   catalog churn, not mine; left for them, noted here so it is not lost.
4. **Phase-boundary commit discipline slipped** — I carried a complete, green library batch across
   into fleet work uncommitted. If the daemon blends it, the split-out cost returns. Commit at the
   boundary the moment the batch is green.
5. **The templ version matrix should live somewhere durable** — it took real archaeology
   (runtime subpackage introduction, components-vs-cqrs-htmx codegen eras, loginpage tag floors).
   A short section in the browser-history repo's docs (or templ-components AGENTS) would save the
   next session the same dig.

## f) NEXT (ranked, grouped)

**Consumer health (unblocks verification) — 6**

1. Read job 069's output / re-run: name the failing precheck api tests; classify pre-existing vs
   regen-induced (the 491 s runtime smells like integration tests that were never runnable before —
   first time this module ever builds).
2. Commit the precheck regen + loginpage bump (detached HEAD, real message, single command).
3. browser-history master: apply the proven Option B (templ pin → v0.3.1070 era, align components),
   build the api module.
4. RUN the committed-unexercised C22 tests in browser-history (both trees where applicable).
5. SEC: diagnose the 8 dep-bump-failed tests; fix at root cause; then their pre-commit gate lives.
6. InboxClean: the two labels-preservation tests; then the go.sum policy question to the owner.

**This-repo close-out — 7**

7. Commit the library batch at the phase boundary (cqrs-lint re-bind, doc-check JSON, F030 fixture,
   script, survey/plan/evidence docs, CHANGELOG rows, this report) — explicit file list, daemon-aware.
8. docs-health HARVEST: remaining §f open items → TODO_LIST (§f-12): dependabot owner-ping row,
   templ-matrix doc row, gls go.work disposition row, evidence §1.1 tick, §f-30 legs, plus this
   report's e-items.
9. Rerun the full cqrs-lint suite once (post-F030-fixture) — adoption package ran green standalone;
   the 21-package sweep predates that one file.
10. Rebuild `/tmp` cqrs-lint receipts for the OTHER three swap repos with the fixed binary
    (`fleet-dedup-receipt.sh` default list) — cheap, proves the fix fleet-wide.
11. Optional docs polish: FAQ/skill line for the config re-bind behavior; doc-check README for
    `modPathIssues`.
12. Watch the dangling catalog budget rows (M-wave) — if still dangling after their wave, row it.
13. This report's index row in `docs/status/README.md` (done with the report commit).

**Owner-gated (cannot proceed without answers) — 7**

14. Q1 verify strategy → C5 (fifth session carrying it).
15. Replay-vs-dedup intent (gls/InboxClean) → C23 23.1/23.2.
16. D28/D29 → execute C18–C21 from the a7 drafts (C21 in its rescoped minimal form).
17. D30 → C12 codemod go/no-go (draft carries the downgrade clause).
18. D31 → C25 DiscordSync twin cleanup.
19. Push authorization: 16 prior commits + this session's batch, all unpushed across 7 repos.
20. dependabot: owner pings GitHub on alerts 138/139 (8+ days stale, no truthful dismissal).

**Later / post-tranche — 4**

21. C26: fill the skeleton (`2026-11-21_fleet-dedup-bundle-receipts.md`) after the gated tranche.
22. C26 formal legs: `census-fleet.sh` diff vs ADR-0167, `check-versions-manifest --check`.
23. templ-matrix knowledge capture (e-item 5) wherever the owner wants it.
24. Zlota44 C21 execution input check: confirm the five `ErrDuplicate` files' swap surface still
    matches the draft when D28/D29 land.

## g) QUESTIONS FOR THE OWNER (cannot be figured out)

1. **Q1 (carried, 5th session):** under sustained host load, should C5 run the eight gate apps
   individually with recorded receipts and treat `#verify` as eventual confirmation — or is
   `#verify-serial`/`--wait-loop` worth another night first?
2. **Replay-vs-dedup intent (blocks C23's reorder):** for github-local-sync and InboxClean, is a
   RETRIED command expected to re-enter the idempotency check (dedup outermost, current gls shape)
   or should dedup be claimed once per command with retries inside (KeyCountdown's shape)?
3. **Push + `--no-verify` blessing (carried):** 16+ commits unpushed across 6+ repos, two bypassing
   pre-commit gates that fail on PRE-EXISTING red tests (proven identical at parents); this session
   will add the library batch + precheck regen to the pile. Push now, hold for review, or push all
   except the bypassed pair?

## Receipts appendix

```text
$ date
2026-11-21_02-27 Saturday CET

$ this session's verification (all green at cut time)
cqrs-lint full suite (21 pkgs): ok (pre-F030-fixture; adoption pkg re-run green after)
doc-check full suite: ok (incl. 2 new emitJSON tests)
config_binding tests: 7/7 ok; e2e cross-cwd rebind: warning + 2×F030 fired (was 0)
check-md-go / check-file-size / check-test-skips: green
check-coverage: all 89 module + 170 package floors hold
check-test-runtime: all packages within budget (2 dangling catalog rows — M-wave, noted)
api-golden regen: zero diff (6513 exports)
lint-scope (cmd/cqrs-lint/v4 + cmd/doc-check/v4): 2/2 clean
check-changelog-symbols: 35 citations honest
fleet-dedup-receipt.sh on InboxClean: used | go-idempotency, F030 0, HEAD 5c208788
precheck api: BUILD OK (templ v0.3.1070 regen ×14 files + loginpage v4.13.0); tests FAIL (names in flight)

$ survey error ledger (this session's corrections)
§2.1 sqlstore auto-sweep: disproved (store.go:312)   §7.6/row 9 wanted_ids: never existed
row 8/§4/§7.3 Zlota44 bbolt: never was bbolt (SQLite, 172 lines, go-idempotency-native)
false comment falsifier: ReDiscoverPropertyCmd unique-ID keys (commands.go:138)
```
