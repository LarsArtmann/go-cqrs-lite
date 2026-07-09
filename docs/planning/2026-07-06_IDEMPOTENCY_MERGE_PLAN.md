# Idempotency Merge Plan: `idempotency/` → `middleware/idempotency/`

> **Status:** ✅ COMPLETED (2026-07-08)
> **Effort:** ~30 minutes execution (mechanical moves + one new file)
> **Risk:** Low — zero internal consumers, not in api-stability tracking, not in flake.nix testModules
> **Breaking:** Import path change for external consumers (`idempotency/v3` → `middleware/v3/idempotency`). Pre-v4, acceptable.

> ### ⚠ Architectural Deviation from Plan
>
> The plan below describes moving the Store primitive into `middleware/idempotency/`
> (a sub-package sharing middleware's go.mod). **The actual implementation deviated:**
> `idempotency/` was kept as a **separate lightweight module** (depends on `kv/v3` +
> `go-error-family` only), and only the generic middleware factory was added to
> `middleware/`.
>
> **Rationale:** Composability — `middleware/` has heavy deps (otel, ginkgo, gomega,
> sqlite). A consumer who wants just the dedup Store shouldn't drag those in. Keeping
> `idempotency/` as a leaf module preserves its independence.
>
> **What was actually done:**
>
> - `middleware/idempotency.go` — generic `NewIdempotency[M]` + 3 wrappers (created)
> - `middleware/idempotency_test.go` — 11 tests (created)
> - `idempotency/middleware.go` + `middleware_test.go` — deleted (replaced by above)
> - `idempotency/` module — kept as separate module, deps slimmed to `kv/` + `go-error-family`
> - `idempotency/` added to `flake.nix` testModules + `cmd/api-stability` tracking
> - `QueryIdempotency` panics at construction if `keyExtractor` is nil (nil safety guard)

---

## 1. Problem Statement

The `idempotency/` module has three architectural problems:

1. **Command-only middleware in a domain-agnostic package.** `CommandIdempotency` is the only wired middleware, but the `Store` primitive is key-type-agnostic. The middleware should generalize to events and queries (the `middleware/` pattern already supports this).

2. **Artificial dependencies.** The module depends on `event/` solely for error classification (`event.NewConflict`, `event.Wrapf`) — which is a line-for-line re-export of `go-error-family`. It depends on `command/` solely for the middleware factory. Neither dependency is essential to the storage primitive.

3. **Zero adoption despite being built and tested.** No module imports it. No `go.mod` requires it. It's not in `flake.nix` testModules. It's not in the api-stability golden file. Dead infrastructure.

---

## 2. Target Architecture

### 2.1 The Split

The current module conflates two concerns. Split them:

| Concern                                                               | Current location            | New location                                 | Dependencies                                                     |
| --------------------------------------------------------------------- | --------------------------- | -------------------------------------------- | ---------------------------------------------------------------- |
| Storage primitive (`Store`, `MemoryStore`, `KVStore`, `ErrDuplicate`) | `idempotency/`              | `middleware/idempotency/` (sub-package)      | `go-error-family` (direct), `kv/` (optional, for `KVStore` only) |
| Middleware factory (generic + command/event/query wrappers)           | `idempotency/middleware.go` | `middleware/idempotency.go` (parent package) | `command/`, `event/`, `query/`, `idempotency.Store`              |

### 2.2 Dependency Graph Change

```
BEFORE                                     AFTER
──────                                     ─────

  command/ ←─── idempotency/                command/ ←─── middleware/
     ↑              ↑                          ↑              ↑
     |              |                          |         middleware/idempotency/
  event/ ←─────── (for errors)             event/           (Store + MemoryStore + KVStore)
     ↑              ↑                          ↑              ↑
  go-error-family  kv/                    go-error-family  kv/

Layer 2: idempotency/ → event, kv, command   middleware/idempotency/ → event (errors), kv (optional)
                                             middleware/ → command, event, query + middleware/idempotency/
```

The sub-package sheds its `command/` dependency entirely. `go-error-family` is imported directly instead of through the `event/` re-export (see Architecture Layers Reconsidered for the repo-wide fix).

### 2.3 Why a Sub-Package, Not Flatten?

|                    | Flatten into `middleware/`                                                                    | Sub-package `middleware/idempotency/`                             |
| ------------------ | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Name collisions    | `Store`, `MemoryStore`, `ErrDuplicate` too generic — need renaming to `IdempotencyStore` etc. | Names stay clean: `idempotency.Store`, `idempotency.ErrDuplicate` |
| Precedent          | None — middleware/ is one flat package                                                        | Established: `catalog/schema/`, `kv/viewstoretest/` share go.mod  |
| Conceptual clarity | Storage primitive mixed with handler middleware                                               | Storage primitive has own namespace; middleware factory in parent |

**Decision: sub-package.** One extra import line is worth keeping clean names.

---

## 3. File Layout

### 3.1 Target Structure

```
middleware/
├── idempotency.go                     # NEW: generic NewIdempotency[M] + 3 wrappers
├── idempotency_test.go                # NEW: tests for generic factory + 3 wrappers
├── idempotency/
│   ├── doc.go                         # MOVED from idempotency/doc.go (updated)
│   ├── store.go                       # MOVED from idempotency/store.go (import fix)
│   ├── store_test.go                  # MOVED from idempotency/store_test.go (import fix)
│   ├── kv_store.go                    # MOVED from idempotency/kv_store.go (import fix)
│   └── kv_store_test.go               # MOVED from idempotency/kv_store_test.go (import fix)
├── ... (all existing middleware files unchanged)

DELETED:
├── idempotency/                       # Entire module deleted
│   ├── go.mod                         # Gone
│   ├── go.sum                         # Gone
│   ├── README.md                      # Gone (content merged into doc.go)
│   ├── doc.go                         # Moved
│   ├── store.go                       # Moved
│   ├── store_test.go                  # Moved
│   ├── kv_store.go                    # Moved
│   ├── kv_store_test.go               # Moved
│   ├── middleware.go                  # NOT moved — logic replaced by generic factory
│   └── middleware_test.go             # NOT moved — replaced by idempotency_test.go
```

### 3.2 Deleted Symbols

| Symbol                     | Why deleted                                                                            | Replacement                                    |
| -------------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `idempotency.KeyExtractor` | Command-specific type alias. Each wrapper uses `func(M) string` directly.              | Inline `func(command.Command) string`          |
| `idempotency.CommandIDKey` | One-liner: `cmd.ID().String()`. Default behavior of `CommandIdempotency` when key=nil. | Nil default in `middleware.CommandIdempotency` |

No consumer references either symbol in executable code (only comments in `deriver/` and `id/`).

---

## 4. Code: The Generic Factory

The core middleware follows the exact pattern established by `NewValidation`, `NewTracing` in `middleware/`:

```go
// middleware/idempotency.go
package middleware

import (
    "context"
    "errors"
    "time"

    "github.com/larsartmann/go-cqrs-lite/command/v3"
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/middleware/v3/idempotency"
    "github.com/larsartmann/go-cqrs-lite/query/v3"
)

// NewIdempotency returns a generic middleware that rejects duplicate messages
// using the provided idempotency.Store. On first occurrence of a key, records
// it with the given TTL and passes the message to next. On subsequent
// occurrences within TTL, returns idempotency.ErrDuplicate without calling next.
//
// Empty key ("") skips dedup for that message (pass-through).
func NewIdempotency[M any](
    adapter MessageAdapter[M],
    store idempotency.Store,
    ttl time.Duration,
    keyExtractor func(M) string,
) Middleware[M] {
    return func(next Handler[M]) Handler[M] {
        return func(ctx context.Context, msg M) error {
            key := keyExtractor(msg)
            if key == "" {
                return next(ctx, msg)
            }
            if err := store.CheckAndRecord(ctx, key, ttl); err != nil {
                if errors.Is(err, idempotency.ErrDuplicate) {
                    return err
                }
                return event.Wrapf(err, event.Transient,
                    "middleware."+adapter.Kind+"_idempotency",
                    "check-and-record failed for %s %s",
                    adapter.Kind, adapter.ExtractType(msg))
            }
            return next(ctx, msg)
        }
    }
}

// CommandIdempotency wires the Store into a command.Dispatcher middleware chain.
// Pass nil for keyExtractor to use the command's minted ID (cmd.ID().String()).
func CommandIdempotency(
    store idempotency.Store,
    ttl time.Duration,
    keyExtractor func(command.Command) string,
) command.Middleware {
    if keyExtractor == nil {
        keyExtractor = func(cmd command.Command) string { return cmd.ID().String() }
    }
    return AsCommand(NewIdempotency(CommandAdapter, store, ttl, keyExtractor))
}

// EventIdempotency wires the Store into an event handler middleware chain.
// Pass nil for keyExtractor to use the event's minted ID (evt.ID().String()).
//
// NOTE: For ordered event consumption (projections), checkpoint-based dedup
// (projectionhost) is structurally stronger than key-based dedup. Use this
// middleware when you don't own the checkpoint (webhooks, external sinks,
// cross-system delivery) or as defense-in-depth alongside checkpoints.
func EventIdempotency(
    store idempotency.Store,
    ttl time.Duration,
    keyExtractor func(event.Event) string,
) event.Middleware {
    if keyExtractor == nil {
        keyExtractor = func(evt event.Event) string { return evt.ID().String() }
    }
    return AsEvent(NewIdempotency(EventAdapter, store, ttl, keyExtractor))
}

// QueryIdempotency wires the Store into a query dispatcher middleware chain.
// No default key — queries have no built-in identity. Caller must supply
// a keyExtractor; returning "" skips dedup for that query.
func QueryIdempotency(
    store idempotency.Store,
    ttl time.Duration,
    keyExtractor func(query.Query) string,
) query.Middleware {
    return AsQuery(NewIdempotency(QueryAdapter, store, ttl, keyExtractor))
}
```

### 4.1 The Sub-Package Storage Primitive

`middleware/idempotency/store.go` — moved from `idempotency/store.go` with one change:

```diff
- var ErrDuplicate = event.NewConflict(
+ var ErrDuplicate = errorfamily.NewConflict(
      "idempotency.duplicate",
      "key has already been recorded",
  )
```

And in error wrapping:

```diff
- return event.Wrapf(err, event.Transient, "idempotency.kv.seen", "key %q", key)
+ return errorfamily.Wrapf(err, errorfamily.Transient, "idempotency.kv.seen", "key %q", key)
```

This drops the `event/` dependency from the sub-package entirely. The sub-package depends on:

- `go-error-family` (direct — for error classification)
- `kv/` (for `KVStore` adapter only — optional)

---

## 5. Call-Site Migration

### Before

```go
import "github.com/larsartmann/go-cqrs-lite/idempotency/v3"

store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()
cmds.Use(idempotency.CommandIdempotency(store, 10*time.Minute, nil))
```

### After

```go
import (
    "github.com/larsartmann/go-cqrs-lite/middleware/v3"
    "github.com/larsartmann/go-cqrs-lite/middleware/v3/idempotency"
)

store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()
cmds.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))

// NEW — now possible:
bus.Use(middleware.EventIdempotency(store, 10*time.Minute, nil))
qry.Use(middleware.QueryIdempotency(store, 10*time.Minute, keyFn))
```

One extra import line. Same `idempotency.` qualifier for the Store. Three message types instead of one.

---

## 6. Execution Phases

### Phase 1: Create the sub-package (no deletion)

| Step | Action                                                                        | Detail                                                |
| ---- | ----------------------------------------------------------------------------- | ----------------------------------------------------- |
| 1.1  | `git mv idempotency/store.go middleware/idempotency/store.go`                 | Storage primitive                                     |
| 1.2  | `git mv idempotency/kv_store.go middleware/idempotency/kv_store.go`           | KV adapter                                            |
| 1.3  | `git mv idempotency/doc.go middleware/idempotency/doc.go`                     | Package docs                                          |
| 1.4  | `git mv idempotency/store_test.go middleware/idempotency/store_test.go`       | Store tests                                           |
| 1.5  | `git mv idempotency/kv_store_test.go middleware/idempotency/kv_store_test.go` | KV store tests                                        |
| 1.6  | Fix imports in moved test files                                               | `idempotency/v3` → `middleware/v3/idempotency`        |
| 1.7  | Fix error classification in store.go + kv_store.go                            | `event.NewConflict` → `errorfamily.NewConflict`, etc. |
| 1.8  | Update doc.go                                                                 | Package path, cross-references                        |

### Phase 2: Create the generic middleware in parent

| Step | Action                                                                                                   |
| ---- | -------------------------------------------------------------------------------------------------------- |
| 2.1  | Create `middleware/idempotency.go` (code from §4)                                                        |
| 2.2  | Create `middleware/idempotency_test.go` — port `idempotency/middleware_test.go`, add Event + Query tests |

### Phase 3: Update middleware/go.mod

| Step | Action                                                            |
| ---- | ----------------------------------------------------------------- |
| 3.1  | Add `github.com/larsartmann/go-cqrs-lite/kv/v3` as direct require |
| 3.2  | Add `replace github.com/larsartmann/go-cqrs-lite/kv/v3 => ../kv`  |
| 3.3  | Run `cd middleware && GOWORK=off go mod tidy`                     |

### Phase 4: Delete old module

| Step | Action                                           | Safety                                                       |
| ---- | ------------------------------------------------ | ------------------------------------------------------------ |
| 4.1  | `trash idempotency/`                             | After confirming all needed files are moved. NEVER use `rm`. |
| 4.2  | Remove `./idempotency` from `go.work`            | Line 22                                                      |
| 4.3  | Verify flake.nix doesn't reference `idempotency` | Already not in testModules                                   |

### Phase 5: Update documentation

| File                                        | Change                                                                                 |
| ------------------------------------------- | -------------------------------------------------------------------------------------- |
| `AGENTS.md`                                 | Remove `idempotency` from Modules row; remove from test command; note under middleware |
| `SKILL.md` + `.agents/skills/go-cqrs-lite/` | Update module decision matrix; update import paths                                     |
| `docs/DOMAIN_LANGUAGE.md`                   | Update import paths (lines 202, 475-476)                                               |
| `deriver/deriver.go:111`                    | Comment: `idempotency.CommandIDKey` → `middleware.CommandIdempotency` default key      |
| `deriver/doc.go:12`                         | Same comment update                                                                    |
| `id/command_id.go:40`                       | Same comment update                                                                    |
| `docs/feedback/sec-consumer-feedback.md`    | Update import path references                                                          |
| `CHANGELOG.md`                              | Add entry                                                                              |

### Phase 6: Add to api-stability tracking

| Step | Action                                                                         |
| ---- | ------------------------------------------------------------------------------ |
| 6.1  | Add `"middleware/idempotency"` to module list in `cmd/api-stability/main.go`   |
| 6.2  | Regenerate golden file: `cd cmd/api-stability && GOWORK=off go run . --update` |

### Phase 7: Verify

```bash
nix run .#build                       # all modules compile
nix run .#test                        # all tests pass
nix run .#lint                        # no lint issues
cd middleware && GOWORK=off go test ./... -count=1
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../AGENTS.md
cd cmd/api-stability && GOWORK=off go run .
```

---

## 7. Risks

| Risk                                            | Likelihood                            | Impact                               | Mitigation                                 |
| ----------------------------------------------- | ------------------------------------- | ------------------------------------ | ------------------------------------------ |
| External consumer breakage (import path change) | Low (zero internal consumers, pre-v4) | Medium — consumer must update import | CHANGELOG entry; mechanical find-replace   |
| middleware/ dep budget exceeded (adding kv/)    | Low (kv/ is Layer 0/1)                | Low                                  | Run `nix run .#check-layers` after Phase 3 |
| go.work / flake.nix drift                       | Medium (manual edits)                 | Build failure                        | Phase 7 verification catches this          |
| Dead code in comment references                 | Low                                   | Misleading docs                      | Phase 5 updates all references             |

---

## 8. Why Not Also Generalize dedup/?

`dedup/` (the ring buffer) is a different concern entirely:

- **`dedup.Ring`**: fixed-capacity ring buffer for stream-boundary dedup (replay→live overlap). Unordered, lossy by design (evicts oldest), no TTL, no persistence. Layer 0 leaf (zero deps).
- **`idempotency.Store`**: TTL-based key store for at-least-once delivery dedup. Ordered claim (first writer wins), persistent (KV-backed option), bounded by TTL.

They solve different problems at different boundaries. Keeping them separate is correct.
