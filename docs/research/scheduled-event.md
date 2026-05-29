# ScheduledEvent / TimedEvent — Delayed and Deadline-Based Event Processing

> A `ScheduledEvent` is a command or event that should be processed at a future time, or after a delay. Essential for deadlines, timeouts, retries, and time-based business rules.

## Concept

```
Schedule(cmd, at time.Time) → [wait] → cmd dispatched
```

A message that should not be processed immediately, but at a specific future time or after a duration. This is foundational for:

- **Deadlines** ("Cancel order if not paid within 30 minutes")
- **Reminders** ("Send follow-up email 3 days after registration")
- **Scheduled commands** ("Process subscription renewal on the 1st of each month")
- **Saga timeouts** ("Compensate if step 2 doesn't complete in 1 hour")

## Why This Is More Infrastructure Than Library

The concept is simple, but a **durable** scheduler requires:

| Concern             | Why It Matters                                                       |
| ------------------- | -------------------------------------------------------------------- |
| **Timer store**     | Must survive process restarts (database table with `fire_at` column) |
| **Dispatcher loop** | Must poll or be notified when timers fire                            |
| **Clock skew**      | Distributed nodes may disagree on "now"                              |
| **Cancellation**    | What if the condition already resolved? Need `Cancel(scheduleID)`    |
| **Partitioning**    | Which node owns which timers? Needs coordination                     |
| **Exact-once**      | A fired timer must not fire again after restart                      |
| **Backpressure**    | Thousands of timers firing simultaneously                            |

This is fundamentally a **side-effectful infrastructure concern**, not a pure domain primitive. The library should define the interface; consumers bring their own backend.

## Proposed Interface

```go
package scheduler

type ScheduleID = id.Of[scheduleMarker]

type Scheduler interface {
    // Schedule registers a command to be dispatched at the given time.
    // Returns a schedule ID for cancellation.
    Schedule(ctx context.Context, cmd command.Command, at time.Time) (ScheduleID, error)

    // Cancel removes a pending scheduled command.
    // Returns an error if already fired or not found.
    Cancel(ctx context.Context, id ScheduleID) error
}
```

### Extended Interface (Optional)

```go
type Enqueuer interface {
    Scheduler

    // ScheduleIn schedules a command after a relative duration from now.
    ScheduleIn(ctx context.Context, cmd command.Command, delay time.Duration) (ScheduleID, error)

    // Pending returns pending schedules for a given aggregate (for debugging/inspection).
    Pending(ctx context.Context, aggregateID string) ([]ScheduledCommand, error)
}

type ScheduledCommand struct {
    ID          ScheduleID
    Command     command.Command
    FireAt      time.Time
    CreatedAt   time.Time
    AggregateID string
}
```

## Implementation Tiers

### Tier 1: In-Memory (for testing)

```go
type MemoryScheduler struct {
    mu        sync.Mutex
    pending   []ScheduledCommand
    clock     func() time.Time
    onFire    func(ctx context.Context, cmd command.Command) error
}
```

- Fires via goroutine + `time.AfterFunc`
- Lost on process restart (acceptable for tests)
- Included in `memory` module

### Tier 2: SQL-Based (for production)

```go
type SQLScheduler struct {
    db        *sql.DB
    dialect   dialect.Dialect
    table     string
    clock     func() time.Time
    dispatcher command.Dispatcher
}
```

- Table: `scheduled_commands (id, aggregate_id, command_type, payload, fire_at, created_at, status)`
- Poll loop: `SELECT ... WHERE fire_at <= now() AND status = 'pending' FOR UPDATE SKIP LOCKED`
- Acquires row lock, dispatches, marks `status = 'fired'`
- Could live in `storage` module alongside `SQLEventStore`

### Tier 3: External Backends (consumer choice)

Consumers can implement `Scheduler` with:

- PostgreSQL `pg_partman` + `SELECT FOR UPDATE SKIP LOCKED`
- Temporal workflows
- Watermill scheduler middleware
- NATS JetStream delayed messages
- Redis sorted sets (`ZRANGEBYSCORE`)

The library provides the interface; consumers pick the backend.

## Interaction with Existing Modules

### Saga Timeouts

Sagas currently lack timeout handling. A scheduler would enable:

```go
saga.Step{
    Name:    "process-payment",
    Action:  processPaymentCmd,
    Timeout: 30 * time.Minute,
    // Runner schedules a timeout command; if step doesn't complete, saga compensates
}
```

The saga runner would call `scheduler.ScheduleIn(timeoutCmd, step.Timeout)` and `scheduler.Cancel(scheduleID)` on completion.

### Projection Temporal Queries

Projections that need "time since last event" could use scheduled events as triggers:

```go
// "If no OrderConfirmed within 24h of OrderPlaced, emit OrderExpired"
scheduler.ScheduleIn(cancelCmd, 24*time.Hour)
// On OrderConfirmed: scheduler.Cancel(scheduleID)
```

### Decider Deadlines

Deciders could express deadlines as part of their decision logic:

```go
func decide(cmd Command, state State) ([]event.Event, []ScheduleRequest, error) {
    // ...
    return events, []ScheduleRequest{{Cmd: expireCmd, Delay: 30 * time.Minute}}, nil
}
```

## Design Decisions

### Commands vs Events

Should the scheduler dispatch **commands** or **events**?

| Choice       | Implication                                                                                       |
| ------------ | ------------------------------------------------------------------------------------------------- |
| **Commands** | The scheduled message may be rejected (e.g., order already paid). Handler decides. More flexible. |
| **Events**   | The scheduled message is a fact (deadline reached). Cannot be rejected. Simpler but less nuanced. |

**Recommendation:** Schedule **commands**. A timeout is a request ("check if this needs handling"), not a fact. The handler decides what to do.

### Cancellation Granularity

Should cancellation be:

- By schedule ID (exact, explicit)
- By aggregate ID + command type (pattern-based, automatic)

**Recommendation:** Start with schedule ID. Pattern-based cancellation is a convenience layer on top.

### Distributed Safety

For multi-node deployments:

| Strategy                                     | Trade-off                   |
| -------------------------------------------- | --------------------------- |
| Row-level locking (`FOR UPDATE SKIP LOCKED`) | Simple, PostgreSQL-specific |
| Lease-based (claim timer, renew until done)  | Portable, more complex      |
| Single timer owner (partition by hash)       | Efficient, less flexible    |

**Recommendation:** Row-level locking for SQL implementation. Keep the interface backend-agnostic.

## Open Questions

1. Should the scheduler support **cron-style recurring schedules**, or only one-shot?
2. Should there be a **max retention** for fired/cancelled schedules (auto-purge old entries)?
3. How does this interact with **event signing**? Are scheduled commands signed at schedule time or fire time?
4. Should the scheduler be a **module** (`scheduler/`) or live within an existing module (`storage/`)?
5. How to handle **timezone-aware scheduling** for recurring events?
