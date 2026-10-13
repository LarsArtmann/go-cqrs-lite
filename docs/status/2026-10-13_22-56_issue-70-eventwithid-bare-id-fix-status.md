# Status Report — Issue #70: `EventWithID.ID` display-form fix (full session report)

**Date:** 2026-10-13 22:56 CEST
**Session scope:** LarsArtmann/go-cqrs-lite#70 + fleet companions + repo-gate repairs encountered on the way
**Trigger:** User instruction — READ/UNDERSTAND/RESEARCH/REFLECT, break into steps, execute and verify one at a time, repeat until done.

---

## 0. One-paragraph summary

Issue #70 reported that `projectionadapter.Register`/`RegisterString` stamped `evt.StreamID().String()` — the branded **display form** (`"StreamMarker:<value>"`) — into `EventWithID.ID`, a field documented as "the entity ID as the Map key". Consequence: every metaengine entity was keyed by a string no other layer uses; lookups by the raw stream value (what commands dispatch on, what `ParseStreamID` accepts, what `MarshalText` emits) **silently returned not-found**. The fix (the issue's preferred option) stamps the bare value `evt.StreamID().Get()` at both call sites, pins it with a red-first regression test (including an end-to-end Map lookup through the adapter), states the contract loudly in the godoc, repairs every doc that taught the wrong form, records the behavior change in CHANGELOG `[Unreleased]` with migration guidance, and co-updates the in-scope companion **cqrs-htmx** (53 test-line flips). Two pre-existing master breakages hit during gate verification were repaired on sight: a stale `vendorHash` that had `check-md-go` broken since the 10-12 lock bump, and a red format gate from daemon-raced unformatted test files.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| A1 | Root-cause research: `typed_decoder.go:69,94`, `id/stream_id.go` round-trip law (`ParseStreamID(id.String()) == id`; "String() is a display form"), plugmarket's custom-decoder workaround confirmed as demand evidence (`es_metaengine_installation.go:98` — `rawID := evt.StreamID().Get()`), cqrs-htmx/plugmarket/in-repo consumer census | session research phase |
| A2 | TDD red→green fix: both call sites `evt.StreamID().String()` → `evt.StreamID().Get()` | `metaengine/projectionadapter/typed_decoder.go:69,94` |
| A3 | Failing tests first — exact symptom reproduced: `ID = "StreamMarker:01M4V8VMMJSG5NM4BD8C000001"` and end-to-end Map lookup `result is <nil>, want createdPayload` (silent miss, not even an error) | `typed_decoder_test.go` — `TestTypeDecoder_Register_DecodesPayload` (updated pin) + new `TestRegister_MapLookupByBareStreamID_FindsEntity` |
| A4 | Godoc contract made loud: `EventWithID.ID` carries the BARE stream value; NOT the `String()` display form; Map entries retrievable by raw IDs from any layer; "Before" boilerplate snippet fixed | `typed_decoder.go:16-35,113` |
| A5 | Doc repair everywhere the wrong form was taught: `metaengine/README.md` EventDecoder example, `metaengine/COOKBOOK.md` manual-decoder example, recipes.md §Bridging prose (bare-value note) | `metaengine/README.md:609-625`, `metaengine/COOKBOOK.md:86-101`, `.agents/skills/go-cqrs-lite/references/recipes.md:1587-1592` |
| A6 | CHANGELOG `[Unreleased]` → **Changed** entry with full symptom, workaround history, and migration guidance (re-project via reset+replay, or strip via `ParseStreamID`; `e.ID` round-trippers unaffected) | `CHANGELOG.md` (Changed section, first bullet) |
| A7 | Companion cqrs-htmx co-update: ~53 test lines flipped from display form to bare value across `declarative_test.go`, `declarative_harness_pilot_test.go`, `declarative_harness_usertrain_test.go` — lookups, view-ID assertions, policy Subject/Key asserts, audit AggregateID asserts, AND the `NewActorID(ActorUser, streamID.String())` constructions (the actor referenced the user by its display form — same bug class, only round-tripped because the whole chain shared it) | `~/projects/cqrs-htmx/systemadapter/` (3 files; green via overlay workspace) |
| A8 | Fleet verification: overlay go.work (`/tmp/cqrs-htmx-overlay.go.work`, 31 go-cqrs-lite modules) proved cqrs-htmx red-before → green-after against the working tree; plugmarket twin/metaengine suites green (unaffected — custom decoders); go-appkit grep-verified as non-consumer | overlay run + `cd ~/projects/plugmarket/marketplace && go test -run 'Meta|Twin'` → ok |
| A9 | In-repo verification: projectionadapter full suite + `-race`, system full package, examples (getting-started, taskmanager, graph-native), doc-check (1284 refs / 57 pkgs, zero-warning policy held), recipes compile gate (`TestRecipes`), api-stability meta-tests (`TestEvery`), `check-md-go` (1511 blocks valid), format check, projectionadapter `golangci-lint` (0 issues), `check-changelog-symbols.sh` (13 citations honest) | gate outputs in session log |
| A10 | **Pre-existing breakage repaired:** `check-md-go` had been broken on master since the 2026-10-12 flake.lock bump — `md-go-validator` input moved to rev `3f466de` without re-pinning `vendorHash` (flake.nix:893 still held the 10-11 hash `c6lqyeij…`; FOD mismatch `specified c6lq… got odwV…`). Re-pinned to `sha256-odwV+4mBfAMw9ilgNWi8M495yFc+1FaU59Y2Wkl4gWE=` with dated comment; vendor-hash check + gate green ("All 1511 code blocks valid") | `flake.nix:891-893`; buildflow's `nix-hash-fix` is documented-chronically-broken here (14/14 failures), so the hand re-pin follows the documented escape |
| A11 | **Pre-existing breakage repaired:** format gate red on master — auto-commit daemon (3e141ef07, 19:02) had absorbed a concurrent session's unformatted `systemtest/time_fidelity_test.go` (+ `stack/bundle_test.go`); golines wanted the long `awaitFidelityView` signature split. `nix fmt` healed; `checks.x86_64-linux.format` green | treefmt diff + `nix fmt` output (2 files changed) |
| A12 | Scope discipline: wire/DLQ/OTel display-form uses (`system/adapter_event_serial.go`, `projectionhost/worker.go` DLQ, `adapter.go:128` span attr) classified as diagnostics-layer and deliberately left — they don't feed Map keys | session research |

Both repos' auto-commit daemons absorbed all changes; `git status` clean in go-cqrs-lite and cqrs-htmx at session end.

---

## b) PARTIALLY DONE

| # | Item | What exists | What's missing |
|---|------|-------------|----------------|
| B1 | Fleet consumer verification | cqrs-htmx **systemadapter** module fully verified via overlay | Rest of cqrs-htmx (root, identity-model, usermgmt, examples, integration_test) NOT run against the fixed tree; other fleet repos that import `projectionadapter` not swept (`rg -l projectionadapter ~/projects` was never run) |
| B2 | go-appkit check | grep-verified zero `projectionadapter`/`EventWithID` usage | Its test suite not run (probably genuinely unaffected, but unverified claim) |
| B3 | Bug-class sweep awareness | Noticed sibling display-form key sites during the grep: `storage/aggregate_projection.go:91,119` (aggregates keyed by display form!), `example/goal-shaped-app/app.go:155,174,190` (`TaskCreated{ID: cmd.StreamID().String()}` — display form into PAYLOAD IDs), `example/taskmanager/deriver.go:34` (`TaskID: evt.StreamID().String()` into workqueue payloads) | None of these investigated for the same silent-miss class; none filed |
| B4 | plugmarket awareness | Noticed `es_metaengine.go:37-40` comment ("EventWithID.ID carries the DISPLAY form") becomes FALSE after they adopt the fixed tag; noticed `docs/internal/eventwithid-display-form-upstream-draft.md` exists (their draft of exactly this upstream issue) | Did not read/update either; their workaround retirement not planned |
| B5 | Discovery documentation | The `(nil, nil)` silent-miss ergonomics finding (`store.Execute` on Map miss returns nil result AND nil error — my regression test initially tripped over `result is <nil>` with err==nil) was observed and worked around in the test | Not investigated as a product question: SHOULD a Map Get miss return a typed not-found error (errorfamily) instead of `(nil, nil)`? Not filed |
| B6 | Verification breadth | Changed-module suites, race, format, md-go, changelog-symbols, doc-check, recipes, api-stability all green | Canonical `nix run .#verify` NOT run end-to-end; `#check-arch`, `#check-coverage`, `#check-duplication`, `#vulncheck` not run; CI (GitHub Actions) status not checked after the daemon commits |

---

## c) NOT STARTED

| # | Item | Why it matters |
|---|------|----------------|
| C1 | **Issue #70 has no closure signal** — no comment posted, checkbox "MANUALLY REVIEWED by @Lars Artmann" still unticked, issue open with no link to the fix/CHANGELOG row | The filing itself asks for the fix; future readers can't connect issue → fix |
| C2 | **TODO_LIST harvest (docs-health HARVEST)** — no rows added to go-cqrs-lite `TODO_LIST.md` (projectionadapter tag needed in next wave; migration note for display-form-keyed projections) or cqrs-htmx `TODO_LIST.md` (pin bump required same wave) | A timestamped report is not a backlog home — this report itself risks the "07-37 entombment" failure without harvest |
| C3 | cqrs-htmx `systemadapter/CHANGELOG.md` row for the test flips | Their repo keeps a per-module changelog; the flips are a coupled-behavior change worth recording there |
| C4 | Skill FAQ row (`references/faq.md`): "Why does my metaengine Map lookup miss? `StreamID.String()` vs `.Get()` display law" — the silent-not-found symptom is exactly a FAQ-class pitfall | Consumers hitting the old behavior post-migration will search for exactly this |
| C5 | `cqrs-upgrade` / `cqrs-lint` detection: a suggestion-rule flagging `.StreamID().String()` feeding fold keys / `EventWithID`-shaped wrappers, pointing at `.Get()` | The fleet-wide codemod surface for affected consumers |
| C6 | Class-sweep investigation of `storage/aggregate_projection.go` (keys aggregate projections by display form) | Same silent-miss class, sibling module; if real it's a second bug of the same family as #70 |
| C7 | Class-sweep of payload-carried display-form IDs (goal-shaped-app `TaskCreated.ID`, taskmanager deriver `TaskID`) | If a view ever joins payload.ID against `e.ID` or a bare stream ID, same miss; also `DeriveStreamID` over a display string produces a DIFFERENT stream identity |
| C8 | Overlay-workspace recipe documented in `docs/agents/gotchas-module-management.md` (how to test fleet consumers against an untagged working tree: temp GOWORK file, module-dir path without `/v4` suffix pitfall) | It took two attempts to get right; the recipe is reusable and currently tribal knowledge |
| C9 | ADR/bookkeeping decision: bare-value-at-key-boundary law could merit an ADR-0111-style addendum row (cf. ADR-0111(e) actor-ID boundary law) | Today the law lives only in a CHANGELOG bullet + godoc |
| C10 | BuildFlow close-the-loop: `nix-hash-fix` failing 14/14 in this repo is documented as chronic but was never filed upstream (LarsArtmann/BuildFlow) per the buildflow skill's step-7 rule | The tool's failure is observed, not reported |
| C11 | Cross-project lesson to `crush-config/references/lessons.md`: "display-form vs value-form at ID key boundaries" (generalizes: branded-ID `String()` is for humans; keys/wire/lookups take `Get()`/`MarshalText`) | Exactly the class of lesson that generalizes across the fleet |
| C12 | `RegisterString` lacks its own end-to-end Map-lookup regression test (it shares the handler shape with `Register`, but the pair isn't pinned symmetric) | Cheap belt-and-suspenders |

---

## d) TOTALLY FUCKED UP (honest failures / self-inflicted risks)

| # | Item | Severity | Detail |
|---|------|----------|--------|
| D1 | **cqrs-htmx master is now RED against tagged go-cqrs-lite** | HIGH (deliberate but unsafeguarded) | My test flips (`findUser(ctx, userStreamID.Get())` etc.) match the FIXED working tree but MISS against the pinned `metaengine/projectionadapter/v4 v4.5.3` (verified: `TestDeclarative_TenantRoundTrip` → `system: not found` under plain `GOWORK=off`). Anyone running cqrs-htmx CI/tests before the pin bump eats 16 failures. This is the co-release convention ("they co-release in the same version waves"), but I created the coupling WITHOUT any tracking artifact: no TODO row, no issue comment, no cqrs-htmx changelog note. If the wave slips or someone bisects cqrs-htmx master, this looks like a mystery break. |
| D2 | **The bug sat in the codebase shipping silent data-miss for its entire lifetime** | HIGH (product, pre-existing) | `Register`/`RegisterString` shipped with the display form from day one; the doc example PROMISED `e.ID` was the entity key. Every consumer following the docs got silent not-founds. plugmarket had to write ~8-case custom decoders per family to escape it (their ADR-0009 finding 6). The regression test I added is the first test that ever exercised "project then look up by the real stream ID" through the documented path. |
| D3 | **I almost edited files without View (twice)** | LOW (process) | Two `edit` calls bounced with "read the file first" (COOKBOOK.md, metaengine/README.md) — I'd read them via `sed` in bash instead of the View tool. Wasted round trips; the tool guard caught it. |
| D4 | **`rg -rln` flag misuse produced garbage** | LOW (process) | `-r` is REPLACE in ripgrep; my "provenance" check initially printed mangled pseudo-output (`func ln(...)`) that briefly looked like file corruption. Re-ran correctly. |
| D5 | **Overlay workspace needed two attempts** | LOW (process) | First /tmp go.work used module paths WITH the `/v4` suffix (dirs don't have it) → 19 load errors. Should have known the dir-vs-path split cold (it's in the module map conventions). |

Nothing I shipped is broken: no test I touched is red, no gate I touched regressed, no foreign change was reverted. D1 is the one item with real teeth.

---

## e) WHAT WE SHOULD IMPROVE

1. **Close the loop on issue filings in the same session.** Fixing #70 without commenting on #70 leaves the report half-done. Procedure candidate for AGENTS.md "Close a Wave" checklist: "issue addressed → comment with fix refs, or record why not".
2. **Coupling artifacts BEFORE coupling.** When a companion-repo edit depends on an untagged dependency fix, write the tracking row (TODO_LIST both sides) in the same edit. D1 happened because the coupling lived only in my head + the session summary.
3. **Sweep the bug CLASS, not just the instance.** The grep that found the fix sites ALSO surfaced `storage/aggregate_projection.go` and payload-ID seams. A 10-minute classification pass ("key vs diagnostics") would have converted B3/C6/C7 from open questions into either "safe, documented" or "filed".
4. **Run the canonical gate, not just the pieces.** I verified 12 gates piecemeal; `#verify` is the repo's contract. Partial verification is how the 90-tag `verify=ok` lesson happened.
5. **`Execute` miss ergonomics deserve a decision.** `(nil, nil)` on a Map miss is a silent-miss ENABLER — the exact failure shape this issue punished. A typed not-found error would have turned #70's symptom into a loud signal years ago.
6. **Document the fleet-overlay testing recipe.** It's the standard way to verify consumer repos against untagged dependency work; it should be one documented command sequence, not session folklore.
7. **FAQ-first for silent failure modes.** Any fix for a "silently returns nothing" bug should ship with its FAQ row in the same change.
8. **Prefer property assertions in ID tests.** `ParseStreamID(wrapped.ID) == evt.StreamID()` (round-trip law) is stronger and more intention-revealing than string equality against `.Get()`.

---

## f) NEXT — up to 50 things to get done (rough priority order)

**Immediate closure of this fix (1–8)**
1. Comment on issue #70: fix landed, cite CHANGELOG `[Unreleased]` Changed row, the two fixed call sites, and the regression test; state the migration path. Close-or-hold per Lars's call.
2. Harvest go-cqrs-lite `TODO_LIST.md`: "tag projectionadapter in next wave; bare-value `EventWithID.ID` behavior change ships — migration note for display-form-keyed projections (reset+replay or `ParseStreamID` strip)".
3. Harvest cqrs-htmx `TODO_LIST.md`: "systemadapter tests coupled to next go-cqrs-lite projectionadapter tag — bump pin in same wave (master red against v4.5.3 until then)".
4. Add cqrs-htmx `systemadapter/CHANGELOG.md` row for the 53 test-line flips.
5. Add skill FAQ row: silent Map-miss ↔ `String()` vs `Get()` display law (+ link from recipes §Bridging).
6. Add `RegisterString` end-to-end bare-ID lookup test (symmetry with `Register`).
7. Upgrade the regression test to the round-trip-law property (`ParseStreamID(wrapped.ID) == evt.StreamID()`).
8. Run canonical `nix run .#verify` (quiet window) + check GitHub Actions CI green on master.

**Co-release wave (9–14)**
9. Decide wave contents: projectionadapter (+ nothing else changed?) → tag (e.g. v4.5.4).
10. Per Changed-set module test sweep BEFORE tagging (`GOWORK=off go test -count=1 ./...` per changed module — the verify=ok lesson).
11. Bump cqrs-htmx pin to the new projectionadapter tag in the same wave; run full cqrs-htmx suite green.
12. Refresh versions manifest via the release scripts; proxy-smoke the tag.
13. Verify pkg.go.dev shows the new version and the fixed godoc.
14. Post-wave: re-run cqrs-htmx CI to confirm green master.

**Bug-class sweep (15–24)**
15. Investigate `storage/aggregate_projection.go:91,119` — are aggregate projections keyed by display form? If yes: same silent-miss bug as #70; file + fix or document.
16. Classify every production `StreamID().String()` site as key vs diagnostics (the ~20-site grep list from this session): transport/grpc, commandlifecycle/recorder, AsRecord bridges, middleware, benchkit.
17. Investigate goal-shaped-app `TaskCreated{ID: cmd.StreamID().String()}` payload seam — does any view join payload ID against bare IDs?
18. Investigate taskmanager deriver `TaskID: evt.StreamID().String()` — what stream identity do derived `task.start` commands land on?
19. Sweep fleet repos for OTHER local `EventWithID`-like wrappers stamping `.String()` into metaengine keys (`rg -l "StreamID\(\)\.String\(\)" ~/projects --type go | xfiles-with metaengine`).
20. File the `(nil, nil)` Map-miss ergonomics question (should `store.Execute` return a typed not-found error?) — design ticket.
21. If display-form keys are confirmed anywhere else, add each to the CHANGELOG migration note (one re-projection story per key seam).
22. Consider a `cqrs-lint` rule / `cqrs-upgrade` suggestion: `.StreamID().String()` flowing into fold keys or `EventWithID`-shaped structs.
23. Decide whether `EventWithID.ID` should become branded `id.StreamID` (string today) at v5 — type-safe keys, breaking.
24. Record the bare-value-at-key-boundary law: ADR-0111 addendum row or small ADR; link from `EventWithID` godoc.

**Companion + fleet hygiene (25–32)**
25. plugmarket: read `docs/internal/eventwithid-display-form-upstream-draft.md`; update/retire it (its premise shipped).
26. plugmarket: update the now-stale `es_metaengine.go:37-40` decoder-rationale comment AFTER they bump (their cadence — ask first).
27. plugmarket: evaluate retiring the custom bare-ID decoders back to `Register`/`EventWithID` where extractor synthesis allows (their ADR-0009 finding 6 workaround cost).
28. go-appkit: run its suite against an overlay once to convert "grep says unaffected" into "tested unaffected".
29. Fleet-wide sweep: which ~/projects repos import `projectionadapter` at all? (`rg -l projectionadapter ~/projects`) — the complete affected-consumer list for the wave.
30. cqrs-htmx: full-suite overlay run (all modules, not just systemadapter) before the wave tag.
31. Check whether any fleet repo's systemscenario payload-hash goldens embed display-form stream IDs that would shift on bump (goldens keyed via `TrailWithPayloads` masking should be safe — verify one).
32. Cross-project lesson → crush-config `references/lessons.md`: "branded-ID `String()` is display; keys/wire/lookups take `Get()`/`MarshalText`" (commit there, never in-session global write).

**Tooling/repo health surfaced this session (33–42)**
33. File upstream BuildFlow issue: `nix-hash-fix` chronic failure in go-cqrs-lite (14/14), with the md-go-validator FOD case as repro (close-the-loop per buildflow skill).
34. Document the overlay-workspace recipe in `docs/agents/gotchas-module-management.md` (temp GOWORK file; strip `/v4` from module paths; requires-only-deps trick).
35. Check `check-duplication` after this session's test additions (local `findItem` type + `makeTypedEvent` helper vs adapter_test.go's `makeEvent` — dedupe candidates).
36. Consider `makeTypedEvent`/`makeEvent` consolidation in projectionadapter tests (art-dupl blind to cross-file helpers?).
37. Verify `id` module pins the `ParseStreamID(id.String()) == id` round-trip law with a test (docs claim it; confirm the pin exists).
38. Nightly bench + `benchmark-regression.sh` after the wave (no timing path touched, but the gate exists for exactly this kind of change-adjacent check).
39. `#check-arch` + `#check-coverage` + `#vulncheck` before the wave tag (release-procedure rows).
40. `check-readme-deprecated.sh` nightly green check (no deprecated claims touched, but cheap).
41. AGENTS.md "Close a Wave" checklist: add "fleet-companion coupling recorded BEFORE tagging" + "issue addressed ⇒ commented" rows.
42. Consider updating `.buildflow.yml` to keep `nix-hash-fix` in skip_steps with the documented rationale (it's 14/14; running it wastes a minute every full run) — or leave if the flake repairs keep it green.

**Docs/polish (43–50)**
43. recipes.md: extend §Bridging with the lookup-side example (project, then `Execute(Find{ID: streamID.Get()})`) — the missing half of the story.
44. metaengine README/COOKBOOK: same lookup-side example.
45. `EventWithID` godoc: add "recover the display form via `id.ParseStreamID(...).String()` when needed" pointer (the inverse migration).
46. Systemscenario docs: note that projections keyed via `EventWithID.ID` now key by bare values (affects anyone writing goldens over Map keys).
47. Review `adapter.go:128` OTel `stream_id` attribute: display form is fine for telemetry but say so in SPAN_NAMING/attr docs.
48. CHANGELOG entry: link the projectionhost Reset recipe as the concrete re-projection path (currently prose only).
49. This report → harvest leftovers into TODO_LIST (C-items), then mark report superseded per docs-health ANNOTATE discipline when rows land.
50. Decide + answer the 3 questions below — several of the above rows depend on those answers.

---

## g) Questions I cannot answer myself

1. **Issue #70 closure:** close it NOW with a comment citing the CHANGELOG row and fix, or hold it open until the projectionadapter tag actually ships (and do you want the comment authored in your github-voice, posted as me)?
2. **Wave timing:** cut the co-release wave NOW (projectionadapter tag + cqrs-htmx pin bump, full sweep per the verify=ok lesson), or park everything for the next planned train? D1's red-master window exists either way until the pin bump.
3. **Scope of the class sweep:** do you want `storage/aggregate_projection.go` (and the payload-ID seams in goal-shaped-app/taskmanager) investigated as a follow-up of this same bug family — and if confirmed, fixed in the same wave — or strictly issue-scoped and filed separately?

---

## Verification ledger (what was actually run, this session)

| Gate/suite | Result |
|---|---|
| projectionadapter `go test` (unit, `-count=1`) + `-race` | ok / ok |
| system `go test` (full package, `-short` and full) | ok |
| examples: getting-started, taskmanager, graph-native | ok ×3 |
| cqrs-htmx systemadapter via overlay (fixed tree) | ok (after 53-line flip; 16 failures before) |
| cqrs-htmx systemadapter against TAG v4.5.3 | FAIL (expected — coupling, see D1) |
| plugmarket marketplace `-run 'Meta|Twin'` | ok |
| doc-check SKILL+refs+AGENTS.md | 1284 refs valid, exit 0 |
| `TestRecipes` (recipes compile harness) | ok |
| api-stability meta-tests (`TestEvery`) | ok (no golden regen needed — no surface change) |
| `check-md-go` | green after vendorHash repair (1511 blocks, 105 baselined) |
| `checks.x86_64-linux.format` | green after `nix fmt` (2 daemon-raced files) |
| projectionadapter golangci-lint | 0 issues |
| `check-changelog-symbols.sh` | 13 citations honest |
| NOT run | `#verify` end-to-end, `#check-arch`, `#check-coverage`, `#check-duplication`, `#vulncheck`, go-appkit suite, cqrs-htmx full suite, CI status check |

_Arte in Aeternum_
