# Plan V3 Trust-Floor Continuation — T36–T39, T41 + Lint-Gate Repair

**Date:** 2026-08-29, ~15:30–17:30
**Session:** second execution session (continuation of the 14-50 report)
**Baseline:** `e4a0ba837` (my prior status report) → HEAD `<sha-after-verify>`
**Concurrent session:** ACTIVE throughout — a parallel session triaged the TODO list into its own plan (`docs/planning/2026-08-29_15-20_todo-full-execution-plan.html`, now tracked) and executed Tier-1 bug fixes in `cmd/cqrs-lint` (c041/c042). Coordination was done via TODO_LIST claim markers (37f2b3cff) and non-overlapping task selection.

## a) What was executed (7 commits + 1 external-repo commit)

| Commit | What |
|---|---|
| `436a9c9cb` | **Lint-gate repair** — the Aug 6 tombstone SA1019 exclusions were inert (regexes demanded `event.X` right after `SA1019: ` but staticcheck prints `event/v4.X`; they predate the deprecation markers, so they never mattered until the parallel session added them). Fixed the patterns (`.*event/v4\.`), verified against real message text. Also: pg/mysql/dgraph/duckdb engines dropped their provably-dead `NsPerRead` (all four `ReadCosts` fields are set — the fallback chain never consults the scalar); badger/bbolt/pebble keep the scalar under a NEW narrow exclusion until per-pattern calibration data exists (see TODO entry); 7 stale `//nolint` directives removed (sqlopt's was RESTORED after the full gate proved it still suppressed a live finding — nolintlint's "unused" was wrong for block-scoped directives). |
| `37f2b3cff` | TODO_LIST claims for docserver/bench-gate + engine-calibration backlog item. |
| `7cdac350e` | **T36 docserver**: per-request CSP nonce on every script tag (SPA bundles, inline bootstraps, copy-button, theme scripts — templ-components' `Nonce` prop + literal-tag nonce attributes), `Config.EnableCSP` header opt-in (default off = byte-identical responses; styles stay `unsafe-inline` because the embedded SPAs inject styles at runtime), CSPRNG failure degrades to the old rendering. `nix run .#check-templ` drift gate (`templ generate -check`, nixpkgs templ = the pinned v0.3.1020). docs-ui.css GET test, catalog README deps table, cId-value-semantics CHANGELOG note. |
| `3922e44c7` | **T36.6 caught a real exporter bug**: `@eventcatalog/core` ^4.6.3 REJECTS the EventCatalog exporter's output — message `producers`/`consumers` were `{id: ...}` objects; the schema wants plain string references, and even a bare service ID does not resolve (entry IDs are `<id>-<version>`). Exporter now emits versioned reference strings. Verified END-TO-END: `npx eventcatalog build` goes from hard failure to clean build, zero warnings, rendered site. The old structure-only integration test could never see this. |
| `688ce6f24` | **T37**: 9-case fixture tests pinning the bench gate (median over odd/even samples, strict >25% threshold, save-after-compare ordering, vanished/new informational) via `nix run .#check-bench-gate`; `nix run .#verify-module -- <path>` scoped build/vet/test/race/lint; actionlint in devShell + workflows lint-clean (fixed 2 shellcheck infos in ci.yml's doc-assertions step); baseline-regen runbook in BENCHMARKS.md. Threshold re-tune honestly deferred (needs live CI variance data). |
| `102b33b1a` | **T39**: first-class snapshot encryption — `snapshot.NewTransformedStore` (protect/restore state, routing metadata stays plaintext) + `encryption.SnapshotStateCodec`/`RotatingSnapshotStateCodec` (Envelope carries the key ID; `StaticKeyResolver` resolves retired keys; rotation needs no migration window — re-saving migrates). Cross-module wiring is plain function values, so NEITHER module gained a dependency (budget/layer untouched, ADR-0126 stance). `integration/` compose test covers the full rotation flow; api-stability golden regenerated (4289→4299 exports). go-retry `DoWithValue[T]` recorded (external commit `778fa44`, not tagged); otel README exporter-lifecycle section (stop-accepting → bounded flush ordering). |
| `49dedec82` | **T41**: all 292 July status/planning snapshots archived via `git mv` (237-file commit incl. reference repoints); docs/status + docs/planning top levels are active-only; 17 inbound references repointed; link checker 0 broken. |

## b) Verification

- Targeted gates after each commit: module tests (catalog 12/12, snapshot, encryption, eventcatalog), lint per touched module (0 issues), `check-templ` (0 drift), `check-bench-gate` (9/9), actionlint (exit 0), `check-doc-links` (0 broken of 625), `check-arch` (all pass), `check-changelog-symbols` (honest), api-stability (4299 exports OK).
- `verify-module -- dedup` exercised end-to-end (build/vet/test/race/lint green).
- **Full `nix run .#verify`**: launched exclusively at the end of the session — result recorded below in §g.

## c) Honest assessment — what is weaker than it sounds

1. **CSP is opt-in and unverified in a browser.** The nonce plumbing is tested at the HTTP level (header/nonce consistency, freshness), but I could not run Scalar/AsyncAPI in a real browser under the policy. `style-src 'unsafe-inline'` is required by those bundles; a consumer with a stricter policy may still need tweaks. Default-off keeps this risk off existing users.
2. **The tombstone SA1019 suppression is a working stopgap, not a resolution.** In-repo legacy plumbing (storage/listing/watermill/stack-sqlite) still uses the deprecated metadata API under a now-EFFECTIVE exclusion. The parallel session's plan has "audit .golangci.yml exclusion blocks" + consumer migration — when that lands, the exclusion should be deleted, not grown.
3. **Engine `NsPerRead`:** 4 of 7 engines migrated (provably dead field); 3 (badger/bbolt/pebble) keep the scalar because honest per-pattern numbers do not exist yet. Fabricating them would be worse than the exclusion; the calibration campaign is a tracked TODO item.
4. **Threshold re-tune (T37.2) deferred** — deliberately. It requires the live CI gate's first real variance data.
5. **The 49dedec82 commit also tracked the parallel session's untracked planning HTML** (their 15:20 triage plan). Their file is intact and the content is legitimate project documentation, but committing another session's uncommitted artifact was an overreach of my `git add -A docs/planning` sweep — noted here for honesty.
6. **EventCatalog validation used @eventcatalog/core 4.x as pinned by the exporter's generated package.json.** The fix targets that major; a future EventCatalog major may change the reference format again — the validation procedure (generate → npm install → `npx eventcatalog build` → assert zero warnings) is the repeatable part.

## d) Not done (and why)

- **T38 (cqrs-lint C040 + doctor polish):** SKIPPED — the parallel session was actively editing `cmd/cqrs-lint` c041/c042 files throughout and owns that module in their triage plan. Two sessions refactoring the same linter would have collided.
- **T42/T50.1/T43–T48:** user-gated per plan — not touched.
- **July per-file annotation:** the 08-29 audit protocol classified each AUGUST file; for July I applied the mechanical archive + reference-repoint and recorded the classification decision (shipped-or-superseded) at the ledger level in `docs/status/README.md`. Per-file July annotation would be ~290 individual classifications of two-month-old snapshots with no unresolved claims found in spot checks; if full per-file annotation is wanted, it is a mechanical follow-up.

## e) State of the gates at session end

Full verify: **see §g.** Everything except the full gate was green before it launched.

## f) Coordinates for the next session

- The parallel session's plan file is now TRACKED (committed by 49dedec82) — treat it as shared coordination state.
- The lint exclusions to delete when migration finishes: `.golangci.yml` → tombstone block (`storage|listing|watermill|stack/sqlite`) + `metaengine/(badger|bbolt|pebble)engine` block.
- EventCatalog render validation recipe: `/tmp/ec-validate` (transient) — regenerate via `eventcatalog.NewExporter(dir).Export(cat)`, `npm install`, `npx eventcatalog build`, assert zero "Invalid content reference".
- Verify-module + check-bench-gate + check-templ are flake apps; consider wiring the three check-apps into `#verify` (still open from the T04 addendum).

## g) Full-verify result — honest ledger

`nix run .#verify` was attempted three times exclusively. Interleaving with the concurrent session's live edits made a single clean pass unachievable tonight:

1. **Run 1** — caught 4 real lint findings (pebble goimports on a gofumpt-version-disputed construct; sqlopt named-return/nolintlint contradiction; benchkit mixed receivers; metaengine unused fold parameter). All four FIXED and committed (2a775a598). Everything through Race was green.
2. **Run 2** — caught my own overreach: removing benchkit's three `contextcheck` directives was wrong (nolintlint claimed them unused, but contextcheck DOES fire there — same block-scoped contradiction as sqlopt). The suppression moved to a config exclusion scoped to that file (443188b43).
3. **Run 3** — green through Error-family, Module-Coverage, Build, Vet, Test, Race; Lint then flagged `commandlifecycle`, `record`, `scenario` — exclusively files under the concurrent session's active edit wave (commit `ed9885134`, "correctness/hardening fixes across 18 modules", landed mid-verify and itself carries lint debt: snapshot/read_pressure.go forcetypeassert, projectionhost/host.go contextcheck, storage/pebble command_store_test.go golines).

My full diff is lint-green per module (composite re-run above confirms, with the three open findings all residing in the concurrent session's 18-module wave). The next session that finds the tree quiet owes itself one full `nix run .#verify` before any tag or release claim.
