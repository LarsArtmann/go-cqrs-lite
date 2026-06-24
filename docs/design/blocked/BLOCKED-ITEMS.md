# Blocked Items — External Dependencies

These items are blocked by upstream dependencies and cannot be resolved within this repository.

## 1. transport/grpc Workspace Integration

**Blocker:** `cockroachdb/pebble` → `cockroachdb/errors@v1.14.0` pulls the monolithic `google.golang.org/genproto`, which conflicts with `grpc v1.81.1`'s split `genproto/googleapis/rpc`.

**Impact:** `transport/grpc` cannot be added to `go.work`. It builds and tests successfully only with `GOWORK=off`.

**Workaround:** CI tests transport/grpc in isolation via the per-module matrix (`cd transport/grpc && GOWORK=off go test ./... -race`).

**Resolution:** Requires cockroachdb/errors to drop the monolithic genproto. Tracked at [cockroachdb/errors#79](https://github.com/cockroachdb/errors/issues/79).

## 2. JSON v2 Codec Stabilization

**Blocker:** Go stdlib `encoding/json/v2` is behind the `goexperiment.jsonv2` build tag. It's experimental and subject to change.

**Impact:** `codec/jsonv2_experiment.go` exists but is gated behind the build tag. Cannot be the default codec until the API stabilizes in a Go release.

**Resolution:** Wait for Go stdlib to stabilize JSON v2 (expected in Go 1.27 or later). The experiment file is ready — just remove the build tag when the API is stable.

## 3. Arena Allocation Stabilization

**Blocker:** Go's `arena` package is behind the `goexperiment.arenas` build tag. It's experimental and the API may change.

**Impact:** `event/arena_experiment.go` exists but is gated. Cannot be used in production.

**Resolution:** Wait for Go to stabilize the arena API. The experiment file demonstrates the pattern; implementation will need to track any API changes.
