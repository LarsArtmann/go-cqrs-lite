# Database Architecture Taxonomy

> **Date:** 2026-05-29 | **Scope:** All known database architectures from interface and storage perspectives

---

## How to Read This Document

Two axes of classification:

1. **Interface Model** — What operations the database exposes to consumers (the API shape)
2. **Storage Engine** — How data is physically laid out on disk/in memory (the data structure)

A single interface model can be backed by many storage engines. A single storage engine can power many interface models.

---

## Part 1: Interface Models (The API Shape)

### 1. Key-Value

```
Get(key) → value
Put(key, value)
Delete(key)
```

| Dimension | Detail |
|---|---|
| **Operations** | Get, Put, Delete, Exists, Increment |
| **Query model** | Exact key match only. No range, no filtering, no joins |
| **Ordering** | None (hash) or lexicographic (ordered KV like Pebble/LevelDB) |
| **Schema** | None — opaque byte values |
| **Transactions** | Optional — single-key atomic or multi-key batch |
| **Notable** | Redis, etcd, DynamoDB, LevelDB, RocksDB, Pebble, Consul |

**Variants:**
- **Hash KV** — O(1) point lookup, no range scan (Redis hash, Memcached)
- **Ordered KV** — O(log n) lookup, range scan via iterator (Pebble, LevelDB, RocksDB, BadgerDB)
- **Distributed KV** — Consensus-based, replicated (etcd, TiKV, FoundationDB)

---

### 2. Document

```
Insert(collection, document)
Find(collection, query) → []Document
Update(collection, query, changes)
Delete(collection, query)
Aggregate(collection, pipeline) → []Document
```

| Dimension | Detail |
|---|---|
| **Operations** | CRUD + rich query + aggregation pipeline |
| **Query model** | Field matching, nested path queries, range, regex, geospatial |
| **Ordering** | By indexed fields, by insertion order |
| **Schema** | Optional / flexible — documents can vary within a collection |
| **Transactions** | Optional — MongoDB supports multi-document ACID |
| **Notable** | MongoDB, CouchDB, Couchbase, Firestore, RethinkDB |

**Key property:** A document is a self-contained unit (JSON/BSON). Queries address fields within documents. Indexes are on fields, not the whole document.

---

### 3. Relational (SQL)

```
SELECT columns FROM table WHERE condition JOIN table2 ON ... GROUP BY ... ORDER BY ...
INSERT INTO table (columns) VALUES (...)
UPDATE table SET ... WHERE ...
DELETE FROM table WHERE ...
BEGIN / COMMIT / ROLLBACK
```

| Dimension | Detail |
|---|---|
| **Operations** | Full relational algebra: select, project, join, aggregate, sort |
| **Query model** | Declarative (SQL). Optimizer chooses execution plan |
| **Ordering** | By any indexed column, by expression |
| **Schema** | Strict — typed columns, constraints, foreign keys, normalization |
| **Transactions** | Full ACID — BEGIN/COMMIT/ROLLBACK, isolation levels |
| **Notable** | PostgreSQL, MySQL, SQLite, Oracle, SQL Server, CockroachDB |

**Key property:** Data is normalized across tables. Joins reconstruct relationships. The optimizer is the magic — consumers say WHAT they want, not HOW to get it.

---

### 4. Graph

```
CreateNode(label, properties) → Node
CreateEdge(type, source, target, properties) → Edge
Traverse(start, pattern) → []Path
Query(cypher) → Result  // Cypher, Gremlin, SPARQL
ShortestPath(a, b) → Path
```

| Dimension | Detail |
|---|---|
| **Operations** | Node/edge CRUD + traversal + pattern matching + path algorithms |
| **Query model** | Graph traversal (Cypher, Gremlin, GQL). Pattern matching on topology |
| **Ordering** | By traversal order, by property index |
| **Schema** | Optional — labels/types with optional property constraints |
| **Transactions** | Varies — Neo4j supports ACID, others are eventual |
| **Notable** | Neo4j, Neptune, JanusGraph, ArangoDB (multi-model), TigerGraph |

**Key property:** Relationships are first-class citizens, not reconstructed via joins. Multi-hop traversal is O(k) where k = path length, not O(n²) like SQL joins.

---

### 5. Time-Series

```
Write(metric, tags, value, timestamp)
Query(metric, tags, timeRange) → []Point
Aggregate(metric, tags, window, fn) → []Bucket
Downsample(metric, targetResolution)
```

| Dimension | Detail |
|---|---|
| **Operations** | Write (high-throughput ingest), Query (time range + aggregation) |
| **Query model** | Time-bounded, tag-filtered, with windowed aggregation functions |
| **Ordering** | Strictly by timestamp |
| **Schema** | Metric name + tag set + field set. Often schemaless or lightweight |
| **Transactions** | Rarely supported — append-only semantics |
| **Notable** | InfluxDB, Prometheus, TimescaleDB, TDengine, QuestDB, ClickHouse |

**Key property:** Optimized for append-heavy time-ordered data. Reads are almost always range scans over time windows. Compression exploits temporal locality (delta encoding, run-length).

---

### 6. Search / Full-Text

```
Index(document, fields)
Search(query, filters) → []Hit (ranked)
Suggest(prefix) → []Suggestion
Aggregate(query, facets) → FacetResults
```

| Dimension | Detail |
|---|---|
| **Operations** | Index, Search (with relevance scoring), Facet, Autocomplete |
| **Query model** | Full-text query DSL (boolean, phrase, fuzzy, proximity) |
| **Ordering** | By relevance score (TF-IDF, BM25), by field, by geo distance |
| **Schema** | Mapping definition (field types, analyzers) |
| **Transactions** | Rarely — eventually consistent indexing |
| **Notable** | Elasticsearch, Solr, OpenSearch, Meilisearch, Typesense |

**Key property:** Inverted index maps terms → documents. Relevance scoring is the differentiator. Not a primary store — usually sits alongside a source-of-truth database.

---

### 7. Wide-Column / Column-Family

```
Put(rowKey, columnFamily, column, value)
Get(rowKey, columnFamily, column) → value
Scan(rowKeyRange, columnFamily, columns) → []Row
```

| Dimension | Detail |
|---|---|
| **Operations** | Put, Get, Scan (range over row keys), Multi-Get |
| **Query model** | Row-key based + column-family/column projection. Secondary indexes optional |
| **Ordering** | By row key (token-range partitioned in distributed systems) |
| **Schema** | Column families defined at creation; columns within a family are flexible |
| **Transactions** | Lightweight (LWT in Cassandra) or none (eventual consistency) |
| **Notable** | Cassandra, ScyllaDB, HBase, Bigtable, DynamoDB |

**Key property:** A two-level map: `RowKey → (ColumnFamily → (Column → Value))`. Each row can have different columns. Optimized for write-heavy, wide-row access patterns.

---

### 8. Object Storage

```
PUT(bucket, key, data, metadata)
GET(bucket, key) → data + metadata
DELETE(bucket, key)
LIST(bucket, prefix) → []ObjectInfo
```

| Dimension | Detail |
|---|---|
| **Operations** | PUT, GET, DELETE, LIST, COPY. Versioning. Lifecycle rules |
| **Query model** | Prefix-based listing only. No content query. Metadata filtering via tags |
| **Ordering** | Lexicographic by key within a bucket |
| **Schema** | None — opaque binary blobs with user-defined metadata |
| **Transactions** | None — single-object atomicity only |
| **Notable** | S3, GCS, Azure Blob, MinIO, Cloudflare R2 |

**Key property:** Infinite scale, eventual consistency, very high latency (100ms+). Not a database — a content-addressable file system. Used as a storage tier in many database architectures (S3 as cold storage).

---

### 9. Triple Store / RDF / SPARQL

```
Assert(subject, predicate, object)         // Add triple
Retract(subject, predicate, object)        // Remove triple
Query(sparql) → Result                     // Pattern matching
```

| Dimension | Detail |
|---|---|
| **Operations** | Assert/Retract triples, SPARQL queries, ontology reasoning |
| **Query model** | Graph pattern matching via SPARQL. Logical inference over ontologies |
| **Ordering** | By subject, predicate, or object (multiple indexes) |
| **Schema** | OWL ontologies / RDFS — formal semantic schema |
| **Transactions** | Limited — quad-store support |
| **Notable** | Virtuoso, Stardog, AllegroGraph, Jena, Neptune (RDF mode) |

**Key property:** Every fact is a (subject, predicate, object) triple. Reasoning engine can derive new facts from rules. Used in knowledge graphs and semantic web.

---

### 10. Datalog

```
Assert(fact)           // Add ground fact
Rule(head ← body)      // Define derivation rule
Query(goal) → Result   // Derive answers via rules
```

| Dimension | Detail |
|---|---|
| **Operations** | Fact assertion, rule definition, recursive query |
| **Query model** | Declarative logic programming. Recursive rules. Bottom-up or top-down evaluation |
| **Ordering** | None intrinsic — results are sets |
| **Schema** | Implicit — defined by fact structure and rules |
| **Transactions** | Varies — Datomic provides ACID |
| **Notable** | Datomic, Datahike, XTDB, Soufflé |

**Key property:** Queries are logical deductions. Rules can be recursive (unlike SQL CTEs, recursion is native). Datomic's "database as a value" — every query runs against an immutable snapshot. Natural fit for event sourcing (facts are events, rules are projections).

---

### 11. Streaming

```
Publish(topic, message)
Subscribe(topic, handler)
Process(stream → stream)     // Transform
Window(stream, duration) → []Batch
Join(streamA, streamB) → streamC
```

| Dimension | Detail |
|---|---|
| **Operations** | Publish, Subscribe, Transform, Window, Join, Aggregate |
| **Query model** | Continuous queries over unbounded streams. Event-time processing |
| **Ordering** | By offset/partition, by event time, by processing time |
| **Schema** | Schema registry (Avro, Protobuf, JSON Schema) |
| **Transactions** | Exactly-once semantics via transactional producers |
| **Notable** | Kafka, Flink, Pulsar, NATS JetStream, Redpanda |

**Key property:** Data flows continuously. No "load then query" — queries are always running. Used as the backbone of event-driven architectures. Kafka is the database (log) and the transport.

---

### 12. Vector / Similarity

```
Insert(id, vector, metadata)
Search(queryVector, k) → []Neighbor   // k-nearest neighbors
Query(filter) → []Vector
```

| Dimension | Detail |
|---|---|
| **Operations** | Insert, Search (k-NN), Filter, Hybrid search (vector + keyword) |
| **Query model** | Approximate nearest neighbor (ANN) search. Cosine, L2, inner-product distance |
| **Ordering** | By similarity score (distance metric) |
| **Schema** | Vector dimension + metadata schema |
| **Transactions** | Rarely — eventually consistent indexing |
| **Notable** | Pinecone, Weaviate, Qdrant, Milvus, Chroma, pgvector |

**Key property:** Queries find "things like this" rather than "things matching this exact key." HNSW and IVF are the dominant index structures. The interface is fundamentally different from every other model — similarity replaces equality.

---

### 13. Multi-Model

```
SQL("SELECT ...")              // Relational
Cypher("MATCH ...")            // Graph
KV.Get(key)                    // Key-Value
Search(query)                  // Full-text
Vector.Search(embedding, k)    // Similarity
```

| Dimension | Detail |
|---|---|
| **Operations** | Union of multiple interface models |
| **Query model** | Unified query language (SurrealQL, AQL) or polyglot |
| **Notable** | ArangoDB, SurrealDB, Fauna, Oracle, PostgreSQL (with extensions) |

**Key property:** One database, multiple access patterns. Reduces operational complexity but may not excel at any single model.

---

## Part 2: Storage Engines (The Physical Layout)

### 1. B-Tree / B+Tree

```
           [Internal Node: Keys + Child Pointers]
          /              |                \
  [Leaf: K,V,K,V]  [Leaf: K,V,K,V]  [Leaf: K,V,K,V]
       ↔ linked list for range scans ↔
```

| Dimension | Detail |
|---|---|
| **Structure** | Balanced tree. Internal nodes: keys only. Leaves: key+value, linked for sequential scan |
| **Read** | O(log n) point lookup. O(k) range scan via leaf links |
| **Write** | O(log n) + possible node split/merge. Random I/O |
| **Write amplification** | Medium — page splits touch O(log n) pages |
| **Read amplification** | Low — single tree traversal |
| **Space amplification** | Medium — ~50-70% page fill factor |
| **Best for** | Mixed read/write OLTP. Point lookups + range scans |
| **Used by** | PostgreSQL (default), MySQL InnoDB, SQLite, LMDB, bbolt |
| **Concurrency** | Page-level locking or latch-coupling. MVCC for readers |

**B-Tree vs B+Tree:** B-Tree stores values in internal nodes (shorter tree, but larger nodes). B+Tree stores values only in leaves (denser internal nodes = shallower tree + leaf links for range scan). Most production databases use B+Tree.

---

### 2. LSM-Tree (Log-Structured Merge Tree)

```
Writes → [MemTable (sorted, in-memory)]
              ↓ flush when full
         [SSTable L0] → [SSTable L1] → [SSTable L2] → ...
              ↑ background compaction merges levels ↑
[WAL] → sequential log for crash recovery
```

| Dimension | Detail |
|---|---|
| **Structure** | In-memory sorted table (memtable) + immutable sorted files on disk (SSTables) in levels |
| **Read** | O(log n × L) where L = number of levels. Bloom filters skip empty SSTables |
| **Write** | O(1) amortized — sequential append to WAL + memtable insert |
| **Write amplification** | High — data rewritten during compaction (10-30× for leveled) |
| **Read amplification** | Medium-High — must check multiple SSTables |
| **Space amplification** | Medium — temporary duplication during compaction |
| **Best for** | Write-heavy workloads. Time-series. Event logs. Large datasets |
| **Used by** | RocksDB, LevelDB, Pebble, Cassandra, ScyllaDB, InfluxDB TSM |

**Compaction strategies:**

| Strategy | Write Amp | Read Amp | Space Amp | Best For |
|---|---|---|---|---|
| **Size-tiered** | Low | High | High | Write-heavy, bulk ingest |
| **Leveled** | High | Low | Low | Read-heavy, predictable latency |
| **FIFO** | Lowest | Medium | Lowest | Time-series, cache, TTL data |
| **Hybrid (Tiered+Leveled)** | Medium | Medium | Medium | Mixed workloads |

---

### 3. Hash Index

```
hash("user:123") → bucket 7 → [offset → disk position]
                                      ↓
                              [Log file: key, value, ...]
```

| Dimension | Detail |
|---|---|
| **Structure** | In-memory hash map: key → disk offset. Data stored in append-only log |
| **Read** | O(1) — hash lookup + single disk seek |
| **Write** | O(1) — append to log + update hash map |
| **Write amplification** | Very low — sequential append |
| **Read amplification** | Very low — single lookup |
| **Space amplification** | High without compaction — stale values accumulate in log |
| **Range scan** | ❌ Not possible — hash destroys ordering |
| **Best for** | Pure key-value with no range queries. Caching. Session storage |
| **Used by** | Bitcask (Riak), In-memory stores (Redis hash tables) |

---

### 4. Columnar Storage

```
Row-oriented:  [id=1,name="Alice",age=30] [id=2,name="Bob",age=25] ...
Columnar:      id: [1, 2, 3, ...]
               name: ["Alice", "Bob", ...]
               age: [30, 25, ...]
```

| Dimension | Detail |
|---|---|
| **Structure** | Each column stored separately. Values compressed per-column (RLE, dictionary, delta) |
| **Read** | Only needed columns loaded. Vectorized processing (SIMD). Very fast aggregates |
| **Write** | Slow — must update multiple column files. Batch writes mitigate this |
| **Write amplification** | High for single-row writes. Low for bulk loads |
| **Read amplification** | Very low for analytical queries — only relevant columns touched |
| **Space amplification** | Very low — per-column compression is extremely effective |
| **Best for** | OLAP. Analytics. Aggregation over many rows, few columns |
| **Used by** | ClickHouse, DuckDB, Vertica, Parquet files, Redshift, Snowflake |

---

### 5. Append-Only Log / Event Store

```
[Event 1] → [Event 2] → [Event 3] → ... → [Event N]
  ↑ immutable, never modified, only appended ↑
```

| Dimension | Detail |
|---|---|
| **Structure** | Sequential, immutable records. Each record has a position/offset |
| **Read** | O(n) scan, O(log n) with index. Can replay from any position |
| **Write** | O(1) — append to end. Fastest possible write pattern |
| **Write amplification** | Zero — no in-place updates, no compaction (unless space reclamation) |
| **Read amplification** | High for point lookup without secondary index |
| **Space amplification** | Grows unbounded without compaction. Compaction is optional |
| **Best for** | Event sourcing. Audit logs. CDC. Message queues. Kafka topics |
| **Used by** | Kafka, EventStoreDB, go-cqrs-lite `events` table, Datomic transaction log |

**This is our model.** go-cqrs-lite's event store is an append-only log per aggregate, with version as the sequence number.

---

### 6. Memory-Mapped (mmap)

```
Process Virtual Memory:
[0x0000 ... Database File ... 0xFFFF]
     ↑ OS handles page faults ←→ Disk automatically ↑
```

| Dimension | Detail |
|---|---|
| **Structure** | Entire database file mapped into virtual address space. OS manages paging |
| **Read** | Direct memory access — no system calls. OS handles caching via page cache |
| **Write** | Copy-on-write (LMDB) or in-place with write barriers |
| **Concurrency** | Multiple processes can map the same file. Lock-free reads |
| **Best for** | Read-heavy embedded databases. Single-writer with many concurrent readers |
| **Used by** | LMDB, bbolt (via mmap), SQLite (WAL mode) |

**Key property:** No buffer pool to manage — the OS IS the buffer pool. Reads are just memory dereferences. Writes are copy-on-write for crash safety.

---

### 7. Fractal Tree (Cache-Oblivious B-Tree)

```
[Root with buffer] → [Internal with buffer] → [Leaf]
       ↓ flush              ↓ flush
  Messages (inserts, updates, deletes) buffered in internal nodes
  and flushed to children in batches
```

| Dimension | Detail |
|---|---|
| **Structure** | B-Tree where internal nodes contain message buffers. Inserts are messages that cascade down |
| **Read** | O(log_B N) — must check buffers along the path |
| **Write** | O(log_B N / B) — amortized, because messages are batched in buffers |
| **Write amplification** | Very low — messages are flushed in bulk |
| **Best for** | Write-heavy workloads that also need range scans |
| **Used by** | TokuDB (acquired by Percona, now EOL) |

**Key property:** Combines LSM's write throughput with B-Tree's read performance. The buffer-in-node pattern means writes don't immediately trigger I/O — they accumulate and flush in batches.

---

### 8. Adaptive Radix Tree (ART)

```
Node4:  [≤4 children, compact]   → Node16: [≤16 children]
                                      → Node48: [≤48 children]
                                         → Node256: [256 children, direct index]
```

| Dimension | Detail |
|---|---|
| **Structure** | Trie that adapts node size to occupancy. Each node type optimized for its fill level |
| **Read** | O(k) where k = key length. Cache-friendly due to compact nodes |
| **Write** | O(k) — node type promotion/demotion as needed |
| **Space** | Very compact — empty slots not allocated |
| **Best for** | In-memory indexes. Prefix-heavy workloads. String keys |
| **Used by** | HyperLevelDB (index), Hyrise (in-memory DB), some in-memory stores |

---

### 9. BW-Tree (Latch-Free B-Tree)

```
Page A → [delta record 3] → [delta record 2] → [delta record 1] → [base page]
              ↑ CAS chain — updates without latches ↑
```

| Dimension | Detail |
|---|---|
| **Structure** | B-Tree with latch-free updates via compare-and-swap delta chains |
| **Read** | O(log n) — follow delta chain to base page |
| **Write** | O(log n) — prepend delta record via CAS. No latches, no lock contention |
| **Concurrency** | Excellent — latch-free means no lock contention on multi-core |
| **Best for** | High-concurrency OLTP on many-core machines |
| **Used by** | Microsoft Hekaton (SQL Server in-memory OLTP), research prototype |

---

### 10. Time-Structured Merge Tree (TSM)

```
[Cache] → flush → [TSM File: time-sorted, compressed per field]
                       ↓ compaction → [Larger TSM File]
                            ↓ further compaction → [Even larger TSM File]
```

| Dimension | Detail |
|---|---|
| **Structure** | LSM variant optimized for time-series: shard by time range, compress per-field |
| **Read** | Time-range scan. Delta-of-delta timestamp compression. Gorilla float compression |
| **Write** | O(1) — same as LSM. Points ordered by time |
| **Compression** | Exceptional — 10-20× compression on time-series data |
| **Best for** | Time-series data exclusively |
| **Used by** | InfluxDB |

---

### 11. Vector Index (HNSW / IVF)

```
HNSW:                    IVF:
  Layer 2: [sparse]        [Centroid 1] → [vectors near centroid 1]
  Layer 1: [medium]        [Centroid 2] → [vectors near centroid 2]
  Layer 0: [dense]         [Centroid N] → [vectors near centroid N]
```

| Dimension | Detail |
|---|---|
| **HNSW** | Hierarchical navigable small world graph. Multi-layer skip-list-like graph for ANN |
| **IVF** | Inverted file index. Partition space via centroids, search nearby partitions |
| **Read** | O(log n) approximate. Recall vs speed tradeoff |
| **Write** | O(log n) for insertion. Index rebuild for large batches |
| **Best for** | Similarity search on high-dimensional vectors (embeddings) |
| **Used by** | Pinecone, Weaviate, Qdrant, Milvus, pgvector |

---

### 12. Copy-on-Write B-Tree (CoW)

```
Write: Modify leaf → copy path to root → atomically swap root pointer
Old root still valid → readers see consistent snapshot
```

| Dimension | Detail |
|---|---|
| **Structure** | B-Tree where every write creates new node copies up to root |
| **Read** | O(log n) — no latches needed, readers see consistent snapshot |
| **Write** | O(log n) — copies O(log n) nodes per write |
| **Concurrency** | Excellent — readers never block, single writer |
| **Space** | Higher — old versions kept until no readers reference them |
| **Best for** | Single-writer, multi-reader embedded databases |
| **Used by** | LMDB, ZFS (filesystem), bbolt |

---

## Part 3: Cross-Reference Matrix

### Interface × Storage Engine Combinations in the Wild

| | B+Tree | LSM-Tree | Hash | Columnar | Append-Only | mmap | ART | Fractal |
|---|---|---|---|---|---|---|---|---|
| **Key-Value** | bbolt, LMDB | RocksDB, Pebble, LevelDB | Bitcask, Redis | — | — | LMDB | HyperLevelDB | — |
| **Document** | MongoDB (WiredTiger) | MongoDB (WiredTiger) | — | MongoDB (column store) | — | — | — | — |
| **Relational** | PostgreSQL, MySQL, SQLite | MyRocks | — | ClickHouse, DuckDB | Datomic | SQLite | — | TokuDB |
| **Graph** | Neo4j (native) | JanusGraph (Cassandra) | — | — | — | — | — | — |
| **Time-Series** | TimescaleDB (PG) | InfluxDB TSM, Cassandra | — | ClickHouse, QuestDB | — | — | — | — |
| **Search** | — | — | — | Elasticsearch (inverted+col) | — | — | — | — |
| **Wide-Column** | — | Cassandra, ScyllaDB | — | — | — | — | — | — |
| **Triple Store** | Virtuoso | — | — | — | — | — | — | — |
| **Datalog** | — | — | — | — | Datomic | Datahike | — | — |
| **Streaming** | — | Kafka (log segments) | — | — | Kafka, Pulsar | — | — | — |
| **Vector** | — | Qdrant (rocksdb) | — | Milvus | — | — | — | — |
| **Object** | — | — | — | — | S3 internals | — | — | — |

---

### Storage Engine Trade-off Map

```
                        Write Throughput
                              ↑
                              |
           LSM-Tree ●         |         ● Fractal Tree
                              |
     ScyllaDB (sharded LSM)   |
                              |
                              |
    ──────────────────────────┼──────────────────→ Read Performance
                              |
            Hash Index ●      |         ● B+Tree
                              |              ● mmap (LMDB)
                              |
              Append-Only ●   |
                              |              ● Columnar
                              |                    ● ART (in-memory)
                              |
                        Low Space Usage ←→ High Space Usage
```

---

## Part 4: Relevance to go-cqrs-lite

### What We Use Today

| Component | Interface Model | Storage Engine | Backend |
|---|---|---|---|
| `SQLEventStore` | Ordered KV (keyed by aggregate+version) | B+Tree (via PostgreSQL/SQLite) | PG, SQLite, Turso |
| `PebbleEventStore` | Ordered KV (keyed by aggregate+version) | LSM-Tree | Pebble |
| `SQLSnapshotStore` | Key-Value (keyed by aggregate) | B+Tree | PG, SQLite, Turso |
| `SQLCheckpointStore` | Key-Value (keyed by projection name) | B+Tree | PG, SQLite, Turso |
| `SQLOutbox` | Queue (FIFO, status-filtered) | B+Tree | PG, SQLite, Turso |

### What Our Interface Model Actually Is

Despite using SQL as the backend, the **actual access pattern** is:

1. **Event store** = Ordered KV with prefix scan (load all events for aggregate) + append (save events)
2. **Snapshot store** = Simple KV (Get/Put by aggregate key)
3. **Checkpoint store** = Simple KV (Get/Put by projection name)
4. **Outbox** = FIFO queue with status filter (poll pending, ack)

This confirms that a `kv.Store` interface (Get/Set/Delete/Iterator/Batch) covers all our needs — SQL is just one implementation strategy.

### Storage Engines That Would Fit Our Access Patterns

| Engine | Why It Fits | Trade-off |
|---|---|---|
| **LSM-Tree** (Pebble, BadgerDB) | Append-heavy (events are append-only), prefix scan for aggregate loading | Read amplification for large histories — mitigated by snapshots |
| **B+Tree** (PostgreSQL, SQLite) | Point lookup for snapshots/checkpoints, range scan for events | Write amplification on high-ingest — acceptable for most workloads |
| **Append-Only Log** (Kafka-style) | Natural fit for event sourcing — events ARE append-only | No per-aggregate random access without secondary index |
| **CoW B+Tree** (LMDB) | Snapshot reads for free (time-travel), lock-free readers | Single writer bottleneck — acceptable for embedded use |
| **Hash + Log** (Bitcask-style) | O(1) lookups for snapshots/checkpoints | No range scan — can't load events by aggregate prefix |

### Storage Engines That Would NOT Fit

| Engine | Why Not |
|---|---|
| **Columnar** | We need row-level access (load one aggregate), not column aggregates |
| **Vector Index** | No similarity search in event sourcing |
| **Fractal Tree** | EOL (TokuDB), no maintained Go implementation |
| **ART** | In-memory only, no persistence |
| **BW-Tree** | Research prototype, no Go implementation |

---

## Part 5: Advanced / Hybrid Architectures

### FoundationDB — Layer Architecture

**Storage:** Ordered KV (serialized MVCC) + distributed consensus (Raft-like)
**Interface:** Raw KV is the only interface. SQL, Document, Graph — all built as **layers** on top
**Lesson:** A single well-designed KV interface can power any higher-level model. This is the "data store aware but independent" principle taken to its extreme.

### Datomic — Database as a Value

**Storage:** Append-only transaction log (DynamoDB) + 5 sorted indexes (S3) + cache (EFS)
**Interface:** Datalog queries against immutable database values (snapshots)
**Lesson:** Every query sees a consistent point-in-time snapshot. "Time-travel" is free because history is never overwritten. This is the gold standard for event sourcing.

### TigerBeetle — Deterministic Single-Thread

**Storage:** LSM forest (collection of LSM trees) with direct I/O, no OS page cache
**Interface:** Two-phase accounting transfers (debit + credit in one operation)
**Lesson:** Single-threaded deterministic execution eliminates entire classes of concurrency bugs. For financial data, this is worth the throughput trade-off.

### SurrealDB — Multi-Model Pluggable

**Storage:** Pluggable — RocksDB (embedded), TiKV (distributed), IndexedDB (browser)
**Interface:** SurrealQL covers document, graph, relational, vector in one query language
**Lesson:** The "backend is pluggable" pattern is viable. Same query language, different storage. Aligns with our `kv.Store` abstraction approach.

---

_Sources: DB-Engines ranking, database official documentation, architecture blogs, research papers (HNSW, ART, BW-Tree, Fractal Trees), system design resources._
