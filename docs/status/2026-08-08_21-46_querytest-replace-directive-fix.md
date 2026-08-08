# Status: querytest Replace-Directive Fix & Self-Critique

**Date:** 2026-08-08 21:46
**Session scope:** Fix `querytest.RunStoreSuite` / `querytest.StoreSuite` undefined in storage modules
**Result:** FIXED — all 3 affected modules build and test pass under `GOWORK=off`

---

## a) FULLY DONE

### 1. Diagnosed and fixed `querytest.RunStoreSuite` / `StoreSuite` undefined

**Root cause:** `query/querytest/store_suite.go` was added on master (commit `431c252c9`) but no new `query/v4` tag was cut. The latest tag is `query/v4.2.0`, which predates the file. Three storage modules required `query/v4 v4.2.0`, so CI per-module builds (`GOWORK=off`) couldn't see `RunStoreSuite` / `StoreSuite` / `StoreFactory`.

**Affected modules (3):**
- `storage/memory` — `query_store_test.go` uses `querytest.RunStoreSuite` (NOT mentioned in original bug report, discovered during investigation)
- `storage/pebble` — `query_store_test.go:36` references undefined symbols
- `storage/bbolt` — `query_store_test.go:12` references undefined symbols

**Fix applied:** Added `replace github.com/larsartmann/go-cqrs-lite/query/v4 => ../../query` to all three modules' `go.mod`, following the existing `decider/go.mod` → `flightrecorder` pattern. Ran `go mod tidy` in each module to update `go.sum` (removed stale `query/v4 v4.2.0` hashes, since the replace makes Go resolve locally).

**Verification:**
- `GOWORK=off go test -tags "goexperiment.jsonv2" -count=1 ./...` → PASS (all 3 modules)
- `GOWORK=off go vet -tags "goexperiment.jsonv2" ./...` → PASS (all 3 modules)
- Workspace mode (`go test` with `go.work`) → PASS (all 3 modules)
- `stack/pebble` and `stack/bbolt` → PASS (unaffected, they don't use querytest)
- `storage/turso` → not affected (doesn't use querytest)

### 2. Updated TODO_LIST.md

Marked item as `[x]` with root cause explanation and fix description.

---

## b) PARTIALLY DONE

Nothing partial — the fix is complete and verified for the specific issue.

---

## c) NOT STARTED

### 1. Cut `query/v4.3.0` tag (THE PROPER FIX)

The `replace` directive is a **workaround**. The real fix is to tag `query/v4.3.0` (or higher) so all consumers can drop the replace directives and require the tagged version. This must happen at the next batch release.

**Release steps when ready:**
1. Strip the 3 `replace` directives from `storage/memory`, `storage/pebble`, `storage/bbolt`
2. Bump `query/v4 v4.2.0` → `query/v4 v4.3.0` in all three
3. Tag `query/v4.3.0` (annotated, via `scripts/tag-release.sh`)
4. Run `go mod tidy` in each module
5. Verify with `GOWORK=off go test`

### 2. CI check for pre-tag symbol drift

There should be a CI gate that catches "consumer references symbols that don't exist in the required version tag" — essentially `GOWORK=off go build ./...` across all modules, which is what CI already does per-module. The issue is that the auto-commit daemon or manual commits can add test helpers to source modules without tagging, silently breaking consumers.

### 3. AGENTS.md lesson update

The lesson "when adding exported symbols to a shared test helper package, you MUST tag a new version before consumers can use them in GOWORK=off mode" should be encoded in AGENTS.md. This is the inverse of the existing "Verify module version exists before requiring it" lesson — the lesson from the PRODUCER side.

---

## d) TOTALLY FUCKED UP

### 1. Nothing in this session

The fix is correct and follows the established pattern. However:

### 2. Pre-existing systemic issue: UNTAGGED SHARED TEST HELPERS

The root cause is a **process failure**: `store_suite.go` was added to `query/querytest/` in commit `431c252c9` (Jul 2026), consumers (`storage/memory`, `storage/pebble`, `storage/bbolt`) were updated to use it, but **nobody tagged a new `query/v4` version**. This went unnoticed because:
- Workspace mode (`go.work`) resolves everything locally → tests pass
- `nix run .#test` uses workspace mode → tests pass
- Only `GOWORK=off` per-module CI builds catch this

The same pattern **almost** happened with `command/v4` — `command/v4.4.0` WAS tagged with `store_suite.go`, so `commandtest.RunStoreSuite` works. But `query/v4` was left at `v4.2.0`. This asymmetry suggests a batch-tagging process that missed the `query` module.

### 3. The `scripts/ephemeral-dgraph.sh` change appeared in git diff

A change to `scripts/ephemeral-dgraph.sh` (adding Alpha health endpoint polling instead of `sleep 2`) appeared in `git diff --stat` but was NOT made by this session. It's from the auto-commit daemon or a prior session. I correctly left it untouched.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Tag source modules when adding exported symbols** — When you add `RunStoreSuite`/`StoreSuite` to `query/querytest/`, you MUST tag `query/v4.3.0` before updating consumers. The auto-commit daemon can bump consumer go.mod files but cannot tag.

2. **CI should test GOWORK=off for ALL modules** — The CI already does per-module `GOWORK=off` builds, but the breakage went unnoticed for multiple sessions. Either the CI wasn't running, or the failures were silently swallowed. This is the "stale GREEN" anti-pattern from AGENTS.md.

3. **Batch release tooling should detect untagged changes** — A script that checks: "Are there commits on master that add exported symbols to a module, but no new tag was cut?" This would catch the entire class of pre-tag drift.

4. **Replace directives should be tracked** — The `decider/go.mod` → `flightrecorder` replace directive has been in place for a while. These accumulate. A release checklist should enumerate and strip all replaces before tagging.

### Technical Improvements

5. **The `store_suite.go` pattern should be duplicated to `event/v4/eventtest`** — Currently there's `commandtest.RunStoreSuite` and `querytest.RunStoreSuite`, but no `eventtest.RunStoreSuite`. This is an opportunity to complete the pattern for event stores.

6. **The replace directive approach is fragile** — It works but creates a maintenance burden. A better long-term solution is faster release cycles or a `make tag-all` script that auto-detects changed modules and tags them.

### Session Self-Critique

7. **I should have checked ALL storage backends from the start** — The original bug report mentioned only pebble and bbolt, but storage/memory was equally broken. I found it during investigation, but I should have started with "find all consumers of `querytest.RunStoreSuite`" as my first step.

8. **I should have run `nix run .#build` or at minimum `go build ./...`** — I verified with targeted `go test`, which is actually the correct verification, but I didn't run the full build gate. The targeted tests are sufficient for this fix, but the discipline matters.

9. **I didn't check whether other untagged symbols exist across the repo** — There could be a similar issue in other module pairs. A comprehensive audit would be valuable.

10. **I didn't check whether `query/v4.2.0` has OTHER missing symbols** — The `store_suite.go` file also defines `MustCreateQuery`. If any consumer uses that under `GOWORK=off`, it would also be broken. The three storage modules are the only consumers found, but others may exist.

---

## f) Up to 50 Things to Get Done Next

### Immediate (this fix followup)

1. **Cut `query/v4.3.0` tag** — tag the query module with store_suite.go
2. **Strip replace directives** from storage/memory, storage/pebble, storage/bbolt once tagged
3. **Bump query/v4 version** to v4.3.0 in all three modules' go.mod
4. **Audit ALL module pairs** for pre-tag symbol drift (consumer references symbol not in required version)
5. **Run `nix run .#verify`** to confirm no other breakage from this session

### CI / Process

6. **Add CI gate** that runs `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` per module (may already exist — verify it's not silently passing)
7. **Add meta-test** that checks every exported symbol in querytest/commandtest/eventtest is available in the tagged version, not just locally
8. **Create `scripts/tag-changed-modules.sh`** — auto-detect modules with uncommitted or untagged exported symbol changes since last tag
9. **Document the replace-directive release checklist** — which replaces to strip, when, and how to verify
10. **Review all existing replace directives** — audit `decider/go.mod`, `middleware/go.mod`, `signing/go.mod`, etc. for stale replaces

### Test Suite Completeness

11. **Add `eventtest.RunStoreSuite`** — complete the trilogy (command + query + event store conformance suites)
12. **Verify `command/v4.4.0` tag** is actually reachable by all consumers under `GOWORK=off` (not just storage modules)
13. **Add `snapshottest.RunStoreSuite`** if the snapshot module has a test suite pattern
14. **Check if `storage/turso` needs querytest** — it builds fine but may be missing the conformance suite
15. **Add querytest conformance to `storage/turso`** if it has a QueryStore implementation

### Tagging & Release Hygiene

16. **Full tag audit** — for each module, check: does the latest tag include all master commits? Are there untagged exported symbols?
17. **Batch tag all modules** that have untagged changes since their last tag
18. **Verify `scripts/tag-release.sh`** handles the query module correctly
19. **Add pre-release hook** that fails if any consumer references symbols not in the required version tag
20. **Document the tag-before-consumer pattern** in CONTRIBUTING.md

### AGENTS.md / Documentation

21. **Add lesson to AGENTS.md**: "Adding exported symbols to shared test helper packages requires tagging a new version before consumers can use them under GOWORK=off"
22. **Add lesson**: "When fixing `undefined: X` errors under GOWORK=off, first check if the source module's latest tag includes the symbol"
23. **Update the "Verify module version exists before requiring it" lesson** with the producer-side corollary
24. **Document the replace-directive pattern** as the interim workaround for pre-tag drift
25. **Add the storage/memory finding** to the status report chain (original report only mentioned pebble + bbolt)

### Build System

26. **Run `nix run .#build`** to verify all 77 modules compile
27. **Run `nix run .#lint`** to verify no lint regressions from go.mod changes
28. **Run `nix run .#check-layers`** to verify dependency budgets aren't violated
29. **Run `nix run .#verify`** (full gate) if time permits
30. **Check if the `scripts/ephemeral-dgraph.sh` change** is intentional and should be kept

### Metaengine / Strategic

31. **Check metaengine modules** for similar pre-tag drift (they have many cross-dependencies)
32. **Verify `metaengine/projectionadapter`** builds under GOWORK=off (it has replace directives)
33. **Audit metaengine engine modules** (pebbleengine, duckdbengine, etc.) for version staleness

### Test Coverage

34. **Run race tests** on the three fixed modules: `GOWORK=off go test -race -tags "goexperiment.jsonv2" -count=1 ./...`
35. **Run soak tests** to verify no runtime issues from the version mismatch fix
36. **Check `storage/memory` golden tests** still pass (they use `go-snaps`)
37. **Verify `storage/pebble` CBOR fuzz tests** still pass

### Broader Repository Health

38. **Check if `integration/` module** has the same querytest issue
39. **Check `system/` module** for querytest usage
40. **Run `cmd/api-stability`** to verify no API surface changes from go.mod edits
41. **Run `cmd/doc-check`** to verify no doc assertions reference the wrong version
42. **Check `example/taskmanager`** — it's the flagship example, verify it still builds
43. **Review the auto-commit daemon's recent commits** for other breaking bumps
44. **Check if `query/v4.2.0` → local replace** changes behavior in any other way (e.g., transitive deps)
45. **Verify `go.sum` files are deterministic** (re-run `go mod tidy` in CI mode and diff)

### Future-Proofing

46. **Consider a monorepo tag-all script** that bumps ALL modules in one batch
47. **Add a "canary" consumer test** that imports every test helper package under GOWORK=off
48. **Consider migrating to Go workspace `go.work` in CI** instead of per-module GOWORK=off (controversial — per-module is more rigorous)
49. **Add a pre-commit hook** that warns when adding exported symbols without a version bump
50. **Create a `RELEASE_CHECKLIST.md`** that enumerates all replace directives and their strip conditions

---

## g) Questions (things I CANNOT figure out myself)

### Q1: Should I cut `query/v4.3.0` right now, or wait for the next batch release?

The replace directives are a temporary workaround. Cutting the tag now would let me strip the replaces and require the tagged version. But there may be other untagged changes in `query/` that should go into the same tag, or a release window I'm not aware of. Should I:
- **(a)** Tag `query/v4.3.0` immediately and update consumers
- **(b)** Leave the replaces and wait for the next batch release
- **(c)** Audit `query/` for ALL changes since v4.2.0 first, then tag

### Q2: Is there a reason `query/v4` was skipped during the last batch tagging?

`command/v4.4.0` was tagged with `store_suite.go` but `query/v4` was left at `v4.2.0`. This looks like an oversight, but there may be a reason (e.g., query has other breaking changes pending, or the batch tagging was partial intentionally). Should I investigate the batch release history, or just fix it?

### Q3: Should the replace-directive pattern be formalized for pre-tag drift?

Currently, `decider/go.mod` uses it for `flightrecorder`. Now 3 storage modules use it for `query`. This is an ad-hoc pattern. Should I:
- **(a)** Document it as an official workaround in AGENTS.md / CONTRIBUTING.md
- **(b)** Replace it with a faster tag-cutting process instead
- **(c)** Add a meta-test that enforces "no replace directives at release time" and tracks them as tech debt

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Fixed | 3 modules | ✅ Verified |
| Root cause identified | query/v4 not tagged after store_suite.go added | ✅ |
| Proper fix (tag) | query/v4.3.0 | ⏳ Not started |
| CI prevention | Pre-tag drift detection | ⏳ Not started |
| Docs update | AGENTS.md lesson | ⏳ Not started |

**Files changed this session:**
- `storage/memory/go.mod` — added `replace query/v4 => ../../query`
- `storage/memory/go.sum` — removed stale query/v4 v4.2.0 hashes
- `storage/pebble/go.mod` — added `replace query/v4 => ../../query`
- `storage/pebble/go.sum` — removed stale query/v4 v4.2.0 hashes
- `storage/bbolt/go.mod` — added `replace query/v4 => ../../query`
- `storage/bbolt/go.sum` — removed stale query/v4 v4.2.0 hashes
- `TODO_LIST.md` — marked item [x] with root cause + fix
- *(not touched: `scripts/ephemeral-dgraph.sh` — auto-commit daemon change)*
