# Deduplication Plan — `art-dupl --semantic --sort total-tokens -t 30`

> Generated from `reports/art-dupl/report.json` — 73 clone groups, 186 clones, 828 files.
> Policy: **Zero _harmful_ duplication. Not zero report lines.** (see `deduplicate-code` skill)

## Scan Summary

| Metric                     | Value                          |
| -------------------------- | ------------------------------ |
| Files analyzed             | 828                            |
| Clone groups               | 73                             |
| Total clones               | 186                            |
| Complexity score           | 2.51                           |
| Threshold                  | 30 tokens (semantic)           |
| Category of **all** groups | `idiom` / `low` / `actionable` |

## Triage Outcome (all 73 groups)

Every group is `idiom/low`. After applying the decision checklist, they fall into three buckets:

### EXTRACT (real maintenance burden — "must change together")

The **ID-parse helper family**. Six near-identical panic-on-error wrappers around `id.ParseXxx`,
duplicated across ~20 files / ~173 call sites / 38 definitions. The `id.Parse*` API is central to
the whole library; a signature change today touches 20 files. This is the textbook
"extract" signal.

| Helper             | Underlying fn           | Definitions removed | Call sites |
| ------------------ | ----------------------- | ------------------- | ---------- |
| `parseAggID`       | `id.ParseAggregateID`   | 17                  | ~136       |
| `parseEventID`     | `id.ParseEventID`       | 6                   | ~6         |
| `parseCorrID`      | `id.ParseCorrelationID` | 6                   | ~16        |
| `parseUserID`      | `id.ParseUserID`        | 4                   | ~6         |
| `parseCausationID` | `id.ParseCausationID`   | 3                   | ~5         |
| `parseRequestID`   | `id.ParseRequestID`     | 2                   | ~4         |

**Target package: `id/idtest`** (new subpackage of the `id` module).

- `id` is Layer 0 → every consumer already depends on it; **no go.mod changes** (subpackage of same module).
- Mirrors the established `event/eventtest` shared-test pattern.
- `testutil` is NOT viable: it depends on `event` (Layer 1), so `event` cannot import it (cycle).

**API design** — typed wrappers backed by a private generic (composition, minimal surface, short call sites):

```go
func MustParseAggregateID(s string) id.AggregateID { return must(id.ParseAggregateID(s)) }
// ... MustParseEventID, MustParseCorrelationID, MustParseCausationID, MustParseUserID, MustParseRequestID
```

Secondary: **`mustNewQuery`** (4 defs / 29 sites) → `query/querytest.MustNew` (same pattern, lower volume).

### ACCEPT (idiomatic — documented rationale)

| Groups                                                     | Pattern                                                                                       | Why accepted                                                       |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| 6, 35, 45, 65                                              | same-file self-matches                                                                        | art-dupl matching one file multiple times — not cross-file dup     |
| 24                                                         | `SharedBatchInsertEvents` (storage/sql/helpers.go ×2)                                         | same-file self-match                                               |
| 70                                                         | `Wrapf` (command/query errors)                                                                | idiomatic per-module 1-line error delegator                        |
| 11                                                         | `defer recover()` nil-db guards                                                               | same shape, **different** panic messages/constructors              |
| 25                                                         | `mustMarshal` (examples)                                                                      | examples are self-contained demos, must not import shared test pkg |
| 32                                                         | `fuzzEvent` (encryption/signing)                                                              | 2 files, module-specific fuzz harness                              |
| 58                                                         | `cbor.EncMode` init (codec/pebble)                                                            | pebble intentionally independent of codec                          |
| 3,10,13–23,26–31,33,34,36–44,46,47,49–57,59–64,66–69,71,72 | size-2: `{` literals, `for`/`if err` loops, `bus.Subscribe`, `b.Run` benches, struct literals | standard Go idioms; abstracting would harm readability             |

### EXCLUDE (none)

No generated code was found in-scope (cqrs-gen/api-stability emit to `cmd/` but nothing matched at -t 30).

## Task Breakdown (sorted by impact → effort)

### Phase A — Foundation (HIGH impact)

- **A1** Create `id/idtest/{doc,idtest}.go` (6 typed `MustParse*` + private generic `must`)
- **A2** Create `id/idtest/idtest_test.go` (table-driven, panic + success)
- **A3** Verify: `cd id && GOWORK=off go test ./idtest/... -count=1`

### Phase B — Migrate `event` module (HIGH impact — proves the pattern, lowest layer)

- **B1** Replace helpers in event: `enricher_test.go`, `test_helpers_test.go`, `event_type_clone_test.go`, `event_metadata_test.go`
- **B2** Verify: `cd event && GOWORK=off go test ./... -count=1`

### Phase C — Migrate remaining modules (mechanical, batched)

- **C1** `command/test_helpers_test.go`
- **C2** `schema/{golden,upcaster}_test.go`
- **C3** `signing/{benchmark,golden}_test.go`, `signing/internal/testutil/testutil.go`, `signing/multisig/extract_test.go`
- **C4** `snapshot/golden_test.go`
- **C5** `storage/{command_store,snapshot}_test.go`, `storage/memory/helpers_test.go`, `storage/memory/golden_test.go`, `storage/pebble/golden_test.go`
- **C6** `listing/golden_test.go`, `watermill/golden_test.go`
- **C7** `integration/event/creation_bdd_test.go`
- **C8** Verify each affected module: `GOWORK=off go test ./... -count=1`

### Phase D — Secondary: `query/querytest` (MEDIUM impact)

- **D1** Create `query/querytest` + migrate `mustNewQuery` (4 files: integration×2, middleware, query)
- **D2** Verify query/integration/middleware tests

### Phase E — Finalize

- **E1** Full workspace build + vet
- **E2** Re-run `art-dupl` to confirm clone-group reduction
- **E3** Update `AGENTS.md` module list + lint conventions with `idtest`/`querytest`
