Proposed title: cqrs-lint E004 fires on complete catalogs when ids pass through a local value-converter func

## Problem

E004 (event not in catalog) fires on every emission site when the catalog
registers message ids through a local single-return converter function. The
catalog is complete (pinned by a test), the emissions are fine, but the
analyzer cannot resolve the id through the indirection, so the consumer
suppresses the whole rule to use the tool at all.

The shape (from github.com/LarsArtmann/timesheets `internal/domain/catalog.go`):

```go
// skip-validate
func messageID(t string) catalog.MessageID {
	return catalog.MessageID(t)
}

...
catalog.Event[TimesheetCreatedPayload](messageID(string(EventTimesheetCreated)), catalog.Sends),
```

Timesheets ended up with `//cqrs-lint:ignore-start(E004)` spanning its eight
`Decide*` functions (`internal/domain/decide.go:38-412`) — a complete,
test-pinned catalog, fully suppressed.

## Why the existing recovery paths miss it

Source: `df033ea5e`, `cmd/cqrs-lint/`.

1. `unwrapStringConv` (`pkg/analyzer/scanner_consts.go:76`) unwraps only the
   builtin `string(...)` conversion — a package-qualified conversion
   (`catalog.MessageID(...)`) or a local func call passes through unchanged.
2. The wrapper-helper replay (`scanCatalogHelperFunc`,
   `pkg/analyzer/scanner_catalog_helpers.go:34`) covers the generic-helper
   shape: a function forwarding one of its params into `catalog.Event`.
   `messageID` is not that — its body converts its param's _type_, and its
   _return value_ is the id argument at the call site. Neither path
   resolves `messageID(string(...))` to the constant inside.

## Suggestion

Teach the id resolution to inline one level through local single-return
functions whose body is a pure conversion (`return SomeType(x)`), the same
way the helper replay already follows param-forwarding wrappers. Fallback:
document this indirection under E004 in `RULES.md` so consumers know why a
test-pinned catalog still needs the suppression.

Crush-assisted draft (checked with the voice checker).
