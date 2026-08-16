# Docs-Health AUDIT In Flight — 2026-08-16 13:36

> Session executing the **docs-health skill AUDIT** (BUILD + HARVEST + VERIFY +
> ANNOTATE) over all 17 files dated `2026-08-16*` (15 status + 1 planning + 1
> benchmark; note `2026-08-16_08-19_*.html` is an HTML report, not `.md` —
> text-extracted, and per the user's directive only `.md` files get inline
> strikethrough annotation + archiving) plus the 5 living docs (TODO_LIST,
> CHANGELOG, ROADMAP, FEATURES, AGENTS). Interrupted by a status request
> mid-flight; this report is that snapshot.

---

## a) FULLY DONE

1. **Skill + references loaded** — `docs-health` SKILL.md + 6 companion guides
   (harvest, resolving-items, verify-checklist, annotation-placement,
   doc-ownership, health-report-format).
2. **All 17 dated files read** (the 08:19 HTML text-extracted via Python — no
   CSS noise).
3. **Living docs read**: TODO_LIST + ROADMAP fully; CHANGELOG head + release
   index; FEATURES structure; AGENTS.md.
4. **HARVEST verified against code/git** (docs are leads, not evidence):
   - 82 `go.mod` files ✓
   - 2026-08-16 tag chain live on proxy (id v4.5.0, record v4.3.0,
     metadata v4.5.0, schema v4.3.0, event v4.7.0, query v4.6.0→retracted→
     v4.6.1, command v4.7.0→retracted→v4.7.1, middleware v4.5.0,
     metaengine v4.11.0, sqlite/pebble/pg engines v4.1.0, badger v4.0.2,
     watermill v4.5.0, mysql/bbolt/turso/iroh engines v4.0.0,
     storage v4.7.0→retracted→v4.7.1) ✓
   - Stranded repair commits `092b5e8a8` / `4907b6afc` **NOT on master**
     (merge-base verified) ✓
   - Master `command`/`query` go.mod still pin `metadata/v4 v4.4.0` ✓
   - Local replaces counted: system ×6, cqrs-bench ×7, event/schema/
     projectionhost/integration ×2 each ✓
   - `/tmp/verify-full4.log` (13:15 run #4): 0 FAIL lines ✓
   - 4 stale git worktrees registered (`/tmp/cqrs-tagwt`, `wt-head`,
     `/tmp/gcl-verify`, `go-cqrs-lite-pin`) ✓
5. **TODO_LIST.md updated (7 edits)**:
   - Deleted closed items: "Session verification gap" (verify #4 + coverage
     EXIT=0 at 13:15), "Cut command/v4.6.1" (superseded by v4.7.0 → retract →
     v4.7.1).
   - Rewrote Release/Tagging section around the real 2026-08-16 tag-chain
     state (old text still said "engine v4.0.2+ tags pending" — stale).
   - Added open harvested items: wave-4 tag batch (event/metadata/schema/
     metaengine/irohengine/projectionhost/storage-v4.7.2, with the
     projectionhost dependency constraint), stranded-commit cherry-picks,
     go-codec F46 commit+tag, two judgment-call ratifications (iroh latency
     bound 150ms, OpenSQLiteInMemory pool pin), replace-drop sweep, GitHub
     Releases ×20, retract-pattern documentation, conformance under
     `#test-integration`, iroh graph WriteOp replication, irohengine
     capability-forwarding audit, capability diagnostics beyond Doctor,
     api-stability fail-on-parse-skip, BuildFlow gofmt staged-gate, pre-gate
     load-sweep script, local-path replace guard, worktree cleanup,
     capability-diagnostics skill docs, OpenSQLiteInMemory unique-DSN
     follow-up.

## b) PARTIALLY DONE

1. **CHANGELOG**: gap confirmed — the 2026-08-16 20-tag chain + the three
   retract incidents (command v4.7.0, query v4.6.0, storage v4.7.0) have
   **zero entries**. The 2026-08-13 multi-module section format was located
   as the template; writing was next when interrupted.
2. **FEATURES.md**: gap confirmed — wave-3/4 features absent (grep found 0
   hits for `WithBatchCommit`, `WithCopyAppend`, `DecorateJournal`,
   `WithCheckpointEvery`, keyset pagination, `CapabilityAudit`,
   `WithIdempotencyCapacity`, `AdoptedPayload`, pin-drift). Rows not yet
   added.
3. **ROADMAP.md**: stale claims identified but not yet edited — header still
   says "v4.7.0 tagged (2026-08-10)"; Open Question 1 (tag authorization)
   superseded by the executed chain; Open Question 2 (Go 1.26.6 direction)
   resolved via `ea8fa5072`.

## c) NOT STARTED

- **ANNOTATE — the core user ask**: inline `~~item~~ done at \`hash\``
  resolution of every numbered item in the 17 files (the f-lists run 10-50
  items each; the planning doc carries T1-T27 + F1-F60 tables needing Status
  verdicts). Not one file annotated yet.
- **ARCHIVE**: `git mv` of fully-resolved reports to
  `docs/status/archived/` (directory exists; `docs/status/` holds 465 files).
- AGENTS.md freshness edits (two gotchas identified: modernc `file::memory:`
  per-connection databases; local-path `replace` anti-pattern → `go.work`
  `use`).
- doc-check gate, cross-file consistency pass, and the final health report
  (Accuracy + Fitness scores with visible math).

## d) TOTALLY FUCKED UP (honest ledger)

1. **The todo-list tool went stale** — 12 items set, updated once, then the
   session worked for many steps without reflecting progress. Process miss;
   the list is the resume-context for interrupted sessions.
2. **First grep for the tag-script build gate was too narrow** (`GOWORK=off go
   build` literal) — the broader re-run confirmed the standalone-build gate
   exists ONLY in the stranded worktree commit `092b5e8a8`, not in master's
   `scripts/tag-release.sh`. One wasted cycle, correct final fact.
3. **Status report initially printed inline instead of written to the
   requested `.md` file** — caught by the user ("this is not a .md file!");
   this file is the correction.
4. **No gate run yet** after the TODO_LIST edits (doc-check owed before any
   GREEN claim — the stale-GREEN rule applies to docs too).

## e) WHAT WE SHOULD IMPROVE

1. Update the todo tool as work progresses — it is the resume-context for
   interrupted sessions.
2. Interleave doc edits with gate runs; do not batch verification to the end.
3. Instructions that name an artifact ("write a report AT <path>") mean the
   file write, not an inline print.

## f) NEXT — resume order (docs-health remainder)

1. CHANGELOG: 2026-08-16 tag-chain + retract section (2026-08-13 style).
2. FEATURES.md: add wave-3/4 rows (bbolt WithBatchCommit, pgengine COPY +
   batching, pebble knobs, projectionhost checkpoint batching, journal keyset
   pagination ~285x, packet-safe byte-capped chunking, event.DecorateJournal,
   payload adopt-variant, idempotencyTracker bound, multiSeqCounter pad,
   capability audit + Doctor section, pin-drift meta-test,
   OpenSQLiteInMemory pool fix).
3. ROADMAP: refresh header + Open Questions (drop resolved Q2, rewrite Q1).
4. AGENTS.md: +2 gotchas (`file::memory:` per-connection DBs; local-path
   replace anti-pattern).
5. ANNOTATE the 15 `.md` status files + the planning `.md` (strikethrough +
   `done at` hashes; table rows get Status verdicts; the 13:15 report is
   current-state — minimal touch; the benchmark evidence doc needs no
   item resolution). The 08:19 **HTML** report: no inline strikethrough
   (not `.md` per user directive) — at most an appendix pointer.
6. ARCHIVE fully-resolved `.md` files (candidates after annotation: 02-11,
   03-18 plan; decided per skill default — every item resolved).
7. doc-check gate + cross-file consistency + health report (Accuracy +
   Fitness, printed inline).

## g) QUESTIONS (cannot figure out myself)

1. **Archive threshold**: archive only files where EVERY numbered item
   resolves (skill default), or also same-day files whose open remainder is
   already routed into TODO_LIST? Default taken: skill default.
2. **CHANGELOG shape for the 20-tag chain**: one combined
   `## [2026-08-16 module releases]` section (the 2026-08-13 pattern), or
   per-module sections? Default taken: combined.
3. **The 08:19 HTML report**: leave untouched (not `.md`, outside the
   strikethrough directive), or convert/annotate? Default taken: leave;
   its supersession is already noted by the 09:13 report.

---

**Gate status:** no code changed this session; doc-check NOT yet run after
the TODO_LIST edits. Tree carries the TODO_LIST.md changes uncommitted (the
auto-commit daemon will commit). Resuming at CHANGELOG unless redirected.
