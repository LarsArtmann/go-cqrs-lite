# Docs-Health AUDIT Execution — Living-Docs Truth Landed, Waves D–F Mid-Flight (2026-08-16 14:24)

Session executing
[`docs/planning/2026-08-16_13-40_SUPERB-DOCS-HEALTH-ANNOTATE-ARCHIVE-EXECUTION.md`](../planning/2026-08-16_13-40_SUPERB-DOCS-HEALTH-ANNOTATE-ARCHIVE-EXECUTION.md)
(Waves A–G, 59 fine tasks), resuming after the interrupted 13:36 audit. A
concurrent session is active in this repo the whole time (its seq-carrying
journal-reads work landed mid-run as `a1334d8c5`; its KV/LSM recalibration
report appeared at 14:09).

---

## a) FULLY DONE

1. **Commit 1 — pending harvest preserved** (`73ce6e1b7`): TODO_LIST.md (7
   edits, ~18 open items), the 13:36 in-flight status report, and the 13:40
   execution plan. Explicit paths only; foreign `metaengine/*` WIP untouched.
2. **Wave A — CHANGELOG truth** (`aa00b3ae6`), all claims verified against
   tag trees first (symbol content checks + `git merge-base --is-ancestor`,
   never memory):
   - New combined `## [2026-08-16 module releases]` section covering all **22
     tags** (enumerated with `for-each-ref` by creatordate): id, record,
     metadata, schema, event, command, query, middleware, watermill,
     metaengine + 8 engines, storage ×2 — per-module Added/Fixed entries with
     the real API deltas (verified via tag-pair exported-symbol diffs).
   - The two missing **retract entries** (command/v4.7.0, query/v4.6.0) with
     the verified root cause: tag go.mod pinned `metadata/v4 v4.4.0` while
     `command.Metadata`/`query.Metadata` alias `metadata.Metadata[K]`
     (v4.5.0-only) → `undefined` under `GOWORK=off`. Retract directives
     confirmed live inside the v4.7.1/v4.6.1 tag trees.
   - Dedupe: the duplicated Unreleased storage detail (keyset ~285x,
     packet-safe chunking, deserialize fast path) moved INTO the
     `storage/v4.7.0` section as `### Detail` subsections — single source of
     truth; the released workloadMeter-pad bullet moved to the metaengine
     v4.11.0 entry (pad commit verified ancestor of the tag).
   - Pragmatic `[Unreleased]` fix: a **scope note** instead of physically
     re-homing 84 dated subsections — 08-10→08-15 sections for chained
     modules shipped in the tags; unchained modules + wave-3/4 stay
     unreleased. (NO VERSCHLIMMBESSERUNG; full re-homing deferred — Q1.)
3. **Wave B — FEATURES truth** (16 new/updated rows; landed inside daemon
   commit `a1334d8c5` after it raced my commit): every symbol grep-verified
   in production code BEFORE writing (`AdoptedPayload` resolved to
   `event.ReconstructEventWithAdoptedPayload`). Rows: event actor
   propagation + store/journal transforms + reconstruction variants,
   `query.AsRecord`, projectionhost checkpoint batching, metaengine
   CapabilityAudit + `WithIdempotencyCapacity` (semantics read from source),
   pgengine COPY append, SQL keyset/packet-safe/`file::memory:` pin,
   bbolt group commit, watermill CatchUpSubscriber replay-handoff fix,
   api-stability pin-drift guard, stack/pebble operator knobs (matrix row).
4. **Wave C — ROADMAP + AGENTS** (ROADMAP rode daemon `a6660bfd1`; AGENTS in
   `6046ba0aa`):
   - ROADMAP header: "v4.7.0 tagged (2026-08-10)" → the 22-tag chain + 3
     same-day retracts; remaining Unreleased window correctly scoped to
     wave-3/4. Release History row added. Open Q1 rewritten (tag
     authorization RESOLVED by the executed chain; open residue = wave-4 tag
     batch). Q2 (Go 1.26.6) deleted — resolved by `ea8fa5072`; renumbered
     2-7. Stale metaengine "Remaining" list refreshed (calibration,
     seq-carrying reads, DuckDB float guard all shipped; what remains is
     iroh WriteOp convergence + conformance wiring).
   - AGENTS +2 gotchas: **local-path `replace` hygiene** (rule derived from
     `ceb88738b`: relative sibling paths for unpublished symbols only —
     pre-tag sweep included) and **modernc `file::memory:` per-connection
     DBs** (pin pool to one connection; regression test named). 82 go.mod
     count + all referenced paths re-verified.
5. **Wave D — 03-18 planning doc annotated** (`7d96181f1`): all **87 items**
   (T1–T27 + F1–F60 + 11 Step-1 prose items) verdicted inline — strikethrough
   + `done at <hash>` for 85, **2 OPEN** (F26 pebble/bbolt pin bump, F56
   singleflight leader-ctx, both already TODO_LIST-tracked). Tables got skill
   -sanctioned Status columns.
6. **Live defect discovered + harvested** (the audit doing its job): **storage
   /pebble and storage/bbolt standalone builds are RED** — both pin
   `event/v4 v4.6.0` but `serialization.go` calls
   `event.ReconstructEventWithAdoptedPayload` (intro `5b8a9a615`, verified NOT
   in `event/v4.7.0` — unpublished). `GOWORK=off go build` fails with
   `undefined:`; the workspace masked it — the exact command/v4.7.0 incident
   class. New 🔥🔥 TODO_LIST item with fix guidance (sibling `../event`
   replace until tagged).
7. **Wave E started** (skill format, inline, never renumber):
   - 01-33: f.15–f.30 verdicted (14 strikethroughs; f.27 genproto-comment +
     f.30 milestones entry left open; f.1–f.14 already resolved by §h/§i/§j —
     not re-marked per plan).
   - 02-11: NEXT STEPS 1/2/4 done (verify #4 GREEN, tags cut); item 3
     (size-stable KV/LSM benches) left OPEN — the concurrent session is
     actively mid-flight on it (14-09 report).

## b) PARTIALLY DONE

1. **02-16 fully read, not yet annotated** — its f-list (25 items) maps
   cleanly: f.1–f.11 = the executed 22-tag chain (done), f.12–f.25 = T2–T17
   opens (live TODO_LIST items, stay untouched).
2. **Concurrent-session coordination worked but cost attribution**: three
   daemon/sibling commits absorbed my finished work mid-flight
   (`a1334d8c5`: FEATURES rows; `a6660bfd1`: ROADMAP). Content verified in
   HEAD after each race — nothing lost, but per-wave commit messages of mine
   only captured 2 of 4 waves.

## c) NOT STARTED

- Wave E remainder: 03-10 (f.1–f.50), 03-44 (§h exists), 04-00 (f.1–f.50),
  04-24 (e.4 50-row table + §g).
- Wave F: 07-12, 09-13 (+ stale "10 engines green" iroh claim inline-fix),
  10-51, 11-00, 11-33, 12-39, 13-15 (minimal).
- Wave G: classify/archive, doc-check gate, cross-file consistency sweep,
  health report (Accuracy + Fitness, inline), final commit + push.
- Out of scope by standing decisions: 08:19 HTML report, benchmark evidence
  doc, the three new 14:0x concurrent-session reports.

## d) TOTALLY FUCKED UP (honest ledger)

1. **Multiedit-before-View twice** (FEATURES.md, AGENTS.md): tool rejected
   the edit because I'd only read via bash sed/grep this session. Two wasted
   round trips re-learning the exact rule the 02-16 report already logged
   five times.
2. **CHANGELOG surgery: silent no-op replace** — I removed a block via
   `text.replace(reconstructed_string, "")`, but the reconstruction (header
   stripped, bullets re-joined) no longer matched the file; the replace
   silently did nothing and briefly left the doc with the duplication it was
   supposed to kill. Caught by post-edit seam verification; fixed with
   header-boundary index slicing. Lesson: never replace() a string you
   derived by editing — slice by anchors that exist verbatim.
3. **03-18 annotation script produced malformed tables on first pass** —
   verdicts merged into the last cell (I stripped the closing `|` and never
   re-added the separator), and the T-regex also matched F-table header rows
   (`| # |`), doubling the Status columns. Two fix passes; final state
   verified (0 rows missing verdict cells).
4. **Committed a broken intermediate state of the 03-18 tables?** No — the
   fixes landed before the Wave D commit; `7d96181f1` contains only the
   well-formed tables. (Listed because the daemon could have raced the
   broken intermediate — it did not.)

## e) WHAT WE SHOULD IMPROVE

1. **Race the daemon or pre-write messages**: with the auto-commit daemon +
   concurrent session active, my per-wave commits will keep losing files to
   sibling commits. Better: `git add <paths> && git commit` immediately after
   each wave's last edit, or accept attribution loss and only verify presence
   in HEAD.
2. **Annotate-by-script needs a table-lint step**: any scripted table edit
   should end with a structural check (column count per row, verdict-cell
   presence) BEFORE writing the file — not after.
3. **Derived-string replaces are forbidden** (d.2) — anchor-slice or exact
   verbatim copy only.
4. Minor: 01-33's two open micro-nits (f.27 genproto go.mod comment, f.30
   SESSION_MILESTONES entry) were left in place but NOT added to TODO_LIST —
   they are one-liners; harvest-or-do decision pending (f.6 below).

## f) NEXT — in order

1. Annotate 02-16 (read; f.1–f.11 done via chain, f.12–f.25 open TODO refs).
2. Annotate 03-10, 03-44 (respect existing §h), 04-00, 04-24 (e.4 table →
   Status column pattern from 03-18).
3. Wave F: 07-12, 09-13 (incl. inline correction of the stale "10 engines
   green" claim — iroh was RED), 10-51, 11-00, 11-33, 12-39, 13-15 minimal.
4. Decide f.27/f.30 micro-nits: do them (2 one-line edits) or leave open.
5. Wave G classify: which files have ZERO open items after annotation →
   `git mv` to `docs/status/archived/` (strict default says possibly none;
   see Q3).
6. doc-check gate (`cmd/doc-check` over SKILL.md + references + AGENTS.md) —
   must cover the new FEATURES/CHANGELOG claims that reference symbols.
7. Cross-file consistency sweep: TODO_LIST ↔ CHANGELOG duplicates, link
   integrity (archived paths!), status conflicts between living docs.
8. Health report inline (Accuracy + Fitness with visible math) — NEVER a file.
9. Final commit (explicit paths) + push (authorized by the plan).
10. Post-push: nothing else — stop. The v5-cut CHANGELOG re-homing and the
    pebble/bbolt pin fix are deliberately OUT of this docs session.

## g) QUESTIONS (max 3)

1. **CHANGELOG `[Unreleased]` bulk (Q from Wave A):** it still carries ~1,900
   lines of dated 08-10→08-15 sections for work that shipped in the 08-16
   chain (I added a scope note instead of moving them — verslimmbesserung
   guard). Physically re-home all 84 subsections into dated release sections
   now (≈1h scripted surgery, history-stable), or keep the note and fold at
   the v5 cut?
2. **pebble/bbolt standalone-RED (found during annotation):** fix inside this
   docs session (add sibling `../event` replaces per documented convention —
   2-line go.mod change, restores standalone builds) or leave strictly to a
   code session? The TODO item is written either way.
3. **Archive threshold:** strict skill default (archive only when EVERY item
   resolves) would archive ZERO of the 16 files — each retains at least one
   open/foreign-tracked item. Is "resolved + tiny-nits + owned-elsewhere
   (TODO_LIST/concurrent session)" sufficient to archive the fully-triaged
   ones (e.g. 02-11), or keep everything in place this round?

---

**Gate status:** no code touched (docs only) — per-module GOWORK=off builds
not re-run this session; last full `#verify` GREEN evidence = 13:15 run #4
(13-15 report). Foreign WIP (`metaengine/memory_stream_log.go`,
`metaengine/seq_seek.go` + tests) never staged.

**Commits this session:** `73ce6e1b7` (harvest+plan) · `aa00b3ae6`
(CHANGELOG) · `6046ba0aa` (AGENTS) · `7d96181f1` (03-18 annotation + TODO
defect) · [FEATURES rode `a1334d8c5`, ROADMAP rode `a6660bfd1` — verified in
HEAD]. Not pushed yet.
