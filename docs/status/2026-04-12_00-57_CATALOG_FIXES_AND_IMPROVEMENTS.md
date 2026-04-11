# Status Report: Catalog Fixes, Generic Adapters & Infrastructure

**Date:** 2026-04-12 00:57
**Author:** Crush (AI Assistant)
**Branch:** master
**Commits:** 8 new (a9344e3..2d74652)
**Since:** `docs/status/2026-04-10_12-14_CATALOG_AUTO_DISCOVERY.md`

---

## Summary

Fixed **all 3 pre-existing failures** (event fuzz, xtypes build, depguard lint), added **generic type-safe catalog methods** (`AddCommandFromType[T]`, `AddEventFromType[T]`, `AddQueryFromType[T]`), fixed a **schema generation bug** where embedded `*Core`/`*CatalogCore` leaked into JSON schemas, added **version to `AddService`**, and improved **EventCatalog frontmatter** to include commands/queries.

**280 lines added** across 14 files. **13/13 packages pass** (was 11/13 — `event` and `xtypes` were broken).

---

## Test Results

```
ok  aggregate                0.003s
ok  catalog                  0.005s
ok  catalog/adapters         0.005s   ← 9 tests (was 7)
ok  catalog/asyncapi         0.002s
ok  catalog/eventcatalog     0.026s   ← 12 tests (was 10)
ok  catalog/yaml             0.002s
ok  command                  0.003s
ok  event                    0.003s   ← FIXED (was FAIL: FuzzParseSource)
ok  internal/dispatcher      0.002s
ok  middleware               0.103s
ok  pkg/id                   0.002s
ok  query                    0.003s
ok  xtypes                   0.002s   ← FIXED (was build failure)
```

**ZERO failures. ZERO build errors. ZERO depguard warnings.**

---

## a) FULLY DONE

| # | Task | Files | Impact |
|---|------|-------|--------|
| 1 | Skip anonymous embedded fields in schema generation | `catalog/schema.go:61`, `catalog/schema_test.go` | `*CatalogCore`/`*Core` no longer leak internal fields into JSON schemas. Anonymous fields are now silently skipped. |
| 2 | Add generic `AddCommandFromType[T]()` method | `catalog/adapters/command.go:50-69` | Zero-instance, compile-time-safe command registration. Uses `SchemaFromType[T]()` instead of requiring a constructed value. |
| 3 | Add generic `AddEventFromType[T]()` method | `catalog/adapters/event.go:58-80` | Same for events. Accepts explicit `direction` parameter. |
| 4 | Add generic `AddQueryFromType[T]()` method | `catalog/adapters/query.go:30-48` | Same for queries. |
| 5 | Add version parameter to `AddService()` | `catalog/adapters/builder.go:89` | `AddService(id, name, version, summary)` — services now have proper version metadata in generated catalogs. |
| 6 | Add commands/queries to EventCatalog service frontmatter | `catalog/eventcatalog/exporter.go:122-134` | Service `index.mdx` now lists `commands:` and `queries:` alongside `sends:`/`receives:`. |
| 7 | Fix xtypes build failure (`%s` → `%v` for generic T) | `xtypes/xtypes_test.go:25` | `assertMetadataID[T]` used `%s` format verb which doesn't work with arbitrary comparable types. |
| 8 | Fix event fuzz test failure (vertical tab handling) | `event/fuzz_test.go:78-80` | Replaced manual `trimSpaces` with `strings.TrimSpace` which handles `\v`, `\f`, and unicode spaces. |
| 9 | Fix depguard linter config (51+ → 0 warnings) | `.golangci.yml:128-137` | Added allow-rules for `$gostd`, internal packages, and all 3 declared dependencies. |
| 10 | Add tests for generic FromType methods | `catalog/adapters/adapters_test.go` | `TestBuilder_AddCommandFromType`, `TestBuilder_AddQueryFromType` — verify schema extraction and embedded field skipping. |
| 11 | Add test for commands/queries in frontmatter | `catalog/eventcatalog/exporter_test.go` | `TestExporter_Export_CommandsAndQueriesInServiceFrontmatter` — verifies frontmatter contains both lists. |
| 12 | Add schema tests for anonymous embedded fields | `catalog/schema_test.go` | `TestSchemaFromType_SkipsAnonymousEmbeddedFields`, `TestSchemaFromType_SkipsAnonymousPointerEmbeddedFields` |
| 13 | Update AGENTS.md | `AGENTS.md` | Documented anonymous field filtering, generic adapter API, and EventCatalog frontmatter fields. |
| 14 | Update example/catalog for new AddService signature | `example/catalog/main.go:42` | `AddService("user-service", "User Service", "1.0.0", "Manages user accounts")` |

---

## b) PARTIALLY DONE

| # | Task | Status | What's Missing |
|---|------|--------|----------------|
| 1 | Wire `example/user` aggregate to use catalog-aware event constructors | Constructors `NewUserCreated`/`NewUserEmailChanged` exist in `events.go` but `aggregate.go:62,95` still uses raw `event.NewEvent()`. | The aggregate should call `NewUserCreated(payload)` / `NewUserEmailChanged(payload)` instead of manual `json.Marshal` + `event.NewEvent()`. |
| 2 | `example/user/events.go` missing `AggregateType` in catalog meta | `EventCatalogMeta` at lines 41-44, 68-71 has `Name`, `Version`, `Summary` but omits `AggregateType` field (which exists on the struct). | Should set `AggregateType: AggregateType` in both event constructors for completeness. |
| 3 | Schema generation for `time.Time` fields | `schemaFromReflect` at line 48 checks `t.Kind() != reflect.Struct` but doesn't special-case `time.Time` → generates `{"type":"object","properties":{...}}` with all internal time fields instead of `{"type":"string","format":"date-time"}`. | Need a type-path check: if `t.String() == "time.Time"` or `t.PkgPath() == "time"`, return `{type:"string", format:"date-time"}`. |

---

## c) NOT STARTED

### High Priority

| # | Task | Priority | Effort | Est |
|---|------|----------|--------|-----|
| 1 | Fix `time.Time` → `{type:"string", format:"date-time"}` in schema gen | HIGH | LOW | 8m |
| 2 | Wire `example/user/aggregate.go` to use `NewUserCreated`/`NewUserEmailChanged` | HIGH | LOW | 10m |
| 3 | Add `AggregateType` to example/user event catalog meta | HIGH | LOW | 4m |
| 4 | Add `enum` struct tag support to Schema/Property | MED | LOW | 10m |
| 5 | Add `default` struct tag support to Schema/Property | MED | LOW | 6m |
| 6 | Add `Examples` field to AsyncAPI Message type | MED | LOW | 4m |
| 7 | Wire `catalog.Message.Examples` → AsyncAPI export | MED | LOW | 6m |
| 8 | Wire `catalog.Message.Examples` → EventCatalog export | MED | LOW | 6m |

### Medium Priority

| # | Task | Priority | Effort | Est |
|---|------|----------|--------|-----|
| 9 | Generate `llms.txt` alongside EventCatalog output | MED | LOW | 10m |
| 10 | Add `RegisterCatalogEntry()` to command.Dispatcher (option B) | MED | MED | 10m |
| 11 | Add `CatalogEntries()` accessor to command.Dispatcher | MED | MED | 8m |
| 12 | Add `RegisterCatalogEntry()` to query.Dispatcher | MED | MED | 8m |
| 13 | Add `CatalogEntries()` accessor to query.Dispatcher | MED | MED | 6m |
| 14 | Add `FromDispatcher()` to CatalogBuilder | MED | LOW | 8m |
| 15 | Wire `example/user/main.go` to catalog builder | MED | LOW | 6m |
| 16 | Test `example/catalog` runs in CI (`go run .`) | MED | LOW | 6m |

### Low Priority

| # | Task | Priority | Effort | Est |
|---|------|----------|--------|-----|
| 17 | Add `catalog/adapters` to cattest helpers | LOW | LOW | 8m |
| 18 | Add `AddChannel()` to CatalogBuilder | LOW | LOW | 8m |
| 19 | Add README section for generic FromType methods | LOW | LOW | 8m |
| 20 | Add benchmarks for adapters package | LOW | LOW | 8m |
| 21 | EventCatalog: custom MDX body content via Message field | LOW | MED | 10m |
| 22 | EventCatalog: generate `package.json` for deployment | LOW | MED | 10m |
| 23 | YAML frontmatter: versioned message paths | LOW | LOW | 8m |
| 24 | YAML frontmatter: configurable owners list | LOW | LOW | 6m |
| 25 | Schema: support `nullable`, `deprecated`, `pattern`, `minimum`/`maximum` tags | LOW | MED | 20m |
| 26 | Add `gofumpt`/`goimports` to pre-commit hook | LOW | LOW | 6m |
| 27 | Update ROADMAP.md with catalog improvements | LOW | LOW | 4m |
| 28 | Research jsonschema libs vs. our reflect approach | LOW | MED | 12m |
| 29 | Add GoDoc package examples for catalog/adapters | LOW | LOW | 10m |

---

## d) TOTALLY FUCKED UP

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| 1 | `time.Time` fields produce `{type:"object"}` instead of `{type:"string",format:"date-time"}` in schemas | HIGH | **NOT FIXED** — affects any event/command with timestamp fields. `schemaFromReflect` has no special-case for `time.Time`. |
| 2 | `example/user/aggregate.go` ignores catalog-aware constructors | LOW | Catalog constructors exist but aggregate still uses raw `event.NewEvent()`. Example is misleading. |
| 3 | ~200 golangci-lint warnings remain | COSMETIC | exhaustruct, revive, wsl_v5, tagalign — all non-blocking but noisy. Root cause: aggressive linter config (60+ linters enabled). |

---

## e) WHAT WE SHOULD IMPROVE

1. **`time.Time` schema handling** — `schemaFromReflect` generates `{"type":"object","properties":{...}}` for `time.Time` with all 10+ internal fields. Should return `{"type":"string","format":"date-time"}`. This is a real bug affecting any event with timestamps.

2. **Dead `Examples` field** — `catalog.Message.Examples []json.RawMessage` exists in `catalog/types.go:28` but nothing populates it and nothing reads it. AsyncAPI `Message` type (at `asyncapi/types.go:68-75`) has no `Examples` field. Should either wire it end-to-end or remove it.

3. **No auto-discovery** — The dispatcher stores only `Handler` functions (not command instances). To auto-discover catalog entries, need `RegisterCatalogEntry(type, meta)` side method on dispatcher. This is the recommended "option B" from the previous report.

4. **Schema model too simple** — Missing `enum`, `default`, `nullable`, `pattern`, `minLength`/`maxLength`, `minimum`/`maximum`. All common in API documentation. A jsonschema library (e.g. `github.com/invopop/jsonschema`) would handle these for free but violates the zero-dependency principle.

5. **No `llms.txt` generation** — EventCatalog generates this by default; our exporter should too.

6. **Example/user not fully wired** — The aggregate still uses raw `event.NewEvent()` while catalog-aware constructors sit unused in `events.go`. This defeats the purpose of the example.

---

## f) Top #25 Things To Do Next

Sorted by impact × effort (highest first):

| # | Task | Impact | Effort | Est | Category |
|---|------|--------|--------|-----|----------|
| 1 | Fix `time.Time` → `{type:"string",format:"date-time"}` in schemas | HIGH | LOW | 8m | Bug |
| 2 | Wire `example/user/aggregate.go` to catalog-aware constructors | HIGH | LOW | 10m | Fix |
| 3 | Add `AggregateType` to example/user event catalog meta | MED | LOW | 4m | Fix |
| 4 | Add `enum` struct tag to Schema/Property + parsing + tests | MED | LOW | 18m | Feature |
| 5 | Add `default` struct tag to Schema/Property + parsing + tests | MED | LOW | 10m | Feature |
| 6 | Add `Examples` to AsyncAPI Message + wire from catalog | MED | LOW | 10m | Feature |
| 7 | Wire `Message.Examples` → EventCatalog export | MED | LOW | 6m | Feature |
| 8 | Generate `llms.txt` alongside EventCatalog output + test | MED | LOW | 18m | Feature |
| 9 | Add `RegisterCatalogEntry()` to command.Dispatcher + test | MED | MED | 20m | Feature |
| 10 | Add `CatalogEntries()` accessor to command.Dispatcher | MED | MED | 8m | Feature |
| 11 | Add `RegisterCatalogEntry()` to query.Dispatcher + test | MED | MED | 16m | Feature |
| 12 | Add `FromDispatcher()` to CatalogBuilder + integration test | MED | LOW | 18m | Feature |
| 13 | Wire `example/user/main.go` to catalog builder | MED | LOW | 6m | Example |
| 14 | Test `example/catalog` runs in CI | MED | LOW | 6m | CI |
| 15 | Add `catalog/adapters` to cattest helpers | LOW | LOW | 8m | Refactor |
| 16 | Add `AddChannel()` to CatalogBuilder + test | LOW | LOW | 14m | Feature |
| 17 | Add README section for generic FromType methods | LOW | LOW | 8m | Docs |
| 18 | Add benchmarks for adapters package | LOW | LOW | 8m | Perf |
| 19 | EventCatalog: custom MDX body content | LOW | MED | 10m | Feature |
| 20 | EventCatalog: generate `package.json` for deployment | LOW | MED | 10m | Feature |
| 21 | YAML frontmatter: versioned paths + configurable owners | LOW | LOW | 14m | Feature |
| 22 | Schema: `nullable`/`deprecated`/`pattern`/`min-max` tags | LOW | MED | 20m | Feature |
| 23 | Research jsonschema libs vs. reflect approach | LOW | MED | 12m | Research |
| 24 | Update ROADMAP.md | LOW | LOW | 4m | Docs |
| 25 | Add GoDoc package examples for catalog/adapters | LOW | LOW | 10m | Docs |

---

## g) Top #1 Question

**Should we add `time.Time` handling as a hardcoded special-case in `schemaFromReflect`, or should we design a general "type override" registry?**

Options:
- **A)** Hardcode `if t.String() == "time.Time"` → `{type:"string", format:"date-time"}`. Simple, covers 99% of cases.
- **B)** Add a `SchemaOverride func(reflect.Type) *Schema` option to `SchemaFromType` / `SchemaFromReflect`. Users can register overrides for custom types. More flexible but more complex API.
- **C)** Check for `encoding.TextMarshaler` / `json.Marshaler` interface — if a type implements these, treat it as `string` with the struct tag's `format`. Most general but may produce unexpected results for complex marshalers.

I recommend **A** for now (hardcode `time.Time`). Option C is the "right" answer but has subtle edge cases. We can always add B/C later without breaking A.

---

## Architecture After Changes

```
┌───────────────────────────────────────────────────────┐
│ USER CODE                                              │
│  type CreateUser struct {                              │
│    *command.CatalogCore  ← SKIPPED in schema output   │
│    Name string `json:"name" doc:"Full name"`          │
│  }                                                     │
└──────────────────────┬────────────────────────────────┘
                       │ TWO ways to register:
                       ▼
┌───────────────────────────────────────────────────────┐
│ catalog/adapters.CatalogBuilder                        │
│                                                        │
│  Instance-based (requires constructed value):          │
│    .AddCommand("svc", &CreateUser{...})                │
│                                                        │
│  Generic (zero-instance, compile-time safe):           │
│    adapters.AddCommandFromType[CreateUser](             │
│        builder, "svc", "user.create", meta,            │
│    )                                                   │
│                                                        │
│  .AddService("svc", "Svc", "1.0.0", "summary")        │
│  .Build() → *catalog.Catalog                          │
└──────────┬────────────────────────┬───────────────────┘
           │                        │
           ▼                        ▼
┌─────────────────────┐  ┌─────────────────────────────┐
│ eventcatalog/       │  │ asyncapi/                    │
│ Exporter.Export()   │  │ Exporter.Export()            │
│ → service frontmatter│  │ → Document.MarshalYAML()   │
│   now includes:     │  │ → Document.MarshalJSON()    │
│   sends, receives,  │  └─────────────────────────────┘
│   commands, queries │
│ → schemas/schema.json│
└─────────────────────┘
```

## Commits (8 total)

```
63d3716 fix(catalog): skip anonymous embedded fields in schema generation
8638b3e fix(xtypes): use %v instead of %s for generic type in test helper
0e3932e fix(event): use strings.TrimSpace in fuzz test helper
b237ec7 fix(lint): configure depguard allow-rules for internal and dependency imports
b3a43f9 feat(catalog/adapters): add generic FromType methods for compile-time schema gen
05236d9 feat(catalog/adapters): add version parameter to AddService
76b3b17 feat(eventcatalog): add commands/queries to service frontmatter
2d74652 docs: update AGENTS.md with schema fix and generic adapter docs
```
