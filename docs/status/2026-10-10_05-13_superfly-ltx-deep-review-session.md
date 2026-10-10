# Status Report — superfly/ltx Deep-Review Session (read-only)

**Date:** 2026-10-10 05:13 CEST
**Session type:** External-library research + architectural review. ZERO go-cqrs-lite code changes.
**Scope guard:** This report covers ONLY what this session did and noticed. Per instruction: no unrelated research.

## Session facts

- Task: "Review https://github.com/superfly/ltx for go-cqrs-lite deeply!"
- Clone: `github.com/superfly/ltx` @ `f299ac5` (shallow, depth 50) → `/tmp/ltx-review`
- LTX is Superfly's page-level SQLite transaction-shipping format (LiteFS/Litestream substrate):
  append-only TXID-ranged segments, pre/post-apply CRC64 checksum chain, structural snapshots,
  k-way merge compaction, LZ4 block compression, spill-to-disk encoder index.
- Deliverable produced: in-chat review (7 steal-ideas, 4 avoid-items, relevance matrix,
  2 concrete adoption candidates). NOT yet persisted to any repo doc.

---

## Session self-critique (the three questions asked)

### What did you forget?

1. **Did not read the target surfaces before recommending against them.** The two
   "concrete candidates" (segment-level checksum chain; integrity-bearing resumption tokens
   for `watermill.CatchUpSubscriber`/SSE) were grounded in LTX's design, but I never opened
   `watermill/`'s checkpoint store or `metaengine.ServeSSE`'s `Last-Event-ID` handling to
   confirm the gap actually exists in our code. The recommendation is hypothesis, not verified gap.
2. **Never ran the ltx test suite** (`go test ./...` in the clone — ~30s). I called the
   `pageSeq` validation trick "elegant" from reading production code only; the tests that pin
   that behavior went unread (~2.3k lines of test files: 0% read).
3. **Never opened LICENSE** — irrelevant for stealing _ideas_, mandatory before anyone
   copies _code_. Should have been a 5-second read, flagged either way.
4. **Partial coverage I presented as full**: "production core read 100%" is true, but CLI was
   1/8 files (only `apply.go`), `internal/`, `file_spec.go` unread.
5. **Noticed-then-dropped observations**: `FileInfo.Level` (leveled/LSM-style compaction
   implication) and `NodeID` zeroed-on-compaction (a direct cross-reference to our
   `id.ActorID` merge-provenance semantics) — both read, neither made the delivered review.
6. **No todos tracking** for a multi-phase session (skill → clone → read → map → deliver →
   persist). The omission is exactly why items 1–5 slipped.
7. **`id/derive.go` surfaced in my own grep** (hashing already lives in `id/`) and I did not
   read it before proposing new checksum work — risks reinventing what exists.

### What could you have done better?

1. **Verification order**: clone → tests → LICENSE → production code → consumer repos
   (LiteFS/Litestream) → then write conclusions. I did step 3 first, steps 1–2 never.
2. **Kill the first tool call faster**: GitHub HTML fetch returned nav-garbage; cloning
   should have been the opening move. One wasted round, self-corrected.
3. **Label claim strength**: "no payload compression in go-cqrs-lite" was grep-level verified
   (4 algo names, production files) — defensible but I stated it flat. Should have said
   "grep-level: lz4/zstd/snappy/flate absent from production storage code".
4. **Persist as you go**: the review exists only in chat. This repo has an established pattern
   (`docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`)
   for exactly this kind of external-mapping analysis. Session-end persistence = the analysis
   dies with the context window.

### What could you still improve?

1. Verify or retract the scale claim (see (d)); verify or retract the two adoption candidates
   against the real target code (see (f) #1–#3).
2. Turn the surviving findings into a persisted doc + (if warranted) an ADR-grade proposal
   with a Tier landing-zone analysis (`record/` Tier 0 vs `event/` Tier 1).
3. Internalize: **verify-external-claims applies to my own drafts too** — "validated at
   fly.io scale" was exactly the trophy-case phrasing that skill exists to kill.

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                 | Evidence                                                 |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| 1 | Loaded mandatory `go-cqrs-lite` skill before task work                                                                                                                                                                                                                                                                               | SKILL.md read in first tool call                         |
| 2 | Cloned superfly/ltx @ f299ac5                                                                                                                                                                                                                                                                                                        | `/tmp/ltx-review`, git log inspected (15 recent commits) |
| 3 | Read 100% of LTX production core: `ltx.go` (642L), `encoder.go` (560L), `decoder.go` (574L), `checksum.go` (188L), `compactor.go` (263L)                                                                                                                                                                                             | all five files viewed end-to-end this session            |
| 4 | Read README format spec v3 + CLAUDE.md + go.mod (Go 1.24, single dep `pierrec/lz4/v4`)                                                                                                                                                                                                                                               | in-session                                               |
| 5 | Read `cmd/ltx/apply.go` (apply semantics: pre/post-apply verification loop) + `IsContiguous`                                                                                                                                                                                                                                         | in-session                                               |
| 6 | Surveyed go-cqrs-lite counterpart surfaces: `signing/` (per-event HMAC/Ed25519/COSE), `event` Version/SchemaVersion + AppendBatch, `snapshot/` exists, `Journal`/`SeekableJournal`/`StreamingJournal` interfaces, compression grep (absent in production storage code), hash-usage grep (`signing/`, `encryption/hkdf`, `id/derive`) | greps + targeted views, in-session                       |
| 7 | Delivered structured review with relevance matrix + verdicts                                                                                                                                                                                                                                                                         | in-chat deliverable (this session's only output)         |

## b) PARTIALLY DONE

| # | What works now                                                                                   | What remains open                                                                                                                            | Effort |
| - | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | LTX review delivered with 7 steal-patterns + 4 anti-patterns                                     | NOT persisted to repo docs; no commit hash, no durable artifact                                                                              | S      |
| 2 | Two adoption candidates identified (segment checksum chain; integrity-bearing resumption tokens) | Neither verified against actual `watermill.CatchUpSubscriber` checkpoint code or `ServeSSE` replay code — gap is hypothesized, not confirmed | S–M    |
| 3 | Claim "no journal segment merge in go-cqrs-lite"                                                 | Asserted from module-map knowledge; snapshot/ and storage journal internals not read this session to confirm                                 | S      |
| 4 | LTX repo characterized (health, history, deps)                                                   | Tests never run; consumer repos (LiteFS/Litestream) never opened; LICENSE never read                                                         | S      |

## c) NOT STARTED

| # | Planned                                                                                                                              | Why not started                                                             | Priority         |
| - | ------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- | ---------------- |
| 1 | Persist review to `docs/architecture-understanding/`                                                                                 | Session ended at chat deliverable                                           | High             |
| 2 | ADR-grade proposal for segment-level journal checksum chain                                                                          | Blocked on (b)#2 verification + threat-model decision (see g)               | Medium           |
| 3 | Verification of LTX's production-scale claim via LiteFS/Litestream repos                                                             | Not attempted; relied on code-comment citations (litestream #1477, ltx #96) | High for honesty |
| 4 | Benchmarks quantifying LTX patterns (4x memory claim, spill threshold) — LTX's numbers are their code comments, not our measurements | Out of scope for a review read                                              | Low              |
| 5 | Landing-zone design (Tier 0 `record/` vs Tier 1 `event/` vs per-engine) for any adopted pattern                                      | Premature before (b)#2                                                      | Medium           |

## d) TOTALLY FUCKED UP

**Severity: LOW (reputational/epistemic, not operational — no code touched, nothing broken).**

| # | What                                             | Specifics                                                                                                                                                                                                                                                                                                                                        | Mitigation                                                                                 |
| - | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| 1 | **Unverified external claim in the deliverable** | Final line: "validated by LTX running this design in production at fly.io scale." I never verified fly.io deployment scale in-session. Only real corroboration: LTX code comments cite litestream #1477 / ltx #96 (proves Litestream-adjacent production use, NOT "fly.io scale"). This is precisely the `verify-external-claims` failure class. | Read LiteFS/Litestream source or docs; weaken wording to code-comment-cited production use |
| 2 | **Recommendations without target verification**  | "Consider: integrity-bearing resumption tokens" — delivered before reading our own checkpoint/SSE replay code. If integrity already exists there, the recommendation is noise.                                                                                                                                                                   | Read `watermill/` checkpoint + `metaengine.ServeSSE` (f #2–#3)                             |
| 3 | Wasted first tool call on GitHub HTML fetch      | Returned navigation chrome only                                                                                                                                                                                                                                                                                                                  | Clone-first heuristic adopted                                                              |

## e) WHAT WE SHOULD IMPROVE

| # | Pattern                                                   | Impact                                               | Fix                                                                                                 |
| - | --------------------------------------------------------- | ---------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| 1 | External reviews die in chat                              | High — analysis lost with context window             | Persist every external-review session to `docs/architecture-understanding/<date>_*.md` same-session |
| 2 | Review-order indiscipline                                 | Medium — unverified claims ship                      | Fixed order: clone → test run → LICENSE → code → consumers → write                                  |
| 3 | verify-external-claims blind spot on self-authored claims | Medium — trophy-case phrasing slipped through        | Treat my own "production scale"-class phrasings as claims requiring in-session evidence             |
| 4 | Recommendation-before-target-read                         | Medium — half-work presented as done                 | Read the go-cqrs-lite target surface BEFORE writing any "consider adopting" row                     |
| 5 | No todos in multi-phase research                          | Low–Medium — silent coverage gaps (see forgot #4–#5) | Always todos for 4+ phase sessions                                                                  |
| 6 | Claim-strength labeling                                   | Low — "no compression" stated flat                   | Tag claims: verified-by-read / grep-level / prior-knowledge                                         |

## f) Next tasks (ranked; feeds docs-health HARVEST)

| #  | Task                                                                                                                                                      | Impact         | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- | ------ | ------------- |
| 1  | Read `watermill.CatchUpSubscriber` + its checkpoint store; verify whether resumption tokens carry any integrity today                                     | High           | S–M    | Research      |
| 2  | Read `metaengine.ServeSSE` `Last-Event-ID`/`SSEReplay` handling; same integrity question                                                                  | High           | S      | Research      |
| 3  | Read `id/derive.go` — determine what hashing/derivation already exists before designing ANY checksum chain (do not reinvent)                              | High           | S      | Research      |
| 4  | Run ltx test suite in clone (`cd /tmp/ltx-review && go test ./...`), record pass/fail @ f299ac5                                                           | Medium         | S      | Verification  |
| 5  | Read ltx LICENSE; note code-copy vs idea-copy boundary                                                                                                    | Medium         | S      | Verification  |
| 6  | Verify LiteFS/Litestream actually consume LTX (their repos/docs); correct or confirm the scale claim                                                      | High (honesty) | S–M    | Verification  |
| 7  | Persist this session's LTX review as `docs/architecture-understanding/2026-10-10_superfly-ltx-mapping.md` (pin commit f299ac5)                            | High           | S      | Documentation |
| 8  | If (1)–(3) confirm gaps: draft ADR "journal segment integrity chain (LTX pre/post-apply analog)" with Tier landing-zone analysis                          | High           | M      | Design        |
| 9  | If adopted: decide CRC64-chain (cheap, non-crypto) vs HMAC-chain (rides existing `signing/`) — needs threat-model answer (see g-Q2)                       | High           | S      | Design        |
| 10 | Audit `storage/sql` batch-insert paths against LTX's encoder-abort-FSM pattern (no half-written file can ever be "finished" valid)                        | Medium         | M      | Quality       |
| 11 | Evaluate `pageSeq`-style "validate index against stream without retaining either" for large `ReadStreamFrom`/`StreamingJournal` paths                     | Medium         | M      | Feature       |
| 12 | Audit our snapshot invariants: structural (LTX `IsSnapshot()==MinTXID==1`) vs flag/metadata-based; report drift from ADR-0114 philosophy                  | Medium         | M      | Quality       |
| 13 | Non-zero-guarantee trick (LTX `ChecksumFlag 1<<63`): apply to any zero-able integrity/position fields we introduce                                        | Low            | S      | Design        |
| 14 | Commutative XOR-fold checksum for parallel large-journal verification (LTX `ChecksumPages`, 24 workers) — only if journals get integrity work             | Medium         | M      | Feature       |
| 15 | Add `NodeID`-zeroed-on-compaction ↔ `id.ActorID` merge-provenance cross-reference note to the ADR from (8)                                                | Low            | S      | Documentation |
| 16 | Capture dropped observations (`FileInfo.Level` leveled compaction; WAL salt/offset provenance) in the persisted doc from (7)                              | Low            | S      | Documentation |
| 17 | Feasibility note: per-record LZ4 _block_ compression for journal payloads — ONLY with logical-checksum-before-compress rule; likely park for v5+          | Low            | M      | Research      |
| 18 | Confirm "no compression" claim properly (codec options, `stack` presets, kv/snapshot codec knobs) and record as verified fact                             | Low            | S      | Verification  |
| 19 | If checksum chain adopted: extend `benchkit`/`cqrs-bench` with integrity-on/off phases so cost is measured, not assumed                                   | Medium         | M      | Quality       |
| 20 | Write the "review-order" discipline (e#2) into `docs/agents/gotchas-*.md` or the go-cqrs-lite skill references if it recurs in a second session           | Low            | S      | Documentation |
| 21 | Clean up `/tmp/ltx-review` once items 4–5 are done (trash, not rm)                                                                                        | Low            | S      | Cleanup       |
| 22 | If (1)–(2) show SSE replay integrity matters: consider joint design for `ServeSSE` replay + CatchUp checkpoints (one integrity-token type, two consumers) | Medium         | M      | Design        |
| 23 | Check whether `errorfamily` has a Corruption-family code ready for checksum-chain mismatch errors before ADR (8) names one                                | Low            | S      | Research      |

_Items 24–50 deliberately not padded — 23 real items > 50 filled ones. ROADMAP-fuel items (11, 14, 17) should route via docs-health HARVEST, not TODO_LIST._

## g) Questions I cannot answer myself

1. **Persistence + next step:** Should the LTX review become a persisted
   `docs/architecture-understanding/` doc (cordis-mapping pattern) now, and should the two
   candidates go to ADR drafting immediately or wait behind the live adoption wave
   (T01–T13, tag-wave 4/5 per yesterday's status reports)? I cannot rank cross-session priorities myself.
2. **Threat model:** Is a non-cryptographic CRC64 rolling chain (LTX-style: detects
   corruption/tampering transit at near-zero cost, forgeable by a determined writer) an
   acceptable integrity tier for your fleet's journals — or must integrity always ride the
   existing cryptographic `signing/` (Ed25519/HMAC), making the cheap chain redundant?
   This decides candidate (a)'s entire shape. I tried to infer from `signing/doc.go` presence
   alone; posture is a product decision.
3. **Integration appetite:** LTX's deepest idea is physical page-shipping replication
   (single-writer TXID + state checksums). Is go-cqrs-lite ever intended to grow a
   physical-replication story (e.g., replicating whole engine snapshots between nodes), or is
   logical event-stream replication (watermill/brokers) the forever-answer? Determines whether
   LTX's compaction/segment machinery is relevant at all or just its checksum/FM patterns.

---

_Point-in-time snapshot. Section (f) is docs-health HARVEST input. Auto-commit daemon will absorb this file; no manual commit per harness contract._
