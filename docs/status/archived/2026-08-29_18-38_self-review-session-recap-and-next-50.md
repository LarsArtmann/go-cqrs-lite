# Self-Review + Full Session Recap — T36–T39, T41, Lint-Gate Repair, and the Bug I Shipped

**Date:** 2026-08-29, 18:38 CEST
**Session:** second execution session (continuation), 15:30–18:38
**HEAD:** `6191dadc2` (history was rebased mid-session by the concurrent session — my 11 commits survived with new SHAs, interleaved with their `ce98b2dda` 18-module wave and `684f93dcf` nix go-pin fix)
**Concurrent session:** active the entire time (own triage plan, cqrs-lint + 18-module hardening wave + a TODO-wave status report of their own at 18:2x).

---

## a) FULLY DONE

| Work                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | Commit(s)   | Evidence                                                                       |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------- | ------------------------------------------------------------------------------ |
| **Lint-gate repair** — Aug 6 tombstone SA1019 exclusions were provably inert (patterns demanded `event.X` right after `SA1019:`; staticcheck prints `event/v4.X`); fixed regexes verified against real message text; pg/mysql/dgraph/duckdb engines dropped provably-dead `NsPerRead` (all four `ReadCosts` fields set — fallback chain never consults the scalar); narrow new exclusion for badger/bbolt/pebble until calibration data exists; 7 stale nolint directives removed                                                                                  | `deed17ef3` | per-module lint 0 issues; gate un-red                                          |
| **T36 docserver set** — per-request CSP nonce stamped on every script tag (SPA bundles, inline bootstraps, copy-button, theme scripts), `Config.EnableCSP` header opt-in (default off = byte-identical responses), CSPRNG-failure degradation; `nix run .#check-templ` codegen-drift gate (nixpkgs templ = pinned v0.3.1020); docs-ui.css GET test; catalog README production-deps table; cId-value-semantics CHANGELOG note                                                                                                                                       | `ec98f838f` | docserver tests incl. 3 CSP tests; check-templ exit 0                          |
| **EventCatalog exporter fix, proven against the real CLI** — `producers`/`consumers` were `{id: …}` objects → `InvalidContentEntryDataError` under pinned `@eventcatalog/core` ^4.6.3; bare service IDs also don't resolve (entry IDs are `<id>-<version>`). Exporter now emits versioned reference strings. `npx eventcatalog build` went from hard failure to clean build, zero warnings, rendered site                                                                                                                                                          | `1418a1a1e` | end-to-end render in /tmp (transient); regression assertions in exporter tests |
| **T37 bench-gate hardening** — 9 fixture cases pinning median computation (odd/even samples), strict >25% threshold, save-after-compare ordering, vanished/new informational handling → `nix run .#check-bench-gate`; `nix run .#verify-module -- <path>` scoped build/vet/test/race/lint; actionlint in devShell, workflows lint-clean (2 shellcheck infos fixed in ci.yml); baseline-regen runbook in BENCHMARKS.md                                                                                                                                              | `b417e336b` | fixture tests 9/9; verify-module exercised on dedup                            |
| **T39 first-class snapshot encryption** — `snapshot.NewTransformedStore` (protect/restore on state, routing metadata plaintext, errors on nil/incomplete) + `encryption.SnapshotStateCodec`/`RotatingSnapshotStateCodec` (Envelope carries key ID; `StaticKeyResolver` resolves retired keys; rotation without migration window); cross-module wiring is plain function values → neither module gained a dependency; `Corruption`-classified tamper errors; go-retry `DoWithValue[T]` in external repo (778fa44, untagged); otel README exporter-lifecycle section | `31d4a4638` | integration compose test (rotation flow), codec tests, api golden 4289→4299    |
| **T41 July archive** — all 292 July status/planning snapshots `git mv`'d to `archived/`, top levels active-only, inbound references repointed, link checker 0 broken                                                                                                                                                                                                                                                                                                                                                                                               | `446f76c61` | check-doc-links 625 targets, 0 broken                                          |
| **Self-caught regression fix** — the benchkit `MarshalJSON` receiver "fix" below (§d1) reverted to a value receiver with inline reasoning; permanent regression test added asserting value-marshal stamps `CodecName`                                                                                                                                                                                                                                                                                                                                              | `6191dadc2` | probe test green; benchkit suite green                                         |
| TODO_LIST coordination claims + engine-calibration backlog entry                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `f686e6ecc` | visible to the concurrent session                                              |
| Prior status report                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | `08b7c2220` | committed before this one                                                      |

## b) PARTIALLY DONE

1. **Full `nix run .#verify`** — three attempts, never a single clean exclusive pass (see §d4). Green through Error-family/Module-Coverage/Build/Vet/Test/Race in run 3; Lint stage blocked by the concurrent session's in-flight wave both times it got there (run 1's findings were mine and are fixed; runs 2–3's findings after my fixes were entirely in their files).
2. **CSP support** — HTTP-level tested (header/nonce consistency, default-off, freshness) but never run in a real browser against the Scalar/AsyncAPI bundles; `style-src 'unsafe-inline'` is required by those bundles and is a standing caveat. Default-off keeps the risk off existing users.
3. **Engine `NsPerRead` migration** — 4 of 7 engines migrated; badger/bbolt/pebble keep the scalar behind the exclusion because honest per-pattern calibration numbers don't exist yet. Exclusion is a tracked stopgap, not a resolution.
4. **Tombstone consumer migration** — the 17 SA1019s in storage/listing/watermill/stack-sqlite are suppressed by the now-working exclusion; the actual migration off the deprecated metadata API belongs to the concurrent session's declared work.
5. **T37 threshold re-tune** — deferred until the live CI gate accumulates real per-run variance; documented in the runbook.
6. **Skill references for new public API** — `recipes.md`/`modules.md` were NOT updated with the new snapshot-encryption API or the CSP option. CHANGELOG + godocs carry it; the consumer-facing skill docs lag. Missed obligation under the "Change an Exported Symbol" procedure.
7. **go-retry `DoWithValue[T]`** — committed locally, not pushed/tagged; no consumer in go-cqrs-lite uses it yet.
8. **EventCatalog validation** — done manually in a transient /tmp setup; not yet a repeatable flake app or CI job, so the next envelope format change can silently re-break the exporter.

## c) NOT STARTED

1. **T38** (cqrs-lint C040 + doctor `--format json` / `--fix --dry-run`) — deliberately skipped: the concurrent session was actively editing `cmd/cqrs-lint` all session and owns that module in their triage plan.
2. **T42 (annotation-depth policy), T50.1 (billing), T43–T48** — user-gated, untouched.
3. **Per-file July annotation** — archived at ledger level ("shipped-or-superseded"), not per-file.
4. **Wiring the check-apps into `#verify`** (check-templ, check-bench-gate, check-lint-config, check-arch, …) — still the known gap from the T04 addendum.
5. **GitHub Releases publishing** — unchanged.
6. **Snapshot-encryption PG/SQL store test** (encrypted-at-rest column assertion against a real server).
7. **Rotation write-back option** (re-encrypt-on-read) — loads resolve old keys but nothing migrates them automatically.

## d) TOTALLY FUCKED UP

1. **I shipped a silent behavior regression and only caught it while writing this report.** The recvcheck fix moved `benchkit.Config.MarshalJSON` to a pointer receiver — which drops custom marshaling for `json.Marshal(cfgValue)`, so reports would emit `"CodecName":""` and round-trip to the DEFAULT codec instead of the configured one. Zero tests covered value-shape marshaling. The user's "what did you fuck up" prompt made me probe my own diff; the probe failed exactly as predicted, and the fix (`6191dadc2`) restores the value receiver with the reasoning inline plus a permanent regression test. Lesson now in §e. This is the worst find of the session: it passed build, the full benchkit suite, and the lint gate.
2. **`UPDATE_SNAPS=true` run repo-wide damaged 5 unrelated golden files** (838 deletions in d2/openapi/asyncapi snaps that weren't even under test). Restored via `git restore`; the correct scoping rule (only the package under change) is now in §e.
3. **`git add -A docs/planning` swept the concurrent session's untracked plan HTML into my archive commit** (their file intact, but committing another session's uncommitted artifact was overreach).
4. **Trusted `nolintlint`'s "unused directive" twice; both times wrong.** nolintlint checks the directive's LINE, while golangci applies nolint block-scoped to declarations: (a) sqlopt — removed the directive, the full gate then flagged the named return it was suppressing, restored; (b) benchkit — removed three contextcheck directives, the next verify flagged the calls they suppressed, moved the suppression to a config exclusion. Cost ~1.5 verify cycles (~30 min) plus two extra commits.
5. **Ran the exclusive full verify three times against a tree the concurrent session was actively editing** (~60 min wall time, two runs invalidated mid-flight). A `git worktree` checkout would have isolated the gate from their editor; I never tried it.
6. **My integration compose test initially broke the GOWORK=off gate matrix** (undefined symbols under published pins) — the exact test-compile failure class the wave lessons already documented. Fixed with sibling replaces (`75653d8cb`), but I should have designed for the per-module gate from the start.

## e) WHAT WE SHOULD IMPROVE (process/standing rules)

1. **Never act on `nolintlint` "unused" alone** — verify by removing the directive and re-running the named linter before deleting; block-scoped suppressions are invisible to nolintlint. Candidate AGENTS gotcha.
2. **`UPDATE_SNAPS`/`UPDATE_SNAPS=clean` must be scoped to the package under change** — the clean pass deletes "obsolete" snaps by whichever tests ran; a filtered run will trash goldens of packages that didn't run. Candidate AGENTS gotcha.
3. **Any change to a custom marshaler's receiver set needs a value-shape and pointer-shape marshal probe.** `recvcheck` will keep pushing toward pointer receivers; for json.Marshaler types that's often wrong. Candidate AGENTS gotcha (and a good future cqrs-lint rule).
4. **New exported API ⇒ skill references updated in the same commit** (same-edit rule, like the api-stability golden).
5. **Compose tests in `integration/` must be designed for GOWORK=off** (sibling replaces planned up front, or fake-based coverage inside the owning module).
6. **Exclusive gates need a quiet-tree preflight**: `git status` clean + no concurrent editor activity, or run inside a worktree. Add to the preflight.sh idea.
7. **go.mod-level verification of "would a consumer trust this"**: for every new test, ask which gates compile it (workspace vs GOWORK=off vs CI matrix).
8. **gofumpt/golangci formatter version skew** (pebble's disputed construct): when two formatter versions disagree, restructure the code so neither has an opinion instead of picking a winner.

## f) UP TO 50 THINGS TO GET DONE NEXT

**Immediately actionable (my lane)**

1. One full exclusive `nix run .#verify` in a quiet window (post-concurrent-session).
2. Fix the 3 lint debts from the concurrent 18-module wave: snapshot/read_pressure.go forcetypeassert, projectionhost/host.go contextcheck, storage/pebble command_store_test.go golines.
3. Skill references: snapshot-encryption recipe in `recipes.md`; docserver CSP option; modules.md rows for `NewTransformedStore`/`SnapshotStateCodec`.
4. Tag-wave prep for the new APIs: cut encryption/snapshot tags, bump integration pins, strip the sibling replaces (script does it), re-run the GOWORK=off matrix.
5. Wire check-apps (check-templ, check-bench-gate, check-lint-config, …) into `#verify`.
6. Make the EventCatalog render validation a durable flake app (`check-eventcatalog`) with pinned node/npm from nixpkgs.
7. Add `check-templ` + `check-bench-gate` jobs to ci.yml.
8. CSP browser validation against the embedded Scalar/AsyncAPI bundles.
9. Bench 25% threshold re-tune after live CI variance data lands.
10. Engine `ReadCosts` calibration for badger/bbolt/pebble in a quiet window; then delete the exclusion block.
11. Push + tag go-retry `DoWithValue[T]`; evaluate adopting it at go-cqrs-lite retry call sites.
12. Snapshot encryption: rotation write-back option (transparent re-encrypt-on-read).
13. Snapshot encryption: PG/SQL store integration test asserting ciphertext-at-rest.
14. docserver: document `EnableCSP` in the package doc + a runnable example.
15. docserver: optional security headers (X-Content-Type-Options, Referrer-Policy) behind the same EnableCSP switch.
16. api-stability golden: include the `catalog/v4/docserver` package (currently invisible to the golden — today's new exports weren't tracked).
17. AGENTS gotchas from this session: nolintlint-vs-block-scope, UPDATE_SNAPS scoping, recvcheck/MarshalJSON trap, formatter-version disagreement.
18. `docs/status/README.md` index generation script (file counts are hand-maintained).
19. Preflight script for long gates (cache writable, GOTMPDIR space, tree quiet, load check) — from the wave lessons.
20. metaengine bench cutover flake: poll-until-visible fix (carried over from the 14-50 report).

**Coordinate with the concurrent session (their lanes, don't duplicate)**
21. T38 cqrs-lint C040 + doctor `--format json` / `--fix --dry-run`.
22. Tombstone consumer migration (storage/listing/watermill/stack-sqlite), then delete the SA1019 exclusion block.
23. Their `.golangci.yml` exclusion-block audit — hand them the now-working blocks with deletion criteria.
24. CHANGELOG staleness linter (`[Unreleased]` bullets whose symbols already shipped).
25. Per-tag CHANGELOG-section audit script.
26. go.mod pin-vs-latest-tag linter rule (their V-next idea).
27. GitHub Releases publishing automation.
28. `/mnt/buildcache` permanent replacement for all tool caches.

**Harvest-adjacent (noticed this session, not acted on)**
29. TODO_LIST restructure decision: 688-line harvested format vs the old ≤377 gate — split TODO vs BACKLOG?
30. July per-file annotation depth (optional second pass).
31. `snapshot/` transform store: example in the package README.
32. EventCatalog exporter: derive producers for queries (current auto-derive covers commands/events — verify queries get consumers AND producers symmetrically).
33. Envelope versioning for snapshot state (explicit v2 path before v1 calcifies).
34. `benchkit`: audit remaining mixed-receiver types beyond Config.
35. cqrs-lint rule idea: flag pointer-receiver MarshalJSON/Stringer on types marshaled by value (the benchkit trap, generalized).
36. docserver: ETag/cache headers for the static assets (immutable hashed content would allow aggressive caching).
37. docserver: `httptest`-level test that no script tag lacks a nonce when EnableCSP is on (regex audit over rendered pages).
38. templ-components: upstream a "Nonce auto-injection via templ context" feature request so literal script tags get nonces without the KV dance.
39. flake: pin `golangci-lint fmt` vs treefmt policy in one place so formatter-version skew (pebble case) can't recur.
40. `scripts/check-doc-links.sh`: also validate backtick path references (today's July repoint needed manual grep; the checker only walks markdown links).
41. EventCatalog: store the render-validation project layout under `docs/` so the manual recipe outlives /tmp.
42. integration module: consider splitting the single giant test package into per-domain packages to shrink lint/typecheck surface.
43. `#verify` runtime: document the stage list + expected durations in BENCHMARKS/CONTRIBUTING.
44. TODO_LIST "IN PROGRESS" claim markers: promote to a shared CLAIMS.md convention (TODO_LIST edits raced twice today).
45. snapshot: fuzz the Envelope round-trip (base64/JSON edge inputs) — property test exists for codec, not for the envelope path.
46. Revisit `stack/sqlopt.NewSecondaryBackend` — likely shares the same named-return pattern that conflicted on OpenPrimaryBackend (grep-verify and unify).
47. metaengine: apply the same "provably dead field" audit to `NsPerWrite`/`NetworkRTT` priors that ReadCosts got for reads.
48. benchkit: probe-style guard tests for UnmarshalJSON defaults (empty CodecName → default codec) so the roundtrip contract is explicit.
49. Make `check-docserver-css` and friends skippable/includable via args (verify --module pattern extended to check apps).
50. Post-rebase hygiene: my commit SHAs changed mid-session (rebase by concurrent session) — add a coordination rule: no history rewrites while another session holds unpushed work.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Tombstone exclusions policy:** when the concurrent session's consumer migration lands, do you want the SA1019 suppression block DELETED (strict: internal code must be canonical immediately) or KEPT until the v5 cut (lenient: external consumers migrate on their own schedule)? This decides whether the next migration PR deletes config lines or not.
2. **CSP default:** should `EnableCSP` stay opt-in default-off indefinitely, or flip to default-on at the next docserver minor once someone has browser-validated Scalar + AsyncAPI React under the policy? (I cannot do that browser validation from here; and flipping the default is a behavior change for existing deployments.)
3. **Release sequencing:** the snapshot-encryption/docserver CSP APIs are unreleased and pinned via sibling replaces in `integration/go.mod` — do you want a small tag wave soon (encryption + snapshot + integration pin bump, strip replaces), or should this ride the next planned wave? Related: should go-retry's `DoWithValue[T]` be pushed and tagged now, or held with the next go-retry release?
