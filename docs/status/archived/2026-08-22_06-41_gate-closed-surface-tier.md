# Status Report: Core Data Model v5 Plan — Gate Closed, Surface Tier (T09–T10) — 2026-08-22 06:41

**Session scope:** Resumed per the standing instruction (execute the 25-task plan). This session: closed the pending foundation gate (after root-causing **three** pre-existing test/gate defects it exposed), ticked the TODO_LIST boxes, shipped **T09**, and took **T10** to ~90% (all production code done, verification incomplete).

---

## a) Fully done (verified end-to-end)

1. **Pending foundation gate: CLOSED GREEN.** `nix run .#verify-fast` exit 0 on `a51903531` — build + vet + test + race + lint + doc-check + arch + duplication, all green. This was the single most important open verification from the last report. Getting there took four gate runs because the gate exposed three latent defects (d1–d3); every fix was root-caused, not retried-blind.
2. **Latent defect fix 1 — scheduling retry-delay test race** (`4a72cf2e1`). `waitFor` gated on `attempts >= 2`, but attempt 2 increments the counter *before* storing its timestamp — the test could read a zero `secondAt` and compute a bogus −56-year elapsed. Now gates on the timestamp itself. (HEAD commit of last session was already this fix; this session verified it via the gate.)
3. **Latent defect fix 2 — foundation-tier clone annotations** (`f18591491`). The tier introduced 2 art-dupl clone groups, both intentional: the command/query `AsRecord` lockstep twins (dep isolation) and the zero-dep `record.ActorKind` mirror of `id.ActorKind` (ADR-0111). Annotated with `//art-dupl:accept` directives — 0 new groups, baseline untouched.
4. **Latent defect fix 3 — transport/grpc pub/sub races** (`54ea9eccc`, `a51903531`). The three event pub/sub tests slept fixed delays around publish; under load the gRPC stream setup outlived the sleep and the fan-out silently dropped events. Exposed `EventServer.ClientCount` as the readiness observable; tests now poll for stream registration and delivery instead of sleeping. Also 5×-race green (was deterministically red under `-race` mid-fix), faster (1.1s vs 10s), and the orphaned `raceEnabled` consts are deleted.
5. **TODO_LIST harvest ticks** (`08e7820e3`). All five foundation-tier boxes in "Core Data Model v4.x/v5" ticked with dated annotations; the `Record.Encoding` box text corrected from the sketched `uint8` to the shipped `string` with the rationale.
6. **T09 — metadata capability interfaces: SHIPPED** (`9ecce6dbb` + daemon twin `0c37212ff`). `query.MetadataCarrier` + `query.PayloadCarrier` + `command.MetadataCarrier`; `query/audit.go` migrated off all three inline duck-typed assertions; compile-time conformance asserts on `*BasicQuery`/`*BasicCommand`; hand-rolled-implementation test proves external implementors can opt in. Plan **Appendix C** appended (capability-now vs interface-growth-at-v5 comparison table — g1's reading from the TODO_LIST decision). Golden 4252 exports (+3). CHANGELOG + changelog-symbols gate green. Per-module tests + lint green.

## b) Partially done

1. **T10 — decider `*Ref` forms: production code complete, verification NOT run.** On `id.StreamRef` (g2 resolved autonomously — see g1 below): `Repository.ExecuteRef`/`LoadRef` are the real implementations (internal helpers `loadFromStore`/`loadFromSnapshot`/`loadFromCache` now take the ref); pair forms `Execute`/`Load`/`LoadAtVersion`/`LoadAtTime`/`WaitForVersion` are `Deprecated: removed in v5` one-line forwarders; `TypedRepository.ExecuteCommandRef`/`LoadRef` added with deprecated pair twins; `system/register.go` migrated to `ExecuteRef` (the only production internal caller — scenario/ has zero decider Execute calls, examples go through `system.Execute`, a different layer); decider/doc.go examples updated; `.golangci.yml` SA1019 window extended with `decider`. The daemon committed the production files as `ed2aa7455`. **NOT done: `decider/ref_forms_test.go` written but never run; no `nix fmt`; no golden regen; no CHANGELOG entry; no verify.** Three files sit uncommitted in the tree.
2. **Gate debt:** no exclusive `#verify-fast` has run on the T09/T10 state (T09 passed per-module gates only).

## c) Not started (from this session's scope)

- **T10** completion (see b1) · **T11** TimerID branding + `Timer.Actor id.ActorID` (research done: consumers are only `scheduling/sqlstore` + `storage` re-export; zero example/system usage; wire compat proven — zero ActorID marshals `""`, omitzero preserves today's shape) · **T12** `record.Type` consolidation (event/command/query triplication confirmed identical; alias plan validated) · **T24** post-landing sweep · the long tail **T16–T23, T25**. Plan progress: **T01–T09 done, T10 ~90%** → 11.9 of 25.

## d) What I totally fucked up (honest log)

1. **Designed the grpc readiness signal before reading the implementation.** My first fix gated publish on `miniBus.subscriberCount() > 0` — but the bus is subscribed *eagerly* at `RegisterEventService`; the count was ≥1 from startup and the "wait" was a no-op. All three tests went deterministically red under `-race`. Correct signal (`EventServer` client registry) required reading `event_server.go` — which I should have done *before* designing. Cost: one failed 5×-race cycle.
2. **First capability-test draft was garbage.** Called unexported `requestIDOf` from the external test package and hand-rolled an inline `Tracing` struct literal with invented tags. Rewrote after checking `metadata.Metadata`'s real shape. Cause: wrote the test from memory instead of from the type definitions.
3. **Ran `nix fmt` without the env block** — its gci formatter shelled `go env` against the corrupted `/mnt/buildcache` and the whole run died after 2m12s. The env block is mandatory for *any* tooling that touches go, not just go/nix gates.
4. **Session restart wiped /tmp** (48G tmpfs → 98M used): both go caches and the in-flight gate log died with it; gate 3 re-ran from cold caches (~15 min of re-downloads). /tmp is volatile — anything valuable there must be consumed promptly.
5. **Slow gate loop:** four exclusive gate runs (~10–15 min each) where two failures (dupl groups) were catchable in seconds by running `nix run .#check-duplication` standalone *before* the full gate. Same for changelog-symbols and lint-config. The fast standalone gates should front-load the expensive one.
6. **Uncommitted work while the daemon runs** — the daemon split T09 (took my audit.go/capability changes as `0c37212ff` before my golden/CHANGELOG was ready) and committed T10's production files (`ed2aa7455`) while the test file was still being written. Both splits were verified on inspection, but the lesson repeats: commit per-module-green, immediately, not at task boundaries.

## e) What I can do better

- **Read the implementation before designing a test signal** — the counter-observable must come from the code, not from a plausible mental model.
- **Pre-flight the cheap gates** (`#check-duplication`, `check-changelog-symbols`, per-module golangci) before every exclusive `#verify-fast`.
- **Export the full env block for every command**, including `nix fmt`.
- **Commit the moment a module is green**; the daemon does not respect task boundaries.
- **Never leave a new test file unrun** (ref_forms_test.go is exactly that right now).

## f) Next steps (priority order)

1. **Finish T10:** run `decider` tests (incl. new `ref_forms_test.go`), fix what breaks, `nix fmt`, per-module golangci, golden regen, CHANGELOG entry, commit.
2. Exclusive `#verify-fast` on the T09+T10 state (cheap gates first).
3. **T11·a** `TimerMarker` + branded `TimerID` — **string-backed** (`cbid.ID[TimerMarker, string]`, the `id.StreamID` pattern), NOT the plan's `id.Of` (ULID-backed): idempotent scheduling keys on semantic names ("cancel-order-…", "delay-test"); ULID backing breaks every one of them. `ParseTimerID` sentinel. Flag as deviation (see g3).
4. T11·b `Timer.Actor` → `id.ActorID` (wire-identical: zero → `""`/omitted, non-zero → prefixed string). Keep sqlstore's `timerEnvelope.Actor` a plain string; convert at the boundary, tolerate legacy empty.
5. T11·c/d/e: `id/v4` + `go-branded-id` promoted indirect→direct in scheduling go.mod; `#check-arch`; fix store/scheduler/tests (`t.ID` SQL param rides cbid's Valuer; string concats → `.String()`); golden + CHANGELOG; consumer-pin sweep note for scheduling/sqlstore + storage.
6. **T12·a** `record/type.go`: `Type`, `String`, `IsZero`, `ParseType(s string, emptyErr error)` (parametrized sentinel) + tests.
7. T12·b/c: `type Type = record.Type` aliases in event/command/query; delete local `String`/`IsZero` (compile-error duplicates under the alias); local `ParseType` stays as deprecated thin wrapper keeping each module's sentinel error.
8. T12·d: golden + verify-fast + CHANGELOG.
9. **T24·a** api-stability meta-tests (`TestEvery*`).
10. T24·b doc-check over SKILL.md + references + AGENTS.md — **document the new record API** (Cause/Stamp/Actor/ID/Encoding, NewStreamRefOrZero, capability interfaces, decider *Ref forms, TimerID) in `.agents/skills/go-cqrs-lite/references/`.
11. T24·c consumer-pin sweep (GOWORK=off matrix) for record/metadata/id consumers.
12. Tag `record/v4` + `metadata/v4` (and event/command/query if warranted) so six modules can drop local replaces at the next pre-tag sweep.
13. T17 `snapshot.NewSnapshot` + encoding stamp (ADR-0044 style).
14. T18 snapshot wire-tag audit (pebble/bbolt/sql/memory).
15. T19 deprecation census → `docs/planning/v5-deprecation-sweep.md` (now includes the six record-bridge deprecations + decider pair forms).
16. T20 tombstone v5 deletion prep (migration doc verify + StatusMiddleware bridge coverage).
17. T21 extended review: storage/*, system/, stack/, watermill/, middleware/.
18. T22 naming review: Stream/StreamRef/StreamID/Cause/Stamp/Actor vocabulary.
19. T23 upstream skill fixes (reviews↔brainstorming divergence; read-prior-reports + copy-template steps).
20. T16 report polish (anchor integrity script, CSS diff) on the core review HTML.
21. T25 AGENTS.md memory: data-model conventions section (validating-population pattern, capability-interface rule, *Ref identity convention, string-backed-branded-ID rule).
22. TODO_LIST: tick T09/T10 boxes when verified; tick T11/T12 as they land.
23. Consider `system.Execute` public wrapper: give it a ref form at v5 too (noted during T10; not in plan scope).
24. Re-check Appendix C's stale `audit.go:86,101,114` line references after the audit.go migration (they describe the pre-T09 layout).
25. Sweep `grep -rn "\.Execute(ctx"` consumers under example/ + cmd/ for any pair-form stragglers the window would hide (belt-and-braces after T10 lands).

## g) Questions (max 3)

1. **g2 (resolved autonomously — confirm):** decider's *Ref forms take **`id.StreamRef`** (storage-layer struct). The code had already decided: every internal helper (store.Load, singleflight key, cache, snapshot) is `id.StreamRef`-keyed; `record.StreamRef` would add a parse on the hot path. Proceeding on this; veto means rework before T11.
2. **T11 shape (deviation, confirm):** plan says `TimerID = id.Of[TimerMarker]` — but `id.Of` is **ULID-backed** and timers need semantic IDs for idempotent scheduling/cancel ("cancel-order-…"; the module's own tests use "delay-test"). I will ship **string-backed** `cbid.ID[TimerMarker, string]` (the documented `id.StreamID` pattern) unless you insist on ULID-only timer IDs.
3. **g3 (still open from prior session):** `Record.Encoding` shipped as `string` (codec's self-describing form) instead of the sketched `uint8`. Confirm or veto — veto means a mechanical type swap before any record/v4 tag is cut.

---

*State at pause: HEAD `ed2aa7455` + 3 uncommitted files (`.golangci.yml`, `decider/doc.go`, `decider/ref_forms_test.go` — T10 verification incomplete, test never run). Gate green on `a51903531`; T09+T10 not yet gate-verified. Plan progress: T01–T09 done, T10 ~90%. No push performed.*

---
