# Status Report — Release Train Executed, Composed GREEN Certified

**Date:** 2026-09-08 23:12 CEST · **Session:** Pareto plan execution (Waves 0–2 partial)
**Input:** `docs/planning/2026-09-08_17-45_SUPERB-pareto-execution-plan.md` (executed top-down)
**Result:** the 1%→51% release train shipped in full; local `#verify` composed GREEN (EXIT=0); 6 real defects found and fixed along the way.

---

## a) Current State of the Project

- **The module proxy now serves the entire 2026-09-06 → 09-08 surface.** 60 tags cut, pushed, and audited this session (zero unpushed remain): the 54-module wave (`metaengine/v4.13.0`, `system/v4.7.0` matview, `storage/v4.9.0`, `stack/v4.4.0` + presets, `cmd/cqrs-lint/v4.10.0`, `benchkit/v4.5.0`, `scheduling/sqlstore/v4.0.0` first release, engines, `event/v4.11.0`/`command/v4.10.0` error-code renames, …), the iroh trio (P02), and `stack/sqlite/v4.3.1` (P03). 59 GitHub Releases created with CHANGELOG-accurate bodies. Both previously-broken tags are superseded. The four examples got their FIRST proxy-visible tags (v0.1.0/v0.2.0) — their old v4.x tags were invisible (suffix-less module paths).
- **`nix run .#verify` is GREEN locally** (certified 22:33, EXIT=0): build + vet + test + race + lint 77/77 modules clean + architecture/dep budgets + duplication (0 new clones) + coverage + templ + doc-check (1253 refs) + doc-assertions + CHANGELOG gates. First composed GREEN of the 09-06→09-08 surface (P06 done).
- **master is pushed and in sync** (`8aa26324c`). Working tree clean (daemon absorbs).
- **CI (GitHub) remains red — analyzed, mostly gated.** Last processed run (intermediate commit 20:33) failed; the final commit has NO run started ~35 min after push (runner-minutes/billing suspected — the user-gated class). Failure classes on the intermediate run: (1) FlakeHub-auth nix legs (Nix Flake Check, Dgraph, Ephemeral PG, Redis, gosec, CGo, Module Isolation, Verify-fast) — billing/creds-gated; (2) File Size 350 (P19 backlog, unchanged); (3) go.work sync + WASM — must be re-triaged on a FRESH run of the final commit (locally both pass). The per-tag **Release workflow is fixed** (was permanently red on every tag since 08-29: it built ALL modules including `cmd/cqrs-lint`, which needs the PRIVATE `go-finding` repo; now scoped to the tagged module, cqrs-lint excluded with rationale).

## b) What Got Done (plan coverage)

**Wave 0 — release train (COMPLETE):**
- **P02** irohengine/v4.2.0 + loopback/v4.0.2 + quic/v4.2.0 (quic's local replace dropped as redundant; both standalone green).
- **P03** stack/sqlite/v4.3.1: reproduced the fresh-consumer break exactly (`undefined: storage.SQLiteSetSynchronous` via the v4.3.0 pseudo-pin), tagged the fix, verified a clean-dir consumer `go get`+build.
- **P01** the 58-module wave, cut→push→next with the four hard mechanics honored: the tag script's standalone-build gate aborted 6 engines needing new metaengine symbols → pins pre-bumped → re-tagged; 2 engine tags (bbolt/dgraph) were rescued from a missed push by the end-of-wave unpushed-tag audit.
- **P05** closure: `pin-sweep.sh` bumped 63 modules + refreshed cqrs-lint goldens (`--check` green); workspace sync green; releases created; CI triaged into gated classes; release.yml fixed.
- **P06** six verify rounds → GREEN (see d).

**Wave 1 — trust infrastructure (5 of 6 COMPLETE):**
- **P08** json/v2 determinism: SARIF `run.properties` map → fixed-order struct + 50-render byte-compare pin; `json.Deterministic(true)` on all consumer-visible marshal sites; catalog already deterministic.
- **P12** alias-blindness dead: scanCallExpr + D018/D019 + performance codec heuristic now use `IsQualifierFor`; 3 completeness meta-tests added — one caught a real preset-help-text drift on first run.
- **P11 core** F091 Tier 2: `--typed-info` flag (auto/on/off, typo-warn); **F090(b)** typed attribution of dot-imported removed symbols; **C008 usage-confirmation** (ambient-only weak fields need local evidence); committed `testdata/typedfixture` (replace-based, excluded from api-stability/layers meta-tests) making the typed tier CI-testable. *Remaining: C035/C013 payload-shape confirmation.*
- **P09 essentials** matview safety: Doctor section pinned 4 ways (content/none/grouped-WARN/no-false-WARN), `matViewDDL` exact-DDL golden ×6 + escape guard, multi-tx grouped-SUM exactness pin (honest about the scale envelope).
- **P10** DSN audit: **real leak found+fixed** (`auth_token`/`AUTH_TOKEN` escaped redaction); adversarial per-shape tests + preserves-non-secrets guard; pg/mysql audited clean (no DSN echo). *Strict-vs-lenient typo guard stays user-gated.*
- *Not done: **P07** benchkit load-scaling (benchkit passed all final verify rounds, but the loadScaled pattern work remains).*

**Wave 2 (partial):** **P15+P16** AGENTS indexed-split (92 KB → 28 KB index + `docs/agents/gotchas-{tooling-build,module-management,language-footguns,testing}.md` + `gowork-modes.md` decision table + `module-map.md`; zero content loss proven by bullet/row counts) + check-app quick-ref rows. **P18** covered by the coverage gate inside verify. *P14, P17 (blocked on CI billing for new nix jobs), P19–P24 remain.*

**Wave 3:** P25–P27 untouched.

## c) Real Defects Found and Fixed This Session

1. `stack/sqlite/v4.3.0` shipped an unresolvable pseudo-pin → every fresh consumer build failed (repro'd, superseded by v4.3.1).
2. `tursoengine.redactDSN` leaked `auth_token`/`AUTH_TOKEN` params into error output (exact-spelling matcher; now token/key containment) — security class.
3. SARIF scorecard output was byte-nondeterministic (map iteration order under json/v2).
4. Release workflow permanently red since 08-29 (all-modules build hit private `go-finding`).
5. `TestProbeHandle_FailureCounter` check-then-assert race (counter increments before handler call) — structural bounded-wait cure.
6. decider singleflight test harness: `close of closed channel` panic + scheduling-dependent coalescing assertion (sync.Once + assertion scoped to the deterministic sibling test).
7. `check-changelog-symbols.sh` aborted on a legitimately-empty [Unreleased] (grep no-match under pipefail).
8. cqrs-lint working-tree version const desync after tagging (TestVersionMatchesLatestTag).
9. `cmd/cqrs-upgrade` was missing the CLI-tool lint exclusion class (19 findings); 2 dep-budget underallocations; preset help-text drift; templ codegen drift; benchkit dual-godoc; stale exhaustruct nolints (kv, scenario); snapshot wire tags needed intentional-contract nolints.
10. Example modules' v4 tags were proxy-invisible for years (suffix-less paths) — first valid tags cut.

## d) What Went Wrong / What Could Be Better

- **Six verify rounds (~2.5 h machine time).** Each run uncovered the next layer (version const → probe flake → 30+ lint findings → arch budgets → templ). A mid-session verify after P12 would have surfaced the lint debt an hour earlier. Lesson: after big doc/gate edits, run the LINT phase alone (`nix run .#lint`), not just module tests.
- **Matrix harness bug** (slash-path log redirect) produced 40 false FAILs; worse, I ran it CONCURRENTLY with the pin-sweep, violating the #verify-exclusivity rule — the re-run after the sweep was clean, but it burned a cycle and the go.sum race noise misled for a moment.
- **CHANGELOG restructure**: two botched [Unreleased] placements before the clean line-range rebuild; daemon mod-time collisions forced re-reads twice (recipes.md, CHANGELOG).
- **api-golden same-edit rule violated** for `TypedConfirmations` (verify caught; golden regen + meta-tests green).
- **C008 first draft** misused `EventTypesEmitted` (wrong key domain) and over-suppressed name-corroborated cases — the existing test suite caught both; the refined ambient-only rule is the defensible semantics.
- ec-fixture import block broken by an edit, caught immediately by lint.

## e) The One Thing

**Ship the composed GREEN: publish the 3-day surface and certify the gate.** Done — the proxy serves everything, and `#verify` exits 0 locally. The next session's "one thing": get a fresh CI run of `8aa26324c` processed (billing permitting) and drive the remaining CI legs to green-or-explicitly-gated.

## f) Uplift / Removed

**Uplifted:** the entire unpublished surface is consumer-available (60 tags, 59 releases); deterministic output class dead (SARIF/doctor/profiles); alias-blindness dead end-to-end; typed-confirmation tier live with a CI-testable fixture; matview safety mechanically pinned; a credential leak closed; AGENTS session-start cost down ~70% (indexed split); skill references current (recipes §2.24–2.31 incl. envelope-v2 rotation + pre-v5 snapshot decode; advanced §7 tooling; modules.md claiming/sqlstore row); TODO_LIST resolution markers honest.
**Removed/blocked:** nothing reverted. User-gated items untouched: PR #8257 permalink edit, turso upstream issue filing (A+B verified drafts exist), strict/lenient DSN ruling, CI billing/FlakeHub creds, 350-line policy ruling, doctor-JSON pre-merge semantics, tag-wave authorization for the NEXT wave.

## g) Next Session Should

1. **Re-triage CI on `8aa26324c`** the moment a run processes (expect FlakeHub class + File Size 350; investigate go.work-sync + WASM legs on the fresh state — both green locally).
2. **P07** benchkit load-scaling (`loadScaledCeiling`, checkpoint deadlines, closed-store race hunt) — the last W1 item.
3. **W2**: P14 (cqrs-upgrade `--strict`/`--json`/`--to`/workspace), P19 (350-line ruling + `typed_reader.go` split), P20 (ApplyLayout rule — the `--typed-info` gate it needs now exists), P21 (encryption wire golden), P23/P24 hygiene/docs batches.
4. **W3**: P25 (v5 sweep §4), P26 (T18 migration tail), P27 long tail.
5. **Next tag wave** (when authorized): cqrs-lint minor (P08/P11/P12 surface), tursoengine v4.1.1 (leak fix), metaengine patch (safety tests) — the CHANGELOG `[Unreleased]` section is the manifest; consumers on v4.10.0/v4.1.0 should see these promptly.
6. Harvest this session via docs-health; update TODO_LIST remainder rows.
