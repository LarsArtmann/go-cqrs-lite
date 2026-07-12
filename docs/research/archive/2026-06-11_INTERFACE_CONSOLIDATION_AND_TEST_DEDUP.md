# Interface Consolidation & Test Deduplication Research

**Date**: 2026-06-11
**Scope**: catalog/ exporter API, interface split brains, test boilerplate clones, generic Transform proposal

---

## 1. ErrorExporter Split Brain — RESOLVED

### Problem

`catalog/exporter.go` defined two interfaces for the same concept:

```go
type Exporter[T any] interface {
    Export(cat *Catalog) T
}

type ErrorExporter interface {
    Export(cat *Catalog) error
}
```

`ErrorExporter` was a manual specialization of `Exporter[error]` — a split brain. Only `eventcatalog.Exporter` implemented it. Zero consumers outside this repo.

### Decision

Collapse to a zero-cost type alias with deprecation annotation:

```go
type ErrorExporter = Exporter[error]
// Deprecated: Use Exporter[error] instead.
```

**Rationale**: Type alias is backward-compatible. `staticcheck`/`golint` surfaces the deprecation. No consumer migration needed.

**Status**: Committed in `6828c32f`.

---

## 2. Interface Audit — No Other Split Brains

Full audit of all 37 interface definitions across the codebase. Result: **no other case** of a separate interface that's just a generic specialization.

### Structural Patterns Found (intentional, not harmful)

#### Pattern A: Sink/Source/Store — ISP applied 4 times

| Module     | Sink                              | Source                                                                | Store             |
| ---------- | --------------------------------- | --------------------------------------------------------------------- | ----------------- |
| event      | `EventSink` (Save, AppendBatch)   | `EventSource` (Load, LoadFromVersion, LoadToVersion, LoadToTimestamp) | `Store`           |
| command    | `CommandSink` (Save, AppendBatch) | `CommandSource` (Load, LoadFromTimestamp, LoadToTimestamp)            | `Store`           |
| snapshot   | `SnapshotSink` (Save, Delete)     | `SnapshotSource` (Load, LoadAtVersion)                                | `SnapshotStore`   |
| checkpoint | `CheckpointSink` (Save)           | `CheckpointSource` (Load)                                             | `CheckpointStore` |

All 4 compose via `type Store interface { XxxSink; XxxSource }`. All embed `io.Closer`.

**Why not unify**: Method signatures differ in domain types (`Event` vs `Snapshot` vs `Checkpoint`). Go generics cannot abstract over method parameter types (no higher-kinded types). The pattern is ISP applied consistently — same shape, different domain.

#### Pattern B: Encrypt/Decrypt ↔ Sign/Verify — "do + undo = both"

```go
// encryption
Encrypter         { Encrypt(plaintext []byte) (Ciphertext, error) }
Decrypter         { Decrypt(ciphertext Ciphertext) ([]byte, error) }
EncrypterDecrypter { Encrypter; Decrypter }

// signing
Signer         { Sign(event Event) (Signature, error) }
Verifier       { Verify(event Event, sig Signature) error }
SignerVerifier { Signer; Verifier }
```

Same composite pattern (one-directional + reverse + combined). Different domain types, different semantics.

**Why not unify**: `Encrypt` and `Sign` carry domain meaning that a generic `Apply` would destroy. `Verifier` breaks the Transform pattern entirely (2 args, no return value).

#### Pattern C: Concrete exporters — already unified

All 4 catalog exporter structs (`asyncapi`, `d2`, `eventcatalog`, `openapi`) implicitly satisfy `catalog.Exporter[T]` and share `type Option = catalog.Option[Exporter]`. This is the success story — no change needed.

### Not a duplicate

- `eventtest.AppendBatcher` — ISP split from `EventSink` for test assertions. Exists because `EventSink` embeds `io.Closer` which tests don't need.
- `Middleware func(Handler) Handler` — same shape in `command/` and `event/` but different `Handler` types. Go's type system prevents unification.

---

## 3. Generic Transform Interface — Rejected

### Proposal

```go
type Transform[In, Out any] interface {
    Apply(In) (Out, error)
}
```

This could theoretically unify `Encrypter`, `Signer`, etc.

### PRO

- One abstraction instead of N
- Generic middleware possible: `func Middleware[In, Out any](t Transform[In, Out]) Transform[In, Out]`
- Mathematically elegant (morphism in a category)

### CONTRA

1. **Kills domain vocabulary.** `Encrypt` tells you what happens. `Apply` tells you nothing.
2. **Method name mismatch.** Go interfaces are structural. `Encrypter` has method `Encrypt`, not `Apply`. So `Encrypter` would NOT satisfy `Transform[[]byte, Ciphertext]` unless you rename the method to `Apply`.
3. **Verifier breaks the pattern.** `Verify(event, sig) error` takes 2 args, returns no value. Not a `Transform[In, Out]`. You'd need a separate `Validator[T, U any]` — two generic abstractions instead of four domain interfaces.
4. **State coupling.** `Encrypter` and `Decrypter` must share the same key. Wrapping them in generic interfaces hides this invariant.
5. **Against Go culture.** `io.Reader`, `io.Writer`, `io.Closer` — each has ONE method with a domain-clear name. Nobody unified them into `type IOOp[T any] interface { Apply([]byte) (int, error) }`.

### Prior art

- **samber/lo**: Has `lo.Map(slice, func)` — a function, not an interface. For collection transformation.
- **samber/mo**: Has `Result[T].Map(func(T) U)` — functor bind on a concrete monadic wrapper. Not an interface for domain types.
- Neither provides a `Transform[A, B]` interface.

### Decision

**Rejected.** The `Transform[In, Out]` interface trades 4 self-documenting domain interfaces for 1 opaque generic. Not worth it.

---

## 4. Test Deduplication — Deferred

### Problem

`art-dupl` found 34 clones across `catalog/` test files. All are the same 2-line setup:

```go
reg := catalog.NewRegistry("TestCatalog", "1.0.0")
reg.AddService(catalog.Service{ID: "order-svc", Name: "Order Service", Version: "1.0.0"})
```

### Existing helpers

`catalog/internal/cattest/builders.go` already has `NewRegistry()`, `AddService()`, `AddCommandWithSchema()`, `AddEvent()`, etc. Some tests use them (e.g., `TestExporter_Export_Event`), most don't.

### Migration preview

**Before:**

```go
reg := catalog.NewRegistry("TestCatalog", "1.0.0")
reg.AddService(catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"})
reg.AddCommand("svc", basicCommand("DoStuff"))
cat := reg.Build()
```

**After:**

```go
reg := cattest.NewRegistry(t, "TestCatalog", "1.0.0")
cattest.AddService(t, reg, "svc", "Svc", "1.0.0")
reg.AddCommand("svc", basicCommand("DoStuff"))
cat := cattest.Build(t, reg)
```

Same line count. Struct literal replaced by helper call.

### Decision: Deferred

**Arguments for:**

- Hides phantom type casts (`catalog.Name(name)`, `catalog.Version(version)`)
- Single place to update if `Service` gains a required field
- `tb.Helper()` for cleaner stack traces

**Arguments against:**

- Same line count. Zero lines saved.
- Adds `t` parameter noise to every call
- `catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"}` is self-documenting — you see every field. `cattest.AddService(t, reg, "svc", "Svc", "1.0.0")` hides which fields are set
- The duplication is idiomatic Go test setup — not a maintenance burden
- For 23 of 34 sites, the Message struct is unique per test anyway

**Honest assessment**: Stylistic preference, not a clear improvement. The art-dupl report flagged similarity, but similarity ≠ harmful duplication. The test boilerplate is idiomatic Go.

---

## 5. BuildFlow Pre-Commit Hook — Note

The `.git/hooks/pre-commit` runs `buildflow --build-mode pre-commit --parallel --staged-only` with a 30s budget. It silently resets working tree files during commit. This caused the edit tool's changes to `catalog/exporter.go` to be lost repeatedly (the tool reported success but buildflow reverted the file before commit completed).

**Workaround**: Use python/bash to write files directly instead of the edit tool for this repo.
