# ADR-0032: Merge `readmodel/` into `kv/`

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-20   |
| Status  | Implemented  |
| Decider | Lars Artmann |

## Context

`readmodel/` is a **120-line typed facade** over `kv.Store`:

```
readmodel/
├── store.go        # Store[T,K] wraps kv.Store with typed Get/Save/Delete/List
├── options.go      # prefix, codec options
├── backend.go      # Backend = Store alias (dead weight)
├── errors.go       # 3 sentinels
└── cache/
    ├── cached_store.go  # Cache[T,K] wraps Store[T,K] with an LRU
    └── options.go
```

This does not earn a top-level module. It is the _only_ consumer-facing read-model
abstraction, yet it hides behind `kv/` (Layer 0) which is where consumers
already look for "how do I store my view?" Keeping them separate forces consumers
to learn two packages for one job.

## Decision

**Merge `readmodel/` into `kv/` as typed helpers.**

```
kv/
├── store.go            # (unchanged) Store interface, MemStore
├── typed_store.go      # ← readmodel/store.go: TypedStore[T,K]
├── typed_options.go    # ← readmodel/options.go
├── cache.go            # ← readmodel/cache: Cache[T,K]
└── cache_options.go
```

- `readmodel.Store[T,K]` → `kv.TypedStore[T,K]` (verbatim move, rename type).
- `readmodel.cache.Cache[T,K]` → `kv.Cache[T,K]`.
- `readmodel.Backend` (the alias) is dropped — it added nothing.

## Alternatives Considered

- **Keep `readmodel/` as a thin re-export shim over `kv/`.** Rejected —
  indirection with no value. Two import paths for the same type is confusing.
- **Merge into `stack/` instead.** Rejected — `stack/` is for deployer presets
  (wiring), not for the typed-store abstraction itself. `kv/` is the right home;
  `stack/` consumes it.
- **Make `TypedStore` untyped.** Rejected — generics are the whole point. They
  give compile-time safety on key/value types.

## Consequences

- Import path change: `readmodel/v2` → part of `kv/v2`. This is a **v3 breaking
  change**.
- `kv/` grows by ~330 LOC (159 store + 173 cache), all moved verbatim from
  battle-tested `readmodel/`.
- `stack.Materialize` (ADR-0030) consumes `kv.TypedStore` directly — no extra
  import hop.
- One fewer module in `go.work`, one fewer place to look.

## Forward references

- Execution plan T09 (TypedStore), T10 (Cache).
- ADR-0030 (Materialize) depends on this — `Materialize[V,K].Store` is a
  `kv.TypedStore[V,K]`.
