# Review: DiscordSync cqrs-lint Feedback (Round 2)

**Source feedback:** [`new/2026-08-04_DiscordSync_cqrs-lint-feedback-round2.md`](../new/2026-08-04_DiscordSync_cqrs-lint-feedback-round2.md)
**Date reviewed:** 2026-08-13
**Outcome:** All rule-level issues already addressed. Config/UX suggestions triaged.

---

## P012 — `missing-sqlite-wal` (FALSE POSITIVE) — FIXED

**Status:** ✅ Fixed

The rule now detects DSN-level WAL pragmas via `dsn_resolver.go` (`dsnHasWAL`), covering both modernc.org/sqlite (`?_pragma=journal_mode(WAL)`) and mattn/go-sqlite3 (`?_journal_mode=WAL`) syntaxes. The suggestion text no longer references `storage.SQLiteEnableWAL` — it now recommends DSN pragmas or `db.Exec("PRAGMA journal_mode = WAL")`.

---

## P013 — `missing-sqlite-busy-timeout` (FALSE POSITIVE) — FIXED

**Status:** ✅ Fixed

The rule detects DSN-level `busy_timeout` via `dsnHasBusyTimeout` (case-insensitive substring match), covering all driver DSN syntaxes.

---

## C034 — `goroutine-without-ctx` (FALSE POSITIVE) — FIXED

**Status:** ✅ Fixed

The rule now traces derived contexts via `findDerivedContextVars`. A variable assigned from `context.WithCancel(ctx)` / `WithTimeout(ctx)` / `WithDeadline(ctx)` / `WithValue(ctx)` is recognized as valid context propagation. `goroutineHasCtxArg` treats derived variables as safe. `enclosingFuncHasShutdown` recognizes `<-derivedVar.Done()`.

---

## C036 — `store-backend-mismatch` stale suppression detection — EXCELLENT

No action needed. Stale suppression detection via `--strict` is working as designed.

---

## C008 — `float64-for-money` — WISHLIST (not implemented)

Feature-profile-aware downgrading (`monetary: false` → auto-INFO for float64 fields matching rate/latency patterns) is a future enhancement. Currently suppressible via inline comments or `rules.disable`.

---

## V006 — version mismatch — ALREADY SUPPORTED

V006 can be suppressed via `.cqrs-lint.json` `rules.disable`: `{"rules": {"disable": ["V006"]}}`. The `rules.disable` system is rule-agnostic.

---

## D005 — documentation version drift — SYMPTOM OF V006

If V006 is disabled, D005 noise from submodule version mismatches should also be disabled via the same config mechanism.

---

## Config & UX Feedback

| Suggestion                                                   | Status                                             |
| ------------------------------------------------------------ | -------------------------------------------------- |
| `rules.disable` config                                       | ✅ Already implemented                             |
| `doctor` command + `features` config                         | ✅ Already implemented                             |
| `--doctor --fix` auto-write features                         | ✅ Shipped 2026-08-16 — `doctor --fix` removes stale whole-line suppressions |
| gofumpt interaction with package-level suppressions          | Resolved — `rules.disable` is the recommended path |
| Stale-suppression detection as default (not `--strict`-only) | ✅ Shipped 2026-08-16 — warnings on stderr in every format; `--fail-on-stale-suppressions` stays opt-in |
| Show config-disabled rules in health breakdown               | ✅ Shipped 2026-08-16 — "Excluded from score by config" footer |

---

## Summary

| Rule | Type                | Status    | Action                              |
| ---- | ------------------- | --------- | ----------------------------------- |
| P012 | False positive      | Fixed     | DSN pragma detection implemented    |
| P013 | False positive      | Fixed     | DSN pragma detection implemented    |
| C034 | False positive      | Fixed     | Derived context tracing implemented |
| C036 | Stale suppression   | Excellent | No action needed                    |
| C008 | Feature enhancement | Shipped  | `features.monetary` on/off/unknown override (2026-08-16) |
| V006 | Config request      | Supported | `rules.disable` works               |
| D005 | Symptom of V006     | Supported | Disable via config                  |
