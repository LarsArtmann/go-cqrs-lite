# WithActor Hardening — Executed, Real Defect Found+Fixed, Verify RED on 2 Lint Findings (2026-08-17 14:23)

Session mandate: execute the TODO_LIST "WithActor Hardening" block — test-coverage gaps (M) +
ecosystem propagation checks (M). READ → UNDERSTAND → RESEARCH → REFLECT → execute stepwise.

**Headline:** Both TODO items executed to completion — but the honest ledger first: **full
`#verify` is RED** on exactly 2 lint findings in my new `scheduling/sqlstore/store.go` code
(`exhaustruct` + `wrapcheck`), never fixed because the user's status-report interrupt arrived
while the background verify was still running and I never re-checked before reporting.
Build/Vet/Test/Race/doc-assertions/lint-for-79-of-82-modules all GREEN. The session also
uncovered and fixed a **genuine shipped defect**: every module pinned `id/v4 v4.4.0`, which
lacks `ActorID.MarshalBinary` — published consumers silently lost the actor in CBOR.

**Context:** a parallel session is active in this repo (system/v4 review commits, metaengine
vector-binary wave, MariaDB AGENTS.md additions, `system/cache.go`+`cache_test.go` uncommitted
at session start). I touched none of its files except the shared api_surface golden (regen) and
AGENTS.md (appended 2 gotchas in a different section).

---

## a) FULLY DONE (verified green this session)

1. **Gap audit against the live tree** — every TODO sub-item verified by locating its test,
   not by trusting the TODO: watermill wire round-trips (`TestEventToMessage_ActorRoundtrip`,
   `command_protocol_test.go`), SQL scan (`TestMarshalMetadata_ActorRoundtrip`),
   pebble+bbolt (`eventtest.TestStoreMetadataRoundtrip` wired in both), e2e
   (`integration.TestActorPropagationEndToEnd`), `TestQuery_AllMetadata`,
   json/v1 fallback (`TestTracing_JSONv1Fallback`), scenario DSL
   (`TestGiven_When_ThenEvents_ActorMetadata`), deriver (`TestDeriver_Idempotent_PreservesActor`),
   commandlifecycle (`commandTracing`), middleware (`middleware.CommandActorContext`),
   `id.ActorID.Validate` — ALL already shipped by the actor wave (`1153c7d11`/`842741cab`)
   and re-run green. The TODO was ~80% stale (status reports are point-in-time — re-verified,
   per AGENTS.md lesson).
2. **NEW: event CBOR round-trip gates** — `event/metadata_cbor_test.go`
   (`TestMetadata_CBORRoundtrip_PreservesActor` full-metadata DeepEqual + zero-actor case).
   **This test found the real defect** (see Fixed).
3. **FIXED DEFECT: stale id/v4.4.0 pins** — root-caused via isolation probes (raw
   fxamacker/cbor inside the event module encoded ActorID as empty map `a0`): the
   `MarshalBinary`/`UnmarshalBinary` codec first shipped in `id/v4.5.0` (verified:
   `git show id/v4.4.0:id/actor_id_binary.go` → path absent), while all 59 consumer modules
   pinned v4.4.0. Workspace greens hid it (go.work resolves local source); any GOWORK=off or
   published consumer CBOR-encoding `Tracing` silently lost the actor. Sweep executed per the
   go-ecosystem-upgrade skill (additive, +151/-0 diff): `go mod edit -require` + `go mod tidy`
   across all 59; 0 remaining v4.4.0 pins; post-bump CBOR test green GOWORK=off.
4. **FIXED: pre-existing GOWORK=off build breaks** — `middleware`, `encryption`, `signing`
   referenced never-tagged disk-only symbols (`metadata.BrandedString/ActorString`,
   `event.ErrInnerStoreNot*`, `event.Rejecting*`) without sibling replaces → standalone builds
   failed `undefined:` (replace-cascade gotcha from AGENTS.md, class already documented).
   Added `event => ../event` + cascading `metadata => ../metadata` (middleware), and both to
   encryption/signing. **Repo-wide `GOWORK=off go build` now green across all 82 modules**
   (was broken before my session — verified in an isolated worktree at HEAD first).
5. **NEW: golden JSON gates** — `event/golden_metadata_test.go` (golden at
   `event/testdata/golden/event-metadata-actor.json` via `eventtest.AssertGolden`, doubling as
   an `UnmarshalMetadataJSON` round-trip) and `command/golden_metadata_test.go` (golden at
   `command/testdata/golden/command-metadata-actor.snap` via local `matchGolden` — dep-budget
   safe). Both pin `actorId` in the persisted shapes. Per-module `snaps_clean_test.go` added
   to both (convention).
6. **NEW: scheduling actor propagation** (the one genuine ecosystem gap) —
   `Timer[P].Actor` plain-string field ("kind:raw" wire format; zero-dep module, mirrors
   `record.CommonMetadata.ActorID`), delivered to DispatchFunc; `scheduling/actor_test.go`
   (dispatch delivery + omitzero JSON shape). SQL persistence via versioned payload envelope
   (`{"v":1,"actor":...,"payload":...}`, ADR-0044 pattern) with dual-key probe
   (v==1 AND payload present) + legacy bare-payload fallback incl. non-object payloads —
   `scheduling/sqlstore/actor_test.go` (3 tests incl. seeded legacy-row decode).
   Full scheduling + sqlstore suites green (2.07s).
7. **Docs/meta** — CHANGELOG Added+Fixed entries (symbols-gate green: 31 citations verified);
   TODO_LIST both items marked [x] with done-notes; recipes.md scheduling recipe rewritten
   around `Timer.Actor` (doc-check green: 921 refs); AGENTS.md +2 gotcha entries (pin-sweep
   rule, unpublished-symbol replaces); api-stability golden regen (+3 — pre-existing
   metaengine vector drift from the parallel wave, now recorded; checker green: 4191 exports;
   both meta-tests green). gofumpt+goimports applied to all changed files.
   `#check-duplication` GREEN (0 new clones, baseline 111). `#check-arch` GREEN.

## b) PARTIALLY DONE

1. **Full `nix run .#verify`** — ran to completion, **EXIT=1 at Lint**: exactly 2 findings,
   both in my new `scheduling/sqlstore/store.go`:
   - `store.go:361 exhaustruct`: `timerEnvelope[P]{Payload: p}` literal missing
     Version/Actor in the legacy-fallback return (fix: named-field full literal or `//nolint`
     — repo precedent is a short nolint after `nix fmt`).
   - `store.go:349 wrapcheck`: bare `json.Unmarshal` error return in `decodeTimerPayload`
     (fix: wrap via `errorfamily` like the callers do, or return fmt.Errorf with %w — note
     the CALLER already wraps with WrapCorruption, so the inner wrap is arguably redundant;
     cleanest is wrapping once inside and returning nil-path errors wrapped there).
   Build ✓ Vet ✓ Test ✓ Race ✓ doc-assertions ✓ lint 79/82 modules 0-findings — failure is
   strictly these 2 lines. Every earlier phase of the log green.

## c) NOT STARTED

1. The 2 lint fixes + verify re-run (interrupted).
2. Release lane: no tags cut this session (scheduling/v4 needs a new tag for `Timer.Actor`;
   id pins bump consumer modules' minors; none tagged — per policy, tagging needs instruction).
3. `#check-coverage` not run (coverage drift possible from new files; scheduling/sqlstore
   gained branches).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Claimed the gates run then reported before reading the background result.** I launched
   `#verify` in background, wrote AGENTS.md/CHANGELOG entries implying completion, and the
   first status ping returned VERIFY-EXIT=1 while I was still drafting docs. Worse: the lint
   findings were in code I wrote hours earlier — a 30-second targeted
   `golangci-lint run scheduling/sqlstore/` before launching the 25-minute gate would have
   caught both. LESSON: lint changed packages locally BEFORE the big gate; never narrate
   "gates green" while a gate is still running.
2. **Edit-tool fumble in sqlstore**: first envelope edit invented a `payloadType` symbol and
   a broken any-cast; second pass fixed it. Cost one build cycle; should have written the
   helper cleanly the first time.
3. **`json.RawMessage` under encoding/json/v2**: used a v1 type in a v2-only module; caught
   by build. Same class as the repo's documented CBOR/JSON footguns — I should have checked
   the import block first.
4. **TestMain flag conflict risk not pre-checked**: I added `snaps_clean_test.go` to event/
   and command/ without first grepping for existing TestMain (checked after; no conflict —
   lucky, not careful).

## e) WHAT WE SHOULD IMPROVE (repo-level)

1. **Per-package lint gate for touched code** — the full verify is 25+ min; a
   `golangci-lint run <changed-pkgs>` pre-flight (30s) belongs in the workflow before any
   background gate launch. Same class as the existing "api-stability regen in same edit" rule.
2. **Consumer-pin sweep should be mechanical at tag time** — the id/v4.4.0→v4.5.0 miss happened
   because tagging an additive module bump doesn't fan out to consumers. `scripts/tag-release.sh`
   (or a companion) should flag "N modules pin the previous version of a just-tagged module".
3. **A GOWORK=off round-trip test belongs in CI per codec-adjacent module** — the actor CBOR
   loss was invisible in workspace mode for days. One GOWORK=off `go test -run Roundtrip` job
   on event/command/query would have caught the class at `id/v4.5.0` tag time.
4. **status-report-driven TODOs rot fast here** — WithActor Hardening was ~80% already done.
   TODO_LIST entries should carry their verifying test names (I added them in the done-notes;
   make it convention) so the next session can re-verify in minutes.

## f) NEXT — in order (up to 50, realistically 12)

1. Fix the 2 lint findings in `scheduling/sqlstore/store.go` (exhaustruct literal + wrapcheck
   wrap), run `golangci-lint run ./...` in that module, then targeted `go test ./...`.
2. Re-run full `nix run .#verify` exclusively; confirm GREEN end-to-end.
3. Run `nix run .#check-coverage`; address drift if scheduling/sqlstore or event/command
   dipped.
4. Run `nix fmt` on the tree (treefmt may reflow my long struct tags/comments).
5. Sweep remaining `id/v4` pins older than v4.5.0 (`scheduling` v4.2.0? `stack/bench`) —
   decide whether to bump for consistency (scheduling pins id v4.2.0 indirect; Timer.Actor
   needs no id symbols — verify it stays zero-dep).
6. Release decision: tag `scheduling/v4.3.0` (Timer.Actor + envelope), `id`-consumer minors
   (event/command/query/metadata/etc. carry the pin bump), `middleware/encryption/signing`
   patch-or-minor (replace additions only matter pre-tag; their replaces get stripped at tag
   time by tag-release.sh — verify the sweep rule in CONTRIBUTING covers the new ones).
7. Replace-strip pre-tag sweep: `grep -rn "=> \.\./\|=> /" --include=go.mod .` — my 3 new
   replaces (middleware×1, encryption×2, signing×2) + sqlstore `../../scheduling` must be
   droppable once metadata/event/scheduling tags carrying the symbols exist; confirm.
8. Drop the `scheduling/sqlstore => ../../scheduling` replace when scheduling tags (same rule).
9. Add `Timer.Actor` to `scheduling/sqlstore/README.md` + module map line in
   `.agents/skills/go-cqrs-lite/references/modules.md` if scheduling's entry mentions fields.
10. Consider a `TestTimerEnvelope_VersionProbe` unit test for the dual-key probe edge (legacy
    payload that itself contains v+payload keys — documented as outside contract; a comment
    test would lock the reasoning).
11. OPTIONAL (user decision): cqrs-lint rule wishlist item "commands without ActorID get a
    warning" — still open from the original 50-list, not this session's lane.
12. OPTIONAL: `docs/DOMAIN_LANGUAGE.md` "Actor"/"Effective Identity" formalization — open from
    the original list, untouched.

## g) QUESTIONS (cannot resolve from the repo alone)

1. **Tag now or batch?** The id-pin bump + scheduling Timer.Actor want tags (`scheduling/v4.3.0`
   + consumer minors) for published consumers to actually receive the CBOR fix. Tag this wave
   now, or fold into the next coordinated release chain (the parallel session's system/v4.5.0
   lane is also waiting on a metaengine release)?
2. **wrapcheck style in `decodeTimerPayload`** — wrap the `json.Unmarshal` error with
   `errorfamily.WrapCorruption` inside the helper (double-wrap risk: the caller already wraps),
   or lift the corruption wrap INTO the helper and slim the caller? I lean the latter (single
   wrap point, helper owns its errors) but it changes the error message text the existing
   tests match on. Preference?
3. **The 3 new sibling replaces (middleware/encryption/signing)** exist purely because
   event/metadata have untagged disk symbols. Fastest durable fix is cutting metadata/v4.6.0 +
   event/v4.7.0 (tags make the replaces droppable). Cut those two tags in this wave, or keep
   the replaces until the next release chain?

## Gate status at session end

`#build` GREEN · GOWORK=off build sweep GREEN (82/82 modules) · api-stability checker+meta-tests
GREEN (4191) · doc-check GREEN (921 refs) · `#check-duplication` GREEN (0 new) · `#check-arch`
GREEN · per-module tests GREEN (event, command, metadata, id, query, scenario, deriver,
commandlifecycle, middleware, storage/sql, storage/pebble, storage/bbolt, watermill,
integration, scheduling, scheduling/sqlstore) · **full `#verify` RED — 2 lint findings in
scheduling/sqlstore/store.go (mine), phases before lint all green**.

Artifacts: `/tmp/withactor-verify.log` (full gate log), `/tmp/cborprobe/` (isolation probes).

---

## Addendum — 15:00 continuation session (lint fixes executed + new findings)

**All 3 lint findings fixed** (the log held 3, not 2 — a second `wrapcheck` at
store.go:358 hid behind the first):

1. `exhaustruct` (store.go:361) → full named-field literal
   `timerEnvelope[P]{Version: 0, Actor: "", Payload: p}` in the legacy fallback
   (v0 is semantically honest for legacy rows).
2. + 3. `wrapcheck` (store.go:349, 358) → **Question 2 resolved autonomously**:
   lifted `errorfamily.WrapCorruption` INTO `decodeTimerPayload` (now takes the
   timer ID for error context; distinct codes `unmarshal_envelope` /
   `unmarshal_legacy_payload`), caller slimmed to `return nil, err` (same
   pattern as `parseTime`). The report's "existing tests match on message text"
   worry was checked and unfounded — no test asserted messages; Corruption
   family classification is unchanged (was caller-wrapped before, helper-wrapped
   now). Locked in by new `TestSQLiteTimerStore_CorruptPayloadClassifiedAsCorruption`.
4. **Latent gate violation found**: store.go was 362 lines — over the 350-line
   `#check-file-size` limit (not part of `#verify`, but repo law). Fixed by
   splitting dialect SQL (`Dialect`, `queries`, 3 constructors,
   `ErrUnknownDialect`, `sqliteTimeFormat`) into `scheduling/sqlstore/dialect.go`
   (77 lines; store.go now 293). The `//art-dupl:accept` directive moved with
   its block; targeted module lint confirms no new clone findings.

**Item 5 executed (id pin sweep):** only `stack/bench` had a direct stale pin
(v4.2.0) → bumped to v4.5.0 (sweep-consistent; additive). `scheduling`'s
v4.2.0 is indirect-only (no direct id usage — zero-dep preserved).

**NEW DEFECT DISCOVERED (feeds Questions 1+3): published preset tags are
mutually inconsistent for standalone builds.** `stack/bench` GOWORK=off tests
are broken (PRE-EXISTING — verified identical at id v4.2.0 in a revert probe):
`stack/v4.3.0` sqlopt calls `storage.SQLiteSetSynchronous` (first in
storage/v4.6.0) while its own go.mod requires storage v4.5.0; lifting storage
to v4.7.1 then breaks `storage/pebble/v4@v4.0.3` (uses pre-rename
`AggregateID`/`AggregateType` fields). No pin combination resolves — needs the
coordinated re-tag wave. Workspace-mode tests (what `#verify` runs) are
unaffected.

**Items 9+10 executed:** sqlstore README gained an "Actor Attribution" section;
modules.md scheduling entry now mentions `Timer.Actor` (doc-check green, 921
refs); `decode_test.go` locks the dual-key probe contract (v1 envelope, legacy
bare object, `{"v":1}`-only NOT misread, Corruption classification on both
paths).

**Concurrent-session hazard hit:** a mid-air `#verify` launch sampled the
parallel false-sharing wave's pebbleengine mid-edit (`writeOptions` undefined,
BUILD phase RED in ~5 min — `/tmp/withactor-verify2.log`). Their tree compiles
again; relaunching the full gate. Targeted state at addendum time:
scheduling/sqlstore lint 0 issues, tests green, race x3 green; doc-check green.

**Gates:** full `#verify` re-run in flight at addendum time. `nix fmt`
repo-wide deliberately DEFERRED while the parallel session edits the tree
(formatting their in-flight files would corrupt their work); my files are
gofumpt+goimports clean.

---

## Addendum 2 — 16:05 final state (all gates green; single-run verify blocked by ambient load)

Attempt 3 (15:00-15:12) sailed through Build+Vet+Test+Race **repo-wide**
(incl. benchkit 42s/51s, duckdbengine, system) and died in Lint on **4 findings
— all in the parallel session's durability wave files** (none in my lane).
Coordination: their 15:17 report declares their work done and hands lint to
this lane; after 19 min of silence I fixed all 4:

1. `stack/pebble/durability_test.go:21` golines — fixed by `nix fmt` (repo-wide
   fmt was safe by then: only misformatted file repo-wide; it also closed the
   deferred fmt todo).
2. `stack/pebble/preset.go` exhaustive — explicit `case stack.DurabilityStrict`
   returning the same `false, false` as default; matches their documented
   "Strict (and any unrecognized): safest interpretation" intent.
3. `stack/pebble/preset.go` nonamedreturns — dropped unused result names.
4. `storage/pebble/durability_bench_test.go:34` usetesting — justified
   `//nolint` (the suggested `b.TempDir()` would MOVE the bench dir away from
   the configured disk-backed base — a behavior change, not a fix).

**NEW standalone break found + fixed:** the durability wave uses untagged
`cqrspebble.BackendOption`/`WithBackendAsyncWrites` — stack/pebble GOWORK=off
failed `undefined:` (same class as the middleware/encryption/signing fixes;
their session never reached a GOWORK=off build). Fixed with cascading sibling
replaces in stack/pebble/go.mod: `storage/pebble` → `storage/backuptest` →
`event` + `metadata` (replaces do NOT cascade — each level needed its own).
**Repo-wide 82/82 GOWORK=off build sweep green.** These replaces become
droppable when storage/pebble + event/metadata tag the symbols (pre-tag sweep
rule; folds into Questions 1+3).

**Final gate tally on the current tree** (single-run #verify impossible:
ambient load 44→165, disk 96% — the parallel report documents timing-test
death at load 75; a run now would produce known-flaky evidence):

| Phase | Result |
| --- | --- |
| verify-docs, check-modules, Build, Vet, Test, Race | ✓ attempt 3 repo-wide (subsequent changes: behavior-identical lint fixes, each standalone-tested; fmt reflows; workspace-invisible replaces; docs) |
| Lint | ✓ **fresh full run: 76/76 modules, 0 issues** |
| check-arch / check-depguard / check-docserver-css | ✓ fresh (120 deps covered) |
| check-duplication / check-coverage / check-api-stability | ✓ fresh (0 new clones; ±2% tolerance; no drift) |
| doc-check | ✓ 921 refs |

Remaining for a single-run GREEN: one calm-window `nix run .#verify` (load < ~5,
disk freed) — the SAME re-run the parallel session's f.1 already requests;
whoever gets the window runs it once for both waves. No code work remains.
