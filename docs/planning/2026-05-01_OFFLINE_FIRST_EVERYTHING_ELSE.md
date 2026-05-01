# Offline-First: Everything Else We Should Consider

> Dimensions beyond timing that surface when events and commands are created on the client with full offline support.

**Date:** 2026-05-01
**Companion to:** `2026-05-01_OFFLINE_FIRST_TIMING_ANALYSIS.md` and `2026-05-01_METADATA_AND_TIMING_DELAY_ANALYSIS.md`

The timing analysis covers *when* things happen. This document covers everything else that breaks or changes when you go offline-first.

---

## Table of Contents

1. [Network Reality](#1-network-reality)
2. [Schema Evolution on the Client](#2-schema-evolution-on-the-client)
3. [Storage Constraints](#3-storage-constraints)
4. [Security & Trust](#4-security--trust)
5. [Privacy & Legal](#5-privacy--legal)
6. [Multi-Device Same User](#6-multi-device-same-user)
7. [Aggregate Design Under Offline](#7-aggregate-design-under-offline)
8. [UX Patterns for Eventual Consistency](#8-ux-patterns-for-eventual-consistency)
9. [Compensating Events & Undo](#9-compensating-events--undo)
10. [Client-Side Projections](#10-client-side-projections)
11. [Testing Offline Scenarios](#11-testing-offline-scenarios)
12. [Serialization & Wire Format](#12-serialization--wire-format)
13. [Client SDK Architecture](#13-client-sdk-architecture)
14. [Operational Concerns](#14-operational-concerns)
15. [What go-cqrs-lite's Library Shape Means](#15-what-go-cqrs-lites-library-shape-means)

---

## 1. Network Reality

The existing analysis treats connectivity as binary: online or offline. Reality is a spectrum.

### 1.1 The Connectivity Spectrum

| State | Latency | Packet Loss | Duration | Implication |
|-------|---------|-------------|----------|-------------|
| **Online** | <100ms | <1% | Stable | Normal operation |
| **Degraded** | 500ms-5s | 5-20% | Minutes-hours | Timeouts, partial syncs |
| **Intermittent** | Variable | 20-50% | Hours | Sync may start but not finish |
| **Offline** | ∞ | 100% | Hours-days | Full offline mode |
| **Metered** | Variable | Low | Days | User opts out of sync (roaming charges) |

**Implication:** The sync protocol must handle *partial* sync — a batch of 500 events where the connection drops after 200. The server has 200 events, the client doesn't know if they arrived. Idempotent push (can retry the same 200) + `Ack` per event or per batch is required.

### 1.2 Metered Connectivity

Users on mobile data may decline to sync 500 events over a cellular connection. The client needs:

- **Size estimation** before push ("you have 500 events, ~2MB — sync now?")
- **Selective sync** — push only high-priority events (business events, not UI state)
- **Background sync policy** — WiFi-only, unmetered-only, user preference

This is where `IsLocalOnly` matters: UI-only events never consume bandwidth.

### 1.3 Background vs Foreground Sync

| | Background (push notification) | Foreground (user-initiated) |
|--|-------------------------------|----------------------------|
| **Latency tolerance** | High (seconds) | Low (must feel instant) |
| **Battery budget** | Tight | Relaxed |
| **User expectation** | "It'll sync eventually" | "I need this NOW" |
| **Failure handling** | Silent retry later | Show error, offer retry |
| **Priority** | Low-priority events first | User's current context first |

**Implication:** The sync protocol needs a priority field on events or the ability to push a subset based on the aggregate/type.

---

## 2. Schema Evolution on the Client

This is one of the hardest offline problems and it's not about timing at all.

### 2.1 The Version Split

```
Monday:  Server is on schema v3.  Client app is on v3.  All good.
Tuesday: User goes offline.
Wednesday: Server deploys schema v4.  New event types, new payload shapes.
Thursday: User is still offline.  Client creates events using v3 schema.
Friday:  User updates the app to v4 (while still offline).
          → Client now has v3 events in local store, v4 code running.
          → Client creates events using v4 schema.
          → Local store has a MIX of v3 and v4 events.
Saturday: User comes online.  Server expects v4.
          → Must handle v3 events from Thursday.
          → Must handle v4 events from Friday.
```

**The problem:** Both the client and the server can have multiple schema versions in flight simultaneously. The `SchemaVersion` field on events handles server-side upcasting, but it doesn't help the *client* when the server sends events in a format the old client doesn't understand.

### 2.2 Forward vs Backward Compatibility

| Direction | Who needs it | Example | go-cqrs-lite today |
|-----------|-------------|---------|---------------------|
| **Backward** | Server reading old client events | Server receives `v1.UserCreated` from outdated client | ✅ `UpcasterRegistry` handles this |
| **Forward** | Client reading new server events | Old client receives `v2.UserCreated` with extra fields | ❌ No mechanism |
| **Bidirectional** | Both simultaneously | Client on v3, server on v5, events flowing both ways | ❌ No mechanism |

**Forward compatibility on the client requires:**

1. **Unknown event type handling** — Client receives an event type it doesn't recognize. Options:
   - Skip and continue (safe but loses data)
   - Store as opaque blob for future processing
   - Crash / force update (terrible UX)
2. **Unknown field handling** — New payload fields the client doesn't know about. If using JSON, extra fields are silently ignored by default. But if the client re-serializes (for rebase or re-push), those fields are lost.
3. **Minimum version advertisement** — Client tells the server "I understand schema v3" and the server downconverts or rejects.

### 2.3 The Force-Update Problem

If a schema change is non-backward-compatible, the server must reject events from old clients. The client must force the user to update. But:

- User may be offline (can't download the update)
- User may be on a metered connection (can't afford a 50MB app update)
- App store review may delay the update by days
- Enterprise users may have MDM-controlled update cycles (weeks)

**Implication:** The system must have a **grace period** during which both old and new schemas are accepted, and a **sunset policy** for when old schemas are rejected. This is a product decision, not a technical one, but the SDK must support it.

---

## 3. Storage Constraints

### 3.1 Client-Side Storage Budget

| Device | Typical Available | Events (1KB each) | Events (10KB each) |
|--------|-------------------|-------------------|---------------------|
| Phone | 100MB-1GB | 100K-1M | 10K-100K |
| Tablet | 500MB-5GB | 500K-5M | 50K-500K |
| Laptop | 5-50GB | 5M-50M | 500K-5M |
| Browser tab | 50MB-500MB | 50K-500K | 5K-50K |

A user offline for a week won't create millions of events, but **synced events from other users** (the full eventlog or subscription feed) can exhaust storage.

**Implication:** The client can't store the full eventlog. It needs:

- **Per-aggregate retention policy** — Keep last N events, snapshot older ones
- **Projection-first storage** — Only store the read model, not the raw events (LiveStore pattern)
- **Eviction strategy** — Which events to drop when storage is full? (Oldest? Non-local? Non-subscribed aggregates?)

### 3.2 The Snapshot-or-Replay Tradeoff

On the server, replaying 10,000 events for an aggregate is fine (fast CPU, abundant RAM). On a phone:

| Operation | Server | Phone |
|-----------|--------|-------|
| Replay 10K events | ~10ms | ~100-500ms |
| Load snapshot | ~1ms | ~1-5ms |
| Battery cost | N/A | Significant for replay |

**Implication:** Client-side snapshots must be more aggressive than server-side. The `SnapshotStrategy` interface should be parameterized by device capability.

### 3.3 Event Compaction on Client

The server keeps all events forever (audit, replay, time travel). The client cannot. Client-side event compaction must:

1. Never compact uncommitted events (haven't been acked by server)
2. Never compact events that projections haven't processed yet
3. Only compact events after a confirmed snapshot
4. Preserve the snapshot + enough events for rebase (if the server rejects the snapshot)

This is a **new concept** not in go-cqrs-lite today: the client needs a different lifecycle than the server.

---

## 4. Security & Trust

### 4.1 The Untrusted Client Problem

In server-only CQRS, the server validates commands and creates events. The client is a thin view layer. In offline-first, **the client creates events**. The server must decide: do I trust these events?

| Trust Model | How | Tradeoff |
|-------------|-----|----------|
| **Full trust** | Client creates events, server stores them verbatim | Maximum offline capability; zero server-side validation |
| **Command-only** | Client sends commands (not events), server validates and creates events | Server is authority; offline must queue commands, not events |
| **Event with validation** | Client creates events, server validates and may reject | Balanced; server can reject invalid events |
| **Event with transformation** | Client creates events, server may transform before storing | Server can fix/enrich; client must handle rebase after transformation |
| **Hybrid** | Client creates optimistic events locally, server creates canonical events on sync | Two event streams; client must reconcile |

**Implication:** This is an **architectural decision** that affects everything. go-cqrs-lite currently assumes the server creates events. The `EventSourcedRepository.Save` calls `NewEvent` server-side. If clients create events, the server needs a different ingestion path.

### 4.2 Authentication Token Expiry While Offline

```
User authenticates at 9:00 AM (JWT expires in 1 hour).
User goes offline at 9:30 AM.
User creates events offline for 3 hours.
User comes online at 12:30 PM.
JWT expired 2 hours ago.  Can't push events.
```

**Mitigations:**

- **Long-lived offline tokens** — Separate token for offline event signing (not server auth)
- **Refresh token with offline scope** — Server issues a long-lived refresh token that can be used once after reconnect
- **Event signing** — Client signs events with a device key; server validates signature, not auth token
- **Deferred auth** — Accept events without auth, queue for validation when auth is restored

### 4.3 Event Signing & Tampering

If clients create events offline, a malicious client could create events with forged metadata (fake UserID, fake timestamps, fake CorrelationID). Without cryptographic signing:

- The server can't verify event authenticity
- Other clients can't trust received events
- Audit trails are unreliable

**Implication:** Offline events should be **signed** by the client's device key. The signature covers at minimum: `ID`, `Type`, `AggregateID`, `Version`, `Payload`, `ClientID`, `ClientOccurredAt`. The server verifies the signature before accepting.

### 4.4 Encryption at Rest on Client

Events on the client device may contain sensitive data (PII, financial data, health data). On-device storage is not secure by default:

- Phone theft → attacker reads event store
- Shared device → other users read events
- Browser → localStorage accessible to XSS
- Backup extraction → events in cloud backup

**Implication:** The client event store should encrypt events at rest. The encryption key should be derived from the user's credentials, not stored on the device. This is a client-side concern, not an SDK concern, but the SDK should not assume cleartext storage.

---

## 5. Privacy & Legal

### 5.1 The Immutability vs GDPR Paradox

Event sourcing's core principle: **events are immutable**. GDPR's core requirement: **users can delete their data**. These conflict.

| Principle | Event Sourcing | GDPR / CCPA |
|-----------|---------------|-------------|
| Data retention | Forever (immutable log) | Right to erasure |
| Data modification | Never (append only) | Right to rectification |
| Data portability | Natural (export all events) | Right to portability ✅ |
| Data access | Natural (replay user's events) | Right to access ✅ |

**On the server:** You can use "right to be forgotten" events (tombstones) or crypto-shredding (encrypt per-user, delete the key).

**On the client:** If the user requests deletion, the client must purge all events containing their PII. But those events may not have synced yet. The client must:

1. Delete local events
2. Push a "forget me" command to the server
3. Server deletes or crypto-shreds all events for that user
4. Server notifies other clients to purge that user's data from their local stores

### 5.2 Cross-Border Data (Events Moving Between Jurisdictions)

An event created in the EU (on the client) is subject to GDPR. When it syncs to a server in the US, it may violate data residency requirements. When it syncs to another client in China, it may violate data export laws.

**Implication:** Events may need a `DataResidency` metadata field, and the sync protocol must respect it (don't sync certain events to certain regions).

### 5.3 Consent Tracking

If events contain PII, the user's consent at the time of creation matters. A consent withdrawal at time T doesn't retroactively make pre-T events illegal, but it does mean they must be handled differently going forward.

**Implication:** Events may need a `ConsentContext` metadata field capturing what the user consented to at creation time.

---

## 6. Multi-Device Same User

### 6.1 The Two-Phone Problem

```
User has Phone A (online) and Phone B (offline).
Phone A creates events. Server sees them.
Phone B comes online and pushes conflicting events.

The server sees TWO devices for the same user
with DIFFERENT views of the same aggregates.
```

This is different from the two-client conflict in the timing analysis because both devices belong to the *same user*. The resolution strategy may differ:

- **Merge (git-like)** — Accept both, reorder
- **Last-device-wins** — Accept the device that synced last
- **User chooses** — Show both versions, let user pick
- **Device authority** — Some devices are authoritative (laptop > phone for document editing)

**Implication:** `ClientID` must distinguish *devices*, not just *users*. Two devices for the same user need different `ClientID` values.

### 6.2 Device Handoff

User starts editing on phone, puts phone in pocket, opens laptop. Laptop doesn't have the phone's uncommitted events yet. User edits the same aggregate on laptop.

**Implication:** Near-real-time sync between same-user devices (like Apple's Handoff or Google's Fast Pair) is a separate concern from offline sync. It requires a push notification or WebSocket channel between devices, not just client-server sync.

---

## 7. Aggregate Design Under Offline

### 7.1 Aggregate Size Matters More

On the server, a 10,000-event aggregate is a performance concern. On the client, it's a **blocking concern** — replaying 10,000 events on a phone takes 100-500ms, during which the UI is frozen.

**Implication:** Offline-first systems need *smaller* aggregates than server-only systems. Aggregate boundaries should be drawn to keep streams under ~100 events on the client.

### 7.2 Cross-Aggregate Transactions Don't Exist Offline

On the server, you can wrap multiple aggregate saves in a single database transaction. On the client, there's no database (or a much simpler one). If a business operation creates events across 3 aggregates, and the client crashes after the first 2, you have partial state.

**Impigation:** Client-side operations should be **single-aggregate** where possible, or use a saga/process manager pattern where each step is independently resumable.

### 7.3 Aggregate Identity Generation

Who generates the `AggregateID`? Currently `id.New[AggregateID]()` generates a ULID. If both client and server can generate IDs:

- **Collision risk:** ULID collisions are astronomically unlikely (48-bit timestamp + 80-bit randomness), but the SDK should document that both client and server CAN generate IDs independently.
- **Identity reservation:** Client creates an aggregate offline. Server has a different aggregate with the same ID (collision, or pre-existing data). This must be detected at sync time.

---

## 8. UX Patterns for Eventual Consistency

### 8.1 The Five UI States

| State | Visual | Meaning | Event Status |
|-------|--------|---------|--------------|
| **Local** | Blue dot / spinner | Created locally, not yet attempted sync | In client outbox |
| **In-flight** | Animated spinner | Sync in progress | Being pushed |
| **Confirmed** | Green check | Server acknowledged | `SyncAckedAt` set |
| **Conflict** | Red badge | Server rejected (concurrency or validation) | In client dead letter |
| **Stale** | Gray / dimmed | Local read model is behind server | `SyncPulledAt` is old |

### 8.2 The "How Fresh Is This?" Problem

When a user views a read model, they need to know how stale it is:

```
"You're viewing data from 3 hours ago.
 2 updates are pending sync.
 Tap to sync now."
```

This requires:
- `SyncPulledAt` on the read model (last time we got server data)
- Client outbox depth (how many events are pending)
- Network status indicator

### 8.3 Optimistic Updates with Rollback

When the user takes an action, the UI updates immediately (optimistic). If the server later rejects the event, the UI must **roll back**:

1. Remove the event from local store
2. Revert the projection to the state before the event
3. Re-apply any events that came after it
4. Show the user: "Your change was rejected because [reason]"

This is equivalent to a `git rebase` that drops a commit. The client must be able to reconstruct its state at any event boundary.

### 8.4 The "Pending" Queue Visibility

Some UXes show the user their pending actions:

```
📋 Pending actions (3):
  ✓ Add item to cart        [synced]
  ⟳ Change shipping address [syncing...]
  ○ Place order             [waiting for sync]
```

This requires the client outbox to be queryable by the UI layer, with status per event.

---

## 9. Compensating Events & Undo

### 9.1 Undo Before Sync

User creates an event locally. Immediately undoes it. Neither event has synced.

```
v1: CartCreated     (offline)
v2: ItemAdded       (offline)
v3: ItemRemoved     (offline, undo of v2)
```

**Option A: Event cancellation** — Remove v2 and v3 from the outbox (they cancel out). Only push v1.
**Option B: Push all three** — Server sees the full history, including the undo.

| Approach | Pros | Cons |
|----------|------|------|
| Cancel pairs | Less bandwidth, simpler history | Projection logic must handle cancellation |
| Push all | Full audit trail, server sees true history | Wasted bandwidth, more events to process |

**Implication:** The SDK should support both strategies. `IsLocalOnly` could be repurposed for cancelled pairs, or a new `CancelledBy` field links v3 to v2.

### 9.2 Undo After Sync

User creates event, it syncs, then user undoes it. This is just a compensating event (v3 undoes v2, both on the server). Already handled by event sourcing naturally.

### 9.3 Undo During Conflict

User creates event locally (v2). Before sync, user undoes it (v3). Meanwhile, another client pushed v2 to the server. Now the server has v2 but the local client thinks v2 was undone.

```
Server:  v1 → v2 (from other client)
Local:   v1 → v2 (local, not yet pushed) → v3 (undo of local v2)
```

When the local client syncs, it discovers its v2 conflicts with the server's v2. The undo (v3) may no longer make sense because the server's v2 is for a *different event*.

**Implication:** Compensating events must reference the *specific event ID* they undo, not just the version. `CausationID` should point to the `EventID` being undone.

---

## 10. Client-Side Projections

### 10.1 Projections Must Be Re-runnable

On the server, a projection runs once per event. On the client, projections may need to re-run:

- **After rebase** — Events were reordered; projection must re-run from the rebase point
- **After undo** — An event was removed; projection must re-run from before it
- **After app update** — New projection logic; must re-run to produce updated read model
- **After snapshot restore** — State was rebuilt from snapshot + events

**Implication:** Client-side projections must be **idempotent and rerunnable**. The LiveStore determinism rule applies: no `time.Now()`, no `uuid.New()`, no side effects.

### 10.2 Cross-Event Dependencies in Projections

A client-side projection for a "shopping cart" may need data from both the `Cart` aggregate and the `Product` aggregate. If the client only has partial events (due to storage constraints), the projection may be working with incomplete data.

**Implication:** Client-side projections must handle **missing data gracefully** — the equivalent of `null` checks for events that haven't been synced yet.

### 10.3 Projection Schema on Client

The client's read model schema (e.g., a local SQLite table) must evolve alongside the app. If the projection logic changes (app update), the read model may need to be:

1. Dropped and rebuilt from events (LiveStore approach)
2. Migrated in-place (traditional DB migration)

The rebuild approach is simpler but requires all source events to be available. If the client has compacted old events, the rebuild can only start from the oldest available snapshot.

---

## 11. Testing Offline Scenarios

### 11.1 Testing Dimensions

| Dimension | States | Combinatorial Explosion |
|-----------|--------|------------------------|
| Connectivity | Online, degraded, intermittent, offline | 4 |
| Client schema version | v1, v2, ..., vN | N |
| Server schema version | v1, v2, ..., vN | N |
| Number of offline events | 0, 1, 10, 100, 1000 | 5 |
| Number of concurrent clients | 1, 2, 5, 10 | 4 |
| Conflict type | None, same-field, cross-aggregate, causal | 4 |

Total: 4 × N × N × 5 × 4 × 4 = **320 × N²** test scenarios. For N=3 schema versions: 2,880 scenarios.

### 11.2 What go-cqrs-lite Needs for Testing

1. **Network simulator** — A `Bus` and `Store` wrapper that introduces configurable latency, packet loss, and disconnection periods
2. **Multi-client test harness** — N `InMemoryRunner` instances sharing a single `MemoryStore` (simulating server), each with their own local store and outbox
3. **Schema version matrix test** — Create events in v1 schema, upcast to v2, verify projections produce the same result
4. **Conflict replay** — Record real-world conflict scenarios (two clients editing the same aggregate) and replay them deterministically
5. **Fuzz testing** — Random sequences of: go offline, create events, go online, sync, check invariants

---

## 12. Serialization & Wire Format

### 12.1 Binary vs JSON

| Format | Size | Parse Speed | Schema Evolution | Debuggability |
|--------|------|-------------|-----------------|---------------|
| JSON | Large (2-5x) | Slow | Good (extra fields ignored) | Excellent |
| Protobuf | Small | Fast | Excellent (field numbers) | Poor (binary) |
| MessagePack | Medium | Fast | Good | Moderate |
| FlatBuffers | Smallest | Zero-copy | Good | Poor |

For offline-first, **payload size matters** because it directly affects sync time and bandwidth cost. But schema evolution matters equally because clients will always be behind the server.

**Implication:** Consider offering a compact wire format for sync (Protobuf or MessagePack) alongside the current JSON. The wire format between client and server can differ from the storage format.

### 12.2 The Re-Serialization Problem

When a client rebases events, it may need to:

1. Deserialize events from local store (format A)
2. Reorder them
3. Re-serialize and push to server (format B)

If format A and format B differ, or if the serialization is lossy (e.g., JSON drops unknown fields), information is lost during re-serialization.

**Implication:** The wire format must be **lossless round-trip** — deserialize → re-serialize must produce equivalent data. JSON with `any` field for unknown keys, or a format that preserves unknown fields (Protobuf does this).

### 12.3 Event Size Limits

A single event with a 10MB payload (e.g., image attachment) on a metered connection is a problem. The system needs:

- **Per-event size limits** — Reject events above a configurable threshold
- **Blob storage offloading** — Large payloads stored separately, event contains a reference
- **Chunked sync** — For events that are large but legitimate, sync in chunks

---

## 13. Client SDK Architecture

### 13.1 What Runs on the Client?

go-cqrs-lite is a Go library. For client-side use:

| Client Platform | Go Support | Feasibility |
|----------------|-----------|-------------|
| **Backend services** | Native | ✅ Already works |
| **CLI tools** | Native | ✅ Already works |
| **Desktop apps** | Native | ✅ Wails, Fyne, etc. |
| **Mobile (iOS/Android)** | Gomobile | ⚠️ Possible but cumbersome |
| **Web browser** | WASM | ⚠️ Limited storage APIs |
| **React Native / Flutter** | Not Go | ❌ Need a TypeScript/Dart SDK |

**Implication:** If the target includes web or mobile-native clients, go-cqrs-lite can't run directly. Options:

1. **Generate client SDKs from catalog** — The `catalog` module already generates AsyncAPI specs; a code generator could produce TypeScript/Java/Swift client SDKs
2. **Thin client protocol** — Client sends raw JSON, server uses go-cqrs-lite to validate and create events
3. **Port core types** — A companion library in TypeScript with the same event/command interfaces

### 13.2 What the Client SDK Must Include

| Component | Purpose | go-cqrs-lite Equivalent |
|-----------|---------|-------------------------|
| Event creation | Typed event constructors | `NewEvent` + options |
| Local event store | Durable offline storage | `MemoryStore` (not durable) |
| Local projection runner | Build read model from events | `InMemoryRunner` |
| Local outbox | Track unsynced events | `MemoryOutboxStore` |
| Sync client | Push/pull/rebase protocol | ❌ Doesn't exist |
| Conflict resolver | Domain-specific merge logic | ❌ Doesn't exist |
| Auth token manager | Handle token refresh | ❌ Out of scope |
| Network monitor | Detect connectivity changes | ❌ Out of scope |

### 13.3 The "Thick Client vs Thin Client" Decision

| Approach | Client Creates | Server Creates | Offline Capability |
|----------|---------------|----------------|-------------------|
| **Thick client** | Events (with validation) | Confirms, enriches, stores | Full offline |
| **Thin client** | Commands (intent only) | Events (after validation) | Queue commands offline |
| **Hybrid** | Optimistic events + commands | Canonical events | Full offline with server authority |

The hybrid approach is the most common in practice: the client creates events for optimistic UI updates, but the server creates the *canonical* events. On sync, the client's optimistic events may be replaced by the server's canonical ones, requiring a UI update.

---

## 14. Operational Concerns

### 14.1 Monitoring Offline Clients

You can't monitor a device that's offline. But you need to know:

- **How many clients are offline?** (Can't know until they reconnect)
- **How stale is each client?** (Known at last reconnect)
- **How many events are pending per client?** (Known at last reconnect)
- **What's the oldest unacked event across all clients?** (Can know server-side)

**Implication:** Server-side metrics must track `ServerReceivedAt - ClientOccurredAt` for every event. This gap measures "how offline was the client." Alert on high percentiles (e.g., P99 offline duration > 24 hours).

### 14.2 The Reconnect Storm

A network outage affecting 10,000 users resolves simultaneously. All 10,000 clients reconnect and push events at once.

**Server-side mitigations:**

- **Rate limiting per client** — Accept N events per second per client
- **Batch acceptance** — Client pushes events in batches; server acknowledges per-batch
- **Queue-based ingestion** — Push events to a message queue (Kafka, NATS), process asynchronously
- **Priority-based admission** — Accept high-priority events first; defer low-priority

### 14.3 Event Replay for Bug Fixes

A bug in the client's event creation logic produced malformed events for 3 hours before the app was updated. Those events are now in 10,000 client outboxes. When users sync, the server receives malformed events.

**Mitigations:**

- **Server-side validation** — Reject malformed events, send rejection to client, client quarantines them
- **Client-side kill switch** — Remote config flag that disables the buggy event type, client drops pending events of that type
- **Grace period** — Accept malformed events for a transition period, transform them server-side, then enforce the new schema

### 14.4 Disaster Recovery

If the server's event store is lost, clients have (partial) copies of events in their local stores. This is an accidental "distributed backup." But:

- Not all events are on all clients
- Some clients are offline and unreachable
- Events on clients may be in different schema versions
- Client event order may differ from server order

**Implication:** Reconstructing the server from client stores is possible but complex. It requires a merge protocol similar to distributed database recovery. Document this as a disaster recovery option, not a primary strategy.

---

## 15. What go-cqrs-lite's Library Shape Means

### 15.1 Library, Not Framework

go-cqrs-lite is explicitly "library, not framework." It provides building blocks, not opinions. This is correct for the server-side use case. For offline-first, this means:

- **The sync protocol is the consumer's responsibility** — go-cqrs-lite provides the event/command/event-store primitives; the consumer builds the sync layer
- **The client SDK may be a separate project** — go-cqrs-lite defines the types and interfaces; a companion project (e.g., `go-offline-sync`) implements the sync protocol
- **The conflict resolver is pluggable** — go-cqrs-lite can define the `ConflictResolver` interface; consumers provide the domain-specific implementation

### 15.2 What Should Be in go-cqrs-lite

| Concern | Belongs in go-cqrs-lite? | Why |
|---------|--------------------------|-----|
| Event/command types | ✅ Yes | Core domain types, already here |
| ClientID, timezone metadata | ✅ Yes | Metadata on core types |
| IdempotencyKey on Command | ✅ Yes | Command interface change |
| Wire format for sync | ⚠️ Maybe | Could be a separate module |
| Sync protocol (push/pull/rebase) | ❌ No | Consumer's responsibility; too opinionated |
| Client-side event store | ❌ No | Platform-specific (SQLite, IndexedDB, etc.) |
| Network monitor | ❌ No | Platform-specific |
| Conflict resolver interface | ✅ Yes | `ConflictResolver[T]` interface (from `sync/` module) |
| Vector clock | ✅ Yes | `sync/` module (zero deps, planned) |
| Event signing | ❌ No | Security concern, consumer's responsibility |
| Client-side encryption | ❌ No | Platform-specific |
| Auth token management | ❌ No | Out of scope |

### 15.3 The Minimal Viable Offline Addition

If go-cqrs-lite does nothing else, these three changes unlock offline-first for consumers:

1. **Command metadata** (P0) — `CorrelationID`, `CausationID`, `UserID`, `IdempotencyKey`, `ClientCreatedAt` on commands. Enables tracing through the full lifecycle.
2. **ClientID + timezone on events** (P0) — `WithClientID`, `WithClientTimezone` options. Enables attribution and business-local time.
3. **CRDT primitives module** (P1) — Vector clock, conflict resolver in `sync/`. Gives consumers the building blocks for a sync protocol.

Everything else (sync protocol, client store, network monitor) is the consumer's responsibility, as it should be for a library.

---

## Appendix: Checklist — "Are You Ready for Offline-First?"

### Event Design

- [ ] Every event carries `ClientID` (which device created it)
- [ ] Every event carries `ClientTimezone` + `ClientUTCOffset`
- [ ] Every event carries `CausationID` (what caused it)
- [ ] Events are signed or the server validates them on ingestion
- [ ] Large payloads are offloaded to blob storage; events contain references only
- [ ] Event types are versioned (`v1.UserCreated`) or carry `SchemaVersion`

### Command Design

- [ ] Commands carry `IdempotencyKey` for dedup on retry
- [ ] Commands carry `CorrelationID` for distributed tracing
- [ ] Commands carry `ExpectedVersion` for optimistic concurrency
- [ ] Command handlers are idempotent (same command twice = same result)

### Aggregate Design

- [ ] Aggregates are small enough for client-side replay (<100 events)
- [ ] Cross-aggregate operations are modeled as sagas, not transactions
- [ ] AggregateIDs are generated client-side (ULID) to avoid server round-trips

### Projection Design

- [ ] Projections are deterministic (no `time.Now()`, no random IDs)
- [ ] Projections are idempotent (re-running produces the same state)
- [ ] Projections handle missing data gracefully (events from unsubscribed aggregates)
- [ ] Day-grouping uses a documented timezone strategy (UTC, server, or client)

### Sync Protocol

- [ ] Sync is pull-before-push (client pulls server state before pushing)
- [ ] Sync is idempotent (pushing the same events twice is safe)
- [ ] Sync handles partial completion (connection drops mid-batch)
- [ ] Sync has a priority mechanism (high-priority events first)
- [ ] Sync handles schema version mismatch (client v2, server v5)
- [ ] Sync has a backpressure mechanism (rate-limit on reconnect storm)

### UX

- [ ] UI shows pending/confirmed/conflict state for each action
- [ ] UI shows how stale the read model is ("data from 3 hours ago")
- [ ] UI handles optimistic update rollback (server rejects event)
- [ ] UI handles force-update (server no longer supports client's schema version)

### Security

- [ ] Client auth tokens are refreshed on reconnect after offline period
- [ ] Events are signed by device key (or server validates on ingestion)
- [ ] Client event store is encrypted at rest
- [ ] PII events are tracked for GDPR deletion

### Operational

- [ ] Server logs `ServerReceivedAt - ClientOccurredAt` for every event
- [ ] Server alerts on high offline-duration percentiles
- [ ] Server has rate limiting per client for reconnect storms
- [ ] Server has a grace period for old schema versions
- [ ] Disaster recovery plan considers client-side event stores as partial backup

---

_Companion to `docs/planning/2026-05-01_OFFLINE_FIRST_TIMING_ANALYSIS.md`. That document covers *when*; this covers *everything else*._
