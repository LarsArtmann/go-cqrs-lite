# ADR-0055: cqrs-lint Loader Error Surfacing

Date: 2026-07-17
Status: Accepted

## Context

cqrs-lint's `BuildContext` function uses `go/packages.Load` to parse the consumer
project's Go AST. When package loading failed (e.g., unresolvable dependencies,
syntax errors, missing modules), the loader silently `continue`d past errors
and produced an empty `AnalysisContext`.

The main `run()` function then checked `len(actx.GoFiles) == 0` and printed
"No Go files importing go-cqrs-lite found. Nothing to lint." — a message that
reads as a clean bill of health on a project that doesn't even compile.

This was the root cause of a real-world incident: cqrs-htmx (a 50+ import
consumer) was reported as "nothing to lint" because the go-cqrs-lite v4.0.0
publish bug left zero pseudo-versions in go.mod files, making `go/packages`
fail silently.

A second defect compounded the problem: the `doctor` command read
`ctx.Packages` (which includes errored packages) for feature detection, while
`lint` read `ctx.GoFiles` (which excludes them). The two commands disagreed on
the project's feature profile — a split-brain where `doctor` confidently
reported a profile from broken packages while `lint` said nothing.

## Decision

1. **Collect errors, don't swallow them.** `BuildContext` now appends
   per-package and per-module load errors to `AnalysisContext.LoadErrors`
   (`[]PackageLoadError`) instead of silently `continue`ing. The loader still
   continues (preserving partial-analysis behavior), but errors are retained.

2. **Surface errors in `run()`.** When `GoFiles` is empty and `LoadErrors` is
   non-empty, `run()` prints a clear diagnostic with the per-package errors,
   tells the user to fix the build, and returns `errFindingsWithErrors` (non-zero
   exit). When `GoFiles` is non-empty and `LoadErrors` is non-empty (partial
   analysis), a warning is printed.

3. **`--strict-load` flag.** Makes any `LoadErrors` fatal regardless of whether
   packages were loaded. Useful in CI where partial analysis is unacceptable.

4. **Message split.** "No Go files found. Nothing to lint." (no `.go` files at
   all) vs "Found Go files but none import go-cqrs-lite. Nothing to lint."
   (packages loaded, none import go-cqrs-lite). The old message conflated both.

5. **Doctor warning.** `doctor` now checks `LoadErrors` and prints a
   "WARNING: package loading was partial" block before the feature profile.

6. **Feature detection skips errored packages.** `DetectFeatures` now skips
   packages with `len(pkg.Errors) > 0` in its import-based detection pass,
   making `lint` and `doctor` read the same (error-free) package set.

## Consequences

- A broken build now produces a clear, actionable error message with a non-zero
  exit code instead of a misleading "Nothing to lint."
- `lint` and `doctor` agree on the package set they analyze.
- `--strict` enables CI pipelines to fail on any partial analysis.
- The `PackageLoadError` struct (`Module`, `PkgPath`, `Errors`) provides
  enough context for users to diagnose the root cause without re-running `go
build` manually (though the message suggests doing so).
- Existing callers of `BuildContext` that don't check `LoadErrors` continue to
  work (additive change, not breaking).
