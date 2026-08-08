# Status Report: Pre-Existing Failures — CBOR Encoding Bugfix + API-Stability Guard

**Date:** 2026-08-08 03:04
**Session scope:** Fix pre-existing failures discovered during `EnsureCustom` deprecation verification
**Commits this session:** `eff01d04c`, `b8525e6a8`, `ae2457623` (3 commits, auto-committed by daemon)

---

## a) FULLY DONE

### 1. api-stability tool compilation — RESOLVED (pre-existing, already fixed)

The `collectExports` undefined error in `cmd/api-stability/main.go:172` was **already resolved** by intervening daemon commits. `collectExports` is defined at `collect.go:15`. The tool compiles cleanly with `go build -tags "goexperiment.jsonv2"`.

### 2. api-stability golden regeneration — RESOLVED (already current)

The golden at `docs/api_surface.txt` is current at 3807 exports, including `event/method WithCustom` (line 932). The `--update` idempotent test confirms stability.

### 3. api-stability compile guard meta-test — DONE

Added `TestToolCompiles` in `cmd/api-stability/main_test.go:109-123`. Unlike `TestAPISurfaceCheck` (which skips when the golden is missing), this test runs unconditionally and builds the tool with `go build`. Catches the class of breakage where a helper function is deleted but the golden-based test is skipped.

### 4. Watermill CBOR test failures — ROOT CAUSE FIXED (4 tests)

**Tests fixed:**
- `TestRoundTrip` — payload arrived as CBOR-encoded `P{"name":"Alice"}` instead of `{"name":"Alice"}`
- `TestMessageToEvent_DefaultsJSONWhenNoEncoding` — encoding was `"cbor"` instead of `"json"` for old messages
- `TestEventToMessage_PreservesEncoding/json` — JSON sub-case got CBOR encoding
- `TestEventPublisher_RoundTripCBOR` — `DecodePayloadAuto` failed: "cannot unmarshal byte string into Go value"

**Root cause was TWO bugs, not test drift:**

#### Bug 1: `event.New` silently discarded `WithEncoding` (event/event_new.go:64)

```go
// BEFORE (broken):
enc := c.Encoding()
evt := buildEvent(eventType, streamID, streamType, version, data, opts)
evt.encoding = enc  // unconditional overwrite

// AFTER (fixed):
enc := c.Encoding()
evt := buildEvent(eventType, streamID, streamType, version, data, opts)
if evt.encoding == "" {   // respect explicit WithEncoding
    evt.encoding = enc
}
```

`buildEvent` applies options via `for _, opt := range opts { opt(evt) }`, so `WithEncoding(JSON)` would set `evt.encoding = "json"`. Then `New` overwrote it with `c.Encoding()` (CBOR from DefaultCodec). This meant any caller passing `[]byte` payload + `WithEncoding(JSON)` got an event claiming to be CBOR with JSON bytes.

**Also affected:** `encryption/crypto_helpers.go:72` — decrypted events used `WithEncoding(evt.Encoding())` but got CBOR stamped regardless. The encryption tests passed by coincidence (they tested with default codec = CBOR, so the bug was invisible).

#### Bug 2: `MessageToEvent` didn't default to JSON (watermill/protocol.go:135)

```go
// BEFORE (broken): no else branch → event.New filled in DefaultCodec (CBOR)
if enc := md.Get(metaPayloadEncoding); enc != "" {
    opts = append(opts, event.WithEncoding(codec.Encoding(enc)))
}

// AFTER (fixed): explicit JSON fallback for old messages
if enc := md.Get(metaPayloadEncoding); enc != "" {
    opts = append(opts, event.WithEncoding(codec.Encoding(enc)))
} else {
    opts = append(opts, event.WithEncoding(codec.EncodingJSON))
}
```

Old Watermill messages that predate the `payload_encoding` metadata field carry JSON payloads. Without the explicit fallback, `event.New` applied DefaultCodec (CBOR) to the encoding stamp, making `DecodePayloadAuto` try CBOR decode on JSON bytes.

### 5. Regression test for `WithEncoding` precedence — DONE

Added `TestNew_WithEncodingRespectedForRawBytes` in `event/codec_test.go:484-506`. Passes `[]byte` payload with `WithEncoding(JSON)`, asserts encoding is JSON (not CBOR from DefaultCodec).

### 6. Full test suite verification — GREEN

`nix run .#test` — all 81 modules pass, zero failures. Verified encoding-sensitive modules individually: event, watermill, encryption, schema, signing, codec, system.

---

## b) PARTIALLY DONE

Nothing. All items from the task list are fully resolved.

---

## c) NOT STARTED

Nothing from the original task list. However, see section (e) for improvements identified during this session.

---

## d) TOTALLY FUCKED UP

### Daemon committed with empty message

Commit `b8525e6a8` has **no commit message at all** — the auto-commit daemon committed the core bugfix (event_new.go fix + watermill protocol fix + regression test) with an empty subject line. This is the daemon's behavior, not mine, but it means git history has an empty-message commit for a significant bugfix. Cannot be fixed without rewriting history (which we don't do).

### What I DID NOT verify

- **Did not run `nix run .#lint`** — only ran `go vet` on the two affected modules. The full lint gate (`nix run .#lint`) was not run. Changed files were formatted with `gofumpt` but not through the full treefmt pipeline.
- **Did not run `nix run .#verify`** — the session used `nix run .#test` + targeted `go vet` instead of the full verify gate (build + vet + test + race + lint + doc-check + doc-assertions). The test gate was green, but lint and doc-check were not verified this session.
- **Did not check for other latent `WithEncoding` victims** — I checked `encryption/crypto_helpers.go` and confirmed it passes, but did not do a comprehensive sweep of ALL production callers of `event.New` with `WithEncoding`. There are ~100 call sites of `event.New` across the repo; most pass typed payloads (not `[]byte`) and don't use `WithEncoding`, but a systematic audit would be more rigorous.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture / Code Quality

1. **The `event.New` encoding override bug was a latent landmine.** The unconditional `evt.encoding = enc` line silently overrode explicit options for months. The fix is correct but the pattern of "apply options then overwrite" is fragile. Consider: should `New` and `NewEvent` converge? `NewEvent` (raw bytes path) correctly respects `WithEncoding` because it uses `buildEvent` directly without the overwrite. `New` adds the overwrite for auto-stamping but broke the option contract. A cleaner design: pass the encoding decision into `buildEvent` or use a sentinel.

2. **The watermill `MessageToEvent` fallback was implicit.** The original code said "defaults to JSON" in a comment but relied on `event.Encoding()` returning JSON when empty — which it does (`event.go:111-113`). But `event.New` overwrote the empty value with CBOR before `Encoding()` was ever called. The bug was an interaction between two modules' assumptions about defaults. Document this coupling or eliminate it.

3. **Test coverage gap: no test exercised `event.New` + `[]byte` + `WithEncoding`.** The regression test I added catches this now, but it should have existed when DefaultCodec changed from JSON to CBOR. The codec-default change was a cross-cutting concern that needed systematic test sweeps across all consumers.

4. **The `TestToolCompiles` guard should be a pattern.** Other tools (`cmd/cqrs-lint`, `cmd/cqrs-gen`, `cmd/cqrs-bench`, `cmd/doc-check`) could benefit from similar always-run compile guards. Currently their tests exercise functionality but may skip when preconditions are missing.

### Process

5. **The original report said "pre-existing, confirmed via git stash."** But 2 of the 4 "pre-existing failures" (api-stability compile + golden stale) were already fixed by daemon commits between sessions. The report was stale. Rule: re-verify before trusting a prior session's failure list.

6. **No `nix run .#verify` this session.** The AGENTS.md explicitly warns about "stale GREEN" anti-patterns. I ran `nix run .#test` which covers build + test, but not lint, race, doc-check, or doc-assertions. This is a process gap.

---

## f) Up to 50 things to get done next

### High Priority (this session's follow-ups)

1. Run `nix run .#verify` to confirm full gate (lint, race, doc-check, doc-assertions) is green after the encoding fix
2. Run `nix run .#lint` specifically — `gofumpt` was applied but not the full `treefmt` + `golangci-lint` pipeline
3. Audit ALL production callers of `event.New` with `WithEncoding` for correctness (encryption is confirmed; check signing, storage, transport)
4. Add compile-guard meta-tests to `cmd/cqrs-lint`, `cmd/cqrs-gen`, `cmd/cqrs-bench`, `cmd/doc-check` (same pattern as `TestToolCompiles`)
5. Check if the `b8525e6a8` empty-message commit can be amended via the daemon config (prevent future empty-message commits)

### Medium Priority (encoding/codec system)

6. Systematic sweep: search for ALL places that assume JSON default codec and verify they still work with CBOR default
7. Add an integration test that exercises the full JSON→CBOR→JSON roundtrip through event store → bus → watermill → projection
8. Consider unifying `event.New` and `event.NewEvent` encoding handling — both should respect `WithEncoding` equally
9. Document the three-layer codec default system (event.New → DefaultCodec, kv.TypedStore → CBOR, stack.ReadModel → CBOR) in a single reference
10. Check if `storage/pebble`, `storage/bbolt`, `storage/eventstore` handle mixed JSON+CBOR event streams correctly (self-describing envelope)
11. Verify `transport/http/sse` CBORToJSONTransform works with the fixed encoding stamps
12. Check `projectionhost` golden tests for codec-default assumptions

### Low Priority (cleanup / hardening)

13. The gopls warning `record/v4 should be direct` on `metaengine/graphadapter/go.mod:20` persisted throughout the session — verify with `go mod tidy` in graphadapter
14. The graphadapter/adapter_test.go was modified at conversation start (pre-existing `M` in git status) — verify those changes are intentional and committed
15. Consider adding a `TestCodecDefaultMatrix` that creates events with every codec combination (JSON payload + CBOR default, CBOR payload + JSON default, etc.) and verifies roundtrips
16. The `event.New` doc comment now says "If payload is []byte or json.RawMessage, it is used directly (no marshaling)" — verify `jsontext.Value` is mentioned or handled correctly
17. Run `go mod tidy` across all modules to catch any stale indirect deps
18. Check if the `WithCustom` deprecation work that triggered this investigation has any remaining loose ends

---

## g) Questions I CANNOT figure out myself

### 1. Should `event.New` auto-stamp encoding from DefaultCodec for `[]byte` payloads?

**Currently (after fix):** If payload is `[]byte` and no `WithEncoding` is given, `New` stamps `DefaultCodec.Encoding()` (CBOR) on the event — even though `[]byte` payloads bypass marshaling entirely. The bytes could be anything. Is this intentional? The alternative: stamp JSON for `[]byte` payloads when no `WithEncoding` is given (matching `NewEvent`'s historical behavior), since raw bytes are most likely JSON from external sources. This is a semantic decision about what the default means for pre-marshaled payloads.

### 2. Is the empty-message commit `b8525e6a8` from the auto-commit daemon acceptable, or should we reconfigure the daemon?

The daemon committed a significant bugfix with no subject line. This makes `git log` unreadable for that entry. If the daemon can't be configured to require messages, we may need to accept this or add a pre-commit hook that rejects empty messages (which might break the daemon entirely).

### 3. Was the `metaengine/graphadapter/adapter_test.go` modification at conversation start expected?

Git status showed `M metaengine/graphadapter/adapter_test.go` at the start of this session. I did not touch this file. It appears to have been committed by `eff01d04c` ("test(api): add compile guard and update graphadapter dependencies"). Was this an intentional change from a prior session, or unexpected drift that got swept up by the daemon?
