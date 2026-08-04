# cqrs-lint — Consumer Feedback (DiscordSync, Round 2)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**cqrs-lint version:** system binary (v0.x, 2026-08-04)
**go-cqrs-lite version:** storage/turso/v4 v4.2.1-0.20260803220640-7f4c9116bd08
**Date:** 2026-08-04
**Previous feedback:** [2026-07-16_DiscordSync_cqrs-lint-feedback.md](../2026-07-16_DiscordSync_cqrs-lint-feedback.md)
**Health score this session:** 89→90/100 (Excellent)

---

## Summary

Round 1 feedback was triaged and largely resolved. This round focuses on a
newer class of rules (P-series, C034/C036) and config/UX improvements. The
linter has matured significantly — `doctor` command, feature profiles, and
`rules.disable` config have eliminated entire classes of false-positive churn.

**What's great:** the linter now catches real bugs (stale suppressions!
version drift!) and the config system makes false-positive management
reproducible rather than comment-based.

**What needs work:** P012/P013 fire on a pattern they should recognize (DSN
pragmas), C034 has a context-derivation blind spot, and the gofumpt
interaction story needs one more improvement.

---

## Rule-Level Feedback

### P012 — `missing-sqlite-wal` (FALSE POSITIVE, blocking)

**Severity:** WARNING
**Current behavior:** Fires when it can't find a call to
`storage.SQLiteEnableWAL(ctx, db)`. Suggests calling that function.

**Problem:** `storage.SQLiteEnableWAL` **does not exist** in the
`go-cqrs-lite/storage/v4` module (verified via `go doc`). The function the
linter suggests calling is imaginary. The actual library API for WAL is:

1. DSN-level pragmas (our approach): `?_pragma=journal_mode(WAL)` — applied
   per-connection by the modernc.org/sqlite driver. This is the
   **recommended approach** per `EnsureSQLiteDSNBusyTimeout`'s own doc
   comment ("DSN-level pragmas are applied by the driver on every new
   connection").
2. `ConfigureSQLitePool(db)` — sets pool-level settings.
3. The `stack/sqlite` bundle — auto-configures WAL.

**Suggested fix:** Two options:

1. **Detect DSN-level WAL:** scan for `_pragma=journal_mode(WAL)` or
   `journal_mode=WAL` in the DSN string passed to `sql.Open`. This is the
   canonical way to set WAL in modernc.org/sqlite.
2. **Fix the suggestion:** if the function doesn't exist, don't suggest it.
   At minimum, point to `ConfigureSQLitePool` or the `stack/sqlite` bundle.

**Impact:** Every modernc.org/sqlite consumer that sets WAL via DSN (the
documented best practice) gets a false positive. Currently worked around via
`rules.disable`.

---

### P013 — `missing-sqlite-busy-timeout` (FALSE POSITIVE, blocking)

**Severity:** WARNING
**Current behavior:** Same pattern as P012. Fires on the absence of a call
to `storage.EnsureSQLiteDSNBusyTimeout` or `storage.SQLiteEnableWAL`.

**Problem:** `EnsureSQLiteDSNBusyTimeout` exists and works, but the rule
doesn't recognize that our DSN already contains
`?_pragma=busy_timeout(15000)` — which is exactly what
`EnsureSQLiteDSNBusyTimeout` would append. Calling the function would be
redundant (append the same pragma twice).

**Suggested fix:** Detect `_pragma=busy_timeout(N)` or `busy_timeout=N` in
the DSN string. If present, the rule is satisfied.

**Impact:** Same as P012 — every modernc.org/sqlite consumer using DSN
pragmas.

---

### C034 — `goroutine-without-ctx` (FALSE POSITIVE, context derivation blind spot)

**Severity:** WARNING
**Current behavior:** Fires when `go func() { ... }()` doesn't reference
the `ctx` parameter from the enclosing function signature.

**Problem:** The rule only checks if the literal `ctx` parameter name is
referenced inside the goroutine body. It doesn't recognize **derived
contexts** — when the goroutine uses a context derived FROM `ctx` via
`context.WithCancel(ctx)`, `context.WithTimeout(ctx, ...)`, etc.

**Real example from our codebase** (`flightrecorder_auto.go:78`):

```go
autoCtx, cancel := context.WithCancel(ctx)  // derived from ctx

go func() {
    ticker := time.NewTicker(cfg.PollInterval)
    defer ticker.Stop()
    for {
        select {
        case <-autoCtx.Done():  // ← uses derived ctx, not bare ctx
            return
        case <-ticker.C:
            // ...
        }
    }
}()

return cancel  // ← caller controls shutdown
```

This goroutine is **properly context-bound** — it exits on `autoCtx.Done()`
and the `cancel` func is returned to the caller. But the linter fires
because `ctx` (the original parameter name) doesn't appear in the goroutine
body.

**Same pattern appears at:** `bot.go:386` (already suppressed),
`idempotency_sweeper.go:52` (already suppressed).

**Suggested fix:** Trace context derivation. If a variable is assigned from
`context.With*(ctx, ...)` and that variable is used in the goroutine's
select/cleanup, consider the ctx as "used". Alternatively, at minimum,
recognize `context.WithCancel(ctx)` → variable → `<-variable.Done()` as
satisfying the rule.

**Impact:** Every project that properly cancels goroutines via derived
contexts gets 3+ false positives. This is a common Go pattern.

---

### C036 — `store-backend-mismatch` (stale suppression detection: EXCELLENT)

**Severity:** WARNING
**Experience:** We had a suppression comment at `command_factory.go:25` that
was no longer needed. The linter's `--strict` mode caught this:

```
warning: stale suppression at command_factory.go:25 — rule C036 does not fire here; safe to remove
```

**Feedback:** This is **fantastic**. Stale suppression comments are a silent
form of technical debt — they accumulate over time as code changes. The
`--strict` stale-detection is one of the best features added since round 1.
No improvement needed here.

---

### C008 — `float64-for-money` (well-suppressed, but consider feature-profile awareness)

**Severity:** WARNING
**Current behavior:** Fires on any `float64` field, suggesting decimal for
money.

**Experience:** We suppress 26 instances across our metrics/rate-tracking
code. All are legitimately rates (events/sec, latency seconds), not money.

**Suggested improvement:** If the feature profile included something like
`"monetary": false`, the rule could default to INFO for projects that don't
handle money at all. Alternatively, if the field name matches
`*Rate|*PerSec|*Latency|*Seconds|*Ratio|*Percentage`, auto-downgrade to INFO.

---

### V006 — go-cqrs-lite version mismatch (valid, but unsuppressable in go.mod)

**Severity:** WARNING
**Current behavior:** Detects that `catalog/v4` is at v4.2.0 while
`storage/turso/v4` is at a pseudo-version and `storage/v4` is at v4.5.0.

**Feedback:** This is a real issue — we should ideally pin all submodules to
the same release. The problem is that go-cqrs-lite submodules have
independent release cadences and some (like storage/turso) are published as
pseudo-versions between tags. We can't `//cqrs-lint:ignore` in go.mod.

**Suggestion:** Allow V006 suppression via `.cqrs-lint.json` `rules.disable`
or an `.cqrs-lint-ignore` marker file at the repo root (since you can't put
inline comments in go.mod).

---

### D005 — documentation version drift (valid, but fires on living docs)

**Severity:** WARNING
**Current behavior:** Compares version strings in AGENTS.md against go.mod.
Fires when they don't match.

**Feedback:** The intent is good — docs should match reality. But this
creates a chicken-and-egg problem: the version in AGENTS.md is the go-cqrs-lite
storage/turso pseudo-version, which changes every time we bump the dep. The
D005 finding is really a symptom of V006 (go.mod itself has version
mismatches).

**Suggestion:** If V006 is active for a project, suppress D005 automatically
(docs can't match a go.mod that itself has internal mismatches).

---

## Config & UX Feedback

### `.cqrs-lint.json` `rules.disable` — EXCELLENT ADDITION

The ability to disable rules project-wide via config (rather than inline
comments) is a major improvement for rules like P012/P013 that are
fundamental false positives for entire categories of projects (modernc.org/sqlite
DSN users). This is strictly better than inline comments because:

1. It's centralized (one place to audit all disabled rules)
2. It's gofumpt-proof (no comment-formatting interaction)
3. It works for package-level findings where inline comments break

### `doctor` command + `features` config — EXCELLENT

The `doctor` command detecting the feature profile and suggesting the exact
JSON to paste is great UX. We now pin our feature profile explicitly to
prevent auto-detection drift.

**Minor suggestion:** Add a `--doctor --fix` flag that automatically writes
the detected features into `.cqrs-lint.json`.

### gofumpt interaction — mostly resolved

**Previous state:** gofumpt normalized `//cqrs-lint:ignore` to
`// cqrs-lint:ignore` (adding a space), which broke the suppression parser.

**Current state:** The parser now accepts both `//cqrs-lint:` and
`// cqrs-lint:`. This is confirmed by the `--help` output: "Both //cqrs-lint:
and // cqrs-lint: (with space) are accepted."

**Remaining issue:** For `package db`-level findings (P012/P013), inline
suppression at the package declaration line still doesn't work reliably
because gofumpt reformats doc-level comments and the linter's finding
location (line 2, the `package` line) doesn't have a "line above" that
survives gofumpt normalization. The `rules.disable` config workaround is
correct, but it would be even better if the linter documented this pattern
in `--help`.

### `--strict` stale suppression detection — EXCELLENT

Already praised above, but worth repeating: catching stale suppression
comments is one of the highest-value features. No other linter I've used
does this.

**Suggestion:** Make stale-suppression detection **default** (not gated
behind `--strict`). Stale suppressions are always wrong — there's no reason
to keep them.

---

## Health Score Feedback

The score moved from 89→90 (Good→Excellent) after removing one stale
suppression and properly managing false positives. The score feels
meaningful and tracks well with actual code health.

**Minor suggestion:** The score breakdown table is helpful. Consider
showing the **config-disabled** rules in a separate section (currently they
just vanish from the breakdown, making it hard to audit what's disabled).

---

## Summary Table

| Rule | Type            | Status     | Action Taken                                    |
| ---- | --------------- | ---------- | ----------------------------------------------- |
| P012 | False positive  | Disabled   | `rules.disable` (SQLiteEnableWAL doesn't exist) |
| P013 | False positive  | Disabled   | `rules.disable` (DSN pragma already set)        |
| C034 | False positive  | Suppressed | Inline comment (derived context pattern)        |
| C036 | Stale suppress  | Removed    | Linter caught stale suppression (`--strict`)    |
| C008 | Accepted        | Suppressed | 26 inline comments (rates, not money)           |
| V006 | Valid           | Accepted   | Can't suppress in go.mod                        |
| D005 | Symptom of V006 | Accepted   | Docs can't match mismatched go.mod              |

---

## Wishlist (new rules/features)

1. **Detect DSN-level SQLite pragmas** (fixes P012/P013 for modernc.org/sqlite
   users — scan for `_pragma=journal_mode(WAL)` and `_pragma=busy_timeout(N)`
   in the `sql.Open` DSN string).
2. **Context-derivation tracing for C034** (recognize
   `context.WithCancel(ctx)` → variable → `<-variable.Done()` as satisfying
   the rule).
3. **`--doctor --fix`** flag to auto-write detected features into
   `.cqrs-lint.json`.
4. **Make stale-suppression detection default**, not `--strict`-only.
5. **Show config-disabled rules** in the `--health-score` breakdown.
6. **Allow V006 suppression** via config or marker file (can't use inline
   comments in go.mod).
7. **Feature-profile-aware C008**: if `monetary: false` in feature profile,
   auto-downgrade float64 findings to INFO.
