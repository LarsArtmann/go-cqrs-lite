# cqrs-lint — Consumer Feedback Round 2 (browser-history)

**Consumer:** [browser-history](https://github.com/larsartmann/browser-history) — Go CQRS/ES app that tracks browser history (Chrome/Safari/Firefox) via go-cqrs-lite v4, Huma v2 HTTP, usermgmt WebAuthn auth, templ dashboard
**Version used:** go-cqrs-lite v4.0.x (event v4.0.2, decider v4.0.1, command/query v4.0.0, watermill v4.0.2, storage v4.0.1, middleware v4.0.1, idempotency/kvstore v4.0.0, kv v4.0.1)
**lint version:** `cqrs-lint` (rebuilt from `cmd/cqrs-lint` at HEAD, 2026-08-02)
**Date:** 2026-08-02
**Previous feedback:** [2026-07-20_browser-history_cqrs-lint-feedback.md](../../2026-07-20_browser-history_cqrs-lint-feedback.md) (round 1)

---

## Executive Summary

The round-1 improvements (generic-call detection, closure handler tracing, feature-profile system) eliminated nearly all the false positives from the first pass. The remaining issues are a **suppression parser bug** that silently breaks a documented feature, and two **detection-gap false positives** (C036, S006) that lack context the linter needs.

This report covers a full `cqrs-lint --strict --verbose --show-suppressed` run against 61 Go files across 5 modules. The linter reported **0 active findings** (clean) with **23 inline suppressions**. Of those 23, only **4 were truly lazy** (fixed during this session). The remaining 19 are legitimate, but 5 of them stem from linter false positives that should be fixed upstream.

| Category                                             | Count | Action taken                                                                                         |
| ---------------------------------------------------- | ----- | ---------------------------------------------------------------------------------------------------- |
| **Fixed in code** (lazy suppressions eliminated)     | 4     | C023: Passed logger to `openEventStore`, replaced `_ = db.Close()` with `closeDBOnError(db, logger)` |
| **Confirmed false positives** (detector limitations) | 5     | Reported below for upstream fix                                                                      |
| **Legitimate constraints** (accurate suppression)    | 14    | Verified accurate, kept with corrected justification text                                            |
| **Conscious non-adoptions** (intentional)            | 4     | Reviewed, kept                                                                                       |

Signal-to-noise of the 23 suppressions: **78%** (14 are genuine constraints or false positives, 4 were lazy and now fixed, 4 are conscious non-adoptions). With the parser bug and two false positives fixed, this would drop to ~16 suppressions.

---

## Part 1: Parser Bug

### Bug 1: Suppression comment with space after `//` silently fails to suppress

**Severity: HIGH** — users write `// cqrs-lint:ignore(...)` (the natural Go comment style with a space after `//`), the linter silently accepts it without suppressing, and findings appear unsuppressed with no explanation.

#### What happens

Writing a suppression comment in the standard Go comment style:

```go
// cqrs-lint:ignore(B004) cqrs-gen code generation not adopted
type ClassifyURLCommand struct {
```

The finding for B004 still appears in the output. The suppression is silently ignored.

Changing it to (no space after `//`):

```go
//cqrs-lint:ignore(B004) cqrs-gen code generation not adopted
type ClassifyURLCommand struct {
```

Now the suppression works correctly.

#### Root cause

File: `cmd/cqrs-lint/pkg/suppression/parser.go:249-255`

```go
line = strings.TrimSpace(line)
// Accept both "//cqrs-lint:ignore" and "// cqrs-lint:ignore".
line = strings.TrimPrefix(line, "// ")
if !strings.HasPrefix(line, commentPrefix) {
    continue
}
```

Where `commentPrefix = "//cqrs-lint:ignore"` (line 16).

The comment on line 251 claims to accept both variants. But the logic is broken:

**With space (`// cqrs-lint:ignore(C007)`):**

1. `TrimSpace` → `// cqrs-lint:ignore(C007)`
2. `TrimPrefix(line, "// ")` → `cqrs-lint:ignore(C007)` (strips the `// ` prefix)
3. `HasPrefix("cqrs-lint:ignore(C007)", "//cqrs-lint:ignore")` → **FALSE** (string no longer starts with `//`)
4. `continue` — suppression not recognized

**Without space (`//cqrs-lint:ignore(C007)`):**

1. `TrimSpace` → `//cqrs-lint:ignore(C007)`
2. `TrimPrefix(line, "// ")` → `//cqrs-lint:ignore(C007)` (no change — doesn't start with `"// "`, it starts with `"//c"`)
3. `HasPrefix("//cqrs-lint:ignore(C007)", "//cqrs-lint:ignore")` → **TRUE**
4. Suppression parsed correctly

The `TrimPrefix(line, "// ")` strips the `// ` from the space variant, leaving `cqrs-lint:ignore(...)` which then fails the `HasPrefix(commentPrefix)` check because `commentPrefix` starts with `//`.

#### Impact

This bug affected **4 of 27 suppressions** in browser-history. The predecessor session (round 1) even documented the whitespace bug in its status report's "What went wrong" section but didn't trace it to a parser bug — it was attributed to user error. Multiple feedback files from other consumers may show the same pattern.

This is particularly insidious because:

1. `gofmt` and `goimports` do NOT normalize `//comment` vs `// comment` (both are valid Go comments)
2. The Go convention and most IDEs default to `// comment` (with space)
3. The linter gives no warning that the suppression was not recognized

#### Fix

```go
line = strings.TrimSpace(line)
// Normalize "// cqrs-lint:ignore" → "//cqrs-lint:ignore"
line = strings.Replace(line, "// cqrs-lint:", "//cqrs-lint:", 1)
if !strings.HasPrefix(line, commentPrefix) {
    continue
}
```

Or equivalently, remove the `TrimPrefix` line and check both prefixes:

```go
line = strings.TrimSpace(line)
if !strings.HasPrefix(line, commentPrefix) && !strings.HasPrefix(line, "// cqrs-lint:ignore") {
    continue
}
// Normalize: strip both variants
rest := strings.TrimPrefix(line, commentPrefix)
rest = strings.TrimPrefix(rest, "// cqrs-lint:ignore") // alternative prefix
```

The same bug exists in the block suppression parser (`checkBlockSuppressionInFile`, line 190) which has the same `TrimPrefix(text, "// ")` pattern.

---

## Part 2: False Positives

### FP 1: C036 — Store backend mismatch (no shared `*sql.DB` tracing)

**Severity: MEDIUM** — 2 false positives per project that uses a shared `*sql.DB` for event store + read model.

#### What happens

browser-history opens one SQLite database and passes the same `*sql.DB` to both the event store and the read model:

```go
// api/storage.go — openEventStore
db, err := sql.Open("sqlite", dsn)
// ...
store, err := storage.NewSQLiteEventStore(db)  // event store: backend detected as "custom" by linter
return db, store, nil

// api/server.go — NewServer
db, eventStore, err := openEventStore(cfg.DBPath, logger)
// ...
readStore := storage.NewSQLiteStore(db)  // read model: backend detected as "sqlite" by linter
```

cqrs-lint fires:

```
WARNING  C036  server.go:137  store uses sqlite backend but event store uses custom —
         backends cannot share a transaction, crash-recovery guarantees break
WARNING  C036  storage.go:46  store uses sqlite backend but event store uses custom —
         backends cannot share a transaction, crash-recovery guarantees break
```

The linter classifies the event store as `"custom"` (because `NewSQLiteEventStore` matches `storage` + `SQLite` → `"sqlite"`, but the feature profile detects the store type as `StoreCustom` since browser-history doesn't use the `stack` bundle presets). The read model constructor `NewSQLiteStore` is detected as `"sqlite"`. The mismatch fires.

#### Root cause

File: `cmd/cqrs-lint/pkg/rules/correctness/c036.go`

The detector does **zero data-flow analysis**. It cannot trace that two constructor calls receive the same `*sql.DB` handle. It only compares the string-derived backend label of each store-constructor call against the feature profile's event-store backend label.

The `detectBackend` function (`c036.go:109-122`) is purely string-pattern matching on the call site (`pkg == "storage"` + `fnName` contains `"SQLite"` → `"sqlite"`). There's no mechanism to verify that the `db` argument passed to both constructors is the same variable.

#### Fix suggestions

**Option A: Detect shared `*sql.DB` variable (medium effort)**

Track that both constructors receive the same `*ast.Ident` (variable name) as their first argument. If both `NewSQLiteEventStore(db)` and `NewSQLiteStore(db)` receive the same `db` variable, they share a backend by definition — the mismatch finding is a false positive.

**Option B: Suppress when both constructors are in the same `storage` package (low effort)**

If both calls are to functions in the `storage` package and both names contain `SQLite`, they're clearly the same backend. The `"custom"` classification from the feature profile is the problem — it doesn't match the actual store construction.

**Option C: Include the event store type in the feature profile detection (medium effort)**

The feature profile detects `StoreCustom` for browser-history, but the event store IS SQLite-backed (`NewSQLiteEventStore`). The profile should detect the actual store type from the constructor call, not just from `stack` bundle usage.

---

### FP 2: S006 — Financial data false positive on `Total*` field names

**Severity: LOW** — 1 false positive, INFO severity. But the pattern is broadly applicable.

#### What happens

browser-history has a `SummaryInput` struct that holds browsing analytics for AI summarization:

```go
type SummaryInput struct {
    PeriodStart       time.Time
    PeriodEnd         time.Time
    TotalVisits       int              // ← matches "total" (weak financial keyword)
    TotalDwellTime    time.Duration    // ← matches "total" (weak financial keyword)
    TopDomains        []visit.LeaderboardEntry
    CategoryBreakdown []CategoryStat
    // ...
    Weekly            *WeeklyBreakdown `json:"weekly,omitempty"`
}
```

cqrs-lint fires:

```
INFO  S006  ai_provider.go:26:6  Financial data in struct "SummaryInput" without encryption —
      sensitive monetary data is stored in plaintext
```

#### Root cause

File: `cmd/cqrs-lint/pkg/rules/security/s006.go:180-183`

The `weakFinancial` keyword list includes `"total"`:

```go
var weakFinancial = []string{
    "amount", "price", "total", "balance",
    "cost", "fee", "tax", "subtotal",
    "discount", "currency", "monetary",
}
```

The scoring function (`scoreFinancialStruct`, lines 208-233) evaluates each field name via `strings.Contains(lowercased, keyword)`. Two fields (`TotalVisits`, `TotalDwellTime`) both match `"total"`, meeting the ≥2 weak-indicator threshold. The serialization gate passes because `Weekly` has a `json:` tag.

The keyword `"total"` is too broad. In non-financial codebases, `Total` is one of the most common field-name prefixes: `TotalVisits`, `TotalCount`, `TotalPages`, `TotalSize`, `TotalDuration`, `TotalRequests`, `TotalErrors`. Every one of these would match.

#### Fix suggestions

**Option A: Remove `"total"` from the weak keyword list (simplest)**

`"total"` alone is not a financial indicator. `"subtotal"` is already in the list and is genuinely financial. The jump from "total" to "financial data" is too large.

**Option B: Require `"total"` to co-occur with other financial keywords**

Instead of counting `"total"` matches as independent weak hits, only count it when another non-weak financial keyword exists in the same struct. This way `TotalVisits + TotalDwellTime` = 0 financial score (no other keywords), but `TotalAmount + Balance` = financial score (strong co-occurrence).

**Option C: Add an exclusion for `Total` + non-financial suffixes**

Exclude matches like `TotalVisits`, `TotalCount`, `TotalSize`, `TotalDuration`, `TotalRequests`, `TotalErrors` — where the suffix is clearly not financial. This is more fragile than Option A or B.

---

### FP 3: B022 — Custom enricher false positive (confirmed, same as bank-sync)

**Severity: MEDIUM** — 2 false positives. Already reported by bank-sync in [2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md](../2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md) Bug 1.

Confirming: browser-history has the identical issue. Code uses `event.CommandCausalityEnricher` (the canonical enricher):

```go
// extraction/commands/handlers.go:28
decider.WithEnricher[aggregate.BrowserHistoryState](event.CommandCausalityEnricher),
```

B022 fires, suggesting `decider.CommandCausalityEnricher` which doesn't exist (the function is in `event`, not `decider`). Same root cause, same fix needed. No additional detail — see bank-sync's Bug 1.

---

## Part 3: Legitimate Suppressions (Accurate but Lacking Linter Context)

These 14 suppressions are accurate — the code is correct and the findings don't apply. But they highlight areas where the linter lacks context to make the right decision automatically.

### C033 — Bare return err from CQRS dispatch (2 suppressions)

```go
//cqrs-lint:ignore(C033) dispatch errors are already classified by command middleware
if err := svc.cmdDisp.Dispatch(ctx, cmd); err != nil {
    return err
}
```

The linter wants every `return err` from a CQRS call wrapped with context. But browser-history's command dispatcher is wired with middleware that classifies errors into an error taxonomy (`errorfamily`: Rejection, Transient, Corruption, Infrastructure). Wrapping with `errorfamily.Wrap(err, ...)` would override the middleware-assigned family, changing the HTTP status code mapping (e.g., Rejection→400 becomes Infrastructure→503).

**What the linter lacks:** Awareness of the command middleware chain. If the linter could detect that `cmdDisp` has `middleware.CommandIdempotency` or similar classification middleware wired via `.Use()`, it could recognize that errors are already classified.

### A032 — String path params instead of branded IDs (3 suppressions)

```go
//cqrs-lint:ignore(A032) Huma path param; branded IDs lack TextUnmarshaler
type GetVisitInput struct {
    ID string `path:"id"`
}
```

The linter wants branded IDs (`id.Of[T]`). But Huma path params are inherently `string` — Huma populates them from URL path segments. branded IDs based on `cqrsid.Of[T]` don't implement `TextUnmarshaler`, so Huma can't deserialize them directly. Conversion happens in the handler via `requireVisitID(input.ID)`.

**What the linter lacks:** Awareness of the Huma framework's deserialization constraints. This is similar to the `external-api-struct-prefixes` config — a way to tell the linter "this framework requires string params."

### A017/B025 — No snapshot/StateCache on 1-event streams (4 suppressions)

```go
//cqrs-lint:ignore(A017,B025) aggregates are 1-event-per-stream
repo, err := decider.NewRepository(events, eventBus, aggregate.BrowserHistoryDecider,
```

The linter suggests snapshots and StateCache. But browser-history's aggregates are 1-event-per-stream: a visit is recorded (1 event), a domain classified (1 event), a visit deleted (1 event). Snapshot and StateCache provide zero benefit for single-event streams — there's no state to cache or snapshot.

**What the linter lacks:** Stream-length awareness. If the linter could analyze the decider's `Decide` functions and determine that each produces at most 1 event per stream, it could recognize that snapshot/StateCache are unnecessary.

### E016 — No `bundle.HealthCheck()` (1 suppression)

```go
//cqrs-lint:ignore(E016) project has its own GET /health endpoint; does not use stack.Bundle wiring
```

The linter checks for `bundle.HealthCheck()`. browser-history has its own `GET /health` endpoint registered via Huma. The project doesn't use `stack.Bundle` — it wires CQRS components individually.

**What the linter lacks:** Awareness of alternative health endpoint patterns. If the linter could detect `GET /health` or `GET /healthz` route registration, it could recognize the health check is present.

### E017 — No `GracefulClose` (1 suppression)

```go
//cqrs-lint:ignore(E017) graceful shutdown is implemented via httpServer.Shutdown + s.Close()
```

The linter checks for `stack.Bundle.GracefulClose()` or `GracefulStop()`. browser-history implements graceful shutdown manually via `signal.NotifyContext` + `httpServer.Shutdown(ctx)` + `s.Close()`. Same pattern, different API.

### C013 — `time.Time` in event payload (1 suppression)

```go
//cqrs-lint:ignore(C013) LastVisit is always UTC-normalized; switching to event.Instant would break CBOR wire format
```

The linter correctly identifies that `time.Time` loses timezone info through CBOR encoding. browser-history's `DomainPayload.LastVisit` is always UTC-normalized from browser extraction. Switching to `event.Instant` would break the CBOR wire format of existing events (requires a migration).

### C036 — Shared `*sql.DB` (2 suppressions, see FP 1 above)

Accurate suppression while waiting for the false positive fix.

---

## Part 4: What's Working Great

| Improvement since round 1                           | Impact                                                                                                   |
| --------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| **Stale suppression detection**                     | Caught a C036 suppression that ended up on the wrong line after refactoring. Very useful.                |
| **Block suppression** (`ignore-start`/`ignore-end`) | Clean alternative for multi-line findings. Used in `storage.go` before this session eliminated the need. |
| **Feature-profile system**                          | Correctly gates F004/F013 on server mode. Reduces noise on non-server code paths.                        |
| **`--show-suppressed` flag**                        | Makes it possible to audit suppression quality. Essential for trust.                                     |
| **`--strict` mode**                                 | Catches everything. No surprises from lenient defaults.                                                  |
| **Comma-separated rule IDs**                        | `//cqrs-lint:ignore(A017,B025)` is clean and readable.                                                   |
| **Stale suppression warnings**                      | `warning: stale suppression at storage.go:38` immediately caught my edit mistake. Excellent DX.          |

---

## Part 5: Prioritized Fix List

| Priority | Issue                                                         | Type                               | Effort                                        | Impact                                              |
| -------- | ------------------------------------------------------------- | ---------------------------------- | --------------------------------------------- | --------------------------------------------------- |
| **P0**   | Suppression parser silently fails on `// comment`             | Bug (parser logic)                 | Trivial (1-line fix, see Part 1)              | High — every consumer writing Go-idiomatic comments |
| **P0**   | B022 suggests non-existent `decider.CommandCausalityEnricher` | Bug (wrong text + exemption logic) | Trivial (already reported by bank-sync)       | High — stops misleading users                       |
| **P1**   | C036 can't trace shared `*sql.DB`                             | Enhancement                        | Medium (AST arg matching)                     | Medium — 2 false positives per multi-store project  |
| **P1**   | S006 `"total"` keyword too broad                              | Enhancement                        | Low (remove keyword or require co-occurrence) | Medium — matches any `Total*` field name            |
| **P2**   | C033 lacks middleware-chain awareness                         | Enhancement                        | Hard (requires data-flow analysis)            | Low — legitimate suppression, clear comment         |
| **P2**   | A032 lacks framework deserialization awareness                | Enhancement                        | Medium (framework-aware config)               | Low — Huma-specific, config-gatable                 |

---

## Appendix: Full Finding Summary (browser-history, post-session)

```
FIXED THIS SESSION (4 suppressions eliminated):
  C023 ×4   storage.go     Ignored close errors    Passed logger to openEventStore, used closeDBOnError

SUPPRESSED — FALSE POSITIVES (5):
  B022 ×2   handlers.go:28, tag_handlers.go:65    event.CommandCausalityEnricher misidentified as "custom"
  C036 ×2   server.go:137, storage.go:46           Shared *sql.DB not traced between event store and read model
  S006 ×1   ai_provider.go:26                      TotalVisits/TotalDwellTime matched as financial "total" keyword

SUPPRESSED — LEGITIMATE CONSTRAINTS (9):
  C033 ×2   visit_service.go:68,118               Middleware-classified errors (wrapping overrides family)
  A032 ×3   handlers.go:470,589, visit_service.go:140  Huma path params are inherently string
  A017 ×2   handlers.go:25, tag_handlers.go:62     1-event-per-stream aggregates (snapshot pointless)
  B025 ×2   handlers.go:25, tag_handlers.go:62     Same (StateCache pointless for 1-event streams)
  C013 ×1   browser_history.go:30                  time.Time in CBOR event payload (wire format migration)
  E016 ×1   otel.go:62                             Has own /health endpoint (no stack.Bundle)
  E017 ×1   server.go:631                          Graceful shutdown implemented manually

SUPPRESSED — CONSCIOUS NON-ADOPTIONS (4):
  E009      agent_middleware.go:2                   Uses Huma HTTP handlers (not cqrs transport module)
  F004      agent_middleware.go:2                   No Prometheus module (could adopt — real observability gap)
  F015      agent_middleware.go:2                   No metaengine (SQLite, simple queries)
  B004 ×2   handlers.go:110, tag_handlers.go:16    No cqrs-gen (hand-written constructors)
  F009      ingest_dedup_set.go:50                 No scheduling module (best-effort TTL cache)
```

---

## Resolution (2026-08-03)

All reported bugs fixed per round-2 review: suppression parser space-after-`//` bug FIXED, C036 shared `*sql.DB` detection FIXED (`collectEventStoreBackends`), S006 `"total"` keyword REMOVED, B022 suggestion text FIXED. Remaining items are legitimate suppressions and conscious non-adoptions (accurate, not bugs).
