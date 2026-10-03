# Status Report — cqrs-lint Subcommand Consistency Overhaul

**Date:** 2026-10-03 04:25
**Session scope:** cqrs-lint subcommand audit + fixes (triggered by "I feel like our cqrs-lint has sub command concistency issues!"). This report covers ONLY this session's run and what was noticed along the way. No fresh repo-wide research was performed.
**Baseline at session start:** HEAD `40865b382`, clean tree.
**Evidence anchors:** work absorbed by auto-commit daemon (`81b35f2ac`, `96f90dc6d`, … — chore commits, not authored); verification evidence is test/lint/gate output cited inline.

---

## a) FULLY DONE

All verified by: `GOWORK=off go build/vet/test ./...` (19/19 packages ok, `cmd/cqrs-lint` module), `golangci-lint --config .golangci.yml ./...` → **0 issues**, `nix run .#check-md-go` ✓, `scripts/check-readme-links.sh` ✓ (0 broken), `scripts/check-changelog-symbols.sh` ✓, `nix fmt` → 0 changed.

1. **Empirical consistency audit (10 confirmed findings).** Built probe binaries from source and demonstrated each defect before fixing (`rules --format json` → text; `scorecard --format csv` → text rc=0; root `--format bogus` → silent text; `doctor --format yaml` → error only after full package load; ~16 lint flags accepted as no-ops on every subcommand; root `--scorecard` ≠ subcommand output; config-file `format`/`color` ignored by scorecard/doctor; `init --path` writes to CWD; help-list drift: `changelog` missing from hand-written usage; per-command format vocabularies + case-sensitivity divergence).
2. **F1 — shared `validateFormatFlag` + fail-fast validation** (`cmd/cqrs-lint/output.go:44`, static sentinel `errInvalidFormat`). Root validates its 6-format set before `BuildContext`; doctor validates {text,json} BEFORE the package load (was after — 9ms fail vs full analysis); scorecard validates {text,json,markdown,sarif} before load; rules honors the inherited `--format` with `--json`/`--markdown` booleans taking precedence. Empty string = zero value = default text (case-insensitive everywhere — doctor was previously case-sensitive).
3. **F2 — one scorecard code path.** `runScorecard(ctx, cfg, actx, threshold)` (`scorecard_command.go:52`) is now the ONLY entry point; root `--scorecard` flag and the subcommand produce byte-identical output (verified with `diff`), both include the Deprecated panel; threshold CI gate armed only via subcommand.
4. **F3 — flag scoping.** 16 lint-run-only flags marked `local:"true"` in `AppConfig` (`main.go:107`): fix, dry-run, fast, health-score, only, exclude, exclude-rules, verbose, group-by, quiet, fp-suspects, show-suppressed, strict-load, fail-on-stale-suppressions, adoption, scorecard. `version --fix` is now an unknown-flag error (rc=1) instead of a silent rc=0 no-op. Shared/persistent kept: path, format/-o, color, min-severity, min-confidence, typed-info (all actually read by subcommands). Scoping contract documented in the `AppConfig` doc comment.
5. **F4 — `init` honors `--path`** (`init.go:26`): writes `DIR/.cqrs-lint.json`, errors on missing dir, keeps `errConfigExists` on second run. Default-path message byte-identical to before.
6. **F5 — root help lists `changelog`**; `--scorecard` root-flag help now points at the subcommand for the threshold gate.
7. **Byte-identity preserved:** `rules --markdown > RULES.md` and `rules --json` output `cmp`-identical to the pre-change binary (RULES.md regen workflow untouched; `renderRules` reproduces the exact Println/Print semantics per format).
8. **Regression suite:** `cmd/cqrs-lint/subcommand_consistency_test.go` — 10 tests incl. in-process e2e via `cmdguard ExecuteWithArgs` (unknown-flag matrix on version, doctor/scorecard format rejection, init path handling, `runScorecard` threshold gate, `rulesFormat` table, `validateFormatFlag` table, root fail-fast).
9. **Fixed 5 PRE-EXISTING lint findings on sight** (module was lint-dirty on master): loader.go err113 (dynamic error → `errSilentEmptyLoad` sentinel) + revive unused-param; toolspec.go err113 (→ `errNoPackagesAnalyzed` sentinel); d005_version.go gochecknoglobals ×2 (→ `//nolint:gochecknoglobals // read-only lookup table`, house pattern). Tests pinning old message substrings still pass (assertions chosen survive rewording).
10. **Docs:** README "Subcommand flag contract" section (user-facing contract); CONTRIBUTING.md "Subcommand Flag Contract" (contributor rules for adding flags); CHANGELOG `[Unreleased] → Fixed` entry (prose-only, no pkg.Symbol citations → symbols gate green).

---

## b) PARTIALLY DONE

1. **Config-file parity for `format`/`color`.** Implemented and behavior-verified manually (scorecard/doctor now read `cfg.Format`/`cfg.Color`, so `.cqrs-lint.json` flows in), and CLAIMED in README — but **no automated test pins the config-file → subcommand flow**. If cmdguard's config-loader semantics change, the README claim silently becomes a doc lie. Gap: one e2e test with a temp `.cqrs-lint.json`. Effort: S.
2. **doctor colorization.** `--color` is inherited and accepted by doctor, but doctor's text renderers use plain `fmt.Fprintf` (no color anywhere) — an accepted-flag-that-does-nothing remains ON THIS ONE COMMAND. Principled (doctor never colorizes) but undocumented. Effort: S (document) or M (colorize headers).
3. **`version`/`explain`/`changelog` vs `--format`.** These single-render commands silently ignore the inherited flag (decided: shared vocabulary, single rendering — ignoring is principled). Not pinned by test, not stated in their `--help`. Effort: S.

---

## c) NOT STARTED (noticed this session, deliberately out of scope)

1. **doctor `--fix`/`--dry-run` semantic collision.** Root `--fix` = apply findings autofix; doctor-local `--fix` = remove stale suppressions. Same flag name, two meanings, distinguished only by command context. I preserved it (documented nowhere!) rather than resolving. Product decision required (see g1).
2. **Supported-format lists duplicated 2–3×.** e.g. scorecard's `{text,json,markdown,sarif}` exists in the `validateFormatFlag` call AND the `WithShort` string. Adding a format means editing multiple sites → drift risk I introduced. Fix: per-command `[]string` constant feeding both.
3. **Sibling `cmd/*` tools (cqrs-bench, cqrs-gen, cqrs-upgrade, doc-check) unaudited** for the same persistent-flag/format-vocabulary diseases. Same cmdguard patterns likely present.
4. **`completion`/`help` subcommand flag surface untested** (cobra-generated; inherits persistent flags; never probed).
5. **Binary-level smoke probes.** `scripts/smoke-probes.txt` only probes `cqrs-lint version`. The new consistency contract has no binary-level e2e (tests are in-process `ExecuteWithArgs`).
6. **cmdguard upstream improvements** (see f, items in group D): no `WithSharedFlags(subset)` helper; `local:"true"` semantics are documented only in cmdguard source, not surfaced to flag authors at the use site.
7. **`explain` output teaching the tri-state model.** The session's very first question (why some "bools" are on/off strings — `TracingKind`/`MonetaryKind` carry an `unknown` = "defer to heuristics" state, `feature_kinds.go:125`) shows `explain`'s features table doesn't explain WHY `tracing` is on/off but `server` is true/false. One explanatory row/footnote would close a real user-confusion loop.

---

## d) TOTALLY FUCKED UP (session mistakes — radical honesty)

1. **Multiedit placed a package-level sentinel INSIDE a function body** (loader.go `errSilentEmptyLoad`). Legal-ish Go but semantically wrong (per-call instance, unusable for `errors.Is`). Caught by immediate re-view, fixed before build. Root cause: replacement `old_string` started mid-function and I prepended a `var` without thinking about scope. Severity: zero (caught), but it was a blind edit on a file I'd only partially viewed.
2. **`renderRules` first draft referenced a nonexistent variable** (`colorSetting`) — compile error, caught by immediate build. Root cause: wrote the refactor against an imagined signature instead of re-deriving it.
3. **Sloppy shell probes produced false evidence TWICE.** Final probe matrix printed `rc=0` for all error cases (`rc=$?` after `$(cmd | head)` captures `head`'s status); an earlier `PIPESTATUS[0]` attempt also misfired in mvdan/sh. I verbally disclosed the artifact, but a reviewer reading only the transcript table would conclude the error paths return 0. Root cause: pipe/status interaction carelessness; fix pattern: capture rc before piping, or `set -o pipefail`.
4. **`rulesFormat` returned `""` instead of `"text"`** for the zero value — caught by my own new test (test-after-impl, but the test did its job). One wasted cycle.
5. **Design flip-flop on empty-format semantics.** First version rejected `""` → 4 existing tests failed (`AppConfig{}` zero-value callers) → reversed to "empty = default text". A minute of thinking about programmatic callers upfront would have avoided the cycle.
6. **False alarm on `version --help` "missing" `--format`.** Misread my own awk output (`$1` on `-o --format` lines yields `-o`). Wasted a probe cycle + investigation before re-checking with plain grep.
7. **PRE-EXISTING, noticed, not mine:** master's `cmd/cqrs-lint` module carried 5 lint findings (landed via auto-committed work from the recent issue-#42 fix) — i.e. **the auto-commit daemon ships changes through the tree without running module lint**, and nothing failed. Process hole, not fixed by this session (fixing findings ≠ fixing the hole). See e1.

---

## e) WHAT WE SHOULD IMPROVE (process/design, session-derived)

1. **Auto-commit daemon bypasses gates.** The 5 pre-existing lint findings prove uncommitted work can reach master-shaped state with lint never run. Suggested: daemon pre-commit hook that runs the touched module's lint (or a `#verify-fast` subset) before absorbing; or nightly lint gate wired to alert.
2. **Go to the canonical gate definition FIRST.** I initially theorized about golines findings using my system `golangci-lint` before reading `flake.nix`'s `lint` app (pinned binary, `lintModules = testModules`). Cost: ~15 minutes of confusion about whether master was lint-clean. Rule: when gate authority is unclear, read the flake app before reasoning.
3. **Per-command supported-format lists should be single-sourced** (constant → validation + help + explain). Three-way duplication is a split-brain seed.
4. **Claims shipped in docs need pinning tests in the same session.** README's config-file-parity claim (b1) is currently trust-me prose. House rule candidate: every new behavioral claim in docs ships with a test name in the same commit.
5. **Probe/shell evidence discipline.** Standard snippet for CLI probes: `out=$(cmd 2>&1); rc=$?; echo "$out" | head` — rc captured before any pipe. (My d3 mistake class.)
6. **Test-before-impl for small pure resolvers** (`rulesFormat`, `validateFormatFlag`) would have caught d4/d5 in seconds instead of a cycle each.

---

## f) Next tasks (ranked; feeds docs-health HARVEST — see g3)

**Group A — close this session's loops**

| # | Task                                                                                                                                                           | Impact | Effort | Cat           |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1 | Add e2e test: temp `.cqrs-lint.json` with `"format": "json"` → `scorecard`/`doctor` emit JSON (pins README claim)                                              | High   | S      | Quality       |
| 2 | Single-source per-command supported-format lists (constant feeds validate + WithShort)                                                                         | High   | S      | Cleanup       |
| 3 | Document or resolve doctor `--fix`/`--dry-run` name collision with root autofix flags (needs g1 answer)                                                        | High   | S/M    | Bug           |
| 4 | Decide + pin behavior of `--format` on single-render commands (version/explain/changelog): reject loudly vs documented ignore (needs g2 answer)                | Medium | S      | Feature       |
| 5 | Extend `scripts/smoke-probes.txt` with binary-level consistency probes (rules --format json, scorecard --format csv → rc≠0, version --fix → rc≠0, init --path) | Medium | S      | Quality       |
| 6 | Test the `completion` + `help [command]` flag surface under the new scoping                                                                                    | Medium | S      | Quality       |
| 7 | Colorize doctor's section headers (or drop `--color` from its documented surface)                                                                              | Low    | M      | Feature       |
| 8 | Add `unknown`-sentinel explanation row to `explain` features table (why tracing=on/off but server=true/false)                                                  | Medium | S      | Documentation |

**Group B — cqrs-lint CLI polish**

| #  | Task                                                                                                                                                     | Impact | Effort | Cat           |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 9  | `doctor --format json` schema doc (field table) in advanced.md                                                                                           | Medium | S      | Documentation |
| 10 | Audit remaining accepted-but-ignored flag/command pairs (e.g. `--min-severity` on rules? `--typed-info` on version) and either consume or scope them out | Medium | M      | Quality       |
| 11 | `cqrs-lint init` should print the resolved target path with `--preset` presets summary                                                                   | Low    | S      | Feature       |
| 12 | Consider `--scorecard-threshold` availability on the root flag path (currently subcommand-only, help says so)                                            | Low    | S      | Feature       |
| 13 | Add `changelog` fallback message quality check (currently silent `git log -20` fallback if tag missing)                                                  | Low    | S      | Quality       |

**Group C — siblings & repo**

| #  | Task                                                                                                                                           | Impact | Effort | Cat           |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 14 | Run the same subcommand-consistency audit on `cmd/cqrs-bench`, `cmd/cqrs-gen`, `cmd/cqrs-upgrade`, `cmd/doc-check`                             | High   | M      | Quality       |
| 15 | Extract the audit method into a reusable checklist (probe matrix: format vocabulary, flag scoping, exit codes, config-file parity, help drift) | Medium | S      | Documentation |
| 16 | Archive finished `docs/status/` reports (gate warning: 21 live > 10 — move to `docs/status/archived/`, index rest)                             | Medium | S      | Cleanup       |
| 17 | Root-cause the daemon-bypasses-lint hole (e1) — pre-commit gate or nightly alert                                                               | High   | M      | Quality       |

**Group D — cmdguard upstream (external repo)**

| #  | Task                                                                                                                          | Impact | Effort | Cat           |
| -- | ----------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 18 | Proposal: `WithSharedFlagSubset(cmd, "format", "color", …)` helper so subcommands declare which persistent flags they consume | Medium | M      | Feature       |
| 19 | Proposal: derive `WithShort` format lists from a validator declaration (kills list duplication at the framework level)        | Medium | M      | Feature       |
| 20 | Docs: cmdguard README section on `local:"true"` scoping semantics (currently source-only knowledge)                           | Medium | S      | Documentation |
| 21 | Proposal: lint-time detector for "persistent flag never read by any subcommand" in host CLIs                                  | Low    | L      | Feature       |

**Group E — testing depth**

| #  | Task                                                                                                                             | Impact | Effort | Cat     |
| -- | -------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------- |
| 22 | Golden test: root `--help` hand-written usage block vs cobra COMMANDS section (kills help drift class permanently)               | High   | S      | Quality |
| 23 | Property-ish test: every `validateFormatFlag` supported value actually renders (no supported-value → runtime panic/fallback gap) | Medium | M      | Quality |
| 24 | Add `init --preset <each>` e2e covering all 6 presets generate valid JSONC                                                       | Medium | S      | Quality |
| 25 | Test config-file + CLI-flag precedence for `format` on root (flag should win over file)                                          | Medium | S      | Quality |

**Group F — residual observations from the session (smaller)**

| #  | Task                                                                                                         | Impact | Effort | Cat           |
| -- | ------------------------------------------------------------------------------------------------------------ | ------ | ------ | ------------- |
| 26 | `rules --json` + `--markdown` both set → markdown wins silently; define + test precedence (or error)         | Low    | S      | Quality       |
| 27 | Search docs for other `--format`-family claims that lack tests (doc-trust sweep, cqrs-lint scope only)       | Medium | M      | Documentation |
| 28 | CONTRIBUTING: add "probe snippet" section with the rc-safe pattern from e5                                   | Low    | S      | Documentation |
| 29 | Consider exit-code doc (which errors → which rc) in README CI section                                        | Low    | S      | Documentation |
| 30 | Check `--typed-info` surface: keep persistent (correct today) but document why in CONTRIBUTING flag contract | Low    | S      | Documentation |

---

## g) Questions I cannot answer myself

1. **doctor's `--fix` collision (blocks f3):** root `--fix` applies findings autofix; doctor's local `--fix` removes stale suppressions. Rename doctor's to something like `--prune-suppressions` (breaking change for existing doctor users) — or keep the names and just document the collision? This is a product/compat call, not derivable from code.
2. **Single-render commands and `--format` (blocks f4):** `version`/`explain`/`changelog` currently ignore the inherited `--format` (my call: shared vocabulary, one rendering each). Should they instead hard-reject non-text values (loud, but then a config with `"format": "json"` breaks `cqrs-lint version`)? I cannot derive the intended UX philosophy from the repo alone.
3. **Harvest (per status-report skill):** should I run docs-health HARVEST now and pull section (f) into `TODO_LIST.md`/`ROADMAP.md`, or leave this as a snapshot until you say go?

---

_Snapshot only — point-in-time report. Waiting for instructions._
