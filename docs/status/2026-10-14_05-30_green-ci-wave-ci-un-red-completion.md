# Status Report — Green-CI Wave: CI Un-Red Completion + M10 Verify Chase (2026-10-14 05:30)

Session: 2026-10-14 02:44 → ~05:30 CEST. Continuation of the M10–M20 session
(status 00-40). Standing instruction: execute the plan until done. This session
chased the remaining red to ground across CI and the composed `#verify`.

## a) What was DONE (fully, verified)

1. **M10 root-caused: the end-of-wave `#verify` failure was a REAL cross-mode
   contract divergence, not a flake.** The 01:00 firing failed 2 `system`
   tests ("expected view, got `<nil>`") — deterministic in workspace mode,
   green GOWORK=off. Bisect via per-module `replace` isolated it to
   `metaengine/projectionadapter`: the working tree moved `EventWithID.ID`
   to the BARE stream value (`StreamID.Get()`) while `system`'s tests still
   queried display-form `String()`; GOWORK=off legs stayed green via tagged
   v4.5.3 (old contract). Fixed: tests query `.Get()`; `system/go.mod` gains
   the sibling `replace …projectionadapter/v4 => ../metaengine/projectionadapter`
   (stripped at cut time; projectionadapter+system co-release — schema-replace
   pattern). Both modes green locally. Same branded-ID display-vs-bare family
   as the earlier storage/PG fix: go-branded-id v0.7.0 made `String()` a
   prefixed DISPLAY form.
2. **actionlint+shellcheck reds fixed for real** (not disables):
   `check-adr-numbering.sh` ls-globs → `adr_files()` helper (SC2012 ×2,
   dedups both call sites; `--self-test` still catches every mutation);
   never-assigned exemption via loop (SC2086); `ephemeral-pg.sh` flag
   pass-through as an array (`"${GO_TEST_FLAGS[@]}"`, SC2086) — verified
   LIVE against ephemeral PG.
3. **Ephemeral PG job: scheduling/sqlstore timer tests** compared read-back
   `TimerID.String()` to bare keys (masked until the flag-routing fix let
   the suite actually run). Assertions now `.Get()`; full suite green
   against real PG locally AND on CI run 38497273099.
4. **Nix Flake Check / Verify-fast shared reds:**
   - `cqrs-lint` C014/C015/C016/C025/D006/D017 baked ABSOLUTE paths into
     finding messages — new `(*analyzer.AnalysisContext).RelPos` renders
     project-relative `path:line:col`; taskmanager golden regenerated
     machine-independent; api golden 6301→6302 exports.
   - `tursoengine` vector test asserted bitwise-equal float distances —
     libSQL ULPs differ across builds (4.5e-08 on CI, 0 locally); now ±1e-6
     semantic equality per the ADR-0140 contract.
5. **go.sum tidy classes:** the 02:46 `-mod=mod` superset wave left
   everything untidy. 81 meta-test modules + the 10 CI-only modules
   (cmd/api-stability, integration, 8 examples) re-tidied with the
   dev-shell toolchain — `check-modsums` 104/104, `TestEvery` green.
6. **Benchmark job:** `go-env.sh` GOCACHE/GOLANGCI_LINT_CACHE now behind
   the same creatability probe as TMPDIR (runners keep go defaults — the
   noise gate now actually RUNS); benchmarks.yml baseline-refresh guarded
   (a noise-gate abort no longer cascades into `cp: cannot stat`).
7. **`check-private-deps`:** parallel session adopted
   `github.com/larsartmann/go-sqlitestore` across 13 modules without the
   audit entry — repo verified PUBLIC, added to `AUDITED_PUBLIC`.
8. **benchkit `TestCompare_ThreeBackends`** 60s→150s (CI runner blew the
   60s deadline at 60.36s; 150s is the file's heavy-suite precedent).
9. **golangci config hash re-pinned:** the 19:26 systemscenario carve-outs
   (prior session) landed without the golden re-pin; gvac then regenerated
   the depguard allow-list mid-session (the documented recurrence class).
   `#check-lint-config` green with the new pin.
10. **Docs/records:** CHANGELOG rows (4), TODO_LIST composed-verify STATE,
    plan M10 annotation, gotchas-module-management.md new section
    (toolchain-dependent tidy; ambient 1.27.0 vs nix 1.27.2; flake
    snapshot/daemon race).

## b) IN PROGRESS (background, passive — DO NOT disturb)

- **CI run 38505591805 on tip `f2b2b29d7`** — the authoritative full-matrix
  run carrying every fix. Push-frozen until it completes.
- **Composed verify chain (bg shell 2A0, armed ~05:20, 4h window):**
  `can-run-composed-gate --wait-loop → check-coverage → nix run .#verify`.
  Load 55 at arming; it waits for <10. The prior firing (shell 267) passed
  build+vet+test+race+lint-91/91+arch+modsums and died ONLY at the
  lint-config hash gate — now fixed. If the window lapses, re-arm.

## c) What's NEXT (in order)

1. Watch 38505591805 to green. Expected residual risks: none fixed-class;
   watch Turso Sync / Verify fast for the tursogo runner segfault (see f1).
2. Verify chain green → record the M10 receipt in plan §8 + TODO_LIST row;
   write the F-64 close-out addendum on the 19-26 status report.
3. THEN (gated on both greens + Lars's tag-wave go-ahead): changed-set
   per-module `GOWORK=off go test` sweep (metaengine needs a re-run),
   second tag wave (`stack/v4.6.0`, `metaengine/v4.19.1`,
   `cmd/cqrs-lint/v4.18.0`, `systemscenario/v4.2.0`, plus
   projectionadapter+system co-release pairing), pin-sweep, versions
   manifest, proxy smoke.

## d) What was NOT started

- The tag wave itself (irreversible; awaits verdict, §g).
- Turso upstream segfault ask (tursogo v0.8.2 runner SIGSEGV in
  `turso_database_new` — TODO row exists from prior session).

## e) WTF moments / lessons (recorded in gotchas + CHANGELOG)

1. **GitHub "Autoretry" runs on this repo are HOLLOW** — a "Rerun failed
   jobs (once)" meta-job concludes success WITHOUT re-executing anything.
   I briefly trusted one as evidence (wrongly undid my earlier skepticism);
   only per-job lists count.
2. **Three toolchains on this host:** ambient `go` 1.27.0, nix devShell
   1.27.2, `GOTOOLCHAIN=auto` never upgrades past a satisfied `go 1.27`
   directive. `go mod tidy` output is toolchain-shaped → tidy waves must
   use the dev-shell binary.
3. **`nix run .#verify` snapshots the tree at invocation** — a daemon
   commit between the gate's stability window and nix's evaluation means a
   verify can test a stale tree (the 03:26 tidy failure was this race).
4. **Each push cancels the previous full-matrix run** — final answers need
   a push freeze + one authoritative run.
5. The prior session's "Nix Flake Check = README floors, FIXED" attribution
   was wrong — the real failures were the abs-path golden, the vector
   epsilon, and the system pair. Lesson (again): read the failing TESTS,
   not the job name.

## f) What I would do differently

- Run `nix shell nixpkgs#{shellcheck,shfmt,actionlint}` CI-parity preflight
  BEFORE pushing script changes (two avoidable red push cycles).
- Treat `check-modsums` (CI, ALL modules) and `TestEveryModuleGoSumIsTidy`
  (meta-test, 81 modules) as ONE obligation — tidy the union, not the set
  the failing gate names.

## g) Questions for Lars (blocking or policy)

1. **Benchmark noise-gate policy on shared runners** (VERDICT): the gate
   now runs (GOCACHE fixed) and failed legitimately — `write_throughput`
   CoV 12.2% vs the 10% headline threshold, ONE observation, shared
   ubuntu runner. Options: keep as-is (fails loud on noisy runners),
   raise CI `--noise-threshold`, bump `--noise-repeat`, or demote
   write_throughput with evidence per the write_p99 precedent. I did NOT
   touch the threshold — the repo's own evidence standard (write_p99
   demotion took 5 runs incl. quiet windows) says one observation is not
   enough.
2. **Turso verify-job red**: still accepted until upstream answers?
3. **Tag wave**: scope/timing OK per c3? (projectionadapter+system must
   co-release due to the new sibling replace.)
4. **gvac depguard regeneration** (f-recurrence): the allow-list was ABSENT
   from HEAD (lost between 10-10 and 10-13 — worth its own look) and gvac
   regenerated it this session; pinned and green. Standing verdict on the
   durable guard (TODO_LIST §CI row) still open.
