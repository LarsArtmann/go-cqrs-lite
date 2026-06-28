# Investigation: Concurrent Session File Corruption

## Date

2026-06-28

## Summary

During a multi-session development sprint, two Go source files were found
corrupted with literal diff/patch syntax (`+++`, `@@`) embedded in the Go
source code. This document records the findings.

## Affected Files

- `middleware/generic.go` — contained `+++` patch markers in the middle of
  function bodies
- `middleware/retry_query_test.go` — same pattern, literal patch syntax

## Symptoms

- `go build` / `go test` failed with syntax errors
- The `+++` lines looked like unified diff headers (`+++ b/file.go`)
- Normal Go code appeared between the patch markers

## Root Cause

The corruption pattern (unified-diff syntax written into source files) is
consistent with an AI agent's edit tool writing diff artifacts directly into
the file content instead of applying them. When two or more AI sessions
operate on the same repository simultaneously, race conditions on the working
tree can cause:

1. **Partial writes** — Session A reads a file, Session B modifies it,
   Session A writes the stale version back, clobbering B's changes
2. **Tool failures** — An edit tool may write the diff representation instead
   of the final content when the file changed between read and write
3. **Patch syntax leaks** — The `+++`/`@@` markers are diff metadata that
   should never appear in source files

## Impact

- Build broken until files are restored via `git restore`
- Wasted debugging time investigating "syntax errors" that are actually
  diff artifacts

## Prevention

1. **Session isolation** — Use feature branches (`git switch -c feature`)
   instead of concurrent commits to master
2. **`git status` before work** — Always check for unexpected changes
3. **Pre-commit hooks** — The BuildFlow hook catches syntax errors before
   they reach the repository
4. **Tool validation** — Edit tools should validate that written content
   doesn't contain diff artifacts before saving

## Resolution

Files were restored with `git restore middleware/generic.go
middleware/retry_query_test.go` and the build was verified clean.
