# cqrs-lint project-level findings (F002/F005/F008) anchor at the load-order-first file's package clause — inline suppression lands in an arbitrary file

TL;DR: F002/F005/F008 anchor project-level findings at the package clause of the first non-test file in load order (`firstFilePos`). The only reachable inline `//cqrs-lint:ignore` slot is that file's package doc comment — a file chosen by load order, unrelated to the construct that triggered the finding — and renaming/reordering files silently moves the anchor away and strands the directive as stale. Ask: **anchor at the triggering call site** (F005 → first `event.WithSchemaVersion`, F008 → the `codec.JSONCodec` selector, F002 → an event type declaration).

## Reproduced

cqrs-lint v4.22.2, consumer quotaboard at `709faff` (one event type, `event.WithSchemaVersion(1)` on its constructor, v1-only journal by design, no upcaster):

```
F005  info  internal/history/analyze.go:1:1  "Project uses event.WithSchemaVersion but has no schema.NewUpcaster …"
```

`analyze.go` is merely the first non-test file in load order; the `WithSchemaVersion` call lives in a different file. Verified live: placing `//cqrs-lint:ignore(F005)` as line 1 of `analyze.go` (becoming the package doc comment) DOES suppress it — the anchor follows the package keyword down. So the mechanism is reachable but wrong-shaped:

- the suppression sits in a file a developer would never connect to the finding;
- it masquerades as package documentation (rendered by godoc);
- it is fragile against file renames/reordering, which moves the anchor and orphans the directive into a stale-suppression failure.

Because the honest inline slot is unavailable, quotaboard fell back to project-wide `rules.disable` in `.cqrs-lint.json` for what is really a scoped, one-off refusal — the config field's own doc says inline is preferred for one-offs.

## Root cause (source)

- Anchor: `firstFilePos` — `pkg/rules/adoption/helpers.go:57` → `lintutil.FirstFilePos` (`pkg/rules/lintutil/lintutil.go:266`), returning `ctx.Fset.Position(gf.AST.Package)` of the first non-test GoFile.
- Matcher: `checkSuppressionInFile` — `pkg/suppression/parser.go:88-140` — honors the finding's own line and the first non-blank line above it.
- `firstCallByNameIn` (`pkg/rules/adoption/scan_in.go:220`) already exists and F007 uses it for `NewDispatcher` — the fix is local to the three detectors.

## Design

Anchor F005 at the first `WithSchemaVersion` call, F008 at the `JSONCodec` selector, F002 at the first event type declaration. Suppressions then land next to the triggering construct, stay scoped, and stop forcing config-disables for one-off refusals. Migration note: any consumer who suppressed at a package doc comment would need to move the directive — acceptable for a minor release, and detectable via `--fail-on-stale-suppressions` after the anchor moves.

💘 Generated with Crush
