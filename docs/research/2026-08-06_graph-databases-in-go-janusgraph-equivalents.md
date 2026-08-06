# Graph Databases in Go: JanusGraph Equivalents

> **Research date:** 2026-08-06
>
> **Question:** Is there something like [JanusGraph](https://github.com/janusgraph/janusgraph) in/for Golang?
>
> **Short answer:** No exact match. JanusGraph's defining trio is **distributed + pluggable storage backends + native TinkerPop/Gremlin**. No single Go project reproduces all three.

---

## Table of Contents

- [Quick Comparison](#quick-comparison)
- [Go-Native Graph Databases](#go-native-graph-databases)
- [Best Fit by What You Want from JanusGraph](#best-fit-by-what-you-want-from-janusgraph)
- [Deep Dive: Cayley](#deep-dive-cayley)
- [Deep Dive: GoraphDB](#deep-dive-goraphdb)
- [Deep Dive: NornicDB](#deep-dive-nornicdb)
- [Non-Go Alternatives for Context](#non-go-alternatives-for-context)
- [Relevance to go-cqrs-lite](#relevance-to-go-cqrs-lite)
- [Recommendation Matrix](#recommendation-matrix)

---

## Quick Comparison

### Go-native options

| Project                  | Distributed          | Pluggable backends                | Query language                    | Status                              |
| ------------------------ | -------------------- | --------------------------------- | --------------------------------- | ----------------------------------- |
| **Dgraph** (21.8k stars) | Yes (Raft, sharded)  | No (own engine)                   | GraphQL + DQL (no Gremlin/Cypher) | Active, production                  |
| **Cayley** (15.1k stars) | No (standalone)      | Yes (LevelDB/Bolt/Postgres/Mongo) | Gizmo (Gremlin-_inspired_)        | **Maintenance** (dev stalled ~2021) |
| **NornicDB** (838 stars) | Yes (cluster, MVCC)  | No (built-in)                     | Cypher via Neo4j Bolt protocol    | Active, early                       |
| **GoraphDB** (98 stars)  | Partial (Raft repl.) | No (bbolt embedded)               | Cypher                            | Active, very early                  |
| **EliasDB** (1k stars)   | Experimental         | No                                | EQL/GraphQL                       | **Abandoned** (2022)                |
| **go-graph** (11 stars)  | Yes (sharded)        | No                                | API only                          | **Abandoned** (academic)            |

### Full JanusGraph comparison (including non-Go)

| Database             | Language    | Distributed      | Pluggable Backends | Gremlin/TinkerPop  | Cypher   | Status         |
| -------------------- | ----------- | ---------------- | ------------------ | ------------------ | -------- | -------------- |
| **JanusGraph**       | Java        | Yes              | Yes                | Yes (native)       | No       | Active         |
| **Dgraph**           | **Go**      | Yes              | No                 | No                 | No       | Active         |
| **Cayley**           | **Go**      | No               | Yes                | ~ (Gizmo inspired) | No       | Maintenance    |
| **NornicDB**         | **Go**      | Yes              | No                 | No                 | Yes      | Active         |
| **GoraphDB**         | **Go**      | ~ (Raft repl.)   | No                 | No                 | Yes      | Active (early) |
| **EliasDB**          | **Go**      | ~ (experimental) | No                 | No                 | No       | Abandoned      |
| **go-graph**         | **Go**      | Yes              | No                 | No                 | No       | Abandoned      |
| **Apache HugeGraph** | Java        | Yes              | Yes                | Yes                | Yes      | Active         |
| **Neo4j**            | Java        | ~ (Enterprise)   | No                 | ~ (plugin)         | Yes      | Active         |
| **ArangoDB**         | C++         | Yes              | No                 | No                 | No       | Active         |
| **Memgraph**         | C/C++       | No (in-memory)   | No                 | No                 | Yes      | Active         |
| **Apache AGE**       | C (PG ext.) | ~ (via PG)       | No                 | No                 | Yes      | Active         |
| **NebulaGraph**      | C++         | Yes              | No                 | No                 | ~ (nGQL) | Active         |

---

## Go-Native Graph Databases

### Dgraph (21.8k stars)

| Attribute                    | Details                                                                        |
| ---------------------------- | ------------------------------------------------------------------------------ |
| **Language**                 | Go                                                                             |
| **Architecture**             | Distributed, horizontally scalable, sharded, Raft-based consistent replication |
| **Query Language**           | GraphQL + DQL (Dgraph Query Language)                                          |
| **TinkerPop/Gremlin/Cypher** | No                                                                             |
| **Storage Backends**         | Built-in (own storage engine, Badger-derived). Not pluggable.                  |
| **Status**                   | Active, v25, production-ready, used by Fortune 500 companies                   |
| **License**                  | Apache 2.0                                                                     |
| **GitHub**                   | https://github.com/dgraph-io/dgraph                                            |

Distributed ACID transactions, linearizable reads, native full-text search, and geo search. Does not support Gremlin/TinkerPop. The strongest production-grade Go-native distributed database, but own query language and non-pluggable storage.

### Cayley (15.1k stars)

| Attribute            | Details                                                                                                                      |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| **Language**         | Go                                                                                                                           |
| **Architecture**     | Embedded + standalone server (not distributed)                                                                               |
| **Query Language**   | Gizmo (Gremlin-_inspired_), GraphQL, MQL                                                                                     |
| **Storage Backends** | Pluggable: LevelDB, Bolt, PostgreSQL, MongoDB, CockroachDB, and others                                                       |
| **Status**           | Maintenance mode / low activity. Last commit July 2024 (dependency updates only); substantive development stopped ~2021-2022 |
| **License**          | Apache 2.0                                                                                                                   |
| **GitHub**           | https://github.com/cayleygraph/cayley                                                                                        |

Closest Go-native analog to JanusGraph's pluggable storage architecture. Not distributed. Gremlin support is via an inspired language (Gizmo), not native Apache TinkerPop. See [deep dive below](#deep-dive-cayley).

### NornicDB (838 stars)

| Attribute            | Details                                                              |
| -------------------- | -------------------------------------------------------------------- |
| **Language**         | Go                                                                   |
| **Architecture**     | Distributed: clustering, sharding, MVCC                              |
| **Query Language**   | Cypher via Bolt (Neo4j-compatible), GraphQL, REST, gRPC              |
| **Storage Backends** | Built-in. Not pluggable. Qdrant-compatible gRPC for vector workflows |
| **Status**           | Active, v1.2.2, 1,491 commits, rapidly evolving                      |
| **License**          | MIT                                                                  |
| **GitHub**           | https://github.com/orneryd/NornicDB                                  |

Graph + vector database designed for AI/GraphRAG workloads. Neo4j-compatible via the Bolt protocol and Cypher, meaning existing Neo4j drivers work with zero changes. Supports temporal/historical reads, memory decay, and GPU acceleration. Claims 12-52x faster than Neo4j on LDBC benchmarks. See [deep dive below](#deep-dive-nornicdb).

### GoraphDB (98 stars)

| Attribute          | Details                                                           |
| ------------------ | ----------------------------------------------------------------- |
| **Language**       | Go                                                                |
| **Architecture**   | Embedded: single binary, zero dependencies                        |
| **Query Language** | Cypher (full read/write: MATCH, CREATE, MERGE, SET, DELETE, etc.) |
| **Storage**        | bbolt (B+tree), memory-mapped, MVCC snapshot isolation            |
| **Status**         | Active, 92 commits, new/early-stage project                       |
| **Replication**    | Raft-based single-leader replication with WAL log shipping        |
| **GitHub**         | https://github.com/mstrYoda/goraphdb                              |

High-performance embedded graph database with Cypher support, ACID transactions, built-in graph algorithms, hash-based sharding, and Prometheus metrics. 50 GB+ scale claimed. Not distributed as a primary design goal. See [deep dive below](#deep-dive-goraphdb).

### EliasDB (1k stars)

| Attribute          | Details                                                             |
| ------------------ | ------------------------------------------------------------------- |
| **Language**       | Go                                                                  |
| **Architecture**   | Embedded + standalone (experimental clustering)                     |
| **Query Language** | EQL (SQL-like), GraphQL                                             |
| **Storage**        | Custom key-value store with transactions                            |
| **Status**         | Abandoned. Last commit August 2022; DBDB.io marks it as "Abandoned" |
| **License**        | MPL-2.0                                                             |
| **GitHub**         | https://github.com/krotik/eliasdb                                   |

Lightweight, dependency-free graph database with REST API, full-text phrase search, and a built-in scripting language (ECAL).

### go-graph (11 stars)

| Attribute          | Details                                                     |
| ------------------ | ----------------------------------------------------------- |
| **Language**       | Go                                                          |
| **Architecture**   | Distributed: fault-tolerant, sharded, eventually consistent |
| **Query Language** | None (API-based)                                            |
| **Status**         | Abandoned. Academic/research project                        |
| **License**        | Apache 2.0                                                  |
| **GitHub**         | https://github.com/ashriths/go-graph                        |

A distributed, fault-tolerant graph database conceptually similar to JanusGraph in ambition, but an academic project that was never completed to production quality.

---

## Best Fit by What You Want from JanusGraph

- **Distributed scale + Go** -> **Dgraph** (the strongest production-grade option, but its own query language and non-pluggable storage)
- **Pluggable storage backends** -> **Cayley** (only Go project with this, but unmaintained)
- **Cypher / Neo4j-driver compatibility** -> **NornicDB** (Bolt-protocol drop-in) or **Memgraph** (C++, but Cypher-native)
- **True JanusGraph parity (distributed + pluggable + Gremlin)** -> **Apache HugeGraph** (Java) is the actual closest analog that exists today; **NebulaGraph** (C++) is the distributed-at-massive-scale alternative

---

## Deep Dive: Cayley

Cayley is an open-source graph database for **Linked Data**, written in **Go**. It was created at Google in 2014 by Barak Michener and is explicitly inspired by the graph database behind Google's **Knowledge Graph** (formerly Freebase). The data model is **RDF quads** (subject, predicate, object, label) rather than the property-graph model used by Neo4j. It debuted in Google's own GitHub organization (`google/cayley`) and later moved to the community-owned `cayleygraph/cayley` org.

### At a Glance

|                | Cayley                                                       |
| -------------- | ------------------------------------------------------------ |
| Maturity       | Abandoned (last release v0.7.7, October 2019)                |
| Data model     | RDF quads (S-P-O-Label)                                      |
| Query language | Gizmo (Gremlin-_inspired_, JS VM)                            |
| Distribution   | None (delegate to backend)                                   |
| Storage        | Pluggable: Bolt, LevelDB, Postgres, Mongo, CockroachDB, etc. |
| License        | Apache 2.0                                                   |
| Best for       | Museum-quality reference design                              |

Notable headline facts (verified July-August 2026):

- **License:** Apache 2.0 (permissive, no commercial edition, no strings)
- **~15.1k GitHub stars** -- the most-starred graph database repo, but this reflects 2014-2016 enthusiasm, not current health
- **Last tagged release: v0.7.7 (October 2019)** -- over six years ago
- **Status:** Classified as **"Abandoned"** by DBDB.io. The repo is _not_ archived; the last commit was July 6, 2024, and there is occasional activity, but no release cadence.

### Architecture (genuinely elegant)

Cayley is built with a **layered, interface-driven** architecture:

```
Query Languages (Gizmo / GraphQL / MQL)
        |
   Iterator System (query execution engine)
        |
   QuadStore interface  <- the central abstraction
        |
   Concrete backends (MemStore, Bolt, LevelDB, SQL, NoSQL)
```

#### The QuadStore interface

This is the **central abstraction**. Any storage backend implements a small set of methods:

- `ValueOf` / `NameOf` -- convert between quad values and internal integer references
- `Quad` / `QuadDirection` -- retrieve quads or a specific direction from a reference
- `QuadIterator` -- create an iterator over quads matching a direction+value
- `NodesAllIterator` / `QuadsAllIterator` -- full scans
- `ApplyDeltas` -- atomic batch of adds/removes
- `NewQuadWriter` -- create a writer for streaming quads in

#### The Quad data model

The fundamental unit is a **quad**: `Subject - Predicate - Object - Label`. The `Label` (sometimes called "context" or "graph") acts as an optional subgraph identifier, allowing multiple overlay graphs in one store (queried via `.labelContext()`).

#### Iterator System

The query execution engine is an **iterator tree** -- the same design pattern as Apache TinkerPop. Gizmo/GraphQL/MQL queries all compile down to a tree of composable iterators:

- `AllIterator` -- iterates all nodes or quads
- `QuadIterator` -- iterates quads with a fixed value in a specific direction
- `ValueIterator` -- follows paths through the graph

An **optimizer** reorders the tree based on:

1. **Selectivity** of each iterator (estimated result cardinality)
2. **Cost** of retrieving from different indexes
3. **Optimal ordering** to minimize intermediate results

#### Performance optimizations

- **Bloom filters** -- fast existence checks
- **LRU caching** -- hot values kept in memory
- **Batch operations** -- grouped writes to reduce transaction overhead
- **Lazy evaluation** -- defers computation until results are consumed
- Built-in **metrics** for query performance and storage operations

#### Transactions

A `Transaction` groups multiple `Delta` operations (adds/removes) and applies them **atomically**. It does conflict resolution inline: an add+remove of the same quad cancel out, and duplicates are deduplicated. The `Handle` struct combines a `QuadStore` + `QuadWriter` into one object.

### Pluggable Storage Backend System

This is one of Cayley's most interesting design features. Backends are registered via `graph.RegisterQuadStore()` and selected by a string name in config (`store.backend`). To add a new backend you implement the `QuadStore` interface and register it -- the rest of the system (query languages, iterators, HTTP API) works unchanged.

#### Supported backends

| Backend               | Type             | `store.backend` value | Notes                                                                                                                |
| --------------------- | ---------------- | --------------------- | -------------------------------------------------------------------------------------------------------------------- |
| **MemStore**          | In-memory        | `memstore` (default)  | Maps + trees; loses data on exit. Best for datasets fitting in RAM.                                                  |
| **BTree (in-mem KV)** | In-memory KV     | `btree`               | Used mainly to verify KV-backend functionality.                                                                      |
| **LevelDB**           | Embedded KV      | `leveldb`             | Persistent on-disk LSM-tree store. Tunable write buffer + block cache.                                               |
| **BoltDB**            | Embedded KV      | `bolt`                | Single-file B+tree. Recommended for large/persistent datasets. Faster avg query times than LevelDB for large stores. |
| **PostgreSQL**        | Relational (SQL) | `postgres`            | Requires PG 9.5+. Tunable fill factor, connection pooling.                                                           |
| **CockroachDB**       | Relational (SQL) | `cockroach`           | Distributed SQL -- runs the SQL backend over a CockroachDB cluster.                                                  |
| **MySQL / MariaDB**   | Relational (SQL) | `mysql`               |                                                                                                                      |
| **SQLite**            | Relational (SQL) | `sqlite`              |                                                                                                                      |
| **MongoDB**           | NoSQL            | `mongo`               | Graph data + indices stored in Mongo collections.                                                                    |
| **Elasticsearch**     | NoSQL            | `elastic`             |                                                                                                                      |
| **CouchDB**           | NoSQL            | `couch`               |                                                                                                                      |
| **PouchDB**           | NoSQL            | `pouch`               | Requires building with GopherJS (browser/JS target).                                                                 |
| **GAE Datastore**     | NoSQL            | (appengine)           | Google App Engine Datastore.                                                                                         |

#### How the abstraction works

There are really **two implementation strategies** the backends fall into:

1. **Direct `QuadStore` implementations** -- MemStore (`graph/memstore/`) and the legacy NoSQL backends implement the full interface from scratch.

2. **The generic `kv` QuadStore** (`graph/kv/`) -- A single shared implementation that sits on top of a simple **key-value interface**. LevelDB, BoltDB, and BTree all use this. You only implement the tiny KV interface; the generic layer handles indexing, iteration, and query translation. This is why Bolt and LevelDB are "easy" backends.

#### Indexing

- **KV backends** maintain configurable quad indexes. The defaults are:
  - `{Subject, Predicate}` -- optimizes **forward** traversals (outgoing edges)
  - `{Object, Predicate, Subject}` -- optimizes **reverse** traversals (incoming edges) and mitigates **super-node** problems (high-fan-out literal values)
- **MemStore** uses a `QuadDirectionIndex` maintaining trees for _every_ direction (S, P, O, L), allowing fast lookup by any component.

#### Enabling a backend as a Go library

Backends are pulled in via **blank imports**:

```go
import _ "github.com/cayleygraph/cayley/graph/kv/bolt"
```

then opened by name:

```go
graph.InitQuadStore("bolt", path, nil)
store, _ := cayley.NewGraph("bolt", path, nil)
```

### The Gizmo Query Language (and its relation to Gremlin)

**Gizmo is Cayley's primary query language and is explicitly "inspired by [Gremlin](https://tinkerpop.apache.org/gremlin.html)."** The relationship is important to understand precisely:

- **Gremlin** is Apache TinkerPop's graph traversal language. It's a fluent, method-chaining API (`g.V().out('knows').has('age', gt(30))`) that runs on the JVM and compiles traversals to a "TraversalSource."
- **Gizmo** borrows Gremlin's _concepts and ergonomics_ -- `V()`, `out()`, `in()`, `has()`, path composition, morphisms (reusable sub-paths), tags -- but **reimplements them on a JavaScript runtime** (the `goja` Go-JS engine) rather than running TinkerPop. It is **Gremlin-shaped but not Gremlin-compatible**; you cannot point a TinkerPop Gremlin console at Cayley, and Gremlin queries won't run unmodified.

#### How a Gizmo query executes

```
JS source -> goja VM parses & runs -> builds a Path object (method chaining)
         -> Path compiles to an iterator tree -> iterator executed against QuadStore
         -> results returned as JS objects
```

The `graph` object (alias `g`) is the entry point. `g.V(...)` starts a query; `g.Morphism()` starts a reusable, unqueryable path pattern.

#### Core API surface

| Category     | Methods                                                                  | Purpose                          |
| ------------ | ------------------------------------------------------------------------ | -------------------------------- |
| Start        | `g.V()`, `g.Vertex()`, `g.M()`, `g.Morphism()`                           | Begin a path                     |
| Traversal    | `.out()`, `.in()`, `.both()`                                             | Follow edges (fwd/rev/both)      |
| Filtering    | `.is()`, `.has()`, `.hasR()`, `.filter(regex(...))`                      | Constrain paths                  |
| Set ops      | `.and()`/`.intersect()`, `.or()`/`.union()`, `.except()`/`.difference()` | Combine paths                    |
| Tagging      | `.tag()`/`.as()`, `.back()`, `.save()`, `.saveR()`                       | Mark & recall positions          |
| Finalization | `.all()`, `.count()`, `.getLimit(n)`, `.toArray()`, `.forEach(cb)`       | Execute & return                 |
| Navigation   | `.outPredicates()`, `.inPredicates()`, `.labels()`                       | Inspect schema                   |
| Subgraph     | `.labelContext()`, `.followRecursive()`                                  | Restrict/recursively traverse    |
| Programmatic | `g.emit(obj)`                                                            | Push arbitrary JSON into results |

#### Code examples (querying)

**Find all vertices:**

```js
g.V().getLimit(5);
```

**Find by property (movie graph):**

```js
g.V().has("<name>", "Humphrey Bogart").all();
```

**Multi-hop: actors in a film:**

```js
g.V()
	.has("<name>", "Casablanca")
	.out("</film/film/starring>")
	.out("</film/performance/actor>")
	.out("<name>")
	.all();
```

**Morphisms (reusable path patterns) -- the Gremlin-`as()`/`select()` analogue:**

```js
var filmToActor = g.Morphism().out("</film/film/starring>").out("</film/performance/actor>");

g.V().has("<name>", "Casablanca").follow(filmToActor).out("<name>").all();
```

**Friend-of-friend (social graph):**

```js
var friendOfFriend = g.Morphism().out("<follows>").out("<follows>");
g.V("<charlie>").follow(friendOfFriend).has("<status>", "cool_person").all();
```

**Recursive traversal (whole reachable network):**

```js
var friend = g.Morphism().out("<follows>");
g.V("<charlie>").followRecursive(friend).all();
```

**Intersection (people followed by both X and Y):**

```js
var cFollows = g.V("<charlie>").out("<follows>");
var dFollows = g.V("<dani>").out("<follows>");
cFollows.intersect(dFollows).all();
```

**Subgraph context (traverse only a specific labeled graph):**

```js
g.V("<dani>").labelContext("<smart_graph>").out("<status>").all();
```

#### Data modeling

Cayley has no fixed schema -- data is **quads**, typically loaded from **N-Quads** files:

```nquads
<alice> <follows> <bob> .
<alice> <name> "Alice Smith" .
<bob>   <status> "cool_person" <smart_graph> .
<bob>   <follows> <fred> .
```

There _is_ an optional **Schema System** (`schema/` package) that lets you map Go structs to quads for typed read/write from Go code, and an **Inference Engine** (`inference/`) for RDFS-style subclass/subproperty reasoning -- both are advanced features layered on top of the quad model.

#### Go library usage (the "embedded" mode)

```go
package main

import (
    "fmt"
    "log"
    "github.com/cayleygraph/cayley"
    "github.com/cayleygraph/quad"
)

func main() {
    store, err := cayley.NewMemoryGraph()
    if err != nil { log.Fatalln(err) }

    store.AddQuad(quad.Make("phrase of the day", "is of course", "Hello World!", nil))

    p := cayley.StartPath(store, quad.String("phrase of the day")).
        Out(quad.String("is of course"))

    p.Iterate(nil).EachValue(nil, func(value quad.Value) {
        fmt.Println(quad.NativeOf(value)) // "Hello World!"
    })
}
```

#### Other query languages

- **MQL** -- a simplified version of Freebase's Metaweb Query Language, for ex-Freebase users
- **GraphQL-inspired** language -- a JSON-shaped traversal language
- **LinkedQL** -- a more recent, structured query layer

### Replication & Clustering Story

This is **Cayley's weakest area and a major reason it lost ground.** The honest answer: **there is essentially no first-class clustering/replication.**

- Cayley is fundamentally a **single-node/embedded** database. There is **no built-in distributed consensus, no Raft, no sharding, no replication manager** in the engine itself.
- The configuration docs reference a `replication_options` object and "Per-Replication Options," but this is vestigial/stub-level -- it does not constitute a working HA story.
- The **closest thing to horizontal scaling** is the architectural escape hatch: if you use the **NoSQL backends (MongoDB, CouchDB, Elasticsearch)** or the **distributed SQL backends (CockroachDB)**, then _the underlying store_ provides its own replication/clustering, and multiple Cayley instances can connect to it. The docs describe this explicitly:

  > _"NoSQL backends: Slower, as it incurs network traffic, but multiple Cayley instances can disappear and reconnect at will, across a potentially horizontally-scaled store."_

  CockroachDB similarly gives you a distributed SQL cluster that Cayley sits on top of.

- There is **no multi-master write coordination between Cayley instances** themselves, no write-ahead-log replication, and no automatic failover of a Cayley process. The embedded KV backends (Bolt, LevelDB) are strictly **single-process** -- only one writer can hold the file.

**Bottom line:** For HA you must rely entirely on the backend (Mongo Replica Set, CockroachDB cluster) and treat Cayley as stateless query layers in front of it. This is workable for read-heavy workloads but is not a purpose-built distributed graph database like JanusGraph (which shards a graph across Cassandra/HBase).

### Performance Characteristics

Cayley's own benchmark claim (from the README, measured on 2014 hardware):

> _"On 2014 consumer hardware and an average disk, 134m quads in LevelDB is no problem and a multi-hop intersection query -- films starring X and Y -- takes ~150ms."_

Practical characteristics:

- **Strength:** Fast multi-hop traversals on single-node datasets that fit the index well, especially with Bolt/LevelDB. The dual `{S,P}` / `{O,P,S}` indexes make forward and reverse traversals efficient.
- **Strength:** Low operational footprint -- a single Go binary, no JVM.
- **Weakness:** The `except()` set-difference operation against all nodes (`g.V().except(path)`) is explicitly flagged as **"often very slow"** because it requires materializing the complement.
- **Weakness:** Gizmo runs through a **JavaScript VM (goja)**, which adds overhead vs. the native Go Path API. The docs recommend the Go API "for maximum performance."
- **Weakness:** No vector search, no modern approximate-nearest-neighbor indexing -- entirely outside the embedding era.
- **Weakness:** SQL backends carry network round-trip costs; NoSQL backends even more so.
- **Mitigations available:** per-backend tuning (LevelDB write buffer up to 200+ MiB for bulk loads, Bolt `nosync` for fast but non-durable loads), LRU caching, bloom filters, batch loading (`load.batch`, default 10,000).

Realistic expectation in 2026 terms: for a **read-mostly linked-data store of millions of quads on one machine**, Cayley is perfectly performant. It is not competitive with modern in-memory engines (Memgraph) or columnar analytical engines (Kuzu) on large analytical workloads, and it has no answer for GraphRAG.

### Why Development Stalled

Several converging factors:

1. **Loss of corporate stewardship.** Cayley began inside Google (`google/cayley`). It was always a "20% / experimental" project rather than a Google product. When the original maintainer (Barak Michener) moved on, it transferred to the `cayleygraph` community org but **never attracted a funded maintainer or a company to back it**. Tracxn classifies it as "unfunded, based in Paris, founded 2014."

2. **No business model / no vendor.** Unlike Neo4j (commercial), Memgraph (VC-backed), or JanusGraph (Linux Foundation), Cayley had **no company whose survival depended on it**. Apache 2.0 with no Enterprise edition meant no revenue engine to sustain development.

3. **Last release in 2019.** v0.7.7 shipped in October 2019. After that, the repository saw only sporadic commits (last commit July 6, 2024 per DBDB.io), and never cut another release. DBDB.io formally reclassified it as **"Abandoned."**

4. **The market moved on without it.** The graph-database space consolidated around property graphs (Cypher/Gremlin), distributed stores (JanusGraph over Cassandra), and most recently GraphRAG (graph + vectors). Cayley's RDF-quad + custom-query-language niche shrank.

5. **It missed the two biggest waves:** (a) distributed/cloud-native graph databases, and (b) vector similarity / AI retrieval. As the ArcadeDB 2026 comparison bluntly puts it: _"Cayley predates the entire embedding era and has no answer for semantic retrieval,"_ and its 15k stars _"reflect 2015 enthusiasm rather than 2026 maintenance."_

### Who Used It in Production

- The README claims Cayley is _"well tested and used by various companies for their production workloads,"_ but **does not name them**.
- Third-party aggregator **TheirStack.com lists 48 companies** as using Cayley -- though such lists are inferred from job postings/tech stacks and are of mixed reliability.
- It has **no publicly named marquee production user** (no equivalent to, say, Cisco/Adobe on Neo4j). The clearest signal is that adoption was **read-heavy knowledge-graph / linked-data** use cases: semantic data, ontology exploration, recommendation prototypes, and Freebase/Knowledge-Graph-style applications.
- It retains cultural footprint via the star count and the Cayley CookBook (community tutorials for using it as a Go library).

### Is It Viable Today?

**For new projects: No.**

The 2026 expert consensus is unambiguous:

- **No release in ~7 years.** Bug fixes, security patches, and dependency updates do not arrive on any reliable schedule.
- **No clustering/HA of its own** -- a real problem for load-bearing infrastructure.
- **No vector search / no AI integration** -- disqualifying for modern knowledge-graph / GraphRAG workloads.
- The ArcadeDB comparison's verdict: _"We would not start a new knowledge graph on Cayley in 2026... its star count keeps it near the top of search results, and readers deserve to know that the number reflects 2015 enthusiasm rather than 2026 maintenance."_

#### Viable for very narrow cases

- **Embedded, read-mostly, single-process Go applications** with a static linked-data dataset that fits in memory or a single Bolt file.
- **Learning/teaching** graph-traversal concepts and RDF quads.
- **Prototyping** a knowledge-graph UI -- the built-in query editor/visualizer/REPL is genuinely pleasant.

#### Forking: legal yes, practical caution

- **Legally trivial:** Apache 2.0 means forking is unrestricted. With ~1.2k existing forks, the community has already shown willingness.
- **Practically heavy:** You'd be **adopting a database engine** -- you own its unfixed bugs, its aging Go dependencies, and the burden of modernizing it (adding vectors, clustering, a maintained query language, dependency upgrades). That is a large, ongoing engineering commitment, not a one-time effort.

### Cayley Verdict

Cayley is a **beautifully designed but effectively abandoned** piece of software. Its pluggable-backend architecture, clean Go implementation, and Gremlin-inspired Gizmo API were genuinely ahead of their time in 2014-2016. But seven years without a release, no native clustering, and zero vector/AI capability mean it is a **museum-quality reference implementation** today rather than a recommended foundation for new infrastructure. Fork it only if you're prepared to become its maintainer.

---

## Deep Dive: GoraphDB

GoraphDB (repo: `github.com/mstrYoda/goraphdb`) is an embeddable, single-binary graph database written in Go, built on top of **bbolt** (etcd's B+tree key-value store). It supports a Cypher subset, a fluent Go query builder, optional hash-based sharding, single-leader replication, and a React management UI. License: MIT. As of this writing: **98 stars, 4 forks, 92 commits, single primary author** ("mstrYoda").

### At a Glance

|                | GoraphDB                                      |
| -------------- | --------------------------------------------- |
| Maturity       | Early (98 stars, 92 commits, 1 author)        |
| Data model     | Property graph                                |
| Query language | Cypher **subset** (hand-written lexer/parser) |
| Distribution   | Raft election + async WAL (non-quorum)        |
| Storage        | bbolt (B+tree, mmap)                          |
| License        | MIT                                           |
| Best for       | Embedded single-node Go apps                  |

### Architecture

```
Management UI (React + TS + Tailwind + cytoscape.js + CodeMirror)
        |
HTTP / JSON API  (/api/cypher, /api/nodes, /api/edges, /api/indexes, /api/health, /api/cluster)
        |
Replication Layer  (WAL, gRPC log shipping, Applier, Raft election, Query Router, Write Forwarding)
        |
Public Go API  (Node/Edge CRUD, Labels, Transactions, BFS/DFS, Paths, Query Builder, VerifyIntegrity)
        |
Cypher Engine  (Lexer -> Parser -> AST -> Executor)  EXPLAIN/PROFILE, OPTIONAL MATCH, $params, plan cache, LIMIT push-down, top-K heap
        |
Shard Manager  (hash routing, cross-shard edges, worker pool, sharded LRU node cache)
        |
Storage Layer  (bbolt B+tree, mmap files, MVCC, MessagePack, CRC32, labels index)
```

**Key architectural file map:**

| File                                                     | Responsibility                                                        |
| -------------------------------------------------------- | --------------------------------------------------------------------- |
| `graphdb.go`                                             | Core `DB` struct, `Open`/`Close`, top-level CRUD, shard orchestration |
| `tx.go`                                                  | `Tx` -- multi-statement read-write transactions                       |
| `snapshot.go`                                            | `Snapshot` -- pinned MVCC read views                                  |
| `shard.go`                                               | Hash partitioning (`DefaultShardKey`, `PropertyShardKey`)             |
| `storage.go`                                             | bbolt bucket setup and low-level accessors                            |
| `wal.go` / `wal_entry.go`                                | Segmented write-ahead log                                             |
| `applier.go`                                             | Deterministic/idempotent WAL replay on followers                      |
| `replication/{server,client,election,cluster,router}.go` | gRPC log shipping, hashicorp/raft election, write forwarding          |
| `cypher_{lexer,parser,ast,exec,cache,optional,write}.go` | The Cypher engine pipeline                                            |
| `governor.go`                                            | Query resource limits + panic recovery                                |
| `metrics.go`                                             | Atomic counters + Prometheus text exposition                          |
| `bloom.go` / `node_cache.go` / `compaction.go`           | Bloom filter, byte-budgeted LRU, GC                                   |

### Cypher Implementation & Supported Clauses

GoraphDB implements a **hand-written Cypher engine** (Lexer -> Parser -> AST -> Executor) rather than embedding a third-party Cypher library. The pipeline files are `cypher_lexer.go`, `cypher_parser.go`, `cypher_ast.go`, `cypher_exec.go`, `cypher_cache.go`, `cypher_optional.go`, `cypher_write.go`.

#### Supported clauses / features

**Reading:**

- `MATCH` -- node, label, property, and relationship patterns
- `OPTIONAL MATCH` -- left-outer-join semantics (unmatched bindings -> `nil`)
- `WHERE` -- property predicates (e.g., `WHERE n.age > 25`)
- Inline property filters in node patterns: `(n {name: "Alice"})`
- Variable-length paths: `(a)-[:follows*1..3]->(b)`
- `type(r)` relationship-type function
- `RETURN` with property projection (`b.name`, `a, b`)
- `ORDER BY ... [DESC]` with a **min-heap top-K** optimization
- `LIMIT` with **push-down** (early scan termination when no ORDER BY)
- `SKIP` (pagination)
- `$param` parameterized queries + plan caching
- `EXPLAIN` (zero-I/O plan tree) and `PROFILE` (per-operator rows + timing)

**Writing:**

- `CREATE` -- nodes, edges, comma-separated patterns; optional `RETURN`
- `MERGE` -- match-or-create upsert with `ON CREATE SET` / `ON MATCH SET`
- `MATCH ... SET` (property updates)
- `MATCH ... DELETE` (node removal)

**Execution optimizations claimed:**

- Index-aware execution (auto-selects property/composite/label indexes)
- LIMIT push-down
- ORDER BY + LIMIT min-heap
- Query-plan caching (bounded LRU, default 10K entries)
- Prepared statements: `PrepareCypher` / `ExecutePrepared` / `ExecutePreparedWithParams`
- Streaming via `CypherStream()` returning a lazy `RowIterator` (O(1) memory for non-sorted queries)

#### Example: EXPLAIN output

```go
res, _ := db.Cypher(ctx, `EXPLAIN MATCH (n:Person) WHERE n.age > 25 RETURN n`)
fmt.Println(res.Plan.String())
// EXPLAIN:
// -- ProduceResults (n)
//     -- Filter (WHERE clause)
//         -- NodeByLabelScan (n:Person)
```

#### Example: MERGE upsert

```go
res, _ := db.Cypher(ctx,
  `MERGE (n:Person {name: "Eve"}) ON CREATE SET n.role = "admin" RETURN n`)
```

> **Honest caveat on scope:** This is a _subset_ of Cypher. The README explicitly says "read-only subset" (though write clauses were later added). Notable gaps vs. full Cypher/Neo4j: no `UNION`, no `WITH` subquery chaining, no aggregation functions like `count()`/`collect()` documented, no list comprehensions, no APOC, no full-text/index-backed range scans for `WHERE` on numerics (range indexes are explicitly a roadmap item), no `CALL` for procedures, and no GQL. So the engine is competent for simple pattern matching but not a drop-in Cypher replacement.

### bbolt Storage Internals

GoraphDB stores all graph data in bbolt B+tree buckets:

| Bucket           | Key                                 | Value                                         |
| ---------------- | ----------------------------------- | --------------------------------------------- |
| `nodes`          | `uint64 nodeID`                     | MessagePack props + CRC32                     |
| `edges`          | `uint64 edgeID`                     | Binary edge (from, to, label, props) + CRC32  |
| `adj_out`        | `nodeID \| edgeID`                  | `targetID \| label` (outgoing adjacency list) |
| `adj_in`         | `nodeID \| edgeID`                  | `sourceID \| label` (incoming adjacency list) |
| `idx_prop`       | `"prop:value" \| nodeID`            | (secondary property index)                    |
| `idx_edge_type`  | `"label" \| edgeID`                 | (edge-type index)                             |
| `node_labels`    | `uint64 nodeID`                     | MessagePack `[]string`                        |
| `idx_node_label` | `"label" \| nodeID`                 | (label->node index)                           |
| `idx_unique`     | `"label\0property\0value"`          | `uint64 nodeID` (unique constraint)           |
| `unique_meta`    | `"label\0property"`                 | (unique constraint metadata)                  |
| `meta`           | `"node_counter"` / `"edge_counter"` | `uint64` (ID allocation)                      |

**Encoding & integrity:**

- **MessagePack** (via `github.com/vmihailenco/msgpack/v5`) for properties -- claimed 3-5x faster and 30-50% smaller than JSON, with backward-compatible format detection.
- **CRC32 (Castagnoli)** checksums on all node/edge data, verified on every read. `VerifyIntegrity()` does a full-scan integrity check.
- **Memory-mapped files** with configurable `MmapSize` (default 256 MB), enabling the "50 GB+ ready" claim.
- **Single-writer per shard**: bbolt allows only one concurrent writer; writes are serialized per shard by bbolt's writer lock. Reads are lock-free (MVCC).

Writes within a transaction touch the buckets directly:

```go
btx.Bucket(bucketNodes).Put(encodeNodeID(id), data)
btx.Bucket(bucketAdjOut).Put(encodeAdjKey(from, id), encodeAdjValue(to, label))
btx.Bucket(bucketIdxEdgeTyp).Put(encodeIndexKey(label, uint64(id)), nil)
```

Index maintenance happens within the same transaction (no extra fsync). `AddNodeBatch` intentionally skips auto-indexing for bulk-load speed -- you call `CreateIndex()`/`ReIndex()` afterward.

### MVCC Snapshot Isolation

GoraphDB relies on **bbolt's native MVCC** for snapshot isolation, exposed via the `Snapshot` type (`snapshot.go`):

- Reads (`GetNode`, `BFS`, `Cypher`, etc.) use bbolt read transactions which are **lock-free MVCC views** -- they never acquire a mutex and see a consistent point-in-time state.
- A user can pin an explicit frozen view:

```go
snap, err := db.Snapshot()   // opens a read tx per shard
defer snap.Release()          // frees pinned B+tree pages
result, _ := snap.Cypher(ctx, "MATCH (n) RETURN n")
```

The `Snapshot` struct holds one `*bolt.Tx` per shard, opened eagerly at creation. Critically, **write queries are rejected on snapshots**:

```go
if parsed.write != nil {
    return nil, fmt.Errorf("graphdb: snapshot does not support write queries")
}
```

`Release()` closes (rolls back) all read txs; double-release is safe. The trade-off: snapshots are cheap to create but **pin B+tree pages** that would otherwise be reclaimable by the freelist, so they shouldn't be held open indefinitely.

### Raft Replication Design

Replication is **single-leader with WAL log shipping**. A subtle but important design choice: **Raft is used only for leader election, not for the data path.** Data flows through a separate WAL -> gRPC pipeline.

**Components:**

| Component       | File(s)                                          | Role                                                                                                                         |
| --------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| WAL             | `wal.go`, `wal_entry.go`                         | Append-only segmented log of committed mutations; 64 MB segments; CRC32; msgpack; monotonic LSN; supports live tailing       |
| Applier         | `applier.go`                                     | Replays WAL entries on followers -- **deterministic** (uses leader's IDs), **idempotent** (skips duplicate LSNs), sequential |
| Log Shipping    | `replication/server.go`, `replication/client.go` | gRPC server-streaming RPC (`StreamWAL`); auto-reconnect with exponential backoff                                             |
| Leader Election | `replication/election.go`                        | `hashicorp/raft` for automatic leader election/failover                                                                      |
| Cluster Manager | `replication/cluster.go`                         | Orchestrates election, gRPC server/client, role changes, peer discovery                                                      |
| Query Router    | `replication/router.go`                          | Reads -> local; writes -> forwarded to leader over HTTP                                                                      |

**WAL frame format:**

```
+----------+------------------+----------+
| 4B length| msgpack WALEntry | 4B CRC32 |
+----------+------------------+----------+
```

**18 operation types** are replicated: AddNode, AddNodeBatch, UpdateNode, SetNodeProps, DeleteNode, AddEdge, AddEdgeBatch, DeleteEdge, UpdateEdge, AddNodeWithLabels, AddLabel, RemoveLabel, CreateIndex, DropIndex, CreateCompositeIndex, DropCompositeIndex, CreateUniqueConstraint, DropUniqueConstraint.

**Write forwarding flow** (when a follower receives a write):

1. Local DB rejects with `ErrReadOnlyReplica`
2. Router serializes the op as JSON
3. Forwarded to leader's `/api/write` HTTP endpoint
4. Leader executes, returns result
5. Mutation propagates back to followers via WAL -> gRPC

**WAL group commit:** A background goroutine batches `fsync` calls (2 ms default), buffering writes to the OS immediately but fsyncing in groups. The README claims this eliminates the per-write fsync serialization that would otherwise cap throughput at ~80 ops/s.

**Roles:** `standalone` (default), `leader` (accepts writes, ships WAL), `follower` (read-only). Roles are mutable at runtime via `SetRole` -- the Raft callback flips roles on leadership change.

> **Assessment caveat:** Using Raft only for election while replicating data via a separate asynchronous gRPC/WAL channel is a **pragmatic but non-standard** architecture. It means data durability is **not Raft-quorum-gated** -- a committed write on the leader is acknowledged before followers necessarily have it. This trades strict linearizability/consensus durability for throughput and simplicity. There's also no documented synchronous-replica / `fsync-on-quorum` option.

### Graph Algorithms Included

| Algorithm                  | API                                                               |
| -------------------------- | ----------------------------------------------------------------- |
| BFS                        | `BFS(...)`, `BFSCollect(...)`, `BFSFiltered(...)`                 |
| DFS                        | `DFSCollect(...)`, `DFSFiltered(...)`                             |
| Shortest path (unweighted) | `ShortestPath(from, to)`, `ShortestPathLabeled`                   |
| Shortest path (weighted)   | `ShortestPathWeighted(from, to, "weight", defaultWt)` -- Dijkstra |
| All paths (up to maxDepth) | `AllPaths(from, to, 5)`                                           |
| Connectivity               | `HasPath(from, to)`                                               |
| Connected components       | `ConnectedComponents()`                                           |
| Topological sort           | `TopologicalSort()` -- Kahn's algorithm, errors on cycles         |

Traversals support visitor callbacks with early-exit, `maxDepth`, direction control (`Outgoing`/`Incoming`/`Both`), and edge/node filter predicates. There are **no advanced analytics** (PageRank, community detection, centrality, triangle counting) -- scope is limited to classic graph traversal/pathfinding.

### ACID Transaction Model

Transactions are exposed via `Begin`/`Commit`/`Rollback` (`tx.go`):

- **Atomicity (single-shard):** A `Tx` opens one bbolt write transaction per shard (lazily, on first touch). In single-shard mode this is truly atomic -- one bbolt write tx commits or rolls back as a unit.
- **Atomicity (multi-shard):** Each shard commits **independently**. The code comment is explicit: _"In multi-shard mode each shard commits independently -- full distributed 2PC is a future enhancement."_ So multi-shard transactions are **not atomic** across shards today. `Commit()` loops and commits each shard tx, returning the first error (which can leave a partial commit).
- **Read-your-writes (isolation within tx):** If a shard write tx is open, `GetNode` reads from that uncommitted tx, so writes are visible within the same transaction before commit.
- **Isolation across txs:** Snapshot isolation via bbolt MVCC (reads are lock-free). Only one writer per shard at a time (bbolt constraint) -- effectively serial writes.
- **Durability:** bbolt fsync on commit (unless `NoSync: true`). WAL group commit batches fsyncs in replicated mode.
- **Consistency:** Enforced via constraints (unique constraints) and CRC32 checksums. Index maintenance is within the same transaction.

**Bottom line:** ACID holds cleanly **in single-shard mode**. In multi-shard mode, the "A" (atomicity) and "C" (cross-shard consistency) weaken -- there's no 2PC, so a crash mid-commit across shards can leave partial state.

### Hash-Based Sharding

Sharding is **client-side / in-process** (not a distributed cluster feature):

```go
type ShardKeyFunc func(id NodeID, shardCount int) int

func DefaultShardKey() ShardKeyFunc {
    return func(id NodeID, shardCount int) int {
        return int(uint64(id) % uint64(shardCount))
    }
}
```

**Design decisions:**

- When `ShardCount > 1`, node IDs are partitioned (`nodeID % shardCount`) across separate bbolt files.
- **Edges are co-located with their source node** -> `OutEdges(x)` hits **1 shard**.
- **Incoming adjacency** (`adj_in`) is stored in the **target node's shard** -> `InEdges(x)` also hits 1 shard.
- **Cross-shard edge creation** uses **two separate transactions (two fsyncs)** instead of one -- so cross-shard writes are costlier.
- `RebalanceShards()` is explicitly a **placeholder** that returns an error ("not yet implemented").
- README's recommendation: `ShardCount: 1` is the default and "sufficient for most use cases."

This sharding is about **parallelizing I/O and reducing per-file mmap contention within a single process**, not about horizontal scale-out across machines.

### Prometheus Metrics

`metrics.go` implements a **dependency-free** metrics system -- no `prometheus/client_golang` import. All counters are `atomic` types for zero-contention concurrency. Prometheus text format is generated manually via `WritePrometheus(w io.Writer)`.

**Exposed metrics** (served at `GET /metrics`):

- **Counters:** `graphdb_queries_total`, `graphdb_slow_queries_total`, `graphdb_query_errors_total`, `graphdb_query_duration_microseconds_sum`, `graphdb_cache_hits_total`, `graphdb_cache_misses_total`, `graphdb_nodes_created_total`, `graphdb_nodes_deleted_total`, `graphdb_edges_created_total`, `graphdb_edges_deleted_total`, `graphdb_index_lookups_total`, `graphdb_bloom_negatives_total`
- **Gauges:** `graphdb_node_cache_bytes_used`, `graphdb_node_cache_budget_bytes`, `graphdb_nodes_current`, `graphdb_edges_current`, `graphdb_query_cache_entries`, `graphdb_query_cache_capacity`, `graphdb_query_duration_microseconds_max`

Note: duration tracking is **sum + max** (not a full histogram). There's no Prometheus-native histogram with buckets, so p50/p95/p99 at scrape time isn't directly available from `/metrics`.

### Scale Claims (50 GB+)

The "50 GB+ ready" claim rests on:

- **bbolt memory-mapped storage** with configurable `MmapSize` (default 256 MB, recommended tuning for ~50 GB datasets via `DefaultOptions()`).
- Byte-budgeted **sharded LRU node cache** (default 128 MB) with memory-based eviction -- predictable footprint regardless of node sizes.
- Compaction / GC (`Compact()`) to reclaim freelist pages and shrink files.

**Benchmark figures cited** (Apple M-series, README):

| Operation                              | Throughput |
| -------------------------------------- | ---------- |
| AddNodeBatch (100K nodes)              | ~120 ms    |
| CreateIndex (100K nodes)               | ~180 ms    |
| FindByProperty (indexed)               | < 1 ms     |
| Cypher property filter (indexed, 100K) | < 1 ms     |
| Cypher 1-hop traversal (indexed)       | < 1 ms     |
| Cypher ORDER BY + LIMIT 10 (100K)      | ~60 ms     |
| 1000x repeated Cypher (cached)         | ~200 ms    |

> **Scrutiny:** The 50 GB claim is **plausible in principle** (bbolt/mmap can grow that large), but the published benchmarks are all at **100K nodes** -- orders of magnitude below 50 GB. There are no published load-test results at multi-GB scale, no sustained-throughput numbers, and no third-party benchmarks. Treat "50 GB+" as an architectural ceiling, not a validated production figure.

### API Design

**Embedded Go API** (primary):

```go
db, _ := graphdb.Open("./my.db", graphdb.DefaultOptions())
defer db.Close()

alice, _ := db.AddNode(graphdb.Props{"name": "Alice", "age": 30})
bob, _   := db.AddNode(graphdb.Props{"name": "Bob", "age": 25})
db.AddEdge(alice, bob, "follows", graphdb.Props{"since": "2024"})

// Cypher
res, _ := db.Cypher(ctx, `MATCH (a {name: "Alice"})-[:follows]->(b) RETURN b.name`)

// Fluent builder
result, _ := db.NewQuery().From(alice).FollowEdge("follows").Depth(3).Execute()

// Transaction
tx, _ := db.Begin()
tx.AddEdge(tx_node1, tx_node2, "KNOWS", nil)
tx.Commit()
```

**Production-safety features:**

- **Query Governor**: `MaxResultRows` cap (returns `ErrResultTooLarge`) and `DefaultQueryTimeout`
- **Panic recovery**: `safeExecute`/`safeExecuteResult` wrap every public Cypher entry point; panics -> `ErrQueryPanic` with a 4KB stack trace
- **Write backpressure**: per-shard semaphore (`WriteQueueSize`, `WriteTimeout`) -> `ErrWriteQueueFull`
- **Compaction/GC**: `Compact()` reclaims freelist pages and verifies data integrity post-compaction
- **Streaming**: `CypherStream()` lazy iterator for O(1) memory on large non-sorted result sets

### GoraphDB Verdict

Impressive engineering breadth for a solo project (Cypher engine, WAL replication, Raft, Prometheus metrics, React UI, bloom filters, prepared statements, streaming, panic recovery). But: single author, no releases tagged, no security (no auth/TLS), non-consensus durability, aspirational roadmap (`RebalanceShards()` literally returns "not yet implemented"). **Viable for low-stakes embedded Go apps; not production-ready as a system of record.**

---

## Deep Dive: NornicDB

NornicDB is an open-source, Go-native graph database that combines property graph storage, native vector search, temporal MVCC history, and an AI-oriented knowledge-management layer into a single engine. It targets **GraphRAG, AI agent memory, canonical truth stores, and audit-heavy workloads**. Version 1.2.2 at time of writing; 838 stars, 47 forks, 1,491 commits on GitHub.

### At a Glance

|                | NornicDB                                  |
| -------------- | ----------------------------------------- |
| Maturity       | Active (838 stars, 1,491 commits, v1.2.2) |
| Data model     | Property graph + vectors                  |
| Query language | Cypher (Neo4j Bolt-protocol compatible)   |
| Distribution   | Hot Standby / Raft / Multi-Region         |
| Storage        | BadgerDB (LSM)                            |
| License        | MIT (+ defensive patent grant)            |
| Best for       | AI/GraphRAG workloads                     |

### Architecture

```
Client Layer     -> Neo4j Drivers / HTTP / GraphQL / Qdrant gRPC / MCP
Security Layer   -> TLS 1.3, Basic Auth, JWT, RBAC (Admin/ReadWrite/ReadOnly)
Protocol Layer   -> Bolt :7687, HTTP :7474, Qdrant gRPC :6334, Cluster :7000
Embedding Layer  -> Background worker + WITH EMBEDDING inline + LRU cache + providers
Query Processing -> Cypher Parser (nornic|antlr) -> Query Executor -> Transaction Manager
Storage Layer    -> BadgerDB (LSM-tree KV) + WAL + Schema Manager + Persistence
Search/Indexing  -> HNSW vector index + BM25 fulltext + Hybrid RRF fusion
GPU Acceleration -> Metal / CUDA / OpenCL / Vulkan backends
Replication      -> Hot Standby / Raft Consensus / Multi-Region
```

**Core design philosophy:** consolidate the critical retrieval path -- transport, embedding, search, ranking, transactional state -- into one operational unit so that graph traversal, vector retrieval, and temporal reads share a single execution engine rather than being stitched across microservices.

**Key package structure:**

| Package               | Responsibility                              |
| --------------------- | ------------------------------------------- |
| `pkg/nornicdb`        | Main DB API                                 |
| `pkg/storage`         | BadgerDB + WAL + MVCC                       |
| `pkg/search`          | Vector (HNSW) + BM25 + Hybrid RRF           |
| `pkg/cypher`          | Query parser + executor + vector procedures |
| `pkg/bolt`            | Neo4j Bolt protocol server                  |
| `pkg/server`          | HTTP server                                 |
| `pkg/embed`           | Embedding service + cache + worker          |
| `pkg/gpu`             | GPU backends (metal/cuda/opencl/vulkan)     |
| `pkg/index`           | HNSW vector index                           |
| `pkg/mcp`             | MCP server (6 LLM tools)                    |
| `pkg/knowledgepolicy` | Decay/promotion scoring engine              |
| `pkg/replication`     | HA / Raft / Multi-Region                    |
| `pkg/qdrantgrpc`      | Qdrant-compatible gRPC bridge               |

### Cypher/Bolt Protocol Compatibility

NornicDB implements the **Neo4j Bolt binary protocol** on port 7687 and a **Cypher query engine**. This means existing Neo4j drivers (Python, JavaScript, Go, Java, .NET) connect without changes:

```python
from neo4j import GraphDatabase

driver = GraphDatabase.driver("bolt://localhost:7687")
with driver.session() as session:
    session.run("CREATE (n:Memory {content: 'Hello NornicDB'})")
```

**Two parser modes** (`NORNICDB_PARSER` env var):

| Feature          | `nornic` (default)                                          | `antlr`                               |
| ---------------- | ----------------------------------------------------------- | ------------------------------------- |
| Approach         | String-based validation (regex/indexOf) -> direct execution | ANTLR lexer/parser -> full parse tree |
| Throughput       | 3,000-4,200 ops/sec                                         | 0.8-2,100 ops/sec                     |
| Worst case delta | Baseline                                                    | **4,753x slower** on some queries     |
| Error messages   | Basic                                                       | Detailed (line/column)                |
| Validation       | Lenient                                                     | Strict OpenCypher grammar             |
| Best for         | **Production**                                              | Development/debugging                 |

The `nornic` parser uses shape-specialized streaming executors for common traversal/aggregation patterns (the "hot-path fast paths"), retaining a general Cypher execution path for coverage.

The Cypher engine currently supports **52+ built-in functions** and **950+ APOC functions** (text, math, collections, ML). Custom `.so` plugins can be dropped into `/app/plugins/`.

**NornicDB-specific Cypher extensions:**

- `WITH EMBEDDING` clause on mutations (inline embedding in the transaction)
- String auto-embedding in `db.index.vector.queryNodes` (pass text, get results)
- `CREATE/ALTER/DROP/SHOW DECAY PROFILE` and promotion policy DDL
- `IS TEMPORAL NO OVERLAP` constraints
- `db.temporal.asOf()`, `db.txlog.entries()`, `db.txlog.byTxId()` procedures

### Vector + Graph Hybrid Design for AI/GraphRAG

This is NornicDB's central differentiator. Rather than bolting a vector store onto a graph database, the design goal is **one execution path for graph traversal, vector retrieval, temporal reads, and audit queries**.

**Embedding model -- two separate fields, two purposes:**

- `Node.ChunkEmbeddings` (`[][]float32`): NornicDB-managed embeddings (first chunk is the "main" embedding) + chunked document embeddings
- `Node.NamedEmbeddings` (`map[string][]float32`): Client-managed vectors (e.g., Qdrant gRPC vectors), keyed by name

These are intentionally separate so managed embeddings and client vectors never overwrite each other.

**Embedding generation -- three modes:**

1. **Background Worker** (async, eventual consistency): scans for unembedded nodes every 15 minutes, chunks at 8192 tokens / 50 overlap, retries with backoff (3x). Debounced triggers on writes.
2. **`WITH EMBEDDING`** (sync, transactional): `CREATE (d:Doc {content: '...'}) WITH EMBEDDING RETURN d` -- embeds inline within the same implicit transaction, rolls back both data and embeddings on failure.
3. **External providers**: Ollama, OpenAI, local GGUF (llama.cpp). Default bundled model is **BGE-M3** (1024 dimensions).

**Vector search strategy selection** (runtime adaptive):

| Strategy                 | When Used                                    |
| ------------------------ | -------------------------------------------- |
| K-means clustered search | When clustering is enabled and clustered     |
| GPU brute-force (exact)  | When GPU enabled, dataset within threshold   |
| CPU brute-force (exact)  | Opt-in via `NORNICDB_VECTOR_CPU_BRUTE_MAX_N` |
| HNSW (ANN)               | Default for CPU-only datasets                |

**Hybrid search** uses Reciprocal Rank Fusion (RRF) combining vector similarity + BM25 full-text:

```bash
curl -X POST http://localhost:7474/nornicdb/search \
  -H "Content-Type: application/json" \
  -d '{"query": "machine learning", "limit": 10, "labels": ["Document", "Memory"]}'
```

Response includes `vector_rank`, `bm25_rank`, `rrf_score`, and per-stage timing metrics.

**GraphRAG pattern:** vector search followed by graph expansion in the same engine:

```cypher
CALL db.index.vector.queryNodes(...) YIELD node
MATCH (node)-[:TRANSLATES_TO]->(t) RETURN ...
```

**HNSW construction acceleration:** BM25-seeded insertion order reduced a 1M embedding build from ~27 min to ~10 min (~2.7x). On Metal hosts, GPU-assisted HNSW construction scores nearest-neighbor candidates in batches while CPU performs final graph mutation and linking.

**Compressed ANN (IVFPQ):** `NORNICDB_VECTOR_ANN_QUALITY=compressed` enables IVF/PQ candidate generation with bounded exact reranking for memory economy. Latency tradeoff: HNSW ~5.7us vs IVFPQ ~23-49us at 1.5K-12K vectors.

### Clustering and Distribution Model

NornicDB implements **three replication modes**, all marked as complete with chaos testing:

| Mode         | Nodes | Consistency           | Use Case                      |
| ------------ | ----- | --------------------- | ----------------------------- |
| Standalone   | 1     | N/A                   | Dev, testing, small workloads |
| Hot Standby  | 2     | Eventual              | Simple HA, fast failover      |
| Raft Cluster | 3-5   | Strong (linearizable) | Production HA                 |
| Multi-Region | 6+    | Configurable          | Global distribution, DR       |

**Hot Standby:** Primary accepts writes and streams WAL to standby. Sync modes: `async` (ack immediately, risk of data loss) or `quorum` (wait for standby apply, strongest). Auto-failover when heartbeats stop for `FAILOVER_TIMEOUT`. Standby is read-only; no automatic write forwarding.

**Raft Cluster:** Full Raft consensus with leader election, log replication, and AppendEntries RPCs. Data is replicated to ALL nodes. **Writes hitting a follower are automatically forwarded to the leader** -- no client-side leader routing needed. Reads can go to any node.

**Multi-Region:** Local Raft clusters per region + async cross-region WAL streaming. Conflict resolution strategies: `last_write_wins`, `first_write_wins`, `manual`.

**Wire protocol:** Length-prefixed gob encoded payloads over TCP (port 7000). Message types include VoteRequest/Response, AppendEntries, WALBatch, Heartbeat, Fence, Promote.

**Chaos testing** simulates: packet loss, high latency (2000ms+), connection drops, data corruption, packet duplication/reordering, and Byzantine failures.

### MVCC Implementation

NornicDB implements **Snapshot Isolation** at the storage layer:

- Each transaction captures a read snapshot at `BeginTransaction()`
- All reads inside the transaction see a consistent committed view + the transaction's own pending changes
- **Read-your-writes**: transactions can see their own uncommitted writes
- **Conflict detection at commit**: concurrent graph mutations against the same logical state fail with a normalized `ErrConflict` instead of silently overwriting newer data
- Uncommitted changes from other transactions are never visible

**Graph snapshot consistency:** A single MVCC Version ID covers all node, edge, and property mutations produced by one committed transaction.

**Version chain model:** Each logical record has a chain of committed versions plus a persisted current head. The current head is always preserved during pruning.

MVCC version records use **Msgpack serialization** on the hot path; head metadata also uses Msgpack. The old `gob` format is deprecated for internal metadata.

### Temporal / Historical Reads

MVCC history enables snapshot-consistent reads, temporal procedures ("what was true at system time T?"), and controlled pruning.

**Cypher temporal procedure -- `db.temporal.asOf`:**

```cypher
CALL db.temporal.asOf(
  'Account',           -- label
  'accountId',         -- keyProp
  'acct-42',           -- keyValue
  'valid_from',        -- validFromProp (business time)
  'valid_to',          -- validToProp (business time)
  datetime('2026-03-01T00:00:00Z')  -- asOf
) YIELD node
RETURN node
```

This is a **bitemporal** query: it answers both **business time** (which version was valid for the domain event) and optionally **system time** (which version was committed at a historical database snapshot) when you pass `systemTime` and `systemSequence`.

**Embedded Go APIs:**

```go
head, _ := engine.GetNodeCurrentHead(nodeID)
node, _ := engine.GetNodeVisibleAt(nodeID, head.Version)
// Also: GetNodesByLabelVisibleAt, GetEdgesByTypeVisibleAt, GetEdgesBetweenVisibleAt
```

**Retention policy:**

- `mvcc_retention_max_versions` (default: 100) -- applies to closed historical versions, not the current head
- `mvcc_retention_ttl` (default: 0 = no age-based protection) -- protects versions newer than `now - TTL`
- Pruning tracks a **Minimum Retained Snapshot (MRS)** per logical key
- Reads below the MRS return `ErrNotFound` (safe, expected after pruning)
- Active snapshot readers prevent the most aggressive tombstone compaction

**Critical caveat:** Search indexing remains **current-only**. Historical MVCC versions are not added to search indexes.

### Canonical Graph Ledger + Mutation Log + Receipts

NornicDB provides a **canonical truth store** pattern combining:

- Declarative constraints (UNIQUE, EXISTS, NODE KEY, type, **temporal no-overlap**, cardinality, endpoint policies, domain/enum)
- Versioned facts with validity windows
- Append-only mutation log (graph events + WAL txlog)
- **Receipts** with `tx_id`, `wal_seq_start`, `wal_seq_end`, `hash` for auditability

**Block-style constraint contracts** group multiple checks:

```cypher
CREATE CONSTRAINT person_contract
FOR (n:Person)
REQUIRE {
  n.id IS UNIQUE
  n.name IS NOT NULL
  n.age IS :: INTEGER
  n.status IN ['active', 'inactive']
  (n.tenant, n.externalId) IS NODE KEY
  COUNT { (n)-[:PRIMARY_EMPLOYER]->(:Company) } <= 1
}
```

### Memory Decay / Knowledge-Layer Scoring

NornicDB implements a **policy-driven memory retention system** based on the Ebbinghaus-Roynard four-layer decomposition.

**Four object kinds in the catalog:**

| Kind                                                                | Has Target? | What It Does                                    |
| ------------------------------------------------------------------- | ----------- | ----------------------------------------------- |
| Decay bundle (`CREATE DECAY PROFILE ... OPTIONS {}`)                | No          | Names a parameter set                           |
| Decay binding (`CREATE DECAY PROFILE ... FOR (...) APPLY {}`)       | Yes         | Activates decay scoring for matched entities    |
| Promotion profile (`CREATE PROMOTION PROFILE ... OPTIONS {}`)       | No          | Names multiplier/floor/cap set                  |
| Promotion policy (`CREATE PROMOTION POLICY ... FOR (...) APPLY {}`) | Yes         | Tracks access and/or applies promotion profiles |

**Decay functions:**

- **Exponential:** `e^(-ln(2)/halfLife * t)`
- **Linear:** `max(0, 1 - t/(2*halfLife))`
- **Step:** `1.0` if `t < halfLife`, else `0.0`
- **None:** Always `1.0`
- **Negative halfLife:** Inverts the curve (consolidation pattern)

**Example:**

```cypher
CREATE DECAY PROFILE working_memory OPTIONS {
  halfLifeSeconds: 604800,
  function: 'exponential',
  visibilityThreshold: 0.10
}

CREATE DECAY PROFILE session_retention
FOR (n:SessionRecord)
APPLY {
  DECAY PROFILE 'working_memory'
  n.tenantId NO DECAY
}

MATCH (n:SessionRecord) WHERE decayScore(n) > 0.5
RETURN n ORDER BY decayScore(n) DESC
```

### GPU Acceleration Details

Multi-backend GPU support for vector operations:

| Backend | Platform       | Performance | Notes              |
| ------- | -------------- | ----------- | ------------------ |
| Metal   | Apple Silicon  | Excellent   | Native M1/M2/M3    |
| CUDA    | NVIDIA         | Highest     | Requires toolkit   |
| OpenCL  | Cross-platform | Good        | Best compatibility |
| Vulkan  | Cross-platform | Good        | Future-proof       |

**Automatic CPU fallback** when GPU is unavailable.

**GPU-accelerated operations:**

- Vector similarity search (10-100x speedup)
- Batch processing
- K-Means clustering
- HNSW construction (Metal implemented; CUDA/Vulkan on roadmap)
- Cosine similarity computation

**Measured Metal GPU boost:** +35-47% on Northwind benchmarks; average ~38% across all queries.

### Performance Benchmarks vs Neo4j

#### LDBC Social Network Benchmark (M3 Max, 64GB)

| Query Type                | NornicDB      | Neo4j       | Speedup |
| ------------------------- | ------------- | ----------- | ------- |
| Message content lookup    | 6,389 ops/sec | 518 ops/sec | **12x** |
| Recent messages (friends) | 2,769 ops/sec | 108 ops/sec | **25x** |
| Avg friends per city      | 4,713 ops/sec | 91 ops/sec  | **52x** |
| Tag co-occurrence         | 2,076 ops/sec | 65 ops/sec  | **32x** |

#### Northwind Benchmark (48 nodes, 56 relationships)

| Metric            | NornicDB (Metal) | Neo4j         | Speedup |
| ----------------- | ---------------- | ------------- | ------- |
| Best query speed  | 4,919 ops/sec    | 2,020 ops/sec | 2.4x    |
| Write operations  | 4,920 ops/sec    | 1,489 ops/sec | 3.3x    |
| Index lookups     | 4,010 ops/sec    | 2,020 ops/sec | 2.0x    |
| Consistency (RME) | +/-0.8-1.8%      | +/-1.4-3.8%   | --      |

**Where Neo4j is competitive:** Full table scans, SUM with GROUP BY (mature aggregation optimizer), and very large graphs (>1M nodes).

#### Hybrid Retrieval Benchmarks (67,280 nodes, 40,921 edges, 67,298 embeddings)

| Workload       | Transport | Throughput   | Mean  | P50   | P95    | P99    |
| -------------- | --------- | ------------ | ----- | ----- | ------ | ------ |
| Vector only    | HTTP      | 19,342 req/s | 511us | 470us | 750us  | 869us  |
| Vector only    | Bolt      | 22,309 req/s | 444us | 428us | 629us  | 814us  |
| Vector + 1 hop | HTTP      | 11,523 req/s | 859us | 699us | 1.54ms | 3.46ms |
| Vector + 1 hop | Bolt      | 13,291 req/s | 747us | 637us | 1.29ms | 3.24ms |

Bolt transport is described as "nearly zero allocation." Multi-hop traversal (depths 1-6) stays at P50 < 541us locally via Bolt.

> **Caveat:** these are NornicDB's own benchmarks on their hardware.

### Storage Engine Internals

**Primary storage: BadgerDB** -- an LSM-tree based key-value store written in Go. Provides streaming iteration, LSM-tree storage with compaction, WAL for durability, and incremental snapshots.

**Async write path:** Configurable with `async_writes_enabled`, `async_flush_interval` (50ms default), node cache (50K), edge cache (100K). Async writes are an eventual-consistency mode.

**Serialization:** `storage_serializer: msgpack` (recommended for new deployments).

**Indexes:**

- **HNSW** vector index (O(log n) search)
- **B-tree** property index
- **BM25** full-text index (token indexing, prefix matching)
- Automatic index selection

### API Surface

| Protocol        | Port     | Status | Notes                                                                           |
| --------------- | -------- | ------ | ------------------------------------------------------------------------------- |
| **Bolt**        | 7687     | Yes    | Neo4j-compatible Cypher queries                                                 |
| **HTTP/REST**   | 7474     | Yes    | Auto-commit transactions, `/db/nornic/tx/commit`, `/nornicdb/search`, `/health` |
| **GraphQL**     | 7474     | Yes    | `POST /graphql` with `search(query, options)`                                   |
| **Qdrant gRPC** | 6334     | Yes    | `Points.Search`, `Points.Query`, `Scroll`, `Count`, `Upsert`                    |
| **Nornic gRPC** | --       | Yes    | `NornicSearch/SearchText` (additive native gRPC)                                |
| **MCP**         | 7474/mcp | Yes    | JSON-RPC, 6 LLM tools: `store`, `recall`, `discover`, `link`, `task`, `tasks`   |

**Qdrant gRPC mapping:**

- Collection -> NornicDB database (namespace)
- Point -> Node with labels `QdrantPoint`, `Point`; payload mapped to `node.Properties`
- Vectors -> `Node.NamedEmbeddings`
- Two embedding ownership modes: NornicDB-managed or client-managed (full Qdrant SDK compatibility)

### Maturity Assessment & Production Readiness

**Strengths:**

- **1,491 commits** across a clearly active project with Coveralls CI
- Neo4j Bolt/Cypher and Qdrant gRPC compatibility validated with e2e tests, migration scripts, and compatibility matrices
- All three replication modes marked complete with chaos testing
- Used in **internal production deployments** (stack consolidation replacing Neo4j + Qdrant + embeddings)
- Academic validation at **UCLouvain** for Cyber-Physical Systems modeling
- Comprehensive documentation (architecture, user guides, performance benchmarks with methodology)
- Docker images for all major architectures (ARM64 Metal, AMD64 CPU/CUDA/Vulkan)
- Homebrew tap available
- MCP server for LLM-native integration (Claude, Cursor)

**Caveats and risks:**

- **Version is 1.2.2** but the system design doc references v0.1.4 -- versioning is inconsistent
- **Small dataset benchmarks:** Northwind has 48 nodes/56 relationships; LDBC results lack full methodology transparency
- The default `nornic` parser is **lenient** and "may accept some invalid syntax" -- correctness risk for production workloads needing strict Cypher compliance
- **Search is current-only** -- no historical/temporal vector search
- **No billion-scale testing** documented; project acknowledges Neo4j is better for "very large graphs (>1M nodes)"
- Async writes are eventual consistency -- separate from MVCC snapshot guarantees
- `RebuildMVCCHeads` at startup is O(N) in versioned records -- material startup cost on large stores
- Single-developer-driven project with limited contributor diversity
- Hot Standby has **no automatic write forwarding** from standby to primary
- Cypher compatibility is described as "minimal or no application changes for supported query shapes" -- implying **not all query shapes are supported**

### NornicDB Verdict

Production-capable for **AI-native workloads** (agent memory, GraphRAG, semantic retrieval + graph traversal) where you control the query patterns and can test compatibility. The consolidation value proposition (replacing Neo4j + Qdrant + embedding pipeline with one system) is compelling. For **general-purpose graph database replacement** of Neo4j in enterprise settings with complex Cypher workloads, the lenient default parser and incomplete query coverage represent meaningful risk. The strongest fit is greenfield AI/GraphRAG projects rather than lift-and-shift migration of existing Neo4j estates.

---

## Non-Go Alternatives for Context

### JanusGraph (the reference point)

| Attribute            | Details                                                                                                                         |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| **Language**         | Java                                                                                                                            |
| **Architecture**     | Distributed -- layered on top of distributed storage systems                                                                    |
| **Query Language**   | Gremlin (Apache TinkerPop -- native integration)                                                                                |
| **Storage Backends** | **Pluggable**: Cassandra, HBase, Google Cloud Bigtable, BerkeleyDB Java, ScyllaDB; 3rd-party: Aerospike, DynamoDB, FoundationDB |
| **Search Backends**  | **Pluggable**: Elasticsearch, Solr, Lucene                                                                                      |
| **Analytics**        | Apache Spark (OLAP)                                                                                                             |
| **Status**           | Active -- Linux Foundation project                                                                                              |
| **License**          | Apache 2.0                                                                                                                      |

### Apache HugeGraph -- closest JanusGraph alternative

| Attribute            | Details                                                   |
| -------------------- | --------------------------------------------------------- |
| **Language**         | Java                                                      |
| **Architecture**     | Distributed -- horizontal scaling, distributed storage    |
| **Query Language**   | Gremlin + Cypher                                          |
| **Storage Backends** | Pluggable backend storage engines                         |
| **Status**           | Active -- First Apache Foundation Top-Level Graph Project |
| **License**          | Apache 2.0                                                |

Distributed, pluggable storage backends, TinkerPop/Gremlin + Cypher support, OLTP + OLAP, Spark/Flink integration. The most comparable to JanusGraph.

### Other notable non-Go options

| Database        | Language    | Architecture                            | Query Language               | Notes                                |
| --------------- | ----------- | --------------------------------------- | ---------------------------- | ------------------------------------ |
| **Neo4j**       | Java        | Single-server (clustered in Enterprise) | Cypher (native)              | Most widely deployed graph DB        |
| **ArangoDB**    | C++         | Distributed cluster                     | AQL                          | Multi-model, has official Go driver  |
| **Memgraph**    | C/C++       | Single-server, in-memory                | openCypher                   | Sub-millisecond multi-hop traversals |
| **Apache AGE**  | C (PG ext.) | Embedded in PostgreSQL                  | openCypher                   | PostgreSQL extension                 |
| **NebulaGraph** | C++         | Distributed, Raft                       | nGQL (openCypher-compatible) | Trillions of edges, Go client exists |

---

## Relevance to go-cqrs-lite

The `graph/` module in go-cqrs-lite is a **projection tier sink** (`GraphProjection` over an in-memory or openCypher driver), not a graph _database_. It provides:

- `GraphSink` / `GraphDriver` / `GraphProjection` interfaces
- `MemoryDriver` (in-process, Go-native)
- `ReadableDriver` (Query/Traverse/Neighbors/ShortestPath -- MemoryDriver only)
- `Schema` (closed-world validation -- catch typos at the sink boundary)

If you need a real distributed graph DB backing it, you'd write a `GraphDriver` adapter for:

- **Dgraph** (gRPC, Go-native) -- best Go-native distributed option
- **NebulaGraph** (Go client exists, C++ server, scales to trillions of edges)
- **NornicDB** (Bolt protocol, Go-native, if GraphRAG is the use case)

Not a Go-rewrite of JanusGraph, which doesn't exist.

---

## Recommendation Matrix

| If you need...                                        | Choose                                                                                             |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Study elegant pluggable-backend architecture          | **Cayley** (read the source, don't deploy it)                                                      |
| Embedded single-node graph in a Go binary, low stakes | **GoraphDB** (prototype only)                                                                      |
| GraphRAG / AI agent memory with Neo4j driver compat   | **NornicDB** (the only credible option here)                                                       |
| Distributed graph at scale with Gremlin               | **None of these Go options** -- use JanusGraph or HugeGraph (Java) or NebulaGraph (C++, Go client) |

**Bottom line:** No Go-native equivalent to JanusGraph exists. JanusGraph's defining features -- distributed + pluggable storage backends + native TinkerPop/Gremlin -- are not reproduced by any single Go project. NornicDB is the strongest _Go-native_ option if your use case leans AI/GraphRAG rather than Gremlin analytics.
