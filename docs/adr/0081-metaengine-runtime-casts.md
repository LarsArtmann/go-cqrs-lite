# Why Metaengine Uses Runtime Casts (and Why Go Can't Fix It Yet)

> **Date:** 2026-07-31
> **Context:** Analysis of the `any` boundaries in `metaengine/` and whether upcoming Go language features could eliminate them.

---

## The Root Cause: Go Has No Existential Types

Go generics can't express "a Store holding `Query[A,B]` **and** `Query[C,D]` **and** `Query[E,F]` while preserving their type parameters." Every query has **different** type parameters, but Store is a **single** container. The type parameters must be erased at the container boundary.

This is the **existential type problem** — `∃I,V. Query[I,V]` — which Go doesn't support.

## The 4 Cast Sites (All Same Root Cause)

| Site | Signature | Why `any` |
|------|-----------|-----------|
| `Plan` | `args ...any` | Heterogeneous `Query[I1,V1]`, `Query[I2,V2]` in one call |
| `Apply` | `payload any` | One event type fans out to folds expecting **different** payload types |
| `Execute` | returns `any` | Return type depends on **which** query was dispatched |
| `reify[V]` | `row any → V` | Engine stores values as `any`/`[]byte`; reader reifies |

Every cast traces back to the same constraint: **Store is a heterogeneous container of generic queries, and Go can't type that.**

## The Pattern: Generic at the Edges, Erased in the Middle

```
Consumer code (typed)          Store boundary (erased)         Engine (erased)
─────────────────────          ─────────────────────           ───────────────
Query[I,V]          ──cast──►  queryMeta interface  ──►        Engine
TypedReader[V]      ◄─cast──   []any rows           ◄──        MapScan
ExecuteTyped[I,V]   ◄─cast──   any result           ◄──        Apply
```

The casts happen **exactly** at the Store boundary — the only place where heterogeneous generic types must coexist. Inside the edges (Query declaration, TypedReader usage), everything is fully typed.

## Can We Eliminate Them?

**Option A — Code generation:** Generate a typed Store per query set. Eliminates all casts but adds boilerplate and a build step. This is what `cmd/cqrs-gen` could do.

**Option B — Callback-based:** Instead of storing typed data, pass typed callbacks. But the dispatch problem remains — you still need `any` at the fan-out point.

**Option C — Keep as-is:** The current pattern is the **idiomatic Go solution** for existential types. The casts are bounded, type-safe (they fail loudly on mismatch), and pay zero cost when not hit. Every major Go library with heterogeneous generic containers (`database/sql` drivers, `encoding/json`, `gorilla/mux`) uses the same pattern.

## Verdict

The runtime casts are **structurally necessary** given Go's type system. They're not a design smell — they're the correct boundary between typed consumer code and the type-erased heterogeneous Store. The only way to eliminate them entirely is code generation, which trades runtime casts for build-time complexity.

---

## Will Go 1.27 Help?

**No. Go 1.27 won't solve this.**

### Go 1.27 (August 2026) — What Lands

**Generic methods on concrete types** — the big one ([#77273](https://github.com/golang/go/issues/77273), accepted, implemented):

```go
func (s *Store) Get[I, V any](input I) (V, error) { ... }  // legal in 1.27
```

But **interface methods still can't have type parameters**. A generic concrete method does not satisfy any interface. So `Engine` can't declare `MapScan[V any]()`, which is exactly what we'd need to eliminate the `any` return from engine backends.

The release notes confirm: *"methods of interfaces may not declare type parameters nor can interface methods be implemented by generic methods."*

### Associated Types — Closest to What We Need, But Far Away

Proposal [#80448](https://github.com/golang/go/issues/80448), filed **July 17, 2026**:

```go
type Container interface {
    type Element any
    Add(Element)
    At(int) (Element, bool)
}
```

This would let `Query` declare an associated `Input` and `Result` type, and Store could reference `Q.Input` / `Q.Result` without erasure to `any`.

**Status: Open, early review, no prototype, no target release.** Realistically Go 1.29+ at earliest, possibly never.

And critically: the proposal **explicitly excludes** existential types (storing heterogeneous `Container` values in one slice). That's the exact feature metaengine needs.

> *"Allowing plain `var g Graph` would hide both the implementation and the types needed to call its methods. Supporting that requires existential types, which are outside this proposal."*

### Feature Matrix

| Feature | Go 1.27? | Solves our casts? |
|---------|----------|-------------------|
| Generic concrete methods | **Yes** | Partially — removes casts on `Store.Get[I,V]`, but not inside Engine/Apply |
| Generic interface methods | No (explicitly out of scope) | Would solve Engine-level casts |
| Associated types | No (proposal filed Jul 2026) | Would solve Query erasure |
| Existential types | No (explicitly excluded) | Would solve heterogeneous Store |
| Heterogeneous containers | No proposal at all | Would solve the fundamental problem |

### Looking Further Ahead

**Go 1.28** has a Collections working group proposal ([#80590](https://github.com/golang/go/issues/80590)) for generic stdlib types (`set.Set`, `hash.Map`, `tree.Map`, `heap/v2.Heap`). But these are **standard library** additions, not language-level type system changes that would help metaengine.

### Bottom Line

The `any` boundary at Store/Engine is **structurally necessary for the foreseeable future**. Go's type system simply can't express "a container of heterogeneous generic queries." The casts will remain until:

1. **Associated types** land and include existential support (Go 1.29+ at earliest, probably later)
2. **Generic interface methods** are accepted (no proposal exists)
3. **Code generation** fills the gap (`cmd/cqrs-gen` generating typed `TaskStore` from query declarations)

Until then, the runtime casts are the **idiomatic Go solution** for this class of problem.

---

## References

- [#77273 — Generic type parameters on methods](https://github.com/golang/go/issues/77273) (Accepted, Go 1.27)
- [#80448 — Associated types in interfaces](https://github.com/golang/go/issues/80448) (Open, Jul 2026)
- [#80590 — Generic collections umbrella](https://github.com/golang/go/issues/80590) (Go 1.28)
- [Go 1.27 draft release notes](https://go.dev/doc/go1.27)
- [Generic Interfaces blog post](https://go.dev/blog/generic-interfaces) (Axel Wagner)
