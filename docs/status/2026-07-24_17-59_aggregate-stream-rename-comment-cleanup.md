# Status: Aggregate→Stream Rename Follow-up (Comment & Doc Cleanup)

> **Session:** 2026-07-24 17:59
> **Task:** Complete the two follow-up items from the ADR-0058 aggregate→stream rename:
>
> 1. Comment cleanup in ~70 Go production files
> 2. SKILL.md reference file updates (32 mentions across 6 files)
>    **ADR:** [ADR-0058](../adr/0058-rename-aggregate-to-stream.md)
>    **Plan:** [SUPERB-RENAME-AGGREGATE-TO-STREAM](../planning/2026-07-23_17-51_SUPERB-RENAME-AGGREGATE-TO-STREAM.md)

---

## a) FULLY DONE

### 1. Go Comment Cleanup — 7 Target Directories (41 files)

**Approach:** Wrote a Python script that targets **comment text only** (after `//`), using `\b` word boundaries to avoid touching code symbols. Skipped `event/v3_compat_aliases.go` entirely (intentional backward-compat file).

**Directories cleaned:**

- `event/` — 7 files (doc.go, store.go, event.go, tombstone.go, types.go, streaming_source.go, v4/eventtest/store_suite.go)
- `snapshot/` — 4 files (doc.go, strategy.go, helper.go, read_pressure.go)
- `command/` — 3 files (doc.go, store.go, command.go)
- `decider/` — 8 files (doc.go, decider.go, cache.go, options.go, typed_decider.go, decider_cache_test.go, decider_bdd_test.go, benchmark_singleflight_test.go)
- `listing/` — 4 files (doc.go, builder.go, example_test.go, + read_pressure_test.go via snapshot dir)
- `storage/memory/` — 3 files (store.go, store_load.go, command_store.go)
- `storage/pebble/` — 12 files (doc.go, store.go, snapshot.go, otel.go, helpers.go, journal.go, iteration.go, command_read.go, command_store.go, query_store.go, snapshot_test.go, journal_test.go, journal_scan_test.go)

**Quality fixes applied manually:**

- "aggregate root" → "entity" (not "stream root" — avoids awkwardness)
- "mutable stream interface" → "mutable state management" (decider/decider.go:34)
- "replaces a mutable stream with a pure fold" → "replaces a mutable entity with a pure fold" (decider/doc.go:3)

**Verification:**

- `go build -tags "goexperiment.jsonv2"` — OK
- `go vet` — OK
- `go test` on all 7 module groups — all pass
- `nix fmt` — 0 changes needed (comments don't affect formatting)

### 2. SKILL.md Reference Files — 7 files, 35 mentions cleared

**Files updated:**

| File              | Changes                                                                                                                                                                               |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core.md`         | 10 edits: prose (aggregate→stream) + API symbols (NewAggregateID→NewStreamID, NewAggregateRef→NewStreamRef)                                                                           |
| `advanced.md`     | 10 edits: section heading, TOC link, code examples (NewInMemoryAggregateReader→NewInMemoryStreamReader, NewAggregateProjection→NewStreamProjection, evt.AggregateID()→evt.StreamID()) |
| `recipes.md`      | 3 edits: API symbols + prose                                                                                                                                                          |
| `modules.md`      | 3 edits: AggregateMarker→StreamMarker, listing type names, idtest function name                                                                                                       |
| `readmodels.md`   | 2 edits: id.AggregateID→id.StreamID (type params)                                                                                                                                     |
| `faq.md`          | 3 edits: prose (aggregate→stream)                                                                                                                                                     |
| `SKILL.md` (root) | 3 edits: description frontmatter (event-sourced aggregates→streams, trigger phrases)                                                                                                  |

**Verification:**

- `cmd/doc-check` — **897 references valid across 34 packages** (all Go import paths + qualified symbols confirmed)
- `grep -c "[Aa]ggregate"` — 0 in all 7 files

### What Was Preserved (Correctly Untouched)

- **Backward-compat type aliases:** `AggregateType`, `AggregateID`, `AggregateRef`, `AggregateListing`, `AggregateStatus`, `AggregateReader`, `AggregateMarker` — all retained as deprecated aliases
- **Method names:** `evt.AggregateID()`, `evt.AggregateType()` — deprecated wrappers still exist
- **Error variables:** `ErrNilAggregateID`, `ErrEmptyAggregateType`, `ErrAggregateNotFound`, etc.
- **Interface names:** `AggregateAwareStrategy` (snapshot)
- **Function names:** `NewAggregateRef`, `ParseAggregateType`, `NewInMemoryAggregateReader`, `NewAggregateProjection` — deprecated wrappers
- **SQL schema column names:** `aggregate_type`, `aggregate_id` in DDL/migrations (schema-level, not Go comments)
- **String literals:** error messages, slog keys, JSON tags
- **Test function names:** `TestAggregateRef_IsZero`, `FuzzParseAggregateType`, etc. (test identity)

---

## b) PARTIALLY DONE

### 1. Storage Root Module (`storage/*.go`) — MISSED

**6 comment lines with stale "aggregate" NOT cleaned:**

- `storage/command_store_journal.go:42` — "all aggregates"
- `storage/command_store_load.go:31` — "for an aggregate"
- `storage/pg_bus_test.go:496` — "aggregate reference"
- `storage/turso/indexing/advisor_test.go:152,354` — "aggregate" in test comments
- `storage/turso/indexing/doc.go:6` — "aggregate loads"

**Note:** The task scope listed specific directories (`storage/memory/` and `storage/pebble/`) but not `storage/` root or `storage/turso/`. These are still stale.

### 2. AGENTS.md — NOT TOUCHED (16 stale references)

The project's own `AGENTS.md` has **16 lines** with "aggregate" references in code examples, tree comments, and prose. These are the most visible stale references for any AI session or contributor. Key offenders:

- Line 52: Tree comment `AggregateID, EventID, etc.`
- Line 78: `AggregateListing, AggregateStatus, ... InMemoryAggregateReader`
- Line 138: Prose "for the same aggregate"
- Lines 156-170: Code examples using old API names
- Lines 231, 241: Code examples with `evt.AggregateID()`, `id.NewAggregateRef()`
- Lines 401-422: Code comments and examples with old names
- Lines 473-474: `id.AggregateID` type params

**Why partially done:** AGENTS.md was not in the task scope (task said "comments in Go files" and "SKILL.md references"), but it's the most impactful stale doc.

### 3. No `nix run .#lint` Run

Only ran `go build` + `go vet` + targeted `go test`. Did NOT run the full `nix run .#lint` (golangci-lint with depguard, gosec, golines, etc.). Comments-only changes are extremely unlikely to trigger lint issues, but the gate wasn't fully run.

### 4. No Full Test Suite Run

Only tested the 7 changed module groups, not the full workspace (`nix run .#test`). Again, comment-only changes can't break tests, but the full gate wasn't run.

---

## c) NOT STARTED

1. **AGENTS.md aggregate→stream cleanup** (16 lines) — highest impact remaining work
2. **Storage root + turso comment cleanup** (6 lines)
3. **docs/ planning and status files** — many historical references (mostly archive/, but some active planning docs like `SAGA_DESIGN.md` (5), `event-query-model.md` (6), `go-composable-business-types-usage.md` (64!))
4. **Rename plan status update** — `docs/planning/2026-07-23_17-51_SUPERB-RENAME-AGGREGATE-TO-STREAM.md` should be updated to mark the comment/doc cleanup phases as done
5. **DOMAIN_LANGUAGE.md** — not checked for aggregate references (may be intentional DDD terminology)

---

## d) TOTALLY FUCKED UP

**Nothing.** All changes are safe:

- Comment-only changes in Go (zero behavioral impact)
- Markdown doc updates verified by doc-check (897 refs valid)
- All tests pass, build passes, vet passes
- No code symbols, aliases, or string literals were touched

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Scope was too narrow** — The task listed 7 specific directories, but "aggregate" appears in comments across `storage/` root, `storage/turso/`, and potentially other modules. A full-workspace grep for comment-only aggregate references should have been run first to establish complete scope.

2. **AGENTS.md is the elephant in the room** — It's the #1 file AI sessions read, and it has 16 stale references. This should have been part of the cleanup even if not explicitly scoped.

3. **Script was deleted** — The Python rename script was thrown away. It should be saved as a reusable tool (e.g., `scripts/rename-comments.py`) since the same pattern (conceptual term in comments only, preserving code symbols) will recur.

4. **No commit was made** — All work is uncommitted in the working tree. The changes should be committed with a clear message.

5. **Verification was partial** — `nix run .#lint` and `nix run .#test` (full suite) were not run. While comment-only changes are safe, the project's quality gates exist for a reason.

### Technical Improvements

6. **"aggregate root" handling was ad-hoc** — The script mapped "aggregate root" → "stream", but then I manually fixed 2 spots where this produced awkward English ("mutable stream"). A better mapping would be: "aggregate root" → "entity" (DDD-correct) or "aggregate" (the concept, when not talking about identity).

7. **AGENTS.md code examples use deprecated APIs** — The code blocks in AGENTS.md still show `evt.AggregateID()`, `id.NewAggregateRef()`, etc. These should be updated to the canonical `Stream*` names with a note that aliases exist for backward compat.

8. **The rename plan doc is stale** — It says "Status: In Progress" but phases 7-8 (tests + docs) are partially done. Should be updated.

---

## f) Next 50 Things to Get Done

### Immediate (This Rename Follow-up)

1. **Clean AGENTS.md** — Update 16 aggregate references to stream terminology
2. **Clean storage/ root comments** — 3 stale comment lines in command_store_*.go and pg_bus_test.go
3. **Clean storage/turso/ comments** — 3 stale comment lines in indexing/advisor_test.go and doc.go
4. **Run `nix run .#lint`** — Full lint gate
5. **Run full test suite** — `nix run .#test` or the full `go test ./...` command from AGENTS.md
6. **Commit all changes** — Clear commit message documenting the comment + doc cleanup
7. **Update rename plan doc** — Mark comment/doc cleanup phases as done in `2026-07-23_17-51_SUPERB-RENAME-AGGREGATE-TO-STREAM.md`
8. **Check DOMAIN_LANGUAGE.md** — Decide if "aggregate" is intentional DDD vocabulary or should be "stream"

### Short-Term (Rename Completeness)

9. **Audit all docs/planning/ active files** for stale aggregate references (not archive/)
10. **Check docs/architecture-understanding/** for stale references
11. **Check all ADR files** for consistency with the rename
12. **Check CONTRIBUTING.md** for stale references
13. **Check README.md** for stale references
14. **Save rename script** as `scripts/rename-concept-in-comments.py` for reuse
15. **Run cqrs-lint on the changed modules** to verify no new findings

### Medium-Term (Library Quality)

16. **Consider renaming `AggregateAwareStrategy`** to `StreamAwareStrategy` — it's a public interface, would need deprecation
17. **Consider renaming error variables** (`ErrNilAggregateID` → `ErrNilStreamID`) with deprecated aliases
18. **Consider renaming SQL columns** (`aggregate_type` → `stream_type`) with a migration — BIG breaking change
19. **Consider renaming the `aggregate_projection` table** in storage listing
20. **Audit JSON field names** (`aggregateId`, `aggregateType` in serialization) — client-facing breaking change
21. **Update cqrs-lint rules** to flag deprecated `Aggregate*` API usage in consumer code
22. **Add a deprecation migration guide** for consumers

### Documentation & Consistency

23. **Update FEATURES.md** if it references aggregate terminology
24. **Update TODO_LIST.md** with the rename follow-up status
25. **Update ROADMAP.md** if it references the rename
26. **Update CHANGELOG.md** with the rename entry
27. **Write an ADR index entry** for ADR-0058 if not already done
28. **Update example/ code** to use canonical Stream* names (already done? verify)
29. **Check all module doc.go files** outside the 7 target dirs for stale comments
30. **Verify the otel/ attribute constants** were renamed (AttrStreamType vs AttrAggregateType)

### Testing & Verification

31. **Run `nix run .#verify`** — Full verification gate (build + vet + test + race + lint + doc-check + doc-assertions)
32. **Run race detector** on changed modules
33. **Check golden test outputs** — some golden files may contain "aggregate" in expected text
34. **Run `nix run .#check-layers`** — Dependency budget check
35. **Run `nix flake check`** — Full flake validation
36. **Run api-stability check** — Verify no API surface changes (comments shouldn't affect it, but verify)

### Broader Codebase Health

37. **Run the brutal-self-review skill** on the rename work
38. **Run the docs-health skill** to verify documentation consistency
39. **Check if `deriver` module comments** need cleanup (wasn't in scope but may have stale refs)
40. **Check `projectionhost` comments** for aggregate references
41. **Check `watermill` adapter comments** for aggregate references
42. **Check `transport/` comments** for aggregate references
43. **Check `catalog/` comments** — schema generation may reference aggregates
44. **Check `graph/` comments** — graph projections may reference aggregates
45. **Check `scenario/` test DSL comments** for aggregate references
46. **Check `scheduling/` comments** for aggregate references
47. **Check `integration/` test comments** for aggregate references
48. **Check `benchkit/` comments** for aggregate references
49. **Check `stack/` preset comments** for aggregate references
50. **Full-workspace `grep -rn "[Aa]ggregate" --include="*.go" | grep "//"` to find ANY remaining stale comment**

---

## g) Questions I Cannot Answer Myself

### 1. Should AGENTS.md code examples be updated to canonical Stream* names?

AGENTS.md has 16 "aggregate" references — some are prose comments in the module tree, but several are **code examples** showing the old API (`evt.AggregateID()`, `id.NewAggregateRef()`, `id.AggregateID`). The deprecated aliases still compile, so the examples work. But the question is: should AGENTS.md show the **canonical** API (`evt.StreamID()`, `id.NewStreamRef()`), or keep showing the deprecated names that most existing consumer code still uses? This is a documentation philosophy question — "show the new way" vs "show what most code actually uses today."

### 2. Should "aggregate" remain in DOMAIN_LANGUAGE.md as intentional DDD terminology?

"Aggregate" and "Aggregate Root" are formal Domain-Driven Design terms from Eric Evans. The rename to "stream" was a library-level API decision, but `docs/DOMAIN_LANGUAGE.md` may intentionally keep DDD vocabulary to bridge the conceptual gap for readers familiar with DDD. I cannot decide whether domain language docs should follow the code rename or preserve DDD's canonical vocabulary.

### 3. Should the SQL schema columns (`aggregate_type`, `aggregate_id`) be renamed?

The database tables use `aggregate_type` and `aggregate_id` as column names. Renaming these would require a **migration** for every deployed consumer database — a significant breaking change. The code comments around these SQL references are stale, but the column names themselves are a data-layer decision that affects every consumer's persisted data. This is a product-level decision, not a code cleanup decision.
