# Status Report: Core Data Model v5 Plan — T12 Shipped, Encoding Decision Executed, 8-Tag Release Wave Cut — 2026-08-22 14:10

**Session scope:** Resumed from the WAIT-gated handoff. Unbroken HEAD, **completed T12 end-to-end**, executed the owner's **Encoding=compact-typed decision**, answered the tag gate with **go** and ran the **full release wave: 8 tags cut+pushed, 66-module pin sweep, replace drops, standalone build matrix green**, closed T24 entirely, shipped T25, and cleared the quick-win tail (READMEs, plan annotation, fmt parity). Plan progress: **T01–T12 done, T24 done, T25 done → 16 of 25.**

---

## a) Fully done (verified end-to-end)

1. **HEAD unbroken** (`f5fcc18b8`): one-line record import fix; event build+race+lint green, committed immediately.
2. **T12 complete** (`1fbd604fc`): command.Type + query.Type joined event.Type as aliases of record.Type; deprecated sentinel-preserving ParseType wrappers; alias lockstep tests in all three modules (cross-type comparison form); golden −2 methods/module, +record.Type/ParseType; CHANGELOG; changelog gate green (81 citations).
3. **Exclusive gate GREEN over T11+T12** after `nix fmt` parity fix (`f518d24a0` — the predicted T10/T11 treefmt drift was real: 6 files) + `#check-duplication` + `#check-arch` + changelog-symbols.
4. **TODO_LIST T11/T12 ticked** with deviation notes (`85daafe82`).
5. **T24·a/b**: api-stability meta-tests green; skill `references/modules.md` rows for record/event/command/query/decider/scheduling documented (Cause/Stamp/Actor/ID/Encoding/Type, capabilities, Ref forms, TimerID); doc-check 921 refs green.
6. **§g2 executed — `record.Encoding` is now the compact typed stamp** (`aedb42fdb`): `record.Encoding` (uint8-backed) + `EncodingUnknown`/`EncodingJSON`/`EncodingCBOR` + `ParseEncoding`/`String()` + `ErrUnknownEncoding`. Record stays zero-dep; event bridge converts at the boundary (unknown codecs stamp Unknown rather than guessing); command/query bridges + zero Records carry Unknown. No stored wire format changed (ADR-0044 envelope keeps its string stamp). Full tests+lint+golden+CHANGELOG.
7. **§g3 executed — 8 tags cut AND pushed** (owner approved incl. push): `record/v4.4.0`, `metadata/v4.6.0`, `event/v4.8.0`, `command/v4.8.0`, `query/v4.7.0`, `decider/v4.4.0`, `scheduling/v4.3.0`, plus **`storage/v4.8.0`** (forced: published storage v4.6.0's timer twin predated TimerID and stranded 8 consumers). Every tag verified standalone-buildable at cut time; every tag fetch-verified after push.
8. **Consumer-pin sweep + replace drops**: 66 modules re-pinned (the 7 wave modules + id + storage) and local sibling-replaces for all published targets dropped; per-module tidy + full 66-module GOWORK=off build matrix **green**. Middleware/encryption/signing's 2026-08-17 unpublished-symbol replaces are gone.
9. **Final exclusive `#verify-fast` GREEN** (after one cqrs-lint golden refresh: V006's version enumeration follows go.mod).
10. **T25**: AGENTS.md Internal Contract #21 (data-model conventions) + tag-wave mechanics gotcha; doc-check green. (Daemon-committed as `137d23160`.)
11. **Quick wins**: scheduling README quickstart no longer shows non-compiling string timer IDs; sqlstore README documents typed Actor + boundary conversion; plan Appendix C got a post-T09 note; f23 callout was already present.

## b) Partially done

- Nothing from this session's executed scope. (f7 straggler sweep: the `example/taskmanager` hits are `system.Execute` — the system-level pair API, explicitly out of plan scope per f21. No decider pair-form stragglers exist.)

## c) Not started (remaining plan tail)

- **T16** report polish · **T17** snapshot NewSnapshot+stamp (L) · **T18** snapshot wire audit (after T17) · **T19** deprecation census artifact · **T20** tombstone v5 prep · **T21** extended review storage/system/stack/watermill/middleware (L) · **T22** naming review · **T23** upstream skill fixes · f24 PG integration run for sqlstore.

## d) What I totally fucked up (honest log)

1. **Loop-variable log path bug**: redirecting to `/tmp/$m-test.log` with `m=scheduling/sqlstore` created a phantom failure (redirect failed, not tests). Re-ran with a flat name. Cheap lesson: never build log paths from module dirs containing `/`.
2. **Repeat-edit round-trips on the alias test**: ST1023 (omit redundant type) vs S1021 (merge decl+assign) form a pincer — every var-decl shape trips one of them. Solved with the cross-type comparison form (`event.Type("x") != record.Type("x")` — only compiles while the alias holds). Should have foreseen that the explicit-type compile guard IS the thing the linter hates.
3. **Two extra tag-script iterations** (metadata, command) because `go mod tidy` never bumps require pins — package-level resolution cannot see missing symbols. Pre-bump-then-tag is now memorized in AGENTS.md.
4. **`go env` ran without the GOMODCACHE override** and died on the corrupt /mnt/buildcache — burned a call re-running with the env block. The env block is needed for EVERY go invocation, no exceptions.

## e) What I can do better

- Sweep scripts must sanitize module dirs into flat log filenames from the start.
- For compile-guard tests, reach directly for the comparison form; var-decl forms always lose to stylecheck.
- Before any tag wave: pre-bump ALL dependent pins in one commit, THEN tag in dependency order with push interleaved.

## f) Next steps (priority order)

1. **T17** `snapshot.NewSnapshot` + envelope-style codec stamp + Validate + invariants (kills P10).
2. **T18** snapshot wire-tag migration audit (pebble/bbolt/sql/memory) after T17.
3. **T19** deprecation census → `docs/planning/v5-deprecation-sweep.md` (36 Aggregate aliases, wire tags, stale error codes, record-bridge Deprecated fields, module ParseType wrappers, decider pair forms).
4. **T20** tombstone v5 deletion prep (migration doc verify + StatusMiddleware bridge coverage).
5. **T21** extended data-model review: storage/*, system/, stack/, watermill/, middleware/.
6. **T22** naming review (Stream/StreamRef/StreamID/Cause/Stamp/Actor vocabulary).
7. **T23** upstream skill fixes (reviews↔brainstorming divergence; read-prior-reports step).
8. **T16** report polish (anchor integrity, CSS diff) on the core review HTML.
9. f24: run `#integration-pg` once over scheduling/sqlstore before its next tag (PG path compile-verified but not executed).
10. Consider `system.Execute` ref-form wrapper at v5 (out of plan scope, noted).
11. master is 45+ commits ahead of origin — an owner push decision is pending (tags are already pushed; the branch is not).

## g) Questions (max 3)

1. **API-shape recommendation delivered (was §g1/Q2 "what do you think?"):** KEEP both. (a) Decider Ref forms: one `id.StreamRef` argument makes the (id, type) pair un-transposable — the pair form's two strings silently compile when swapped; internals (singleflight/cache/snapshot) were already StreamRef-keyed, so Ref forms expose the real implementation with less hot-path work; old forms remain as lockstep-tested deprecated forwarders. (b) String-backed TimerID: timer IDs are caller-chosen idempotency keys — ULID backing would mint a new ID per retry and duplicate every scheduled timer, the exact bug class durable timers exist to prevent; follows the documented id.StreamID pattern. Veto either → name the rework scope.
2. **Push master?** 45+ commits ahead; all 8 tags are already on origin but the branch carrying their parent commits is not. CI runs on push.
3. **Continue with T17 (snapshot constructor) next session, or reprioritize the tail?**

---

_State at pause: HEAD `137d23160`, working tree clean, `#verify-fast` GREEN on the final state, GOWORK=off build matrix green for all 66 swept modules, 8 new tags on origin. T01–T12 + T24 + T25 of 25 done; T16–T23 remain. No branch push performed._
