# Dedup Acceptance Register

> Every clone group that was reviewed and explicitly ACCEPTED during dedup sessions.
> Each entry has a one-line rationale so the next session does not re-evaluate from scratch.
>
> **Measurement context**: `art-dupl --type-aware -t 2` shows 75 clone groups (Health Score: A, 0.2% duplication).
> At the skill's recommended threshold `-t 5`, there are **0 clone groups** — all remaining
> duplication is 1-5 statement snippets.

---

## Production code — ACCEPTED (module-specific patterns)

| Clone group | Location | Rationale |
|-------------|----------|-----------|
| ErrHandlerNotFound | `command/errors.go`, `dispatcher/errors.go`, `query/errors.go` | Module-specific sentinel errors with unique codes ("command.handler_not_found" etc). Each module owns its error vocabulary. |
| ErrNoDatabase | `stack/sqlite/view_models.go`, `stack/turso/view_models.go` | Module-specific sentinel. errorfamily compares code+family, not pointer identity, so each must be defined separately. |
| SQLViewModel facade | `stack/sqlite/view_models.go`, `stack/turso/view_models.go` | Thin 1-line wrappers around `sqlopt.SQLViewModel`. The shared implementation is already extracted. The remaining "duplication" is documentation + the facade signature. |
| wrapClosed guard | `storage/memory/*.go` (17 call sites) | `wrapClosed` helper already exists in `errors.go`. The `if err := wrapClosed(...); err != nil { return err }` is the standard Go early-return guard — cannot be simplified further without changing function signatures. |
| OTel span start+defer | `storage/pebble/journal.go:30`, `stream.go:152`, `journal.go:62`, `stream.go:174` | Standard OTel instrumentation. 2-line pattern (start span, defer end). Span name is the parameter. Extracting a helper for 2 occurrences of 2 lines saves nothing. |
| docserver HTML | `catalog/docserver/html.go` (`scalarHTML`, `asyncAPIHTML`) | Two different HTML pages (Scalar.js vs AsyncAPI React) sharing basic HTML document structure. Extracting the HTML skeleton would require passing 20+ lines of HTML as parameters — overengineering. |
| COSE encrypt/decrypt | `encryption/cose.go` (coseXChaCha20, coseAESGCM) | Two different cipher implementations with identical error-wrapping shape but different algorithm identifiers. The structural similarity is inherent to the COSE Encrypt/Decrypt interface contract. |
| Mutex Lock+defer Unlock | `storage/turso/indexing/auto.go`, `decider/cache.go`, etc. | Standard Go mutex pattern. `mu.Lock(); defer mu.Unlock()` is the minimum expression. |
| Multidb secondary backend error | `stack/postgres/multidb.go`, `stack/sqlite/multidb.go`, `stack/turso/multidb.go` | Module-specific error wrapping for secondary backend creation. 3 modules, 3 unique error codes. |
| Metaengine structValue | `metaengine/execute.go:216`, `metaengine/reflect.go:116` | Same guard check in two different contexts (execute vs reflect). 2 occurrences, 4 tokens. |
| strings.Builder init | `catalog/eventcatalog/writer.go`, `writer_schemas_txt.go` | Standard Go `var buf strings.Builder` idiom. |
| kv/mem checkClosed+fn | `kv/mem.go:47`, `kv/mem.go:62` | Standard guard-then-execute pattern. 2 occurrences. |
| cqrs-bench openOutput | `cmd/cqrs-bench/output.go:35,51,65` | 3 one-liner calls to openOutput with different file names. The helper already exists. |
| Turso sync/indexing error wrapping | `storage/turso/sync.go`, `indexing/optimizations.go` | Module-specific error handling for Turso database operations. Unique codes per operation. |

## Test code — ACCEPTED (standard Go testing idioms)

| Clone group | Occurrences | Rationale |
|-------------|-------------|-----------|
| `t.Parallel()` | 200+ across all test files | Standard Go test parallelism. Cannot be "extracted" — it's a 1-line statement that must appear in every test function. |
| `t.Helper()` | 4 in contracttest, 2 in eventtest | Standard Go test helper marker. 1-line statement. |
| `t.TempDir()` | 18 occurrences | Standard Go temp directory creation. |
| `id.NewStreamID(...)` | 23 occurrences | Test fixture creation — different stream IDs in each test. |
| `context.WithTimeout(...)` | 15 occurrences | Standard Go context timeout pattern in tests. |
| `NewWithT(...)` | 19 occurrences | Gomega test setup idiom. |
| `wantErr` sentinel check | 16 occurrences | Standard Go error-checking pattern in tests. |
| `ParseStreamID(...)` | 16 occurrences | Test fixture creation via idtest helpers. |
| `CBORCodec{}` | 16 occurrences | Codec test setup — different payloads in each test. |
| `newTestViewStore(...)` | 12 occurrences | Storage test fixture creation. |

---

## Extractions SHIPPED (previous + this session)

| Helper | Module | Call sites collapsed | Session |
|--------|--------|---------------------|---------|
| `wrapInfraOrOK(err, code, msg)` | storage/pebble | 8 | Session 3 |
| `OpenDBOrErr(driver, dsn, code)` | stack/sqlopt | 2 presets (6 close boilerplate) | Session 3 |
| `loadAndDecrypt(events, err)` | encryption | 5 functions | Session 3 |
| `TestBuilder()` (no-tb variant) | catalog/internal/cattest | 2 BDD test calls | Session 3 |
| `wrapTransientOrOK(err, code, msg)` | storage/readmodel (kv_sql) | 4 | Session 4 |
| `wrapInfraOrOK(err, code, msg)` | storage/readmodel (kv_sql) | 3 | Session 4 |
| `MarshalBase64JSONWithModule(raw, module, noun)` | codec (shared by encryption + signing) | 2 MarshalJSON methods | Session 4 |
