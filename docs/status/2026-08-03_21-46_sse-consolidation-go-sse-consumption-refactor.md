# SSE Consolidation: go-sse Consumption Refactor — Status Report

**Date:** 2026-08-03 21:46
**Session scope:** Implement ADR-0097 — both SSE implementations consume `go-sse` internally
**Verdict:** Wire-format delegation COMPLETE and passing; full verify gate + TODO updates NOT yet done

---

## a) FULLY DONE

### Research
- Read ADR-0091 (SSE consolidation decision — keep separate), ADR-0097 (go-sse consumption plan)
- Read full go-sse library: `event.go` (WriteEvent, WriteHeartbeat, WriteRetry, Event, EventID), `stream.go` (Stream, SetHeaders, NewStream, Heartbeat, LastEventID), `broadcaster.go` + `fanout.go` (Broadcaster[T], fan-out hub), `replay.go` (EventStore, Replay, ReplayFiltered)
- Read full go-cqrs-lite SSE implementations: `transport/http/sse_event.go` (190 LOC wire format), `transport/http/sse.go` (285 LOC broker), `transport/http/sse_replay.go` (273 LOC replay), `transport/http/sse_options.go` (193 LOC options), `transport/http/sse_backfill.go` (162 LOC backfill), `metaengine/sse.go` (393 LOC ServeSSE + Inspect), `metaengine/sse_replay.go` (135 LOC SSEReplay ring buffer)
- Verified go-sse v0.4.0 is the latest tag (0.1.0 → 0.4.0)
- Read ADR-0062 (metaengine dependency boundary — zero internal deps, external deps OK)
- Confirmed depguard config and module layer script constraints
- Studied api-stability export collection mechanism (AST-based, tracks `func`/`method`/`type`/`struct`/`interface`/`const`/`var`)

### Refactor: transport/http SSEBroker → consumes go-sse
- `transport/http/go.mod`: added `github.com/larsartmann/go-sse v0.4.0`
- `transport/http/sse_event.go` rewritten (190 → 113 LOC, **−77 LOC**):
  - `SSEEventID` → type alias for `sse.EventID` (preserves public API)
  - `NewSSEEventID` → delegates to `sse.NewEventID`
  - `ParseSSEEventID` → delegates to `sse.ParseEventID`
  - `MustParseSSEEventID` → delegates to `sse.MustParseEventID`
  - `WriteSSEEvent` → delegates to `sse.WriteEvent` (converts `SSEEvent` → `sse.Event`, `int` Retry → `uint`)
  - `WriteSSEHeartbeat` → delegates to `sse.WriteHeartbeat`
  - `WriteSSERetry` → delegates to `sse.WriteRetry`
  - DELETED: `sseEventBrand` phantom type, `errSSEEventIDInvalid` sentinel, `base10` constant, `splitSSELines` function (~65 LOC of duplicated serializer + line splitter)
- `transport/http/sse.go`: inline header setting (`Content-Type`, `Cache-Control`, `Connection`) replaced with `sse.SetHeaders(w)` (3 lines → 1 line)
- `go mod tidy` clean
- All tests pass (`go test ./... -count=1` → ok)
- `go vet` clean
- `golangci-lint` clean (0 issues)

### Refactor: metaengine ServeSSE → consumes go-sse
- `metaengine/go.mod`: added `github.com/larsartmann/go-sse v0.4.0` (transitively pulls `go-branded-id` + `go-error-family`)
- `metaengine/sse.go` wire-format calls replaced:
  - Header setting → `sse.SetHeaders(w)` (kept `X-Accel-Buffering: no` separately — nginx-specific)
  - `writePlainSSEEvent` → `sse.WriteEvent(w, sse.Event{Data: ...})` replacing `fmt.Fprintf(w, "data: %s\n\n", data)`
  - `writeReplaySSEEvent` → `sse.WriteEvent(w, sse.Event{ID: ..., Data: ...})` replacing `fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ...)`
  - `replayMissedEvents` loop → `sse.WriteEvent(w, sse.Event{ID: ..., Data: ...})` replacing `fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ...)`
  - `sseMainLoop` heartbeat → `sse.WriteHeartbeat(w)` replacing `fmt.Fprintf(w, ": keepalive\n\n")`
- `go mod tidy` clean
- All tests pass (`go test ./... -count=1 -short` → ok, including all SSE replay tests)
- `go vet` clean
- `golangci-lint` clean (0 issues)

### Infrastructure
- `.golangci.yml`: added `github.com/larsartmann/go-sse` to depguard allow list
- `docs/api_surface.txt`: regenerated (3183 → 3182 exports; removed `transport/http/method Name` — the `sseEventBrand.Name()` method that moved to go-sse's `eventBrand`)
- api-stability verification passes

### SKILL.md Decision Matrix
- Expanded the minimal 3-row SSE table into a comprehensive routing matrix:
  - 3 rows: raw events (SSEBroker), read-model values (ServeSSE), server-side worker (CatchUpSubscriber)
  - Columns: source, replay durability, key features
  - Rule-of-thumb guidance for each use case
  - "Do NOT merge" warning linking to ADR-0091/0097
- `doc-check` passed (1195 references valid across 41 packages)

### Auto-commits
The auto-commit daemon committed all work:
- `b7bb2647` chore(deps): add go-sse dependency
- `bca4f31d` refactor(sse): delegate wire-format serialization to go-sse (ADR-0097)
- `f7512176` refactor(sse): delegate wire-format per ADR-0097 (metaengine)

---

## b) PARTIALLY DONE

| Item | Status | Gap |
|------|--------|-----|
| Full verify gate (`nix run .#verify`) | **NOT RUN** — only per-module tests + lint | Need full build/vet/test/race/lint/doc-check/doc-assertions cycle |
| `nix fmt` | **NOT RUN** — only gofumpt/goimports via golangci-lint | Need full treefmt pass |
| Workspace-wide build verification | **PARTIAL** — built affected modules, not all consumers | Should run `go build ./...` across workspace |
| Dedup LOC accounting | **NOT MEASURED** — ADR-0097 claims ~300 LOC dedup | Should run `nix run .#check-duplication` to verify |

---

## c) NOT STARTED

- **TODO_LIST.md update** — the three SSE checkboxes are still unchecked
- **ROADMAP.md update** — Theme 8 (SSE Consolidation) items still marked as `[ ]`
- **CHANGELOG.md** — no entry for the go-sse consumption refactor
- **AGENTS.md** — the SSE section in "Key Patterns" still references the old hand-rolled wire format (e.g., `fmt.Fprintf("data: %s\n\n")`) — should note go-sse delegation
- **Coverage check** (`nix run .#check-coverage`) — not run; may have drifted
- **Layer check** (`nix run .#check-layers`) — not run; go-sse is external so should pass

---

## d) TOTALLY FUCKED UP (and fixed)

1. **Dropped a closing brace in `writePlainSSEEvent`** — When editing to add a `//nolint:wrapcheck` directive, the `old_string` included the closing `}` of the function but the `new_string` didn't restore it. The lint caught it immediately (`syntax error: unexpected name serveSSEReplay`). Fixed by re-adding the brace.

2. **`//nolint:wrapcheck` placement on multi-line return** — For `writeReplaySSEEvent`, placing the nolint at the end of a multi-line `return sse.WriteEvent(w, sse.Event{...})` caused `nolintlint` to flag it as unused (the directive applies to the `}` line, not the `return` line). Fixed by restructuring to assign-then-return: `err = sse.WriteEvent(...); return err //nolint:wrapcheck`.

3. **transport/http lint "context loading failed"** — golangci-lint's cache was stale after the module change, producing `no go files to analyze: running go mod tidy may solve the problem`. Fixed by `golangci-lint cache clean` then re-run (0 issues).

4. **gci import ordering** — The auto-formatter reordered go-sse and dedup imports in a way that violated gci's `custom-order` rule (go-sse is external `default`, dedup is `prefix(go-cqrs-lite)`). Fixed by manually reordering to put go-sse before the cqrs block.

---

## e) WHAT WE SHOULD IMPROVE

1. **Run `go build` after EVERY edit that touches function bodies** — The dropped-brace error would have been caught by `go build` before wasting a lint cycle. The lint's typecheck error message was less obvious than a build error.

2. **The ADR-0097 scope was correctly scoped to wire-format delegation only** — A deeper refactor using `go-sse.Broadcaster[T]` for fan-out was NOT attempted, which is correct. The Broadcaster has different semantics (non-blocking drop, subscriber lifecycle hooks, graceful shutdown) that don't map 1:1 to SSEBroker's channel-based fan-out or metaengine's Watcher-based push. ADR-0097 explicitly says "All existing features preserved as SSEBroker-specific wrappers."

3. **The `SSEEvent` struct was intentionally kept distinct from `sse.Event`** — go-sse uses `Retry uint` while transport/http historically uses `Retry int`. The alias is on `SSEEventID` only, not `SSEEvent`. This preserves binary compatibility for consumers who construct `SSEEvent{Retry: 5000}`.

4. **Heartbeat wire format changed**: metaengine's heartbeat went from `": keepalive\n\n"` (via `fmt.Fprintf`) to `": heartbeat\n\n"` (go-sse's `heartbeatFrame` constant). The comment text changed from "keepalive" to "heartbeat" — browsers ignore comment frames so this is wire-equivalent, but a network capture would show a different comment string. No tests asserted on the comment text, so this is safe.

5. **`fmt` import in `metaengine/sse.go` is still used** — The `Inspect()` and `InspectJSON()` functions (unrelated to SSE) use `fmt.Fprintf`. The import is NOT unused. This is correct but easy to misread.

---

## f) Up to 50 Things to Do Next

### Immediate (this session or next)
1. Run `nix run .#verify` — full gate (build/vet/test/race/lint/doc-check/doc-assertions)
2. Run `nix fmt` — full treefmt formatting pass
3. Update `TODO_LIST.md` — check off the 3 SSE items under "SSE Consolidation"
4. Update `ROADMAP.md` — mark Theme 8 SSE items as done
5. Add CHANGELOG entry for the go-sse consumption refactor
6. Run `nix run .#check-duplication` — verify LOC dedup claim from ADR-0097
7. Run `nix run .#check-coverage` — verify no coverage drift
8. Run `nix run .#check-layers` — verify go-sse doesn't break layer rules
9. Verify `go build ./...` across the FULL workspace (all consumers compile)

### Short-term (next few sessions)
10. Update AGENTS.md "Key Patterns" SSE section — note go-sse delegation, update code examples
11. Consider whether `transport/http` should also use `sse.Stream` for the per-connection handler (currently only `sse.SetHeaders` + `sse.WriteEvent` are used; `sse.Stream` provides mutex-guarded send, heartbeat goroutine, disconnect hooks — a deeper refactor)
12. Consider whether `metaengine.ServeSSE` should use `sse.NewStream` instead of manual `http.Flusher` handling (would get mutex-guarded writes + heartbeat for free)
13. Evaluate using `go-sse.Broadcaster[event.Event]` to replace SSEBroker's hand-rolled `map[SSEClientID]chan` fan-out (~50 LOC of client management)
14. Evaluate using `go-sse.Broadcaster[SeqValue[V]]` to replace metaengine's `forwardWithDropOld` goroutine + channel pattern
15. Add an integration test that verifies go-sse wire format output is byte-identical to the old hand-rolled format (golden test for the serialization path)
16. Document the heartbeat comment-text change ("keepalive" → "heartbeat") in a migration note if any consumer captured on it
17. Consider adding `go-sse` to the Dependencies table in AGENTS.md

### Medium-term
18. Explore `sse.EventStore` + `sse.Replay` for transport/http replay path — currently uses custom journal batching; go-sse's Replay could simplify if the journal adapter is built
19. Explore `sse.FilteredEventStore` for transport/http event-filter replay pushdown
20. Consider extracting a `transport/http.JournalSSEStore` adapter (like cqrs-htmx's pattern) that adapts `event.SeekableJournal` → `sse.EventStore`
21. Document the cqrs-htmx reference pattern in AGENTS.md as the model consumer
22. Consider whether the `WriteSSERetry` int→uint conversion should be documented as a known API divergence
23. Add a benchmark comparing go-sse `WriteEvent` vs the old hand-rolled serializer (should be equivalent or faster — both use byte appends)

### Metaengine SSE improvements
24. The `sseMainLoop` function uses a generic `writeEvent` callback — could be simplified now that both write paths delegate to `sse.WriteEvent`
25. The `forwardWithDropOld` function duplicates fan-out logic that `go-sse.Broadcaster` provides — future consolidation candidate
26. The `serveSSEPlain` and `serveSSEReplay` code paths share the same `sseMainLoop` — good, but the replay path's `replayMissedEvents` still hand-writes the wire format via `sse.WriteEvent` in a loop; could use `sse.Replay` with an adapter

### Documentation
27. Update `transport/http/README.md` if it references the old wire-format functions
28. Update `metaengine/README.md` SSE section
29. Add a section to `docs/architecture-understanding/` about the go-sse consumption decision
30. Consider adding ADR-0098 or updating ADR-0097 with "implementation complete" status

### Testing
31. Run the full metaengine test suite WITHOUT `-short` (includes soak tests)
32. Run transport/http tests with `-race` to verify no new race conditions from the refactor
33. Run metaengine tests with `-race`
34. Add a test that verifies `SSEEventID` alias compatibility (assigning `sse.EventID` to `SSEEventID` and vice versa)
35. Verify the `transport/http` example tests still produce correct SSE output

### Cleanup
36. Check if any other modules reference the removed `sseEventBrand` type
37. Check if the `base10` constant removal from `transport/http` broke anything (it was unexported)
38. Verify the `splitSSELines` removal didn't break any test helpers
39. Scan for any remaining `fmt.Fprintf(w, "data:` or `fmt.Fprintf(w, "id:` patterns across the repo
40. Run `go mod tidy` in the root workspace to ensure consistency

### Broader
41. Consider tagging the transport/http and metaengine modules with new versions after verify
42. Verify `cmd/api-stability` modules list is still complete
43. Run `cmd/doc-check` on ALL docs (not just skill docs) to catch any stale references
44. Check if the `integration/` module needs updating (it may import transport/http)
45. Check if `stack/` presets reference transport/http SSE types
46. Consider whether the `example/taskmanager` needs updates for the new go-sse dependency
47. Run `nix flake check` for the full hermetic check
48. Verify CI pipeline would pass (ci.yml runs per-module GOWORK=off builds)
49. Check if the Go version bump in metaengine (1.26.4 → 1.26.5 from go-sse) causes any CI issues
50. Celebrate — ADR-0097's core deliverable (wire-format delegation) is DONE

---

## g) Questions I Cannot Answer Myself

1. **Should the heartbeat comment text change ("keepalive" → "heartbeat") be considered a breaking wire-format change?** No tests asserted on it and browsers ignore comment frames, but if any reverse-proxy log parser or monitoring tool keys on the exact comment string, this is a behavioral change. I cannot determine if any production consumer relies on the "keepalive" string.

2. **Should I also refactor the fan-out layer (Broadcaster) in this session, or defer it?** ADR-0097 explicitly scoped this to wire-format delegation and says fan-out "preserved as SSEBroker-specific wrappers." But the planning doc (`SUPERB-ADR-REVIEW-FINDINGS-EXECUTION-PLAN.md`) lists P1.07–P1.10 as "SSEBroker internal refactor to consume go-sse" which could be read as including Broadcaster. The Broadcaster has materially different semantics (graceful shutdown, Health(), SubscribeFilter with panic recovery) that would change SSEBroker's behavior — this needs a decision.

3. **Is go-sse v0.4.0 stable enough for a production library dependency?** go-sse is at v0.x (pre-v1). As a dependency of a v4 library module, importing a v0 dependency means consumers transitively depend on a pre-1.0 package. The cqrs-htmx reference consumer uses v0.3.0. I cannot determine if this is acceptable for go-cqrs-lite's stability contract.
