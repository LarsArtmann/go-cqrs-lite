# Status: erraudit full-repo sweep + `--max` audit (session 2026-10-27 ~18:55–19:15 CET)

**Scope:** this session only, per operator instruction — no unrelated project research.
**Trigger:** operator ran `erraudit lint ./... --max` at the repo root; the tool scanned only the root module and reported `98 nested go.mod file(s) … NOT analyzed` (exit 0). Task: complete the analysis across the whole multi-module repo and triage everything it surfaces.
**Binary under test:** nix-installed `erraudit` at `/run/current-system/sw/bin/erraudit`, version string `61a744c` (git hash, NOT a tag — see b1 and g-Q1). CI pins v0.7.0 via the vendored tarball (`tools/erraudit/`).

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **Gate-config sweep across every module — 0 findings.** Ran `scripts/erraudit-sweep.sh` (the CI `error-audit` job's local mirror: `lint ./... --enforce-go-error-family --type-aware --disable-extensions`, per-module roots, `GOWORK=off`, go-env.sh chain sourced) over all go.mod roots. Result: `modules=102 failing=0`; every per-module log 0 bytes. Zero baseline (2026-09-13 @22 modules, 2026-10-16 @104) re-confirmed 2026-10-27 @102. | `build/erraudit-sweep/*.log` (102 files, all empty); sweep stdout |
| A2 | **`--max` strictness audit across every module — delta fully classified.** Replicated the operator's exact flags (`lint ./... --max` = `--type-aware --no-suppress --explain --disable-extensions` + every `--enforce-*`) per module. Result: 67/102 modules exit rc=2, **801 findings, 100% of class `--enforce-generic-return`** ("returns generic error interface"). Zero `errors.As` findings, zero `errors.Is` findings, zero empty-coded go-error-family constructors, zero deferred-close findings, zero oops findings, zero context_loss. 35/102 modules clean even at `--max`. | `build/erraudit-max-sweep/*.log` (102 files); aggregation grep: 801/801 generic-return |
| A3 | **Honest-zero proof.** `--max` includes `--no-suppress`; it surfaced nothing — therefore the tree contains **zero `//nolint:legacyerrors` suppressions** and the gate's zero baseline hides nothing. | A2 logs (no suppressed-finding rows anywhere) |
| A4 | **Triage verdict: no code changes warranted.** The entire `--max` delta is `--enforce-generic-return`, a check the binary's own help marks *off by default — "returning error is standard Go practice"*. Acting on it would mean a bespoke error type per function across the library's entire `error`-returning API surface (every Store/Journal/Engine method) — a breaking-change catastrophe with zero safety gain, and exactly the drive-to-zero cargo-cult the go-error-modernization skill exists to prevent. Read-only audit ⇒ no build/test cycle required. | tools/erraudit README note (A5); erraudit `lint --help` text |
| A5 | **Documented the gotcha + re-verification** in `tools/erraudit/README.md`: 2026-10-27 re-verification line (102 modules, build 61a744c), the `--max` ≠ gate-config explanation, expected ~800 generic-return findings, honest-zero proof. Uncommitted — auto-commit daemon absorbs. | `tools/erraudit/README.md:37-53` |
| A6 | **Stale-artifact cleanup.** Trashed 3 orphaned sweep logs from modules removed since the 2026-10-23 run (`idempotency_kvstore.log`, `transport_grpc.log`, `transport_http.log`); `build/erraudit-sweep/` now maps 1:1 to the 102 live go.mod roots. | `trash` + recount: 102/102 |
| A7 | **Coverage proof.** Both sweeps enumerate every go.mod root themselves, so nested modules (e.g. `event/v4/eventtest`, which erraudit's parent-run hint names) each got their own analysis — no module-boundary gaps like the operator's root-only run. | log inventory: 102 entries == `find . -name go.mod … \| wc -l` = 102 |

## b) PARTIALLY DONE

| # | What | Works | Open | Blocker | Effort |
|---|------|-------|------|---------|--------|
| B1 | **Binary provenance vs CI pin** | Established the installed binary is a nix system-profile build `61a744c`, not the CI tarball; grep of this repo's `flake.nix` shows NO erraudit input (it comes from the OS config, not the repo). | Never built `erraudit-v0.7.0-src.tgz` and cross-checked that `61a744c` and v0.7.0 agree on this tree. Until then, today's local clean is *strong evidence*, not *proof*, of CI parity (CI runs the gate itself on every push, so drift would surface there). | None — sequencing choice, not a hard blocker | S |
| B2 | **Module-count drift (104 → 102) audit** | Noticed and localized: three modules present on 2026-10-23 are gone today (`transport/grpc`, `transport/http`, `idempotency/kvstore` — inferred from the stale logs in A6). | Did NOT verify each removal's documentation trail (CHANGELOG, FEATURES.md, module-map, AGENTS.md Tier-4 list still naming "transports", and ADR-0158's "deletion at v5.0.0" vs kvstore apparently already gone). Operator said report-then-wait, so this is flagged, not researched. | None | M |
| B3 | **TODO_LIST harvest** | Section (f) below is written, specific, and harvest-ready. | Not harvested into `TODO_LIST.md` — operator demanded WAIT after the report; HARVEST (docs-health) is the first proposed action. | Operator instruction | S |

## c) NOT STARTED

| # | What | Why not started | Priority |
|---|------|-----------------|----------|
| C1 | `--max` audit lane in `scripts/erraudit-sweep.sh` (separate report dir + rc legend) | Awaiting owner answer (g-Q3 shape); one-off inline loop sufficed this session | Medium |
| C2 | nix app wrapper `nix run .#erraudit-sweep` + AGENTS.md Quick-Reference row | Discoverability improvement; not required to answer the operator's question | Medium |
| C3 | GOEXPERIMENT=jsonv2 audit of the sweep script | Noticed contradiction (script comment says "REQUIRED (encoding/json/v2)"; AGENTS.md gotcha #10 says the env is a post-graduation no-op that was swept REMOVED from scripts) but never A/B-tested dropping it | Low |
| C4 | `--self-test` for the sweep script (house style: gating scripts carry fault-injecting self-tests, cf. calibration-gate.sh) | Script predates the convention; never required | Low |
| C5 | Durable policy record for rejecting `--enforce-generic-return` (ADR or standing note beyond the README paragraph) | Awaiting owner answer (g-Q2) | Medium |

## d) TOTALLY FUCKED UP

Nothing is broken. Radical honesty, nearest candidates — all LOW severity:

1. **Sweep script accumulates stale logs forever.** `scripts/erraudit-sweep.sh` does `mkdir -p` but never prunes; removed modules leave orphaned logs that mislead future readers (today's `catalog.log`-from-Oct-23 confusion class). Fixed ad hoc (A6), root cause remains. Workaround: manual cleanup. No data loss.
2. **Local-vs-CI binary equivalence is unproven** (B1). If `61a744c` diverges from v0.7.0, today's local "clean" could theoretically differ from CI's verdict. Mitigation: the CI `error-audit` job runs the exact pinned binary on every push.
3. **go.mod-count split-brain in docs.** AGENTS.md Quick Reference says "101 `go.mod` files (incl. root)"; module-map says "102 incl. root"; today's `find` says 102. At least one prose count is a lie right now, and per AGENTS.md's own note the canonical-facts gate cannot see backticked counts, so no gate catches it. Needs a hand fix (f-4). Severity: doc-only, but it is exactly the drift class the repo keeps ratcheting against.

## e) WHAT WE SHOULD IMPROVE

1. **Sweep-script durability** (pruning or timestamped report dirs) — today's manual `trash` doesn't survive the next module removal. Impact: recurring confusion; fix: ~5-line script change.
2. **Make `--max` semantics impossible to re-stumble over.** The README note helps, but the durable fix is a standing owner decision (g-Q2) + optionally a gotchas one-liner in `docs/agents/gotchas-tooling-build.md` ("running erraudit at repo root scans the root module only; use scripts/erraudit-sweep.sh").
3. **erraudit version strings should carry the tag.** `erraudit --version` → `61a744c` makes provenance audits (like B1) guesswork. erraudit is a first-party project: print `vX.Y.Z (hash)`. Impact: every future bump/audit saves a research detour.
4. **Discoverability:** every other gate has a `nix run .#check-*` app and a Quick-Reference row; the erraudit sweep has neither (C2). Impact: the operator's exact confusion this session.
5. **My own sequencing:** I could have built the v0.7.0 tarball binary concurrently with the first sweep and verified provenance BEFORE writing "re-verified" into the README (I hedged the wording to "nix erraudit build 61a744c", but provenance-first ordering would have removed the hedge entirely).

## f) NEXT TASKS (session-derived, harvest-ready — 28 items, impact-ranked)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | HARVEST this section into `TODO_LIST.md` / `ROADMAP.md` (docs-health HARVEST mode) | High | S | Cleanup |
| 2 | Build erraudit v0.7.0 from `tools/erraudit/erraudit-v0.7.0-src.tgz`; re-run the gate sweep with THAT binary; confirm 102/102 clean | High | S | Verification |
| 3 | Determine `61a744c`'s relationship to v0.7.0 (ahead/behind/equal) from the erraudit checkout | Medium | S | Verification |
| 4 | Hand-fix the go.mod count split-brain (AGENTS.md quick-ref 101 vs module-map 102 vs actual 102); remember the canonical-facts gate can't see backticked counts | Medium | S | Documentation |
| 5 | Verify documentation trails for the three removed modules (`transport/grpc`, `transport/http`, `idempotency/kvstore`): CHANGELOG, FEATURES.md, module-map, AGENTS.md Tier-4 "transports" mention, ADR-0158's "deletion at v5.0.0" claim vs kvstore already gone | High | M | Verification |
| 6 | Add stale-log pruning (or timestamped report dirs) to `scripts/erraudit-sweep.sh` | Medium | S | Tooling |
| 7 | Record the `--enforce-generic-return` rejection decision durably (pending g-Q2) | Medium | S | Policy |
| 8 | Add optional `--max` audit lane to the sweep script (pending g-Q3) | Medium | S | Tooling |
| 9 | Wrap the sweep as `nix run .#erraudit-sweep` | Medium | S | Tooling |
| 10 | Add an erraudit row to AGENTS.md Quick Reference (sweep script + tools/erraudit/README.md) | Medium | S | Documentation |
| 11 | A/B test `GOEXPERIMENT=jsonv2` in the sweep script (drop it for one module, compare) then fix the stale "REQUIRED" comment per gotcha #10 | Low | S | Cleanup |
| 12 | Add `--self-test` to the sweep script (calibration-gate.sh pattern) | Low | M | Tooling |
| 13 | Machine-readable sweep summary (`--report`/`--json`, matching other gates) | Low | M | Tooling |
| 14 | Honor `ERRAUDIT_BIN` env in the sweep script (generate-error-taxonomy.sh consistency) instead of positional-arg-only | Low | S | Tooling |
| 15 | Make erraudit `--version` print `vX.Y.Z (hash)` (first-party upstream fix) | Low | S | Feature |
| 16 | One-line gotcha in `docs/agents/gotchas-tooling-build.md`: root-only erraudit runs scan only the root module; use the sweep script | Low | S | Documentation |
| 17 | Tabulate the zero-baseline history in tools/erraudit/README.md (three inline dates now) | Low | S | Documentation |
| 18 | Per-module duration output in the sweep script (spot slow modules) | Low | S | Tooling |
| 19 | Consider parallelizing the sweep with a load gate (7 min serial today; GT_P-style) | Low | M | Tooling |
| 20 | Verify the root module really contains 1 Go file and that this is intentional (erraudit's "scanned 1 file(s)" hint) | Low | S | Verification |
| 21 | Confirm `build/erraudit-max-sweep/` is covered by the build/ gitignore for future runs (verified clean post-hoc this session) | Low | S | Cleanup |
| 22 | Decide whether local `#verify`/verify-chain should mirror CI's error-audit leg (pending g-Q3) | Medium | S | Policy |
| 23 | CHANGELOG `[Unreleased]` row for the README gotcha note if judged user-visible (check-changelog-symbols.sh gates `pkg.Symbol` citations) | Low | S | Documentation |
| 24 | Consider a catalog/docserver cross-ref: erraudit findings vs cqrs-lint E-rules overlap note (one paragraph) | Low | M | Documentation |
| 25 | Add a sweep-script header note: rc legend + "logs are appended across runs until pruned" (interim until #6) | Low | S | Documentation |
| 26 | When the next erraudit bump happens: follow tools/erraudit/README.md refresh flow AND re-verify with the built binary (this session's flow exercised; nothing broken) | Low | S | Process |
| 27 | Consider annotating `--max` noise classes in the binary itself (first-party: a `--enforce-generic-return=exclude-exported` knob would kill 90% of the noise) | Low | M | Feature |
| 28 | Re-check ADR-0158 wording if kvstore was genuinely deleted early ("Nothing is removed in v4" vs observed absence) — fold into #5 or split if confirmed | Medium | S | Documentation |

(Most items beyond #12 are ROADMAP fuel, not commitments — HARVEST should apply routing rigor.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Pin policy:** Should the repo's erraudit gate pin (vendored tarball v0.7.0 + CI checksum + filename in ci.yml) track the desktop nix build (`61a744c`, apparently newer than the tag), or stay at v0.7.0 until a deliberate bump? *What I tried:* readlink on the binary, grepping this repo's flake.nix (no erraudit input — it ships via the OS config), reading the tools/erraudit provenance table. I cannot resolve a bare git hash to its tag relationship without the erraudit checkout, which is outside this session's scope.
2. **`--enforce-generic-return` policy:** Permanently rejected for this repo (801 findings = by-design noise; the go-error-family 6-family taxonomy is the chosen error architecture), or do you want a targeted subset triaged someday (e.g. unexported helpers only)? This decides between recording a standing rejection note vs opening a triage backlog.
3. **Local gate parity:** Should the local verify chain mirror CI's error-audit job (adding the ~7-minute sweep to `#verify`/verify-chain.sh as a nix check app), or is per-push CI coverage sufficient with local sweeps opt-in?

---

## Self-critique (what did I forget / could do better)

- **Forgot (biggest):** proving binary equivalence before writing "re-verified" (B1). The README note is honestly worded, but building the tarball binary was a parallelizable S task I sequenced away.
- **Forgot (small):** A/B-testing the GOEXPERIMENT=jsonv2 contradiction I myself noticed (C3); post-hoc rather than prior verification that `build/` is gitignored.
- **Could do better:** provenance-first ordering; the README note is prose-heavy where a table would scan faster; HARVEST ideally happens before session end rather than after an explicit wait.
- **Still improvable:** everything in (e) — the sweep script's durability, gate discoverability, and the upstream version-string fix are the three with recurring payoff.

*Report is a point-in-time snapshot; section (f) is the HARVEST input. Awaiting instructions.*
