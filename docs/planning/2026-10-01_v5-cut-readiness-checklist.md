# v5 Cut — Readiness Checklist (every v5-gated row, dependency-ordered)

> Status: LIVING PLAN until the cut, then a dated record. Do not execute here —
> execution happens ON the v5 branch per [ADR-0123](../adr/0123-v5-unification-single-composition-root.md).
> Sources: TODO_LIST "v5 Unification" section (2026-10-01 state), G-T14,
> T19–T21, ADR-0139, `docs/WIRE-FORMAT-KEYS.md`, `docs/V5-MIGRATION-GUIDE.md`.
> The v5-REMOVAL CENSUS is gate machinery (`cmd/cqrs-lint/pkg/rules/version/v007_tables.go`,
> bidirectional drift tests) — this checklist references it, never duplicates it.

## Layer 0 — Owner rulings that gate implementation (PRE-CUT, blocking)

| Ruling                                                                                                                                                | Blocks         | State 2026-10-01                                                |
| ----------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- | --------------------------------------------------------------- |
| ADR-0139 encryption-at-rest: 4 open questions (provider call semantics, reference validation timing, read-model scope, plaintext→encrypted migration) | Layer 12       | pending (M15.2 pack re-asks)                                    |
| Sweep §4(a): SQL `events`/`commands` column renames — v5.x expand-contract vs v5.0 cut                                                                | Layer 13       | recommendation on table (expand-contract); owner ruling pending |
| ADR-0146 SingleWriter / ADR-0147 Direction                                                                                                            | Layer 10 scope | pending (M15.2 pack)                                            |

## Layer 1 — Pre-cut verifications (v4.x, quiet-window)

- [ ] Live MySQL/MariaDB snapshot-migration run — `integration/snapshot_migration_mysql_integration_test.go` is ready; gate on `#integration-mysql-nspawn`/`-vm` in a quiet window (T18 tail).
- [ ] Golden/meta-tests pass: api-stability `TestEvery*` green; extended-review E-items' "golden/meta-tests pass + cut" tail.
- [ ] `nix run .#verify` green on master at the last v4.x tag.

## Layer 2 — Pre-cut call-site migrations (v4.x-safe, do BEFORE the branch)

- [x] Type-driven status in `listing` replacing the `DetectTombstone` call (via `listing.StatusMiddleware(deleteTypes, rebirthTypes)` or a `StatusOf(Type)` hook). DONE in-tree 2026-10-01: `listing.StatusClassifier`/`WithStatusClassifier`/`ClassifyLast` are live (listing/status.go), `buildRefs` classifies by last-event type (listing/in_memory.go:164); the only remaining `DetectTombstone` references are the deprecated API itself (event/tombstone.go, removed Layer 4), deliberate legacy-equivalence tests (listing/status_test.go:160), and the cqrs-lint v007 removal census.
- [x] Migrate `example/taskmanager` off `stack.Materialize.OnTombstone`/`OnRebirth` to the domain-event style (branch on `evt.Type()` in `OnUpdate`). DONE in-tree 2026-10-01: taskmanager already uses the `task.deleted` domain event (events.go:28, decider.go:111) with zero `OnTombstone`/`OnRebirth` references anywhere under example/; the only remaining users are stack's own bridge-pinning test (stack/materialize_tombstone_bridge_test.go).
- [x] `record.NewStreamRef` empty-entityID rejection: audit + migrate in-repo call sites. AUDITED 2026-10-01: boundary call sites (example/, transport/) pass typed `id.StreamID`/command stream IDs — no site constructs refs from unvalidated raw strings; `record.NewStreamRefOrZero` (record/record.go:178) already exists as the empty-explicit escape hatch. No v4.x migration needed; the signature change itself stays Layer 9.
- [ ] Consumer grep for old wire strings in sibling alert/dashboard configs (sweep §4(b)) — run once more at the cut.

## Layer 3 — Branch mechanics

- [ ] Create the v5 branch from the last green v4.x tag; module-path bumps `v4`→`v5` across all 98 `go.mod` files (+ workspace `go.work`, flake `testModules`, api-stability `modules`, module-layers/budget lists, cqrs-lint catalog meta-test — the new-module gate sweep in reverse).

## Layer 4 — Independent small deletions (no cross-deps)

- [ ] `storage/sql.BuildWhereClause` (XS; `BuildWhereClauseChecked` is the replacement).
- [ ] ADR-0126 shells (S): `schema.VersionedStore` + `NewVersionedStore`, `signing.Rejecting*` forwarders, `encryption.ErrInnerStoreNot*` aliases, `metadata.CustomData`.
- [ ] Deprecated `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/`TombstoneStatus`/`Metadata.Tombstone` — REQUIRES Layer 2 rows done (ADR-0114 completion).

## Layer 5 — Projection deletion cascade (auto-projection must cover every consumer)

Order within the layer matters: each deletion must leave `system.New` consumers whole.

- [ ] `stack.Materialize` (S) — auto-projection replaces it.
- [ ] `storage.RelationalProjection` + `storage/view` SQLViewStore (M) — multi-collection batch atomicity + auto-projection replace them; kills the remaining `aggregate_*` SQL surfaces.
- [ ] `graph.GraphProjection` (S) — auto-projection + graphadapter replaces it.

## Layer 6 — Composition-root consolidation

- [ ] Delete `stack.Bundle` + all 8 stack presets + `stack/` entirely (incl. `stack/bench`, `stack.RunProjections`→`projectionhost.Host`) — `system.System` is the only composition root (ADR-0123).
- [ ] `systemtest` tag-wave tail: strip the module's sibling replaces (XS, at tag time).

## Layer 7 — Transport deletions

- [ ] Delete `transport/http` + `transport/grpc` (ADR-0127): drop from go.work/flake/api-stability/catalog, then delete. Do this BEFORE anyone renames their proto fields (sweep §4).

## Layer 8 — Breaking core-type change

- [ ] `record.NewStreamRef(streamType, entityID) (StreamRef, error)` rejecting empty entityID (owner-confirmed 2026-08-22, decision memo Appendix B). After Layer 2 call-site migration this is a regen + verify.

## Layer 9 — Universal Engine fold (the big one; scope per ADR-0146/0147 rulings)

- [ ] T19–T21: fold capabilities into the universal `Engine`, delete the duplicate SQL stacks, run the release train (ADR-0142 §decision; growing core interfaces is the v5 license).

## Layer 10 — Behavior flips that are breaking by definition

- [ ] **Scan-default flip (G-T14, Option C ruled 2026-09-21):** flip the built-in 100 limit to unbounded ON THIS BRANCH; keep `metaengine.WithDefaultLimit` as the operator ceiling and cqrs-lint F031 `scan-without-limit` as the nudge; update godoc + FAQ rows that carry the documented-100 status quo. Explicit `WithLimit` semantics unchanged.
- [ ] Delete the duplicate SQL stacks (with Layer 9).

## Layer 11 — Encryption-at-rest implementation (after ADR-0139 rulings)

- [ ] `DriverConfig.Encryption` + `KeyProvider func(ctx) ([]byte, error)` + `system/` DeploymentConfig key-reference slot; engines fail construction loudly when unable to honor. Sets the precedent for pg/mysql password providers.

## Layer 12 — Docs and guides at the cut

- [ ] Expand `docs/V5-MIGRATION-GUIDE.md`: before/after per v1 tier (incl. `relational → metaengine`), envelope-v2 consumer note, snapshot-migration operator verification snippets; sweep asrecord/MIGRATION_TO_STACK/PRESETS guides.
- [ ] Sweep §4 remainder: (c) `listing.aggregate_projection` collection-name rename (TBD → decide here).
- [ ] CHANGELOG v5.0.0 section, README, SKILL.md + references refresh (the skill mirrors the post-cut surface; run the doc-check + recipes gates).

## Layer 13 — The cut

- [ ] Full `nix run .#verify` + `nix run .#vulncheck` + integration suites (all backends) on the v5 branch.
- [ ] Tag v5.0.0 for all modules (tag-wave mechanics; proxy propagation check per go-release discipline).

## Readiness verdict (2026-10-01)

Blocked on: Layer 0 rulings (3 items) + Layer 1 quiet-window legs (1 item) +
Layer 2 migrations (2 items) + Layer 9 scope ruling. Everything else is
mechanics with existing gate coverage (v007 census, api-stability, doc-check).
The census gate means no v5 deletion can be forgotten: every `Deprecated:`
marker is held against the removal tables bidirectionally.
