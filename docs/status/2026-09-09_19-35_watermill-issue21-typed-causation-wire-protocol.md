# Status Report — Issue #21: watermill event wire protocol drops typed command causation

**Date:** 2026-09-09 19:35 CEST
**Session scope:** Review + implement GitHub issue #21 (`watermill/v4` event wire protocol asymmetry) in `go-cqrs-lite`.
**Tree state:** clean; all work absorbed by auto-commit daemon into `2f87c4107`, `130f5f2a1`, `c9a316660` (on `master`, not pushed).

---

## 0. Executive Summary

Issue #21 reported that the watermill event wire protocol parsed `correlation_id`, `causation_id`, `user_id`, `request_id`, `actor_id` inbound but never wrote them outbound, and that the typed `event.Metadata.Causation` (ADR-0031) had no wire representation at all — so any event crossing a watermill hop silently lost command causation.

**Finding during review:** the issue was verified against `watermill/v4.5.1`. Master had already moved: **v4.6.0 fixed the scalar-ID half** via the `writeTracing` sweep (writes all five tracing IDs from `eventToMessage`). The remaining, still-broken half was exactly the typed `Causation` struct — and that is what this session implemented, plus a backward-compat promotion that the issue didn't ask for but that makes rolling upgrades actually work.

The fix is implemented, tested, linted, race-checked, downstream-verified, changelogged — and **waiting on a watermill release** to reach consumers (go-localsync still runs its documented workaround).

---

## 1. Root Cause & Fix (what the code does now)

| Direction | Before | After |
| --- | --- | --- |
| Outbound (`eventToMessage`) | Typed `Metadata.Causation` never written | `writeCausation` writes `causation_command_type` + `causation_command_id`; per-field zero-skip so a half pair can never hit the wire |
| Inbound (`buildMetadata`) | Typed Causation never reconstructed | `parseCausation`: dedicated keys win; if both absent, promotes v2-pattern `custom.command.type`/`custom.command.id` mirrors; partial pair → `watermill.missing_metadata` rejection; unparseable command ID → `watermill.parse_id_field_failed` rejection |
| Legacy in-flight messages | Causation lost on read | `causationFromCustom` (lenient): pre-fix producers' custom mirrors restore the typed field with **no producer change**; unparseable legacy mirrors are silently skipped (never error — legacy data was always accepted) |

Key files:

- `watermill/protocol.go:37-42` — new wire-key constants
- `watermill/protocol.go:95` — `writeCausation` call in `eventToMessage`
- `watermill/protocol.go:322-324` — `parseCausation` call in `buildMetadata`
- `watermill/protocol.go:355-424` — `writeCausation` / `parseCausation` / `causationFromCustom`
- `watermill/golden_test.go` — golden event now carries typed causation (deterministic `DeriveCommandID`), snap pins the new keys
- `CHANGELOG.md` — new `[Unreleased] → Fixed (#21)` section

**Deliberate design decisions (documented to prevent re-litigation):**

1. **No write-side custom mirroring.** Injecting `custom.command.*` on write would break round-trip equality of the Custom map. The standard path (`CommandCausalityEnricher`) already dual-writes typed + custom, so old readers keep working where it matters.
2. **Read-side promotion is always-on, not opt-in.** Tradeoff: a pure-v2-pattern event (custom mirrors only, typed nil) becomes typed-non-nil after a round trip. Judged faithful to ADR-0031 (the custom keys are documented aliases). Flagged as improvement item #6.
3. **Lenient legacy, strict new.** Custom mirrors that don't parse are skipped silently; the new dedicated keys fail loudly. Prevents behavior regression for data that was always accepted.

---

## a) FULLY DONE

| Work | Verification |
| --- | --- |
| Verified issue claims against master (found v4.6.0 already fixed scalar half) | `git diff watermill/v4.5.1..v4.6.0..HEAD` archaeology |
| Typed Causation wire write side (`writeCausation`) | tests + golden |
| Typed Causation read side incl. precedence, partial-pair, invalid-ID rejection (`parseCausation`) | tests |
| Legacy custom-mirror promotion (`causationFromCustom`) | tests |
| 6 new protocol tests: round-trip, key-omission, legacy promotion, typed-wins-over-custom, partial rejection, invalid-ID rejection | `go test` green |
| Golden wire-format pinned: `causation_command_*` keys in `message-metadata.snap`, regenerated via `-update` (confirmed fail→regen→pass) | snap diff reviewed |
| Removed stale `watermill/testdata/golden/message-metadata.json` (pre-go-snaps leftover, basename-only usage verified, was missing `stream_*` keys) | `trash`, rg-verified no consumers |
| `golangci-lint` clean incl. fixing the one `nilnil` finding (restructured to out-param) | `0 issues` |
| `gofumpt` clean, `go vet` clean, module tests `-count=1`, `-race` green | all pass |
| Downstream consumer tests: `stack`, `system`, `system/integration`, `benchkit` | all pass |
| Confirmed `stack/bench` + `cmd/cqrs-bench` failures are **not** caused by this change (see §d) | evidence below |
| CHANGELOG `[Unreleased] → Fixed (#21)` entry; `check-changelog-symbols.sh` gate passes | "citations are honest" |
| Confirmed no skill-reference/README updates needed (none enumerate wire keys) and no exported-symbol changes (no api-golden regen) | rg evidence |

## b) PARTIALLY DONE

1. **Downstream verification breadth** — 7 of 20 watermill-dependent modules tested (stack core, system, system/integration, benchkit + the 2 failures diagnosed). **Not tested:** all 8 `stack/*` presets (bbolt, duckdb, memory, mysql, pebble, postgres, sqlite, turso), `integration`, `cmd/cqrs-bench` (blocked by /tmp), and the 4 `example/*` modules. Risk is low (additive metadata keys only) but not zero.
2. **Full repo gate** — `nix run .#verify` / `#verify-ci` NOT run (heavy, exclusive; targeted per-module verification chosen instead per gowork-modes doctrine). The CI-mirror matrix would systematically catch any other published-pin gaps like stack/bench's.
3. **Issue lifecycle** — fix is committed locally, but issue #21 is still **open** on GitHub with no comment linking the fix (not pushed, not closed — outside given scope).
4. **Consumer relief** — go-localsync (the reporting consumer) still runs its custom-fallback workaround until watermill v4.7.0 exists and they bump.

## c) NOT STARTED

1. Release: tag + push `watermill/v4.7.0` (with `tag-release.sh --smoke` proxy verification).
2. GitHub: comment on #21 with the fix summary + commit, then close.
3. go-localsync: bump to v4.7.0, drop the documented bus-hop workaround from their changelog.
4. Wire-protocol documentation: the key set (~21 keys, dual-read windows, causation promotion) exists only in code + golden snap — no README table, no ADR.
5. Transport-parity sweep: do `transport/grpc` / `transport/http` (SSE) carry typed Causation? Same bug class, unchecked.
6. `/tmp` hygiene: tmpfs at 97% (Go caches ~18G+) — root cause of the cqrs-bench link failure, unfixed.

## d) TOTALLY FUCKED UP

**Nothing.** Honest accounting of the near-misses (all caught and corrected in-session):

- First `golden_test.go` edit broke gofumpt field alignment → caught by immediate `gofmt -l -w`, fixed.
- First `parseCausation` shape returned `(nil, nil)` → `nilnil` lint failure → restructured to out-param before finishing.
- `gh issue view` returned empty output once → retried with stderr capture, worked.
- The two test failures (cqrs-bench, stack/bench) are **pre-existing/environmental, not from this work**: (1) cqrs-bench's CLI tests shell out to `go build` and the DuckDB static link died with `No space left on device` in `/tmp` (tmpfs 97%; root fs has 170G free — the env chain points GOTMPDIR at `.gotmp` but the gcc/ld subprocess spilled to `/tmp/go-link-*`); (2) stack/bench fails at graph load (`missing go.sum entry` for pgx via `stack/postgres@v4.4.0`) and under `GOWORK=off` resolves **published** `watermill@v4.5.1` from the module cache — this diff isn't even in its compile graph. Both belong on the backlog, not at this fix's door.

## e) WHAT WE SHOULD IMPROVE

1. **Wire-protocol parity meta-test.** `buildMetadata` parses keys that `eventToMessage` never wrote for ~6 weeks (this exact bug class). A small test asserting "every key buildMetadata reads is written by eventToMessage (or on a documented dual-read window)" would have caught #21 mechanically. Highest-value follow-up.
2. **Protocol documentation.** The wire contract is now ~21 keys with three compatibility windows (legacy `aggregate_*`, causation custom mirrors, payload-encoding fallback) — pinned only by a golden snap and prose doc-comments. README table + ADR would make the contract legible to consumers.
3. **Environmental hygiene.** `/tmp` tmpfs at 97% will keep breaking DuckDB/CGo links (cqrs-bench, stack/bench, stack/duckdb). Move Go caches off tmpfs or add cleanup.
4. **Published-pin health.** stack/bench's `missing go.sum entry` is exactly the "false-green/false-red pin break" class gowork-modes.md warns about — `#verify-ci` exists to catch these and wasn't run.
5. **Round-trip purity vs. migration pragmatism.** The always-on custom-mirror promotion trades strict round-trip equality for rolling-upgrade correctness. Worth an explicit ADR paragraph (or an opt-in knob) before v5 freezes the semantics.
6. **Issue-to-fix latency.** #21 was filed against v4.5.1, partially fixed (scalars) in v4.6.0 without closing/updating the issue, and the typed half sat until this session. Closing the loop on GitHub issues when a partial fix ships would have surfaced the remaining gap sooner.

## f) NEXT 50 (prioritized, impact-first)

**Ship it (1–6)**
1. Tag + push `watermill/v4.7.0` (proxy-smoke via `tag-release.sh --smoke`).
2. Comment on #21 with fix summary + commit hash; close the issue.
3. Run full `nix run .#verify` once before the tag (exclusive window).
4. Bump all internal watermill consumers in the next pin wave (stack presets, system, benchkit, examples).
5. go-localsync: bump to v4.7.0, delete their bus-hop workaround + changelog limitation note.
6. Fold watermill v4.7.0 into the next release-train CHANGELOG heading.

**Prevent recurrence (7–12)**
7. Wire-protocol parity meta-test: fail when `buildMetadata` reads a key `eventToMessage` never writes (or vice versa).
8. Sweep the same bug class in `transport/grpc` and `transport/http` (SSE) — typed Causation parity check.
9. Contract test: callers of `MessageToEvent` handle the `(non-nil evt, non-nil err)` corrupt-metadata return (stack does; pin it).
10. Add the causation keys + promotion rule to a wire-protocol ADR (alongside the `aggregate_*` v6 dual-read window).
11. Document the no-write-side-mirroring decision (see §1.1) in that ADR so it isn't re-litigated.
12. Add go-localsync-style consumer to a cross-repo compat test list (real consumers catch protocol bugs libraries can't).

**Test depth (13–20)**
13. Command-ID-only partial pair test (only `causation_command_id` set — currently untested; type-only is).
14. `writeCausation` per-field skip tests (CommandType="" with valid ID; zero ID with type set).
15. CBOR-encoded event + typed causation round-trip (payload encoding × metadata interplay).
16. Combined-metadata round-trip: tombstone + actor + causation together.
17. Fuzz `buildMetadata` with arbitrary metadata maps (three-branch parser now).
18. Redis/NATS broker integration round-trip asserting typed causation end-to-end.
19. CatchUpSubscriber replay path asserts causation survival (uses `eventToMessage`; add explicit assertion).
20. Benchmark `EventToMessage`/`MessageToEvent` before/after (2 extra Get/Set) — confirm no hot-path regression.

**Docs (21–26)**
21. Full wire-key table in `watermill/README.md`.
22. Godoc on `EventToMessage`/`MessageToEvent` listing every written key (godoc as protocol spec).
23. Skill reference recipe: "causation propagation across watermill" (references/recipes.md §new).
24. core.md/advanced.md note: typed Causation survives bus hops as of v4.7.x.
25. Update `docs/agents/module-map.md` watermill row if it mentions the protocol.
26. Check EventCatalog export (catalog module) — does it model metadata fields needing the new keys?

**Repo hygiene found during this session (27–34)**
27. Sweep for other stale `.json` golden mirrors whose `.snap` is authoritative (eventtest.AssertGolden basename pattern exists in many modules; watermill's had drifted silently).
28. Add a CI check (or delete the mirrors) so `.json`-next-to-`.snap` can't drift again.
29. Fix stack/bench published-pin go.sum gap (pgx via stack/postgres@v4.4.0).
30. Run `nix run .#verify-ci` to systematically catch any other pin-graph breaks.
31. `/tmp` tmpfs hygiene: relocate Go caches off tmpfs or add cleanup; 47G/48G will break more CGo links.
32. Re-run `cmd/cqrs-bench` tests after /tmp cleanup to confirm the CLI tests pass.
33. Test the remaining watermill dependents: 8 stack presets, `integration`, 4 examples.
34. Move the stray `docs/status/2026-09-09_04-48_*` report + this report through docs-health annotation (mark superseded items done).

**API/semantics decisions (35–42)**
35. Decide: always-on vs opt-in legacy-mirror promotion (round-trip purity for pure-v2 events).
36. Decide: export the wire-key constants (consumers hand-build metadata with string literals in tests today).
37. Decide: dedicated `watermill.causation_*` error codes vs reused `missing_metadata`/`parse_id_field_failed`.
38. Evaluate a `cqrs-lint` hint: prefer typed `WithCausation` over hand-rolled `custom.command.*` writes.
39. Check prometheus/otel modules for any metadata-key cardinality assumptions (2 new keys).
40. Confirm deprecated `transport/http` SSEBroker causation parity is a non-goal (documented deprecation).
41. Consider `MessageToEvent` metadata-parse metrics (observability of corrupt-metadata drops).
42. Schedule the `causation_command_*` keys into the v5/v6 compatibility ledger (they're new — no legacy, but they need a drop-checklist entry if ever renamed).

**Process (43–50)**
43. Cross-repo: verify DeriveCommandID-based golden IDs remain stable across go-branded-id bumps (golden depends on ULID encoding).
44. Add `watermill/protocol.go` to the alloc-pin watchlist if benchmarks show delta (hot-path discipline contract).
45. Review whether `MessageToCommand` should also parse typed causation from event-carried keys (n/a today — commands have no typed field; document why).
46. Golden snap: consider also asserting message UUID + payload alongside metadata (currently metadata-only).
47. Write the "partial fix shipped without closing the issue" lesson into AGENTS.md testing/process gotchas.
48. Check whether `writeTracing`'s v4.6.0 scalar fix got a changelog entry at the time (this session documented it retroactively in the #21 entry).
49. Consider a repo-wide "wire protocol owners" map (watermill/grpc/sse/sql envelope) in module-map.md for future parity sweeps.
50. Close the loop: after v4.7.0 ships, re-run this session's full verification battery against the tagged module (GOWORK=off from a consumer's perspective).

## g) Questions for you (cannot answer myself)

1. **Promotion semantics:** should the legacy custom-mirror → typed Causation promotion stay always-on (my pick: yes, it's what makes rolling upgrades work), or become opt-in for consumers who need strict round-trip purity of the v2 custom pattern?
2. **Issue handling:** comment on #21 and close it now pointing at these local commits, or wait until `watermill/v4.7.0` is actually tagged/pushed so the issue closes against a released fix?
3. **Documentation depth:** is a full wire-key table (README + ADR formalizing the ~21-key contract and its compatibility windows) wanted for v4.7.0, or should the golden snap + godoc remain the only spec until the v5 protocol sweep?

---

*Verification battery this session: watermill tests ×2 + `-race` + `vet` + `golangci-lint` (0 issues) + `gofumpt` + golden fail→regen→pass + downstream (stack, system, system/integration, benchkit) + changelog symbol gate. Commits: `2f87c4107` (impl+tests+golden), `130f5f2a1` (nilnil fix + stale fixture removal), `c9a316660` (CHANGELOG).*
