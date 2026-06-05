# Storage Environment Mapping

> **Date:** 2026-06-05 | **Scope:** All modules that persist or may persist data
>
> **Purpose:** Identify every storage touchpoint in go-cqrs-lite and map each to its
> **native backend per runtime environment**. "Native" means: already present, no extra
> infrastructure to provision, idiomatic for that platform.

---

## 1. Storage Touchpoint Inventory

These are the only places in go-cqrs-lite that actually store bytes on disk or over the network.

| # | Touchpoint | Interface | Status | Data Shape | Write Pattern | Read Pattern | Size Growth |
|---|-----------|-----------|--------|-----------|---------------|--------------|-------------|
| 1 | **Event Store** | `event.Store` (Sink + Source + Journal + Seekable) | ✅ Implemented | Append-only log of immutable events | Heavy append, no update/delete | By aggregate, by time range, global journal scan | Unbounded — primary data store |
| 2 | **Snapshot Store** | `snapshot.SnapshotStore` | ✅ Implemented | Serialized aggregate state at version | Occasional upsert (every N events) | Point lookup by aggregate | Bounded — one per aggregate |
| 3 | **Checkpoint Store** | `event.CheckpointStore` | ✅ Implemented | Projection name → last event ID | Frequent overwrite (per batch) | Point lookup by projection name | Bounded — one per projection |
| 4 | **Command Store** | `command.Store` | ⚠️ Interface only | Persisted command audit log | Append-only | By aggregate, by timestamp | Unbounded — audit trail |
| 5 | **Read Model Store** | _(no interface yet — examples only)_ | 📝 Planned | Projected queryable state (Get/List/Put/Delete) | Write on projection, read-heavy | Point lookup, filtered list | Bounded — rebuildable from events |
| 6 | **Aggregate Listing** | `listing.AggregateReader` | ✅ Implemented | Derived index of aggregates + tombstone status | Read-only (derived from events) | Cursor-pagination list | Derived — no independent storage |
| 7 | **Message Bus** | `event.Bus` / `event.Publisher` | ✅ In-memory only | Transient message delivery | Publish | Subscribe | None — in-memory only |

`watermill/` provides protocol adapters ( PublisherAdapter / SubscriberAdapter ) to bridge with the Watermill ecosystem, but the underlying bus is still `memory.MemoryBus`. No persistent backends (NATS, Redis, SQS, Pub/Sub) exist.

**Not storage modules:** `catalog/`, `schema/`, `signing/`, `middleware/`, `otel/`, `dispatcher/`, `query/`, `decider/`, `id/`, `codec/` — these are computation, transformation, or type modules. They do not persist data.

---

## 2. Environment Detection Signals

| Environment | Primary Signal | Secondary Signals | Confidence |
|-------------|---------------|-------------------|------------|
| **Kubernetes** | `KUBERNETES_SERVICE_HOST` env var | `/var/run/secrets/kubernetes.io/` exists | High |
| **AWS Lambda** | `AWS_LAMBDA_FUNCTION_NAME` env var | `AWS_REGION` set, `/var/runtime/` exists | High |
| **GCP Cloud Run** | `K_SERVICE` env var | `GOOGLE_CLOUD_PROJECT` set | High |
| **Azure Functions** | `FUNCTIONS_WORKER_RUNTIME` env var | `WEBSITE_SITE_NAME` set | High |
| **GitHub Actions / CI** | `GITHUB_ACTIONS=true` | `CI=true` | High |
| **Local Dev** | None of the above | `HOME` set, interactive TTY | Low (default fallback) |

---

## 3. Native Backend Matrix

| Storage Need | Local Dev | Kubernetes | AWS Lambda | GCP Cloud Run | Azure Functions | CI/Tests |
|-------------|-----------|------------|------------|---------------|-----------------|----------|
| **Event Store** | SQLite / Pebble | PostgreSQL¹ | Aurora / RDS² | Cloud SQL (PG) | Azure SQL / Cosmos³ | memory |
| **Snapshot Store** | SQLite / Pebble | **etcd**⁴ | DynamoDB | Firestore | Cosmos DB | memory |
| **Checkpoint Store** | SQLite / Pebble | **etcd**⁴ | DynamoDB | Firestore | Cosmos DB | memory |
| **Command Store** | SQLite / Pebble | PostgreSQL¹ | DynamoDB | Firestore | Cosmos DB | memory |
| **Read Model Store** | SQLite / Pebble | PostgreSQL¹ | DynamoDB | Firestore | Cosmos DB | memory |
| **Aggregate Listing** | SQLite (SQL reader) | PostgreSQL (SQL reader) | DynamoDB (GSI scan) | Firestore (query) | Cosmos DB (query) | memory (journal scan) |
| **Message Bus** | memory | NATS / Redis⁵ | SNS/SQS | Pub/Sub | Event Grid / SB | memory |

¹ **K8s PostgreSQL nuance:** If the cluster already runs a PostgreSQL StatefulSet or uses a managed cloud PG (Cloud SQL, RDS, Azure Database), use that. Do NOT provision PG just for CQRS.

² **AWS Lambda Event Store nuance:** DynamoDB is the "native" serverless DB, but it is a **poor event store** (see §5.1). Aurora Serverless v2 or RDS Proxy + PostgreSQL is the pragmatic choice for the event store. Use DynamoDB for snapshots, checkpoints, and read models.

³ **Azure Event Store nuance:** Cosmos DB with transactional batch can work for small event stores, but Azure SQL or PostgreSQL on Azure is more robust for the append-only log.

⁴ **etcd is the K8s superpower:** Already running in-cluster. Perfect for small KV (snapshots, checkpoints). **Never use etcd for the event store** — it has a 8MB value limit and is not designed for append-only logs.

⁵ **K8s Bus nuance:** NATS JetStream (deployed via Helm) or Redis Streams are common. The `memory.MemoryBus` only works single-process.

---

## 4. Per-Storage-Need Deep Dive

### 4.1 Event Store

**Characteristics:** Append-only, immutable, time-ordered, unbounded growth, needs strong consistency on aggregate version check.

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite | Excellent. Zero ops, WAL mode, file-based. Pebble also excellent for write-heavy workloads. |
| K8s | PostgreSQL | Excellent. Use existing PG if available. ScyllaDB is even better for write-heavy but requires provisioning. |
| AWS Lambda | Aurora Serverless PG | Good. DynamoDB is native but **journal reads (ReadAll, ReadFrom) require full table scans = expensive**. |
| GCP Cloud Run | Cloud SQL (PG) | Good. Firestore has no cross-document ordering guarantee — event journal is painful. |
| Azure Functions | Azure SQL / Cosmos | Cosmos with sort key on `occurred_at` works for small scale. Azure SQL is safer. |

**Never use for event store:** etcd (8MB limit, not a log), Redis (not durable by default), DynamoDB at high throughput (scan costs).

### 4.2 Snapshot Store

**Characteristics:** Point lookup by aggregate, occasional upsert, bounded size (one per aggregate), moderate value size (serialized state).

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite / Pebble | Excellent. Single-row upsert. |
| K8s | **etcd** | Excellent. Already running. Aggregate count × snapshot size fits easily in etcd. Use with TTL if desired. |
| AWS Lambda | DynamoDB | Excellent. Single-row Get/Put. Cheap. |
| GCP Cloud Run | Firestore | Excellent. Document get/set. |
| Azure Functions | Cosmos DB | Excellent. Point reads are cheap and fast. |

### 4.3 Checkpoint Store

**Characteristics:** Tiny (projection name → event ID), very frequent overwrites, needs durability (projection restart must resume correctly).

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite / Pebble | Excellent. Single-row table. |
| K8s | **etcd** | Excellent. Already running. Perfect KV semantics. |
| AWS Lambda | DynamoDB | Excellent. Single-item Put/Get. Extremely cheap. |
| GCP Cloud Run | Firestore | Excellent. Single-document write. |
| Azure Functions | Cosmos DB | Excellent. Point operations. |

### 4.4 Command Store

**Characteristics:** Append-only audit log, similar to event store but smaller volume. Interface exists (`command.Store`) but **no implementations exist yet**.

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite / Pebble | Excellent. Same schema as events, simpler (no journal needed). |
| K8s | PostgreSQL | Good. Can share events database, separate table. |
| AWS Lambda | DynamoDB | Good. Can use same table as snapshots with different prefix. |
| GCP Cloud Run | Firestore | Good. Can share DB with snapshots. |
| Azure Functions | Cosmos DB | Good. Can share DB with snapshots. |

### 4.5 Read Model Store

**Characteristics:** Projected state for queries. Get by ID, filtered List, Put, Delete. Rebuildable from events (can be ephemeral).

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite / Pebble | Excellent. Flexible queries on SQLite; fast point ops on Pebble. |
| K8s | PostgreSQL / Redis | PG for complex queries. Redis for pure KV read models (cache layer). |
| AWS Lambda | DynamoDB | Excellent. Flexible with GSIs. Pay-per-request is cheap for read-heavy. |
| GCP Cloud Run | Firestore | Excellent. Native querying, auto-scaling. |
| Azure Functions | Cosmos DB | Excellent. SQL API for flexible queries. |

### 4.6 Aggregate Listing

**Characteristics:** Derived view — NOT independent storage. In SQL, it's a query over the events table. In-memory, it's a cache rebuilt from `Journal.ReadAll`.

**Best backends by environment:**

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | SQLite (query) | Excellent. Indexed query over events table. |
| K8s | PostgreSQL (query) | Excellent. `SQLAggregateReader` with proper indexes. |
| AWS Lambda | DynamoDB (GSI) | **Poor.** DynamoDB GSIs on event tables are expensive. Prefer maintaining a separate "aggregates" table as read model. |
| GCP Cloud Run | Firestore (collection) | Moderate. Better to maintain a read-model collection. |
| Azure Functions | Cosmos DB (query) | Moderate. Better to maintain a read-model container. |

**Key insight:** Aggregate listing on non-SQL backends should be treated as a **read model**, not a derived query. Maintain a separate aggregate index table that projections update.

### 4.7 Message Bus

**Characteristics:** Transient delivery. go-cqrs-lite only provides in-memory bus. For multi-process or multi-node, external broker required.

| Environment | Native | Honest Assessment |
|-------------|--------|-----------------|
| Local | memory | Excellent. Single process only. |
| K8s | NATS JetStream / Redis Streams | NATS is lightweight, K8s-native via Helm. Redis if already deployed. |
| AWS Lambda | SNS / SQS / EventBridge | SNS for fan-out, SQS for queue, EventBridge for routing. |
| GCP Cloud Run | Pub/Sub | Native. Serverless push subscriptions. |
| Azure Functions | Event Grid / Service Bus | Event Grid for fan-out, Service Bus for queues. |

---

## 5. Backend Capability Summary

| Backend | Event Store | Snapshot | Checkpoint | Read Model | Bus | K8s Native | Serverless | Embedded |
|---------|:-----------:|:--------:|:----------:|:----------:|:---:|:----------:|:----------:|:--------:|
| **SQLite** | ✅ Excellent | ✅ | ✅ | ✅ | ❌ | ⚠️ PV needed | ❌ | ✅ |
| **PostgreSQL** | ✅ Excellent | ✅ | ✅ | ✅ | ❌ | ❌ Deploy | ❌ | ❌ |
| **PebbleDB** | ✅ Excellent | ✅ | ✅ | ✅ | ❌ | ⚠️ PV needed | ❌ | ✅ |
| **etcd** | ❌ Never | ✅ Perfect | ✅ Perfect | ⚠️ Small only | ❌ | ✅ Already there | ❌ | ❌ |
| **DynamoDB** | ⚠️ Poor scan | ✅ Perfect | ✅ Perfect | ✅ Excellent | ❌ | ❌ | ✅ AWS | ❌ |
| **Firestore** | ⚠️ No ordering | ✅ Perfect | ✅ Perfect | ✅ Excellent | ❌ | ❌ | ✅ GCP | ❌ |
| **Cosmos DB** | ⚠️ Small only | ✅ Perfect | ✅ Perfect | ✅ Excellent | ❌ | ❌ | ✅ Azure | ❌ |
| **ScyllaDB** | ✅ Excellent | ✅ | ✅ | ⚠️ GSI limited | ❌ | ❌ | ❌ | ❌ |
| **NATS JetStream** | ⚠️ Log ok | ❌ | ❌ | ❌ | ✅ Excellent | ✅ Helm | ❌ | ❌ |
| **Redis** | ❌ Not durable | ✅ Cache | ✅ Cache | ✅ Cache | ✅ Excellent | ✅ Helm | ❌ | ❌ |
| **memory** | ✅ Tests | ✅ Tests | ✅ Tests | ✅ Tests | ✅ Tests | ❌ | ❌ | ✅ |

---

## 6. Honest Limitations: When "Native" ≠ "Best"

### 6.1 DynamoDB for Event Store

DynamoDB is the native AWS serverless database, but it is a **bad event store** for two reasons:

1. **No native ordering:** `Journal.ReadAll()` requires a full table scan (`Scan` operation) or a global secondary index scan. At scale, this is prohibitively expensive.
2. **No cross-partition ordering:** Events for different aggregates are in different partitions. Time-ordering across all aggregates cannot be guaranteed without application-level sorting.

**Verdict:** Use DynamoDB for snapshots, checkpoints, and read models. Use Aurora Serverless PostgreSQL for the event store on AWS.

### 6.2 Firestore for Event Store

Firestore has no guaranteed ordering across document collections. Event journal reads would require querying all documents and sorting in-memory. Document size limits (1MB) also constrain large event batches.

**Verdict:** Same as DynamoDB — use for snapshots/checkpoints/read models, use Cloud SQL (PG) for event store.

### 6.3 etcd for Event Store

etcd has a hard 8MB value size limit and a soft 1.5MB recommended limit. It is optimized for configuration, coordination, and small KV — not for unbounded append-only logs. The etcd WAL is an internal implementation detail, not a consumer-facing storage API.

**Verdict:** Use etcd for checkpoints and snapshots only. Never for events.

### 6.4 Redis for Anything Durable

Redis is an in-memory data structure store. Without AOF + RDB persistence configured, data is lost on restart. Even with persistence, Redis is not designed for the consistency guarantees required by event sourcing (optimistic concurrency, strict ordering).

**Verdict:** Use Redis as a read-model cache or bus only. Never as the source of truth for events or checkpoints.

### 6.5 Pebble / SQLite on Kubernetes

Both are embedded, single-writer databases. Running them in Kubernetes requires a `ReadWriteOnce` PersistentVolume, which ties the pod to a specific node. This prevents horizontal scaling and creates a single point of failure.

**Verdict:** Pebble and SQLite are excellent for local dev and sidecar patterns. Use PostgreSQL or etcd for shared storage in K8s.

---

## 7. Module Gaps & Implementation Status

| Module | Event Store | Snapshot | Checkpoint | Command Store | Read Model | Listing | Bus |
|--------|:-----------:|:--------:|:----------:|:-------------:|:----------:|:-------:|:---:|
| `memory/` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ |
| `storage/` (SQL) | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| `pebble/` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `turso/` | ✅ (via SQL) | ✅ (via SQL) | ✅ (via SQL) | ❌ | ❌ | ❌ | ❌ |
| `listing/` | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (memory) | ❌ |
| `watermill/` | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (adapter) |

**Missing implementations (opportunities):**

- `pebble.SnapshotStore` — trivial to add (single key per aggregate)
- `pebble.CheckpointStore` — trivial to add (single key per projection)
- `command.Store` implementations — no implementations exist anywhere
- `readmodel/` module — only exists in examples (`example/todo/storage/`)
- `storage.AggregateProjection` — hardcoded `?` placeholders and SQLite `ON CONFLICT` syntax; not dialect-aware
- Persistent bus implementations (NATS, Redis, SQS, Pub/Sub adapters)

---

## 8. Decision Tree

```
What environment are you in?
├── KUBERNETES_SERVICE_HOST set?
│   ├── Already have PostgreSQL in-cluster or managed?
│   │   └── Event Store → PG | Snapshot → etcd | Checkpoint → etcd | Read Model → PG
│   └── No PostgreSQL?
│       └── Event Store → SQLite (sidecar with PV) or provision PG
│           Snapshot → etcd | Checkpoint → etcd | Read Model → SQLite or etcd
├── AWS_LAMBDA_FUNCTION_NAME set?
│   └── Event Store → Aurora Serverless PG
│       Snapshot → DynamoDB | Checkpoint → DynamoDB | Read Model → DynamoDB
├── K_SERVICE set (GCP)?
│   └── Event Store → Cloud SQL (PG)
│       Snapshot → Firestore | Checkpoint → Firestore | Read Model → Firestore
├── FUNCTIONS_WORKER_RUNTIME set (Azure)?
│   └── Event Store → Azure SQL or PostgreSQL on Azure
│       Snapshot → Cosmos DB | Checkpoint → Cosmos DB | Read Model → Cosmos DB
├── GITHUB_ACTIONS=true or CI=true?
│   └── Everything → memory (fast, ephemeral, no cleanup)
└── None of the above (local dev)
    └── Everything → SQLite (default) or Pebble (write-heavy)
```

---

## 9. Key Takeaways

1. **etcd is Kubernetes' hidden superpower.** Already running, consistent, distributed. Use it for snapshots and checkpoints. Never for events.

2. **Serverless databases (DynamoDB, Firestore, Cosmos DB) are poor event stores** but excellent for everything else. Always pair with SQL for the event log.

3. **Command store has zero implementations.** The interface exists but no backend implements it. This is a real gap.

4. **Read model store has no formal module.** Every consumer reinvents it (`example/todo/storage/`, `example/user/projection.go`). A `readmodel/` module would be the highest-impact addition.

5. **Pebble is incomplete.** Only event store is implemented. Snapshots and checkpoints are trivial to add and would make Pebble a complete embedded solution.

6. **Aggregate listing on non-SQL backends should be a read model**, not a derived query. The `listing.SQLAggregateReader` approach doesn't translate to DynamoDB/Firestore efficiently.

7. **There is no persistent bus.** `memory.MemoryBus` is single-process only. NATS, Redis, SQS, Pub/Sub adapters would unlock multi-node deployments.
