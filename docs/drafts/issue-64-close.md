Both fixed on master in 46187d98a. Not in a tag yet; lands with the next cqrs-lint cut.

C009: the `len(name) > 4` / `len(name) > 3` guards are gone — bare `must` and `New` are exempt by prefix match alone (`cmd/cqrs-lint/pkg/rules/correctness/c009.go:97`). No more renaming the composition-root helper.

E005: the scanner tracks `system.RegisterCommand[MyCmd, MyState](sys, ...)` as a registration — the command type comes from the type parameter into `CommandTypesRegistered` (`cmd/cqrs-lint/pkg/analyzer/scanner_calls.go:125`), so every ADR-0123 system-rooted app stops getting "no registered handler". The suggestion text now leads with `system.RegisterCommand` too (`cmd/cqrs-lint/pkg/rules/architecture/rules.go:94`).

Tests: `TestC009_NoFindingInBareMustFunc` (bare `must` helper, no finding) and `TestE005_NoFindingWhenRegisteredViaSystem` (system-rooted app, zero E005) — green at `3dbf04192`.
