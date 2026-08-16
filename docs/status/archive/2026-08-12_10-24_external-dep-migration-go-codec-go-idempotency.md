# Status Report: External Dependency Migration (go-codec, go-idempotency, go-retry, go-flightrecorder)

**Date:** 2026-08-12 10:24
**Session scope:** Wiring published external packages (`go-codec@v0.1.0`, `go-idempotency@v0.1.2`, `go-retry@v0.3.1`, `go-flightrecorder@v0.2.0`) into the go-cqrs-lite workspace.

---

## A) FULLY DONE

### 1. go-codec v0.1.0 workspace wiring

- **`go.work`** — Removed `replace github.com/larsartmann/go-codec => ../go-codec` directive and its multi-line comment. Added `../go-codec` to the `use` block (same pattern as `../go-retry`, `../go-idempotency`).
- **`codec/alias.go`** — Fixed compilation error: renamed `COSESign1String` → `COSESign1Diagnostic` and `COSEEncrypt0String` → `COSEEncrypt0Diagnostic` to match go-codec's actual v0.1.0 API (the `*String` names never existed in go-codec — they were renamed to `*Diagnostic` before extraction but the shim wasn't updated).
- **`docs/api_surface.txt`** — Regenerated golden file (4097 exports, was 4097 before — the rename was 1:1).
- **`go mod tidy`** — Ran on all 30+ modules that depend on go-codec. All clean.

### 2. go-idempotency import migration (15 .go files)

- Migrated all source imports from `github.com/larsartmann/go-cqrs-lite/idempotency/v4` → `github.com/larsartmann/go-idempotency` in:
  - `middleware/idempotency.go` + 4 test files
  - `idempotency/kvstore/store.go` + 3 test files
  - `idempotency/sqlstore/store.go` + 2 test files
  - `integration/idempotency_test.go`
  - `example/taskmanager/setup.go` + 1 test file
- Package name `idempotency` is identical in both modules — pure import path swap, zero code changes needed.
- `go mod tidy` on all affected modules. Direct deps now reference `go-idempotency v0.1.2`.

### 3. cqrs-linter updates (3 files)

- **`module_catalog_data.go`** — Added external ImportHints for all three deprecated modules: `go-flightrecorder`, `go-idempotency`, `go-retry` (matching the existing `go-codec` pattern).
- **`scan_in.go`** — Made `importsPathIn()` variadic (was single-suffix, now accepts `suffixes ...string`).
- **`f007_f008.go`** — F007 detector now recognizes both `go-cqrs-lite/idempotency` AND `go-idempotency` as valid imports.

### 4. go-retry & go-flightrecorder — already clean

- Zero internal source imports found — both were already wired correctly (in `go.work` use block, no replace directives, all source files already importing the external packages directly). No action needed.

### 5. Verification passed

- Workspace `go build` + `go vet`: clean
- Per-module `GOWORK=off` builds: `codec`, `kv`, `idempotency/kvstore`, `idempotency/sqlstore` all fetch v0.1.0/v0.1.2 from proxy correctly
- API surface: 4097 exports verified
- Doc-check: 779 references valid across 44 packages
- Tests: all pass for middleware, idempotency, kvstore, sqlstore, integration, cqrs-lint, example/taskmanager, event, encryption, signing, decider, snapshot, command, query, schema, stack, storage, transport, watermill, projectionhost

---

## B) PARTIALLY DONE

### 1. Backward-compat aliases for renamed COSE diagnostic functions

**STATUS: USER EXPLICITLY ASKED — NOT DONE. SEE SECTION D.**

### 2. Indirect go.mod references to deprecated modules

- Many modules still have `go-cqrs-lite/codec/v4`, `go-cqrs-lite/idempotency/v4`, `go-cqrs-lite/retry/v4`, `go-cqrs-lite/flightrecorder/v4` as **indirect** deps in their go.mod files. These come from published tags of transitive internal deps (e.g., `record/v4@v4.1.0` still requires `idempotency/v4`). They will clean up as new module versions are published. This is expected and not actionable now.

### 3. cqrs-lint F008 detector (CBOR coaching)

- F008 checks for `codec.JSONCodec` / `codec.CBORCodec` selectors (package name `codec`). If consumers import `go-codec` directly (package name `codec` — same name), the selector check still works. No change needed, but wasn't explicitly verified with a consumer that imports `go-codec` directly.

---

## C) NOT STARTED

1. **Publishing new module versions** — No tags were created for any internal module. The import migrations only take effect for consumers once each affected module gets a new tag.
2. **AGENTS.md module map updates** — The descriptions for `codec/`, `retry/`, `idempotency/` already say "DEPRECATED" but could note the external package is now published.
3. **SKILL.md / references/*.md updates** — No skill docs were updated to reflect the external package imports for idempotency consumers.
4. **CHANGELOG.md** — No entries written for this migration.
5. **CONTRIBUTING.md release notes** — Not updated.

---

## D) TOTALLY FUCKED UP

### 1. FORGOT backward-compat aliases — USER EXPLICITLY ASKED AND I SKIPPED IT

The user said:

> "Keep COSESign1String and COSEEncrypt0String for gocodec.COSEEncrypt0Diagnostic and gocodec.COSESign1Diagnostic also!"

I read the file, confirmed the current state, then **immediately moved on to the idempotency migration without doing it.** The aliases are NOT in `codec/alias.go`. Any consumer who was using `codec.COSESign1String` or `codec.COSEEncrypt0String` (which existed in the pre-extraction codec module) will get a compilation error.

**Impact:** Breaking change for any consumer using the old names through the `codec/v4` shim.

**Fix needed:** Add to `codec/alias.go`:

```go
// Deprecated: use COSESign1Diagnostic (renamed in go-codec v0.1.0).
var COSESign1String = gocodec.COSESign1Diagnostic

// Deprecated: use COSEEncrypt0Diagnostic (renamed in go-codec v0.1.0).
var COSEEncrypt0String = gocodec.COSEEncrypt0Diagnostic
```

---

## E) WHAT WE SHOULD IMPROVE

~~1. **User instruction tracking** — I should have added the backward-compat alias request to my todo list immediately. Instead I let it get buried by the next task. Every explicit user request goes on the todo list or gets done before moving on.~~ done at 6f9199f0c
~~2. **Backward compatibility discipline** — When renaming re-exported symbols, always keep deprecated aliases. The rename from `*String` → `*Diagnostic` happened in go-codec, but the shim's job is backward compat. I should have added the aliases proactively, not waited to be asked — and then I forgot even when asked.~~ done at 6f9199f0c (4099 exports)
~~3. **Session focus** — The user gave three instructions in sequence (go-codec, keep aliases, migrate three deps). I should have tracked all three as todos from the start rather than treating them as independent.~~ done at 6f9199f0c

---

## F) UP TO 50 THINGS TO DO NEXT

### Critical (blocking)

~~1. **Add `COSESign1String` and `COSEEncrypt0String` deprecated backward-compat aliases** in `codec/alias.go`~~ done at 6f9199f0c
~~2. **Regenerate API surface golden** after adding the aliases (will go from 4097 → 4099 exports)~~ done at 6f9199f0c (4099 exports)
~~3. **Verify `go build` and tests pass** after alias addition~~ done at 6f9199f0c

### Should do soon

~~4. **Update AGENTS.md** — Note that `go-codec`, `go-idempotency`, `go-retry`, `go-flightrecorder` are now published and the replace directive is gone~~ done at 5127039da + 2e9a2fc28
~~5. **Run `nix run .#verify`** — Full verification gate (build + vet + test + race + lint + doc-check + doc-assertions) to catch anything missed~~ done. GREEN 3x since (5f2198189)
~~6. **Run `nix run .#check-arch`** — Verify dependency budgets aren't broken by the module changes~~ done (green after spaced-keys fix, 5127039da)
~~7. **Run `nix run .#check-duplication`** — Verify no new duplication clones from the migration~~ done. baseline re-pinned at 875bb689b
~~8. **Run `nix run .#check-coverage`** — Verify coverage didn't drift~~ done. gate repaired at 875bb689b

### Module versioning (enables consumer migration)

9. **Tag new versions of modules with direct go-codec dep** (codec, event, encryption, signing, kv, snapshot, command, query, decider, schema, stack, storage, transport, middleware, watermill, projectionhost, benchkit, integration, system, etc.) <- OPEN. TODO_LIST Release/Tagging
10. **Tag new versions of modules with go-idempotency migration** (middleware, idempotency/kvstore, idempotency/sqlstore, integration, example/taskmanager, commandlifecycle) <- OPEN. TODO_LIST Release/Tagging
    ~~11. **Verify version-sequence correctness** before tagging (monotonic semver + commit ancestry)~~ done (tag-release.sh protocol)
11. **Run `nix run .#vulncheck`** — Per-module standalone build catches version-sequence breaks <- OPEN. TODO_LIST Release/Tagging pre-tag checklist

### Documentation

~~13. **Update SKILL.md references** — Ensure recipes that show `codec/v4` imports also mention `go-codec` direct import~~ done at 5127039da + 2e9a2fc28
~~14. **Update `.agents/skills/go-cqrs-lite/references/core.md`** — If it references deprecated modules~~ done
~~15. **Update `.agents/skills/go-cqrs-lite/references/recipes.md`** — If it references deprecated modules~~ done
~~16. **Update `.agents/skills/go-cqrs-lite/references/modules.md`** — Module lookup table~~ done at 5127039da
~~17. **Update `.agents/skills/go-cqrs-lite/references/faq.md`** — If it references deprecated modules~~ done at 2e9a2fc28
18. **Update CONTRIBUTING.md** — Release notes for external dep extraction completion <- OPEN. minor; fold into the v5 release pass
~~19. **Update CHANGELOG.md** — Document the migration~~ done at 5127039da (ADR-0128 entry)

### Linter enhancements

~~20. **Update F008 detector** — Consider recognizing `go-codec` direct imports for CBOR coaching (currently checks for `codec` package selector — works by coincidence since package name matches)~~ NOT-DO. package name codec is identical; selector check works unchanged
~~21. **Update C026 detector** — `idempotency.NewMemoryStore` detection works on package name `idempotency` — verify it still works when imported from `go-idempotency` (package name is the same, so likely fine)~~ done (same package name; verified by lint gates)
~~22. **Update A016 detector** — Same package-name-based detection, verify works with external import~~ done (same)
23. **Add cqrs-lint coaching test** — Test that F007 recognizes `go-idempotency` import as satisfying the idempotency check <- OPEN. cqrs-lint coaching-tests TODO item covers it
24. **Add cqrs-lint coaching test** — Test that module catalog recognizes `go-retry` and `go-flightrecorder` imports <- OPEN. cqrs-lint coaching-tests TODO item covers it

### Cleanup

25. **Audit remaining indirect `go-cqrs-lite/codec/v4` references** — Will naturally clean up as modules are re-tagged, but track <- OPEN. TODO_LIST Release/Tagging (indirect refs, ~49 files)
26. **Audit remaining indirect `go-cqrs-lite/idempotency/v4` references** — Same <- OPEN. same item
27. **Audit remaining indirect `go-cqrs-lite/retry/v4` references** — Same <- OPEN. same item
28. **Audit remaining indirect `go-cqrs-lite/flightrecorder/v4` references** — Same <- OPEN. same item
    ~~29. **Consider deprecation timeline** — When can the `codec/`, `retry/`, `idempotency/`, `flightrecorder/` shim modules be deleted entirely?~~ done. deleted 2026-08-14 at 5127039da (ADR-0128)

### Examples

~~30. **Update `example/taskmanager/codec_init.go`** — Verify it uses `go-codec` directly (it does, but double-check)~~ done
~~31. **Update example go.mod files** — Ensure all examples reference external packages directly~~ done

### Testing

32. **Run per-module `GOWORK=off` tests** on all go-codec-dependent modules after tagging <- OPEN. gated on the release chain
    ~~33. **Run integration tests** (`nix run .#test-integration`) to verify end-to-end~~ done (verify gates since)
    ~~34. **Run soak tests** — Verify the migration didn't break long-running scenarios~~ done
    ~~35. **Run race tests** — `go test -race` on affected modules~~ done

### Pre-existing issues noticed (not caused by this session)

~~36. **`record/v4@v4.1.0` type mismatch** — `record.go:41` references `id.ActorID` which doesn't exist in the published `id/v4` tag. Causes GOWORK=off build failures in event, encryption, system, middleware. This is a pre-existing published-tag staleness issue.~~ done. record/v4.2.0 + id/v4.4.0 tagged 2026-08-13 (2026-08-13_02-33 report)
~~37. **`command/v4@v4.4.0` type mismatch** — `asrecord.go` has branded type mismatches (`string` vs `CorrelationID`/`CausationID`/`ActorID`). Same pre-existing issue.~~ done. command/v4.6.0 tagged 2026-08-13
~~38. **`event/v4@v4.4.0` type mismatch** — Same pattern in `asrecord.go`.~~ done. event/v4.6.0 tagged 2026-08-13
~~39. **`cmd/cqrs-bench/layout.go:206`** — `enc.SetIndent` undefined on `*jsontext.Encoder` (gopls phantom error or real build tag issue). Pre-existing, unrelated to this session.~~ done. fixed in the 21:36 recovery; lint gates green since

### Meta

~~40. **Update `.art-dupl-baseline.json`** if the `importsPathIn` signature change affected duplication metrics~~ done (baseline re-pinned at 875bb689b)
~~41. **Review `go.work` use block ordering** — `../go-codec` was added at the top of the external deps; verify alphabetical consistency~~ done
42. **Consider adding a meta-test** that verifies no source files import deprecated local modules when external packages exist <- WONT-IMPLEMENT. shims now deleted; nothing to guard
~~43. **Review the `cmd/cqrs-lint/pkg/analyzer/module_catalog_test.go:267`** — It already skips external repos (`go-idempotency, go-retry`); verify it also skips `go-codec` and `go-flightrecorder`~~ done

---

## G) QUESTIONS I CANNOT FIGURE OUT MYSELF

### 1. Should the deprecated shim modules (`codec/`, `retry/`, `idempotency/`, `flightrecorder/`) be deleted now, or kept for a deprecation period?

The shims still compile and work. Deleting them is a breaking change for consumers who haven't migrated yet. But keeping them adds maintenance burden and module count. What's the deprecation policy?

### 2. Should I tag new versions of all affected internal modules right now, or wait for more changes to batch into a single release?

Tagging now makes the migration available to consumers. But if more changes are coming (e.g., the COSE aliases fix), it might make sense to batch them. What's the release cadence preference?

### 3. The pre-existing `record/v4@v4.1.0` → `id/v4` type mismatch breaks GOWORK=off builds for event, encryption, system, middleware. Should I fix this as part of this work, or is it tracked separately?

This blocks per-module standalone builds and will block `nix run .#vulncheck`. It's not caused by this session but it prevents full verification of the migration.

---

## Resolution (2026-08-15)

Superseded the same day by 6f9199f0c (COSE aliases + wiring; the "blocked"
items were the COSE aliases, fixed in the 11:03 close-out report) and then by
ADR-0128 (5127039da): all four shim modules were deleted outright on
2026-08-14, so the entire shim-lifecycle discussion (keep vs delete, F008/C026/
A016 detector edge cases, indirect-ref audits) is moot except for the release
chain, which lives in TODO_LIST -> Release/Tagging. All 43 next-items carry
inline verdicts. The record/v4@v4.1.0 poisoned-tag blockers (36-38) were
resolved by the 2026-08-13 tag chain. Archived.
