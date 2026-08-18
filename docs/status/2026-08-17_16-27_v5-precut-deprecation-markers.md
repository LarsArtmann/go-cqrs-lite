# Status: v5 Pre-Cut Deprecation Markers (ADR-0123 Phase 8 preparation)

**Date:** 2026-08-17 16:27
**Task:** Mark ALL v5-deletion-target APIs as `Deprecated:` (from TODO_LIST "v5
Unification Phase 8: Deletion + Cut") WITHOUT deleting anything, so v4 consumers
get compile-time warnings (SA1019) and godoc banners before the v5 breaking cut.
Explicitly sanctioned by ADR-0123 Migration Path step 2: _"v4.x+1: mark v1 tiers
as deprecated."_

---

## Scope executed this session

### Deprecation markers applied (uniform phrase: `Deprecated: removed in v5 (ADR-0123): <replacement>`)

| Area                                                                        | Marked symbols                                                                                                                                                                                                                                                                                                        |
| --------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `stack/` (package)                                                          | package doc (whole module dies at v5)                                                                                                                                                                                                                                                                                 |
| `stack/bundle.go`                                                           | `Bundle` type, `New`                                                                                                                                                                                                                                                                                                  |
| `stack/materialize.go`                                                      | `Materialize`, `TombstonePolicy` + 3 constants                                                                                                                                                                                                                                                                        |
| `stack/accessors.go`                                                        | `NewMaterialize`                                                                                                                                                                                                                                                                                                      |
| `stack/run_projections.go`                                                  | `(*Bundle).RunProjections`                                                                                                                                                                                                                                                                                            |
| 8 presets (`memory, sqlite, pebble, bbolt, postgres, mysql, turso, duckdb`) | package docs (doc.go; bbolt inline) + `New`; plus `pebble.Bundle`, `turso.Bundle`, `turso.NewSync`                                                                                                                                                                                                                    |
| `storage/view` (package)                                                    | package doc + `SQLViewStore`, `ViewMapper`, `ViewColumn`, `IndexSpec`, `ViewStoreOption`, `NewSQLiteViewStore`, `NewSQLViewStore`, `NewViewStoreWithDialect`, `AutoMapper`, `AutoMapperWithTombstone`, `WithoutViewAutoMigrate`                                                                                       |
| `storage/view_aliases.go` (facade re-exports)                               | all 5 type aliases + all 7 functions                                                                                                                                                                                                                                                                                  |
| `storage/relational` (package)                                              | package doc + `RelationalHandler`, `RelationalProjection`, `RelationalProjectionOption`, `NewRelationalProjection`, `WithoutRelationalAutoMigrate`, `Row`, `ProjectionSink`, `SetExpr`, `RelationalStore`, `NewRelationalStore`, `RelationalSchema`, `RelationalTable`, `RelationalColumn`, `IndexSpec`, `UniqueSpec` |
| `graph/projection.go`                                                       | `Handler`, `GraphProjection`, `ProjectionOption`, `WithSchema`, `NewGraphProjection`                                                                                                                                                                                                                                  |
| `record/record.go`                                                          | `NewStreamRef` — NOT deprecated (survives v5); added v5-breaking-change NOTE pointing to `Validate()`                                                                                                                                                                                                                 |

**Deliberately NOT marked (survive v5):** `graph.GraphSink`, `graph.GraphDriver`,
`graph.Schema` internals (graphadapter — the v5 replacement — uses them);
`storage.StreamProjection`/`AggregateProjection` (listing projection, not in the
Phase 8 list); `stack.sqlopt`, `stack/contracttest`, `stack/bench` (not named in
Phase 8); `stack` durability types, `With*` options, `MultiCloser`/`FuncCloser`,
accessor methods (see open questions).

### Supporting changes

- **`.golangci.yml`**: one new SA1019 exclusion keyed to the uniform phrase
  (`text: 'SA1019: .*removed in v5.*'`), scoped to
  `(stack|storage|graph|benchkit|cmd/cqrs-bench|example|integration)/.*\.go$` —
  internal callers stay quiet until the cut; every OTHER deprecation stays loud.
  `nix run .#check-lint-config` passes.
- Verified `cmd/cqrs-lint` stack references are string data (module catalog), not
  imports — no exclusion needed there.
- Verified `system/` does NOT import stack — the v5 composition root is clean.

### Verification state

- Builds GREEN (`GOWORK=off`, `-tags goexperiment.jsonv2`): all 10 stack modules,
  storage, graph, record.
- One self-inflicted multiedit collision (duplicated `package relational`
  clause in `storage/relational/projection.go`) — caught and fixed immediately;
  build green after.

---

## a) Fully done

1. Research: ADR-0123 alignment, full export inventory of every target module,
   internal-usage map (benchkit, cqrs-bench, example/getting-started,
   integration tests, storage facade), lint-config impact analysis.
2. All deprecation markers listed above (~60 symbols across 12 modules).
3. Lint exclusion rule + config validation.

## b) Partially done

1. **Lint verification**: spot-run of stack/storage/graph/record exits 3 —
   NOT a findings failure. Root cause (known gotcha, AGENTS.md): manual
   golangci-lint run saw the workspace `go.work` requiring go ≥ 1.26.6 while
   its embedded toolchain is 1.26.5 with GOTOOLCHAIN=local. Zero SA1019 lines
   reached the log; fix is `GOWORK=off` (the flake `#lint` app does this
   implicitly). Not yet re-run.
2. **Method-level markers**: only types + constructors are marked. Calls like
   `mat.View()`, `bundle.Repository()`, `viewStore.Get()`, `relStore.Query()`
   do NOT individually trigger SA1019 (staticcheck warns on identifier use;
   deprecated-type methods aren't automatically deprecated). Decision pending
   (question 1).

## c) Not started

1. api-stability golden regen (expected no-op: comments only, but unverified —
   if the golden captures docs it WILL change).
2. `nix fmt` (golines could reflow long doc lines).
3. Module test suites (stack, storage, graph, record).
4. CHANGELOG `[Unreleased]` Deprecated section.
5. Skill reference updates (`.agents/skills/go-cqrs-lite/references/*.md`
   recommend stack presets/Materialize/RunProjections/relational/view heavily).
6. AGENTS.md module-map annotations for the deprecated modules.
7. TODO_LIST Phase 8 annotation ("deprecation markers landed 2026-08-17").
8. `nix run .#verify` / `#verify-fast` end-to-end gate.
9. doc-check gate.
10. Example/benchkit migration off stack (v5 prep, larger effort).

## d) Totally fucked up?

Nothing damaged, nothing deleted, no reverts. Honest fuckups this session:

1. **Three fumbled lint attempts before the useful one**: (a) shell glob
   matched a wrong/old golangci binary → "unknown flag: --config"; (b) v2
   binary wants `-c`, not `--config PATH` (works, but…); (c) relative config
   path `../../.golangci.yml` from module dir overshoots the repo root. Then
   (d) the correct invocation still failed on the go.work toolchain gotcha —
   I should have remembered `GOWORK=off` from AGENTS.md on attempt one.
2. **Multiedit collision** duplicating `package relational` (fixed in-line,
   caught by immediate build — the process worked as intended).
3. Waited too long to inspect the exit=3 logs; the diagnostic was one
   `head` away for two tool rounds.

## e) What could be improved / what I forgot

- Forgot the golden-regen-when-touching-exports reflex until late (doc-only
  changes probably don't affect it, but "probably" is not verified).
- Did not run module tests immediately after each module's edits (build-only).
- Did not run `nix fmt` before verifying — doc comments I added are short lines,
  but the discipline is format-then-verify.
- Method-level deprecation ambiguity should have been resolved BEFORE the
  60-symbol sweep (see questions).
- The lint spot-check should have reused the flake's exact invocation
  (`GOWORK=off` + absolute config path) from the start.

## f) Next steps (ordered, ~35 concrete)

1. Re-run spot lint with `GOWORK=off` on stack/storage/graph/record; confirm
   zero SA1019 leakage from the new markers.
2. Decide + apply method-level markers if wanted (Materialize.View/List/
   HandlerFunc, Bundle accessors, SQLViewStore methods, RelationalStore
   methods, GraphProjection.Handle/Name/EventTypes, pebble/turso Bundle
   methods).
3. Decide fate of `stack.With*` options, `DurabilityTier`+`WithDurability`,
   `MultiCloser`/`FuncCloser`, `Capabilities`, health/shutdown/debug exports
   (module is deleted entirely at v5 → arguably mark everything).
4. Regen/verify api-stability golden: `cd cmd/api-stability && GOWORK=off go
   run -tags "goexperiment.jsonv2" . --update` + meta-tests.
5. `nix fmt` then re-verify.
6. Module tests: stack (+presets), storage, graph, record.
7. `nix run .#lint` full gate (all modules, ~minutes).
8. `nix run .#verify-fast`, then exclusive `nix run .#verify`.
9. CHANGELOG `[Unreleased]` → `### Deprecated` entry listing the modules.
10. Skill refs: add deprecation notes — `references/core.md` (decision matrix),
    `references/recipes.md` (preset/ES recipes), `references/readmodels.md`
    (relational/view tiers), `references/advanced.md`, `references/modules.md`,
    `references/faq.md`.
11. Run doc-check over SKILL.md + references.
12. AGENTS.md module map: annotate deprecated modules/rows.
13. TODO_LIST Phase 8: check off "mark as deprecated" sub-step; note the
    v4.x+1 ADR migration-path step is satisfied.
14. **Investigate Phase 8 blocker discovered this session**: `storage/pebble`
    imports `stack.DurabilityTier` in production (`backend.go`) — deleting
    `stack/` at v5 requires re-homing the durability types (likely into
    `system/` or `storage/pebble`), else the cut breaks storage/pebble.
15. Same class: `benchkit` imports `stack/contracttest` and `stack/bench`;
    `cmd/cqrs-bench` factories import presets + durability — all need a v5
    migration path (bench harness re-homing).
16. `example/getting-started` still on stack presets — migrate to
    `system.New` (it is the deployer-first demo; keeping it on deprecated API
    post-deprecation is bad optics).
17. Consider a cqrs-lint coaching rule (F-series, like F030 for transport) for
    the newly deprecated imports.
18. Migrate `integration/` tests off `GraphProjection` where trivial.
19. `nix run .#check-duplication` (doc-comment blocks are clones by nature;
    `.art-dupl` operates on code, but verify no gate trips).
20. Tag v4.x patch releases for stack + all presets + storage + graph + record
    so CONSUMERS actually receive the deprecation warnings (uncommitted markers
    help nobody).
21. Consumer-pin sweep per AGENTS.md gotcha (codec-critical methods rule) —
    not applicable here (no codec methods), skip unless verify says otherwise.
22. Plan `record.NewStreamRef` v5 signature-change call-site sweep
    (`Validate()` adoption) as its own task before the cut.
23. Re-check that the api-stability golden's `modules` list is untouched
    (no module additions/removals this session).
24. After all gates green: full `nix run .#verify` exclusive run, then leave
    the tree for the auto-commit daemon and re-build once (daemon gotcha).
25. Draft the v5 migration guide skeleton (Phase 8, Effort L) referencing the
    new godoc deprecation notes.
26. Decide whether `stack/bench` module gets its own deprecation (dies with
    presets; currently unmarked).
27. Sweep `docs/` + `README.md` for recommendations of now-deprecated APIs.
28. Verify godoc rendering of package-level deprecations
    (`go doc github.com/larsartmann/go-cqrs-lite/stack/v4`).

## g) Questions I cannot answer myself

1. **Method-level markers?** "Mark them ALL" — do you want EVERY method on the
   deprecated types (≈60 additional markers: `Bundle` accessors + `With*`
   options, `Materialize.View/List/HandlerFunc`, `SQLViewStore.*`,
   `RelationalStore.*`, `GraphProjection.*`, preset `Bundle` methods)? That is
   the only way SA1019 fires on every consumer call site; type+constructor
   marking (current state) only warns on construction/type references.
2. **Durability re-homing (Phase 8 blocker):** `storage/pebble/backend.go`
   imports `stack.DurabilityTier` in production. When `stack/` is deleted at
   v5, where do the durability tiers live — move to `storage/pebble`, a new
   Tier-0 module, or `system/`? This decides whether I mark them deprecated
   now (with a migration pointer) or leave them unmarked as "moving, not dying".
3. **Release timing:** the markers only reach consumers in a tagged v4.x wave
   (stack + 8 presets + storage + graph + record). Cut that release now (before
   further v5 work), or batch with the transport/http + grpc final patch
   releases already pending in the Phase 8 list?

---

**Working tree:** all changes uncommitted; auto-commit daemon may pick them up.
No deletions, no behavior changes — doc comments + one lint-config rule only.
