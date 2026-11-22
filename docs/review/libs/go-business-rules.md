# go-business-rules — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/LarsArtmann/go-business-rules/v2` @ 53473a3 (latest local tag v2.2.1) · Consumer go-cqrs-lite @ feec9bc74 · Group G4

**Verdict:** NOT-ADOPTED-JUSTIFIED
**Score:** 0/1 capabilities fully leveraged = 0% (low)

Zero requires — and the zero is doctrinally load-bearing. go-cqrs-lite's
center of gravity is the pure-function decider: business rules ARE the
`Decide`/`Apply` functions, compiled and type-checked, not data evaluated
by an external engine (decider/v4/decider.go:13-19). A severity-aware
rule-evaluation library competes with that, and the places it might
complement it do not want it: event admission is legitimately binary (an
event either conforms to its schema or is rejected —
schema/v4/validator.go:159), lint findings already have a severity model
via go-finding (the very type go-business-rules re-exports,
go-business-rules/README.md:137), and the library's own cqrs-lite adapter
depends on the consumer's modules, inverting the arrow. Exactly one surface
survives scrutiny as a genuine (if small) gap: `catalog/v4` hand-rolls a
322-line severity-blind rule engine around `Validate() []Violation`, where
named rules with error/warning grades would let the docserver distinguish
"catalog is wrong" from "catalog is incomplete."

## 1. What the library is

"Severity-aware validation for Go — because not all validation failures are
equal" (go-business-rules/README.md:5-7). Standard validators return
pass/fail; businessrules returns the _degree_ of failure: `SeverityInfo` /
`SeverityWarning` / `SeverityError` / `SeverityCritical`, re-exported from
`go-finding` so the type is fleet-interchangeable (README.md:134-140). Core
API: an immutable `Rule` interface (`Name`/`Check`/`Severity`/`Message`,
README.md:149-160), a validator builder producing `ValidationResultError`
with filtering by severity (`Errors()`, `Warnings()`, `BySeverity()`,
`Merge()`), ~60 pre-built rules (numeric/string/collection/time/network/
format, README.md:285-385), composites (`All`/`Any`/`When`/`Not`/`Or`/
`Xor`), observable validation runs (`WithListener` on `RuleEvaluated`/
`ValidationCompleted` events, and `Stream` for concurrent I/O-bound rules
with `ContextRule` + `WithConcurrency`), and JSON marshaling. One runtime
dependency (`go-finding`) (README.md:400-402). Nested modules:
`adapters/cqrslite` — publishes validation events onto a go-cqrs-lite
event bus as durable, CBOR-safe domain events (README.md:396) — and
`listeners/otel`. Latest tag **v2.2.1** (2026-11-04, tooling-only churn:
"fix(ci): align tool pins", `git log v2.2.0..v2.2.1`); README still
advertises v2.2.0 as latest.

## 2. Usage census in go-cqrs-lite (evidence)

- **Zero requires**: `rg -n 'go-business-rules' -g go.mod` across all 104
  go.mod files returns nothing — direct, indirect, example, tool, or test.
- **Zero imports**: no `.go` file in the consumer references the package.
- The overlap candidates, inspected:
  - `schema/v4/validator.go:85` `NewValidator` / `:111` `RegisterType` /
    `:118` `RegisterTypeWithValidator` / `:159` `Validate` — the
    consumer's structural event-payload validator (decode-into-type +
    optional per-type semantic hook), returning a binary `Rejection`.
  - `catalog/v4/validate.go:25` `(*Catalog).Validate() []Violation` with
    `Violation{Path, Message}` (validate.go:6-9) — a hand-rolled,
    severity-blind catalog linter (322 lines).
  - `cmd/cqrs-lint/v4` — severity-aware _static_ findings via
    `github.com/larsartmann/go-finding` (imported at
    `cmd/cqrs-lint/v4/output.go:23`; severity overrides at
    `cmd/cqrs-lint/v4/pkg/analyzer/project_config.go:173`).
- The library's cqrs-lite integration points at the consumer, not away
  from it: `go-business-rules/adapters/cqrslite/go.mod:5-9` requires
  `go-cqrs-lite/event/v4 v4.15.2` and `go-cqrs-lite/id/v4 v4.7.4`.

## 3. Capability inventory

| # | Capability                                                  | Grade  | Evidence / reason                                                                                                                                                                                                                                                                                                              |
| - | ----------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | Severity-aware domain rule evaluation (core Rule/Validator) | N/A    | competes with the doctrine: deciders are pure functions, rules are code (`decider/v4/decider.go:13-19` "Apply must be a pure function"); externalizing them into Rule data would move business logic out of the typed, property-tested core                                                                                    |
| 2 | Event-payload validation (vs `schema.Validator`)            | N/A    | admission to an append-only journal is binary: `Validate` returns a `Rejection` or nil (`schema/v4/validator.go:159-200`); "warning" severity at append time is incoherent, and the semantic-hook need is already served by `RegisterTypeWithValidator` (`schema/v4/validator.go:118-124`)                                     |
| 3 | Catalog config validation                                   | Missed | `catalog/v4/validate.go:25-74` hand-rolls `Validate() []Violation` over services/domains/channels/entities/data products — structurally the library's Rule + ValidationResult shape (322 lines) — with no severity, so docserver consumers cannot tell "duplicate operation path" (error) from "missing description" (warning) |
| 4 | Severity vocabulary for findings                            | N/A    | cqrs-lint already uses `go-finding` directly, whose `Severity` is what businessrules re-exports (go-business-rules/README.md:137) — the shared vocabulary is in the fleet without the rule engine                                                                                                                              |
| 5 | Validation events → OTel                                    | N/A    | observability doctrine routes through cqrsotel (`otel/v4`); no second metrics bridge wanted                                                                                                                                                                                                                                    |
| 6 | `adapters/cqrslite` event-bus publishing                    | N/A    | dependency direction inverted: the adapter requires the consumer's `event/v4`/`id/v4` (go-business-rules/adapters/cqrslite/go.mod:5-9); adopting it into the bus it publishes onto is circular — it exists for fleet _apps_                                                                                                    |
| 7 | Concurrent I/O-bound rules (`Stream`, `ContextRule`)        | N/A    | no I/O-bound validation in the consumer's runtime; deciders must stay pure                                                                                                                                                                                                                                                     |

Score derivation: rows N=7, N/A rows 1–2 and 4–7 → **A=1**. Full=0,
Partial=0, Missed=1 (0). Score = round(100 × 0 / 1) = **0% → low band**.

## 4. Gap analysis

1. **The doctrine conflict is real and decisive — for the runtime.**
   go-cqrs-lite's product IS the doctrine "business rules as pure functions
   with event-sourced truth" (`decider/v4/decider.go:13-19`). A rule engine
   (rules as data, evaluated generically) is the architectural opposite:
   it trades compile-time exhaustiveness and the property/BDD test suites
   for runtime flexibility the library never promised. Middleware is no
   escape: validation middleware in this repo gates journal admission
   (`bus.Use(middleware.EventValidation(v.Validate))`,
   `schema/v4/validator.go:25-27`), and admission is binary by
   construction. Verdict: the library's center of gravity has no lawful
   home in go-cqrs-lite.
2. **One honest overlap, two honest counterweights.** `catalog/v4/
   validate.go` is a hand-rolled cousin of the library (named checks over a
   document, aggregated result) and severity would add real value at the
   docserver edge. Counterweight one: `catalog/v4/go.mod` carries a strict
   production dep budget (10 direct requires), and this would add an
   eleventh for ~322 lines that already work and are golden-tested.
   Counterweight two: the win is UX (error vs warning badges), which could
   also be had by adding a `Severity` field to the existing `Violation`
   struct — zero new deps. The library adoption is defensible only if the
   catalog rule set is expected to grow substantially or fleet apps want
   identical catalog-check semantics.
3. **The dependency arrow is inverted twice.** Not only does the consumer
   not need the library; the library needs the consumer — its flagship
   adapter publishes validation events onto go-cqrs-lite buses
   (go-business-rules/README.md:396,
   go-business-rules/adapters/cqrslite/go.mod:5-9). The
   correct fleet pattern is exactly what exists: fleet apps that want
   severity-graded edge validation (command admission at the HTTP
   boundary, where "warn but accept" IS coherent) adopt businessrules and,
   if they are event-sourced, use the adapter to journal those verdicts.

## 5. Version currency

- Pinned: **nothing**; latest local tag **v2.2.1** (2026-11-04), HEAD
  53473a3 (2026-11-19) — moot for this consumer, relevant for fleet apps.
- v2.2.1 over v2.2.0 (2026-09-17) is tooling-only: CI tool-pin alignment,
  dependency churn, lint-config convergence — no API delta
  (`git log v2.2.0..v2.2.1 --oneline`); the README's "Latest release
  v2.2.0" table (go-business-rules/README.md:27) and the adapter's own
  `go-business-rules/adapters/cqrslite/go.mod:6` pin of v2.2.0 lag the tag by one patch.
- A build-flag footnote that matters fleet-wide: the README claims
  `GOEXPERIMENT=jsonv2` is required (go-business-rules/README.md:19,24,418),
  but json/v2 graduated in go1.27.2 and the consumer removed the experiment
  from all tooling on 2026-09-19 (`AGENTS.md:145`). On the consumer's
  go1.27.3 toolchain the flag is a no-op — stale doc, not a real blocker.
  The library's `go 1.27` minor-form directive vs the consumer's uniform
  `go 1.27.2` patch-form is likewise cosmetic today.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only
fix bugs all day long." For most utilities that argument pushes toward
adoption; here it pushes toward _placement_. Severity-aware validation
genuinely should not be re-implemented per fleet project — at application
edges, where an API can warn-and-accept. go-cqrs-lite's contribution to
that fleet story is to stay out of the way: keep deciders pure, keep
journal admission binary, keep the severity vocabulary shared through
go-finding (already the case), and let fleet apps adopt businessrules at
their boundaries with the cqrslite adapter journaling verdicts onto
go-cqrs-lite buses. The one in-repo duplication (catalog/v4/validate.go)
is small enough that the cheaper dedup is a `Severity` field on
`Violation`, not a new dependency — but the library's `adapters/cqrslite`
should be listed in fleet docs as the blessed way for apps to record
validation runs as events, or the next fleet app will hand-roll that
bridge.

## 7. Ranked opportunities

1. **Decide catalog/v4/validate.go's future explicitly** (medium): either
   add a `Severity` field to `Violation` (`catalog/v4/validate.go:6-9`) and
   render warnings-vs-errors in the docserver — the zero-dependency path —
   or adopt the library if the rule set is expected to grow. Record the
   choice where `nix run .#check-arch` budgets live, so the next audit
   doesn't re-litigate.
2. **Document the app-edge pattern** (low): one paragraph in
   docs/ (validation guidance near SCHEMA-COMPATIBILITY.md) stating that
   runtime business-rule evaluation belongs to apps via
   go-business-rules/v2 + `adapters/cqrslite`, not to library modules —
   turning the doctrine into discoverable guidance.
3. **Fleet-doc the severity vocabulary** (low): note that
   `businessrules.Severity` is `go-finding.Severity`
   (go-business-rules/README.md:137) so apps can share severity semantics
   with cqrs-lint findings without either depending on the other.

## 8. Reconsider when

- The docserver wants graded catalog checks (errors block export, warnings
  advisory) and the Severity-field approach has already been rejected as
  too thin — then row 3 flips to a real adoption in `catalog/v4`.
- A fleet app (or the taskmanager example) adds an HTTP command edge where
  warn-but-accept is product behavior — adopt there, with
  `adapters/cqrslite` journaling `ValidationCompleted` as events.
- The library grows a pure, dependency-free "rules as code" assertion
  helper (no engine, no events) that could replace hand-rolled test
  helpers in the consumer's BDD suites — that would be an upstream-legal,
  test-only surface.
- go-cqrs-lite ever introduces advisory (non-blocking) findings in its own
  runtime API — today none exist.

## 9. How to verify

```
rg -n 'go-business-rules' -g go.mod /home/lars/projects/go-cqrs-lite    # no output
rg -rn 'businessrules' --type go /home/lars/projects/go-cqrs-lite        # no output
git -C /home/lars/projects/go-business-rules log -1 --format='%H' && git -C /home/lars/projects/go-business-rules tag --list | sort -V | tail -3
git -C /home/lars/projects/go-business-rules log v2.2.0..v2.2.1 --oneline
sed -n '13,19p' /home/lars/projects/go-cqrs-lite/decider/v4/decider.go
sed -n '17,27p;85,124p;159,175p' /home/lars/projects/go-cqrs-lite/schema/v4/validator.go
sed -n '1,74p' /home/lars/projects/go-cqrs-lite/catalog/v4/validate.go
sed -n '5,9p' /home/lars/projects/go-business-rules/adapters/cqrslite/go.mod
sed -n '134,160p;390,402p' /home/lars/projects/go-business-rules/README.md
sed -n '145p' /home/lars/projects/go-cqrs-lite/AGENTS.md
```

Related reports in this directory: `go-error-family.md` (the rejection
taxonomy `schema.Validator` emits), `go-output.md`/`cmdguard.md` (the
tooling cluster where the fleet's severity vocabulary already lives via
go-finding).
