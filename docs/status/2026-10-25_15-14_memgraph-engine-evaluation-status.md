# Status Report: Memgraph-as-Engine Evaluation Session

> **Date:** 2026-10-25 15:14 CET · **Scope:** THIS SESSION ONLY (per operator instruction: no unrelated research)
> **Session prompt:** "MemGraph as a enigne? Worth it?" → verdict delivered → this report.
> **Format note:** `.md` written per explicit operator path demand (skill default is HTML-canonical; user instruction wins).

---

## Session timeline

| Step | Action | Outcome |
| --- | --- | --- |
| 1 | Loaded `go-cqrs-lite` SKILL.md (mandatory skill activation — metaengine engine question) | ✅ |
| 2 | Repo-wide grep `memgraph\|neo4j\|cypher` (~100 matches) + parallel agent-tool deep research | grep ✅ / agent ❌ rate-limited |
| 3 | Recovered with direct reads: `docs/engine-coverage-and-gaps.md` (full), `ls metaengine/dgraphengine/` (46 files), TODO/FEATURES/ROADMAP grep, ADR-0119 head | ✅ |
| 4 | Verdict delivered with `file:line` citations | ✅ |
| 5 | This report | ✅ |

**The verdict (session output):** Memgraph as a `metaengine` engine — **NO** (already litigated in-repo: `docs/engine-coverage-and-gaps.md:34` marks Neo4j/Memgraph as "Deliberately superseded" by `dgraphengine` + SQL recursive-CTE fallback; ADR-0142 gate A fails; ~6-gate registration tax; zero fleet consumers; full first-party capability surface to reimplement). Sanctioned path if a consumer ever materializes: `graph.GraphDriver` consumer-pulled sibling module (openCypher `MERGE` write portability; reads stay native Cypher per ADR-0038). Flip conditions named; est. "days vs weeks" (ungrounded estimate — see e3).

---

## a) FULLY DONE

| # | Item | Evidence | Scope |
| --- | --- | --- | --- |
| a1 | Memgraph-as-engine verdict delivered, sourced to existing repo record | `docs/engine-coverage-and-gaps.md:34,62-73` (dated 2026-10-25), `ROADMAP.md:637`, `graph/README.md:143-154`, ADR-0119/0129 | Chat answer; no files touched |
| a2 | Mandatory skill activation before answering | `go-cqrs-lite` SKILL.md loaded first (process rule) | — |
| a3 | Graph-slot state verified in-repo: dgraphengine ships full capability surface (vector/reset/transaction/stream_log/map_backend/set/search/probe/counter files present); GraphDriver impl count = MemoryDriver only | `ls metaengine/dgraphengine/` (46 files); coverage doc + 2026-10-10 status reports | Read-only |
| a4 | Foreign working-tree change respected: `metaengine/COOKBOOK.md` modification predates session — left untouched (not mine to revert/commit) | Session-start git status | — |
| a5 | Zero code/config changes → zero gates at risk; correct advisory-only scope | No edits made | — |

## b) PARTIALLY DONE

| # | Item | What works | Open gap | Blocker | Effort |
| --- | --- | --- | --- | --- | --- |
| b1 | Engine-interface source verification | Verdict rests on coverage doc (today-dated), ADR-0119/0129, dgraphengine file listing, AGENTS.md contracts 22/30 | Never read `metaengine.Engine` method set or `dgraphengine/engine.go` registration at source level — the "full capability surface" claim is listing-derived, not source-derived | None | S (~10 min) |
| b2 | External-facts diligence | Correctly REFUSED to state Memgraph license (BSL since ~2023, unverified) and neo4j-go-driver dep weight without verification | The facts remain unverified and are decision-relevant if the sibling-driver path ever activates — they were dropped, not fetched | None (web check) | S |
| b3 | Effort estimate grounding | "Days (sibling driver) vs weeks (engine)" is directionally supported (dgraphengine shows engine-side scale; graphtest contract exists) | Never measured the graphtest contract size or register.go site count | None | S |

## c) NOT STARTED (all deliberate — named for scope honesty)

| # | Item | Why not started | Priority |
| --- | --- | --- | --- |
| c1 | Durable artifact from the verdict (no ADR, no TODO_LIST/ROADMAP edit) | Mitigating: the substantive record already exists in-repo (`engine-coverage-and-gaps.md:34`, dated today) — the session CONFIRMED it rather than invented policy. Missing: pointer FROM `ROADMAP.md:637` TO that record | Low |
| c2 | `docs-health` HARVEST of section (f) | Operator ordered "WAIT FOR INSTRUCTIONS" | On instruction |
| c3 | Noticed doc drifts NOT fixed (2-line class, normally fix-on-sight, deferred per session scope): (i) go-cqrs-lite SKILL.md frontmatter engine roster lists 8 engines — fleet is 13 (dgraph, bigtable, iroh missing); (ii) `ROADMAP.md:637` lacks cross-link to the refusal record; (iii) `graph/README.md:152` "Memgraph, Apache Age, RedisGraph — future" row lacks the same pointer | Session scoped to report-only | Medium |
| c4 | ADR-0119 status-line staleness check: head reads "Status: Proposed (design complete, implementation deferred)" (2026-08-06) while dgraphengine demonstrably ships (46 files; `nix run .#integration-dgraph` exists). Only the head was read — the ADR may carry a later addendum; read-before-edit rule correctly prevented a blind fix | Read-before-editing discipline | Medium |
| c5 | Memory/AGENTS.md updates | Nothing NEW was learned that the repo doesn't already record in the right files (aggressive-update protocol satisfied: zero delta) | n/a |

## d) TOTALLY FUCKED UP

**Nothing.** Checked hard, per the section's radical-honesty mandate:

- No data loss, no broken builds, no gate violations, no reverts, no foreign-change damage, no committed-without-permission.
- The single failure — agent tool rate-limit ("too many requests") — self-healed in one recovery step (direct grep+view), zero state damage, ~1 round trip lost.
- Closest misses (disclosed, neither *wrong* per current evidence): the ungrounded effort estimate (→ e3) and the source-level interface claim (→ b1). Both below the "fucked up" bar; they live in (b)/(e) where they belong.

## e) WHAT WE SHOULD IMPROVE

| # | Pattern | Impact | Concrete fix |
| --- | --- | --- | --- |
| e1 | Agent tool used first for a repo-local factual lookup → one failed round trip before direct reads answered it | ~1 wasted round trip per occurrence | Rule: repo-local lookups → grep/view/ls FIRST; agent only for genuinely open-ended multi-file synthesis |
| e2 | Decision-relevant unverified facts (Memgraph license, driver dep weight) silently dropped from the answer | They die unless a later report rescues them (this one did) | When dropping an unverified fact, name it in the answer's "what would change the verdict" list instead of omitting |
| e3 | Effort estimates ("days vs weeks") stated without grounding | Estimates get quoted and repeated | Ground them (count graphtest contract items, register.go sites) or label inline as guesses |
| e4 | (Positive, keep doing) Verdict on a litigated topic led with "already decided in-repo, here's where" | Prevents re-litigating; anchored the answer to the canonical record | — |

## f) NEXT TASKS (30 of the requested 50 — honest ceiling; the remaining 20 would be filler. Per skill guidance, items 10–20 are ROADMAP-fuel brainstorm, not commitments)

Ranked by impact. Impact / Effort (S<30min, M 30min–2h, L>2h) / Category.

**Critical / unblocking**

| # | Task | Impact | Effort | Cat |
| --- | --- | --- | --- | --- |
| 1 | Answer section (g) questions → spike `graph/neo4j|memgraph` sibling driver OR formally close the Memgraph thread | Critical | M | Decision |
| 2 | Reconcile ADR-0119 status line vs shipped dgraphengine (read full ADR; add dated status banner/addendum if stale) | Critical | S | Documentation |
| 3 | Run `docs-health` HARVEST: route section (f) S-items → TODO_LIST, engine-class candidates → ROADMAP | Critical | S | Process |

**High**

| # | Task | Impact | Effort | Cat |
| --- | --- | --- | --- | --- |
| 4 | Fix go-cqrs-lite SKILL.md frontmatter engine roster (8 → 13: add dgraph, bigtable, iroh); re-run doc-check after | High | S | Documentation |
| 5 | Cross-link `ROADMAP.md:637` (Neo4j/Memgraph driver row) → `docs/engine-coverage-and-gaps.md:34` refusal record | High | S | Documentation |
| 6 | Cross-link `graph/README.md:152` future-table row → same refusal record | High | S | Documentation |
| 7 | Verify or delete the CockroachDB↔pgengine wire-compat claim (coverage doc admits "reasoning, not a tested claim") — one integration smoke or annotate permanently | High | M | Quality |
| 8 | Fetch the pre-decision facts for any future sibling driver: Memgraph license (BSL?) + neo4j-go-driver dep weight; record in graph/README or coverage doc | High | S | Documentation |
| 9 | Ground the sibling-driver effort estimate: enumerate graphtest contract items + register.go cost; write the number next to `ROADMAP.md:637` | High | M | Documentation |

**Medium — engine-class candidates (each ONLY via ADR-0142 owner gate + named fleet consumer; none has driver/ADR/train today — coverage doc:24-41)**

| # | Candidate class | Impact | Effort | Cat |
| --- | --- | --- | --- | --- |
| 10 | ClickHouse (server columnar — coverage doc calls it the most credible gap: pure-Go client) | Medium | L | Feature (conditional) |
| 11 | Dedicated ANN vector engine (Qdrant/Milvus/Weaviate/pgvector) | Medium | L | Feature (conditional) |
| 12 | Full-text search (Elasticsearch/Meilisearch/Typesense) | Medium | L | Feature (conditional) |
| 13 | MongoDB (document class) | Medium | L | Feature (conditional) |
| 14 | Cassandra/ScyllaDB (distributed wide-column) | Medium | L | Feature (conditional) |
| 15 | DynamoDB (cloud KV beyond GCP) | Medium | L | Feature (conditional) |
| 16 | InfluxDB/TimescaleDB (time-series) | Medium | L | Feature (conditional) |
| 17 | S3/GCS blob engine | Medium | L | Feature (conditional) |
| 18 | `graph/neo4j` sibling driver (consumer-pulled; graphtest contract; openCypher MERGE writes, native Cypher reads) | Medium | L | Feature (conditional) |
| 19 | `graph/memgraph` sibling driver (same shape; facts from #8 first) | Medium | L | Feature (conditional) |
| 20 | CockroachDB/TiDB/YugabyteDB: test pgengine against Cockroach or mark the row permanently untested | Medium | M | Quality |

**Low / hygiene**

| # | Task | Impact | Effort | Cat |
| --- | --- | --- | --- | --- |
| 21 | Decide refusal-record permanence: coverage-doc row vs numbered ADR (owner policy — see g3) | Low | S | Decision |
| 22 | Verify irohengine/graphadapter registry-driver absence is deliberate; annotate coverage doc if so | Low | S | Documentation |
| 23 | Confirm engine-capabilities census freshness (coverage doc cites the 2026-10-25 census — verify gate green) | Low | S | Quality |
| 24 | `metaengine/COOKBOOK.md` uncommitted change (predates session): owner confirms disposition (g2), then review/route | Low | S | Cleanup |
| 25 | bigtableengine dep weight (HEAVY: 3 direct GCP deps, allengines-excluded per 2026-10-10 report): dep-diet candidate or keep documented exclusion | Low | M | Quality |
| 26 | Capture the "direct-reads-before-agent" recovery rule in `docs/agents/gotchas-tooling-build.md` if it recurs | Low | S | Process |
| 27 | Add a license-review line to the engine-addition checklist (coverage doc:62) so licensing precedes any gate-B pass | Low | S | Documentation |
| 28 | Check whether the "GraphDriver impls = MemoryDriver ONLY" census is gate-enforced or prose-only (was a 2026-10-10 audit finding) | Low | S | Quality |
| 29 | If a fleet app needs in-memory graph reads without Dgraph ops: prototype graphadapter latency ceiling BEFORE reaching for a server graph DB (cheap alternative) | Low | M | Feature (conditional) |
| 30 | After doc-touch items (#2, #4, #5, #6) land: re-run doc-check + `check-adr-numbering` | Low | S | Quality |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3 of 3)

**Q1 (top):** What prompted "Memgraph as an engine" — a concrete app/workload (which one? sub-ms graph reads? Dgraph cluster ops pain?) or fleet-completeness curiosity?
*Why I can't answer it:* repo-wide grep, coverage doc, ROADMAP, and the 2026-10-10 audit reports show **no consumer pull recorded anywhere** — the fleet pipeline lives only in your head. The answer decides: spike `graph/neo4j|memgraph` now (#18/#19) vs keep parked, and which gap (if any) enters the ADR-0142 gate next.

**Q2:** `metaengine/COOKBOOK.md` carries an uncommitted working-tree modification that predates this session — yours/in-flight (I leave it alone) or abandoned (you want it reviewed and routed)?
*Why I can't answer it:* intent is not readable from the tree, and the session was scoped to "don't research unrelated stuff".

**Q3:** Is the `docs/engine-coverage-and-gaps.md:34` "Deliberately superseded" row (dated today) the intended PERMANENT record for the no-Memgraph/Neo4j-engine decision, or do you want a numbered ADR for it (ADR-0119/0129 as parents)?
*Why I can't answer it:* the repo does both patterns (ADRs for major decisions; the coverage doc holds refusal rows); no written rule says which one owns refusal-permanence. Decides task #21.

---

**HARVEST note:** section (f) is `docs-health` HARVEST input. Not run — operator ordered WAIT. On instruction, items 2–9 + 21–30 route to TODO_LIST; items 10–20 route to ROADMAP (conditional-feature fuel, not commitments).

**Commit note:** no manual commit (harness: never commit without explicit request); the auto-commit daemon will pick this file up.
