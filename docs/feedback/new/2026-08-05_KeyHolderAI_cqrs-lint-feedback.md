# cqrs-lint — Consumer Feedback (KeyHolderAI, Round 1)

**Consumer:** [KeyHolderAI](https://github.com/larsartmann/KeyHolderAI) — Multi-AI chat interface (11 AI Keyholder personas), Go 1.26 + Gin + Templ + HTMX v2 + SSE + CQRS
**cqrs-lint version:** 4.3.0 (commit: 7f4c911, built: 20260803220640)
**go-cqrs-lite version:** v4.2.0 (command/query/id direct; dispatcher/event/idempotency/metadata/codec indirect)
**cqrs-htmx version:** v4.6.1
**Date:** 2026-08-05
**Previous feedback:** None (first round for this consumer)
**Findings this session:** 24 total (1 real defect fixed, 23 false positives / intentional / deferred)

---

## TL;DR

Five issues, in priority order:

1. **C031 flags correct error-returning code as a bug** (FALSE POSITIVE, source-confirmed). The rule matches the literal `return nil` token without parsing that the _second_ return value — the error — is non-nil. This fires on the canonical Go error-wrapping pattern `return nil, wrappedErr` inside `(any, error)` handlers. **This is the most dangerous false positive class: it trains developers to ignore real C031 bugs.**

2. **F007/A016 suggests `middleware.CommandIdempotency()` which does not exist** (IMAGINARY API). Same class of bug as DiscordSync's P012 (`storage.SQLiteEnableWAL`). The vendored `command/v4` exports `Middleware func(Handler) Handler` and `idempotency/v4` exports `MemoryStore`, but there is **no pre-built middleware wiring them together**. The linter points consumers at a function they must hand-roll.

3. **D005 false-positives on multi-module repos with mixed direct/indirect sub-modules** (DESIGN GAP). `go-cqrs-lite` is a multi-module repo: `command/v4`, `query/v4`, `id/v4` are direct imports (no `// indirect`), while `dispatcher/v4`, `event/v4`, `idempotency/v4`, `metadata/v4`, `codec/v4` are transitive (`// indirect`). D005 sees _any_ `// indirect` on a `go-cqrs-lite/*` line and concludes the version is "indirect" — even when the docs match the direct imports exactly.

4. **`server: false` is wrong — feature detection misses `http.Server` + `ListenAndServe`** (DETECTOR MISS). KeyHolderAI is a real Gin HTTP server (`main.go:171` `srv := &http.Server{`, `main.go:181` `srv.ListenAndServe()`). The profile reports `server: false`, which then suppresses server-correct rules and enables local-only rules incorrectly.

5. **S006 fires and then immediately admits it shouldn't** (SELF-CONTRADICTING). Every S006 finding's own suggestion text reads: _"This appears to be a local-only project (no HTTP/gRPC server). Add encryption if this data may be exposed to networks."_ If the linter can detect it's local-only, it shouldn't emit the finding — or should emit at `DEBUG` level, not `INFO`.

---

## Consumer Context

KeyHolderAI is a **CQRS-without-event-sourcing** application by deliberate design (documented in `AGENTS.md` §Known Issues). It uses:

- `command/v4` + `query/v4` dispatchers (direct, via `dispatch/` package)
- `id/v4` for strongly-typed IDs
- `cqrs-htmx/v4` as the HTTP transport layer (Gin adapter)
- **No event sourcing** — in-memory state with optional SQLite persistence
- **SSE** for streaming (not through CQRS dispatch — direct writer access)
- Commands are dispatched **only from tests today**; HTTP handlers call services directly (incremental CQRS migration in progress)

This means several "missing wiring" findings (A018, A025, E009) are factually correct observations about absent features, but the _suggestions_ are misleading because they assume the consumer wants full event sourcing.

---

## Issue 1: C031 flags correct error-returning code (FALSE POSITIVE)

### Severity: HIGH (false-positive on the canonical Go error pattern)

### What happens

Three query handlers in `dispatch/dispatcher.go` use the standard Go error-wrapping pattern:

```go
func(ctx context.Context, getQuery *GetSimulationQuery) (*models.Simulation, error) {
    sim, err := cfg.SimService.GetSimulation(ctx, getQuery.SimulationID)
    if err != nil {
        return nil, crerrors.Wrapf(err, "simulation %s not found", getQuery.SimulationID)
    }
    return sim, nil
},
```

cqrs-lint reports C031 on each `return nil, crerrors.Wrapf(...)` line:

```
WARNING dispatcher.go:259:6 Error is checked but handler returns nil — the command/query appears successful when it actually failed [C031]
  Suggestion: Return the error: `return fmt.Errorf("handler: %w", err)`
  |> return nil, crerrors.Wrapf(err, "simulation %s not found", getQuery.SimulationID)
```

### Why this is wrong

The query handler contract is `func(context.Context, Query) (any, error)` — verified in vendored `query/v4/dispatcher.go:26`:

```go
type Handler = func(context.Context, Query) (any, error)
```

The flagged line returns **two values**: `nil` (the result) and `crerrors.Wrapf(err, ...)` (a **non-nil error**). This is the canonical, correct Go error-return pattern. The dispatcher's `Dispatch` method propagates the non-nil error to the caller. The query **does not** "appear successful" — it returns an error.

The C031 detector appears to match the literal token `return nil` without parsing the full return statement. For a multi-value return `(any, error)`, `return nil` is the _result_ value; the error is the _second_ value and is non-nil.

### The suggestion is actively harmful

> Suggestion: Return the error: `return fmt.Errorf("handler: %w", err)`

This suggestion proposes replacing a typed `(*models.Simulation, error)` return with a single `error` — which **would not compile** because the handler signature requires two return values. A developer who trusts the suggestion and attempts the fix will get a compile error.

### Root cause (inferred)

The C031 detector was likely written for single-value command handlers (`type Handler func(ctx context.Context, cmd Command) error`), where `return nil` after an error check IS a real bug (swallowed error). But the same detector fires on query handlers (`(any, error)`) where `return nil, err` is correct.

### The fix

The detector must distinguish:

1. **Command handlers** (`func(...) error`): `return nil` after `if err != nil` IS a bug. Keep firing.
2. **Query handlers** (`func(...) (any, error)`): `return nil, wrappedErr` is correct. Check whether the **second** return value is a non-nil error expression. If so, do NOT fire.

At minimum, the detector should count the return values in the `return` statement. If there are two values and the second is an error expression (not the literal `nil`), the finding is a false positive.

### Impact

Every consumer with typed query handlers that wrap errors will get this false positive. This is the **canonical pattern** for `RegisterTyped[Q Query, R any]` handlers — the exact API the library recommends. The linter is flagging the library's own recommended usage pattern as a bug.

---

## Issue 2: F007/A016 suggests `middleware.CommandIdempotency()` which does not exist (IMAGINARY API)

### Severity: HIGH (points consumers at a non-existent function)

### What happens

cqrs-lint reports:

```
WARNING dispatcher.go:48:13 Command dispatcher has no idempotency middleware — duplicate commands cause duplicate side effects under at-least-once delivery [F007]
  Suggestion: Add middleware.CommandIdempotency(store, ttl, nil) to your command dispatcher's Use() chain. Requires an idempotency.Store (MemoryStore for single-process, KVStore/SQLStore for distributed).

WARNING dispatcher.go:48:1 Command dispatcher lacks idempotency middleware — duplicate commands may execute twice [A016]
  Suggestion: Add middleware.CommandIdempotency(store, ttl, nil) to your dispatcher
```

### The suggested function does not exist

I verified the vendored `command/v4` and `idempotency/v4` packages:

**`command/v4/handler.go`:**

```go
type Handler func(ctx context.Context, cmd Command) error
type Middleware func(Handler) Handler
```

**`idempotency/v4/store.go`:**

```go
type Store interface {
    Seen(ctx context.Context, key string) (bool, error)
    Record(ctx context.Context, key string, ttl time.Duration) error
    CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error
}
type MemoryStore struct { ... }
func NewMemoryStore(sweepInterval time.Duration) *MemoryStore
```

There is **no `middleware` package** and **no `CommandIdempotency` function** anywhere in the vendored `go-cqrs-lite/v4.2.0`. The linter suggests an API that the consumer would have to hand-roll from `command.Middleware` + `idempotency.Store`.

This is the **same class of bug** as DiscordSync's P012 report (2026-08-04 feedback), where `storage.SQLiteEnableWAL` was suggested but did not exist.

### What the linter should do

Two options:

1. **If the function exists in a newer version** (v4.3.0+): the linter should detect the consumer's pinned version and only suggest APIs available in that version, OR prominently note "requires go-cqrs-lite ≥ vX.Y.Z".

2. **If the function does not exist yet**: the linter should suggest the actual building blocks:
   ```
   Suggestion: Implement idempotency using command.Middleware + idempotency.MemoryStore.
   Key commands on cmd.ID() and call store.CheckAndRecord() before dispatch.
   ```

### Impact

Any consumer without idempotency middleware gets pointed at a phantom function. This erodes trust in all other suggestions — if this one is imaginary, which others are?

---

## Issue 3: D005 false-positives on multi-module repos with mixed direct/indirect sub-modules

### Severity: MEDIUM (noisy after the real drift is fixed)

### What happens

`go-cqrs-lite` is a multi-module Go repo. KeyHolderAI's `go.mod` lists:

```
github.com/larsartmann/go-cqrs-lite/command/v4 v4.2.0            // DIRECT (no marker)
github.com/larsartmann/go-cqrs-lite/query/v4 v4.2.0              // DIRECT (no marker)
github.com/larsartmann/go-cqrs-lite/id/v4 v4.2.0                 // DIRECT (no marker)
github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0 // indirect
github.com/larsartmann/go-cqrs-lite/idempotency/v4 v4.2.0 // indirect
github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.2.0 // indirect
github.com/larsartmann/go-cqrs-lite/codec/v4 v4.2.0 // indirect
```

After I fixed the real D005 defect (AGENTS.md said `v2`, go.mod has `v4`), the linter emitted a **new, spurious D005**:

```
WARNING AGENTS.md:1:1 AGENTS.md references go-cqrs-lite v4.2.0 but go.mod has indirect [D005]
  Suggestion: Update documentation to match the version in go.mod
```

The AGENTS.md version (`v4.2.0`) **exactly matches** the direct imports. The linter sees `// indirect` on the _transitive sibling_ modules (`dispatcher/`, `event/`, `idempotency/`, `metadata/`, `codec/`) and concludes the version is "indirect" — even though the _directly imported_ modules (`command/`, `query/`, `id/`) carry no indirect marker.

### Why this is wrong

`// indirect` on a `go.mod` line means "not directly imported by this module's packages." In a multi-module repo like `go-cqrs-lite`, each sub-module (`command/`, `query/`, `event/`, etc.) is a separate Go module with its own `go.mod`. A consumer can directly import `command/v4` while `event/v4` is pulled in transitively by `command/v4`'s own dependencies. Both are at `v4.2.0`. The `// indirect` marker is about **import topology**, not **version resolution**.

D005 conflates the two: it treats _any_ `// indirect` on a matching version line as evidence that the documented version is wrong.

### The fix

D005 should check the **direct** imports (lines without `// indirect`) for the version, not any line matching the module path prefix. If the direct-import version matches the documentation, the finding should not fire — regardless of what the indirect siblings say.

---

## Issue 4: `server: false` is wrong — feature detection misses `http.Server` + `ListenAndServe`

### Severity: MEDIUM (cascades into wrong rule selection)

### What happens

The detected feature profile reports:

```
server:        false
server-local:  false
```

But KeyHolderAI IS a server. From `cmd/keyholderai/main.go`:

```go
srv := &http.Server{           // line 171
    Addr:    cfg.ServerAddr,
    Handler: engine,
}
err := srv.ListenAndServe()    // line 181
```

The app uses Gin (`gin.New()`) wrapped in a standard `http.Server` with `ListenAndServe`. This is a textbook HTTP server.

### Impact

`server: false` causes two cascading problems:

1. **E009 fires** ("No HTTP/gRPC transport layer") — false positive, because the linter doesn't recognize Gin + cqrs-htmx as a transport layer. The transport clearly exists.
2. **Server-only rules are suppressed** — rules that should validate server-specific patterns (graceful shutdown, request timeouts, etc.) don't run.

### Root cause (inferred)

The feature detector likely looks for specific patterns like `http.ListenAndServe(":8080", ...)` or `gin.Default().Run()`, but doesn't recognize the `srv := &http.Server{...}; srv.ListenAndServe()` construction pattern, or the Gin engine passed as `Handler`.

### The fix

Broaden the server detection to recognize:

- `http.Server` struct literal construction
- `.ListenAndServe()` method calls on any `*http.Server` variable
- Gin's `engine.Run()` / `engine.ListenAndServe()`
- Any call to `http.ListenAndServe` or `http.Serve`

---

## Issue 5: S006 fires and then admits it shouldn't (self-contradicting)

### Severity: LOW (noise, but corrosive to trust)

### What happens

Seven S006 findings fire on wager-related command/query structs. Each finding's suggestion text reads:

> This appears to be a local-only project (no HTTP/gRPC server). Add encryption if this data may be exposed to networks.

The linter **has already determined** the project is local-only (derived from `server: false`) — yet it emits the finding anyway at `INFO` level.

### Why this is a problem

1. **Self-contradicting**: if the linter knows it's local-only, the encryption recommendation is conditional ("if exposed to networks") — so it shouldn't fire unconditionally.
2. **Compounds with Issue 4**: `server: false` is wrong (see above), so the "local-only" determination is itself based on a false premise. KeyHolderAI does serve over HTTP.
3. **The data is virtual game tokens**: the wager structs carry in-game token amounts, not real currency. But even setting that aside, the self-contradiction is the real issue.

### The fix

If `server: false` AND `server-local: false` (no network exposure), either:

- **Suppress S006 entirely** for local-only projects, or
- **Emit at DEBUG level** (not INFO), or
- **Rephrase** to a single aggregate finding: "Project has unencrypted financial-data fields and is local-only. If this will ever be exposed to a network, add encryption." (one finding, not seven)

---

## Rule-Level Feedback

### C031 — `nil-return-on-error`

**Verdict:** FALSE POSITIVE on query handlers. See Issue 1.

The detector must distinguish single-return command handlers (`func(...) error`) from multi-return query handlers (`func(...) (any, error)`). For the latter, `return nil, nonNilErr` is correct.

---

### D005 — `doc-version-mismatch`

**Verdict:** Caught a REAL defect (v2 → v4 drift), then FALSE-POSITIVED on multi-module indirect markers. See Issue 3.

The first pass was genuinely useful — AGENTS.md was stale. The second pass (post-fix) is noise.

---

### F007 / A016 — `missing-idempotency-middleware`

**Verdict:** Legitimate concern, **IMAGINARY suggested API**. See Issue 2.

The observation (no idempotency middleware) is factually correct. The suggestion (`middleware.CommandIdempotency(...)`) points at a non-existent function.

---

### E009 — `missing-transport`

**Verdict:** FALSE POSITIVE. Transport exists via Gin + `cqrs-htmx` (`httputil/` package, `cqrshtmx.App`, `WrapHandler`, `WrapMiddleware`).

The linter only recognizes `transport/http` or `transport/grpc` directory patterns. It should also recognize `cqrs-htmx` as a transport layer (it's the library's own HTTP framework).

---

### A018 — `dead-import`

**Verdict:** MISLEADING. The finding says "Project imports go-cqrs-lite but never calls Save/Publish — possible dead import or missing wiring."

The import is NOT dead — `command.Dispatcher` and `query.Dispatcher` are actively used for dispatch. Only the **event-sourcing subset** (Save/Publish) is unused. The linter conflates "doesn't use event sourcing" with "dead import."

**Suggestion:** Split into two findings:

- `A018` (dead import) — only fire if NO go-cqrs-lite API is called at all
- `A025` (no event sourcing) — already exists, covers the "Save/Publish not called" case

---

### A025 — `missing-event-sourcing`

**Verdict:** CORRECT OBSERVATION, but the suggestion is one-sided.

> Suggestion: Consider adding event sourcing (event/ + decider/) for audit trail, replay, and temporal queries, or keep as-is if CQRS-without-ES is intentional

The "or keep as-is if intentional" escape hatch is appreciated. This is the **best-worded finding** in the output. More like this please.

---

### A009 — `missing-stack-preset`

**Verdict:** N/A. The `stack/` package is not vendored (not imported by this consumer). Manual wiring is deliberate for this project's complexity.

The finding is harmless INFO, but the suggestion ("Use stack/sqlite.New(dsn) for one-call setup") assumes the consumer wants opinionated defaults. For a project with custom SSE streaming and a multi-service architecture, the stack preset would hide too much.

---

### B004 — `consider-cqrs-gen`

**Verdict:** COSMETIC NOISE (×7).

Every command with 3+ fields gets a B004 "consider using cqrs-gen to generate constructors." KeyHolderAI has hand-written constructors with validation (`NewCreateSimulationCommand`, etc.). The generated constructors would lack the validation logic.

**Suggestion:** If constructors already exist for a command type (detected via `New<CommandName>` function), don't suggest cqrs-gen.

---

## Feature Profile Analysis

### Detected profile (current)

```
store:         sqlite       ← CORRECT (modernc.org/sqlite)
command-flow:  commands     ← CORRECT (command dispatcher used)
server:        false        ← WRONG (Gin + http.Server + ListenAndServe)
soft-delete:   false        ← CORRECT
tracing:       off          ← CORRECT
snapshot:      off          ← CORRECT
domain:        unknown      ← CORRECT (chat/roleplay, not financial)
transport:     true         ← CORRECT (cqrs-htmx)
server-local:  false        ← WRONG (derived from wrong server=false)
```

### What `server: false` gets wrong

KeyHolderAI has a real HTTP server (`main.go:171-181`). The `server: false` detection causes:

- E009 to fire (false positive — transport exists)
- S006 to hedge with "appears to be local-only" (wrong — it serves over HTTP)
- Server-correctness rules (shutdown, timeouts) to be suppressed

Fixing `server: true` would eliminate E009 and reclassify S006 from "local-only hedge" to "real HTTP exposure — consider encryption."

---

## Suppression Configuration: Not Yet Adopted

KeyHolderAI has **no `.cqrs-lint.json`** today. This is a consumer gap (same as cqrs-htmx's round-1 feedback). Based on this session's triage, the appropriate config would be:

```json
{
	"rules": {
		"disable": ["B004"]
	}
}
```

And inline suppressions on the three C031 false positives (once the parser reliably supports them — see cqrs-htmx round-2 feedback on the end-of-line suppression bug):

```go
//cqrs-lint:ignore(C031) query handler returns (any, error); nil result + non-nil error is correct
return nil, crerrors.Wrapf(err, "simulation %s not found", getQuery.SimulationID)
```

The remaining findings (A009, A018, A025, E009, S006, F007, A016) should resolve once the detector issues above are fixed, rather than being suppressed.

---

## Summary of Recommendations

| #   | Issue                                                 | Type      | Fix effort                    | Impact                                                 |
| --- | ----------------------------------------------------- | --------- | ----------------------------- | ------------------------------------------------------ |
| 1   | C031 false positive on `(any, error)` returns         | BUG       | Medium (parse return arity)   | HIGH — flags the library's recommended handler pattern |
| 2   | F007/A016 suggests non-existent `CommandIdempotency`  | IMAGINARY | Small (fix suggestion text)   | HIGH — erodes trust in all suggestions                 |
| 3   | D005 indirect-marker false positive on multi-module   | DESIGN    | Small (check direct lines)    | MEDIUM — noisy after real drift is fixed               |
| 4   | `server: false` misses `http.Server`+`ListenAndServe` | DETECTOR  | Medium (broaden patterns)     | MEDIUM — cascades into E009/S006 false positives       |
| 5   | S006 self-contradicts on local-only projects          | DESIGN    | Small (suppress or DEBUG)     | LOW — noise, but corrosive                             |
| 6   | E009 doesn't recognize `cqrs-htmx` as transport       | DETECTOR  | Small (add pattern)           | MEDIUM — false positive on every cqrs-htmx consumer    |
| 7   | A018 conflates "no event sourcing" with "dead import" | DESIGN    | Small (split finding)         | LOW — misleading wording                               |
| 8   | B004 fires when constructors already exist            | DETECTOR  | Small (check for `New<Type>`) | LOW — cosmetic noise (×7)                              |

---

## What's Great

Credit where due — these aspects worked well:

- **D005 caught a real version-drift defect.** AGENTS.md was genuinely stale (`v2` vs `v4`). The first finding was accurate and actionable. Only the post-fix residual is a false positive.
- **A025's escape hatch wording.** "…or keep as-is if CQRS-without-ES is intentional" is the gold standard for adoption-coaching findings. It informs without preaching. Every F-series and A-series finding should follow this pattern.
- **Feature profile output.** Printing the detected profile (`server:`, `store:`, `command-flow:`, etc.) makes debugging _why_ a rule fired much easier. The `doctor` command (if available) would be even better.
- **Comma-separated suppressions.** `//cqrs-lint:ignore(C031,D005)` on one line — haven't needed it yet but the design is right.
- **Stale-suppression detection.** Haven't triggered it yet (no suppressions to go stale), but the feature prevents the classic lint-comment-rot problem.
- **`--show-suppressed` flag.** Transparency about what's hidden is excellent DX.

---

## Appendix: Verification Commands

```bash
# Version check
cqrs-lint version
# → cqrs-lint 4.3.0 (commit: 7f4c911, built: 20260803220640)

# Full strict run (pre-fix)
cqrs-lint --strict --verbose --show-suppressed
# → 24 findings (1 real defect, 23 false positive / intentional / deferred)

# Full strict run (post-D005-fix)
cqrs-lint --strict --verbose
# → 24 findings (D005 re-fires with different message — see Issue 3)

# Confirm C031 handler contract
grep "type Handler" vendor/github.com/larsartmann/go-cqrs-lite/query/v4/dispatcher.go
# → type Handler = func(context.Context, Query) (any, error)

# Confirm CommandIdempotency does not exist
grep -r "CommandIdempotency" vendor/github.com/larsartmann/go-cqrs-lite/
# → (no output)

# Confirm server construction
grep -n "ListenAndServe\|http.Server" cmd/keyholderai/main.go
# → 171: srv := &http.Server{
# → 181: err := srv.ListenAndServe()
```

---

## Appendix: Full Finding Triage

| Code | Count | Severity | Verdict            | Notes                                             |
| ---- | ----- | -------- | ------------------ | ------------------------------------------------- |
| D005 | 1     | WARNING  | REAL then FALSE    | v2→v4 drift (real); indirect-marker (false)       |
| C031 | 3     | WARNING  | FALSE POSITIVE     | `return nil, nonNilErr` in `(any, error)` handler |
| F007 | 1     | WARNING  | IMAGINARY API      | `middleware.CommandIdempotency` doesn't exist     |
| A016 | 1     | WARNING  | IMAGINARY API      | Same as F007                                      |
| E009 | 1     | INFO     | FALSE POSITIVE     | Transport via Gin + cqrs-htmx                     |
| A018 | 1     | INFO     | MISLEADING         | Conflates no-ES with dead import                  |
| A025 | 1     | INFO     | CORRECT            | Best-worded finding in the output                 |
| A009 | 1     | INFO     | N/A                | stack/ not vendored; manual wiring deliberate     |
| S006 | 7     | INFO     | SELF-CONTRADICTING | Fires then says "appears local-only"              |
| B004 | 7     | INFO     | COSMETIC           | Constructors already exist with validation        |
