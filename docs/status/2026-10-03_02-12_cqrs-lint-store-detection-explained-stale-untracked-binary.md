# Status Report: cqrs-lint `store` Detection Explained; Stale Untracked Binary Found

- **Date:** 2026-10-03 02:12
- **Scope:** This session only — answering "how does the `store` config key in `cqrs-lint explain` work for `system/` and `metaengine/`?" plus everything noticed along the way. No code was modified; all work was read-only investigation + one throwaway build in `/tmp`.
- **Working tree at report time:** clean (auto-commit daemon owns any absorption).
- **Prior art:** `docs/reviews/2026-09-19_16-22_dogfooding-brutal-self-review.html` is the latest self-review in the series; not re-read this session (user scoped this report to session-local findings).

---

## a) FULLY DONE

1. **The question is fully answered** (evidence: the assistant reply earlier this session; all claims re-verified against source):
   - `cqrs-lint explain` renders the `store` row statically from `AllStoreKinds()` — `cmd/cqrs-lint/pkg/analyzer/feature_kinds.go:6-22`; the values column is never derived from the scanned project.
   - Actual store resolution is import-based, per nearest `go.mod`, with test files excluded twice (`packages.Load` with `Tests: false` at `pkg/analyzer/loader.go:24`, plus the `gf.IsTest` skip in Pass 1b at `feature_detect.go:96`). Precedence: `stack/<name>` preset → `metaengine/<x>engine` import → `storage/` → `custom`, `modernc.org/sqlite`/`mattn` driver fallback → `sqlite`, else `none` (`feature_detect.go:182-269`).
2. **Empirical verification with a freshly built go1.27 binary** (`/tmp/cqrs-lint-fresh`, built after sourcing `scripts/go-env.sh` per AGENTS.md):

   | Module | store | metaengine | pushdown |
   |---|---|---|---|
   | `system/` | none | true | true |
   | `metaengine/` (core) | none | true | false |
   | `metaengine/sqliteengine` | sqlite | true | false |
   | `metaengine/duckdbengine` | none | true | false |
   | `metaengine/pebbleengine` | none | true | false |

3. **Root-caused why each module resolves the way it does:**
   - `system/` imports only the metaengine *core* (`metaengine/v4` → engine `""` → no store hint, `feature_detect.go:279-306`); engines are consumer-registered (database/sql pattern); its `storage/memory` import is test-only (`lifecycle_with_test.go`). `none` is semantically honest — system is deployment-agnostic by design (ADR-0123).
   - `metaengine/` core: its go.mod *direct* deps on `sqliteengine` + `modernc.org/sqlite` are test-only (cross-engine ADT tests); `engine.go`'s mentions are comments (engine.go:640, engine.go:693).
   - `sqliteengine` self-detects `sqlite` only via the modernc driver-import fallback — a module never imports itself, so the engine→store mapping never fires for engine modules themselves.
4. **Documented the override path:** `{"features":{"store":"…"}}` in `.cqrs-lint.json` beats detection (explicit config > preset > auto-detect, `feature_profile.go:89-135`).

## b) PARTIALLY DONE

1. **Stale-binary finding — flagged, not remediated.** An untracked, go1.26-built binary sits at `cmd/cqrs-lint/cqrs-lint` and silently mis-detects on this repo (reproduced: reported `metaengine: false` for `system/` under partial package load). What remains open: rebuild it in place, trash it, or replace with a flake app; and decide whether tool staleness should ever be silent. Effort: S.
2. **`pushdown` derivation — looked up, not explained.** Found the field (`MetaenginePushdown`, `feature_profile.go:47-54`, gates F022–F025) but never established *why* `system/` is `true` and `metaengine/` is `false` (which API adoption — e.g. `SortOnField` in system's query constructors — trips it). Thread abandoned when the user's question didn't need it. Effort: S.
3. **Engine self-detection gap — observed, not adjudicated.** `duckdbengine`/`pebbleengine` → `store: none` (no driver-import fallback exists for `go-duckdb`/pebble). Reported as a quirk; did not check whether an existing test pins this as *intended* (`feature_detect_permodule_test.go` covers consumer-side engine imports; engine-self cases unverified). Effort: S to verify, M to change.

## c) NOT STARTED

1. Any fix for the engine-module self-detection gap (driver fallbacks for go-duckdb/pebble/bbolt/badger/dgraph/turso/iroh/bigtable, or seeding detection with the module's own package path).
2. Doctor config-discovery semantics: running doctor on a submodule prints `CONFIG FILE: NOT FOUND` — no upward search, so the root `.cqrs-lint.json` (`preset: library`) does not apply to per-module runs. Neither investigated nor reported to the user (see d-4).
3. The empty `monetary:` line in doctor output (renders blank where `domain: unknown` prints its sentinel — possible rendering bug or intended omission).
4. Rebuilding/trashing the stale binary; adding build-Go-version visibility to `cqrs-lint version`/doctor.
5. Doctor evidence mode (print which import/call site set each feature).
6. Documenting the `pushdown` sub-key in `explain` (it renders in doctor but is absent from the `featureKeys` table, explain.go:304-377).
7. HARVEST of section (f) into `TODO_LIST.md` — deliberately deferred; user instructed "report and wait."

## d) TOTALLY FUCKED UP

1. **False claim in my final answer: "the committed binary."** `git ls-files cmd/cqrs-lint/cqrs-lint` returns nothing — the stale binary is an *untracked local build artifact*, not committed. The mis-detection finding itself stands (empirically reproduced), but I asserted repo state ("committed") without checking it. Root cause: inferred from directory presence instead of asking git. Lesson: verify tracked/committed status before characterizing anything as repo-owned.
2. **I nearly reported from a garbage profile.** The first doctor run used the stale go1.26 binary against the go1.27 repo; package loading failed per-file, and the output still printed a confident-looking `store: none / metaengine: false` feature profile with only a WARNING banner above it. I initially treated those numbers as data before catching the toolchain mismatch. Mitigation worked (rebuilt, re-ran, final numbers are clean), but the failure mode is real: partial-load output looks authoritative. The tool should degrade louder (see f-20).
3. **Left a 24 MB orphan in `/tmp`** (`/tmp/cqrs-lint-fresh`). Harmless, but unowned artifacts are how sessions rot.
4. **My answer omitted a fact I had already observed that directly affects the modules asked about:** doctor on `system/` and `metaengine/` finds NO config file (no upward search), so my "overrides beat detection" advice is inert for exactly those modules unless the user places a config there or lints from the repo root. I saw the `NOT FOUND` line mid-session and dropped it from the final answer. The answer was correct but incomplete.

## e) WHAT WE SHOULD IMPROVE

1. **Assert repo state from git, not from `ls`.** "Committed", "tracked", "shipped" are claims with cheap verifiers (`git ls-files`, `git log`). Impact: prevented one false claim this session; generalizes to every session.
2. **Treat partial-load WARNING as total invalidation.** Any feature profile produced under a package-load warning should be discarded, not interpreted. The session recovered only because I happened to re-read the warning.
3. **When explaining a config key, always include its discovery/inheritance semantics.** For a 98-module monorepo, "does the submodule see the root config?" is half the answer.
4. **Batch module sweeps into one command from the start.** Three sequential doctor invocations (one auto-backgrounded) where one `for m in …` loop would have done. Saved ~2 minutes of wall clock.
5. **Make tool staleness loud and cheap to detect.** A binary built with an older Go toolchain silently corrupts analysis; embedding the build Go version into `version`/doctor output turns this class of rot self-announcing.

## f) Next tasks (25 — impact-ranked; deliberately not harvested into TODO_LIST.md pending user instruction)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Rebuild or trash the untracked go1.26 binary at `cmd/cqrs-lint/cqrs-lint`; prefer replacing with a flake app so it cannot go stale | High | S | Cleanup |
| 2 | `cqrs-lint version`/doctor: print the Go toolchain the binary was built with | Medium | S | Feature |
| 3 | Doctor: escalate partial-package-load from a WARNING banner to a visible per-profile marker (or `--strict-load` exit code) | High | S | Quality |
| 4 | Decide: should engine modules self-detect their store kind? If yes, add driver-import fallbacks (go-duckdb, pebble, bbolt, badger, dgraph-go, turso, iroh, bigtable) or seed detection with the module's own path | Medium | M | Feature |
| 5 | Add engine-module self-detection cases to `feature_detect_permodule_test.go` (all 11 engines), pinning today's behavior whatever it is decided to be | Medium | S | Quality |
| 6 | Doctor/lint config discovery: upward search for `.cqrs-lint.json` (golangci-style) OR keep local-only and document it in `explain` | Medium | M | Feature |
| 7 | `explain`: document config discovery semantics (which directory is searched; submodule runs see nothing) | Medium | S | Documentation |
| 8 | Investigate empty `monetary:` line in doctor output (blank vs `domain: unknown` sentinel) | Low | S | Bug |
| 9 | Add `pushdown` (MetaenginePushdown) as a documented feature key in `explain`'s featureKeys table | Low | S | Documentation |
| 10 | Determine + document which APIs set `MetaenginePushdown` (why system=true, metaengine=false) | Low | S | Documentation |
| 11 | Doctor `--evidence` flag: print the import path / call site that resolved each feature | Medium | M | Feature |
| 12 | RULES.md/cqrs-lint README: document the store-detection precedence chain with consumer examples | Medium | S | Documentation |
| 13 | Self-lint pin: doctor on `system/` must stay `store: none, metaengine: true` — guards against system accidentally importing an engine in prod code | High | S | Quality |
| 14 | Consider an architecture lint rule: `system/` (composition tier) must not import `metaengine/*engine` in non-test code | Medium | M | Feature |
| 15 | Verify `system/integration`'s own profile (imports duckdbengine → expect `store: duckdb`) and pin it | Low | S | Quality |
| 16 | Verify `DetectFeaturesPerModule` end-to-end for the `system/` tree (nested `system/integration` module appeared in doctor's package list — confirm per-module partitioning in lint runs, not just doctor listing) | Medium | S | Quality |
| 17 | Consider a `store: "pluggable"` kind (or explain-text note) for composition roots that defer engine choice — `none` currently reads as "no persistence" | Low | M | Feature |
| 18 | Confirm CI lints from repo root so the `preset: library` config actually covers system/metaengine despite their local NOT FOUND | Medium | S | Quality |
| 19 | `explain` CONFIG RESOLUTION ORDER section: state that a pinned value disables detection for that key entirely | Low | S | Documentation |
| 20 | Extend `library_self_lint_test.go` with a feature-profile snapshot for system/metaengine | Medium | S | Quality |
| 21 | Trash `/tmp/cqrs-lint-fresh` (or promote to the canonical local build location) | Low | S | Cleanup |
| 22 | Doctor: when a profile was resolved under partial load, print it with a `partial` tag instead of a clean table | Medium | S | Quality |
| 23 | Add a `TestMetaengineEngineFromImport_CoversShippedEngines` companion asserting every engine also appears in the engine→StoreKind switch *and* in any new self-detection mechanism | Medium | S | Quality |
| 24 | Re-check the 2026-09-19 dogfooding self-review for overlap with f-1..f-3 (tool staleness may already be a known theme) before harvesting | Low | S | Documentation |
| 25 | HARVEST this section into TODO_LIST.md/ROADMAP via docs-health once user confirms which items are wanted | High | S | Documentation |

## g) Questions (cannot be figured out from code alone — design intent/ownership)

1. **Engine self-detection intent:** *should* an engine module (e.g. `metaengine/duckdbengine`) report its own store kind, or is `store: none` for engine modules the intended semantics — engines ARE the backend rather than a consumer of one? I can implement either (driver fallbacks vs. path-seeding vs. leave-as-is + pinned test); the intent call is yours.
2. **Config inheritance:** should submodule doctor/lint runs inherit the root `.cqrs-lint.json` (upward search), or is per-module config absence intended for this monorepo? Both are defensible; the current behavior (local-only, silently) surprised me mid-session.
3. **The stale binary's fate:** rebuild it in place as a local convenience, trash it, or replace with a flake app (`nix run .#cqrs-lint`-style) so a stale toolchain build can never masquerade as current again?

---

*Point-in-time snapshot. Section (f) not yet harvested into TODO_LIST.md/ROADMAP.md — awaiting user instruction per docs-health HARVEST.*
