# Dedup Acceptance Register

> Every clone group that was reviewed and explicitly ACCEPTED during dedup sessions.
> Each entry has a one-line rationale so the next session does not re-evaluate from scratch.
>
> **Measurement context**: `art-dupl --type-aware -t 2` shows 50 clone groups (down from 72 in prior session, 0.2% duplication, ~290 duplicate lines / 199k LOC).
> At the skill's recommended threshold `-t 5`, there are **0 clone groups** — all remaining
> duplication is 1-5 statement snippets.

---

## Production code — ACCEPTED (module-specific patterns)

| Clone group                            | Location                                                                                       | Rationale                                                                                                                                                                                                                                                                                                         |
| -------------------------------------- | ---------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ErrHandlerNotFound                     | `command/errors.go`, `dispatcher/errors.go`, `query/errors.go`                                 | Module-specific sentinel errors with unique codes ("command.handler_not_found" etc). Each module owns its error vocabulary.                                                                                                                                                                                       |
| ErrNoDatabase                          | `stack/sqlite/view_models.go`, `stack/turso/view_models.go`                                    | Module-specific sentinel. errorfamily compares code+family, not pointer identity, so each must be defined separately.                                                                                                                                                                                             |
| SQLViewModel facade                    | `stack/sqlite/view_models.go`, `stack/turso/view_models.go`                                    | Thin 1-line wrappers around `sqlopt.SQLViewModel`. The shared implementation is already extracted. The remaining "duplication" is documentation + the facade signature.                                                                                                                                           |
| wrapClosed guard                       | `storage/memory/*.go` (17 call sites)                                                          | `wrapClosed` helper already exists in `errors.go`. The `if err := wrapClosed(...); err != nil { return err }` is the standard Go early-return guard — cannot be simplified further without changing function signatures.                                                                                          |
| OTel span start+defer                  | `storage/pebble/journal.go:30`, `stream.go:152`, `journal.go:62`, `stream.go:174`              | Standard OTel instrumentation. 2-line pattern (start span, defer end). Span name is the parameter. Extracting a helper for 2 occurrences of 2 lines saves nothing.                                                                                                                                                |
| docserver HTML                         | `catalog/docserver/html.go` (`scalarHTML`, `asyncAPIHTML`)                                     | Two different HTML pages (Scalar.js vs AsyncAPI React) sharing basic HTML document structure. Extracting the HTML skeleton would require passing 20+ lines of HTML as parameters — overengineering.                                                                                                               |
| COSE encrypt/decrypt                   | `encryption/cose.go` (coseXChaCha20, coseAESGCM)                                               | Two different cipher implementations with identical error-wrapping shape but different algorithm identifiers. The structural similarity is inherent to the COSE Encrypt/Decrypt interface contract.                                                                                                               |
| Mutex Lock+defer Unlock                | `storage/turso/indexing/auto.go`, `decider/cache.go`, etc.                                     | Standard Go mutex pattern. `mu.Lock(); defer mu.Unlock()` is the minimum expression.                                                                                                                                                                                                                              |
| Multidb secondary backend error        | `stack/postgres/multidb.go`, `stack/sqlite/multidb.go`, `stack/turso/multidb.go`               | Module-specific error wrapping for secondary backend creation. 3 modules, 3 unique error codes.                                                                                                                                                                                                                   |
| Metaengine structValue                 | `metaengine/execute.go:216`, `metaengine/reflect.go:116`                                       | Same guard check in two different contexts (execute vs reflect). 2 occurrences, 4 tokens.                                                                                                                                                                                                                         |
| strings.Builder init                   | `catalog/eventcatalog/writer.go`, `writer_schemas_txt.go`                                      | Standard Go `var buf strings.Builder` idiom.                                                                                                                                                                                                                                                                      |
| kv/mem checkClosed+fn                  | `kv/mem.go:47`, `kv/mem.go:62`                                                                 | Standard guard-then-execute pattern. 2 occurrences.                                                                                                                                                                                                                                                               |
| cqrs-bench openOutput                  | `cmd/cqrs-bench/output.go:35,51,65`                                                            | 3 one-liner calls to openOutput with different file names. The helper already exists.                                                                                                                                                                                                                             |
| Per-module `wrapInfraOrOK` helper body | `storage/memory/errors.go`, `storage/pebble/helpers.go`, `storage/readmodel/kv_sql.go`         | The 5-line helper body (`if err == nil { return nil }; return errorfamily.WrapInfrastructure(...)`) appears in 3 modules with separate go.mod files. Module isolation principle: pebble is not SQL-backed, so not all 3 share storage/sql/. NOT promoted to a shared package for a 5-line function. See ADR-0069. |
| cqrs-lint SelectorFromExpr guard       | `cmd/cqrs-lint/pkg/analyzer/scanner.go:144`, `scanner_calls.go:22`                             | Standard AST guard: extract selector, bail if not found. Only 2 occurrences of the exact 4-statement pattern. ADR-0069 says inline for 1-2 occurrences. Functions diverge after the guard.                                                                                                                        |
| Turso sync/indexing error wrapping     | `storage/turso/sync.go` (Push, Checkpoint, HealthCheck), `indexing/optimizations.go` (Analyze) | 4 sites of `if err != nil { return WrapInfrastructure(...) }; return nil`. Separate module (own go.mod). Adding wrapInfraOrOK here would extend the 3-way per-module clone to 4-way. Unique error codes per operation.                                                                                            |

## Test code — ACCEPTED (standard Go testing idioms)

| Clone group                | Occurrences                       | Rationale                                                                                                              |
| -------------------------- | --------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `t.Parallel()`             | 200+ across all test files        | Standard Go test parallelism. Cannot be "extracted" — it's a 1-line statement that must appear in every test function. |
| `t.Helper()`               | 4 in contracttest, 2 in eventtest | Standard Go test helper marker. 1-line statement.                                                                      |
| `t.TempDir()`              | 18 occurrences                    | Standard Go temp directory creation.                                                                                   |
| `id.NewStreamID(...)`      | 23 occurrences                    | Test fixture creation — different stream IDs in each test.                                                             |
| `context.WithTimeout(...)` | 15 occurrences                    | Standard Go context timeout pattern in tests.                                                                          |
| `NewWithT(...)`            | 19 occurrences                    | Gomega test setup idiom.                                                                                               |
| `wantErr` sentinel check   | 16 occurrences                    | Standard Go error-checking pattern in tests.                                                                           |
| `ParseStreamID(...)`       | 16 occurrences                    | Test fixture creation via idtest helpers.                                                                              |
| `CBORCodec{}`              | 16 occurrences                    | Codec test setup — different payloads in each test.                                                                    |
| `newTestViewStore(...)`    | 12 occurrences                    | Storage test fixture creation.                                                                                         |

---

## Extractions SHIPPED (previous + this session)

| Helper                                           | Module                                 | Call sites collapsed            | Session   |
| ------------------------------------------------ | -------------------------------------- | ------------------------------- | --------- |
| `wrapInfraOrOK(err, code, msg)`                  | storage/pebble                         | 8                               | Session 3 |
| `OpenDBOrErr(driver, dsn, code)`                 | stack/sqlopt                           | 2 presets (6 close boilerplate) | Session 3 |
| `loadAndDecrypt(events, err)`                    | encryption                             | 5 functions                     | Session 3 |
| `TestBuilder()` (no-tb variant)                  | catalog/internal/cattest               | 2 BDD test calls                | Session 3 |
| `wrapTransientOrOK(err, code, msg)`              | storage/readmodel (kv_sql)             | 4                               | Session 4 |
| `wrapInfraOrOK(err, code, msg)`                  | storage/readmodel (kv_sql)             | 3                               | Session 4 |
| `MarshalBase64JSONWithModule(raw, module, noun)` | codec (shared by encryption + signing) | 2 MarshalJSON methods           | Session 4 |
| `parseTimePointer(src, dialect)`                 | storage/sql                            | Postgres + DuckDB ParseTime     | Session 7 |
| `iterJSON(prefix, upperBound, yield)`            | metaengine/pebbleengine                | MultiGet + LogTail(limit<=0)    | Session 7 |

---

## Clone Reduction Summary (75 → 50 groups)

Over sessions 1–7 (2026-07-23 through 2026-07-29), art-dupl clone groups were
reduced from 75 to 50. The 25 eliminated groups span production-code extractions
plus accepted test idioms already inventoried above.

The eliminated groups follow the same triage pattern across sessions:

1. **kv_sql error wrapping** — `wrapInfraOrOK`/`wrapTransientOrOK` extracted as
   per-module helpers (ADR-0069). Eliminated 7 inline call-site clones across
   `storage/memory`, `storage/pebble`, `storage/readmodel`. The helper body
   itself appears in 3 modules (capped per ADR-0069), a net -2 groups.

2. **catalog MarshalBase64JSON** — extracted `MarshalBase64JSONWithModule` as a
   shared helper in `codec/`, eliminating 2 MarshalJSON method clones in
   `encryption` and `signing`.

3. **eventtest TestBuilder** — eliminated the no-tb variant call duplication
   by routing both BDD paths through the same builder.

4. **storage/sql ParseTime** — Postgres and DuckDB both extracted the same
   `*time.Time` pointer assertion. Extracted `parseTimePointer(src, dialect)`
   that takes the dialect name as the unique parameter. Eliminated 1 group.

5. **metaengine/pebbleengine JSON iter** — `MultiGet` and `LogTail(limit<=0)`
   both opened a Pebble iterator and decoded every value as JSON. Extracted
   `iterJSON(prefix, upperBound, yield)` callback-style helper. Eliminated 1
   group. The reverse-iteration branch of `LogTail` (limit>0) cannot use the
   helper because `Prev()` semantics differ.

The remaining 50 accepted groups are documented in the table above and in the
session reports. Each is either:

- A cross-module pattern that can't be shared without violating multi-module
  isolation (per ADR-0069)
- A test helper specific to a module's types
- A standard Go idiom (e.g., `if err != nil { return err }`)
