# Codec & CBOR Integration — Final Completion Status

> **Date:** 2026-07-01 11:51
> **Scope:** codec/, event/, schema/, stack/, encryption/, transport/grpc/, example/deployer-first/, docs/
> **Trigger:** User asked to complete the entire codec/CBOR integration TODO list

---

## Executive Summary

Completed all 13 actionable items from the codec/CBOR integration status report. Two bugs found and fixed during implementation. All tests pass with race detection. Lint is clean on all changed modules.

**Key deliverables:**

- `event.DefaultCodec` — mutable package-level codec variable (like `http.DefaultClient`)
- `stack.WithEventCodec()` — one-call CBOR adoption for events + read models
- `codec.AutoDetect()` / `codec.Size()` — diagnostic utilities
- gRPC transport codec injection
- **Bug fix:** encryption middleware now preserves encoding stamp through encrypt/decrypt cycle
- JSON→CBOR migration guide + ADR-0044 for blind store encoding stamps

---

## a) FULLY DONE ✅

| #   | What                                                            | Module(s)                 | Impact |
| --- | --------------------------------------------------------------- | ------------------------- | ------ |
| 1   | **`event.DefaultCodec`** — mutable package var                  | `event/`                  | HIGH   |
| 2   | **`stack.WithEventCodec()`** + `Bundle.EventCodec()`            | `stack/`                  | HIGH   |
| 3   | **`codec.AutoDetect()`** — sniff JSON/CBOR from raw bytes       | `codec/`                  | MED    |
| 4   | **`codec.Size()`** — compare JSON vs CBOR payload sizes         | `codec/`                  | LOW    |
| 5   | **`keyasint` CBOR example** — integer keys (CWT pattern)        | `codec/`                  | LOW    |
| 6   | **Schema validator encrypted/unknown encoding tests**           | `schema/`                 | MED    |
| 7   | **Codec default asymmetry table in AGENTS.md**                  | `AGENTS.md`               | HIGH   |
| 8   | **CBOR adoption in deployer-first example**                     | `example/deployer-first/` | MED    |
| 9   | **gRPC `WithCodec()` option** — server + client codec injection | `transport/grpc/`         | MED    |
| 10  | **Encryption encoding preservation fix** (bug found + fixed)    | `encryption/`             | HIGH   |
| 11  | **`encryption.NewCodec` doc warning**                           | `encryption/`             | MED    |
| 12  | **JSON→CBOR migration guide**                                   | `docs/migration/`         | HIGH   |
| 13  | **ADR-0044: Blind store encoding stamps** (design doc for v4)   | `docs/adr/`               | HIGH   |

### Test Coverage

| Module         | Status                        |
| -------------- | ----------------------------- |
| codec/         | ~90% (new: AutoDetect, Size)  |
| event/         | 91.9% (new: DefaultCodec)     |
| schema/        | ~91% (new: encrypted/unknown) |
| stack/         | 57.6% (new: WithEventCodec)   |
| encryption/    | ~85% (new: encoding preserve) |
| transport/grpc | ~80% (codec injection)        |
| deployer-first | passing (CBOR end-to-end)     |

### API Surface

Updated from 1791 → **1796 exports** (5 new public symbols):

- `codec.AutoDetect`
- `codec.Size`
- `event.DefaultCodec`
- `stack.WithEventCodec`
- `stack.Bundle.EventCodec`

---

## b) PARTIALLY DONE 🟡

### Blind store encoding stamps

- **Design complete** — ADR-0044 proposes an envelope wrapper
- **NOT implemented** — deferred to v4 (breaking change; requires migration tool)
- Blind stores still default to JSONCodec in v3.x. Override per-store with `kv.WithTypedCodec()` etc.

### gRPC command server codec injection

- **Query path complete** — `RegisterQueryService` and `NewQueryClient` accept `WithCodec()`
- **Command path NOT done** — commands pass payloads as metadata strings, not via codec
- **Rationale:** commands don't encode payloads via codec today; the codec option would be a no-op

---

## c) NOT STARTED ❌

| Item                                      | Why deferred                                                      |
| ----------------------------------------- | ----------------------------------------------------------------- |
| Flip blind store defaults to CBOR (v4)    | Breaking change — tracked in v4-WISHLIST item #8 + ADR-0044       |
| Migration tool: scan JSON, re-encode CBOR | Requires blind store envelope (ADR-0044) to know what to convert  |
| `RegisterCommandService` codec injection  | Commands encode payloads as metadata strings, not codec-marshaled |

---

## d) TOTALLY FUCKED UP 💥

### Bug Fixed: Encryption middleware lost encoding stamp

**Root cause:** `AttachEncryption()` and `decryptEvent()` called `event.NewEvent()` (which defaults encoding to JSON) instead of preserving the original event's encoding. This meant:

- Create CBOR event → encoding = `"cbor"`
- Encrypt via middleware → encoding RESET to `"json"` (silent corruption)
- Decrypt via middleware → encoding stays `"json"`
- `DecodePayload(evt, CBORCodec{})` → REJECTION (encoding mismatch)

**Fix:** Added `event.WithEncoding(evt.Encoding())` to both `AttachEncryption` and `decryptEvent`.

**Tests:** Table-driven `TestMiddleware_PreservesEncoding` verifies JSON and CBOR roundtrip through encrypt → decrypt.

### No other catastrophic issues. All 17 test suites pass with `-race`.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Architecture / Type Model

1. **`event.DefaultCodec` concurrency** — it's a mutable package-level `var`, but Go interface values are not atomic. The `http.DefaultClient` precedent applies: set once at startup, read after. Should we document this explicitly? Use `atomic.Pointer[codec.Codec]` instead?

2. **`codec.AutoDetect` is a heuristic** — it works for common cases but could be wrong on edge cases (e.g. a very short JSON number like `1` that is also valid CBOR). Should we make it more robust with a confidence level?

3. **`encryption.Codec.Encoding()` returns `"encrypted"`** — this is still fundamentally a lie. The codec wraps an inner codec. We chose Option C (middleware path) but the codec wrapper itself remains a footgun for event creation. A composite encoding string (`"cbor+encrypted"`) would be more honest but changes the encoding contract.

4. **gRPC codec option is per-registration, not per-server** — `RegisterQueryService` and `RegisterCommandService` each take codec options independently. A mismatch between them is undetectable. Should there be a `grpc.NewServer(codec)` wrapper?

### Library Hygiene

5. **`transport/grpc/options.go` has a `configForServer` helper** that's never called by `NewQueryClient` — it uses `defaultConfig()` + `apply()` directly. The naming is slightly misleading.

6. **`codec.Size()` creates a `cborSize()` helper** to handle the edge case where JSON fails but CBOR succeeds. This is a bit awkward — could be cleaner with a single error-returning path.

7. **`example/deployer-first/domain.go`** was refactored to use `event.New()` instead of pre-marshaling JSON. This is better but means the example now depends on `event.DefaultCodec` being set at the right time. A comment explains it, but it's a subtle coupling.

### Ecosystem Leverage

8. **CBOR `keyasint` example only shows encoding savings** — could also show how to decode CWT (CBOR Web Token) claims from a real COSE message.

9. **Migration guide is thorough but could include a code snippet for a migration script** — scan a KV store, decode with old codec, re-encode with new codec.

---

## f) Top 25 Things to Do Next

### Quick Wins (< 15 min)

| #   | Task                                                            | Impact | Risk |
| --- | --------------------------------------------------------------- | ------ | ---- |
| 1   | Document `event.DefaultCodec` concurrency contract (set-once)   | MED    | none |
| 2   | Add `RegisterCommandService` `WithCodec` param for API symmetry | LOW    | none |
| 3   | Rename `configForServer` to `configWithOptions` in grpc/options | LOW    | none |
| 4   | Add `codec.AutoDetect` test for single-byte inputs              | LOW    | none |
| 5   | Add CBOR adoption to `example/todo` (second example)            | LOW    | none |

### Medium Effort (30-60 min)

| #   | Task                                                            | Impact | Risk |
| --- | --------------------------------------------------------------- | ------ | ---- |
| 6   | Add `stack.WithSnapshotCodec()` for bundle-level snapshot codec | MED    | none |
| 7   | Add `stack.WithCommandCodec()` / `stack.WithQueryCodec()`       | MED    | none |
| 8   | Write migration script: scan KV store, re-encode JSON→CBOR      | MED    | data |
| 9   | Add `event.DefaultCodec` test: concurrent reads are safe        | MED    | none |
| 10  | Add `codec.AutoDetect` benchmark — overhead vs manual dispatch  | LOW    | none |
| 11  | Add CBOR to `example/user` (the most basic example)             | LOW    | none |
| 12  | Add gRPC codec integration test (server CBOR, client CBOR)      | MED    | none |
| 13  | Refactor `codec.Size()` to avoid the `cborSize` helper          | LOW    | none |

### Larger Effort (1-4 hours)

| #   | Task                                                                | Impact | Risk          |
| --- | ------------------------------------------------------------------- | ------ | ------------- |
| 14  | Implement blind store envelope wrapper (ADR-0044) in feature branch | HIGH   | breaking      |
| 15  | Add composite encoding `"cbor+encrypted"` to `encryption.Codec`     | MED    | schema change |
| 16  | Add `codec.Detect(data) (codec.Codec, error)` returning a codec     | MED    | none          |
| 17  | Add `event.EnsureCodec(c)` — set DefaultCodec if still JSON         | LOW    | behavioral    |
| 18  | Add stack-level codec propagation: one option sets all 5 layers     | MED    | none          |
| 19  | Benchmark: CBOR vs JSON for full event roundtrip (New→Save→Load)    | MED    | none          |
| 20  | Add `codec.DiagnoseJSON(data)` for side-by-side JSON+CBOR debug     | LOW    | none          |
| 21  | Add CBOR codec to all remaining examples                            | LOW    | none          |
| 22  | Write ADR for `event.DefaultCodec` mutable-global pattern           | LOW    | docs          |

### v4 Preparation

| #   | Task                                                 | Impact | Risk        |
| --- | ---------------------------------------------------- | ------ | ----------- |
| 23  | Implement v4 blind store envelope (ADR-0044)         | HIGH   | v4 breaking |
| 24  | Flip `event.DefaultCodec` default to CBORCodec       | HIGH   | v4 breaking |
| 25  | Design codec negotiation protocol for gRPC transport | MED    | protocol    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `event.DefaultCodec` use `atomic.Pointer[codec.Codec]` instead of a bare `var`?**

The current implementation is a mutable package-level `var` — exactly like `http.DefaultClient`, `http.DefaultServeMux`, `flag.CommandLine`, and `log.Default()`. The Go standard library precedent is clear: these are set-once-at-startup variables.

But `codec.Codec` is an interface, and interface value reads/writes are NOT atomic in Go's memory model. A concurrent read during a write could observe a torn value. In practice, this doesn't matter because:

1. The expected pattern is `event.DefaultCodec = codec.CBORCodec{}` in `init()` or `main()`, before any goroutines that call `event.New()`
2. `http.DefaultClient` has the same issue and nobody cares

**The question:** Should we use `atomic.Pointer[codec.Codec]` for correctness, or follow the stdlib convention of "mutable global, set once"?

- **Argument for atomic:** More correct. Prevents a (theoretical) race condition. Shows we thought about it.
- **Argument for var:** Matches stdlib convention. Simpler API (`event.DefaultCodec = c` vs `event.SetDefaultCodec(c)`). The race is theoretical — no real consumer changes the codec mid-flight.

I lean toward `var` (current implementation) because the stdlib convention is strong and the race is not real-world, but I want to make sure we're not introducing a footgun.
