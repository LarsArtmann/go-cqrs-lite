# API Surface Reduction — Improvement Ideas

> **Date:** 2026-06-21
> **Context:** 1,605 exports across 38 modules. Library IS used by external consumers.
> Every export is a contract. Every name collision is friction.
> **Goal:** Identify concrete, actionable reductions without breaking legitimate consumer needs.

---

## Current State

| Module | Exports | % | Verdict |
|--------|---------|---|---------|
| event | 315 | 20% | Core — mostly justified (87 eventtest + 228 prod) |
| catalog | 222 | 14% | 🔴 Most bloated — doc generator with 18 string-newtypes |
| storage | 198 | 12% | 🟡 75 exports in storage/sql subpkg are internal-only |
| command | 99 | 6% | Reasonable |
| query | 97 | 6% | Reasonable |
| stack | 85 | 5% | Reasonable (presets + Bundle + Materialize) |
| encryption | 85 | 5% | 🟡 4 parallel interfaces for encrypt+decrypt |
| middleware | 74 | 5% | Reasonable |
| otel | 69 | 4% | 🟡 High for a shim, but justified |
| kv | 52 | 3% | Reasonable |
| id | 46 | 3% | Excellent — phantom-typed IDs are the correct pattern |
| watermill | 37 | 2% | Reasonable |
| codec | 36 | 2% | Reasonable |
| signing | 34 | 2% | Reasonable |
| listing | 29 | 2% | Reasonable |
| decider | 27 | 2% | Lean — no bloat |
| schema | 24 | 1.5% | Reasonable |
| snapshot | 22 | 1% | Reasonable |
| transport/http | 20 | 1% | Reasonable |
| dispatcher | 18 | 1% | Reasonable |
| testutil | 8 | 0.5% | Minimal |
| prometheus | 8 | 0.5% | Minimal |

**Total:** 1,605 exports. By kind: 457 func, 438 method, 133 type, 108 struct, 92 var, 90 const, 60 interface.

---

## Problem 1: catalog String-Newtype Explosion (18 exports)

### What

`catalog/types.go` and `catalog/types_phantom.go` define 18 bare `type X string` aliases:

```go
// catalog/types_phantom.go
type Name string
type Version string
type Summary string
type Title string
type Address string
type Protocol string
type Host string
type Email string
type URL string
type ContentType string
type Method string
type Icon string
type Color string
type Language string
type Role string

// catalog/types.go
type UserID string
type Direction string
```

### Why It's Bad

1. **Not phantom-typed.** Unlike `id.Of[T]` which prevents mixing, these are bare aliases — `Name("x")` and `Title("x")` are interchangeable. They provide zero type safety.
2. **Name collisions.** Consumers importing `event` and `catalog` get two `Version` types, two `ContentType` types. This is the worst kind of collision — both compile, both work, both mean different things.
3. **18 exports for zero value.** A `string` field on the builder struct accomplishes the same thing with zero exports.

### Proposal

**Option A (non-breaking, additive):** Deprecate the types, keep them as aliases for `string`. Add a comment. Remove in v4.

**Option B (breaking, clean):** Replace all usages with `string`. The catalog builder methods already take typed params — change them to accept `string` directly. Consumers who used `catalog.Name("x")` just use `"x"`.

**Recommendation:** Option B at the v3 boundary. The catalog is a doc generator — its consumers are few, and the migration is mechanical (delete the type wrapper).

**Exports removed:** 18

---

## Problem 2: catalog Functional Option Verbosity (25 exports)

### What

25 exported functions, each returning an Option type:

```go
// catalog/channel_config.go
func ChannelAddress(addr string) ChannelOption
func ChannelProtocols(protocols ...string) ChannelOption
func ChannelMessages(msgIDs ...MessageID) ChannelOption
func ChannelDeliveryGuarantee(guarantee string) ChannelOption
func ChannelParameters(params map[string]ChannelParam) ChannelOption
func ChannelRoutes(routes ...ChannelRoute) ChannelOption
func ChannelOwners(owners ...string) ChannelOption
func ChannelBadges(badges ...Badge) ChannelOption

// catalog/domain_config.go
func DomainSends(msgs ...Ref) DomainOption
func DomainReceives(msgs ...Ref) DomainOption
func DomainEntities(entities ...string) DomainOption
func DomainBadges(badges ...Badge) DomainOption
func DomainOwners(owners ...string) DomainOption
func DomainAttachments(attachments ...Attachment) DomainOption

// catalog/service_config.go
func ServiceBadges(badges ...Badge) ServiceOption
func ServiceRepository(language, url string) ServiceOption
func ServiceWritesTo(storeIDs ...DataStoreID) ServiceOption
func ServiceReadsFrom(storeIDs ...DataStoreID) ServiceOption
func ServiceOwners(owners ...string) ServiceOption
func ServiceEntities(entities ...string) ServiceOption
func ServiceSpecifications(specs ...Specification) ServiceOption
func ServiceAttachments(attachments ...Attachment) ServiceOption

// catalog/message_config.go
func MsgOperation(method, path string, statusCodes ...string) MessageOption
func MsgBadges(badges ...Badge) MessageOption
func MsgRepository(language, url string) MessageOption
```

### Why It's Bad

Each function is `func X(value) Option { return func(b *builder) { b.field = value } }`. The functional-option pattern is overkill for a builder — a fluent builder (`builder.WithName("x").WithVersion("1")`) would be cleaner and have zero exported option functions.

### Proposal

Convert to fluent builder methods on the struct. The builder is already mutable internally — the option functions are just setters.

**Before (3 exports per builder field):**
```go
type Name string       // 1 export
type Option func(*T)   // 1 export
func WithName(n Name) Option { ... } // 1 export
```

**After (0 exports per builder field):**
```go
func (b *Builder) WithName(name string) *Builder { b.name = name; return b }
```

**Exports removed:** ~25 option funcs + ~5 option types = ~30

---

## Problem 3: storage/sql Internal-Only Exports (50 exports)

### What

`storage/sql` has 75 exports. Only consumer is `storage/` itself (uses 36 symbols). The remaining ~39 are public only because of the multi-module split.

Specifically:
- **Column constants:** `EventColumns`, `CommandColumns`, `QueryColumns` — SQL implementation details
- **Table name constants:** `TableEvents`, `TableCommands`, `TableQueries`, `TableSnapshots`, `TableCheckpoints`
- **Schema embeds:** `PostgresSchemaEmbed`, `SQLiteSchemaEmbed` — internal DDL
- **Scan helpers:** `ReconstructEvent`, `ScanSlice`, `MarshalMetadata` — internal plumbing

### Why It's Bad

These are implementation details of the SQL store. A consumer building a custom query against the events table doesn't need `EventColumns` — they need to know the schema (which is in the DDL files). Exporting `EventColumns` invites coupling to the exact column order.

### Proposal

**Option A (low effort, non-breaking for external consumers):** Collapse `storage/sql` back into `storage/`. Move all files from `storage/sql/*.go` → `storage/*.go`. Unexport everything that's only used internally (~50 symbols). The 36 symbols that `storage` uses become same-package references (no `sqlpkg.` prefix).

**Option B (zero effort):** Leave as-is but document that `storage/sql` is internal API.

**Recommendation:** Option A. The multi-module split provides zero value here — `storage/sql` has no external consumers and exists only to make `storage/` files shorter.

**Exports removed:** ~50

---

## Problem 4: encryption Interface Proliferation (5 interfaces)

### What

```go
// encryption/interfaces.go
interface Algorithmer { Algorithm() string }
interface Encrypter { Encrypt(plaintext []byte) (Ciphertext, error) }
interface Decrypter { Decrypt(ciphertext Ciphertext) ([]byte, error) }
interface EncrypterDecrypter { Encrypt + Decrypt + Algorithm() string }
interface KeyResolver { ResolveKey(ctx, keyID) (Key, error) }
```

### Why It's Bad

1. **`Algorithmer` is a one-method interface.** It exists only so `Algorithm()` can be called on an `EncrypterDecrypter`. But `EncrypterDecrypter` already embeds `Algorithm()` — so `Algorithmer` is redundant.
2. **`Encrypter` and `Decrypter` as separate interfaces follow ISP,** but in practice every consumer needs both. Nobody encrypts without also decrypting. The split adds cognitive overhead with no real-world benefit.
3. **`KeyResolver` is fine** — that's a genuinely separate concern.

### Proposal

1. **Delete `Algorithmer`** — fold `Algorithm() string` into `Encrypter` and `Decrypter` directly.
2. **Keep `Encrypter` and `Decrypter`** as separate interfaces (ISP is correct in principle), but document that `EncrypterDecrypter` is the primary interface consumers should depend on.
3. **Delete `EncrypterDecrypter`** if it's just `Encrypter + Decrypter + Algorithmer` — consumers can write `interface { Encrypter; Decrypter }` inline, or we keep it as a convenience alias.

**Recommendation:** Delete `Algorithmer`. Keep `Encrypter`, `Decrypter`, `EncrypterDecrypter` (convenience). Net: -1 interface.

**Exports removed:** 1 interface type

---

## Problem 5: Option Type Proliferation (10+ types)

### What

Every module exports its own `Option` type:

```
event.Option, command.Option, query.Option, catalog.Option,
signing.Option, encryption.Option, stack.Option, kv.CacheOption,
snapshot.Option, schema.ValidatorOption, decider.RepositoryOption
```

### Why It's Mildly Bad

This is inherent to functional options in Go — each module needs its own option type. It's not a bug, it's a pattern cost. But it contributes to the export count and requires consumers to import the right `Option` type.

### Proposal

**Not actionable without a major pattern change.** The options are correct. Leave them.

**Alternative (if we want radical reduction):** Switch to struct-based config (`type Config struct { ... }`) which eliminates all option types but loses the ergonomic `WithX()` API. Not recommended — the current pattern is idiomatic Go.

**Exports removed:** 0 (not worth the tradeoff)

---

## Problem 6: catalog Module Scope (222 exports = 14% of total)

### What

The `catalog` module is a documentation generator: it reads Go types and produces AsyncAPI, OpenAPI, EventCatalog, and D2 diagrams. It has:
- 26 exported structs (Builder, Catalog, Channel, Domain, Flow, Message, Service, etc.)
- 40 exported types (mostly string aliases)
- 47 exported functions (mostly option funcs)
- 90 exported methods (builder accessors)
- 12 exported constants
- 5 exported vars
- 2 exported interfaces

### Why It's Bad

The catalog is NOT a CQRS primitive. It's a developer tool that happens to work with CQRS types. A consumer who just wants event sourcing doesn't need 222 exports of documentation generator types in their dependency graph.

### Proposal

**Option A:** Move `catalog/` to `contrib/catalog/` or `tooling/catalog/`. Signals "this is optional, not core CQRS."

**Option B:** Keep it where it is but split internally: `catalog` (core types + Registry) vs `catalog/exporters` (AsyncAPI/OpenAPI/D2/EventCatalog generators). The exporters are where most of the builder complexity lives.

**Option C (nuclear):** Delete catalog entirely. It's 5,694 LOC of code that generates docs nobody has confirmed they need. If a consumer needs AsyncAPI, they can use the `SchemaFromType[T]()` function directly.

**Recommendation:** Option A (move to `contrib/`). Low risk, clear signal.

**Exports removed from core:** 222

---

## Problem 7: Duplicate Type Names Across Modules

### What

These types have the same name in multiple modules:

| Name | Modules | Count |
|------|---------|-------|
| Option | event, command, query, catalog, signing, encryption, stack, snapshot, schema, kv | 10 |
| Middleware | event, command, query, middleware | 4 |
| Handler | event, command, query, dispatcher | 4 |
| Type | event, command, query | 3 |
| MetadataKey | event, command, query | 3 |
| Family | event, encryption, otel | 3 |
| Error | event, command, query | 3 |
| Dispatcher | event, command, query | 3 |
| Version | event, catalog | 2 |
| UserID | catalog, (example) | 2 |
| AggregateType | event, catalog | 2 |
| AggregateRef | event, catalog | 2 |

### Why It's Bad

When a consumer imports multiple modules, they get qualified-name collisions. `event.Version` vs `catalog.Version` — both compile, both are called `Version`, but they're different types. This is confusing.

### Proposal

Most of these are **correct** — each module's `Option` is a different type for a different purpose. The qualified import (`event.Option` vs `command.Option`) is how Go resolves this.

The only real problem is `Version` (event vs catalog) and `AggregateType` (event vs catalog) — these should use the `event` types, not redefine them. If catalog's `Version` is just `string`, use `string` (see Problem 1).

**Exports removed:** 0 (the duplicates are inherent to multi-module Go)

---

## Summary: Impact Ranking

| # | Problem | Exports Removed | Effort | Breaking | Priority |
|---|---------|----------------|--------|----------|----------|
| 1 | catalog string-newtypes | 18 | 1h | Yes | High |
| 2 | catalog option verbosity | ~30 | 2h | Yes | High |
| 3 | storage/sql internals | ~50 | 2h | No | **Highest** |
| 4 | encryption interface proliferation | 1 | 30min | Yes | Medium |
| 5 | Option type proliferation | 0 | — | — | Not actionable |
| 6 | catalog module scope | 222 | 5min (move) | No | High |
| 7 | Duplicate type names | 0 | — | — | Not actionable |

### Non-Breaking Quick Wins (can do NOW)

| Action | Exports | Effort | Breaking |
|--------|---------|--------|----------|
| Collapse `storage/sql` → `storage` | ~50 | 2h | No |
| Move `catalog` → `contrib/catalog` | 222 (from core) | 5min | No |
| **Total non-breaking reduction** | **~272** | ~2h | No |

### Breaking Changes (for v3)

| Action | Exports | Effort | Breaking |
|--------|---------|--------|----------|
| Replace catalog string-newtypes with `string` | 18 | 1h | Yes |
| Convert catalog option funcs to builder methods | ~30 | 2h | Yes |
| Delete `encryption.Algorithmer` | 1 | 30min | Yes |
| **Total breaking reduction** | **~49** | ~3.5h | Yes |

### Theoretical Maximum (all changes)

| Scenario | Exports | Reduction |
|----------|---------|-----------|
| Non-breaking only | 1,605 → 1,333 | -17% |
| Breaking + non-breaking | 1,605 → 1,284 | -20% |
| + Delete catalog entirely | 1,605 → 1,062 | -34% |

---

## What's Already Well-Designed (Don't Touch)

| Module | Why It's Good |
|--------|---------------|
| `id` (46) | Phantom-typed IDs via `id.Of[T]` are the correct pattern |
| `event` errors (15) | Well-classified sentinels with 5-family taxonomy |
| `event` consts (16) | `MetadataKeyTombstone`, `ModeLive`, `ModeReplay` are genuinely needed |
| `decider` (27) | Lean — `Decider[State]`, `Repository[State]`, `Execute`, `Load` |
| `codec` (36) | Clean codec interface with JSON/CBOR/Raw implementations |
| `snapshot` (22) | Minimal types + strategy pattern |
| `dispatcher` (18) | Generic `Dispatcher[H, M]` — no bloat |

---

## Open Questions

1. **How many external consumers does `catalog` have?** If zero, moving to `contrib/` or deleting is obvious. If some, we need migration path.

2. **Does anyone import `storage/sql` directly?** If not, collapsing into `storage/` is zero-risk. If yes, we need a deprecation period.

3. **Are the encryption sub-interfaces (`Encrypter` without `Decrypter`) used by any consumer?** If not, consolidating is safe.

4. **Should we adopt a maximum-exports-per-module guideline?** E.g., "core modules ≤ 100 exports, contrib modules unlimited." This would force catalog to be split or moved.

---

_Next step: Validate with real consumers which exports they actually use, then prioritize the non-breaking reductions first._
