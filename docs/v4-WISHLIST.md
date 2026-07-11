# v4 Wishlist — Breaking Changes Batched for go-cqrs-lite/v4

> **Status:** PREP COMPLETE. ADR-0044 blind store envelopes shipped (v3.8).
> Deprecated aliases marked, internal code cleaned, migration guide written.
> Ready to cut when trigger criteria met.
> Trigger check: 2 items concrete (codec flip + alias removal), need 1 more.
> **Current major:** v3.8.0 (49 modules, 27 consumer projects)
> **Created:** 2026-06-29

---

## When to Cut v4

A major version is justified when:

1. **3+ breaking changes** accumulate that type aliases CANNOT solve
2. A consumer explicitly requests an API cleanup
3. The alias/shim layer exceeds maintainability (500+ lines of pure aliases)
4. A dependency tree requires incompatible module paths

None of these are met today. This document exists so we **batch** breaks instead
of spreading them across multiple majors.

---

## Candidate Breaking Changes

### 1. Phantom-type CommandID (derive vs fresh)

**Current:** `CommandID = Of[CommandMarker]` — one type. `IsDerivedCommandID()`
runtime predicate distinguishes derived IDs.

**v4:** Introduce `DerivedCommandMarker` so `Of[DerivedCommandMarker]` is a
distinct compile-time type. Derived commands can't be passed where a fresh ID
is expected (and vice versa).

**Blast radius:** Every consumer using `CommandID` as a parameter type.
**Alias-able?** No — phantom types can't be aliased backward.
**Priority:** Medium (no consumer has hit a bug from the runtime predicate).

### 2. event/ god-package split

**Current:** `event/v4` has 37 files, ~5000 LOC. `Event = *ImmutableEvent` is
the most imported type across 27 consumer projects.

**v4:** Split into `event/core`, `event/metadata`, `event/builder`, etc.
**Blast radius:** 27 projects import `event/v4`. Catastrophic migration.
**Alias-able?** Partially — type aliases help, but interface changes don't alias.
**Priority:** LOW. Cohesion is real; 350-line file limit suffices.
**Verdict:** DO NOT SPLIT even in v4 unless a concrete pain point emerges.

### 3. Typed metadata values

**Current:** `command.Metadata.Custom = map[MetadataKey]string` — all values are
strings.

**v4:** Generic typed metadata or a value union type.
**Blast radius:** Consumers reading `md.Custom[key]` would need to change.
**Alias-able?** No — map value type change is breaking.
**Priority:** Low. String values are simple and work.

### 4. Remove SQLBackend facade return-error variants

**Current:** `NewSQLiteBackend(db)` returns `(*SQLBackend, error)`, but some
stores return lazy errors. Consumers must handle both.

**v4:** Unify to always-lazy or always-eager.
**Blast radius:** 15 direct consumers of `storage`.
**Alias-able?** No.
**Priority:** Low. Current API works; just slightly inconsistent.

### 5. Consolidate Event interface removal

**Current:** `Event = *ImmutableEvent` (type alias, not interface). Some old
docs still reference `Event` as an interface.

**v4:** Clean up all interface references, ensure `*ImmutableEvent` is the
canonical name everywhere.
**Blast radius:** Documentation-only; code already works.
**Alias-able?** Already aliased.
**Priority:** None (already done in v3).

### 6. Unified constructor naming

**Current:** Mix of `NewX`, `NewSQLiteX`, `NewSQLXWithDialect` across stores.

**v4:** Consistent functional options: `New(db, WithDialect(d))`.
**Blast radius:** Every constructor call across all consumers.
**Alias-able?** Partially — old constructors can be kept as thin wrappers.
**Priority:** Medium. Would significantly clean up the API surface.

### 7. Remove genproto replace directive

**Current:** `go.work` has a replace directive for `google.golang.org/genproto`
to work around cockroachdb/errors#79.

**v4:** Remove when cockroachdb/errors drops the monolithic genproto dep.
**Blast radius:** None if upstream fixes first.
**Alias-able?** N/A.
**Priority:** Blocked on upstream.

### 8. CBOR as universal default

**Current:** `event.New()` defaults to `JSONCodec`. All blind stores
(`kv.TypedStore`, `snapshot.TypedStore`, `command.TypedStore`,
`query.TypedStore`) default to `JSONCodec`. The stack's `DefaultCodec()`
already returns `CBORCodec` (v3.5.0). The `schema.Validator` auto-detects
encoding (v3.5.0).

**v4:** Flip all remaining defaults to `CBORCodec`:

- `event.New()` → stamp `"cbor"` instead of `"json"`
- `kv.NewTypedStore` → `CBORCodec` default
- `snapshot.TypedStore` → `CBORCodec` default
- `command.TypedStore` → `CBORCodec` default
- `query.TypedStore` → `CBORCodec` default

**Blast radius:** All consumers with existing JSON-encoded data in blind stores.
Requires data migration (clear and rebuild projections; re-snapshot aggregates).
Events are self-describing (`evt.Encoding()` is stamped), so mixed JSON/CBOR
event streams work after the change — only blind stores need migration.
**Alias-able?** No — default codec change is behavioral, not signature.
**Priority:** Medium. v3.5.0 made the stack + validator CBOR-ready; v4 completes it.

---

## Tracked But NOT Breaking (v3.x)

These are tracked here for context but can ship in v3.x:

- **storage/ split** — sub-packages with type aliases (SHIPPING NOW, v3.5.0)
- **storage/ module split** — if storage/ sub-packages become separate go.mod
  files (only if consumers want to depend on just one store type)
- **Neo4j driver** — new module, not breaking
- **Durability profiles** — new types, not breaking

---

## Decision Log

| Date       | Decision                              | Rationale                                                         |
| ---------- | ------------------------------------- | ----------------------------------------------------------------- |
| 2026-06-29 | Do NOT cut v4 yet                     | 27 consumers, aliases solve everything, no concrete break request |
| 2026-06-29 | Ship storage/ split as v3.5.0 aliases | Zero consumer migration                                           |
| 2026-06-29 | Do NOT split event/ even in v4        | 27 importers, cohesion is real                                    |
| 2026-07-10 | ADR-0044 envelopes shipped (v3.8)     | Blind stores now self-describing; codec flip unblocked            |
| 2026-07-10 | Deprecated aliases cleaned internally | ~200 usages across 42 files updated; SA1019 lint clean            |
| 2026-07-10 | json Deterministic + CaseInsensitive  | All Marshal/Unmarshal calls audited (ADR-0047, ADR-0048)          |
