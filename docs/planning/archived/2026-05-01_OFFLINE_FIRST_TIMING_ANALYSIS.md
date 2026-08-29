# Offline-First Timing Analysis

> Every timing dimension that matters when events and commands are created on the client with full offline support.

**Date:** 2026-05-01
**Context:** Extends `2026-05-01_METADATA_AND_TIMING_DELAY_ANALYSIS.md` with the client-side/offline dimension.
**Cross-references:** LiveStore deep dive (`docs/research/2026-05-01_LIVESTORE_DEEP_DIVE.md`), CRDT primitives (`docs/planning/2026-04-25_CROSS_PROJECT_REVIEW.md`), Time travel (`docs/research/time-travel-options.md`)

---

## Table of Contents

1. [The Fundamental Shift](#1-the-fundamental-shift)
2. [Timing Dimensions (Complete Inventory)](#2-timing-dimensions-complete-inventory)
3. [Per-Phase Latency Budget](#3-per-phase-latency-budget)
4. [Clock Problems](#4-clock-problems)
5. [Time Zone Problems](#5-time-zone-problems)
6. [Conflict Scenarios (Timing-Induced)](#6-conflict-scenarios-timing-induced)
7. [What Needs to Exist on Every Message](#7-what-needs-to-exist-on-every-message)
8. [What go-cqrs-lite Has vs. Needs](#8-what-go-cqrs-lite-has-vs-needs)
9. [Architecture Implications](#9-architecture-implications)
10. [Recommendations](#10-recommendations)

---

## 1. The Fundamental Shift

When all events and commands originate on the server, timing is mostly about **latency** (how fast can we process?). When clients create events and commands offline, timing becomes about **causality, ordering, and reconciliation**. The shift is fundamental:

| Server-Only                   | Client-Side + Offline                      |
| ----------------------------- | ------------------------------------------ |
| Single clock (server)         | N clocks (N clients + server)              |
| Events are ordered at write   | Events may be reordered at sync            |
| Conflicts are rare (lock/CAS) | Conflicts are common (offline edits)       |
| `time.Now()` is authoritative | `time.Now()` is unreliable                 |
| Version = global order        | Version = per-stream order only            |
| OccurredAt ≈ PublishedAt      | OccurredAt may be minutes/days before sync |

---

## 2. Timing Dimensions (Complete Inventory)

### 2.1 Client-Side Timestamps

| Timestamp                                                | When Set                                    | By Whom             | Purpose                                         | Problem if Missing                                   |
| -------------------------------------------------------- | ------------------------------------------- | ------------------- | ----------------------------------------------- | ---------------------------------------------------- |
| **`ClientOccurredAt`**                                   | When the event happened on the client       | Client device clock | Business truth: "user actually clicked at 9:03" | Can't distinguish real-time from stale events        |
| **`ClientCreatedAt`**                                    | When the event object was instantiated      | Client SDK          | Debugging, client-side ordering                 | Can't debug client-side race conditions              |
| **`ClientReceivedAt`**                                   | When the command was received by the server | Server clock        | Measuring upload lag, SLA                       | Can't measure how stale the command was              |
| **`ClientID`** (not a timestamp but needed for ordering) | At client registration                      | Server or client    | Attribution, conflict detection                 | Can't detect concurrent edits from different clients |

### 2.2 Server-Side Timestamps

| Timestamp               | When Set                                 | Purpose                                          | Problem if Missing                        |
| ----------------------- | ---------------------------------------- | ------------------------------------------------ | ----------------------------------------- |
| **`ServerReceivedAt`**  | When the event/command arrives at server | Upload lag measurement, dedup window             | Can't detect replayed/delayed messages    |
| **`ServerStoredAt`**    | When the event is durably persisted      | Ground truth for server-side ordering            | Can't establish canonical order           |
| **`ServerPublishedAt`** | When the event is published to the bus   | Outbox lag measurement                           | Can't detect stuck outbox entries         |
| **`GlobalPosition`**    | Monotonic counter at persist time        | Cross-aggregate ordering, catch-up subscriptions | Can't do cross-aggregate temporal queries |
| **`SchemaVersion`**     | At event creation                        | Upcasting decisions                              | Already exists ✅                         |

### 2.3 Sync Timestamps

| Timestamp          | When Set                               | Purpose                             | Problem if Missing                            |
| ------------------ | -------------------------------------- | ----------------------------------- | --------------------------------------------- |
| **`SyncPushedAt`** | When the client pushes to server       | Measuring offline duration          | Can't measure how long the client was offline |
| **`SyncAckedAt`**  | When server acknowledges the push      | Confirming durable receipt          | Client doesn't know if push succeeded         |
| **`SyncPulledAt`** | When the client last pulled            | Staleness of local read model       | Client reads stale data without knowing       |
| **`RebaseAt`**     | When events were rebased onto upstream | Audit trail for conflict resolution | Can't explain why events changed order        |

### 2.4 Projection / Read-Model Timestamps

| Timestamp                   | When Set                                   | Purpose                    | Problem if Missing                        |
| --------------------------- | ------------------------------------------ | -------------------------- | ----------------------------------------- |
| **`ProjectionProcessedAt`** | When the projection handled the event      | Projection lag measurement | Can't tell how far behind a projection is |
| **`CheckpointSavedAt`**     | When the checkpoint was saved              | Lag dashboards             | Currently only stores EventID, not time   |
| **`ReadModelUpdatedAt`**    | When the read model state was last updated | Staleness signals for UI   | UI can't show "data from 5 min ago"       |

### 2.5 UX / Application Timestamps

| Timestamp            | When Set                                    | Purpose                                   | Problem if Missing                               |
| -------------------- | ------------------------------------------- | ----------------------------------------- | ------------------------------------------------ |
| **`LocalAppliedAt`** | When the event was applied to local state   | Optimistic UI updates                     | Can't show "pending" vs "confirmed" state        |
| **`ConfirmedAt`**    | When the server confirmed the event         | UI state transition (pending → confirmed) | UI stuck in "saving..." forever on network error |
| **`TombstonedAt`**   | When a locally-deleted event was tombstoned | Soft deletes during sync                  | Hard deletes conflict with concurrent edits      |

---

## 3. Per-Phase Latency Budget

The full lifecycle of an offline-created event, with every delay point:

```
CLIENT DEVICE (offline)
│
│  0ms   User action occurs
│         → event.ClientOccurredAt = device clock
│         → event.ClientCreatedAt = SDK clock
│         → Local store save (event + projection)
│         → UI updates optimistically
│
│  ???    Device is OFFLINE (seconds to days)
│         → Events accumulate in local outbox
│         → Read model diverges from server
│         → Other clients may mutate same aggregate
│
│  +0ms  Connectivity restored
│         → SyncPushedAt = now()
│
│  +Δ₁   Network latency (client → server)
│         → 50ms-2s depending on connection
│
│  +Δ₂   Server receives
│         → ServerReceivedAt = server clock
│         → Clock skew: ServerReceivedAt ≠ ClientOccurredAt
│
│  +Δ₃   Conflict detection & resolution
│         → Pull-before-push: server sends current state
│         → Rebase: local events rearranged onto server events
│         → Or: reject with current version, client must merge
│
│  +Δ₄   Server persists (Store.Save)
│         → ServerStoredAt = server DB clock
│         → GlobalPosition assigned (monotonic)
│
│  +Δ₅   Outbox append (same TX)
│         → Pending publish
│
│  +Δ₆   Outbox poll interval (up to 1s default)
│         → Background goroutine picks up
│
│  +Δ₇   Bus.Publish to subscribers
│         → ServerPublishedAt = server clock
│
│  +Δ₈   Projection processing
│         → ProjectionProcessedAt
│         → CheckpointSavedAt
│
│  +Δ₉   Other client pulls
│         → SyncPulledAt on other device
│         → Other client replays + updates local read model
│         → Other client UI updates
│
│  ────────────────────────────────────────────
│  TOTAL: ClientOccurredAt → OtherClientSeesIt
│         = offline_duration + Δ₁..Δ₉
│         Could be: 50ms (online) → days (offline) → seconds (re-sync)
```

### Key Insight

The **total latency** from "event happened" to "all clients see it" is:

```
total = offline_duration + network_latency + conflict_resolution + server_processing + outbox_poll + bus_publish + projection + other_client_pull
```

Currently go-cqrs-lite can measure **none** of these phases individually. Every Δ is invisible.

---

## 4. Clock Problems

### 4.1 Clock Skew Between Client and Server

**Problem:** `ClientOccurredAt` uses the device clock. Device clocks drift. A phone that was in airplane mode for 3 days may have its clock off by seconds or even minutes.

**Impact:**

- Events appear to happen "in the future" relative to server time
- Temporal queries break (`LoadToTimestamp` returns wrong results)
- Causation chains look impossible (effect before cause)

**Mitigations:**

| Approach                            | How                                                                              | Tradeoff                                          |
| ----------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------- |
| **Server timestamps are canonical** | Ignore client `OccurredAt` for ordering; use `ServerStoredAt` / `GlobalPosition` | Loses business-time fidelity                      |
| **HLC (Hybrid Logical Clock)**      | Combine physical clock + logical counter; detect skew                            | More complex; requires HLC on every message       |
| **Clock offset protocol**           | Client measures RTT to server, computes offset                                   | Requires connectivity; doesn't work fully offline |
| **NTP discipline**                  | Assume devices use NTP; document max expected skew                               | Pragmatic; phones do this reasonably well         |
| **CausationID chain**               | Don't order by time; order by causation links                                    | Only works for directly-caused events             |

### 4.2 Clock Skew Between Clients

**Problem:** Two offline clients editing the same aggregate. Client A's clock says 10:00, Client B's says 10:05. They both create events "at" different times, but the real ordering is unknown.

**Impact:**

- Last-write-wins by timestamp is **wrong** if clocks disagree
- Merging by `ClientOccurredAt` produces incorrect order

**Mitigations:**

| Approach                         | How                                                                      | Tradeoff                                               |
| -------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------------ |
| **Version-based ordering**       | Order by `(AggregateID, Version)` not by timestamp                       | Only works per-aggregate; doesn't cross aggregates     |
| **Vector clocks**                | Track causal history per client; detect concurrent edits                 | Requires `ClientID` on every event; vector clock grows |
| **Server as tiebreaker**         | Server assigns `GlobalPosition` at persist time; this is canonical order | Client may need to rebase (reorder) local events       |
| **Operational Transform / CRDT** | Make events commutative; order doesn't matter                            | Requires domain-specific CRDT design per event type    |

### 4.3 Clock Going Backward

**Problem:** Device clock is corrected (NTP step), user changes timezone, or device resyncs after airplane mode. `time.Now()` can return a value **earlier** than a previous call.

**Impact:**

- ULIDs generated after the step may sort **before** earlier ULIDs
- `OccurredAt` timestamps are not monotonically increasing
- Projections that rely on time ordering break

**Mitigations:**

| Approach                                | How                                                                        | Tradeoff                                                             |
| --------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **Monotonic clock**                     | Use `time.Now()` with monotonic reading (Go's default)                     | Only monotonic within a single process invocation; resets on restart |
| **HLC**                                 | Logical counter increments when physical clock goes backward               | Standard solution; requires HLC implementation                       |
| **Don't use client clock for ordering** | Use `Version` + `GlobalPosition` only                                      | Timestamps become metadata, not ordering keys                        |
| **Detect and flag**                     | If `time.Now() < lastKnownTime`, mark event with `ClockWentBackward: true` | Defensive; lets projections decide what to do                        |

---

## 5. Time Zone Problems

Time zones are a **separate axis** from clock skew. Even with perfectly synchronized clocks, two clients in different time zones will disagree on "what day is it" — and business rules often depend on local dates.

### 5.1 The Business-Day Problem

```
Client in Tokyo (UTC+9):  Order placed at 2026-05-01 01:00 JST
                          = 2026-04-30 16:00 UTC

Server in Berlin (UTC+1): Sees the order on April 30 (UTC)

Business question: "Was this order placed in May?"
  - Tokyo says: YES (local date is May 1)
  - Berlin/UTC says: NO (UTC date is April 30)
```

If `OccurredAt` stores UTC only (as `time.Now()` does in Go), the **business-local date is lost**. Every downstream system (reporting, SLA calculations, regulatory filing) must guess the user's timezone.

**Impact:** Financial reporting, SLA deadlines, "orders today" dashboards, labor law compliance (overtime rules depend on local clock time).

### 5.2 The Traveler's Paradox

```
User in Berlin creates event at 09:00 CET (UTC+1).
User boards flight to New York.
User arrives, device timezone changes to EST (UTC-5).
User creates event at 09:00 EST (UTC+1 was 6 hours ago).

Now the device's local clock says 09:00, but the two events
are 6 hours apart in UTC — yet both say "9am" to the user.
```

If the event records `ClientTimezone` at creation time, downstream systems can reconstruct the user's local time. Without it, the local-time context is permanently lost.

### 5.3 Daylight Saving Time Ambiguity

```
Fall-back (clocks set back 1 hour):
  1:30 AM EDT → 1:30 AM EST  (same local time, two different UTC offsets)
  An event at "1:30 AM" is ambiguous — which 1:30 AM?

Spring-forward (clocks jump forward 1 hour):
  2:00 AM → 3:00 AM  (2:00-2:59 AM never exists)
  An event logged at "2:30 AM" by a buggy client is impossible.
```

**Impact:** Events created during DST transitions are ambiguous without the UTC offset. `time.Time` in Go stores both wall clock and monotonic time, so `OccurredAt` recorded as `time.Now()` is internally unambiguous. But when serialized to JSON/database and read back, the monotonic component is lost. If the original timezone offset wasn't stored, ambiguity returns.

### 5.4 Projection Determinism Across Time Zones

A projection that groups events by "business day" produces **different results** depending on which timezone is used for the day boundary:

```
Event at 2026-05-01 00:30 UTC

- Grouped by UTC:       → May 1
- Grouped by EST (UTC-5): → April 30
- Grouped by JST (UTC+9): → May 1
```

If the projection uses the server's timezone, events from Tokyo clients appear on the "wrong" day. If it uses the client's timezone, the same event appears on different days depending on which client you ask. If it uses UTC, it's consistent but may not match business expectations.

**The LiveStore determinism rule applies here too:** Projections must choose a single timezone strategy and apply it consistently. Mixing client local time with server time in projections is a source of silent data corruption.

### 5.5 Required Time Zone Metadata

| Field                 | Type                                         | Source        | Purpose                                                     |
| --------------------- | -------------------------------------------- | ------------- | ----------------------------------------------------------- |
| **`ClientTimezone`**  | `string` (IANA name, e.g. `"Europe/Berlin"`) | Client SDK    | Reconstruct business-local time from UTC                    |
| **`ClientUTCOffset`** | `int` (seconds)                              | Client SDK    | Disambiguate DST transitions; offset at event creation time |
| **`ServerTimezone`**  | `string` (IANA name)                         | Server config | Default for server-side events; projection grouping         |

**Why IANA name, not just offset?** An offset of `UTC+1` could be CET, CET/CEST (DST), or WAT. The IANA name (e.g. `"Europe/Berlin"`) encodes the DST rules, enabling correct historical calculations. The offset alone is insufficient.

**Why both IANA name AND offset?** The IANA name gives you the rules; the offset is a snapshot of the rules at event creation time. If DST rules change retroactively (they do — governments legislating timezone changes), the offset preserves the _original intent_ while the IANA name allows recalculation.

### 5.6 Time Zone Strategy Matrix

| Strategy                          | Pros                                                      | Cons                                                 | When to Use                                 |
| --------------------------------- | --------------------------------------------------------- | ---------------------------------------------------- | ------------------------------------------- |
| **UTC everywhere**                | Simple, deterministic, no ambiguity                       | Loses business-local date context                    | Internal systems, non-regulatory            |
| **UTC + ClientTimezone metadata** | Preserves business context; UTC is canonical for ordering | Two interpretations of every event                   | Default recommendation                      |
| **Server timezone**               | Simple for single-region deployments                      | Wrong for global users                               | Internal tools, single-region apps          |
| **Client timezone (no UTC)**      | "What the user saw"                                       | Non-deterministic across clients; can't order events | Never — only for display, never for storage |
| **Bi-temporal with timezone**     | Full fidelity                                             | Complex; requires `ValidAt` + timezone               | Financial/regulatory compliance             |

**Recommended default:** Store all timestamps as UTC. Add `ClientTimezone` + `ClientUTCOffset` to event metadata. Use UTC for ordering and projections. Use `ClientTimezone` for display and business-day calculations.

---

## 6. Conflict Scenarios (Timing-Induced)

### 6.1 The Lost Update (Same Aggregate, Two Offline Clients)

```
Server:  User(v1) ──── event1: NameChanged("Alice") ──── v2
Client A (offline since v1):                    event1a: EmailChanged("a@x.com") ──── v2
Client B (offline since v1):                    event1b: EmailChanged("b@y.com") ──── v2
```

Both clients expect version 2. When they sync, one will fail the optimistic concurrency check. **Timing dimension:** The longer the offline period, the higher the probability of this conflict.

**Required metadata:** `ClientID`, `ExpectedVersion`, `ClientOccurredAt`

### 6.2 The Causal Reversal (Cross-Aggregate)

```
Client A (offline):  OrderCreated → PaymentProcessed
Client B (online):   OrderCreated → OrderCancelled

After sync, server sees:
  OrderCreated, PaymentProcessed, OrderCancelled

But in business reality, the cancellation happened BEFORE the payment.
The server's version-based ordering can't express this.
```

**Timing dimension:** `CausationID` chain is needed to express that `PaymentProcessed` was caused by `OrderCreated` at a specific time, not by the current server state.

**Required metadata:** `CausationID`, `ClientOccurredAt`, possibly `ValidAt` (bi-temporal)

### 6.3 The Thundering Herd (Long Offline → Sudden Sync)

```
Client goes offline for 3 days.
Accumulates 500 events across 20 aggregates.
Comes online: all 500 events hit the server simultaneously.

Server-side effects:
- 500 Store.Save calls (some will hit concurrency conflicts)
- 500 Outbox entries to publish
- Projections overwhelmed
- Bus overloaded
```

**Timing dimension:** No backpressure mechanism exists. `OutboxPublisher` processes all entries in one batch. The server has no way to rate-limit incoming sync batches.

**Required metadata:** `SyncPushedAt`, `OfflineEventCount` (for the server to know the scale of the incoming batch)

### 6.4 The Stale Read (Client Reads Before Pull)

```
Client was offline for 1 hour.
User opens app → sees old read model.
User takes action based on stale state.
Event conflicts with server state.

Example: User sees item "in stock" (stale), adds to cart.
Server: item was sold out 30 minutes ago.
```

**Timing dimension:** `SyncPulledAt` tells the client how stale its read model is. Without it, the client can't warn the user or make informed decisions.

**Required metadata:** `SyncPulledAt` on the client, `ReadModelUpdatedAt` on the read model

### 6.5 The Duplicate Command (Network Timeout + Retry)

```
Client sends CreateOrder command.
Network times out (TCP timeout, not app-level).
Client doesn't know if it succeeded.
Client retries (or user retries).

If the first attempt succeeded: duplicate order.
```

**Timing dimension:** Commands need idempotency keys. The `RequestID` exists on events but not commands.

**Required metadata:** `IdempotencyKey` on Command, `RequestID` on Command

### 6.6 The Timezone Illusion (Same Event, Different Business Day)

```
Client in Tokyo creates event at 2026-05-01 00:30 JST (April 30 UTC).
Client in Berlin sees the same event, grouped under "April 30" by UTC.
Business question: "How many orders in May?"
  - Tokyo's view: 1 order in May
  - Berlin/UTC view: 0 orders in May
  - Regulatory filing (Japan): May
  - Regulatory filing (EU): April
```

**Timing dimension:** Without `ClientTimezone` metadata, the business-local date is permanently lost. Reports, SLAs, and regulatory filings are wrong. This is a **silent data corruption** scenario — the data looks correct but answers different questions depending on the reader's timezone.

**Required metadata:** `ClientTimezone`, `ClientUTCOffset` on Event; projection must document which timezone strategy it uses for day-grouping

---

## 7. What Needs to Exist on Every Message

### 7.1 Event (Extended for Offline)

| Field                  | Type                     | Source                  | Why                                                                                 |
| ---------------------- | ------------------------ | ----------------------- | ----------------------------------------------------------------------------------- |
| `ID`                   | `id.EventID` (ULID)      | Client SDK              | Already exists ✅                                                                   |
| `Type`                 | `Type`                   | Client                  | Already exists ✅                                                                   |
| `AggregateID`          | `id.AggregateID`         | Client                  | Already exists ✅                                                                   |
| `AggregateType`        | `AggregateType`          | Client                  | Already exists ✅                                                                   |
| `Version`              | `Version`                | Client (expected)       | Already exists ✅                                                                   |
| `SchemaVersion`        | `int`                    | Client                  | Already exists ✅                                                                   |
| `Payload`              | `[]byte`                 | Client                  | Already exists ✅                                                                   |
| `OccurredAt`           | `time.Time`              | **Client device clock** | Already exists ✅ — but now it's the CLIENT clock, not the server clock             |
| **`ClientID`**         | `id.ClientID`            | Client SDK              | **NEW** — Who created this event? Needed for conflict attribution, vector clocks    |
| **`ClientCreatedAt`**  | `time.Time`              | Client SDK              | **NEW** — SDK-level timestamp (may differ from OccurredAt for reconstructed events) |
| **`ServerReceivedAt`** | `time.Time`              | Server                  | **NEW** — When the server first saw this event; measures upload lag                 |
| **`ServerStoredAt`**   | `time.Time`              | Server DB               | **NEW** — When the event was durably persisted; canonical timestamp                 |
| **`GlobalPosition`**   | `uint64`                 | Server DB               | **NEW** — Monotonic counter; cross-aggregate ordering                               |
| **`IsLocalOnly`**      | `bool`                   | Client SDK              | **NEW** — Skip sync for UI-only events (LiveStore pattern)                          |
| **`ClientTimezone`**   | `string` (IANA)          | Client SDK              | **NEW** — e.g. `"Europe/Berlin"`; enables business-local time reconstruction        |
| **`ClientUTCOffset`**  | `int` (seconds)          | Client SDK              | **NEW** — Offset snapshot at creation time; disambiguates DST                       |
| **`SyncPushedAt`**     | `time.Time`              | Client SDK              | **NEW** — When this event was pushed to the server                                  |
| **`SyncAckedAt`**      | `time.Time`              | Server                  | **NEW** — When the server confirmed durable receipt                                 |
| `CorrelationID`        | `id.CorrelationID`       | Client/Server           | Already exists ✅                                                                   |
| `CausationID`          | `id.CausationID`         | Client                  | Already exists ✅ — Critical for offline causation chains                           |
| `UserID`               | `id.UserID`              | Client                  | Already exists ✅                                                                   |
| `RequestID`            | `id.RequestID`           | Client                  | Already exists ✅                                                                   |
| `Source`               | `Source`                 | Client                  | Already exists ✅ — Value would be "client:ios", "client:web", etc.                 |
| `Custom`               | `map[MetadataKey]string` | Any                     | Already exists ✅ — Extensibility for domain-specific timing data                   |

### 7.2 Command (Extended for Offline)

| Field                 | Type                     | Source        | Why                                               |
| --------------------- | ------------------------ | ------------- | ------------------------------------------------- |
| `Type`                | `Type`                   | Client        | Already exists ✅                                 |
| `AggregateID`         | `id.AggregateID`         | Client        | Already exists ✅                                 |
| **`ClientID`**        | `id.ClientID`            | Client SDK    | **NEW** — For dedup and attribution               |
| **`IdempotencyKey`**  | `string`                 | Client SDK    | **NEW** — Prevents duplicate execution on retry   |
| **`ClientCreatedAt`** | `time.Time`              | Client SDK    | **NEW** — When the command was issued             |
| **`CorrelationID`**   | `id.CorrelationID`       | Client/Server | **NEW** — Distributed tracing                     |
| **`CausationID`**     | `id.CausationID`         | Client        | **NEW** — What triggered this command             |
| **`UserID`**          | `id.UserID`              | Client        | **NEW** — Who issued the command                  |
| **`RequestID`**       | `id.RequestID`           | Client        | **NEW** — Per-request debugging                   |
| **`ExpectedVersion`** | `Version`                | Client        | **NEW** — Optimistic concurrency at command level |
| **`Custom`**          | `map[MetadataKey]string` | Any           | **NEW** — Extensibility                           |

### 7.3 Aggregate Root (Extended for Offline)

| Field                | Type             | Source             | Why                                                                         |
| -------------------- | ---------------- | ------------------ | --------------------------------------------------------------------------- |
| `ID`                 | `id.AggregateID` | Client/Server      | Already exists ✅                                                           |
| `Type`               | `AggregateType`  | Client/Server      | Already exists ✅                                                           |
| `Version`            | `Version`        | Server (canonical) | Already exists ✅                                                           |
| **`LastModifiedBy`** | `id.ClientID`    | Server             | **NEW** — Which client last modified this aggregate?                        |
| **`LastModifiedAt`** | `time.Time`      | Server             | **NEW** — When was this aggregate last changed? (for conflict detection UI) |

---

## 8. What go-cqrs-lite Has vs. Needs

### 8.1 Current State

| Concern                  | Status | Detail                                   |
| ------------------------ | ------ | ---------------------------------------- |
| Per-aggregate versioning | ✅     | `event.Version` on every event           |
| Optimistic concurrency   | ✅     | `Store.Save(events, expectedVersion)`    |
| Event timestamps         | ✅     | `OccurredAt` (single clock, server-only) |
| Correlation/Causation    | ✅     | On events only, not commands             |
| ULID EventIDs            | ✅     | Time-sortable within a single process    |
| Transactional outbox     | ✅     | Server-side only                         |
| Retry with backoff       | ✅     | Middleware for transient failures        |
| Context enricher         | ✅     | Library API, unwired                     |

### 8.2 Gaps for Offline Support

| Concern                                           | Status     | Impact                                                       | Priority |
| ------------------------------------------------- | ---------- | ------------------------------------------------------------ | -------- |
| **ClientTimezone on events**                      | ❌         | Can't reconstruct business-local time from stored events     | **P0**   |
| **ClientUTCOffset on events**                     | ❌         | Can't disambiguate DST transitions                           | **P0**   |
| **ClientID on events**                            | ❌         | Can't attribute events to clients; can't build vector clocks | **P0**   |
| **IdempotencyKey on commands**                    | ❌         | Duplicate commands on network retry                          | **P0**   |
| **Command metadata**                              | ❌         | No tracing through command → event lifecycle                 | **P0**   |
| **Server-side timestamps** (ReceivedAt, StoredAt) | ❌         | Can't measure any latency phase                              | **P0**   |
| **Global position**                               | ❌         | No cross-aggregate ordering; no catch-up subscriptions       | **P1**   |
| **PublishedAt on outbox**                         | ❌         | Can't measure outbox lag                                     | **P1**   |
| **ProcessedAt on checkpoints**                    | ❌         | Can't measure projection lag                                 | **P1**   |
| **IsLocalOnly on events**                         | ❌         | All events go to outbox; no UI-only event support            | **P2**   |
| **Sync timestamps** (PushedAt, AckedAt, PulledAt) | ❌         | Client can't measure offline duration or sync status         | **P2**   |
| **Clock skew detection**                          | ❌         | `time.Now()` is treated as authoritative                     | **P2**   |
| **Pull-before-push / rebase**                     | ❌         | No sync protocol at all                                      | **P3**   |
| **Vector clocks**                                 | 📐 Planned | In `docs/planning/2026-04-25_CROSS_PROJECT_REVIEW.md`        | **P3**   |
| **Conflict resolver**                             | 📐 Planned | In `docs/planning/2026-04-25_CROSS_PROJECT_REVIEW.md`        | **P3**   |
| **Backpressure on sync**                          | ❌         | Thundering herd on reconnect                                 | **P3**   |
| **Bi-temporal (ValidAt)**                         | ❌         | Niche but needed for retroactive corrections                 | **P4**   |
| **HLC**                                           | ❌         | Considered in time-travel research                           | **P4**   |

---

## 9. Architecture Implications

### 9.1 The Five Timestamps Every Event Needs

For offline-first, every event operates across **five temporal dimensions**:

```
1. ClientOccurredAt    "When did this really happen?"     (client clock, unreliable)
2. ServerStoredAt      "When did the server learn about it?" (server DB clock, canonical)
3. GlobalPosition      "In what order vs. all other events?"  (monotonic, machine-order)
4. ValidAt              "When was this true in the business?"  (optional, domain-specific)
5. CausationID         "What caused this?"               (causal ordering, not temporal)
6. ClientTimezone      "In what local context did this happen?" (business-day grouping, display)
```

Ordering by `ClientOccurredAt` is **wrong** for distributed systems. Ordering by `ServerStoredAt` + `GlobalPosition` is **correct** but loses business-time fidelity. The solution: store all five, let each consumer choose which axis to query.

### 9.2 The Sync Protocol (What's Missing)

go-cqrs-lite has the **outbox** (push) but not the **pull** or **rebase**:

```
CURRENT (push-only):
  Client → [events] → Server Store → Outbox → Bus → Projections

NEEDED (pull-before-push):
  Client → Pull (server state) → Rebase local events → Push (rebased) → Server
                                                                 ↓
  Other Client ← Push notification ← Bus ← Outbox ← Store ← ┘
       ↓
  Pull → Rebase local events → Update read model → Update UI
```

The pull-before-push pattern (from LiveStore) eliminates most timing-related conflicts by ensuring the client's events are always rebased onto the server's canonical order before persisting.

### 9.3 The Client Outbox (New Concept)

Currently, `Outbox` is server-side. For offline support, the **client needs its own outbox**:

| Server Outbox                                  | Client Outbox                                     |
| ---------------------------------------------- | ------------------------------------------------- |
| Ensures `Store.Save` + `Bus.Publish` atomicity | Ensures events survive app crash / device restart |
| Poll-based publishing                          | Push-based on connectivity change                 |
| `Ack` = published to bus                       | `Ack` = server confirmed durable receipt          |
| `PollPending` returns entries                  | `PollPending` = all unsynced events               |
| Default 1s poll interval                       | Push immediately on connectivity restore          |

**Key timing difference:** The server outbox delay is measured in **seconds** (poll interval). The client outbox delay is measured in **seconds to days** (offline duration).

### 9.4 The Confirm/Pending Pattern

Offline-first UIs show three states:

| State         | Meaning                                | Visual                    |
| ------------- | -------------------------------------- | ------------------------- |
| **Pending**   | Event created locally, not yet synced  | Gray spinner, "saving..." |
| **Confirmed** | Server acknowledged durable receipt    | Green checkmark           |
| **Conflict**  | Server rejected (concurrency conflict) | Red warning, merge dialog |

Currently, go-cqrs-lite has no concept of "pending" vs "confirmed" events. The `MarkChangesAsCommitted` method name is misleading — it means "cleared from the uncommitted list," not "server confirmed."

---

## 10. Recommendations

### 10.1 P0 — Enable Offline Metadata (Non-Breaking)

These are additive changes that don't break existing APIs:

1. **Add `ClientID` branded type** — `id.Of[ClientMarker]` in `core/pkg/id/`
2. **Add `WithClientID` option** — Sets `Metadata.ClientID` on events
3. **Add `ClientTimezone` and `ClientUTCOffset` to Metadata** — `WithClientTimezone(name string, offset int)` option; enables business-local time reconstruction
4. **Add `IdempotencyKey` to Command** — New field on `command.Core`; `WithIdempotencyKey` option
5. **Add command metadata** — Mirror a subset of event metadata (`CorrelationID`, `CausationID`, `UserID`, `RequestID`, `Custom`)
6. **Add `ServerReceivedAt` to OutboxEntry** — Set by the server when the entry is first created
7. **Add `WithLocalOnly` option** — Marks events to skip the outbox; `EventSourcedRepository.Save` checks this flag

### 10.2 P1 — Enable Latency Measurement (Mostly Additive)

7. **Add `GlobalPosition` to Store interface** — `LoadAllFromPosition(ctx, position, limit)` method
8. **Add `PublishedAt` to OutboxEntry** — Set by `OutboxPublisher` after successful publish
9. **Add `ProcessedAt` to CheckpointStore** — Store `(EventID, time.Time)` instead of just `EventID`
10. **Log outbox poll errors** — Don't silently swallow `PollPending` errors

### 10.3 P2 — Enable Sync Protocol (New Module)

11. **Extract CRDT primitives** — VectorClock, ConflictResolver, LWWResolver from go-localfirst into `sync/` module (per cross-project review)
12. **Define SyncMessage type** — The wire format for client ↔ server sync
13. **Add client outbox interface** — Same `Outbox` interface, but with `Ack` semantics tied to server confirmation
14. **Add sync timestamps** — `SyncPushedAt`, `SyncAckedAt`, `SyncPulledAt` as custom metadata or first-class fields
15. **Clock skew detection** — Add `ClockOffset` metadata field; client measures RTT to server and records estimated offset

### 10.4 P3 — Full Offline-First Architecture

16. **Pull-before-push sync protocol** — Client pulls server state before pushing local events; rebases if needed
17. **Rebase mechanism** — Reorder local events on top of server events; adjust versions
18. **Backpressure on sync** — Rate-limit incoming sync batches; server sends `Retry-After` header
19. **Confirm/pending event states** — Track which events the server has acknowledged; UI shows pending state
20. **Conflict resolver integration** — Wire `sync.ConflictResolver[T]` into the sync protocol for domain-specific merge

### 10.5 P4 — Advanced Temporal Features

21. **HLC implementation** — Hybrid Logical Clock in `sync/` module; replaces `time.Now()` for timestamp generation
22. **Bi-temporal support** — `WithValidAt` option; `LoadToValidTime` store method
23. **Temporal queries** — `LoadToVersion`, `LoadToTimestamp`, `ReadBackwards` on Store interface (per time-travel research)

---

## Appendix: Timing Checklist for Every Message

When adding a new Event or Command type to an offline-first system, verify:

- [ ] Does it carry `ClientTimezone` and `ClientUTCOffset`? (what local context was the user in)
- [ ] Is `OccurredAt` stored as UTC? (always UTC internally; timezone is metadata for display)
- [ ] Do projections group by UTC date, server timezone, or client timezone? (document which)
- [ ] Is the projection deterministic even if the same event is replayed with different `ClientTimezone` values? (grouping strategy must be explicit)
- [ ] Does it carry `ClientID`? (who created it)
- [ ] Does it carry `CausationID`? (what caused it)
- [ ] Does it carry `IdempotencyKey`? (for commands: dedup on retry)
- [ ] Does it carry `ExpectedVersion`? (for commands: optimistic concurrency)
- [ ] Is `OccurredAt` set from the **client clock** or **server clock**? (document which)
- [ ] Is there a `ServerStoredAt` assigned by the server? (canonical timestamp)
- [ ] Is there a `GlobalPosition` for cross-aggregate ordering?
- [ ] Is there an `IsLocalOnly` flag for UI-only events?
- [ ] Are sync timestamps (`SyncPushedAt`, `SyncAckedAt`) tracked?
- [ ] Can the projection handle events arriving out of `OccurredAt` order? (they will)
- [ ] Is the projection deterministic? (no `time.Now()`, no `uuid.New()` inside)
- [ ] Is there a conflict resolution strategy documented for this event type?
- [ ] Is there a UX state for "pending" vs "confirmed"?

---

_Extends `docs/planning/2026-05-01_METADATA_AND_TIMING_DELAY_ANALYSIS.md`. Cross-references: LiveStore deep dive, Time travel research, CRDT cross-project review._
