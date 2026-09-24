# Status Report — Data-Mesh Pareto Execution Session (T01–T27)

**Date:** 2026-09-24 12:26 (Wednesday)
**Session scope:** Full execution of the
[data-mesh & federation Pareto plan](../planning/2026-09-23_22-12_SUPERB-data-mesh-federation-pareto-plan.md)
(T01–T27 / f01–f72) under the user's "GET SHIT DONE — the WHOLE TODO LIST"
mandate. This report covers what this session did, found, and broke.
**Format note:** user explicitly demanded `.md` (skill default is HTML) —
override honored, not propagated.

---

## a) FULLY DONE

All verifiably complete: committed, tested, gates green.

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | Evidence                                                                                                                                                                                    |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **T01 — `catalog.index.json` export manifest** (id/kind/version/path, deterministic order, written last; golden snap + 5 tests incl. re-export byte-stability + paths-exist)                                                                                                                                                                                                                                                                                                                                                                                 | `catalog/eventcatalog/index_manifest.go` + tests; golden `catalog-index-manifest.snap` (containers included); README hub-diff section; landed via daemon `00ec6901a` + authored docs commit |
| 2  | **T02 — `WithSkipBootstrapFiles` option** (exactly the 2 bootstrap files omitted, content byte-identical; 3 tests; api golden regen; CHANGELOG)                                                                                                                                                                                                                                                                                                                                                                                                              | commit `902936e25`                                                                                                                                                                          |
| 3  | **T03 — `WithPlainRefIDs` + full-severity linter gate.** Root cause CORRECTED by experiment: core resolves composite `<id>-<version>` Astro entry IDs, linter indexes bare frontmatter IDs — no single format satisfies both; versioned dirs (the plan's mechanism) would violate `duplicate-resource-ids`. `check-eventcatalog` now render-validates BOTH profiles AND runs `@eventcatalog/linter@1.1.20` at full error severity — **"No problems found"**, i.e. the hub's two warn-suppressed rules are re-armed                                           | options_test (both formats pinned); `scripts/check-eventcatalog.sh` 5/5 step; full gate run green                                                                                           |
| 4  | **T04 — owners on EVERY ownable kind**: `catalog.Flow.Owners` added (the only missing kind) + registry copy wired + fixture owners/summaries/OrderItem-entity/contract-file; `TestExporter_OwnersEmittedOnEveryOwnableKind` pins all 9 kinds                                                                                                                                                                                                                                                                                                                 | T03+T04 commit; catalog module tests green                                                                                                                                                  |
| 5  | **T05 — data-mesh conformance mapping doc** (4-principle table, module map, honest gaps) + bidirectional Cordis cross-links                                                                                                                                                                                                                                                                                                                                                                                                                                  | commit `e2ca86b15`; doc-check green                                                                                                                                                         |
| 6  | **T06/T07 — ADR-0146 (no federated query engine; journal replication is the cross-domain mechanism) + ADR-0147 (mesh-level policy enforcement = non-goal)**; ADR index repaired (0143–0145 were missing — fixed on sight); meta-engine-design cross-ref; status-report execution addendum (f70/f26)                                                                                                                                                                                                                                                          | `docs/adr/0146…`, `0147…`, `docs/adr/README.md`, all in `e2ca86b15`                                                                                                                         |
| 7  | **T08–T10 — `example/mesh-demo`**: orders + billing bounded contexts wired only through bilateral event contracts (`order.placed` →, `invoice.issued` ←), per-context catalogs with explicit producers/consumers, data products, teams; headless export commands (`-plain`, `-skip-bootstrap`); 9 tests green incl. coeffects dangling-free per source, cross-domain round trip, manifest union w/o ID collisions, both-copies-carry-both-sides; registered in go.work, flake exampleModules, module-layers, cqrs-lint catalog, api-stability exclusion maps | commit `0f1e10803` (+daemon); `go run . demo` verified end-to-end                                                                                                                           |
| 8  | **T11 — recipes.md §2.41** "Declare data products + contracts end-to-end" + doc-check compile classification + core.md §3.9 cross-ref                                                                                                                                                                                                                                                                                                                                                                                                                        | `TestRecipes` green; doc-check 1215 refs green                                                                                                                                              |
| 9  | **T13–T15 — advanced.md §6.21** "Data Products: Ports, Replication, and Contract Evolution" (journal-as-outbox input ports w/ ADR-0016/0146, ServeSSE/query output ports, contract-evolution w/ upcasting); ADR-016 filename link verified against disk                                                                                                                                                                                                                                                                                                      | doc-check green                                                                                                                                                                             |
| 10 | **T16 — docserver renders DataProducts**: overview section + detail page (owners, input ports, output ports with contract paths), route `/docs/eventcatalog/data-products/{id}`, templ regenerated, 3 tests                                                                                                                                                                                                                                                                                                                                                  | commit `1bdd5d5d7`                                                                                                                                                                          |
| 11 | **T17 — api-stability golden verified** to cover DataProduct/DataContract/options/Flow.Owners; `TestEvery` green                                                                                                                                                                                                                                                                                                                                                                                                                                             | golden at 7501 exports                                                                                                                                                                      |
| 12 | **T18 — cqrs-lint E019 `data-product-without-contract`** (AddDataProduct literal scanner w/ typed-receiver + literal-type fallback; fires on contractless outputs and output-less products; RULES.md regenerated; meta-test + README counts bumped)                                                                                                                                                                                                                                                                                                          | commit `d098b57a8`; 3 detector tests; full cqrs-lint pkg green                                                                                                                              |
| 13 | **T19 — `docs/MIGRATION-grpc-to-v5.md`** (surface inventory, capability→replacement table, decision help); faq.md v5-removal entry + transport/grpc README both link it. The errorfamily claim was corrected after source verification (outcomes arrive as `command.rejected` events, NOT wire metadata — first draft was wrong)                                                                                                                                                                                                                             | doc-check green                                                                                                                                                                             |
| 14 | **T20 — ROADMAP Theme 11 "Data-Mesh-Ready SDK"** (honest PARTIAL status) + Non-Goals entries citing ADR-0146/0147; Declined guard checked first — no re-litigation                                                                                                                                                                                                                                                                                                                                                                                           | ROADMAP.md                                                                                                                                                                                  |
| 15 | **f14 — hub `.eventcatalogrc.js` comment corrected** (real cause: ref-format incompatibility, not unversioned dirs)                                                                                                                                                                                                                                                                                                                                                                                                                                          | hub commit `646afcd`                                                                                                                                                                        |
| 16 | **Mass formatting restoration** — 296 files reflowed to the canonical treefmt 3-group import layout (a prior auto-committed wave had collapsed it; verified import-block-only before committing)                                                                                                                                                                                                                                                                                                                                                             | daemon `2865e2c79`                                                                                                                                                                          |
| 17 | **Pre-existing CI repairs (found while registering mesh-demo)**: storage pins `v4.10.0`→`v4.10.1` in projectionadapter + system (+ tidy); `system/integration` sibling replace w/ documented strip-at-wave comment; systemtest tidy; storage dep budget 13→14 for the un-budgeted go-sqlitestore; api-stability `TestEvery` + `check-module-layers` green again                                                                                                                                                                                              | `0f1e10803` + follow-ups                                                                                                                                                                    |

## b) PARTIALLY DONE

| Item                                         | What works                                                                                                                               | What remains                                                                                             | Blocker                                                                                                                                                                | Effort                        |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| **T21 — goal-shaped-app data-product view**  | cqrs.yaml documents the materialized-view upgrade path precisely; app boots, tests green (`e843c0470`)                                   | NOT an active view — shipped as a commented block                                                        | IVM views need libSQL (`turso` driver); the `metaengine/tursoengine/v4.2.0` tag is POISONED in the module proxy (malformed zip path — `go get` fails creating the zip) | S once tag retracted + re-cut |
| **CHANGELOG receipts**                       | Exporter-wave items (manifest, both options, Flow.Owners, fixture) have `[Unreleased]` entries; symbols-gate green                       | No entries for mesh-demo, E019, docserver DataProduct pages, recipes §2.41, gRPC guide                   | none — simply not written                                                                                                                                              | S                             |
| **T26 — spot-verify 3 single-source claims** | ADR-016 outbox wording VERIFIED (read directly for the migration guide)                                                                  | watermill broker matrix claims + `system/config_types.go:124-205` DeploymentConfig field list unverified | none                                                                                                                                                                   | S                             |
| **Final verification**                       | All touched modules individually green (catalog, cqrs-lint, examples, api-stability meta-tests, doc-check, check-eventcatalog full gate) | `nix run .#verify` NOT yet run as the closing gate; TODO_LIST strike-through harvest not done            | session interrupted by this report                                                                                                                                     | M                             |
| **D1–D3 autonomous decisions**               | All three executed (mapping doc, non-goal ADR, ownership split)                                                                          | Never ratified by the user — overridable by design                                                       | user input                                                                                                                                                             | —                             |

## c) NOT STARTED

| Item                                                                          | Why not started                                                                              | Still wanted?                               |
| ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------------- |
| **T22 — [hub] stale-source CI** (export-command probe per sources.json entry) | Deferred to P5; hub repo untouched this session beyond the rc-comment fix                    | Yes — routed per D3                         |
| **T23 — [hub] fail union-merge on dangling coeffect refs**                    | same                                                                                         | Yes — the mesh-level governance enforcement |
| **T24 — [hub] owners/teams dedupe across exports**                            | same                                                                                         | Yes                                         |
| **T25 — EventCatalog SLA/freshness schema probe → proposal**                  | not reached                                                                                  | Yes (Low)                                   |
| **f72 — SystemNix hub-plan cross-reference note**                             | lives in the SystemNix repo; no work done there                                              | Yes (Low)                                   |
| **Push to remote**                                                            | last push authorization covered only the plan commit (5e329b981); ~30 local commits unpushed | User decision                               |

## d) TOTALLY FUCKED UP

1. **I briefly shipped a config that broke the example's default run.** The
   first T21 attempt put an ACTIVE `materialized_views` block on the sqlite
   driver — the app died at startup (`SQL logic error: near "MATERIALIZED"`).
   Caught by a manual smoke run ~10 minutes later; fixed + verified
   (`e843c0470`). Root cause: I wrote config from the API surface
   (`WithMaterializedViews` is wired into the sqlite driver factory) without
   checking the ENGINE capability first. Lesson applied: smoke-run every
   shipped config; the capability doc (not the wiring) is the truth.
2. **The daemon committed the broken turso intermediate into history**
   (`a3a7f35d8`, `599d3a62a`): main.go importing tursoengine + go.mod
   requiring the poisoned tag. HEAD is fixed, but those two commits are
   un-buildable GOWORK=off. Mitigation: none needed for master; history
   archaeology will hit it.
3. **`metaengine/tursoengine/v4.2.0` tag is POISONED in the module proxy**
   (zip contains files with `\x06` control chars — `go get` fails with
   "malformed file path"). Pre-existing, discovered this session. Blocks
   T21's active view AND any consumer pinning that version. Needs
   retraction + re-tag.
4. **Published `system/v4.9.0` graph is poisoned by the dead `storage/v4.10.0`
   tag** (cut then deleted; every consumer resolving published system v4.9.0
   fails tidy). Pre-existing (2026-09-22 wave). I repaired the LOCAL modules
   (pins → v4.10.1 + documented replace in system/integration), but the
   published graph stays broken until the next system tag.
5. **`nix run .#check-md-go` is broken** (md-go-validator vendorHash
   mismatch — upstream dep drift). Pre-existing infra failure; my new docs
   carry no `go` fences so they're untested by that gate this session.
6. **BuildFlow pre-commit cqrs-lint step fails** (D007 "unsafe path
   resolves outside root" on eventtest/signing/backuptest/watermill files —
   report-only in pre-commit, but it means cqrs-lint's repair path is
   broken repo-wide). Pre-existing, not touched.
7. **lychee 404 on `https://github.com/LarsArtmann/eventcatalog-hub`** in
   catalog/README (repo is private/forgejo-only). Pre-existing warning.
8. **GitHub reports 3 high Dependabot vulnerabilities on master**
   (carried over from the previous session; unaddressed, out of scope).

## e) WHAT WE SHOULD IMPROVE

1. **Tag hygiene has no deletion guard.** Two poisoned/dead tags
   (storage/v4.10.0 deleted-after-push; tursoengine/v4.2.0 binary junk)
   broke the published dependency graph for EVERY consumer. Fix: never
   delete a pushed tag (retract instead); add a pre-push zip-content check
   to tag-release.sh (reject control chars / unexpected binary files); add
   a weekly "proxy resolves every module's own latest tag" probe.
2. **The changelog-symbols gate has an honesty blind spot:** it verifies
   CITED symbols but never flags uncited consumer-visible work (this
   session: mesh-demo, E019, docserver pages all missed `[Unreleased]`
   entries and the gate stayed green). Fix: diff api_surface.txt exports +
   new modules against CHANGELOG mentions; warn on uncited surface.
3. **Config-carrying examples need boot-smoke tests.** goal-shaped-app's
   test suite was green while its shipped cqrs.yaml broke startup. Fix: one
   `TestAppBootsWithShippedConfig` per example that loads the repo's actual
   cqrs.yaml and constructs the system.
4. **The auto-commit daemon raced EVERY authored commit this session** (6+
   `cannot lock ref 'HEAD'` failures; it absorbed staged files mid-hook).
   `--no-verify` is the documented workaround for mechanical commits, but
   substantive commits then skip cqrs-lint/BuildFlow. Fix options: daemon
   lock coordination (respect index.lock), or a `--no-verify`-for-daemon
   mode with a post-hoc gate runner.
5. **My own discipline drift:** I ran `gofmt -w` directly once (the repo's
   canonical formatter is treefmt via `nix fmt` — the two disagree on
   import grouping, which fed the 296-file reflow churn). Rule: never call
   bare gofmt in this repo.
6. **Evidence-first habit paid off — keep it:** three plan assumptions died
   on contact with experiments this session (versioned-dir layout,
   "owners not emitted", linter-dir theory). The plan's mechanisms were
   wrong twice; the GOALS survived because I verified before building.

## f) Next tasks (harvest candidates)

| #  | Task                                                                                                                                                 | Impact   | Effort | Category       |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------------- |
| 1  | Run `nix run .#verify` as the closing gate for the wave                                                                                              | Critical | M      | Quality        |
| 2  | CHANGELOG `[Unreleased]` receipts: mesh-demo, E019, docserver DataProducts, recipes §2.41, gRPC guide                                                | High     | S      | Documentation  |
| 3  | docs-health strike-through pass on TODO_LIST data-mesh rows (T01–T21 done)                                                                           | High     | S      | Cleanup        |
| 4  | [hub T22] CI stale-source detection: run each sources.json export command, report drift                                                              | High     | M      | Feature        |
| 5  | [hub T23] fail union-merge on dangling cross-source coeffect refs (parse coeffects.md at merge)                                                      | High     | M      | Feature        |
| 6  | [hub T24] owners/teams dedupe strategy + implement owner union at merge                                                                              | Medium   | M      | Feature        |
| 7  | [hub] wire governance lint into hub CI: plain-refs re-export per source + @eventcatalog/linter at error (the path the corrected rc comment promises) | High     | M      | Feature        |
| 8  | T25: probe EventCatalog data-product schema for SLA/freshness fields → ROADMAP raw idea                                                              | Low      | S      | Documentation  |
| 9  | T26 tail: verify watermill broker matrix claims + DeploymentConfig field list                                                                        | Medium   | S      | Documentation  |
| 10 | f72: SystemNix repo — cross-reference note to this plan + hub plan                                                                                   | Low      | S      | Documentation  |
| 11 | Release train: cut catalog/v4.6+ tag wave (manifest + options + Flow.Owners + docserver) so mesh-demo's pre-release replace can be stripped          | Critical | M      | Release        |
| 12 | Retract poisoned `metaengine/tursoengine/v4.2.0` + re-cut clean tag                                                                                  | Critical | S      | Release        |
| 13 | Decide storage/v4.10.0 handling: re-cut the tag (repopulates proxy) vs retraction; document in tag-hygiene rule                                      | Critical | S      | Release        |
| 14 | Pin-sweep: bump example pins to the new catalog tag + strip mesh-demo replace block                                                                  | High     | S      | Release        |
| 15 | Add boot-smoke tests for shipped cqrs.yaml in goal-shaped-app (+ scheduler-otel-status if config-carrying)                                           | High     | S      | Quality        |
| 16 | tag-release.sh: pre-push zip-content guard (control chars, binary junk)                                                                              | High     | S      | Quality        |
| 17 | changelog gate: flag uncited consumer-visible surface (api_surface diff vs CHANGELOG)                                                                | Medium   | M      | Quality        |
| 18 | Repair `nix run .#check-md-go` vendorHash drift                                                                                                      | Medium   | S      | Infrastructure |
| 19 | Fix cqrs-lint D007 repair-path "unsafe path" failures (report-only today)                                                                            | Medium   | M      | Bug            |
| 20 | Replace/annotate the 404 hub URL in catalog/README (forgejo link or private marker)                                                                  | Low      | S      | Documentation  |
| 21 | Address the 3 high Dependabot vulnerabilities on master                                                                                              | High     | M      | Security       |
| 22 | mesh-demo: add a system.New-backed variant (coeffect runtime gate demo; current demo is pure deciders)                                               | Medium   | M      | Feature        |
| 23 | goal-shaped-app: once turso tag is clean — activate the IVM view + boot test                                                                         | Medium   | S      | Feature        |
| 24 | Hub sources.json: onboard mesh-demo's two export commands (living dogfood of the onboarding contract)                                                | Medium   | S      | Feature        |
| 25 | E019: teach the scanner non-literal DataProduct args (variable-passed products are invisible today — documented, but a gap)                          | Low      | M      | Feature        |
| 26 | docserver: DataProduct badges/hidden flag rendering (declared but unrendered fields)                                                                 | Low      | S      | Feature        |
| 27 | check-eventcatalog: pin the linter install to a lockfile (currently @1.1.20 ad hoc)                                                                  | Low      | S      | Infrastructure |
| 28 | Sweep remaining daemon-committed formatting anomalies (spot-check 5 random files vs `nix fmt --fail-on-change`)                                      | Low      | S      | Cleanup        |
| 29 | Catalog: document the two-export workflow (render + lint) in SKILL.md routing table, not just catalog/README                                         | Medium   | S      | Documentation  |
| 30 | Archive this session's temp artifacts (`/tmp/ec-*` fixtures) — no repo action, listed for completeness                                               | —        | XS     | Cleanup        |

## g) Questions I cannot answer myself

1. **Release trigger:** may I cut the next tag wave (catalog + dependent
   modules) to publish the manifest/options/Flow.Owners surface and strip
   mesh-demo's pre-release replace? I did NOT release anything this session
   — proxy-facing tags are irreversible-ish and the release train is
   owner-gated. (Unblocks: task 11/14, hub onboarding of the new options.)
2. **Poisoned/dead tags:** for `tursoengine/v4.2.0` (binary junk) and
   `storage/v4.10.0` (deleted after push) — retract-and-re-cut, or
   re-create `storage/v4.10.0` at the commit where v4.10.1's content
   landed? Re-creating a deleted tag re-poisons anyone who cached the
   absence; retraction is cleaner but leaves published system/v4.9.0's
   graph broken until the next system tag. This trades off your consumers'
   pain vs tag aesthetics — your call.
3. **Scope of "the WHOLE TODO LIST":** T22–T24 + f72 live in OTHER repos
   (eventcatalog-hub, SystemNix) per decision D3. Execute them there under
   this mandate (I have local checkouts and committed the hub rc fix
   already), or does the mandate end at go-cqrs-lite's boundary?

---

**Report discipline:** this file is a snapshot — section (f) belongs in
TODO_LIST/ROADMAP via docs-health HARVEST once the session resumes; section
(b)'s receipts and the final `#verify` gate are the immediate next actions.
