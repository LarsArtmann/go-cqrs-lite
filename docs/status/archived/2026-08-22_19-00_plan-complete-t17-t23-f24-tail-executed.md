# Status Report: Core Data Model v5 Plan — Tail Executed (T16–T23, f24) — 2026-08-22 19:00

**Session scope:** Resumed from the 14:10 handoff. Answer-gate resolved (all three questions), master pushed (owner-approved), then the **entire remaining plan tail executed: T17, T18, T19, T20, T21, T22, T23, T16, f24 → 25 of 25 tasks done.** The plan is COMPLETE. Final exclusive gate run recorded at the bottom.

---

## a) Fully done (verified end-to-end)

1. **Answer-gate** (question tool): KEEP-BOTH identity shapes **confirmed** by owner; **push master approved** (executed: `327c3d2f9..5d3ce030d`); T17-next confirmed.
2. **T17** (`1a7be0816` + daemon split `b9e5f52b8`): `snapshot.NewSnapshot(ref, version, state, encoding)` validating ctor (ref.Validate, version ≥ 1, non-empty state; Rejection family with per-codec codes; clones state; UTC-stamps time) + `Snapshot.Validate` + `Snapshot.Ref` + `ErrInvalidSnapshot` + `Encoding record.Encoding` field (ADR-0044 pattern). TypedStore + decider repository stamp on save; `SaveSnapshot` deprecated; decider migrated to ctor + `sink.Save`. Tests, lint, golden (+4 symbols), CHANGELOG, arch exception (`snapshot→storage/memory` test-only), modules.md row.
3. **T18** (`59799970e` + `6d388140a`): wire audit table in TODO_LIST §v5 (memory=in-process, pebble=old-vocab CBOR + legacy-JSON fallback, bbolt=already-honest stream tags, SQL=columns, keep-old-tags-until-v5 decision). **pebble+bbolt now persist the encoding stamp** (additive `omitempty` CBOR field, old rows → Unknown, roundtrip tests pin it); SQL stays envelope-authoritative.
4. **T19** (`f464b7936`): `docs/planning/v5-deprecation-sweep.md` — 42 aggregate-vocabulary symbols across 6 modules (id 12, event 8, command 6, query 2, listing 5, misc 9) + 5 record-bridge fields + tombstone API + wire-tag/error-code inventory + deprecated-module list + execution rules.
5. **T20** (`463bb525e` + `d5c8f78d9`): migration doc verified accurate against shipped APIs. **Real gap found and FIXED: `Materialize.OnTombstone`/`OnRebirth` had ZERO trigger coverage anywhere** — new `stack/materialize_tombstone_bridge_test.go` pins the full StatusMiddleware→mark→handler chain. v5 deletion pre-reqs listed.
6. **T21** (`9b863467e`): extended data-model review → `docs/reviews/2026-08-22_extended-data-model-review.md`. 15 findings (E1–E15) + verified capability matrix. Verdict: no any-typed values anywhere in storage/system/stack/watermill/middleware; the disease is cross-backend drift (Encoding string vs typed, bbolt bare fmt.Errorf, RetryConfig name collision, 7 middleware signatures). Pebble lying-doc ("Panics" vs returns error) fixed on the spot (`71c2d75e7`). Follow-ups extracted to TODO_LIST (3×S + v5 batch).
7. **T22** (`8e3f701fe`): naming decision recorded as plan **Appendix D**. Trio KEEPS adjective-bearing names with role table. **Real smell found: `record.StreamRef` (string, `/` separator) collides with `id.StreamRef` (struct, `:` separator)** — same name, different concept AND different canonical form. v5 rename → `record.StreamKey` queued in the sweep. ActorKind twin enums re-confirmed as accepted zero-dep mirror (extend both or neither).
8. **T23** (skill dir, tick `9b562f7c6`): data-model-review skill output-guide now says `docs/reviews/` (was `docs/brainstorming/` — the divergence); Step 5 gained "read prior reports in the series" + "copy the template, never transcribe".
9. **T16** (`fc2b18fe2`): core review HTML polish — anchor check now **0 broken / 0 orphans** (Amendments gained its TOC entry); CSS diff vs kit template: **zero token drift** (clean subset + 2 legit additions); Related Reviews + Next Skills sections added with cross-links.
10. **f24** (`043467885` + `c1814ae7f`): **first actual execution of the PG path post-TimerID — and it caught two real breaks**: (a) `pg_integration_test.go` compared TimerID against untyped strings — the `-tags=integration` file was never compiled by the normal gates (handoff's "compile-verified" claim was wrong for this file); (b) **benchkit + projectionhost still pinned stranded storage tags** (v4.6.0/v4.4.0) incompatible with scheduling v4.3.0. Both fixed; scheduling/sqlstore ran **green on PostgreSQL (5.4s)**. Stranded-pin sweep verified all remaining old pins are internally consistent (build OK).

## b) Partially done

- Nothing from this session's executed scope.

## c) Not started (nothing left in the plan)

- **Plan complete: 25/25.** Remaining repo work lives in TODO_LIST (see §f).

## d) What I totally fucked up (honest log)

1. **Fell into the documented `/`-in-log-path trap AGAIN** (storage/memory build loop): phantom FAILs from failed redirects, exactly the §d1 lesson from the 14:10 report. Two calls burned. The `tr / -` sanitizer must be muscle memory from the first loop iteration.
2. **T22 appendix append blunder**: appended a duplicate T22 section at the file END instead of editing the body section in place; had to surgically remove it (python rfind) before doing the correct in-place edit.
3. **Edit-tool mtime staleness** bit once (plan doc modified by my own python between reads) — re-read then edit; no damage.
4. **`../snapshot` replace depth wrong** for storage/pebble + storage/bbolt (needed `../../snapshot` — nested module dirs). One round trip.

## e) What I can do better

- Log-path sanitizer in the FIRST version of any module loop, not the second.
- For plan-doc annotations: edit the body section in place; never append sections that already exist.
- When a handoff claims "compile-verified", ask "with which tags?" — the integration-tagged file class is invisible to every normal gate.

## f) Next steps (priority order — all new TODO_LIST entries from this session)

1. **PG integration test isolation under explicit DSN** (S) — `TestPostgresEventStore_CRUD` reads a shared `cqrs_test` DB ("expected 2 events, got 27", pre-existing, reproduced twice). Per-test DB creation even under POSTGRES_TEST_DSN.
2. **bbolt error-family parity** (review E3, S) + **turso.Policy nil-map guard** (E9, S) + **system.ShutdownDependency name validation** (E10, S).
3. **Snapshot tag wave** when convenient: snapshot/decider/storage/pebble/storage/bbolt carry unpublished symbols (local replaces active in decider, pebble, bbolt). Tag in dependency order with the push-interleave protocol; drop replaces in the sweep.
4. **v5 cut wave** (the big one): execute `docs/planning/v5-deprecation-sweep.md` (42 aliases + 5 bridge fields + tombstone API + wire tags + error-code batch + extended-review E1/E7/E8/E11/E13/E15 + `record.StreamKey` rename) per its execution rules.
5. Consider `system.Execute` ref-form wrapper at v5 (out of plan scope, noted in 14:10 report).

## g) Questions (max 3)

1. **Snapshot release wave now or batched?** Six modules carry local replaces for the new snapshot API (see §f3). Cut tags this session, or leave for the next planned wave?
2. **Pre-existing PG isolation bug (§f1):** fix next session (S effort), or is the shared-DSN suite acceptable as-is since per-package suites still gate the modules they exercise?
3. The plan is complete — **next major move: start the v5 deletion wave, or pause for an owner review of the sweep artifact first?**

---

_State at pause: HEAD `043467885`, working tree clean, all 25 plan tasks done (T01–T25), master pushed through `5d3ce030d` (later commits local), 8+0 tags on origin, final `#verify-fast` result recorded in the conversation. Plan artifacts: sweep doc, extended review, Appendix D naming table, bridge test, encoding stamp persisted end-to-end._
