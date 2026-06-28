# Blocked Items — External Dependencies

These items are blocked by upstream dependencies and cannot be resolved within this repository.

## 1. transport/grpc Workspace Integration ✅ Resolved

**Original blocker:** `cockroachdb/pebble` → `cockroachdb/errors@v1.14.0` pulled the monolithic `google.golang.org/genproto`, which conflicted with `grpc v1.81.1`'s split `genproto/googleapis/rpc`.

**Resolution:** A workspace-level `replace` directive in `go.work` pins the monolithic `google.golang.org/genproto` to `v0.0.0-20250603155806-513f23925822`, a version where the `googleapis/rpc` packages have been split out into their own module. This removes the ambiguous package overlap while keeping the packages still hosted in the monolithic module available to other workspace members.

**Result:** `transport/grpc` is now a first-class member of `go.work`; `go build ./...` and `go test ./...` across the workspace include it.

## 2. JSON v2 Codec Stabilization

**Blocker:** Go stdlib `encoding/json/v2` is behind the `goexperiment.jsonv2` build tag. It's experimental and subject to change.

**Impact:** `codec/jsonv2_experiment.go` exists but is gated behind the build tag. Cannot be the default codec until the API stabilizes in a Go release.

**Resolution:** Wait for Go stdlib to stabilize JSON v2 (expected in Go 1.27 or later). The experiment file is ready — just remove the build tag when the API is stable.

## 3. Arena Allocation Stabilization

**Blocker:** Go's `arena` package is behind the `goexperiment.arenas` build tag. It's experimental and the API may change.

**Impact:** `event/arena_experiment.go` exists but is gated. Cannot be used in production.

**Resolution:** Wait for Go to stabilize the arena API. The experiment file demonstrates the pattern; implementation will need to track any API changes.
