# ADR-0014: Test-Only Dependencies in go.mod

> **SUPERSEDED (2026-07-05):** This ADR deferred extracting `eventtest` as
> a separate module. That extraction was subsequently done by
> [ADR-0045](0045-eventtest-module-path-fix.md), which moved the directory
> to match the module path and tagged it as v0.1.0. The analysis below is
> preserved as historical context.

## Status

Superseded by [0045](0045-eventtest-module-path-fix.md)

## Context

Go does not separate test-only dependencies from production dependencies in `go.mod`. When a module like `event` imports `command` and `memory` only in `_test.go` files and `eventtest/`, those dependencies appear in `go.mod` as if they were production imports.

This creates the illusion of circular dependencies in the module graph. Static analysis tools, architecture reviews, and `go mod graph` all report bidirectional edges between `event ↔ command`, `event ↔ memory`, and `memory ↔ snapshot`.

## Decision

We acknowledge that:

1. **Production code is a clean DAG.** The actual import graph (excluding `_test.go` and test helper packages) has no cycles. `event` production code imports exactly: `id`, `codec`, `go-branded-id`, `go-error-family`, and `oklog/ulid`.

2. **Test-only deps are not cycles.** The apparent bidirectional edges exist only because Go bundles test deps into `go.mod`. They do not affect compilation order, runtime behavior, or consumer builds.

3. **Extracting `eventtest` as a separate module** would eliminate the test-dep bloat in `event/go.mod`, but is deferred because it requires significant structural changes (new `go.mod`, updated imports across all consumers).

4. **We do NOT treat go.mod cycles as architecture problems.** Architecture reviews and diagrams must distinguish between production imports and test-only imports before claiming "circular dependencies."

## Consequences

- Consumers who import `event/v2` transitively pull in test deps of the event module. This increases binary size but does not affect correctness.
- Architecture tooling should analyze production imports (excluding `_test.go` and `eventtest/`) separately from `go.mod` listings.
- The `nix run .#check-layers` command enforces the production dependency DAG, not the full `go.mod` listing.

## Date

2026-06-10
