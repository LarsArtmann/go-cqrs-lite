# Future: TypeSpec Extension for the Event-Query Model

> **Status:** Future exploration. Not started. The current implementation path is
> pure Go type inference (runtime reflection). This document captures the idea for
> when the meta-engine reaches the point where multi-target generation justifies a
> dedicated schema language.

**Created:** 2026-07-23
**Prerequisite reading:** [event-query-model.md](event-query-model.md)

---

## Why TypeSpec (Not a Custom DSL)

Building a custom DSL (parser, type checker, code generator, LSP, error messages,
formatter) is a 2-3 month project that produces a poor shadow of Go's tooling.
TypeSpec solves this:

- **Industry-standard** — Microsoft's open language for API description, with a
  mature parser, type system, and ecosystem.
- **Multi-target generation** — one `.tsp` file emits OpenAPI, AsyncAPI, JSON
  Schema, protobuf, and **custom emitters** (Go code, SQL DDL, D2 diagrams).
- **IDE support** — VS Code extension exists today, with autocomplete, validation,
  and go-to-definition.
- **Not another language** — TypeSpec is a schema language, not a programming
  language. Fold functions stay in Go. TypeSpec describes the data shapes and
  relationships; Go implements the domain logic.

---

## What the Extension Would Look Like

```tsp
// domain.tsp — the single source of truth for the read side

using MetaEngine;

// ═══ Events (data shapes, not logic) ═══

model UserCreated {
  id: UserID;
  email: string;
  name: string;
  country: string;
  at: utcDateTime;
}

model UserSuspended {
  id: UserID;
  at: utcDateTime;
}

model UserDeleted {
  id: UserID;
  at: utcDateTime;
}

model Friendship {
  from: UserID;
  to: UserID;
  at: utcDateTime;
}

// ═══ Queries (read intent + result shape) ═══

@projection(
  on: [
    UserCreated => insert key=id,
    UserSuspended => update { status: "suspended" },
    UserDeleted => remove
  ],
  volume: 1_000_000
)
model FindUser {
  id: UserID;           // filter: point lookup by ID
}
model FindUserResult {
  id: UserID;
  name: string;
  email: string;
  status: string;
  country: string;
  joinedAt: utcDateTime;
}

@projection(
  on: [
    UserCreated => insert key=id,
    UserSuspended => update { status: "suspended" },
    UserDeleted => remove
  ],
  filter: [status],
  sort: joinedAt desc,
  paginated: true
)
model ListByStatus {
  status: string;       // filter field
}
// Result type: Page<FindUserResult> — emitted automatically

@projection(
  on: [
    UserCreated => count { active: +1 },
    UserSuspended => count { active: -1, suspended: +1 },
    UserDeleted => count { active: -1, deleted: +1 }
  ]
)
model CountByStatus {}
model CountByStatusResult {
  active: int64;
  suspended: int64;
  deleted: int64;
}

@projection(
  on: [
    Friendship => edge(from, to)
  ]
)
model FriendsOf {
  id: UserID;
  depth: int32;
}
model FriendsOfResult {
  ids: UserID[];
}
```

---

## What the TypeSpec Emitter Would Generate

The custom `MetaEngine` TypeSpec emitter would produce:

| Target                    | What it generates                                                            |
| ------------------------- | ---------------------------------------------------------------------------- |
| **Go event structs**      | `UserCreated`, `UserSuspended`, etc. with JSON tags                          |
| **Go query types**        | `FindUser`, `ListByStatus`, `FindUserResult`, etc.                           |
| **Go fold registrations** | `metaengine.OnInsert(...)`, `OnUpdate(...)`, `OnRemove(...)` calls           |
| **Go query declarations** | `metaengine.Query[Q, R](...)` with inferred filters/sort/pagination          |
| **Plan metadata**         | A `plan.json` with filter fields, sort keys, ADT classification, cardinality |
| **OpenAPI spec**          | REST endpoints for each query (automatic API documentation)                  |
| **AsyncAPI spec**         | Event schemas for the command/event bus                                      |
| **D2 topology diagram**   | Visual: event stream → projections → engines → queries                       |
| **SQL DDL** (per engine)  | `CREATE TABLE`, `CREATE INDEX` for SQLite/Postgres projections               |

---

## The Boundary: TypeSpec vs Go

| Lives in `.tsp` (TypeSpec)               | Stays in `.go` (Go)                                 |
| ---------------------------------------- | --------------------------------------------------- |
| Event data shapes                        | Decider logic (command → event)                     |
| Query input shapes                       | Fold function bodies (event → result mapping)       |
| Result shapes                            | Query handler logic (custom reads, computed fields) |
| Projection declarations (on/filter/sort) | Transport layer (HTTP/gRPC/SSE handlers)            |
| Cardinality hints                        | Auth enforcement (upstream concern)                 |
| Relationships (filter/sort/pagination)   | Middleware (retry, idempotency, tracing)            |

**The key boundary:** TypeSpec describes **what** the data looks like and **how it
relates**. Go describes **what to do about it** (fold logic, business rules).

The fold functions remain in Go because they contain domain logic that is
expressible only in a real programming language. TypeSpec declares the projection
shape; the emitter generates the wiring; Go implements the folds.

---

## Why This Is Phase 3 (Not Phase 1)

1. **The optimizer doesn't need it.** The cost-based planner works identically
   whether input comes from Go reflection or a TypeSpec emitter. The optimizer is
   the novel contribution — build it first.

2. **Multi-target generation must earn its complexity.** A TypeSpec extension is
   justified only when the meta-engine needs to emit OpenAPI + AsyncAPI + D2 + SQL
   DDL + Go code from one source. That's 5 targets — enough to justify a schema
   language. One target (Go code) does not justify it.

3. **The Go reflection API is proven first.** If reflection handles 90% of
   real-world query patterns, the TypeSpec extension only adds ergonomics and
   multi-target generation — nice to have, not essential.

4. **TypeSpec adoption requires a stable model.** The event-query model is still
   evolving. Locking it into a schema language prematurely freezes the design.
   Let the Go API stabilize first, then codify it in TypeSpec.

---

## What Other Projects Do

| Project       | Schema language        | When they adopted it                       |
| ------------- | ---------------------- | ------------------------------------------ |
| **Prisma**    | `.prisma` file         | From day 1 — owns the entire data layer    |
| **protobuf**  | `.proto` file          | From day 1 — cross-language from the start |
| **GraphQL**   | `.graphql` schema      | From day 1 — query language is the product |
| **Ent (Go)**  | Go code + codegen      | Started as Go-only, added codegen later    |
| **sqlc (Go)** | SQL queries + Go types | Go-native, no external language            |
| **TypeSpec**  | `.tsp` files           | Multi-target API description from day 1    |

The Go-ecosystem pattern (Ent, sqlc, gRPC-go) is: **start Go-native, add codegen
later if needed.** The meta-engine should follow the same trajectory.

---

## Trigger Criteria (When to Start Phase 3)

Start the TypeSpec extension when **all** of these are true:

- [ ] The Go reflection API handles all 7 ADTs (Map, Set, Counter, Graph, Log, SortedMap, Multimap)
- [ ] The cost-based optimizer is working and validated against benchmarks
- [ ] Consumers are requesting auto-generated API documentation (OpenAPI/AsyncAPI)
- [ ] Consumers are requesting auto-generated SQL DDL
- [ ] The event-query model API has been stable for 1+ release cycle
- [ ] At least 3 of the 5 multi-target outputs are actively requested

If fewer than 3 targets are needed, the codegen validator (`metaengine-check`)
over Go AST is the better investment.
