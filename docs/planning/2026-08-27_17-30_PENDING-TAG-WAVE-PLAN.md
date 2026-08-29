# Pending v4 Tag Wave — Module-Order Plan (T03 sign-off input)

> **Date:** 2026-08-27 17:30 CEST · **Task:** T01/F01.4 of the
> [goal-gap-closure Pareto plan](archived/2026-08-27_16-18_GOAL-GAP-CLOSURE-PARETO-PLAN.md) ·
> **Status: WAVE COMPLETE — ALL BATCHES B1–B7 CUT 2026-08-29 (39 tags).** Sign-off
> (2026-08-29): staged → upgraded to full authorization ("GET SHIT DONE").
> B1: event/v4.9.0, schema/v4.3.1, dedup/v4.2.1, dispatcher/v4.3.1.
> B2: command/v4.8.1, query/v4.7.1, middleware/v4.5.1, scheduling/v4.3.1,
> listing/v4.3.0, pgtestcontainer/v4.1.0.
> B3 (versions corrected per Added⇒minor rule): snapshot/v4.4.0,
> decider/**v4.5.0** (plan said v4.4.1; ErrNilDecide is new API),
> storage/v4.8.1, storage/memory/**v4.4.0** (LogStore core is new API),
> storage/pebble/v4.3.0, storage/bbolt/**v4.1.0** (batch-commit option),
> storage/turso/**v4.3.0** (quota classification), storage/backuptest/**v4.1.0**
> (incremental suite).
> B4: kv/v4.2.1, commandlifecycle/v4.0.1, commandlifecycle/projections/v4.0.1,
> idempotency/kvstore/v4.2.1, idempotency/sqlstore/**v4.3.0** (NewMySQLStore),
> system/v4.6.0 (three new sentinels).
> B5: watermill/v4.5.1, signing/v4.2.1 (plan guessed minor; diff is hygiene),
> encryption/v4.3.0 (store transforms), projectionhost/**v4.4.0** (checkpoint
> options are new API).
> B6 (versions corrected): dgraphengine/duckdbengine/mysqlengine/irohengine/
> quic **v4.1.0** each (durability tiers, vector/graph waves, MariaDB dialect —
> plan's patch proposals were wrong), tursoengine/loopback v4.0.1,
> projectionadapter v4.4.1, graph/v4.2.1.
> B7: transport/http/**v4.3.0** (SSEEventID — plan's "notices only" missed the
> new exports), transport/grpc/v4.2.1. Dead eventtest tags deleted.
> **§4 sweep DONE:** zero local replaces remain repo-wide (26 files dropped,
> every module standalone-builds; stack/pebble → storage/pebble v4.3.0 +
> backuptest v4.1.0; duckdb/mysql/iroh engines → metaengine v4.12.0).
> **§5 verification:** clean-room go get per batch ✓, clean-room system/v4.6.0
> consumer build ✓, api-stability golden 4289 exports ✓, cqrs-lint suite ✓,
> changelog-symbols ✓, verify-ci matrix ✓ (76 modules; two post-sweep pin
> gaps caught and fixed: system → watermill v4.5.1 handler-independence,
> system/integration → duckdbengine v4.1.0 driver registration — the sweep's
> build-only check missed test-only and init-time symbol deps).
> **Re-verified 2026-08-29:** the three post-plan code commits (5ec4b1b39
> listing/pgtestcontainer, c9e464eda watermill/system, 9455f687a standalone-matrix
> pin fixes) are folded into B2/B4/B5 below; the remaining post-plan commits are
> docs-only. Tree clean at 93948cbbb.
> **Evidence:** per-module `git diff <latest-tag>..HEAD -- '*.go'` (prod files,
> tests excluded), CHANGELOG `[Unreleased]`, `git tag -l`, GOWORK=off build
> probes. All verified 2026-08-27.

## 0. Hard prerequisites

1. **Clean tree.** `scripts/tag-release.sh` must run from a clean tree. The
   deep-review session's uncommitted files (decider/, kv/, commandlifecycle/,
   system/, projectionhost/, snapshot/, AGENTS.md, cqrs-lint, idempotency/
   sqlstore) must land first — several are IN this wave.
2. **User authorization per module batch** (repo rule: never tag/push without
   explicit instruction).
3. **Pre-tag checklist per module** (`F03.1`): standalone
   `GOWORK=off go test ./... -count=1` (incl. test subpackages), `#vulncheck`,
   `#check-arch`, api-stability golden current.
4. **Cut→push interleave:** with siblings unpublished, a dependent module's
   tag-time tidy resolves siblings via direct VCS fetch (GOPRIVATE) — each tag
   must be PUSHED before the next module that requires it is cut.

## 1. Already shipped (stale TODO claims, do NOT retag)

| Module     | Tag             | Covered claim                        |
| ---------- | --------------- | ------------------------------------ |
| event      | v4.8.0 (08-22)  | DecorateJournal (ADR-0126)           |
| metadata   | v4.6.0 (08-22)  | BrandedString/ActorString            |
| metaengine | v4.12.0 (08-22) | capability audit + iroh exports      |
| storage    | v4.8.0 (08-22)  | OpenSQLiteInMemory shared-cache DSNs |
| decider    | v4.4.0 (08-22)  | *Ref identity forms                  |

Stranded-commit repair: `092b5e8a8` landed as cherry-pick `491379a2b`
(verified on master); `4907b6afc` obsolete (bench go.mod has no
pseudo-versions). Stranded-commit TODO item closed.

## 2. Proposed cut order (7 batches, dependency-first)

Versions are proposals; final call at cut time from CHANGELOG
(Added ⇒ minor, Fixed-only ⇒ patch). "Replaces unlocked" = local sibling
replaces that become droppable once the batch is pushed.

| #  | Batch                        | Modules (current → proposed)                                                                                                                                                                                                                                                                                                                                | Rationale / unlocks                                                                                                                                             |
| -- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 | core fixes                   | `event v4.8.0 → v4.9.0` (Orchestration alias + streaming/family fixes), `schema v4.3.0 → v4.3.1`, `dedup v4.2.0 → v4.2.1`, `dispatcher v4.3.0 → v4.3.1`                                                                                                                                                                                                     | event first: nearly everything pins it. Unlocks encryption/signing replaces (event.ErrInnerStoreNot*, Rejecting* on-disk refs).                                 |
| B2 | command/query tier + listing | `command v4.8.0 → v4.8.1`, `query v4.7.0 → v4.7.1`, `middleware v4.5.0 → v4.5.1`, `scheduling v4.3.0 → v4.3.1` (Due ordering, WithMaxRetries clamp), `listing v4.2.0 → v4.3.0` (Status/StatusClassifier feat, StatusMiddleware deprecated — 5ec4b1b39), `testutil/pgtestcontainer v4.0.0 → v4.1.0` (AfterRun; B3 storage standalone test builds resolve it) | listing is a MINOR (new API); rest Fixed-only patches; depend on published B1 pins. listing MUST precede B3 storage (its SQL reader surfaces `listing.Status`). |
| B3 | snapshot chain               | `snapshot v4.3.0 → v4.4.0` (Encoding field, validated ctor), `decider v4.4.0 → v4.4.1` (clamp fixes), `storage v4.8.0 → v4.8.1` (error-family truth, DuckDB PK), `storage/memory v4.3.1`, `storage/pebble v4.3.0` (verify Added vs Fixed), `storage/bbolt v4.0.1`, `storage/turso v4.2.1`, `storage/backuptest v4.0.1`                                      | Gap-wave ordering snapshot→decider→storage→pebble/bbolt. Drops the `decider => ../snapshot` replace.                                                            |
| B4 | lifecycle + kv + idem        | `kv v4.2.1` (Cache.Set invalidation), `commandlifecycle v4.0.1` (+`projections v4.0.1`) (Recorder version fix), `idempotency/kvstore v4.2.1`, `idempotency/sqlstore v4.2.1` (corrupt-row skip), `system v4.5.0 → v4.6.0` (shutdown-dependency validation + synthetic engine names + ErrShutdownDependencyInvalid — 96ecbf1f2, 3358d3794, c9e464eda)         | system minor (new sentinel + validation). Drops `idempotency/kvstore => ../sqlstore`, `integration => ../middleware` replaces.                                  |
| B5 | crypto + host + watermill    | `encryption v4.2.0 → v4.3.0` _, `signing v4.2.0 → v4.3.0`_, `projectionhost v4.3.0 → v4.3.1` (worker hardening), `watermill v4.5.0 → v4.5.1` (at-least-once catch-up checkpoints — c9e464eda)                                                                                                                                                               | *verify CHANGELOG Added entries since 08-16 for minor vs patch. Requires B1 event tags + replace-drop verify.                                                   |
| B6 | engine patches               | `dgraphengine v4.0.3`, `duckdbengine v4.0.2`, `mysqlengine v4.0.1`, `graphadapter v4.0.1`, `projectionadapter v4.4.1`, `tursoengine v4.0.1`, `metaengine/irohengine v4.0.1` (+`loopback v4.0.1`, `quic v4.0.1`)                                                                                                                                             | Engines are dep-isolated; metaengine v4.12.0 already published. Drops engine `=> ../metaengine` replaces where symbols are now published.                       |
| B7 | transport finals             | `transport/http v4.2.1`, `transport/grpc v4.2.1` — deprecation notices only (ADR-0127); prerequisite of the v5 deletion (plan T08)                                                                                                                                                                                                                          | Last v4 tags these modules ever get.                                                                                                                            |

**Total: 36 tags across 7 batches** (B1 4, B2 6, B3 8, B4 6, B5 4, B6 7+3 sub-modules, B7 2).

## 3. Explicitly NOT tagged (v5 deletes them)

`stack/*` (9 modules), `storage/view`, `storage/relational`,
`graph.GraphProjection` surface, `example/*`, `integration/`, `benchkit`,
`cmd/*` (tooling, tagged independently if ever), `metaengine/bench` (CGo,
no changes needed), `scheduling/sqlstore` (never tagged, unpublished).

## 4. Replace-drop sweep (after B1–B6 pushed)

30 go.mod files carry local sibling replaces (2026-08-27 census). Drop rule:
remove a replace iff its target module's published tag carries the symbols
the module needs, then `go mod tidy` + GOWORK=off build re-verify. Known
clusters: system ×6, cqrs-bench ×2, event ×1 (schema), engines ×14,
examples ×8, integration ×1, idempotency ×1, decider ×1, commandlifecycle/
projections ×1. Expect the irohengine/quic+loopback `=> ../../metaengine`
replaces to drop only if B6 exposes everything (verify per symbol).

## 5. Post-wave verification

1. GOWORK=off build matrix over ALL swept modules (the 08-22 lesson:
   published-consumer stranding).
2. `nix run .#verify-ci` (mirror of CI per-module matrix).
3. cqrs-lint taskmanager golden V006 version-set refresh (pins the version
   set — same wave).
4. api-stability meta-tests + changelog-symbols gate.
5. Clean-room consumer build of `system/v4@v4.6.0`.

## 6. User decision points

- [ ] Authorize B1–B7 as proposed (or trim: minimum viable = B1+B3+B4 —
      unblocks v5 deletion prereqs and the snapshot chain).
- [ ] B5 bump levels (patch vs minor) after CHANGELOG Added-check — resolved at
      cut time, flagged to user only if it contradicts the proposal.
- [ ] B7 transport final tags now vs after v5 code lands (must precede T08
      deletion regardless).
- [ ] Dead tags `event/v4/eventtest/v4.0.0` + `v4.2.0`: module path has no
      `/vN` suffix, so Go rejects them (`go mod download` proof, 2026-08-27).
      Delete remotely or document-and-ignore; consumers stay on v0.3.0.
