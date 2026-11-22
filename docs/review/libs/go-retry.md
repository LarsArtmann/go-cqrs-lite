# go-retry — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-retry` @ 356fbd4 (latest local tag v0.9.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G1

**Verdict:** UNDERUSED-SEVERE
**Score:** 1.5/8 capabilities fully leveraged = 19% (low)

go-cqrs-lite is the library's only direct consumer and is _named by the
library_ as the CQRS-wrapped face of it — yet the wrapper does not call
`retry.Do` at all. `middleware/v4` hand-rolls its own attempt loop and
borrows exactly one function, `ComputeDelay`. The repo's own linter tells
fleet apps to "use retry.Do" while the repo's own event publisher hand-rolls
a fifth loop with no jitter and no exhaustion sentinel. ADR-0145 sanctions
per-concern loops for three internal classes; it does not sanction the
middleware drift, and its claim that `middleware/retry.go` is
"go-retry-powered" is only half true.

## 1. What the library is

A dependency-light retry core: `Do(ctx, cfg, fn)` and `DoWithValue` with
exponential backoff + additive jitter, a validated `Config`
(`MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`, `IsRetryable`
defaulting to `errorfamily.IsRetryable`, `DelayFunc`, `OnRetry`,
`OnExhausted`), per-call options (`WithOnRetry`, `WithExhausted`,
`WithJitter`, `WithRandomSource`, `WithIsRetryable`, `WithDelayFunc`),
exported `Backoff`/`ComputeDelay`, `FromPolicy(errorfamily.RetryPolicy)`,
and terminal sentinels `ErrExhausted` / `ErrCanceled` /
`ErrDeadlineExceeded` that chain the last attempt error and promote its
error-family code outermost. Deliberately no CQRS types and no OTel SDK:
`go-retry/README.md:17` points CQRS users at
`go-cqrs-lite/middleware/v4` for the wrapped variant. Extracted from this
repo (ADR-0064; shim removed by ADR-0128).

Sources: `go-retry/README.md` (config table, error model, nesting),
`go-retry/retry.go`, `go-retry/config.go`.

## 2. Usage census in go-cqrs-lite (evidence)

- Pins: **v0.9.0 in 5 go.mod files** — one direct (`middleware/v4/go.mod:18`)
  and four indirect (`commandlifecycle`, `example/taskmanager`,
  `integration/v4:102`, `system/v4:66`). Latest local tag is v0.9.0 → **on
  latest**.
- Production import sites: exactly **one** — `middleware/v4/retry.go:10`
  (alias `retrypkg`). The other two mentions are documentation artifacts:
  `cmd/cqrs-lint/v4/pkg/rules/adoption/f030.go` and
  `cmd/doc-check/v4/recipes_catalog_meta.go` (lint-rule and recipe text).
- Real call surface: **one symbol**. `middleware/v4/retry.go:152` calls
  `retrypkg.ComputeDelay(config.InitialDelay, config.MaxDelay,
  config.Multiplier, attempt)`. That is the entire production usage.
- Everything else in `middleware/v4/retry.go:75-149` is a hand-rolled loop:
  own span-per-attempt (`:86`), own retryable check (`:105`), own
  backoff sleep with `time.NewTimer` + `ctx.Done()` (`:126-139`), own
  exhaustion wrap into local sentinel `ErrRetryExhausted`
  (`middleware/v4/errors.go:21`) and cancellation wrap into
  `ErrRetryCanceled` (`middleware/v4/errors.go:27`).
- Meanwhile the repo's own tooling advertises the library:
  `cmd/cqrs-lint/v4/pkg/rules/boilerplate/b004_b008.go:241-252` flags
  "hand-written retry loops" and suggests "use retry.Do" /
  `retry.WithMaxRetries(3)`; `cmd/cqrs-lint/v4/pkg/rules/performance/p007.go:65-71`
  flags manual backoff loops with the same replacement.

## 3. Capability inventory

| #  | Capability                                                                                      | Grade    | Evidence / reason                                                                                                                                                                                                                                          |
| -- | ----------------------------------------------------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Retry loop `Do`                                                                                 | Missed   | 0 production call sites; `middleware/v4/retry.go:75-149` re-implements it inline                                                                                                                                                                           |
| 2  | `ComputeDelay` / `Backoff`                                                                      | **Full** | `middleware/v4/retry.go:152` — the one borrowed symbol                                                                                                                                                                                                     |
| 3  | `Config`/`DefaultConfig`/`Validate`                                                             | Missed   | `middleware/v4` declares its own `RetryConfig` + `Validate` (`middleware/v4/middleware.go:31`, `middleware/v4/retry.go:22-27`) — parallel knobs, no reuse                                                                                                  |
| 4  | Hooks (`OnRetry`, `OnExhausted`)                                                                | Missed   | logging and dead-letter entry are hand-coded inside the loop (`middleware/v4/retry.go:114-122, 140-145`)                                                                                                                                                   |
| 5  | Terminal sentinels + deadline/cancel split (`ErrExhausted`/`ErrCanceled`/`ErrDeadlineExceeded`) | Missed   | local duplicates `ErrRetryExhausted`/`ErrRetryCanceled` (`middleware/v4/errors.go:21,27`); the loop maps _every_ `ctx.Done()` to cancel (`middleware/v4/retry.go:130-136`) — the exact mislabeling go-retry v0.4.0 fixed (`go-retry/CHANGELOG.md` [0.4.0]) |
| 6  | Jitter strategies (`WithJitter`, `WithRandomSource`)                                            | Partial  | additive jitter is inherited via `ComputeDelay`, but not injectable — middleware retry timings stay untestable/deterministic-uncontrolled                                                                                                                  |
| 7  | Per-call options (`With*` tail)                                                                 | Missed   | 0 call sites (nothing calls `Do` to pass them to)                                                                                                                                                                                                          |
| 8  | `DoWithValue`                                                                                   | N/A      | consumer's retry surface is error-only middleware handlers; no value-returning retry call site exists today                                                                                                                                                |
| 9  | `DelayFunc` (server-provided Retry-After)                                                       | N/A      | transport-agnostic library has no server-delay feedback channel; flips the day an HTTP ingress honors Retry-After                                                                                                                                          |
| 10 | `FromPolicy(errorfamily.RetryPolicy)`                                                           | Missed   | 0 call sites; would bridge errorfamily's advisory policies into the wrapper's config (see `go-error-family.md` gap 3)                                                                                                                                      |

Score derivation: rows N=10, N/A rows 8 and 9 → **A=8**. Full=1 (row 2),
Partial=1 (row 6 → 0.5), Missed rows 1, 3, 4, 5, 7, 10 → 0.
Score = round(100 × 1.5 / 8) = **19% → low band**.

## 4. Gap analysis

1. **The wrapper isn't what either repo says it is.** `go-retry/README.md:17`
   markets `middleware/v4` as the CQRS-wrapped variant; ADR-0145's table
   (`docs/adr/0145-retry-idioms-are-per-concern.md`) describes
   `middleware/retry.go` as "go-retry-powered". In code, the middleware
   shares only the delay formula. Consequences already shipped: the
   deadline-vs-cancel distinction go-retry fixed in v0.4.0 never arrived
   here (a shutdown and a timeout both surface as `ErrRetryCanceled`,
   `middleware/v4/retry.go:130-140`), and v0.9.0's
   promote-last-attempt's-code-outermost behavior is absent (the wrapper's
   `middleware.retry_exhausted` code always wins,
   `middleware/v4/retry.go:147-149`). Both are loop-internals the library
   maintains and this fork does not. The rebuild is small because go-retry's
   hooks were designed for exactly this wrap:

   ```go
   // skip-validate
   return retry.Do(ctx, cfg, func(ctx context.Context, attempt int) error {
       attemptCtx, span := cqrsotel.StartSpan(ctx, retryTracer(),
           fmt.Sprintf("retry.attempt.%d", attempt), cqrsotel.SpanKindInternal)
       defer span.End()
       return next(attemptCtx, msg)
   },
       retry.WithOnRetry(func(attempt int, delay time.Duration, err error) {
           logger.Warn("retry attempt", "operation", entry.Type,
               "attempt", attempt, "delay", delay, "error", err)
       }),
       retry.WithExhausted(func(attempts int, err error) {
           if config.OnDeadLetter != nil {
               config.OnDeadLetter(ctx, deadLetter(entry, attempts, err))
           }
       }),
   )
   ```

   Sentinel compatibility is one `errors.Is(err, retry.ErrCanceled)` → wrap
   `ErrRetryCanceled` mapping; spans, DLQ, and the config surface stay.

2. **`event/v4.PublishRetry` is a fifth loop that escaped ADR-0145's
   table.** `event/v4/middleware.go:48-89` (and its v5 copy-forward twin
   `core/v5/event/middleware.go:46`) hand-rolls `math.Pow` exponential
   backoff with **no jitter** (thundering-herd on broker blips), returns
   **raw `ctx.Err()`** on cancellation (loses family classification —
   everything else in the repo wraps), and returns the **bare last error**
   on exhaustion (no sentinel, no attempt count). ADR-0145 class 1 says
   transport/pipeline op retry belongs to the go-retry idiom;
   `PublishRetry` is exactly that class. Internally delegating to
   `retry.Do` keeps the `PublishMiddleware` signature and fixes all three
   defects at once.

3. **The linter teaches what the repo doesn't practice.** `cqrs-lint` rules
   B004/B008 (`cmd/cqrs-lint/v4/pkg/rules/boilerplate/b004_b008.go:149`)
   and P007 (`cmd/cqrs-lint/v4/pkg/rules/performance/p007.go:13`) flag
   manual retry loops and prescribe `retry.Do`. Fleet apps that obey will
   adopt go-retry more deeply than the library that ships them the rule.

4. **The sanctioned loops stay, and that is fine — with one caveat.**
   `metaengine/v4/transient_commit.go:50` (bitshift + jitter),
   `metaengine/v4/replicator.go:134` (linear 50ms), and
   `metaengine/dgraphengine/v4/transaction.go:190` (classified contention)
   are ADR-0145 classes 3-4 with tuned semantics; adding a go-retry require
   to those Tier-3 modules would spend dep budget for no fit. The caveat:
   none of the four sanctioned sites distinguishes deadline from cancel
   either, so the ADR's own alignment rule ("every loop must be ctx-aware")
   could cheaply standardize on go-retry's split sentinels as the fleet
   vocabulary.

## 5. Version currency

- Pinned **v0.9.0** (1 direct + 4 indirect) = latest local tag v0.9.0
  (2026-10-27). **ON LATEST.** The bump was completed as
  `go-retry/TODO_LIST.md` T77 asked ("Consumer sweep: bump go-cqrs-lite
  `middleware/` to v0.9.0") — that library TODO can now be closed from the
  consumer side.
- Because only `ComputeDelay` is used, none of v0.3-v0.9's loop-behavior
  improvements (deadline split, chained context errors, last-attempt code
  promotion, hook semantics fixes) reached the consumer. Being "on latest"
  here is nearly vacuous: the pin is current, the capability is not.
- `go-retry/CHANGELOG.md` [Unreleased] is a lint-stack note only; nothing
  pending.

## 6. The fleet argument

"I have MANY projects and limited time; if every project re-implements or
reintegrates everything I will only fix bugs all day long." Retry loops are
the canonical example: they look trivial and accrete edge cases (jitter
caps, overflow, deadline-vs-cancel, nested-loop amplification — go-retry's
own changelog fixed real bugs in every one of those classes). This repo
currently maintains four in-house loop implementations that the library
already hardened. When the next retry bug is found in `Do`, the middleware
wrapper and `PublishRetry` will not inherit the fix — the fleet-wide
guarantee the extraction doctrine (ADR-0128) was supposed to buy.

## 7. Ranked opportunities

1. **Rebuild `middleware/v4/retry.go`'s loop on `retry.Do` + hooks**
   (high): ~70 lines of loop become ~30 lines of wiring; the wrapper gains
   the deadline sentinel, code promotion, and every future library fix.
2. **Delegate `event/v4.PublishRetry` (and the v5 twin) to `retry.Do`**
   (high): adds jitter, wrapped cancellation, and an exhaustion sentinel to
   the fleet's publish path.
3. **Close T77 in `go-retry/TODO_LIST.md`** (trivial): the consumer sweep
   it tracks is done.
4. **Fix the two "go-retry-powered" claims** (trivial): ADR-0145's table
   row and `go-retry/README.md:17` should describe reality after
   opportunity 1 lands (or before, honestly).
5. **Adopt the deadline/cancel sentinel vocabulary in the sanctioned
   loops** (low): map ADR-0145's ctx-awareness rule onto
   `retry.ErrDeadlineExceeded` vs `retry.ErrCanceled` naming without adding
   the dependency.

## 8. Reconsider when

- An HTTP ingress lands in the fleet (`DelayFunc` flips from N/A to the
  Retry-After handler).
- A value-returning retry site appears (dispatcher lookups, kv reads) —
  `DoWithValue` stops being N/A.
- go-retry ships per-attempt context decoration or span hooks natively —
  then even less wrapper code remains.
- The v5 core cut reconsiders middleware shape (free window per
  ADR-0152): the `retry.Do`-based wrapper is the natural v5 baseline.

## 9. How to verify

```
cd /home/lars/projects/go-cqrs-lite
rg -n 'larsartmann/go-retry' -g go.mod            # 5 pins, all v0.9.0
rg -l 'larsartmann/go-retry' --type go            # middleware/v4/retry.go + 2 doc/lint files
rg -n 'retry\.Do|retrypkg\.' --type go            # real call sites: only retrypkg.ComputeDelay
sed -n '75,160p' middleware/v4/retry.go           # the hand-rolled loop + ComputeDelay
sed -n '20,28p' middleware/v4/errors.go           # ErrRetryExhausted / ErrRetryCanceled
sed -n '48,89p' event/v4/middleware.go            # PublishRetry: math.Pow, no jitter
sed -n '46,60p' core/v5/event/middleware.go       # v5 twin (art-dupl:accept)
sed -n '48,70p' metaengine/v4/transient_commit.go # sanctioned loop (ADR-0145 class 3)
cat docs/adr/0145-retry-idioms-are-per-concern.md
git -C /home/lars/projects/go-retry log -1 --format='%H'
git -C /home/lars/projects/go-retry tag --list | sort -V | tail -1   # v0.9.0
```

Sibling reports in this directory: `go-error-family.md`, `go-codec.md`,
`go-idempotency.md`.
