# Design Note: `systemcatalog` — catalog derivation from `system.DomainConfig`

**Date:** 2026-10-26 (executes F01 of the [SUPERB Pareto plan](2026-10-26_11-21_SUPERB-catalog-truth-derivation-pareto-plan.md))
**Status:** DECIDED (defaults for G1/G2 assumed; both honored — see verdicts below)

## Problem

Fleet consumers hand-copy their message inventory into `catalog.Builder` calls.
The list drifts the moment a command/event/query is added. `system.DomainConfig`
already declares the full event universe (`Events`, `Schema`) and registers
every command/query via closures — but that truth is invisible to `catalog`.

## Constraints discovered (source-verified)

1. `DomainConfig.Commands`/`Queries` are `func(*System)` **closures** — no static
   enumeration. They run inside `system.New` (`constructor.go:334-339`).
2. `system.RegisterCommand[Cmd, State]` **erases** `Cmd` to `command.Command`
   in the dispatcher map (`register.go:102`); only a count survives
   (`cmdHandlerCount`). Same for queries. → runtime inventory needs an additive
   recording surface.
3. `schema.EventSchema` is opaque wire-shape data (`Type()`, `CurrentVersion()`,
   upcast ops) — it carries **no Go payload type** (`schema/declaration.go:19`).
   Event payload schemas are therefore NOT derivable from config; names+versions are.
4. `catalog.MessageConfig` has an unexported `apply` method — external packages
   **cannot** implement it; only `Command[T]`/`Event[T]`/`Query[T]` construct
   messages, and all require a payload type parameter.
5. `catalog.SchemaFromReflect(reflect.Type)` already exists; `FromReflect`
   returns a shared cached schema that callers must `Clone()` before mutating.
6. `systemscenario.Memory()` is the proven minimal ephemeral deployment
   (two memory engines: primary + timers).

## Decision

### D1 — Module topology: new LAYER-7 adapter module `systemcatalog`

`github.com/larsartmann/go-cqrs-lite/systemcatalog/v4` — sibling of `catalog`
(LAYER 7, tooling tier). Imports (budget **6**): `system`, `catalog`, `schema`,
`event`, `command`, `query` — all downward-legal (6/7/2/1/1/1 ≤ 7).

**G2 honored:** catalog stays dep-free of system (no reverse edge); the adapter
owns the bridge. **system also stays catalog-free** — the adapter is the only
new edge, avoiding any system dep-budget pressure (its budget is 21, full).

Alternatives rejected:
- *catalog imports system* — inverts dependency direction (catalog is L7 pure
  data with 1 direct dep; system would drag engines into every catalog user).
- *system imports catalog* — burns system's tight budget for a docs concern.

### D2 — catalog gets an additive schema-parameterized constructor family

```go
func CommandFromSchema(id MessageID, sch *Schema, opts ...MessageOption) MessageConfig
func EventFromSchema(id MessageID, direction Direction, sch *Schema, opts ...MessageOption) MessageConfig
func QueryFromSchema(id MessageID, sch *Schema, opts ...MessageOption) MessageConfig
```

`sch` may be nil (no payload schema — the DomainConfig event case). Name is
humanized from the id (kebab/snake/dot → "Record Invoice State"). This is the
minimum viable external construction surface; catalog stays reflection-optional.

### D3 — system gets additive registration recording

`RegisterCommand`/`RegisterQuery` additionally record
`{Type, PayloadType reflect.Type}`; new accessors return sorted snapshots:

```go
type CommandRegistration struct { Type command.Type; PayloadType reflect.Type }
func (s *System) CommandRegistrations() []CommandRegistration
type QueryRegistration struct { Type query.Type; PayloadType reflect.Type }
func (s *System) QueryRegistrations() []QueryRegistration
```

This is runtime truth extraction — reused later by the auditlog bridge (T07)
and live inventory gates. Mirrors the `cmdHandlerCount` introspection precedent.

### D4 — systemcatalog API (shape)

```go
// One-call: boots an ephemeral memory system (systemscenario.Memory recipe),
// runs the closures via system.New, derives, closes.
func FromDomainConfig(ctx context.Context, domain system.DomainConfig, opts ...Option) (*catalog.Catalog, error)

// For already-booted systems (tests, running binaries): reads registrations.
func FromSystem(sys *system.System, domain system.DomainConfig, opts ...Option) (*catalog.Catalog, error)

// Curated-merge path (T04): derive messages onto an existing builder;
// consumer-added Flows/Owners/DataStores survive by construction.
func DeriveInto(b *catalog.Builder, sys *system.System, domain system.DomainConfig, opts ...Option) error
```

Options: `WithTitle`, `WithVersion`, `WithServiceID`, `WithServiceSummary`,
`WithServiceOwners`, `WithCurator(func(*catalog.Builder))`.

Derivation rules:
- **Commands** → `CommandFromSchema(type, SchemaFromReflect(payloadType).Clone())`, Receives.
- **Queries** → same, Receives.
- **Events** = union of `domain.Events` and `domain.Schema` types →
  `EventFromSchema(type, Sends, nil)`; `WithVersion(schema.CurrentVersion())`
  when declared (the T09 lockstep seed: version truth rides on the message).
- Producers/consumers: single service Sends its events, Receives commands and
  queries — `Catalog.DeriveProducersConsumers()` then fills the graph.
- G1 honored: **1 service per deployable** (default), room for a context-split
  option later.

### D5 — What derivation does NOT claim

Event payload field schemas (not declared in config — curate via `WithCurator`
or typed `catalog.Event[T]` additions), per-flow documentation, external
producers. Derivation adds truth; it never culls curated resources.

## Failure modes handled

- Ephemeral boot fails (coeffect gate, bad config) → error returned honestly.
- `FromSystem` on a nil/closed system → error, not panic.
- Duplicate id across kinds → catalog `Build()` validation already rejects.

## Gate checklist (F02)

8-gate ritual per AGENTS "Add a New Module": go.mod, go.work, flake testModules,
api-stability modules slice, module-layers LAYER+DEP_BUDGET, cqrs-lint
module_catalog_test, api-stability exclusion maps (as tooling module), api
golden regen + `TestEvery`.
