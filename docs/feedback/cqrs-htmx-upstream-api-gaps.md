# Upstream Feedback: Two API Gaps Blocking cqrs-htmx Integration

**From:** cqrs-htmx consumer · **Date:** 2026-06-18

> **Status: RESOLVED** (2026-06-18) — Both gaps addressed in commits
> `132106e6` (projection split + 3 markers) and `c565d183` (remaining 5 markers
> exported for API consistency). `RunReplay`/`RunLive` shipped; all 8 marker
> types exported. See CHANGELOG `[Unreleased]` and ADR-0024.

---

## 1. `projection/v2`: Split `Runner.Run()` into replay + live phases

### The problem

cqrs-htmx's `usermgmt/es_projection_setup.go` needs read-your-writes consistency: after `StartProjections()` returns, the read model must reflect all previously committed events. Currently we hack it with a blind sleep:

```go
go func() {
    if err := runner.Run(context.Background()); err != nil { ... }
}()

time.Sleep(50 * time.Millisecond) // ← race-prone, flaky on slow CI, wrong
```

This is because `Runner.Run()` (`runner.go:103`) is a single blocking method that does replay + live subscription in one shot:

```go
func (r *Runner) Run(ctx context.Context) error {
    // ...
    if r.journal != nil {
        err := r.replay(ctx)          // ← happens-before subscribe, but caller can't observe
        if err != nil { return err }
    }
    return r.subscribeLive(ctx)       // ← blocks forever
}
```

The replay already completes synchronously before live subscription begins — we just can't get at that boundary.

### Proposed fix

Expose replay and live as separate public methods, keeping `Run()` as a convenience that calls both:

```go
// RunReplay replays historical events from the journal and returns when
// all registered projections have caught up to the current event stream.
// Returns ErrNoProjections if none are registered. No-op if journal is nil.
func (r *Runner) RunReplay(ctx context.Context) error { ... }

// RunLive subscribes to live events from the bus and blocks until ctx
// is cancelled or Close is called. Requires RunReplay to have been
// called first (if a journal is configured) for catch-up.
func (r *Runner) RunLive(ctx context.Context) error { ... }

// Run is a convenience wrapper: RunReplay + RunLive.
func (r *Runner) Run(ctx context.Context) error {
    if err := r.RunReplay(ctx); err != nil { return err }
    return r.RunLive(ctx)
}
```

### What this unblocks for cqrs-htmx

```go
if err := runner.RunReplay(ctx); err != nil { return err }  // synchronous
go func() { _ = runner.RunLive(ctx) }()                      // background
// ← read model is guaranteed caught up here, no sleep needed
```

Removes the only `time.Sleep` in the entire cqrs-htmx codebase and eliminates the flaky-test class it creates.

---

## 2. `id` package: Export marker types for downstream branding

### The problem

cqrs-htmx uses `go-branded-id` for its own `UserID` type in `usermgmt`. We'd like to provide a `BrandNamer` integration so consumers get human-readable ULID prefixes like `user_01HK...` instead of raw ULIDs.

But `go-branded-id`'s `BrandNamer` requires referencing the marker type:

```go
// What we want to do in cqrs-htmx:
namer := brandid.NewBrandNamer[id.UserMarker]()  // ← won't compile: userMarker is unexported
```

The three marker types that affect us are all lowercase:

| File                     | Type                | Used by                  |
| ------------------------ | ------------------- | ------------------------ |
| `id/user_id.go:4`        | `userMarker`        | `cqrshtmx.UserID`        |
| `id/correlation_id.go:4` | `correlationMarker` | `cqrshtmx.CorrelationID` |
| `id/request_id.go:4`     | `requestMarker`     | `cqrshtmx.RequestID`     |

### Proposed fix

Export the three marker types. This is a mechanical rename with zero behavior change — they're empty structs used only as type parameters:

```go
// user_id.go
type UserMarker struct{}       // was: userMarker
type UserID = Of[UserMarker]   // was: Of[userMarker]
func NewUserID() UserID        // etc — no internal calls change

// correlation_id.go
type CorrelationMarker struct{}

// request_id.go
type RequestMarker struct{}
```

### What this unblocks

Downstream packages can integrate `go-branded-id`'s `BrandNamer`, JSON formatters, and other type-parameterized tooling against the root module's ID types.

---

## Summary

| #   | Package         | Change                                                    | Impact on cqrs-htmx                     |
| --- | --------------- | --------------------------------------------------------- | --------------------------------------- |
| 1   | `projection/v2` | Expose `RunReplay()` + `RunLive()`                        | Eliminates `time.Sleep(50ms)` race hack |
| 2   | `id`            | Export `UserMarker`, `CorrelationMarker`, `RequestMarker` | Enables `BrandNamer` integration        |

Both are small, additive, backward-compatible changes (no existing API removed or renamed).
