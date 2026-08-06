# ADR-0117: Command Lifecycle as Event Streams

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0111 (Record type — Command adds nothing)

## Context

Commands are immutable intents: "do X." They have no lifecycle, no status
machine. An earlier proposal (in this conversation) suggested adding an
`IntentStatus` field to Commands (Pending → Accepted → Completed). This was
rejected because:

1. **A Command is a single immutable record.** Status transitions would require
   either mutation (violating immutability) or multiple Command records encoding
   the same intent at different statuses (split brain — which record is the
   "real" command?).

2. **Status IS an event.** "Command accepted," "command rejected," "command
   failed" are facts that happened after the command was submitted. Facts are
   Events. Encoding them as Command status fields conflates intents with facts.

3. **Dead-letter queues, retry tracking, and failure analysis are projections.**
   They are derived data built from lifecycle events, not intrinsic to the
   Command record itself.

## Decision

**Command lifecycle (accepted, rejected, retried, dead-lettered) is tracked via
Event streams. Commands have no status field. Dead-letter queues are projections.**

### Model

```
Stream: Command/cmd-123
  Record 1: {Type: "create_user", StreamType: "Command"}  // one immutable intent

Stream: CommandLifecycle/cmd-123
  Record 1: {Type: "command.received",   CausationID: "cmd-123"}  // server got it
  Record 2: {Type: "command.failed",     CausationID: "cmd-123"}  // transient error
  Record 3: {Type: "command.retried",    CausationID: "cmd-123"}  // retry attempted
  Record 4: {Type: "command.dead-lettered", CausationID: "cmd-123"} // exhausted retries
```

### Projections Over Lifecycle Streams

| Projection        | Source                                   | ADT     | Query                               |
| ----------------- | ---------------------------------------- | ------- | ----------------------------------- |
| Dead-letter queue | `command.dead-lettered` events           | Set     | "Which commands are dead-lettered?" |
| Retry count       | `command.retried` events                 | Counter | "How many retries has cmd-123 had?" |
| Failed commands   | `command.failed` events                  | Log     | "Show recent failures"              |
| Processing time   | `command.received` + `command.completed` | N/A     | "Average processing latency"        |

The planner auto-routes these projections (ADR-0116): Set for DLQ membership,
Counter for retry counts, Log for failure history. The consumer declares the
query; the planner builds the projection.

### What This Means for Record

Commands and Events have **identical Record shape** (ADR-0111). The only
difference is conceptual:

- Command = intent (pre-decision, may be rejected by the decider)
- Event = fact (post-decision, immutable truth)

`StreamType` distinguishes them: `"Command"` vs `"Event"` (or any domain stream
type like `"User"`).

### Full Command+Event Replay

Because commands and events are both Records, the planner can replay them
together. This enables:

- **"What-if" time-travel** — replay with different decisions
- **Command audit** — "what commands were sent to this aggregate?"
- **Idempotency tracking** — "has this command been processed?" (check for
  matching CausationID in the event stream)
- **System reconstruction** — full state from command log + event log

## Alternatives Considered

### A. IntentStatus on Command records

**Rejected.** Mutates immutable records (or requires status-transition records
that are really events). Conflates intent with outcome.

### B. Separate CommandStatus type

**Rejected.** Same problem with a different name. Status transitions are facts;
facts are events.

## Consequences

- **Positive:** Commands are pure immutable intents. No status fields, no
  mutation, no split brain.
- **Positive:** Dead-letter queues, retry tracking, and failure analysis emerge
  naturally as projections over lifecycle event streams.
- **Positive:** The planner can auto-generate DLQ and retry projections
  (ADR-0116) from lifecycle event types alone.
- **Positive:** Full command+event replay enables time-travel and system
  reconstruction.
- **Negative:** Querying command status requires reading lifecycle events, not
  a simple field lookup. This is a projection concern — the planner handles it.
