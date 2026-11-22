# go-error-family — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-error-family` @ 898004e (latest local tag v0.12.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G1

**Verdict:** UNDERUSED-MODERATE
**Score:** 5/15 capabilities fully leveraged = 33% (low)

The core protocol is adopted to the maximum depth the doctrine allows: every
production module constructs classified errors through the library, the
repo's own CI gates drift-check its error codes against a taxonomy doc, and
its linter enforces the constructor style. What is untouched is the
surrounding *decision* and *presentation* surface: exit codes, CLI handling,
structured logging, retry policies, idempotent wrapping, the test-assertion
kit, and the `diagnose` rules engine. This is "share the protocol" done
right with "and here is everything else we built" left in the box.

## 1. What the library is

A zero-dependency structured error protocol: six behavioral Families
(Rejection, Conflict, Transient, Corruption, Infrastructure, Orchestration),
classification of arbitrary errors (`Classify`, `IsRetryable`), stable string
codes, sysexits-derived `ExitCode`, HTTP status mapping, CLI boundary
handling with message templates, `slog` helpers, registries for third-party
sentinels/classifiers, and four submodules (`agent`, `bridge` for
samber/oops, `diagnose`, `errorfamilytest`). Root module, Go 1.26+, no third
party deps; the six interfaces (`Coded`, `Classified`, `Contextual`,
`Retryable`, `ExitCoder`, `HTTPStatuser`) are the contract, the `Error`
struct a reference implementation.

Sources: `go-error-family/README.md:9` ("Share the protocol, not the
implementation"), `go-error-family/interfaces.go:16-62`, constructors at
`go-error-family/constructors.go:57-178`.

## 2. Usage census in go-cqrs-lite (evidence)

The deepest adoption of the four G1 libraries:

- **96 go.mod files** require `go-error-family v0.12.0` (55 direct, 39
  indirect, 2 explicit) — every production module and every test module.
- **445 Go files** import it (2026-11-22 census,
  `rg -l 'larsartmann/go-error-family' --type go | wc -l`).
- Symbol census (non-unique occurrences): `WrapInfrastructure` 387,
  `NewRejection` 367, `Rejection` 171, `Classify` 162, `WrapCorruption` 151,
  `WrapRejection` 136, `Wrapf` 127, `Newf` 98, `Wrap` 60, `Transient` 60,
  `NewInfrastructure` 48, `WrapTransient` 45, `NewConflict` 41, `IsRetryable`
  29, `WrapConflict` 26, plus all six families' constructors and wraps.
- The repo **builds its own tooling on top**: `scripts/check-error-taxonomy.sh:1-9`
  extracts every `errorfamily.(New|Wrap)<Family>("code", …)` call site and
  diffs it against `docs/error-taxonomy.md` in both directions (a CI drift
  gate, flake.nix:1433-1436); `cmd/cqrs-lint` rule D006
  (`cmd/cqrs-lint/v4/pkg/rules/consistency/d006.go:118`) requires the
  errorfamily constructors over ad-hoc `fmt.Errorf`.
- Boundary use: `example/taskmanager/http.go:37` mounts
  `errorfamily.HTTPHandler` for the API surface; `commandlifecycle/v4/recorder.go:146`
  records `errorfamily.Code(err)` on every command outcome.

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Six-family constructors and wraps (`New*`/`Wrap*`/`*f`) | **Full** | 445 files; e.g. `watermill/v4/errors.go:25`, `queue/v4/errors.go:43` |
| 2 | Classification core (`Classify`, `IsRetryable`, `Family` constants) | **Full** | 162 `Classify`, 29 `IsRetryable`; drives retry at `event/v4/middleware.go:86` |
| 3 | Stable codes (`Coded`, `Code(err)`) | **Full** | `commandlifecycle/v4/recorder.go:146`, `metaengine/v4/adttest/value_encode.go:53`, taxonomy gate keys off them |
| 4 | Contextual errors (`.WithContext`) | Partial | 5 non-test call sites repo-wide; context fields are mostly minted via log attributes instead |
| 5 | Third-party sentinel/classifier registration | Partial | `storage/v4/sql/classify_init.go:17-23` (stdlib + SQLite/PG/MySQL), `storage/pebble/v4/classify_init.go:14`; but `metaengine/duckdbengine/v4/typed_wal.go:187` and `metaengine/v4/claimkit/dialect.go:150` still classify by `strings.Contains(err.Error(), …)` |
| 6 | Scoped registries (`NewRegistry`, `Clone`) | Missed | only `DefaultRegistry` is referenced (2 sites); tests mutate global state |
| 7 | Multi-error worst-family (`errors.Join` + `Classify`) | Partial | semantics inherited for free (`system/v4/lifecycle.go:114` joins shutdown errors), never asserted or relied on deliberately |
| 8 | Exit codes (`ExitCode`, `ExitCoder`, `WithExitCode`) | Missed | exercised only by a taxonomy test (`event/v4/errors_taxonomy_test.go:292-299`); no CLI in the repo exits through it |
| 9 | HTTP boundary (`HTTPStatus`, `HTTPHandler`) | Partial | the example app uses it (`example/taskmanager/http.go:37`); library modules correctly stay transport-agnostic |
| 10 | CLI boundary (`HandleError*`, `RegisterTemplate`, `TemplateForCode`) | Missed | `HandleErrorDetailed` appears only in `event/v4/errors_taxonomy_test.go:306`; `cmd/cqrs-lint`, `cmd/cqrs-gen`, `cmd/doc-check` hand-roll flag parsing, printing, and `os.Exit` |
| 11 | `LogError` / `LogErrorContext` (slog) | Missed | 0 call sites; every module hand-rolls `logger.Warn/Error` with ad-hoc attribute sets |
| 12 | `RetryPolicy()` advisory defaults | Missed | 0 call sites; `middleware/v4` defines its own `RetryConfig` (`middleware/v4/middleware.go:31`) and `go-retry.FromPolicy` (built for exactly this) is unused |
| 13 | `WrapOnce` / `WrapOncef` (idempotent boundary wrap) | Missed | 0 call sites; API boundaries re-wrap defensively |
| 14 | `errorfamilytest` assertion kit | Missed | 0 imports; 98 hand-rolled `errorfamily.Classify`/`.Family()` assertions across test files (e.g. `integration/v4/event/classify_test.go`, 12 in one file) |
| 15 | `bridge` (samber/oops interop) | N/A | library doctrine: lib modules never import an app-enrichment dep; fleet apps may adopt it themselves |
| 16 | `diagnose` rules (fs/network/postgres/git) | Missed | 0 imports; the pg-backed story (sqlstore, pgengine, integration) is exactly the audience `DefaultRunner` targets |
| 17 | `agent` submodule | N/A | no runtime AI-agent error surface in this library; nothing to consume |

Score derivation: rows N=17, N/A rows 15 and 17 → **A=15**. Full=3 (rows 1,
2, 3), Partial=4 (rows 4, 5, 7, 9 → 2.0), Missed rows 6, 8, 10, 11, 12, 13,
14, 16 → 0. Score = round(100 × 5.0 / 15) = **33% → low band**.

## 4. Gap analysis

1. **Test assertions are the biggest quiet tax.** 98 hand-rolled
   family/code assertions vs. `errorfamilytest.AssertFamily/AssertCode/…`
   (`go-error-family/README.md:108`), which ship failure messages that name
   the family, the code, and the expectation. Typical hand-rolled shape
   (`event/v4/errors_taxonomy_test.go`, `integration/v4/event/classify_test.go`):

   ```go
   // skip-validate
   if got := errorfamily.Classify(err); got != errorfamily.Conflict {
       t.Errorf("family = %v, want Conflict", got)
   }
   // one-line replacement:
   // errorfamilytest.AssertFamily(t, err, errorfamily.Conflict)
   ```

   The kit is in the root module (no new require for the 55 direct modules;
   test-only anyway, outside dep budgets).

2. **Classification still leaks to string matching at two engine seams.**
   `metaengine/duckdbengine/v4/typed_wal.go:187` and
   `metaengine/v4/claimkit/dialect.go:150` detect duplicate/constraint errors via
   `strings.Contains(err.Error(), …)` — the exact anti-pattern the taxonomy
   doc's preamble ("without string matching or type assertions",
   `docs/error-taxonomy.md:7`) rules out. Registering a classifier the way
   `storage/v4/sql/classify_init.go:23` already does is the in-repo
   precedent:

   ```go
   // skip-validate
   errorfamily.RegisterClassifiers(classifyDuckDBConstraint)
   ```

3. **`RetryPolicy` is the missing link between two G1 libraries.** The
   library mints advisory retry defaults per family; go-retry ships
   `FromPolicy(errorfamily.RetryPolicy)` to convert them; the consumer's
   `middleware.RetryConfig` re-declares the same four knobs by hand
   (`middleware/v4/middleware.go:31`). One conversion would let a single
   policy source drive both classification and retry.

4. **CLIs ignore the CLI surface.** The repo ships five command tools that
   all fail the same way `HandleError` exists to prevent: raw
   `fmt.Fprintln(os.Stderr, err)` + `os.Exit(1)`. `ExitCode`'s sysexits
   mapping (Transient→75, Infrastructure→69) is precisely what script
   wrappers around `cqrs-lint`/`cqrs-gen` want.

## 5. Version currency

- Pinned **v0.12.0 in all 96 go.mod files** — identical to the latest local
  tag v0.12.0 (2026-10-14). **ON LATEST**, no drift.
- `go-error-family/CHANGELOG.md` [Unreleased] is CI-lint-only (golangci
  v2.15 wrapcheck exemption); nothing consumer-visible pending.
- One stale doc: `docs/error-taxonomy.md:3` still says the library "uses
  go-error-family **v0.10.0**" — two minors behind the actual v0.12.0 pin.
  Cosmetic, but the taxonomy doc is the gate's reference text.

## 6. The fleet argument

The owner's lens: "if every project re-implements/reintegrates everything I
will only fix bugs all day long." go-error-family is the one library where
this repo already behaves like a fleet citizen: 96 module pins, a CI drift
gate over the error taxonomy, and a lint rule that forces the constructor
style, so every bug fixed in classification semantics reaches the whole
fleet at the next bump. The remaining misses are the places where fleet apps
will re-integrate by hand instead: 98 bespoke test assertions, five CLIs
without exit-code contracts, two engines classifying by substring. Each is a
small fix here that never has to be fixed again anywhere.

## 7. Ranked opportunities

1. **Adopt `errorfamilytest` in the five heaviest asserting packages**
   (high, mechanical): `event/v4`, `integration/v4/event`, `command/v4`,
   `query/v4`, `benchkit/v4` — deletes dozens of hand-rolled assertions and
   improves failure output fleet-wide.
2. **Replace substring classification in duckdbengine and claimkit**
   (high, small): two `RegisterClassifier` registrations bring the last
   engines inside the taxonomy gate.
3. **Wire `RetryPolicy` → `middleware.RetryConfig`** (medium): one
   `FromPolicy`-shaped conversion in `middleware/v4` single-sources retry
   defaults with family semantics.
4. **Route the CLIs' fatal path through `HandleError`** (medium): five
   `cmd/` tools gain What/Why/Fix messages and correct sysexits for free.
5. **Fix `docs/error-taxonomy.md:3` v0.10.0 → v0.12.0** (trivial).

## 8. Reconsider when

- A fleet app standardizes on samber/oops (then `bridge` flips from N/A to
  the documented integration point for that app, not for this library).
- An AI-assisted operator surface lands in the fleet (`agent` submodule).
- `diagnose` gains rules this repo's operators would run against real PG
  incidents (today it duplicates nothing here, so adoption stays optional).
- The v5 core cut reconsiders error context: `.WithContext` at 5 call sites
  is either grown deliberately or documented as log-attributes-only.

## 9. How to verify

```
cd /home/lars/projects/go-cqrs-lite
rg -l 'larsartmann/go-error-family' --type go | wc -l          # 445
rg 'go-error-family v' -g go.mod | wc -l                        # 96, all v0.12.0
rg -o 'errorfamily\.[A-Za-z0-9_]+' --type go --no-filename | sort | uniq -c | sort -rn | head -12
rg -n 'errorfamilytest|WrapOnce|LogError|RetryPolicy|TemplateForCode' --type go   # 0 prod hits
rg -n 'strings\.Contains\(err\.Error\(\)' --type go -g '!*_test.go'
sed -n '17,23p' storage/v4/sql/classify_init.go
sed -n '186,188p' metaengine/duckdbengine/v4/typed_wal.go
sed -n '146,148p' commandlifecycle/v4/recorder.go
sed -n '3p' docs/error-taxonomy.md
head -9 scripts/check-error-taxonomy.sh
git -C /home/lars/projects/go-error-family log -1 --format='%H'
git -C /home/lars/projects/go-error-family tag --list | sort -V | tail -1   # v0.12.0
```

Sibling reports in this directory: `go-codec.md`, `go-retry.md`,
`go-idempotency.md` (the other three G1 core libraries).
