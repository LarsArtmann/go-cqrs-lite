# Blocked Items — External Dependencies

These items are blocked by upstream dependencies and cannot be resolved within this repository.

## 1. transport/grpc Workspace Integration ✅ Resolved

**Original blocker:** `cockroachdb/pebble` → `cockroachdb/errors@v1.14.0` pulled the monolithic `google.golang.org/genproto`, which conflicted with `grpc v1.81.1`'s split `genproto/googleapis/rpc`.

**Resolution:** A workspace-level `replace` directive in `go.work` pins the monolithic `google.golang.org/genproto` to `v0.0.0-20250603155806-513f23925822`, a version where the `googleapis/rpc` packages have been split out into their own module. This removes the ambiguous package overlap while keeping the packages still hosted in the monolithic module available to other workspace members.

**Result:** `transport/grpc` is now a first-class member of `go.work`; `go build ./...` and `go test ./...` across the workspace include it.

## 2. JSON v2 Build Tag Removal

**Blocker:** Go stdlib `encoding/json/v2` is behind the `goexperiment.jsonv2` build tag in Go 1.26. It's experimental at the toolchain level.

**Impact:** JSON v2 is **fully adopted** — ~25 production files import `encoding/json/v2` directly (`codec/json.go`, `event/types.go`, `schema/validator.go`, all `catalog/*`, `encryption/*`, `signing/*`, etc.). The `goexperiment.jsonv2` tag is required on every `go build`/`go test` invocation. The tag itself is the only remaining friction.

**Resolution:** Remove the build tag when Go graduates json/v2 from experimental (expected Go 1.27+). No code changes needed — just drop the tag from `flake.nix` and `scripts/check-module-isolation.sh`.

## 3. Arena Allocation — Removed

**Status:** Deleted (2026-07-11). The 36-line `event/arena_experiment.go` stub had zero consumers, no tests, and provided no real value — arena-allocating an `ImmutableEvent` struct header while its slice/map fields remain heap-allocated saves nothing on GC pressure. The `goexperiment.arenas` build tag was removed from `flake.nix` and `scripts/check-module-isolation.sh`.
