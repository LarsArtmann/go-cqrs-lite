# Review: KeyHolderAI cqrs-lint Feedback (Round 1)

**Source feedback:** [`new/2026-08-05_KeyHolderAI_cqrs-lint-feedback.md`](../new/2026-08-05_KeyHolderAI_cqrs-lint-feedback.md)
**Date reviewed:** 2026-08-05
**Outcome:** 7 of 8 actionable issues addressed. Issue 2 (F007/A016 "imaginary API") is a consumer-side misunderstanding — `middleware.CommandIdempotency` exists and the suggestion is correct.

---

## Issue 1: C031 false positive on `(any, error)` returns — FIXED

**Severity:** HIGH
**Status:** Fixed

**Root cause confirmed:** `isSwallowingReturn` checked if ANY return value was `nil`. For `return nil, err` in `(any, error)` query handlers, the first value (`nil`) triggered the detector even though the second value (the error) was correctly non-nil. The canonical `RegisterTyped[Q,R]` handler pattern was flagged as a bug.

**Fix:** `pkg/rules/correctness/c031.go` — `isSwallowingReturn` now returns true only when ALL results are `nil` (or bare `return`). `return nil, fmt.Errorf(...)` correctly returns false because the second value is non-nil. Added `isNilLiteral` helper.

**Tests added:** `TestC031_NoFindingWhenReturnNilWithError` (regression: `return nil, fmt.Errorf(...)` in `(any, error)` handler → no finding), `TestC031_FiresWhenBothResultsNil` (`return nil, nil` → still fires).

---

## Issue 2: F007/A016 suggests `middleware.CommandIdempotency()` — NOT A BUG (consumer misunderstanding)

**Severity:** N/A
**Status:** No change needed — the function exists

**Analysis:** The consumer verified their vendored `go-cqrs-lite/v4.2.0` and concluded the function doesn't exist. However, the consumer does not vendor the `middleware/v4` module at all (they don't import it yet). The function **does exist** at `middleware/idempotency.go:57`:

```go
func CommandIdempotency(
    store idempotency.Store,
    ttl time.Duration,
    keyExtractor func(command.Command) string,
) command.Middleware
```

The linter correctly suggests this function. The consumer's confusion stems from not having the middleware module in their vendor directory — they need to add `middleware/v4` to their `go.mod` to use it.

**Action:** No code change. The suggestion is accurate. A future improvement could note "requires importing middleware/v4" in the suggestion text for consumers who haven't vendored it yet.

---

## Issue 3: D005 false-positives on multi-module indirect markers — FIXED

**Severity:** MEDIUM
**Status:** Fixed

**Root cause confirmed:** `readGoModCQRSVersion` took `parts[len(parts)-1]` from the first matching go.mod line. On `// indirect` lines, the last field was `"indirect"`, not the version. The finding message literally said `"go.mod has indirect"`.

**Fix:** `pkg/rules/consistency/d003_d005.go` — `readGoModCQRSVersion` now:

1. Strips `// ...` comments before field-splitting (so `"indirect"` is never the last field).
2. Prefers direct imports (lines without `// indirect`) over indirect ones. In multi-module repos, direct imports (`command/v4`, `query/v4`) carry the authoritative version; indirect siblings (`dispatcher/v4`, `event/v4`) are transitive topology, not version signals.
3. Falls back to indirect version if no direct import exists.

**Tests added:** `TestReadGoModCQRSVersion_PrefersDirectOverIndirect`, `TestReadGoModCQRSVersion_FallsBackToIndirect`, `TestReadGoModCQRSVersion_OldBugReturnedIndirect` (regression: ensures `"indirect"` is never returned as a version).

---

## Issue 4: `server: false` misses `http.Server` + `ListenAndServe` — FIXED

**Severity:** MEDIUM
**Status:** Fixed

**Root cause:** The detector already checked for `ListenAndServe` as a method name (which should catch `srv.ListenAndServe()`), but this wasn't sufficient for all consumers. HTTP framework imports (Gin, Echo, Fiber, Chi) were not recognized as server signals.

**Fix:** `pkg/analyzer/feature_detect.go` — added:

1. **HTTP framework import detection:** `isHTTPFrameworkImport()` checks for `gin-gonic/gin`, `labstack/echo`, `gofiber/fiber`, `go-chi/chi`. Any of these imports sets `hasHTTPFramework=true`, which resolves `HasServer=true`. `net/http` is intentionally excluded (too broad — HTTP clients import it too).
2. **Gin `Run` method detection:** `method == "Run" && hasHTTPFramework` sets `HasServer=true`. This catches Gin's `engine.Run(":8080")` pattern.
3. Both Pass 1 (import-based) and Pass 1b (AST-based) now check framework imports.

**Tests added:** `TestDetectFeatures_GinImportDetectsServer`, `TestDetectFeatures_HttpServerListenAndServe` (regression: `srv := &http.Server{}; srv.ListenAndServe()`).

---

## Issue 5: S006 self-contradicts on local-only projects — FIXED

**Severity:** LOW
**Status:** Fixed

**Root cause:** S006 fired for all tiers (STRONG/MEDIUM/WEAK) even on local-only projects (`!HasServer`), then downgraded to INFO with a hedge: "This appears to be a local-only project." This was self-contradicting — if it knows it's local-only, it shouldn't fire (at least not for weak signals).

**Fix:** `pkg/rules/security/s006.go` — WEAK-tier findings (generic monetary lexemes like `amount`, `price`, `balance`) are now suppressed entirely when `!HasServer`. STRONG and MEDIUM tiers still fire at INFO + Low confidence (card numbers and payment-related fields warrant attention even in CLI tools). This eliminates the 7-finding noise on wager structs while keeping real payment-instrument detection.

---

## Issue 6: E009 doesn't recognize `cqrs-htmx` as transport — ALREADY HANDLED

**Severity:** MEDIUM
**Status:** No code change needed (detection already present); suggestion text updated

**Analysis:** The `cqrs-htmx` transport detection was already present in `feature_detect.go` (lines 142-145): `strings.Contains(path, "cqrs-htmx")` sets `HasTransport=true`. The consumer was on cqrs-lint v4.3.0 which likely predates this check.

**Fix:** Updated E009 suggestion text to mention `cqrs-htmx` as a valid transport option alongside `transport/http` and `transport/grpc`.

---

## Issue 7: A018 conflates "no event sourcing" with "dead import" — FIXED

**Severity:** LOW
**Status:** Fixed

**Root cause:** A018 fired whenever Save/Publish/AppendBatch was absent, even when command/query dispatch was actively used. The consumer's CQRS-without-ES architecture (using `command.Dispatcher` + `query.Dispatcher`) was flagged as a "dead import."

**Fix:** `pkg/rules/api/a015_a019.go` — A018 now also checks for `Dispatch`, `DispatchTyped`, `RegisterTyped`, `RegisterQuery`, and `NewDispatcher` calls. If any dispatch activity is found, A018 is suppressed (the import is NOT dead — A025 already covers the "consider event sourcing" coaching case). Updated message to say "never calls Save/Publish/Dispatch" and lowered confidence from High to Medium.

---

## Issue 8: B004 fires when constructors already exist — FIXED

**Severity:** LOW
**Status:** Fixed

**Root cause:** B004 fired for every command with 3+ fields, suggesting `cqrs-gen`. The consumer had hand-written constructors with validation (`NewCreateSimulationCommand`), which generated constructors would lack.

**Fix:** `pkg/rules/boilerplate/b004_b008.go` — B004 now scans all top-level function declarations for `New*` patterns. If a `New<CommandName>` or `New<CommandName>Command` function exists, the finding is suppressed. Lowered confidence from High to Medium.

---

## Summary

| #   | Issue                                         | Status          | Files Changed                                  |
| --- | --------------------------------------------- | --------------- | ---------------------------------------------- |
| 1   | C031 false positive on `(any, error)` returns | Fixed           | `c031.go`, `c031_test.go`                      |
| 2   | F007/A016 "imaginary API"                     | Not a bug       | None — function exists                         |
| 3   | D005 indirect-marker false positive           | Fixed           | `d003_d005.go`, `d005_internal_test.go`        |
| 4   | `server: false` misses HTTP frameworks        | Fixed           | `feature_detect.go`, `feature_profile_test.go` |
| 5   | S006 self-contradicts on local-only           | Fixed           | `s006.go`                                      |
| 6   | E009 doesn't recognize cqrs-htmx              | Already handled | `e008_e011.go` (suggestion text only)          |
| 7   | A018 conflates no-ES with dead import         | Fixed           | `a015_a019.go`                                 |
| 8   | B004 fires when constructors exist            | Fixed           | `b004_b008.go`                                 |
