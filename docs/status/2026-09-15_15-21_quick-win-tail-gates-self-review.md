# Status Report — quick-win tail batch: calibration-gate self-test + golden, recipes compile harness

> **SUPERSEDED (2026-09-15 18:19):** this session's §f1–12 (catalog edits, 12 doc-lie fixes, `TestRecipesCompile` to zero, verification sweep, docs reconciliation) were EXECUTED by the completion session [`2026-09-15_18-19_quick-win-tail-gates-completion-self-review.md`](2026-09-15_18-19_quick-win-tail-gates-completion-self-review.md) — its §a table is the successor record. The three parent TODO items are ticked DONE 2026-09-15 in `TODO_LIST.md`. Remaining harness-hardening tails live in that report's §c/§f and are harvested into TODO_LIST (recipes-gate CI posture row).

> **Point-in-time snapshot:** 2026-09-15 15:21 CEST. Session scope: execute the three
> "quick-win batch follow-ups (2026-09-13)" TODO items — (1) calibration-gate.sh
> `--self-test`, (2) recipes.md snippet compile harness, (3) calibration-gate
> failure-message golden. This report covers ONLY this session; no unrelated research.
> Host load was 40-195 for most of the session (a compile storm) — noted where relevant.

---

## a) FULLY DONE (verified this session)

| # | Work                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Verification                                                                                                                                                                                                          |
| - | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **calibration-gate.sh `--self-test`** — new fault-injection suite: `CALIB_GATE_LOADAVG_FILE` env hook points the load probe at a temp fixture (never a live tracked file); 8 checks: quiet-host PASS shape pinned, load1-over fails, burst-drain load5-over fails (the v2 regression guard), warn-only override stays green, CI never aborts, provenance probes a stub binary + captures its version output                                                                                                                                   | `bash scripts/calibration-gate.sh --self-test` → 8× ✓ PASS, rc=0; `nix run nixpkgs#shellcheck` → clean (fixed SC2034 unused `name`, SC2001 sed → `${out//…}` param expansion)                                         |
| 2 | **FAIL-message golden** — `scripts/testdata/calibration-gate-fail-message.golden` pins the operator-facing text byte-exact; only the uptime line is normalized (content-anchored sed: indented lines that are not the `(load1=` line → `<UPTIME>`); golden generated from a live failing run and reviewed by eye                                                                                                                                                                                                                              | Mutation test: corrupted one word in the heredoc → self-test fails with "FAIL-message shape drifted from the golden" (rc=1); restored → rc=0. The pin provably bites                                                  |
| 3 | **CI wiring** — `check-release-scripts` flake app now runs the self-test leg; app comment updated (release tooling + gate-script smoke tests); runs in the existing CI `lint-scripts` job (ci.yml:638)                                                                                                                                                                                                                                                                                                                                        | `nix eval .#apps.x86_64-linux.check-release-scripts.program` → evals clean                                                                                                                                            |
| 4 | **Recipes compile harness (core machinery)** — `cmd/doc-check/recipes_extract.go` (fence scanner keyed by heading+ordinal; type/func decl hoister; whole-program detection) + `recipes_compile_test.go` (spec-driven generator: imports merge, package-level decls, per-block packages, one `go build ./...` per run)                                                                                                                                                                                                                         | `go test -run TestRecipesExtractor` green (unit fixture incl. hoist expectations); generated packages build in a temp module against a mirrored, absolutized `go.work` (GOWORK + GOEXPERIMENT=jsonv2 + CGO_ENABLED=0) |
| 5 | **Full classification ratchet** — `TestRecipesCatalogCoversFile` enforces EVERY fenced Go block in recipes.md has exactly one catalog entry (compiled scaffold or documented skip); stale entries fail too. 77/77 classified: 69 compile-scaffolded, 8 skipped with reasons (multi-`:=` fence, external go-appkit, 2× ellipsis bodies, CGO DuckDB, TOCTOU anti-pattern, 2× inner-scope-unused locals)                                                                                                                                         | Test green at time of full catalog; immediately catches new/reworded sections (this is the anti-rot ratchet the TODO asked for)                                                                                       |
| 6 | **Real doc lies found and FIXED in recipes.md** (the harness did its job before even being green): `otel.GetTracerProvider()/GetMeterProvider()` → `otel.NewTracer()/NewMeter()` (2 blocks); `middleware.CommandRetry(3, time.Second)`/`QueryRetry(3, …)` → `middleware.RetryConfig{MaxAttempts: 3, InitialDelay: time.Second}` (2 blocks); `sqlite.New(dsn, stack.WithFlightRecorder(…))` → `sqlite.New(dsn, sqlite.WithStack(stack.WithFlightRecorder(…)))`; `decider.WithEnricher(event.ActorEnricher)` → `decider.WithEnricher[State](…)` | Each fix verified against source signatures before editing (middleware/retry.go, otel/tracer.go, otel/meter.go, decider/options.go, stack/sqlite/preset.go, event/actor_context.go)                                   |
| 7 | **Same sqlite lie fixed at its source** — stack/options.go `WithFlightRecorder` doc comment showed the non-compiling wiring; recipes.md had copied it verbatim                                                                                                                                                                                                                                                                                                                                                                                | edit applied; signature re-verified against stack/sqlite `WithStack(opts ...stack.Option)`                                                                                                                            |

## b) PARTIALLY DONE

1. **`TestRecipesCompile` not yet green** — last run: 12 snippet packages failing, ALL
   with researched, queued fixes (compiler found them; that is the gate working):
   - Catalog edits pending: `_ = found` trailer (Filtered Scan), `_ = err` trailer +
     drop unused `time` import (§2.26 claiming), remove duplicate `StatusCounts`
     preamble decl (§2.21b), drop unused `record` import (Encoded Applies), add
     `metaengine` import + `CreatedPayload`/`UpdatedPayload` preamble types (Bridging
     Stream IDs), drop stray `ctx :=` preamble (§2.11), add `command` import (§2.13
     idempotency).
   - Doc edits pending (all researched against source): §2.9 catalog exporters —
     `asyncapi.NewExporter("My API","1.0.0").Export(cat).MarshalYAML()` (pointer
     method; `Exporter{}` literal is wrong) and `openapi…Export(cat)` returns ONE value
     → chain `.MarshalYAML()`; §2.13b retry — Config has no `Jitter` field (fields:
     MaxAttempts/InitialDelay/MaxDelay/Multiplier) and `AttemptFunc` is
     `func(ctx context.Context, attempt int) error`; block 10 minimal-ES recipe —
     `*CreateUser` does not implement `command.Command` (Type/StreamID/ID) → needs the
     `*command.BasicCommand` embedding pattern from
     `example/getting-started/docs_compile_test.go:140`; block 14 — explicit
     `WithSnapshotStore[UserState]`/`WithSnapshotStrategy[UserState]` type params;
     block 17 — `key` used before declaration (move the GenerateKey block above
     `NewXChaCha20Poly1305(key)`); block 13 — `&pebble.Options{}` ambiguous
     (storage/pebble.Open takes cockroach's options) → `pebble.Open(dir, nil, logger)`;
     block 37 — `config := system.DomainConfig{…}` redeclares the retry-config var →
     rename result to `cfg`; block 46 timers — `Timer.Actor` is now typed
     `id.ActorID` (ADR-0111e) so the `PrefixedString()`/`ParseActorID` round-trip is
     stale, `ID` needs `scheduling.TimerID(...)`, and `cmds.Dispatch(ctx,
     &CancelOrderCmd{…})` needs a real Command (plan: payload type +
     `command.New("order.cancel", …, command.WithActor(t.Actor))` at fire time).
2. **Harness hardening** — generator survived every documented Go shape so far, but
   only recipes.md §2.0–§2.35 blocks have been through it; other reference docs
   (core.md, advanced.md, readmodels.md, FAQ) are untouched by design ("start with
   recipes.md").

## c) NOT STARTED

- Full doc-check module verification after all edits (suite + the doc-check binary
  zero-warning gate over SKILL.md/references/AGENTS.md — my recipes.md edits changed
  symbol mentions).
- check-file-size on the three new files (`recipes_catalog_meta.go` may exceed the
  350-line cap — unverified; new files cannot be baselined).
- flake-level run of the doc-check module tests (the harness currently proven only via
  direct `go test`); `nix run .#check-release-scripts` end-to-end not yet executed.
- TODO_LIST.md checkboxes for the three items + CHANGELOG.md [Unreleased] entries +
  changelog-symbols gate.
- AGENTS.md/skill documentation of the harness (procedures + gotchas: bash
  dispatch-after-definitions, flag-reset-after-parse, env-export-for-subprocess,
  CALIB_GATE_LOADAVG_FILE contract).
- docs/status/README.md index entry for this report.
- Authored commits at task boundaries (daemon absorbed everything into `chore:` commits).

## d) TOTALLY FUCKED UP

1. **Trusted unreliable tool renders over compilers — repeatedly.** View/sed output
   this session kept contradicting itself and the test results (em-dashes flipping to
   `--`, letters transposing: cqrs/go-cqrs, pebble variants, sqlite variants; shifting
   line numbers). I burned multiple edit-retry cycles building `old_string`s from
   garbled renders ("modified since read", "must read first", "no changes made —
   identical content"). The compiler was right every single time. Protocol now burned
   in: **compiler/test output is ground truth; never "fix" a file to match a
   render.**
2. **Hand-rewrote a 440-line file from memory.** After a stale-read rejection on
   recipes_catalog_meta.go I full-rewrote it instead of editing surgically —
   introducing dozens of transcription corruptions (wrong import paths, wrong type
   names) in one shot. Caught it, recovered via the daemon's git history
   (`git show <sha>:path`), restored, re-applied only the researched fixes. Lesson:
   never full-rewrite long files from memory in this repo; restore from git and patch.
3. **Two multiedits issued from paraphrase memory** (flake.nix, recipes.md) with
   old_strings that didn't match the file — both rejected, no damage, but pure wasted
   round trips. The edit tool's staleness rejection is the only reason no damage
   occurred.
4. **No authored commits at task boundaries** — tasks 1+3 (self-test + golden + flake
   wiring) were fully green mid-session; the daemon absorbed them into `chore:
   auto-commit` commits, losing authored history (the exact trap the 09-13 lesson
   "commit per task when explicit commits are authorized" warns about).

## e) WHAT WE SHOULD IMPROVE (process observations from this session)

1. **Verification hierarchy for this repo:** compiler/test > raw-byte grep/sed > View
   render. When a render disagrees with a passing test, stop and re-run the test
   instead of editing.
2. **Edit discipline under an active co-writer:** something (formatter daemon and/or a
   second session) rewrote files mid-session — including reintroducing formatting and
   moving code I had just placed (SELFTEST_MODE init migration was done by it,
   correctly). Check `git log`/mod-times before edits; expect drift between read and
   edit.
3. **Run the compile gate after every 2-3 fixes**, not in big batches — the error
   list shifts between runs when the tree is being concurrently reformatted.
4. **Whole-file `write` beats multiedit** for heavily-changed regions; multiedit only
   with freshly-viewed exact text.
5. **The daemon is also a recovery mechanism:** its commits gave me the restore point
   that saved the catalog. Keep committing per task anyway — authored history and
   recovery points are both needed.
6. **Harness runtime economics:** first compile run = 104 s (CGO_ENABLED=0 cache-miss
   vs workspace's CGO builds); warm runs ≈ 5-15 s. Worth documenting, or aligning
   CGO setting with the workspace to reuse cache.
7. **Golden tests need mutation-proofing as part of the definition of done** — done
   here for the calibration golden (corrupt → fail → restore → green); do the same
   for any future golden.

## f) NEXT (most valuable first)

1. Apply the 7 queued catalog-B edits (trailers `_ = found`/`_ = err`; drop dup
   `StatusCounts`; drop unused `time`/`record` imports; add `command`/`metaengine`
   imports; add `CreatedPayload`/`UpdatedPayload` preamble types; drop stray `ctx` in §2.11).
2. Doc fix §2.9 catalog block: asyncapi/openapi Export shapes (researched, see b.1).
3. Doc fix §2.13b retry block: drop `Jitter`, `AttemptFunc(ctx, attempt int)` signature.
4. Doc fix block 10 (minimal ES): BasicCommand embedding per docs_compile_test.go:140.
5. Doc fix block 14: explicit `WithSnapshotStore[UserState]`/`WithSnapshotStrategy[UserState]`.
6. Doc fix block 17: move key-generation above `NewXChaCha20Poly1305(key)`.
7. Doc fix block 13: `pebble.Open(dir, nil, logger)`.
8. Doc fix block 37: `config :=` → `cfg :=` for the DomainConfig result.
9. Doc fix block 46 timers: payload type + typed `Actor: actor` + `scheduling.TimerID(...)`
   - `command.New(…, command.WithActor(t.Actor))` at fire time.
10. Drive `TestRecipesCompile` to zero; keep the classification ratchet green.
11. check-file-size on the three new files; split `recipes_catalog_meta.go` if over 350.
12. Full doc-check module test suite + doc-check binary over the docs (zero-warning gate).
13. `nix run .#check-release-scripts` end-to-end (includes the new self-test leg).
14. Decide CI posture of `TestRecipesCompile` (cold cache + proxy fetches): in `#verify`
    unconditionally vs separate `check-recipes-compile` flake app.
15. TODO_LIST: tick the three items with dated evidence notes.
16. CHANGELOG [Unreleased]: Added (self-test, golden, harness) + Fixed (six recipes.md
    lies, stack/options.go comment); run check-changelog-symbols.
17. AGENTS.md: document the harness (Change-an-Exported-Symbol procedure + Testing
    section) and the CALIB_GATE_LOADAVG_FILE contract.
18. docs/status/README.md index entry for this report.
19. gotchas-tooling-build.md: bash gotchas found this session (dispatch-after-definitions,
    flag reset after parse, exported-env-for-subprocess).
20. Grep-sweep other skill references for the same `Get*Provider`/positional-Retry lies
    (cheap; harness only covers recipes.md).
21. Extend harness to advanced.md/readmodels.md/core.md (original TODO said "start
    with recipes.md") — new follow-up entry.
22. calibration-gate: validate `--max-load` is numeric (currently awk-errors on junk).
23. calibration-gate: help text `sed -n '2,36p'` is brittle — consider anchor-based range.
24. Consider pinning PASS and WARN-ONLY message shapes with goldens too (currently
    grep-substring asserts).
25. Harness: unit tests for `extractBodyImports`/`importPath`.
26. Harness: failure message should name snippet headings, not `recipe_lNNN` package names.
27. Harness: consider `-short` behavior decision (currently always runs; warm ≈ 5-15 s).
28. Block 3 (multi-`:=` fence): consider splitting the fence into three fences so it
    becomes compilable and the skip retires.
29. Block 63 (TOCTOU): consider splitting WRONG/RIGHT halves so the RIGHT half compiles.
30. Block 9 (go-appkit): decide fixture-module go.mod vs permanent skip.
31. Verify calibration-drift.sh still composes with the gate (exit codes unchanged).
32. Grep for other scripts reading /proc/loadavg that might want the same fixture hook.
33. Re-run md-go-validator over recipes.md after doc fixes (no new pseudo-code regressions).
34. Confirm no go.mod/go.sum/go.work.sum mutations leaked from harness runs
    (go.work.sum is gitignored; verify `git status` clean of those).
35. Authored commits per task before the daemon absorbs the final state.
36. `nix run .#verify-fast` in a quiet window (host load 40-195 all session — the gate
    itself would refuse a bench right now).
37. Session note in AGENTS.md memory: "compiler is ground truth; renders lie" incident
    - protocol.
38. Re-check that `example/getting-started` pattern (docs_compile_test.go) and the new
    harness don't duplicate coverage — document the boundary (hand-written compile
    tests vs generated from fences).
39. Consider a `--list` mode for the harness (print compiled/skipped inventory) for
    review ergonomics.
40. Sweep: does any consumer doc claim `middleware.QueryRetry(3, time.Second)`-style
    signatures elsewhere (FAQ/core) — same class as fix #3.

## g) QUESTIONS (cannot be resolved from the repo alone)

1. **Who is the co-writer?** Mid-session, files I had just written were reformatted and
   reordered by something else (my SELFTEST_MODE fix appeared pre-applied; catalogs
   absorbed formatting passes; daemon commits `eaafa93b5`/`0a958a2e2` absorbed my WIP).
   Is that a save-hook formatter (treefmt daemon) or a second agent session? If a
   second session is active on this tree, I should stop and coordinate before finishing
   the harness rather than race it.
2. **CI posture for the compile gate:** `TestRecipesCompile` fetches proxy deps on cold
   cache (go-idempotency, go-retry, go-flightrecorder, pebble, otel are doc-imported)
   and costs ~104 s on first run (CGO-flag cache split). Do you want it inside
   `#verify`/`#test` unconditionally, or as a separate `check-recipes-compile` flake
   app that CI runs as its own leg (my recommendation: separate app, so a proxy hiccup
   doesn't fail the whole verify)?
3. **Skip philosophy for the 8 non-compiling blocks:** are documented skips acceptable
   as the permanent state (my recommendation for ellipsis/anti-pattern/CGO blocks), or
   do you want the doc fences restructured so they compile (blocks 3 and 63 are
   restructurable) — i.e., is the goal "every fence compiles eventually" or "every
   fence is DECIDED"?

---

**Session bottom line:** Tasks 1+3 (calibration-gate `--self-test` + failure-message
golden + CI wiring) are fully done and mutation-verified. Task 2 (recipes compile
harness) is functionally complete as machinery — 77/77 blocks classified, ratchet
green, and it already caught and fixed six real doc lies before being fully green —
but the compile leg still has 12 known-and-researched failures queued. The session's
biggest self-inflicted cost was trusting garbled tool renders over compiler output;
the biggest external risk noticed is the unidentified co-writer actively reformatting
this tree mid-session.
