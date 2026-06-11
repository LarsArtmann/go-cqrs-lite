# Superb Types Sprint — Comprehensive Status Report

**Date:** 2026-06-11 04:43
**Branch:** master
**Working Tree:** Clean (all committed)
**Test Suite:** 38/38 packages pass, zero failures

---

## A. FULLY DONE

### Sprint Accomplishments (commits `4a542363` → `c521af96`)

| Commit | What |
|--------|------|
| `4a542363` | Strong IDs + library phantom types (healthcheck, SSE, saga, turso, event reconstruction chain) |
| `8f5f0d31` | Example compilation fixes (catalog-server, user Protocol cast) |
| `6828c32f` | ErrorExporter deprecation alias (`ErrorExporter = Exporter[error]`) |
| `812868b0` | Catalog strong-ID fixes (DisplayID, MessageID, cattest typed params) |
| `360e41e9` | Docs: interface consolidation & test dedup research findings |
| `c521af96` | Example/todo phantom types complete + example/user phantom types |
| `41b15b95` | Golden test fixture refresh (codec + middleware) |

### Phantom Type Inventory (21 phantom types across codebase)

**Catalog domain types** (`catalog/types_phantom.go`):
`Name`, `Version`, `Summary`, `Title`, `Description`, `Address`, `Protocol`, `Host`, `Email`, `URL`, `ContentType`, `DeliveryGuarantee`, `Method`, `Icon`, `Color`, `Language`, `Role` + `DisplayID` in d2/connections.go

**Library types:**
- `event.Type`, `event.AggregateType`, `event.Source`, `event.IPAddress`, `event.UserAgent`, `event.MetadataKey`
- `command.Type`, `query.Type`
- `middleware.ReleaseID`, `middleware.ComponentID`, `middleware.HealthStatus`, `middleware.SSEClientID`
- `turso.DbPath`, `turso.RemoteURL`, `turso.AuthToken`
- `signing.SignatureAlgorithm`, `signing.Actor`

**Example types (local, per-module):**
- `example/todo/domain`: `Title`, `Description`, `Priority`
- `example/user`: `Email`, `DisplayName`, `Reason`

### Metric Progress

| Metric | Start | Current | Target | Status |
|--------|-------|---------|--------|--------|
| Phantom violations | 315 | 233 | <150 | 🟡 In progress |
| Strong-ID violations | 25 | 3 | 0 (3 intentional) | ✅ Done |
| Error handling score | 92/100 | 95/100 | 95+ | ✅ Done |
| Composition health | 98/100 | 99/100 | 95+ | ✅ Done |
| Test pass rate | 37/38 | 38/38 | 38/38 | ✅ Done |
| Golden tests | 2 stale | 0 stale | 0 | ✅ Done |

---

## B. PARTIALLY DONE

### Phantom Types — 233 remaining, but ~60% are by-design false positives

**Breakdown of the 233 violations:**

| Category | Count | Fixable? | ROI |
|----------|-------|----------|-----|
| **Catalog serialization structs** (asyncapi/types.go, openapi/types.go, d2/exporter.go) | ~90 | No — JSON/YAML spec fields must be `string` for marshaling | Zero |
| **Catalog option/builder functions** (build.go, builders.go, config fns) | ~40 | Technically yes — use catalog.Name etc. | Low (internal, no consumers) |
| **Catalog test helpers** (cattest/) | ~20 | Technically yes | Very low |
| **Middleware internal params** (msgKind, label, Kind) | ~28 | No — these are log/trace formatting strings | Zero |
| **Storage/SQL internals** (table, aggType, prefix, query) | ~26 | No — SQL query builder internals | Zero |
| **OTel helpers** (component, eventType, unit) | ~14 | No — constrained by OTel API (`string` attrs) | Zero |
| **Pebble internals** (prefix, limit, syncWrites) | ~8 | No — store config, not domain types | Zero |
| **Example modules** (user, todo, projection, etc.) | ~12 | Partially — JSON payloads must stay `string` | Low |
| **Event internals** (eventtest helpers, replay filter) | ~9 | Partially — internal function params | Low |
| **Other** (memory, listing, dispatcher, query, projection, watermill) | ~15 | Mostly no — constrained by interfaces | Zero |

**True actionable violations: ~15-25** (catalog option functions + a few example struct fields that aren't JSON payloads).

**Key insight:** We've already picked all the high-value fruit. The domain model types in catalog, event, command, query, middleware, turso, and signing all use phantom types. The remaining violations are:
1. **Serialization struct fields** — must stay `string` for JSON/YAML interop
2. **Internal function params** — log messages, SQL query builders, OTel attributes
3. **Interface-constrained params** — Watermill's `topic string`, etc.

---

## C. NOT STARTED

### What the Sprint Plan Called For But Hasn't Been Touched

1. **Library module phantom types** — Wave 5 items for event/eventtest, memory, listing, dispatcher internals. Research shows ~95% are false positives (internal params, not domain types).

2. **`bool` → enum conversions** — branching-flow flags `Deprecated bool`, `Deleted bool`, `ownDB bool`, `syncWrites bool`, `Healthy bool`, `RequireHit bool`, `update bool`, `Required bool`. These are all simple toggles — converting to enums would add complexity without domain value.

3. **Mixin extraction** — 16 opportunities flagged, all low confidence. The "large structs" (catalog.Message at 17 fields, catalog.Service at 16 fields) are already well-organized domain models.

### What SHOULD Have Been Done But Wasn't

1. **The `catalog/types_phantom.go` phantom types should have `String()` methods.** Currently they're bare `type X string` — the `event.Type` pattern includes `String() string` and `IsZero() bool`. Inconsistent.

2. **The `catalog/types_phantom.go` types should implement `fmt.Stringer` and `encoding.TextMarshaler`** for consistent JSON marshaling through the domain model → serialization boundary.

3. **The example/todo `domain.Priority` should have `Int()` not just `String()`** — it's `type Priority int`, and comparisons with `*filter.Priority` (which is `*int`) require explicit casts everywhere.

---

## D. TOTALLY FUCKED UP

### Session Mistakes

1. **Sed damage to test files** — Previous session used `sed` to replace `DecideCreate(aggID, "Title", "desc", 1, nil)` with domain-typed calls. The sed mangled closing quotes: `"new desc")` → `domain.Description("new desc)")`. This caused test failures that looked like logic bugs but were actually syntax errors from bad sed.

2. **Lost DecideChangeStatus logic** — The phantom type rewrite of `decider.go` inadvertently removed the `status.IsValid()` check, the `CompletedAt` timestamp setting, and the `EventCompleted` event type selection from `DecideChangeStatus`. This caused 5 test failures. The function was rewritten to use a separate `DecideComplete` but the tests still call `DecideChangeStatus(StatusCompleted)`. Fixed by restoring the original logic.

3. **BuildFlow pre-commit hook staging edits silently** — The broken BuildFlow hook staged working tree changes during unrelated commits (the docs commit `360e41e9` silently committed todo/user phantom type changes). This made it hard to track what was committed vs still pending.

4. **ErrorExporter revert saga** — The `ErrorExporter = Exporter[error]` deprecation alias was reverted THREE times by the previous session before it finally stuck. The user had to intervene. Lesson: never revert changes you didn't make without reading them first.

### Architectural Concerns

1. **233 phantom violations is misleading** — The branching-flow tool doesn't distinguish between "domain model string that should be typed" and "SQL query parameter that happens to be a string". The true count of actionable violations is ~15-25, not 233.

2. **The `catalog/types_phantom.go` has 17 phantom types** but many aren't used in the serialization layer. The exporters have to `string()` cast every field. This creates a lot of noise in the codebase without clear benefit for the catalog module specifically — the catalog types already have branded IDs (`ServiceID`, `MessageID`, etc.) which prevent the real bugs (mixing up services/messages). Name/Version/Summary phantom types prevent... what exactly? Typos in builder calls?

---

## E. WHAT WE SHOULD IMPROVE

### Type Model Improvements (High Value)

1. **Add `String()` and `IsZero()` to all catalog phantom types** — Consistency with event.Type pattern. Currently only `catalog/types_phantom.go` has bare type definitions without methods.

2. **Add `encoding.TextMarshaler`/`TextUnmarshaler` to catalog phantom types** — This would allow phantom-typed fields to be used directly in JSON/YAML structs without manual `string()` casts, potentially eliminating ~40 "serialization" violations.

3. **Consider `fmt.Stringer` interface for all phantom types** — Go convention, enables `%s` formatting, `%v` in error messages.

4. **The `query.Pagination` struct should use typed `Page` and `PageSize`** — These ARE domain types with invariants (page >= 1, pageSize 1-100). Currently `uint` with no validation.

5. **`projection.HealthReport` should use typed `Healthy` status** — `bool` is boolean-blind. `HealthStatus Healthy|Degraded|Unhealthy` would be clearer.

### Library Improvements (Medium Value)

6. **Extract shared `Stringer` interface pattern** — All phantom types follow the same pattern: `type X string; func (x X) String() string { return string(x) }; func (x X) IsZero() bool { return x == "" }`. Consider go:generate or a generic helper.

7. **The `catalog/asyncapi/types.go` and `catalog/openapi/types.go` are spec-mirroring structs** — These should NOT use phantom types. Add a lint suppression comment or configure branching-flow to exclude them.

8. **watermill `topic string` params are constrained by the Watermill library interface** — Cannot change. Add lint suppression.

### Process Improvements (High Value)

9. **Fix or remove the BuildFlow pre-commit hook** — It silently stages and commits working tree changes during unrelated commits. This is dangerous and caused confusion across the entire session.

10. **Never use `sed` for multi-line code refactoring** — Use `edit` tool with exact match instead. Sed doesn't understand Go syntax and mangles string literals.

---

## F. TOP 25 THINGS TO DO NEXT

Sorted by **impact × effort** (Pareto principle):

### Tier 1: Quick Wins (5 min each, high impact)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Add `String()` + `IsZero()` methods to all 17 catalog phantom types in `types_phantom.go` | Consistency, pattern alignment | 5 min |
| 2 | Add `String()` + `IsZero()` to `catalog.DisplayID` | Same pattern | 1 min |
| 3 | Fix `example/todo/domain.Priority` — add `Int()` method (it's `type Priority int`, not string) | Cleaner boundary casts | 2 min |
| 4 | Configure branching-flow to exclude serialization structs from phantom analysis (or add lint suppression comments) | Accurate violation count | 5 min |
| 5 | Fix/Remove the broken BuildFlow pre-commit hook | Prevent silent staging | 5 min |

### Tier 2: Medium Effort, Good Value (15-30 min each)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 6 | Use catalog phantom types in builder/option functions (`build.go`, `channel_config.go`, `service_config.go`, `message_config.go`) | ~15 violations eliminated | 20 min |
| 7 | Use catalog phantom types in cattest helpers (`builders.go` title/summary params) | ~10 violations eliminated | 15 min |
| 8 | Add `encoding.TextMarshaler` to catalog phantom types — enables direct JSON use | Eliminates ~40 string() casts in exporters | 30 min |
| 9 | Type `query.Pagination` fields as `Page uint` and `PageSize uint` phantom types with validation | Domain safety | 15 min |
| 10 | Type `projection.HealthReport` fields — `Healthy` → `HealthStatus` enum, `Checkpoint` → typed | Clarity | 15 min |
| 11 | Fix error context in `memory/store_load.go:35` and `middleware/recovery.go:34` (include `op`/`msgKind`/`typeName` in error format strings) | Error handling 95→97 | 10 min |

### Tier 3: Lower ROI, Consider Carefully

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 12 | Type `storage/sql` query builder params (`table`, `aggType`, `prefix`) | Internal safety | 30 min |
| 13 | Type `pebble` store config fields (`prefix`, `journalPrefix`) | Config safety | 15 min |
| 14 | Type `otel` helper params (`component`, `eventType`, `unit`) | Constrained by OTel API | 20 min |
| 15 | Type `watermill` topic params | Constrained by Watermill API | 10 min |
| 16 | Type `memory` store internal params (`op`) | Internal only | 10 min |
| 17 | Type `listing` reader params | Internal only | 5 min |
| 18 | Type `dispatcher` lifecycle state | Internal only | 5 min |
| 19 | Convert `bool` fields to enums where flagged (`Deprecated`, `Deleted`, `ownDB`, `syncWrites`) | Debatable value | 45 min |
| 20 | Extract shared phantom type generation pattern (go:generate or generic helper) | DRY | 30 min |

### Tier 4: Strategic / Future

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 21 | Evaluate `mnd` (magic number detector) — fix the `10` in gracefulshutdown default timeout | Code quality | 5 min |
| 22 | Consider `bool` enum conversion for `catalog.Message.Deprecated` — `DeprecatedStatus Active|Deprecated` | Domain clarity | 15 min |
| 23 | Add catalog diff/breaking-change detection tool (from TODO_LIST.md) | Consumer safety | 4+ hours |
| 24 | Consider `go-branded-id` patterns for remaining strong-ID violations (`openapi.OperationID`, `FlowStep.ID`, `FlowEdge.ID`) | Completeness | 30 min |
| 25 | Evaluate whether branching-flow phantom analysis should have a "serialization struct" exclusion mode | Tooling | 2+ hours |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should we invest more time driving the phantom violation count from 233 down, or is the current state "good enough"?**

The analysis shows:
- **233 violations** sounds bad, but ~60% (140) are in serialization structs, OTel/Watermill-constrained params, or SQL query builder internals where phantom types add no value and would create noise
- **~25 are truly actionable** (catalog builder functions, test helpers, a few example struct fields)
- Fixing those 25 would take ~2 hours and bring the count to ~208 — still looks "high" but is honestly the floor
- The **high-value phantom types are already done**: `event.Type`, `event.AggregateType`, `catalog.Name/Version/Summary/Title`, `middleware.SSEClientID`, `turso.DbPath`, `signing.Actor`, etc.
- The **remaining violations are in plumbing code**, not domain boundaries

The question is: **should we configure branching-flow to exclude serialization structs and internal plumbing, accept the "real" count is ~25, and move on to other work?** Or keep driving toward a lower raw number even though each fix has diminishing returns?

---

## Current branching-flow Scores

| Linter | Score/Count | Status |
|--------|-------------|--------|
| Error Handling | 95.0/100 (Good) | ✅ |
| Duplicate Types | 15 groups (5 actionable) | 🟡 Examples only |
| Phantom Types | 233 violations | 🟡 ~60% false positives |
| Panic Conditions | 2 detections | ✅ Both guarded |
| Strong ID | 3 violations | ✅ All intentional |
| Boolean Blindness | 0 violations | ✅ |
| Composition Health | 99/100 (Good) | ✅ |
| Anti-Patterns | 2 large structs | 🟡 Message/Service 16-17 fields |
