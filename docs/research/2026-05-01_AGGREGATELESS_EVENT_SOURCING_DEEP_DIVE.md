# Deep Dive: Aggregateless Event Sourcing

> **Status:** RESOLVED — Aggregate identity retained; fold/decider ideas absorbed into ADR-0001 (decider/)

**Author**: Based on the work of [Rico Fritzsche](https://ricofritzsche.me/)
**Reference implementations**: [eventstore-typescript](https://github.com/ricofritzsche/eventstore-typescript), [fcis-event-sourcing-rust](https://github.com/ricofritzsche/fcis-event-sourcing-rust)

---

## Table of Contents

1. [What It Is](#1-what-it-is)
2. [The Problem It Solves](#2-the-problem-it-solves)
3. [Architecture](#3-architecture)
4. [Event Store Design](#4-event-store-design)
5. [Event Filters & Context Queries](#5-event-filters--context-queries)
6. [Consistency via CTE](#6-consistency-via-cte)
7. [Feature Slices](#7-feature-slices)
8. [State Folding](#8-state-folding)
9. [The Decide Function](#9-the-decide-function)
10. [The Imperative Shell](#10-the-imperative-shell)
11. [Full Code Example (TypeScript)](#11-full-code-example-typescript)
12. [Full Code Example (Rust)](#12-full-code-example-rust)
13. [Comparison with Traditional ES](#13-comparison-with-traditional-es)
14. [Strengths](#14-strengths)
15. [Weaknesses & Open Questions](#15-weaknesses--open-questions)
16. [When to Consider It](#16-when-to-consider-it)
17. [Implications for go-cqrs-lite](#17-implications-for-go-cqrs-lite)

---

## 1. What It Is

Aggregateless event sourcing is an architectural approach that **eliminates DDD aggregates** from event-sourced systems. Instead of grouping events under aggregate streams, events exist as **standalone, self-contained facts** in a single append-only table. Consistency boundaries are defined dynamically through **SQL queries** rather than statically through aggregate identity.

**Core premise**: "You don't model objects — you record facts."

The system is built on three pillars:

| Pillar                    | Description                                                  |
| ------------------------- | ------------------------------------------------------------ |
| **Pure Events**           | Self-contained facts with no aggregate binding               |
| **Context Queries**       | SQL queries that define consistency boundaries per operation |
| **CTE Atomic Operations** | PostgreSQL CTEs for optimistic check-and-insert              |

---

## 2. The Problem It Solves

### The Aggregate Boundary Problem

Traditional event sourcing forces you to answer: "What is the aggregate?" This question is notoriously difficult:

- **Too small** → Cross-aggregate operations require sagas, process managers, domain events, eventual consistency
- **Too large** → Performance degradation, concurrency conflicts, single point of failure
- **Wrong boundary** → Expensive refactoring when business processes evolve
- **Multiple valid boundaries** → Legitimate disagreement among team members

### The Domain Service Problem

Cross-aggregate coordination requires:

- Domain services (anemic, procedural code outside aggregates)
- Saga orchestrators (complex state machines)
- Process managers (additional event handlers)
- Distributed transactions (or careful compensation)

### The Aggregateless Answer

What if you just... don't have aggregates?

- No aggregate boundaries to get wrong
- No domain services for cross-aggregate coordination
- No sagas for operations that span what would be multiple aggregates
- No process managers for complex workflows

Instead: every operation defines its own **context** — the set of relevant events — and makes a **pure decision** based on those facts.

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     FEATURE SLICES (N parallel)                      │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Register      │  │ Deposit      │  │ Transfer     │              │
│  │ Account       │  │ Money        │  │ Money        │              │
│  │               │  │              │  │              │              │
│  │ Filter:       │  │ Filter:      │  │ Filter:      │              │
│  │ AccountOpened │  │ AccountOpened│  │ AccountOpened│              │
│  │               │  │ MoneyDepos.  │  │ MoneyDepos.  │              │
│  │ Fold:         │  │ MoneyWithdr. │  │ MoneyWithdr. │              │
│  │ {exists}      │  │              │  │ MoneyTransf. │              │
│  │               │  │ Fold:        │  │              │              │
│  │ Decide:       │  │ {exists}     │  │ Fold:        │              │
│  │ Opened event  │  │              │  │ {exists,     │              │
│  └──────┬───────┘  │ Decide:      │  │  balance}    │              │
│         │          │ Deposited evt│  │              │              │
│         │          └──────┬───────┘  │ Decide:      │              │
│         │                 │          │ Transfer evts│              │
│         │                 │          └──────┬───────┘              │
└─────────┼─────────────────┼─────────────────┼──────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        EVENT STORE (PostgreSQL)                       │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  events                                                        │ │
│  │  ┌──────┬────────────┬───────────────────┬──────────────────┐  │ │
│  │  │ seq  │ type       │ payload (JSONB)    │ metadata (JSONB) │  │ │
│  │  ├──────┼────────────┼───────────────────┼──────────────────┤  │ │
│  │  │ 1    │ AccountOp. │ {accountId, ...}  │ {version:"1.0"}  │  │ │
│  │  │ 2    │ MoneyDep.  │ {accountId, 100}  │ {version:"1.0"}  │  │ │
│  │  │ 3    │ MoneyWith. │ {accountId, 30}   │ {version:"1.0"}  │  │ │
│  │  └──────┴────────────┴───────────────────┴──────────────────┘  │ │
│  │                                                                │ │
│  │  Indexes: GIN(payload), B-tree(event_type)                     │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Differences from Traditional ES

| Aspect              | Traditional ES                        | Aggregateless ES                  |
| ------------------- | ------------------------------------- | --------------------------------- |
| Storage             | Stream per aggregate                  | Single events table               |
| Consistency         | Aggregate version / expected revision | CTE with context filter           |
| Boundary definition | At design time (aggregate ID)         | At runtime (query filter)         |
| Cross-boundary ops  | Sagas / process managers              | Naturally via wider context query |
| State shape         | One per aggregate                     | One per feature slice             |
| Read model          | Separate projection handlers          | Per-slice fold functions          |

---

## 4. Event Store Design

### Schema

A single table. No streams. No aggregate IDs as first-class columns.

```sql
CREATE TABLE events (
    sequence_number BIGSERIAL PRIMARY KEY,  -- total global ordering
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}'
);

-- Fast lookups by event type
CREATE INDEX ix_events_type ON events(event_type);

-- Fast lookups by payload contents (JSONB containment)
CREATE INDEX ix_events_payload_gin ON events USING GIN (payload);
```

### Why This Works

- **`sequence_number`**: Global, monotonically increasing. Provides total ordering of all facts.
- **`JSONB payload`**: Flexible schema. Events can evolve without migration.
- **`GIN index`**: PostgreSQL's JSONB containment operator `@>` uses this index for fast filtered queries.
- **No `aggregate_id` column**: Identity lives inside the JSONB payload, not as a first-class concept.

### Events as Pure Facts

Events are **not tied to any aggregate**. They are standalone records of things that happened:

```json
{
  "eventType": "DeviceRegistered",
  "payload": {
    "deviceId": "550e8400-e29b-41d4-a716-446655440000",
    "registeredAt": "2025-06-22T10:00:00Z"
  }
}
```

```json
{
  "eventType": "AssetRegistered",
  "payload": {
    "assetId": "660e8400-e29b-41d4-a716-446655440000",
    "assetName": "Server Rack",
    "registeredAt": "2025-06-22T10:30:00Z"
  }
}
```

```json
{
  "eventType": "DeviceBoundToAsset",
  "payload": {
    "deviceId": "550e8400-e29b-41d4-a716-446655440000",
    "assetId": "660e8400-e29b-41d4-a716-446655440000",
    "boundAt": "2025-06-22T11:00:00Z"
  }
}
```

Notice: `DeviceBoundToAsset` references **both** a device and an asset. In traditional ES, which aggregate "owns" this event? With aggregateless, the question doesn't arise — it's just a fact.

---

## 5. Event Filters & Context Queries

### The Filter Concept

An **EventFilter** defines which events are relevant for a specific operation. It's the aggregateless equivalent of "load aggregate by ID."

```typescript
interface EventFilter {
  eventTypes: string[]; // OR condition on event types
  payloadPredicates?: Record<string, unknown>[]; // OR condition on payload fields
}
```

### Examples

**"Does this account exist?" (for deposit)**

```typescript
const filter = EventFilter.createFilter(["AccountOpened"]).withPayloadPredicate(
  "accountId",
  "acc-123",
);
```

**"What's the balance of this account?" (for withdrawal)**

```typescript
const filter = EventFilter.createFilter(
  ["AccountOpened", "MoneyDeposited", "MoneyWithdrawn"],
  [{ accountId: "acc-123" }],
);
```

**"Can I bind this device to this asset?" (cross-entity)**

```typescript
const filter = EventFilter.createFilter(
  ["DeviceRegistered", "AssetRegistered", "DeviceBoundToAsset", "DeviceUnboundFromAsset"],
  [
    { deviceId: "dev-456" }, // events about this device
    { assetId: "asset-789" }, // OR events about this asset
  ],
);
```

### SQL Generated

```sql
SELECT *
FROM events
WHERE event_type = ANY($1)                          -- matches any of the event types
  AND (
    payload @> $2                                    -- OR: payload contains first predicate
    OR payload @> $3                                 -- OR: payload contains second predicate
  )
ORDER BY sequence_number ASC;
```

The `@>` operator is PostgreSQL's JSONB containment check. With a GIN index, this is fast.

### Why This Replaces Aggregates

In traditional ES, you'd load an aggregate stream: `SELECT * FROM events WHERE stream_id = $1`.

In aggregateless ES, you load a **context**: `SELECT * FROM events WHERE [relevant to this decision]`.

The context can span what would be multiple aggregates. The "binding device to asset" operation naturally queries across devices and assets — no domain service or saga needed.

---

## 6. Consistency via CTE

This is the **key innovation** — how aggregateless ES achieves consistency without aggregate versions.

### The Problem

Two concurrent commands might both query the same context, make decisions, and try to append events. How do you prevent lost updates?

### Traditional ES Answer

```
Load stream at version N → Decide → Append with expected version N → Conflict if version != N
```

### Aggregateless ES Answer

```
Load context, get max_seq → Decide → CTE INSERT checking max_seq unchanged → Conflict if context changed
```

### The CTE (Common Table Expression)

```sql
WITH context AS (
  SELECT MAX(sequence_number) AS max_seq
  FROM events
  WHERE event_type = ANY($1)
    AND (payload @> $2 OR payload @> $3)
)
INSERT INTO events (event_type, payload, metadata)
SELECT
  unnest($4::text[]),
  unnest($5::jsonb[]),
  unnest($6::jsonb[])
FROM context
WHERE COALESCE(max_seq, 0) = $7   -- expected max sequence from query time
```

### How It Works Step by Step

```
1. Query context:
   SELECT * FROM events WHERE [filter]
   → Returns events + max_seq = 42

2. Pure decide:
   state = fold(events)
   newEvents = decide(state, command)

3. Atomic append:
   WITH context AS (
     SELECT MAX(sequence_number) AS max_seq FROM events WHERE [same filter]
   )
   INSERT INTO events (...)
   SELECT [newEvents] FROM context WHERE max_seq = 42
   → If max_seq is still 42: INSERT succeeds, returns rowCount > 0
   → If max_seq is 43 (another writer): INSERT produces 0 rows → conflict
```

### Why This Is Elegant

1. **Single round trip** — The CTE does the check and insert in one atomic SQL statement
2. **No database locks** — Uses optimistic concurrency, not pessimistic locking
3. **Context-specific** — The consistency check is scoped to the exact filter, not a global version
4. **Serializable** — PostgreSQL guarantees the CTE sees a consistent snapshot
5. **No stream concept needed** — The filter IS the consistency boundary

### Concurrent Write Example

```
Timeline:
  T1: Command A queries context (device+asset) → max_seq = 10
  T2: Command B queries context (device+asset) → max_seq = 10
  T3: Command A decides → CTE INSERT with expected max_seq = 10 → SUCCESS (now max_seq = 11)
  T4: Command B decides → CTE INSERT with expected max_seq = 10 → FAILS (max_seq is 11)
  T5: Command B retries → queries again → max_seq = 11 → decides → CTE INSERT → SUCCESS
```

---

## 7. Feature Slices

### Concept

Instead of organizing code by aggregate (Account, Device, Asset), organize by **business feature** (OpenAccount, DepositMoney, TransferMoney). Each feature slice is a self-contained unit with:

```
Feature Slice = {
  filter:    EventFilter           // what events are relevant
  fold:      (events) → state      // reconstruct needed state
  decide:    (state, cmd) → events // pure business logic
  execute:   (pool, cmd) → result  // imperative shell orchestrating the above
}
```

### Example: Banking System

```
┌─────────────────────────────────────────────────────────────┐
│                     Banking System                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ OpenAccount   │  │ DepositMoney │  │ TransferMon. │      │
│  │               │  │              │  │              │      │
│  │ filter:       │  │ filter:      │  │ filter:      │      │
│  │  AccountOpen. │  │  AccountOpen.│  │  AccountOpen.│      │
│  │               │  │              │  │  MoneyDepos. │      │
│  │ fold:         │  │ fold:        │  │  MoneyWith.  │      │
│  │  {exists}     │  │  {exists}    │  │  MoneyTrans. │      │
│  │               │  │              │  │              │      │
│  │ decide:       │  │ decide:      │  │ fold:        │      │
│  │  already? err │  │  exists? ok  │  │  {exists,    │      │
│  │  else: opened │  │  else: err   │  │   balance}   │      │
│  └──────────────┘  └──────────────┘  │              │      │
│                                      │ decide:       │      │
│                                      │  sufficient?  │      │
│                                      │  exists?      │      │
│                                      │  else: err    │      │
│                                      └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Key Insight: Different Slices Need Different State

For the same account's event stream:

| Feature        | State Needed                      | Events Folded                                       |
| -------------- | --------------------------------- | --------------------------------------------------- |
| Open Account   | `{exists: bool}`                  | `AccountOpened` only                                |
| Deposit Money  | `{exists: bool}`                  | `AccountOpened` only                                |
| Withdraw Money | `{exists: bool, balance: number}` | `AccountOpened`, `MoneyDeposited`, `MoneyWithdrawn` |
| Transfer Money | `{exists: bool, balance: number}` | All account events + counterpart account events     |

Each slice reconstructs **only the state it needs**. No wasted computation loading irrelevant events.

---

## 8. State Folding

### The Fold Function

State reconstruction is a **pure function** that folds events into a minimal state shape:

```typescript
// Open Account slice — only cares about existence
function foldAccountState(events: AccountOpened[]): AccountState {
  return { exists: events.length > 0 };
}

// Withdraw Money slice — needs balance too
function foldBalanceState(events: AccountEvent[]): BalanceState {
  let exists = false;
  let balance = 0;

  for (const event of events) {
    switch (event.type) {
      case "AccountOpened":
        exists = true;
        break;
      case "MoneyDeposited":
        balance += event.amount;
        break;
      case "MoneyWithdrawn":
        balance -= event.amount;
        break;
    }
  }

  return { exists, balance };
}
```

### Why Folds Beat Aggregate Hydration

| Aspect             | Aggregate Hydration                           | Feature Fold                                  |
| ------------------ | --------------------------------------------- | --------------------------------------------- |
| State shape        | Fixed (one per aggregate)                     | Flexible (one per feature)                    |
| Events loaded      | All events in stream                          | Only relevant event types                     |
| Unused fields      | Hydrated but ignored                          | Not loaded at all                             |
| Multiple consumers | All see same state                            | Each sees tailored state                      |
| Evolution          | Aggregate schema changes affect all consumers | New fold for new feature, old folds untouched |

---

## 9. The Decide Function

### Pure Business Logic

The `decide` function is **pure** — no IO, no side effects, no mocks needed for testing:

```typescript
function decideOpenAccount(
  state: AccountState,
  command: OpenAccountCommand,
): Result<AccountOpened, OpenAccountError> {
  if (state.exists) {
    return {
      success: false,
      error: { type: "AlreadyExists", message: "Account already opened" },
    };
  }

  return {
    success: true,
    event: new AccountOpened(
      command.accountId,
      command.customerName,
      command.accountType || "checking",
      command.initialDeposit || 0,
      command.currency || "USD",
    ),
  };
}
```

### The Result Type

```typescript
type Result<T, E> = { success: true; event: T } | { success: false; error: E };
```

### Testing

```typescript
// Two-line test. No database. No mocks. No setup.
test("opens account when it does not exist", () => {
  const state = { exists: false };
  const result = decideOpenAccount(state, { accountId: "123", customerName: "Alice" });
  expect(result.success).toBe(true);
});

test("rejects opening when account exists", () => {
  const state = { exists: true };
  const result = decideOpenAccount(state, { accountId: "123", customerName: "Alice" });
  expect(result.success).toBe(false);
  expect(result.error.type).toBe("AlreadyExists");
});
```

---

## 10. The Imperative Shell

The shell handles IO — loading events, calling the pure core, persisting results:

```typescript
async function executeOpenAccount(pool: Pool, command: OpenAccountCommand): Promise<void> {
  // 1. Define context filter
  const filter = EventFilter.createFilter(["AccountOpened"]).withPayloadPredicate(
    "accountId",
    command.accountId,
  );

  // 2. Load context
  const { events, maxSequenceNumber } = await store.query(filter);

  // 3. Reconstruct state
  const state = foldAccountState(events);

  // 4. Pure decision
  const result = decideOpenAccount(state, command);
  if (!result.success) {
    throw new Error(result.error.message);
  }

  // 5. Atomic append (CTE)
  await store.append(filter, [result.event], maxSequenceNumber);
  // If context changed between query and append, this throws.
  // Caller retries.
}
```

### The Complete Flow

```
Command
  │
  ▼
┌──────────────────────────┐
│ 1. Build EventFilter     │  ← Define context
└──────────┬───────────────┘
           ▼
┌──────────────────────────┐
│ 2. store.query(filter)   │  ← Load relevant events + max_seq
└──────────┬───────────────┘
           ▼
┌──────────────────────────┐
│ 3. fold(events) → state  │  ← Pure reconstruction
└──────────┬───────────────┘
           ▼
┌──────────────────────────┐
│ 4. decide(state, cmd)    │  ← Pure decision
│    → events or error     │
└──────────┬───────────────┘
           ▼
┌──────────────────────────┐
│ 5. store.append(         │  ← CTE atomic check-and-insert
│    filter, events,       │
│    expectedMaxSeq)       │
└──────────┬───────────────┘
           ▼
     Success or Retry
```

---

## 11. Full Code Example (TypeScript)

### Event Types

```typescript
interface HasEventType {
  eventType(): string;
  eventVersion?(): string;
}

class AccountOpened implements HasEventType {
  constructor(
    public readonly accountId: string,
    public readonly customerName: string,
    public readonly accountType: string,
    public readonly initialDeposit: number,
    public readonly currency: string,
    public readonly openedAt: Date = new Date(),
  ) {}

  eventType() {
    return "AccountOpened";
  }
  eventVersion() {
    return "1.0";
  }
}

class MoneyDeposited implements HasEventType {
  constructor(
    public readonly accountId: string,
    public readonly amount: number,
    public readonly depositedAt: Date = new Date(),
  ) {}

  eventType() {
    return "MoneyDeposited";
  }
  eventVersion() {
    return "1.0";
  }
}

class MoneyWithdrawn implements HasEventType {
  constructor(
    public readonly accountId: string,
    public readonly amount: number,
    public readonly withdrawnAt: Date = new Date(),
  ) {}

  eventType() {
    return "MoneyWithdrawn";
  }
  eventVersion() {
    return "1.0";
  }
}
```

### Store Interface

```typescript
interface QueryResult<T extends HasEventType> {
  events: T[];
  maxSequenceNumber: number;
}

interface IEventStore {
  query<T extends HasEventType>(filter: EventFilter): Promise<QueryResult<T>>;
  append<T extends HasEventType>(
    filter: EventFilter,
    events: T[],
    expectedMaxSequence: number,
  ): Promise<void>;
  close(): Promise<void>;
}
```

### Query Implementation

```typescript
async query<T extends HasEventType>(filter: EventFilter): Promise<QueryResult<T>> {
  const sql = `
    SELECT * FROM events
    WHERE event_type = ANY($1)
      AND payload @> ANY($2::jsonb[])
    ORDER BY sequence_number ASC
  `;
  const result = await this.pool.query(sql, [
    filter.eventTypes,
    filter.payloadPredicates.map(p => JSON.stringify(p))
  ]);
  return {
    events: result.rows.map(row => this.deserializeEvent<T>(row)),
    maxSequenceNumber: result.rows.length > 0
      ? result.rows[result.rows.length - 1].sequence_number
      : 0
  };
}
```

### Append Implementation (The CTE)

```typescript
async append<T extends HasEventType>(
  filter: EventFilter,
  events: T[],
  expectedMaxSequence: number
): Promise<void> {
  const eventTypes = events.map(e => e.eventType());
  const payloads = events.map(e => JSON.stringify(e));
  const metadata = events.map(e => JSON.stringify({ version: e.eventVersion?.() || '1.0' }));

  const contextCondition = this.buildContextCondition(filter);
  const sql = `
    WITH context AS (
      SELECT MAX(sequence_number) AS max_seq
      FROM events
      WHERE ${contextCondition}
    )
    INSERT INTO events (event_type, payload, metadata)
    SELECT
      unnest($1::text[]),
      unnest($2::jsonb[]),
      unnest($3::jsonb[])
    FROM context
    WHERE COALESCE(max_seq, 0) = $4
  `;

  const result = await this.pool.query(sql, [
    eventTypes, payloads, metadata, expectedMaxSequence
  ]);

  if (result.rowCount === 0) {
    throw new ConcurrencyError(
      'Context changed: events were modified between query and append'
    );
  }
}
```

### Complete Feature: Deposit Money

```typescript
// State
interface DepositState {
  exists: boolean;
}

// Fold
function foldDepositState(events: HasEventType[]): DepositState {
  return {
    exists: events.some((e) => e.eventType() === "AccountOpened"),
  };
}

// Decide
function decideDeposit(
  state: DepositState,
  command: { accountId: string; amount: number },
): Result<MoneyDeposited, { type: string; message: string }> {
  if (!state.exists) {
    return {
      success: false,
      error: { type: "AccountNotFound", message: "Account does not exist" },
    };
  }
  if (command.amount <= 0) {
    return { success: false, error: { type: "InvalidAmount", message: "Amount must be positive" } };
  }
  return {
    success: true,
    event: new MoneyDeposited(command.accountId, command.amount),
  };
}

// Execute (imperative shell)
async function executeDeposit(store: IEventStore, command: DepositCommand): Promise<void> {
  const filter = EventFilter.createFilter(["AccountOpened"]).withPayloadPredicate(
    "accountId",
    command.accountId,
  );

  const { events, maxSequenceNumber } = await store.query(filter);
  const state = foldDepositState(events);
  const result = decideDeposit(state, command);

  if (!result.success) {
    throw new DomainError(result.error.type, result.error.message);
  }

  await store.append(filter, [result.event], maxSequenceNumber);
}
```

---

## 12. Full Code Example (Rust)

### Event Definitions

```rust
pub enum AccountEvent {
    AccountCreated { account_id: Uuid, version: String },
    MoneyDeposited { account_id: Uuid, amount: f64, version: String },
    MoneyWithdrawn { account_id: Uuid, amount: f64, version: String },
}

pub trait AccountEventTrait {
    fn event_type(&self) -> &'static str;
}
```

### State & Fold

```rust
#[derive(Debug, Clone, PartialEq)]
pub struct AccountState {
    pub exists: bool,
}

pub fn fold_state(events: &[AccountEvent]) -> AccountState {
    AccountState {
        exists: events.iter().any(|e| matches!(e, AccountEvent::AccountCreated { .. })),
    }
}
```

### Decide

```rust
#[derive(Debug, Clone)]
pub struct DepositMoney {
    pub account_id: Uuid,
    pub amount: f64,
}

#[derive(Debug, thiserror::Error, PartialEq)]
pub enum DepositError {
    #[error("Account does not exist")]
    AccountNotFound,
    #[error("Amount must be positive")]
    NegativeOrZeroAmount,
}

pub fn process_command(
    state: &AccountState,
    cmd: &DepositMoney,
) -> Result<Vec<AccountEvent>, DepositError> {
    if !state.exists {
        return Err(DepositError::AccountNotFound);
    }
    if cmd.amount <= 0.0 {
        return Err(DepositError::NegativeOrZeroAmount);
    }
    Ok(vec![AccountEvent::MoneyDeposited {
        account_id: cmd.account_id,
        amount: cmd.amount,
        version: default_version(),
    }])
}
```

### Imperative Shell

```rust
pub async fn execute(
    pool: &Pool<sqlx::Postgres>,
    command: DepositMoney,
) -> Result<(), ExecuteError> {
    // Load context
    let past = load_events(pool, command.account_id).await
        .map_err(ExecuteError::Infrastructure)?;

    // Pure fold
    let state = fold_state(&past);

    // Pure decide
    let new_events = process_command(&state, &command)
        .map_err(ExecuteError::Domain)?;

    // Atomic append
    let mut tx = pool.begin().await.map_err(PostgresError::Sqlx)?;
    for ev in &new_events {
        tx = append_event(tx, command.account_id, ev, ev.event_type()).await
            .map_err(ExecuteError::Infrastructure)?;
    }
    tx.commit().await.map_err(PostgresError::Sqlx)?;

    Ok(())
}
```

### Test

```rust
#[test]
fn deposit_succeeds_for_existing_account() {
    let id = Uuid::new_v4();
    let state = AccountState { exists: true };
    let cmd = DepositMoney { account_id: id, amount: 42.0 };

    let events = process_command(&state, &cmd).expect("should succeed");

    assert_eq!(events.len(), 1);
    assert!(matches!(&events[0], AccountEvent::MoneyDeposited { amount, .. } if *amount == 42.0));
}

#[test]
fn deposit_fails_for_nonexistent_account() {
    let state = AccountState { exists: false };
    let cmd = DepositMoney { account_id: Uuid::new_v4(), amount: 10.0 };

    let result = process_command(&state, &cmd);

    assert_eq!(result, Err(DepositError::AccountNotFound));
}
```

---

## 13. Comparison with Traditional ES

### Side-by-Side: Opening a Bank Account

**Traditional (aggregate-based)**:

```
1. Load aggregate by ID (stream: "account-123")
   → Empty stream (new account)

2. Aggregate.decide(OpenAccount command)
   → AccountOpened event

3. Append to stream "account-123" with expected version 0
   → Success (stream was empty)
```

**Aggregateless**:

```
1. Query context: events where type='AccountOpened' AND payload->>'accountId'='123'
   → Empty result, max_seq = 0

2. fold([]) → { exists: false }
   decide({ exists: false }, command) → AccountOpened event

3. CTE INSERT checking max_seq = 0
   → Success (no new matching events)
```

### Side-by-Side: Binding Device to Asset (Cross-Aggregate)

**Traditional**:

```
1. This spans TWO aggregates (Device, Asset)
   → Need a domain service or saga

2. Domain Service:
   - Load Device aggregate
   - Load Asset aggregate
   - Check both exist
   - Check neither is already bound
   - Emit DeviceBoundToAsset event
   - BUT: which aggregate owns this event?
   - Emit to BOTH streams? (dual-write problem)
   - Use a saga/process manager? (complex state machine)

3. Multiple approaches, all with tradeoffs:
   - Dual-write: risk of inconsistency
   - Saga: 3+ classes, state machine, compensation
   - Process manager: additional event handler, complexity
```

**Aggregateless**:

```
1. Query context:
   events where type IN ('DeviceRegistered', 'AssetRegistered',
                         'DeviceBoundToAsset', 'DeviceUnboundFromAsset')
   AND (payload->>'deviceId' = 'dev-456' OR payload->>'assetId' = 'asset-789')

2. fold(events) → { deviceExists: true, assetExists: true, binding: null }
   decide(state, command) → DeviceBoundToAsset event

3. CTE INSERT checking max_seq unchanged
   → Done. No saga. No process manager. No dual-write.
```

### Comparison Matrix

| Aspect                          | Traditional ES                           | Aggregateless ES                 |
| ------------------------------- | ---------------------------------------- | -------------------------------- |
| **Boundary definition**         | Design-time (aggregate ID)               | Runtime (query filter)           |
| **Cross-boundary ops**          | Sagas, process managers, domain services | Wider context query              |
| **Storage layout**              | Stream per aggregate                     | Single events table              |
| **Consistency check**           | Expected stream version                  | CTE with context filter          |
| **State shape**                 | One per aggregate (fixed)                | One per feature (flexible)       |
| **Schema evolution**            | Upcasting per stream                     | Filter-based, more forgiving     |
| **Tooling**                     | Mature (EventStoreDB, Axon, etc.)        | Early (DIY on PostgreSQL)        |
| **Learning curve**              | DDD + ES concepts                        | SQL + functional programming     |
| **Performance (single entity)** | Stream read (fast)                       | Filtered query (indexed, fast)   |
| **Performance (cross-entity)**  | Multiple reads + coordination            | Single query                     |
| **Snapshotting**                | Per-aggregate snapshots                  | Per-feature snapshots (optional) |
| **Event replay**                | Per-aggregate replay                     | Per-feature replay               |
| **Projections**                 | Subscribe to all events / stream         | Per-feature fold or subscription |

---

## 14. Strengths

### 14.1 Eliminates the Hardest Problem in DDD

Aggregate boundary design is the #1 source of complexity in traditional ES. Aggregateless removes it entirely.

### 14.2 Natural Cross-Entity Operations

Operations spanning multiple entities (device + asset, account + counterparty) work naturally through wider context queries. No sagas or process managers needed.

### 14.3 Minimal State Per Feature

Each feature fold reconstructs only the state it needs. No wasted computation hydrating full aggregate state for simple operations.

### 14.4 Pure Functions Everywhere

`fold` and `decide` are pure functions. Testing is trivial — no database, no mocks, no setup:

```typescript
// Complete test. No infrastructure.
expect(decide({ exists: false }, cmd)).toEqual({ success: true, event: ... });
```

### 14.5 Single Table, Simple Schema

One `events` table with JSONB. No stream management. No complex indexing strategies. PostgreSQL does the heavy lifting with GIN indexes.

### 14.6 Feature Independence

Feature slices evolve independently. Adding a new feature doesn't require changing existing aggregates or their tests. New fold, new filter, new decide function — zero impact on existing code.

### 14.7 Transparent Consistency

The consistency boundary is explicit in the filter definition. You can see exactly which events are considered for each operation. No hidden aggregate invariants.

---

## 15. Weaknesses & Open Questions

### 15.1 Performance at Scale

- **Single table**: All events in one table. Partitioning needed for high-volume systems.
- **JSONB queries**: GIN indexes help, but complex filters may be slower than stream reads by ID.
- **No native snapshotting**: Each feature must implement its own snapshot strategy.
- **Growing table**: No built-in archival or compaction (unlike EventStoreDB's scavenge).

### 15.2 Tooling Immaturity

- **No dedicated database**: Must use PostgreSQL. No purpose-built event store (cf. EventStoreDB/KurrentDB).
- **No framework support**: No off-the-shelf library. You build the store, filters, and CTE logic yourself.
- **No admin UI**: No event browser, no stream inspector, no projection dashboard.
- **No subscription mechanism**: Built on queries, not subscriptions. Real-time projections need additional infrastructure (listen/notify, CDC, polling).

### 15.3 Schema Evolution

- **JSONB is flexible but dangerous**: No enforced schema. Typos in field names silently create "new" fields.
- **No built-in upcasting**: Upcasting must be handled in the fold function, mixing schema migration with business logic.
- **No schema registry**: No way to enforce compatibility between event producers and consumers.

### 15.4 Eventual Consistency of Projections

- The approach focuses on write-side consistency. Read-side projections (materialized views) are not addressed.
- No built-in mechanism for subscribing to new events and updating read models.
- Would need PostgreSQL LISTEN/NOTIFY, Debezium CDC, or polling to keep projections current.

### 15.5 Context Query Correctness

- The filter defines the consistency boundary. Getting the filter wrong means incorrect consistency.
- No compile-time or runtime validation that a filter is "complete" for a given decision.
- Easy to accidentally exclude relevant event types from a filter.

### 15.6 Concurrency Under Load

- CTE approach serializes writes to the same context. High-contention contexts become bottlenecks.
- No sharding strategy for contexts that attract many concurrent writers.
- Retry storms possible under high contention.

### 15.7 No Temporal Queries

- The approach is optimized for "decide now based on current state." Point-in-time queries ("what was the state at time T?") are possible but require filtering by `occurred_at`.
- No bi-temporal support.

---

## 16. When to Consider It

### Good Fit

- **Ambiguous aggregate boundaries** — Domains where DDD experts disagree on aggregate design
- **Cross-entity workflows** — Business processes that naturally span multiple entities
- **Greenfield projects** — No legacy aggregate commitments
- **PostgreSQL-centric stacks** — Already using PostgreSQL, comfortable with JSONB and CTEs
- **Small-to-medium event volume** — Not processing millions of events per second
- **Feature teams** — Teams organized by feature, not by aggregate/domain

### Bad Fit

- **High-volume event streams** — Single table becomes a bottleneck (needs partitioning)
- **Existing aggregate commitments** — Already have working aggregates; migration cost > benefit
- **Need purpose-built event store** — Already using EventStoreDB/KurrentDB with mature operations
- **Real-time projections required** — Need push-based subscriptions, not pull-based queries
- **Strong schema enforcement needed** — Require schema registry, compatibility checks, contract testing

---

## 17. Implications for go-cqrs-lite

### What Aligns

| Aspect                 | go-cqrs-lite                             | Aggregateless                  |
| ---------------------- | ---------------------------------------- | ------------------------------ |
| Library, not framework | ✅ Core principle                        | ✅ Same philosophy             |
| Decider pattern        | ✅ `aggregate.Core` follows decide/apply | ✅ Same pattern                |
| Functional core        | ✅ Pure handler functions                | ✅ Same approach               |
| Feature slices         | ✅ Command handlers are independent      | ✅ Same organization           |
| Backend-agnostic       | ✅ `event.Store` interface               | ✅ Same idea (PostgreSQL impl) |

### What Could Be Explored

| Idea                       | Description                                                                                 |
| -------------------------- | ------------------------------------------------------------------------------------------- |
| **Context-based queries**  | `event.Store.Query(filter)` method that queries across streams                              |
| **CTE append**             | `storage.SQLEventStore` could offer a CTE-based append for cross-stream atomicity           |
| **Feature-specific folds** | Each command handler defines its own fold function (already natural in the Decider pattern) |
| **EventFilter type**       | A reusable filter type for context queries (event types + payload predicates)               |

### What Doesn't Fit

| Aspect                         | Reason                                                                            |
| ------------------------------ | --------------------------------------------------------------------------------- |
| **Dropping aggregate concept** | `aggregate.Root` and `aggregate.Repository` are core abstractions with real value |
| **Single events table**        | `storage.SQLEventStore` uses stream-based layout, optimized for aggregate loading |
| **No subscriptions**           | `event.Bus` provides push-based subscriptions for projections                     |

### Bottom Line

Aggregateless ES is a **thought-provoking critique** of aggregate-based design. Its core insight — that consistency boundaries can be defined by queries rather than aggregate identity — is valuable even if you keep aggregates. The **feature slice** pattern (each handler defines its own state shape and fold function) is already natural in go-cqrs-lite's Decider-based architecture.

The strongest takeaway: **make the consistency boundary explicit and adaptable**, whether through aggregate identity or context queries.
