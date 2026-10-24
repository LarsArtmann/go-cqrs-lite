# Disk-Truth Tail Session II — Brutal Status — 2026-10-24 09:40 CEST

> Resuming-session report for the 2026-10-24 tail execution (sessions ran
> ~05:30–09:40 CEST). Predecessor status:
> `docs/status/2026-10-24_07-00_disk-truth-tail-II-wave-cut-status.md` (the
> optimistic mid-session version — this one is the honest one).
> Plan: `docs/planning/2026-10-23_17-05_SUPERB-disk-truth-tail-delivery-pareto-plan-v3.md`.

## a) FULLY DONE (verified, receipts in TODO_LIST/CHANGELOG)

1. **C02 — the release chain: 14 TAGS LIVE and proxy-served.**
   metaengine/v4.20.0 → sqliteengine/v4.6.0, pebbleengine/v4.4.4,
   projectionadapter/v4.6.2, storage/turso/v4.3.5, catalog/v4.10.0,
   cmd/cqrs-lint/v4.18.0, stack/v4.5.1, otel/v4.5.3, prometheus/v4.3.4,
   schema/v4.7.1, event/v4.15.2 → system/v4.15.4. G1 scratch-consumer probe
   green on the full graph (see d-1 for the v4.15.3 race).
2. **Typed-WAL family complete (A11 shipped prior; this session A12a/A12b/A13/A14):**
   pgengine (both write paths, JSONB metadata, scratch-DB legacy pin),
   mysqlengine (INSTANT-only ALTERs = no-recreate by construction; rerun-safe
   pins), duckdbengine (unique-index story verified live — dedup-table
   fallback not needed), adttest conformance wired into all four suites.
   Uniform contract decision: identity-only backfill (a malformed foreign row
   must never abort a migration).
3. **A15 layout-stamp default-ON warn-first** — the exported-but-unwired
   primitives are now wired at both apply paths; pinned on the type-drift
   class; honest boundary documented (column-SET conflicts hard-error at
   apply first — the stronger guarantee).
4. **A10 C044 `branded-id-display-string`** — implemented, warn-first, typed
   fixtures 3-good/3-bad, immediately caught 2 TRUE wire leaks in the
   taskmanager example (golden updated). Rules 213→214.
5. **Tail items A16/A17/A18/A19/A20/A21/A22/A23 all done with pins** — docs
   truth pack (doc-check 1349 refs green), deriver coverage verified, timers
   on the gcdc envelope (mixed-format migration pin), time-discipline rule
   doc, wire tags + golden bytes, recipes §2.5b (compile-verified), cursor
   audit (no suspect sites), sticky-key legacy pin + benchkit's five
   display-form keys fixed.
6. **C03 go-appkit half** — both modules green on system/v4.15.4,
   bump committed.
7. Between-session hygiene: 35-module go.sum re-tidy (dev-shell toolchain),
   api golden 6270→6301, V007 drift entry for the codec session's
   deprecation, typedfixture budget bump, A10 re-rowed in TODO_LIST + plan v3
   addendum after being dropped.

## b) PARTIALLY DONE

- **C04 closeout — verify chain FAILED on 2 doc assertions (see d-2/d-3) and
  stopped BEFORE the heavy `#verify` leg.** `#verify` has still NEVER run to
  completion across both tail sessions. The two CI runs on master
  (b78476219, ddef3c5dc) were in_progress/pending at session end — the
  post-wave matrix is UNCONFIRMED; the doc-assertions job will be red on the
  wave commit until d-2 is fixed.
- **C03 cqrs-htmx half** — a parallel session is aligning their pins (6 align
  commits observed: metaengine, schema, catalog, otel, prometheus, stack).
  Correctly not raced; their battery result NOT yet verified.

## c) NOT STARTED

- **A9 keycodec v2 engine adoption** — the one remaining disk-truth item;
  deliberately its own migration wave; TODO_LIST carries the per-file map and
  the pin patterns to copy from this wave.

## d) TOTALLY FUCKED UP (session misses, all owned)

1. **The v4.15.3 dead tag — I re-fell into the documented trap.** The
   "dependent cuts need two rounds / tag-time tidy races the proxy" lesson was
   IN MY CONTEXT, and I even smoke-checked the deps first — but the system
   cut's `go mod tidy` still resolved OLD siblings (proxy cache lag between
   smoke and tidy). v4.15.3 is permanently in the proxy pinning metaengine
   v4.19.0. v4.15.4 fixed it. The missing step: after pushing deps, run the
   dependent's `go get <dep>@<new>` LOCALLY and commit BEFORE cutting its tag
   — never let tag-time tidy do first-resolution.
2. **My CHANGELOG wave-heading cut broke the doc-assertions gate.** The
   python section move put the wave heading FIRST and [Unreleased] second;
   the gate (correctly) requires [Unreleased] directly under the header
   block. The verify chain caught it — but it should never have been pushed;
   my wave-protocol checklist lacked "run the doc-assertions gate BEFORE
   pushing the heading cut".
3. **ADR index drift surfaced and left:** 159 ADR files, 158 indexed — one
   parallel-session ADR is un-indexed in docs/README.md. Foreign, but the
   failing gate is now on master and blocks C04's green.
4. **Write-then-read instead of read-then-write burned ~8 compile cycles**:
   invented `schema.Producing[...](schema.Target(...))` (wrong API — the
   recipe compile gate caught it), `idtest.ParseCommandID` (doesn't exist),
   wrong `id.NewStreamRef` arity, wrong `adapter.Save` signature, missing
   `fmt` import, 27-char ULIDs. Every one was a 30-second grep away.
5. **Reran the adttest lesson instead of applying it:** the typed-WAL pins
   shipped with STABLE collection names; run 2's first append conflicted
   (deterministic red). The fix (run-unique suffixes) was literally my own
   CHANGELOG row from the same session.
6. **A15 took four iterations** (Store.Close closes engines; t.Parallel
   logger-swap races; and the scenario premise itself was wrong — column
   add/rename hard-errors at apply, only type-drift reaches the stamp). The
   last one was a genuine discovery; the first two were readable-in-advance.
7. **One greedy TODO_LIST edit clobbered the head of the "Run the composed
   proof" row** (old_string matched a prefix of an existing row); caught and
   repaired two edits later.
8. **Leftovers:** MariaDB server still running (background shell 148 — kill
   or keep for A9); /tmp/duckidx probe module left in /tmp (volatile, will
   vanish); the foreign SQL-engine planned-path failures (pgengine explain +
   mysqlengine ×2, verified pre-existing at `d0e86a117`) are STILL RED and
   will fail the integration gates of the very wave I shipped.

## e) IMPROVE

- Wave protocol amendment (lesson d-1 + d-2): (1) dependent tags are cut only
  after a local `go get` pin-bump commit; (2) doc-assertions run BEFORE the
  CHANGELOG heading cut is pushed.
- Read signatures before writing code against them; a compile failure is a
  reading debt, not a iteration step.
- The verify chain's fail-fast worked exactly as designed — trust it EARLIER
  (run it before pushing, not only after).

## f) NEXT (in order)

1. **Fix the two doc-assertion failures** (swap [Unreleased] back above the
   wave heading; index the 159th ADR) — unblocks the CI matrix on the wave
   commit. Then re-run the verify chain to its `#verify` conclusion.
2. Verify the post-wave CI matrix per-job; confirm the cqrs-htmx parallel
   session's battery.
3. A9 per the TODO_LIST row.
4. The two open owner questions (deployed-DB census; A9 scheduling).

## g) OWNER QUESTIONS (2 open, do not block)

1. **Deployed-DB census** — prefixed stream keys in production? Determines
   sticky-key permanence vs re-key + v1-range drop (G3). The sticky path is
   pinned green either way.
2. **A9 scheduling confirmation** — own migration wave (recommended) vs
   pulled ahead.
