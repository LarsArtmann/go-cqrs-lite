# ADR-0045: eventtest Module Path / Directory Alignment

Date: 2026-07-05
Status: Accepted

## Context

The `eventtest` module (test helpers for the `event` package) had:

- **Module path:** `github.com/larsartmann/go-cqrs-lite/event/v3/eventtest`
- **Directory:** `event/eventtest/` (mismatch!)
- **Tags:** `event/v3/eventtest/v3.3.0`, `event/v3/eventtest/v3.5.0` (v3 tags on a v0-path module)

External consumers could not resolve the module:

```
go get github.com/larsartmann/go-cqrs-lite/event/v3/eventtest
→ module github.com/.../event/v3@v3.5.0 found, but does not contain package .../event/v3/eventtest
```

Previous analysis attributed this to "Go doesn't support nested modules whose path is a sub-path of the parent module." **This was incorrect.**

## Root Cause (Two Defects)

### Defect 1: Directory / Module-Path Mismatch

Per the [Go module spec](https://go.dev/ref/mod), when fetching a module from VCS, Go computes the expected directory from the module path:

> The module's `go.mod` file must be in the subdirectory matching the part of the module's path after the repository root path.

A major-version suffix (`/v2+`) is only recognized when it is the **last** path element. For `.../event/v3/eventtest`:

- The `/v3/` is NOT the last element → it is a **literal directory name**, not a major-version suffix
- Go looks for `go.mod` at `event/v3/eventtest/go.mod`
- The actual file was at `event/eventtest/go.mod` → **permanent mismatch → unresolvable**

No tag (v0, v3, or otherwise) could fix this — the directory simply didn't match the path.

### Defect 2: Wrong Major-Version Tags

The module path's last element is `eventtest` (not `/vN`), so Go treats this as a **v0 module**. The tags `v3.3.0` and `v3.5.0` are v3 semver — incompatible with a v0-path module. Even if the directory had been correct, these tags would have been rejected by Go's MVS.

## Decision

1. **Move the directory** from `event/eventtest/` to `event/v3/eventtest/` so it matches the module path. This preserves ALL import paths (the module path is unchanged) — only the physical directory moves.

2. **Tag as v0:** Delete the wrong `v3.3.0`/`v3.5.0` tags. Create `event/v3/eventtest/v0.1.0` at the fix commit.

3. **Sync replace directives:** Added `scripts/sync-replaces.sh` to ensure every `go.mod` has `replace` directives for ALL transitive sibling deps (needed for `GOWORK=off` per-module builds). This fixed 28 pre-existing build failures across the monorepo.

## Consequences

- External consumers can now `go get github.com/larsartmann/go-cqrs-lite/event/v3/eventtest@v0.1.0` (with `GOPRIVATE=github.com/larsartmann/*` for the private repo, or once the repo is made public).
- All 53 modules build cleanly with `GOWORK=off`.
- The `-e` flag for `go mod tidy` is still needed for modules that transitively depend on eventtest through event's test files (event's test imports create a graph edge that `go mod tidy` follows). This is a minor inconvenience, not a correctness issue.
- `scripts/sync-replaces.sh` should be re-run after adding new sibling module dependencies.

## What Was NOT Done (and Why)

- **Did NOT flatten to a top-level module.** The import path `.../event/v3/eventtest` is used by ~92 files. Moving the directory (not the path) achieves the same fix with zero import-path churn.
- **Did NOT change the module path.** The path `.../event/v3/eventtest` correctly reflects that eventtest is part of the event v3 release line, even though it's technically a v0 module.
