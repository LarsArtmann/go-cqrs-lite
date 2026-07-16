# cqrs-lint — Consumer Feedback (DiscordSync)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** go-cqrs-lite v4.0.0 (flake pin at `c34dd604`)
**lint version:** `go run -tags "goexperiment.jsonv2" ./cmd/cqrs-lint/` (master, 2026-07-16)
**Date:** 2026-07-16

---

## Executive Summary

Ran cqrs-lint against the full DiscordSync codebase (126 Go files). The tool
reported a **47/100 health score** across 34 findings (1 critical, 4 error,
15 warning, 14 info). After source-level verification of every finding, the
results break into three categories:

| Category                                              | Count | Action taken                                |
| ----------------------------------------------------- | ----- | ------------------------------------------- |
| **Valid findings we fixed**                           | 3     | All 3 panics replaced with error returns    |
| **False positives (heuristic mismatch)**              | 14    | Documented below — no code changes          |
| **Intentional design decisions (documented in ADRs)** | 17    | No code changes — architectural constraints |

The 3 panics flagged by **C009** were legitimate. The **C001** critical was a
dangerous false positive that would have introduced a double-commit bug if
"fixed."

---

## Part 1: Valid Findings — Fixed

### C009: `panic()` in production code (3 sites)

**Verdict: Correct. We fixed all three.**

The linter flagged three `panic()` calls. After review, we agreed that `panic()`
is never acceptable in production code, even for "impossible" failures. A clean
error return produces structured logs, lets deferred cleanup run, and doesn't
dump a stack trace that looks like a bug.

#### Fix 1: `DiscordID()` — `internal/events/types.go:71`

**Before:**

```go
func DiscordID(snowflake string) id.AggregateID {
    aggID, err := id.ParseAggregateID(snowflake)
    if err != nil {
        panic("discord: invalid snowflake as aggregate ID: " + err.Error())
    }
    return aggID
}
```

**After:**

```go
func DiscordID(snowflake string) (id.AggregateID, error) {
    aggID, err := id.ParseAggregateID(snowflake)
    if err != nil {
        return id.AggregateID{}, errorfamily.WrapCorruption(err, "event.error", "parse snowflake as aggregate ID")
    }
    return aggID, nil
}
```

**Impact:** Zero production callers (only used in its own test). Clean fix.

#### Fix 2: `Journal()` — `internal/storage/storage.go:79`

**Before:**

```go
func (ec *EventCapture) Journal() event.SeekableJournal {
    journal, err := cqrsschema.NewVersionedSeekableJournal(ec.store, eventschema.Registry()...)
    if err != nil {
        panic(fmt.Sprintf("failed to create versioned journal: %v", err))
    }
    return journal
}
```

**After:**

```go
func (ec *EventCapture) Journal() (event.SeekableJournal, error) {
    journal, err := cqrsschema.NewVersionedSeekableJournal(ec.store, eventschema.Registry()...)
    if err != nil {
        return nil, errkit.Infrastructure(err, eventInitCode, "create versioned journal")
    }
    return journal, nil
}
```

**Impact:** 11 call sites updated (2 production, 9 tests). Production callers
wrap the error with `errkit.Infrastructure()` for structured error context.
The `fmt` import was removed (it was only used by the panic).

#### Fix 3: `sseCBORDecMode` — `internal/api/sse.go:50`

**Before:**

```go
var sseCBORDecMode = sync.OnceValue(func() cbor.DecMode {
    opts := cbor.DecOptions{...}
    mode, err := opts.DecMode()
    if err != nil {
        panic("sse: CBOR DecMode creation failed: " + err.Error())
    }
    return mode
})
```

**After:**

```go
var (
    sseCBORDecModeOnce sync.Once
    sseCBORDecModeVal  cbor.DecMode
    sseCBORDecModeErr  error
)

func getSSECBORDecMode() (cbor.DecMode, error) {
    sseCBORDecModeOnce.Do(func() {
        opts := cbor.DecOptions{...}
        sseCBORDecModeVal, sseCBORDecModeErr = opts.DecMode()
    })
    return sseCBORDecModeVal, sseCBORDecModeErr
}
```

The caller (`jsonPayloadForSSE`) now handles the error gracefully by logging
and falling back to the raw payload.

---

## Part 2: False Positives — No Action Taken

### C001: `withTx` never commits — CRITICAL (DANGEROUS)

**Verdict: FALSE POSITIVE. Applying the suggested fix would introduce a double-commit bug.**

The linter sees `BeginTx` without `Commit` in the same function body. But
`withTx` is a **closure-based transaction helper** — the `body` callback is
contractually responsible for calling `tx.Commit()`, as documented at
`messages.go:13-16`:

> _"body MUST call tx.Commit() on the success path (commit makes rollback a no-op)"_

All 9 call sites commit inside the closure: `messages.go:145,249,284`,
`members.go:31,51`, `roles.go:40`, `threads.go:25,45`, `reactions.go:34`.

Applying the linter's suggested fix (`return tx.Commit()`) would double-commit
every transaction — `sql.Tx.Commit()` after the body already committed returns
`sql.ErrTxDone`.

**Suggested improvement:** C001 should trace whether the tx variable is passed
to a callback/closure before flagging. If the tx escapes to a function-typed
parameter, the commit cannot be statically verified within the function body.

---

### E006: `GCSMigrationCandidate` emitted but no projection handles it

**Verdict: FALSE POSITIVE. Not an event type.**

`GCSMigrationCandidate` is a SQL row struct (`db/attachment_downloads.go:55`)
used by a read query (`GetGCSMigrationCandidates`). It is never emitted as an
event — the linter matched the "Candidate" suffix as an event naming pattern.

**Suggested improvement:** E006 should cross-reference the event type against
`event.Type` constants or `Emit()` call sites, not just match on naming
conventions.

---

### A016: Command dispatcher lacks idempotency middleware

**Verdict: FALSE POSITIVE. No command dispatcher exists.**

DiscordSync uses event sourcing (`EventCapture.Emit`), not command dispatching.
Zero matches for `CommandDispatcher` / `Dispatch` in the codebase. The linter
fired because it found a `NewDispatcher` or `Use` call somewhere, but this is
the event bus middleware chain, not a command dispatcher.

**Suggested improvement:** A016 should verify the dispatcher is actually a
`command.Dispatcher` (not an `event.Bus`) before flagging.

---

### C008: `float64` for money (2 sites)

**Verdict: FALSE POSITIVE. Neither field is monetary.**

| Field                      | Actual purpose                        |
| -------------------------- | ------------------------------------- |
| `rateTracker.value`        | Prometheus counter delta (events/sec) |
| `SparklineSample.LagValue` | Projection lag in seconds             |

The linter matches `float64` fields whose names contain money keywords
(`value`, `amount`, `price`, etc.). `value` is too generic — it matches any
field named "value" regardless of domain.

**Suggested improvement:** C008 should require multiple money signals (e.g.,
field name + struct name + package context) before flagging. A field named
`value` in an `observability` package is not money.

---

### A005: Manual projection via `bus.SubscribeAll` (2 sites)

**Verdict: FALSE POSITIVE. Both are SSE/stats fan-out, not projections.**

| Site            | Purpose                                                                         |
| --------------- | ------------------------------------------------------------------------------- |
| `server.go:279` | Stats notifier — fires `notifier.Notify()` on every event for dashboard polling |
| `sse.go:130`    | SSE broadcaster — fans out events to HTTP clients                               |

Neither needs checkpoint persistence, DLQ, or crash recovery — they are
fire-and-forget notifications. The real projections already use
`projectionhost.New`.

**Suggested improvement:** A005 should check whether the `SubscribeAll`
callback writes to a database or mutable state (projection pattern) vs.
broadcasts/notifies (pub-sub pattern). The current heuristic is too broad.

---

### D005: Documentation version mismatch (2 sites)

**Verdict: FALSE POSITIVE. Version regex extracts empty string.**

The linter's `extractCQRSVersion` function collects whitespace-delimited tokens
prefixed with `v` from lines mentioning `go-cqrs-lite`. Our docs deliberately
don't hardcode a version — per our docs-health audit:
_"never hardcode the dependency version in docs; link to go.mod instead."_

The linter extracted an empty version token from prose like "via go-cqrs-lite"
and compared it against `v4.0.0` from `go.mod`.

**Suggested improvement:** D005 should skip lines where no version token is
found, rather than comparing an empty string against the go.mod version.

---

## Part 3: Intentional Design Decisions — No Action Taken

### A009: No `stack/` preset

**Verdict: INTENTIONAL. Documented in `docs/go-cqrs-lite-usage.md:44`.**

DiscordSync uses one shared `*sql.DB` for both CQRS events AND relational
reads. The `stack/*` presets create separate `*sql.DB` connections, which is
incompatible with this architecture without dual connections or a major
refactor.

---

### D002 / D004: Mixed JSON key casing (19 files, 492 tags)

**Verdict: INTENTIONAL. Snake_case comes from Discord's API.**

Discord's Gateway API uses snake_case for all event payloads (`author_id`,
`channel_id`, `guild_id`, etc.). DiscordSync's event payload structs mirror
Discord's wire format with snake_case JSON tags for fidelity and debuggability.
The camelCase tags are on internal HTTP API response structs, which are a
separate concern.

Standardizing on one convention would require either:

1. Adding a translation layer between Discord payloads and event structs (breaks
   fidelity, adds complexity)
2. Renaming all Discord API fields (breaks compatibility with debugging tools
   that expect Discord's naming)

Neither has a correctness benefit.

---

### B007: Consecutive handler registrations (2 sites)

**Verdict: STYLE PREFERENCE. Go 1.22+ pattern routing is idiomatic.**

The 8/7 consecutive `mux.HandleFunc` calls in `registerRoutes` use Go 1.22+
pattern routing with per-route middleware (`ETag`, `postBodyLimit`). A
table-driven approach would lose readability because each handler has different
middleware wrapping.

---

## Health Score Breakdown

The 47/100 score is computed as: `100 - sum(deductions)`.

| Rule      | Severity | Count  | Per-finding | Subtotal |
| --------- | -------- | ------ | ----------- | -------- |
| C001      | Critical | 1      | -10         | -10      |
| C009      | Warning  | 3      | -2          | -6       |
| A005      | Warning  | 2      | -2          | -4       |
| A016      | Warning  | 1      | -2          | -2       |
| C008      | Warning  | 2      | -2          | -4       |
| D005      | Warning  | 2      | -2          | -4       |
| A009      | Info     | 1      | -1          | -1       |
| B007      | Info     | 2      | -1          | -2       |
| D002      | Info     | 13     | -1          | -13      |
| D004      | Info     | 1      | -1          | -1       |
| E006      | Info     | 1      | -1          | -1       |
| **Total** |          | **29** |             | **-48**  |

After fixing the 3 C009 panics (-6 points), the effective score would be 53/100.
The remaining deductions are all false positives or intentional decisions.

---

## Suppression Mechanism Usage

We discovered cqrs-lint supports inline suppression via:

```go
//cqrs-lint:ignore(RULE_ID) optional reason
```

We are **not** adding suppressions for the false positives at this time because:

1. The suppressions would need to be on specific lines, not file-level
2. The false positives are in documentation files (D005), package-level (A009,
   A016), and struct-level (C008) — there's no natural single-line suppression
   point
3. We prefer the linter to improve its heuristics rather than suppress
   legitimate-but-misfiring rules

---

## Suggestions for cqrs-lint Improvement

Ranked by impact (would remove the most false positives):

1. **C001: Trace closures** — If the tx variable is passed to a function-typed
   parameter, the commit cannot be statically verified. Skip or lower severity.

2. **C008: Require multiple money signals** — A field named `value` in an
   `observability` package is not money. Require field name + struct/package
   context before flagging.

3. **E006: Cross-reference event registry** — Match against actual `event.Type`
   constants or `Emit()` call sites, not naming conventions.

4. **A005: Check callback behavior** — Distinguish projection (writes to DB)
   from pub-sub (broadcasts/notifies) by inspecting the callback body.

5. **A016: Verify dispatcher type** — Check that `NewDispatcher` is actually a
   `command.Dispatcher`, not an `event.Bus`.

6. **D005: Skip empty version tokens** — If no `v`-prefixed token is found on a
   line, don't flag it as a version mismatch.

7. **D002/D004: Respect external API contracts** — When snake_case tags match a
   known external API (Discord, Stripe, GitHub), they are not a project style
   choice and should be excluded from the consistency check.

---

## Summary Table

| Rule | Severity | Count | Verdict                        | Action                                                 |
| ---- | -------- | ----- | ------------------------------ | ------------------------------------------------------ |
| C001 | Critical | 1     | **FALSE POSITIVE** (dangerous) | None — fixing would double-commit                      |
| C009 | Warning  | 3     | **VALID**                      | Fixed — all panics replaced with error returns         |
| A005 | Warning  | 2     | False positive                 | None — SSE/stats fan-out, not projections              |
| A016 | Warning  | 1     | False positive                 | None — no command dispatcher exists                    |
| C008 | Warning  | 2     | False positive                 | None — not monetary values                             |
| D005 | Warning  | 2     | False positive                 | None — version regex extracts empty string             |
| A009 | Info     | 1     | Intentional                    | None — shared-DB architecture incompatible with stack/ |
| B007 | Info     | 2     | Style preference               | None — Go 1.22+ pattern routing is idiomatic           |
| D002 | Info     | 13    | Intentional                    | None — Discord API uses snake_case                     |
| D004 | Info     | 1     | Intentional                    | None — same as D002, project-wide                      |
| E006 | Info     | 1     | False positive                 | None — SQL row struct, not an event type               |
