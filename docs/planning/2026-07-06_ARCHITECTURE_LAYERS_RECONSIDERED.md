# Architecture Layers Reconsidered: Current State & First-Principles Future

> **Status:** ANALYSIS — no execution without review
> **Scope:** Full module dependency architecture of go-cqrs-lite
> **Trigger:** "Why is `kv/` Layer 0?" — the question that exposed the cracks

---

## 1. The Question That Started This

> "Why is kv/ Layer 0?"

**It isn't.** `kv/` depends on `codec/` (for `TypedStore` serialization) and `otter/v2` (for `Cache`). By the project's own layering rules, `kv/` is Layer 1, not Layer 0.

But this is a symptom, not the disease. The layer numbering is broken because it conflates **three distinct architectural concerns** into a single linear stack. The numbers don't represent real boundaries — they're dependency-order annotations that lost their meaning as the codebase grew.

This document audits the current layering honestly, then proposes a first-principles target architecture.

---

## 2. Current State: The Honest Audit

### 2.1 The Documented Layer System (from AGENTS.md)

```
Layer 0: id/, dispatcher/, codec/, kv/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher, ro), query/ (→dispatcher, ro)
Layer 2: schema/ (→event), snapshot/ (→event), projection/ (→event), idempotency/ (→command, event, id, kv), deriver/ (→command, event, id)
Layer 3: decider/ (→event, snapshot), graph/, scenario/, projectionhost/
Layer 4: storage/memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, listing/, watermill/, transport/http/, transport/grpc/, storage/pebble/, storage/turso/, prometheus/
Layer 6: stack/, stack/memory/, stack/sqlite/, stack/pebble/, stack/postgres/
Layer 7: catalog/, integration/, stack/bench/, examples/, cmd/
```

### 2.2 Layer Violations (what's actually true)

#### `kv/` is NOT Layer 0

```
kv/go.mod requires:
  github.com/larsartmann/go-cqrs-lite/codec/v3   ← Layer 0 dep
  github.com/maypok86/otter/v2                    ← external dep
  github.com/larsartmann/go-error-family          ← external dep
```

`kv/` depends on `codec/`. It's Layer 1. The `AGENTS.md` classification is wrong.

The raw KV interface (`Store`, `Reader`, `Writer`, `Iterator`, `Batch`, `MemStore`) IS Layer 0 — pure byte-level abstraction. But `kv/` also contains `TypedStore[T,K]` (depends on `codec.Codec` for serialization) and `Cache[T,K]` (depends on `otter`). These are higher-level concerns stuffed into the same module.

#### `event/` is NOT Layer 1 — it's a god module

```
event/go.mod DIRECT deps (as declared):
  codec/v3           ← Layer 0 (legitimate: payload encoding)
  event/v3/eventtest ← Layer 1 (test helper — should be test-only dep)
  id/v3              ← Layer 0 (legitimate: branded IDs)
  schema/v3          ← Layer 2 (test-only usage — NOT production code)
  snapshot/v3        ← Layer 2 (test-only usage — NOT production code)
  storage/memory/v3  ← Layer 4 (test-only usage — NOT production code)
```

Three of these "direct production dependencies" (`schema/`, `snapshot/`, `storage/memory/`) are **only imported in test and example files**:

```
event/event_bdd_test.go:13:  "schema/v3"
event/event_bdd_test.go:14:  "storage/memory/v3"
event/example_test.go:9:     "schema/v3"
event/example_test.go:10:    "storage/memory/v3"
event/stream_test.go:13:     "storage/memory/v3"
event/errors_taxonomy_test.go:12: "snapshot/v3"
```

Go's `go mod tidy` cannot separate test-only deps from production deps in a single-module package, so they leak into the production `require` block. This makes `event/` appear to be Layer 4+ when its real production deps are only `codec/` and `id/`.

#### `event/` conflates three architectural concerns

This is the core problem. `event/` contains:

| Concern | What's in it | Who needs it |
|---|---|---|
| **Domain model** | `ImmutableEvent`, `NewEvent`, `Type`, `Metadata`, `Causality`, `SchemaVersion`, tombstone detection | Pure domain code (decider, deriver, command, query) |
| **Error taxonomy** | `NewConflict`, `NewRejection`, `Wrapf`, `Classify`, `Transient`, `Conflict` — line-for-line re-export of `go-error-family` | **Every module that classifies errors** (30+ modules) |
| **Infrastructure contracts** | `EventSink`, `EventSource`, `Store`, `Journal`, `SeekableJournal`, `Bus`, `Publisher`, `PublishMiddleware` | Storage and transport implementations |

Every module that needs error classification pulls in the entire event domain model + event store/bus contracts. This is why `idempotency/` depends on `event/` — not for events, but for `event.NewConflict`.

### 2.3 The Error Taxonomy Trap

`event/errors.go` is a **1:1 re-export** of `go-error-family`:

```go
// event/errors.go                         // what go-error-family provides
func NewConflict(code, msg string) *Error  { return errorfamily.NewConflict(code, msg) }
func NewRejection(code, msg string) *Error { return errorfamily.NewRejection(code, msg) }
func Wrapf(err error, f Family, ...) *Error { return errorfamily.Wrapf(err, f, ...) }
func Classify(err error) Family            { return errorfamily.Classify(err) }
```

This creates artificial coupling across the entire codebase:

| Pattern | Modules | Count |
|---|---|---|
| **Import `event/` ONLY for error classification** | command, query, decider, middleware, signing, schema, snapshot, listing, deriver, scenario, projectionhost, graph, idempotency, watermill, transport/http, transport/grpc, storage/*, stack/* | ~30 |
| **Import `go-error-family` directly** (bypassing event/) | id, kv, codec, encryption, dispatcher, catalog, cmd/api-stability, cmd/cqrs-gen | 8 |
| **Mixed** (both patterns in same module) | storage (1 file direct, 25 files via event/), query (1 file direct, 5 files via event/) | 2 |

The inconsistency tells the story: leaf modules that don't already depend on `event/` import `go-error-family` directly. Everything else routes through the re-export and picks up the full event domain model as baggage.

### 2.4 The Command/Query-Depend-On-Event Problem

`command/` lists `event/` as a direct dependency. What it actually uses:

```go
// Domain types (legitimate coupling):
command/aggregate_ref.go:  type AggregateRef = event.AggregateRef
command/aggregate_ref.go:  event.NewAggregateRef(...)
command/aggregate_ref.go:  event.ParseAggregateType(...)

// Error classification (artificial coupling):
command/errors.go:          event.NewRejection(...)
command/errors.go:          event.NewConflict(...)
command/errors.go:          event.NewInfrastructure(...)
command/dispatcher.go:      event.WrapRejection(...)
command/dispatcher.go:      event.Wrap(...)
command/dispatcher.go:      event.Classify(...)
command/command.go:         event.WrapRejection(...)
```

`AggregateRef` is a shared domain type — that coupling is legitimate. But 15+ error-classification calls are artificial. `command/` pulls in the entire event store/bus interface surface to get `event.NewRejection`.

`query/` has the same pattern — and has already started bypassing it (`query/store.go` imports `go-error-family` directly while the rest of the module uses `event.*`).

### 2.5 The Fake Layer Summary

| Module | Documented Layer | Actual Layer (by deps) | Why |
|---|---|---|---|
| `kv/` | 0 | 1 | Depends on `codec/` + `otter` |
| `event/` | 1 | 2 (production) / 4 (declared) | Test deps leak into go.mod as direct |
| `command/` | 1 | 2 | Depends on `event/` (Layer 2) + `storage/memory/` (Layer 4) |
| `query/` | 1 | 2 | Depends on `event/` + `storage/memory/` |
| `idempotency/` | 2 | 2 | Depends on `event/`, `kv/`, `command/` |
| `scenario/` | 3 | 3 | OK |
| `decider/` | 3 | 4 | Depends on `event/`, `snapshot/`, `otel/`, `storage/memory/` |

---

## 3. The Core Problem: One Package, Three Concerns

```
                    ┌──────────────────────────────────────┐
                    │              event/                   │
                    │                                       │
                    │  ┌─────────────────────────────────┐  │
                    │  │  DOMAIN MODEL                   │  │
                    │  │  ImmutableEvent, NewEvent       │  │
                    │  │  Type, Metadata, Causality      │  │
                    │  │  SchemaVersion, Tombstone       │  │
                    │  │  AggregateRef, AggregateID      │  │
                    │  └─────────────────────────────────┘  │
                    │  ┌─────────────────────────────────┐  │
                    │  │  ERROR TAXONOMY                 │  │
                    │  │  NewConflict, NewRejection      │  │
                    │  │  Wrapf, Classify, Transient     │  │
                    │  │  (1:1 re-export of              │  │
                    │  │   go-error-family)              │  │
                    │  └─────────────────────────────────┘  │
                    │  ┌─────────────────────────────────┐  │
                    │  │  INFRASTRUCTURE CONTRACTS       │  │
                    │  │  EventSink, EventSource, Store  │  │
                    │  │  Journal, SeekableJournal       │  │
                    │  │  Bus, Publisher, PublishMW      │  │
                    │  └─────────────────────────────────┘  │
                    └──────────────────────────────────────┘
                                       │
                    Every module that imports event/ gets
                    ALL THREE concerns whether it needs
                    them or not.
```

**Who needs what:**

| Consumer | Needs domain model? | Needs error taxonomy? | Needs infrastructure contracts? |
|---|---|---|---|
| `decider/` | Yes (Event type) | Yes (error wrapping) | Yes (Store, to load/save) |
| `command/` | Partially (AggregateRef) | Yes (error wrapping) | No |
| `query/` | No | Yes (error wrapping) | No |
| `idempotency/` | No | Yes (error wrapping) | No |
| `middleware/` | Partially (Event type for EventTracing) | Yes (error wrapping) | No |
| `signing/` | Yes (Event payload, ImmutableEvent) | Yes (error wrapping) | No |
| `storage/` | Yes (ImmutableEvent) | Yes (error wrapping) | Yes (implements Store) |
| `kv/` | No | Yes (but imports go-error-family directly) | No |

Only `storage/` needs all three. Most modules need one or two. But the package boundary forces all-or-nothing.

---

## 4. First-Principles Future Architecture

### 4.1 The Principle: Separate What Changes for Different Reasons

The three concerns in `event/` change for different reasons:

- **Domain model** changes when the business changes (new event fields, new metadata)
- **Error taxonomy** changes when we add error families or classification rules
- **Infrastructure contracts** change when we add new storage/transport capabilities

Conflating them means every change to any one concern potentially affects every consumer. Separating them lets each consumer depend on exactly what it needs.

### 4.2 The Proposed Architecture: Four Tiers (Not Seven Layers)

```
┌─────────────────────────────────────────────────────────────────────┐
│                        COMPOSITION TIER                              │
│   stack/, stack/sqlite/, stack/postgres/, catalog/,                  │
│   scenario/, testutil/, cmd/, example/                               │
│   "Wire everything together for the consumer"                        │
├─────────────────────────────────────────────────────────────────────┤
│                      OPERATIONS TIER                                 │
│   middleware/ (+ idempotency/), otel/, prometheus/,                  │
│   signing/, encryption/, projectionhost/, listing/, scheduling/      │
│   "Cross-cutting concerns applied across the system"                 │
├─────────────────────────────────────────────────────────────────────┤
│                    INFRASTRUCTURE TIER                               │
│   storage/ (+ sql/, pebble/, turso/, memory/, view/, relational/)    │
│   transport/http/, transport/grpc/, watermill/                       │
│   "Implement the contracts for specific backends"                    │
├─────────────────────────────────────────────────────────────────────┤
│                        DOMAIN TIER                                   │
│   id/, event/, command/, query/, decider/, deriver/,                 │
│   projection/, schema/, dispatcher/, codec/                          │
│   "Pure domain model + contracts. No backend knowledge."             │
└─────────────────────────────────────────────────────────────────────┘
```

**Key difference from current:** Tiers represent **architectural boundaries**, not dependency order. Within each tier, modules can depend on each other freely. Cross-tier dependencies flow upward only.

### 4.3 The Domain Tier Split

The domain tier itself has internal structure:

```
DOMAIN TIER
├── PRIMITIVES (zero internal deps)
│   ├── id/           — branded IDs (ULID-based, marker types)
│   ├── codec/        — encoding contract (Codec interface + JSON/CBOR impls)
│   └── errors/       — error taxonomy
│                        (either a thin module re-exporting go-error-family,
│                         or just: import go-error-family directly everywhere)
│
├── MESSAGE TYPES (depend on primitives)
│   ├── event/        — ImmutableEvent, Type, Metadata, Causality, NewEvent
│   │                   AggregateRef, tombstone detection
│   │                   NO Store/Sink/Source/Bus — those move to contracts
│   ├── command/      — Command, BasicCommand, Type, AggregateRef
│   │                   NO Dispatcher/Store/Bus
│   └── query/        — Query, Type
│                       NO Dispatcher/Store
│
├── CONTRACTS (depend on message types)
│   ├── store/        — EventSink, EventSource, Store, Journal, SeekableJournal
│   │                   CommandSink, CommandSource, CommandJournal
│   │                   QuerySink, QuerySource, QueryJournal
│   │                   SnapshotSink, SnapshotSource, SnapshotStore
│   │                   CheckpointStore
│   │                   (currently embedded in event/, command/, query/)
│   ├── bus/          — EventBus, Publisher, PublishMiddleware
│   │                   CommandBus, Publisher, Subscriber
│   │                   (currently embedded in event/, command/)
│   └── kv/           — raw KV interface (Store, Reader, Writer, Iterator, Batch)
│                        NO TypedStore, NO Cache (those need codec, move up)
│
└── DOMAIN LOGIC (depend on message types + contracts)
    ├── decider/      — Decider[State], Repository[State] (needs Store contract)
    ├── deriver/      — event→command derivation rules
    ├── projection/   — Projection interface (consumer-side contract)
    ├── dispatcher/   — generic dispatch mechanism
    └── schema/       — upcasting/versioning (Upcaster, VersionedStore)
```

### 4.4 What Changes: The `event/` Decomposition

**The biggest single change.** Split `event/` into:

| New package | Contains | Deps |
|---|---|---|
| `event/` (slimmed) | `ImmutableEvent`, `NewEvent`, `Type`, `Metadata`, `Causality`, `SchemaVersion`, `AggregateRef`, tombstone, `CloneEvent`, `PayloadReadOnly` | `id/`, `codec/`, `go-error-family` |
| `store/` (new) | `EventSink`, `EventSource`, `Store`, `Journal`, `SeekableJournal`, `BackwardsSource`, `EventStore` (facade), `MemoryStore` (in-memory impl) | `event/` |
| `bus/` (new) | `Bus`, `Publisher`, `PublishMiddleware`, `EventBus`, `MemoryBus` | `event/` |

Wait — `MemoryStore` and `MemoryBus` are currently in `storage/memory/`. Let me reconsider.

Actually, the cleanest split is:

| Package | Contains |
|---|---|
| `event/` | Domain model only: `ImmutableEvent`, `NewEvent`, `Type`, `Metadata`, etc. + the Sink/Source/Store/Bus **interfaces** (they're just interfaces, they belong with the domain types) |
| `errors/` | Error taxonomy re-export (or use `go-error-family` directly everywhere) |

The interfaces (`EventSink`, `EventSource`, etc.) are **domain contracts** — they define what an event store IS, not how it's implemented. They belong in `event/`. The implementations belong in infrastructure.

The real problem isn't that interfaces and types are in the same package — it's that **error classification** is trapped there too.

### 4.5 The Minimal Fix: Extract the Error Taxonomy

If we do nothing else, extracting the error taxonomy from `event/` into its own module (or importing `go-error-family` directly) would eliminate the artificial coupling:

```
BEFORE: 30+ modules → event/ → go-error-family
AFTER:  30+ modules → go-error-family (direct)
              event/ → go-error-family (for its own internal use)
```

**Two options:**

| Option | Description | Cost | Benefit |
|---|---|---|---|
| **A: Import go-error-family directly** | Every module that currently does `event.NewConflict(...)` changes to `errorfamily.NewConflict(...)`. `event/errors.go` is deleted. | ~50 files changed (mechanical find-replace) | Zero new modules. Coupling eliminated. Consistent with the 8 modules that already do this. |
| **B: Create `errors/` module** | New thin module that re-exports go-error-family with CQRS-specific defaults. Every module imports `errors/v3` instead of `event/`. | New module + ~50 files changed | Single import point. Can add CQRS-specific error helpers. But... it's still a re-export. |

**Recommendation: Option A.** A re-export module is just indirection. `go-error-family` IS the error taxonomy — it already has the right API. The 8 modules that import it directly prove the pattern works. Adding a CQRS-specific wrapper adds a layer without adding value.

### 4.6 The `kv/` Decomposition

Split `kv/` into raw and typed layers:

| Package | Contains | Deps |
|---|---|---|
| `kv/` (raw) | `Store`, `Reader`, `Writer`, `Iterator`, `Batch`, `ConditionalWriter`, `MemStore`, `ErrNotFound`, `ErrClosed` | `go-error-family` only |
| `kv/typed/` (new sub-package) | `TypedStore[T,K]`, `Cache[T,K]`, `ViewStore[V,K]`, `ViewQuery`, `ViewQuerier`, `Condition`, `Operator` | `kv/`, `codec/`, `otter` |

This makes `kv/` a true Layer 0 leaf module. The typed layer is explicitly higher-level.

### 4.7 The Full Target Dependency Graph

```
DOMAIN TIER
═══════════
  Primitives:
    id/           → (stdlib + go-branded-id + go-error-family)
    codec/        → (stdlib + go-error-family + cbor)
    kv/           → go-error-family               [TRUE Layer 0]

  Message Types:
    event/        → id/, codec/, go-error-family   [domain model + interfaces]
    command/      → id/, event/, go-error-family   [AggregateRef is shared]
    query/        → dispatcher/, go-error-family    [no event/ dep!]

  Domain Logic:
    dispatcher/   → go-error-family
    decider/      → event/, snapshot/, go-error-family
    deriver/      → command/, event/, id/
    projection/   → event/, id/
    schema/       → event/, id/, codec/
    snapshot/     → event/, codec/, id/
    kv/typed/     → kv/, codec/, otter

INFRASTRUCTURE TIER
══════════════════
    storage/      → event/, command/, query/, kv/, codec/, otel/, ...
    storage/pebble/ → storage/, kv/, ...
    storage/turso/  → storage/, kv/, ...
    storage/memory/ → event/, command/, query/, snapshot/, ...
    transport/http/ → event/, dedup/, ...
    transport/grpc/ → command/, event/, query/, ...
    watermill/    → event/, command/, dedup/, ...

OPERATIONS TIER
═══════════════
    middleware/   → command/, event/, query/, otel/, idempotency/
    middleware/idempotency/ → go-error-family, kv/  [storage primitive]
    otel/         → (stdlib + OTel SDK)
    prometheus/   → otel/
    signing/      → event/, id/
    encryption/   → event/, codec/, id/
    projectionhost/ → event/, projection/, storage/, ...
    listing/      → event/, id/
    scheduling/   → testutil/

COMPOSITION TIER
════════════════
    stack/        → everything (this is the point)
    catalog/      → event/, id/
    scenario/     → event/, id/, projection/
    testutil/     → command/, event/, id/
    cmd/          → various
    example/      → various
```

### 4.8 Key Differences From Current

| Change | Impact | Effort |
|---|---|---|
| **Error taxonomy: `event.*` → `go-error-family` direct** | 30+ modules shed their artificial `event/` dep | ~50 files mechanical find-replace |
| **`kv/` split into raw + typed** | `kv/` becomes true Layer 0; typed concerns explicit | Move `typed_store.go`, `cache.go`, `view_store.go` to sub-package |
| **`command/` stops depending on `event/` for errors** | Still depends on `event/` for `AggregateRef` (legitimate) | Mechanical: `event.NewRejection` → `errorfamily.NewRejection` |
| **`query/` stops depending on `event/`** entirely** | Query has no domain coupling to events — it was ALL error classification | Mechanical: already started in `query/store.go` |
| **`idempotency/` merges into `middleware/idempotency/`** | See dedicated plan | Separate execution plan |

**`query/` is the most interesting case.** If we extract the error taxonomy, `query/` has **zero dependency on `event/`**. This means:
- `query/` and `event/` are independent domain concerns
- A consumer can use query dispatch without pulling in the event sourcing machinery
- This validates the CQRS separation: commands and queries don't need events

---

## 5. Migration Path (Prioritized by Impact/Effort)

### Phase 1: Extract Error Taxonomy (Pareto: 80% benefit, 20% effort)

**The single highest-impact change.** Eliminates artificial coupling across 30+ modules.

1. Audit every `event.New*`, `event.Wrap*`, `event.Classify` call across the repo
2. Replace with `go-error-family` direct imports
3. Delete `event/errors.go`
4. Remove `go-error-family` from modules that only had it via `event/` indirect

**Effort:** ~50 files, mechanical find-replace. No API change visible to consumers (they call the same functions, just from a different import path).

**Risk:** Low. The API is identical — `errorfamily.NewConflict` has the same signature as `event.NewConflict`.

### Phase 2: Fix `kv/` Layering

1. Move `typed_store.go`, `cache.go`, `view_store.go`, `typed_options.go` to `kv/typed/` sub-package
2. `kv/` keeps only raw byte-level types: `Store`, `Reader`, `Writer`, `Iterator`, `Batch`, `MemStore`, errors
3. Update consumers: `kv.TypedStore` → `kv/typed.TypedStore`
4. `kv/` go.mod drops `codec/` and `otter` deps

**Effort:** ~15 files moved + import updates.

### Phase 3: Merge `idempotency/` (separate plan)

See [`2026-07-06_IDEMPOTENCY_MERGE_PLAN.md`](2026-07-06_IDEMPOTENCY_MERGE_PLAN.md).

### Phase 4: Clean Up `event/` Test Deps

1. Move test-only deps (`schema/`, `snapshot/`, `storage/memory/`) to a separate test module, OR accept the leak and document it
2. Alternative: move `event/` tests that need these deps into `integration/`

**Effort:** Medium — Go doesn't natively support test-only module deps. May need a separate `event/eventtest/` module (which already exists for `event/v3/eventtest`).

### Phase 5: Document the Tier Model

1. Update `AGENTS.md` Module Graph to use the four-tier model
2. Update `flake.nix` testModules to reflect corrected layers
3. Update `cmd/api-stability/main.go` to scan all packages including currently-missing ones

---

## 6. What NOT to Change

| Thing | Why it stays |
|---|---|
| `event/` contains both domain types AND store/bus interfaces | Interfaces are domain contracts, not implementations. Moving them to a separate `store/` package adds a module boundary without adding clarity — every event consumer needs both. The problem was never "Store is in event/" — it was "error taxonomy is in event/". |
| `command/` depends on `event/` for `AggregateRef` | This is legitimate domain coupling. Commands target aggregates. `AggregateRef` is the shared vocabulary. |
| `decider/` depends on `event/` + `snapshot/` | Legitimate — the decider loads events from a store and snapshots state. This is the core ES pattern. |
| `dedup/` as a separate Layer 0 leaf | It IS a true leaf (zero deps). Different concern from idempotency (ring buffer vs TTL store). Correctly separate. |
| Multi-module workspace (`go.work`) | Each module having its own `go.mod` is a deliberate design choice for consumer import isolation. The workspace ties them together for development. This is correct. |

---

## 7. The "Should We?" Question

**Is this refactoring worth it?**

| Argument FOR | Argument AGAINST |
|---|---|
| 30+ modules have artificial coupling to `event/` | It works today. The coupling doesn't cause runtime bugs. |
| Error taxonomy trap makes every new module carry event baggage | Consumers don't see the coupling — they import what they need. |
| `kv/` Layer 0 claim is false and misleading | Just fix the AGENTS.md text, don't restructure. |
| Phase 1 (error taxonomy) is ~50 file mechanical change for 80% of the benefit | 50 files is a lot of churn for "just change the import path." |
| `query/` could become truly independent of `event/` | `query/` already works — who cares if it imports `event/`? |
| Sets up clean v4 boundaries | v4 is not planned. |

**My recommendation:** Phase 1 (extract error taxonomy) is worth doing. The coupling is real, the fix is mechanical, and it unblocks independent use of query/command without event baggage. The rest is nice-to-have documentation cleanup.

Phase 2 (kv/ split) is worth doing because the false Layer 0 claim will mislead contributors and consumers. It's a small change with high clarity payoff.

Everything else can wait for v4.

---

## Appendix A: Current Module Dependency Matrix (Verified)

```
MODULE                    DIRECT INTERNAL DEPS
────────────────────────  ─────────────────────────────────────────────────────────
id/                       (none — true leaf)
codec/                    (none — true leaf)
dispatcher/               (none — true leaf)
dedup/                    (none — true leaf)
otel/                     (none — true leaf)
prometheus/               (none — true leaf)
catalog/                  (none — true leaf)

kv/                       codec/
event/                    codec/, id/, eventtest, schema/, snapshot/, storage/memory/
event/v3/eventtest/       event/, id/, snapshot/
command/                  dispatcher/, event/, id/, storage/memory/, codec/
query/                    dispatcher/, event/, id/, storage/memory/, codec/

snapshot/                 codec/, event/, eventtest, id/
schema/                   codec/, event/, eventtest, id/, storage/memory/
projection/               event/, id/
idempotency/              command/, event/, id/, kv/
deriver/                  command/, event/, id/
testutil/                 command/, event/, id/
scenario/                 event/, id/, projection/

decider/                  codec/, event/, eventtest, id/, otel/, snapshot/, storage/memory/
signing/                  event/, eventtest, id/
encryption/               codec/, event/, eventtest, id/
graph/                    event/, id/, projection/
listing/                  event/, eventtest, id/, storage/memory/
scheduling/               testutil/

middleware/               command/, event/, eventtest, id/, otel/, query/
storage/                  codec/, command/, event/, eventtest, id/, kv/, listing/, otel/, projection/, query/, scheduling/, snapshot/
storage/pebble/           codec/, command/, event/, eventtest, id/, kv/, otel/, query/, snapshot/
storage/turso/            command/, event/, eventtest, id/, kv/, otel/, query/, snapshot/, storage/
storage/memory/           command/, dispatcher/, event/, eventtest, id/, query/, snapshot/
watermill/                codec/, command/, dedup/, event/, eventtest, id/, otel/, storage/memory/
transport/http/           dedup/, event/, eventtest, id/, otel/, storage/memory/, testutil/
transport/grpc/           codec/, command/, event/, id/, otel/, query/
projectionhost/           dedup/, event/, id/, otel/, projection/, storage/memory/, storage/, testutil/

stack/                    codec/, command/, decider/, event/, eventtest/, id/, kv/, projection/, query/, snapshot/, storage/memory/, storage/, watermill/
stack/sqlite/             codec/, command/, event/, id/, kv/, stack/, storage/, watermill/
stack/postgres/           event/, id/, stack/, storage/, watermill/
stack/pebble/             codec/, event/, id/, kv/, stack/, storage/pebble/, watermill/
stack/memory/             codec/, event/, id/, kv/, stack/, storage/memory/, watermill/
stack/turso/              codec/, command/, event/, id/, kv/, stack/, storage/turso/, storage/, watermill/
stack/bench/              event/, id/, kv/, stack/, storage/memory/

integration/              (everything — by design)
example/taskmanager/      (most of the tree — by design)
example/getting-started/  decider/, event/, id/, kv/, stack/sqlite/, stack/, storage/memory/, watermill/
```

## Appendix B: The Error Taxonomy Audit

Every `.go` file (non-test) that calls `event.*` error functions, grouped by module:

| Module | Files using event.* errors | Should use go-error-family directly? |
|---|---|---|
| `command/` | 8 files | YES — only `AggregateRef` is legitimate event coupling |
| `query/` | 5 files | YES — zero legitimate event coupling (already started) |
| `decider/` | 3 files | YES — error wrapping only |
| `middleware/` | 9 files | YES — error wrapping only |
| `signing/` | 10 files | YES — error wrapping only |
| `schema/` | 5 files | YES — error wrapping only |
| `snapshot/` | 3 files | YES — error wrapping only |
| `listing/` | 3 files | YES — error wrapping only |
| `idempotency/` | 3 files | YES — being merged into middleware/ |
| `storage/` | ~25 files | YES — error wrapping only |
| `watermill/` | ~12 files | YES — error wrapping only |
| `transport/http/` | 2 files | YES — error wrapping only |
| `transport/grpc/` | 7 files | YES — error wrapping only |
| `projectionhost/` | 2 files | YES — error wrapping only |
| `deriver/` | 1 file | YES — error wrapping only |
| `scenario/` | 1 file | YES — error wrapping only |
| `graph/` | 4 files | YES — error wrapping only |
| `stack/*` | ~20 files | YES — error wrapping only |
| `example/*` | 4 files | YES — error wrapping only |
| **Total** | **~129 files** | |
