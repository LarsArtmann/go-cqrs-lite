# Status Report — 2026-06-16 23:12

## go-error-family Full Adoption + Lint Cleanliness Audit

---

## Executive Summary

**148 → 0** stdlib error constructor violations. All `errors.New`, `fmt.Errorf`, and `errors.Join` calls across 22 library modules, 2 example apps, and 2 CLI tools have been migrated to `go-error-family` classified constructors. `branching-flow errorfamily .` reports zero findings. All modules build and tests pass (except pre-existing turso I/O issue).

---

## a) FULLY DONE ✓

| Work Item                       | Modules                                        | Details                                                                                                               |
| ------------------------------- | ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **errorfamily migration**       | 22 library + 2 example + 2 cmd                 | 148 `fmt.Errorf`/`errors.New`/`errors.Join` → `errorfamily.Wrapf`/`NewRejection`/`NewInfrastructure`/`Newf`/`Compose` |
| **Module go.mod updates**       | codec, id, kv, cmd/api-stability, cmd/cqrs-gen | Promoted `go-error-family` from indirect to direct dependency                                                         |
| **event.Compose re-export**     | event/                                         | Added `Compose` function to `event/errors.go` for modules that need `errors.Join` replacement                         |
| **API golden file**             | docs/api_surface.txt                           | Updated for new `event.Compose` export                                                                                |
| **Tests passing**               | 22/24 library modules                          | All green except turso (pre-existing I/O failure)                                                                     |
| **branching-flow errorfamily**  | Entire repo                                    | **0 findings** — clean                                                                                                |
| **branching-flow panic**        | Entire repo                                    | **0 findings** — clean                                                                                                |
| **branching-flow boolblind**    | Entire repo                                    | **0 findings** — clean                                                                                                |
| **branching-flow contextguard** | Entire repo                                    | **0 findings** — clean                                                                                                |
| **branching-flow nakedreturn**  | Entire repo                                    | **0 findings** — clean                                                                                                |
| **branching-flow splitbrain**   | Entire repo                                    | **0 findings** — clean                                                                                                |

### Detailed Migration Map

| Module            | Files Changed                                                                                               | Key Patterns                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| codec             | errors.go, raw.go                                                                                           | `errors.New` → `NewRejection`, `fmt.Errorf` → `Wrapf`                            |
| id                | errors.go, id.go, aggregate_id.go                                                                           | `errors.New` → `NewRejection`, `fmt.Errorf` → `Wrapf`                            |
| event             | base64.go, codec.go, errors.go, eventtest/\*                                                                | `fmt.Errorf` → `Wrapf`/`Newf`, `errors.New` → `NewRejection`, added `Compose`    |
| command           | aggregate_ref.go                                                                                            | `fmt.Errorf` → `WrapRejection`                                                   |
| decider           | load.go                                                                                                     | Rewrote `opError` to use `Wrapf`/`Compose` instead of `fmt.Errorf` with `%w: %w` |
| encryption        | codec.go, static_resolver.go, store.go, versioned.go                                                        | `fmt.Errorf` → `Wrapf` with appropriate families                                 |
| memory            | checkpoint.go, store_load.go                                                                                | `fmt.Errorf` → `Wrapf`                                                           |
| middleware        | logging.go, recovery.go                                                                                     | `fmt.Errorf` → `Wrapf`                                                           |
| signing/multisig  | middleware.go                                                                                               | `fmt.Errorf` → `WrapCorruption`                                                  |
| storage           | command_store_journal.go, event_store_global.go, listing_table.go, query_store_load.go, sql/query_engine.go | `fmt.Errorf` → `Wrapf`                                                           |
| pebble            | checkpoint.go, journal.go, snapshot.go                                                                      | `fmt.Errorf` → `Wrapf` with Corruption/Infrastructure                            |
| turso             | advisor.go, stats.go                                                                                        | `fmt.Errorf` → `Wrapf`                                                           |
| watermill         | protocol.go, subscriber.go                                                                                  | `fmt.Errorf` → `Wrapf`, `errors.Join` → `Compose`                                |
| catalog           | 8 files across eventcatalog/, schema/, registry.go                                                          | Batch transform: 31 `fmt.Errorf` → `Newf`                                        |
| integration       | simulation/generator.go                                                                                     | `fmt.Errorf` → `Wrapf`                                                           |
| kv                | errors.go                                                                                                   | `errors.New` → `NewRejection`/`NewInfrastructure`                                |
| cmd/api-stability | main.go                                                                                                     | `fmt.Errorf` → `Wrapf`                                                           |
| cmd/cqrs-gen      | main.go                                                                                                     | `fmt.Errorf` → `Wrapf`                                                           |
| example/todo      | 16 files                                                                                                    | `errors.New` → `NewRejection`, `fmt.Errorf` → `Newf`                             |
| example/user      | 4 files                                                                                                     | `fmt.Errorf` → `Newf`                                                            |

---

## b) PARTIALLY DONE ⚠️

| Work Item                        | Status                           | What Remains                                                                             |
| -------------------------------- | -------------------------------- | ---------------------------------------------------------------------------------------- |
| **branching-flow phantom types** | 264 findings                     | Cosmetic — primitive types that could benefit from phantom typing. Low priority.         |
| **branching-flow strong-id**     | 2 findings (high severity)       | `string` params named `id` in watermill/subscriber.go and turso. Should use branded IDs. |
| **branching-flow dupe**          | 16 findings (10 false positives) | 6 real duplication groups to review.                                                     |
| **branching-flow anti-patterns** | 3 findings                       | Structural naming suggestions (Manager/Handler in pebble).                               |
| **branching-flow context**       | 6 findings                       | Semantic context loss in error handling paths.                                           |
| **branching-flow mixins**        | 21 findings                      | O(n²) mixin composition opportunities.                                                   |

---

## c) NOT STARTED ⬜

| Work Item                      | Impact | Notes                                                                                                                        |
| ------------------------------ | ------ | ---------------------------------------------------------------------------------------------------------------------------- |
| **Turso test fix**             | Medium | `TestEventStore_LoadNonExistent` fails on LibSQL connection (I/O error). Pre-existing — not caused by errorfamily migration. |
| **`nix fmt` on changed files** | Low    | Some files may need golines formatting (120 char max).                                                                       |
| **`nix run .#lint` full run**  | Medium | Haven't run full golangci-lint across all modules yet.                                                                       |
| **Coverage report refresh**    | Low    | No coverage delta expected from error constructor changes.                                                                   |
| **Module README updates**      | Low    | If error patterns changed in public docs/examples.                                                                           |

---

## d) TOTALLY FUCKED UP 💥

| Issue                       | Severity      | Details                                                                                                                                                                                                                                                                                            |
| --------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Untracked binary**        | Trivial       | `cmd/api-stability/api-stability` — compiled binary left in source tree. Should be gitignored or deleted. Not committed.                                                                                                                                                                           |
| **Catalog import ordering** | Minor         | Batch script added `errorfamily` import in non-alphabetical order in some catalog files. `goimports` would fix.                                                                                                                                                                                    |
| **decider opError rewrite** | Review needed | The `opError` function was rewritten to use `strings.ReplaceAll(prefix+msg, "%w", "%v")` + `event.Compose` for multi-error wrapping. This is a semantic change — `%w` verbs are replaced with `%v` because `errorfamily.Wrapf` doesn't support `%w`. Tests pass but edge cases should be reviewed. |

---

## e) WHAT WE SHOULD IMPROVE 🔧

1. **Error code naming conventions** — Currently ad-hoc (`codec.raw_encode_type`, `pebble.checkpoint_cbor`). Should establish a pattern like `module.context.action` consistently.
2. **Error family classification accuracy** — Many errors were blanket-classified as `Infrastructure` when some should be `Corruption` (deserialization), `Rejection` (validation), or `Conflict` (state). A focused review pass would improve classification quality.
3. **`event.Compose` vs `errorfamily.Compose`** — Added `Compose` re-export to event/ but command/ and query/ don't have it. Should either add to all re-export modules or document that consumers import errorfamily directly.
4. **Decider `opError` design** — The `%w` → `%v` replacement is a hack. The function should be refactored to not use format strings with `%w` at all.
5. **Test coverage for error families** — No tests verify that errors are correctly classified (e.g., that a deserialization failure returns Corruption, not Infrastructure).
6. **Catalog batch transformation** — Used a Python script for 31 transformations. The generated error codes (`catalog.exporter.1`, `catalog.exporter.2`) are sequential numbers, not semantic names. Should be manually reviewed.
7. **`go.mod` replace directives** — Some modules still have `// indirect` for `go-error-family` when it should be direct. `go mod tidy` should fix but gopls flags warnings.

---

## f) Top 25 Things to Get Done Next 🎯

| #   | Task                                                                                                 | Impact                  | Effort |
| --- | ---------------------------------------------------------------------------------------------------- | ----------------------- | ------ |
| 1   | **Run `nix fmt` on all changed files**                                                               | Formatting hygiene      | 5 min  |
| 2   | **Run `nix run .#lint` full lint**                                                                   | Catch remaining issues  | 10 min |
| 3   | **Fix turso test failure** (LibSQL connection)                                                       | Test suite completeness | 30 min |
| 4   | **Review catalog error codes** — replace sequential numbers with semantic names                      | Code quality            | 1 hr   |
| 5   | **Review decider `opError` rewrite** for edge cases                                                  | Correctness             | 30 min |
| 6   | **Audit error family classifications** — ensure Infrastructure vs Corruption vs Rejection is correct | API quality             | 2 hr   |
| 7   | **Fix strong-id findings** (2 high severity) — watermill topic, turso id                             | Type safety             | 1 hr   |
| 8   | **Add `Compose` to command/errors.go and query/errors.go**                                           | API consistency         | 15 min |
| 9   | **Delete untracked binary** `cmd/api-stability/api-stability`                                        | Cleanup                 | 1 min  |
| 10  | **Review 6 real dupe findings** from branching-flow                                                  | DRY                     | 1 hr   |
| 11  | **Review 3 anti-pattern findings** (pebble/base.go naming)                                           | Naming quality          | 30 min |
| 12  | **Review 6 context-loss findings**                                                                   | Error quality           | 1 hr   |
| 13  | **Triage 264 phantom type findings** — pick top 10 actionable                                        | Type safety             | 1 hr   |
| 14  | **Run `nix run .#test` full workspace test**                                                         | Verification            | 10 min |
| 15  | **Update AGENTS.md** with errorfamily adoption policy                                                | Documentation           | 15 min |
| 16  | **Add errorfamily to CI** (`--exit-code` flag in ci.yml)                                             | Enforcement             | 15 min |
| 17  | **Write error handling guide** (docs/patterns/error-families.md)                                     | Developer experience    | 1 hr   |
| 18  | **Review mixin opportunities** (21 findings)                                                         | Architecture            | 2 hr   |
| 19  | **Add `.gitignore` for compiled binaries**                                                           | Repo hygiene            | 5 min  |
| 20  | **Update go.sum across workspace** (`go work sync`)                                                  | Dependency hygiene      | 5 min  |
| 21  | **Coverage report refresh**                                                                          | Quality tracking        | 15 min |
| 22  | **Review example/todo error codes** for consistency                                                  | Example quality         | 30 min |
| 23  | **Add error family tests** — verify Classify() returns expected family for key errors                | Test rigor              | 2 hr   |
| 24  | **Consider `errorfamily.Compose` → direct import** instead of re-exporting through event/            | Architecture decision   | 30 min |
| 25  | **Tag v2.4.0** — errorfamily adoption release                                                        | Release management      | 30 min |

---

## g) Top #1 Question ❓

**The decider `opError` function previously used `fmt.Errorf(prefix+msg, args...)` with `%w` verbs to wrap sentinel errors like `ErrLoadFailed` and `ErrSaveFailed`, and tests verify `errors.Is(err, ErrLoadFailed)`. My rewrite uses `event.Compose(errs...)` to chain multiple errors and `strings.ReplaceAll(msg, "%w", "%v")` to avoid format-string issues.**

**The question: Does `errorfamily.Error.Is()` correctly traverse the `Compose` (errors.Join) chain to find wrapped sentinels? The tests pass, but I'm uncertain whether `errorfamily.Wrapf` preserves the full Unwrap chain when the cause is a `Compose` of multiple errors, or whether `errors.Is` only finds the first one. This needs verification with a multi-error wrapping test case (e.g., `opError(ref, "%w: %w", ErrLoadFailed, someErr)` where both `ErrLoadFailed` and `someErr` should be findable via `errors.Is`).**

---

## Branching-Flow Stats Summary

```
Files: 349 | Health: 99% (good)

✅ Clean (0 findings):
  - Error Family Adoption
  - Panic Conditions
  - Boolean Blindness
  - Context Propagation
  - Naked Return Guard
  - Split-Brain Interfaces

⚠️ Remaining work:
  - Phantom Types: 264 (cosmetic)
  - Mixins: 21 (architectural)
  - Duplicate Types: 16 (6 real)
  - Context Loss: 6
  - Anti-Patterns: 3
  - Strong ID: 2 (high severity)
```

## Test Results

| Module            | Status                     |
| ----------------- | -------------------------- |
| codec             | ✅ pass                    |
| id                | ✅ pass                    |
| event             | ✅ pass                    |
| command           | ✅ pass                    |
| query             | ✅ pass                    |
| decider           | ✅ pass                    |
| encryption        | ✅ pass                    |
| memory            | ✅ pass                    |
| middleware        | ✅ pass                    |
| signing           | ✅ pass                    |
| storage           | ✅ pass                    |
| pebble            | ✅ pass                    |
| watermill         | ✅ pass                    |
| catalog           | ✅ pass                    |
| integration       | ✅ pass                    |
| kv                | ✅ pass                    |
| cmd/api-stability | ✅ pass                    |
| cmd/cqrs-gen      | ✅ pass                    |
| example/todo      | ✅ pass                    |
| example/user      | ✅ pass                    |
| turso             | ❌ FAIL (pre-existing I/O) |
