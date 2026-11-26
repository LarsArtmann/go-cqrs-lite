# ADR-0168: cqrs-gen derives message wiring for the composition root

- Status: Accepted (2026-11-26)
- Date: 2026-11-26

## Context

`cmd/cqrs-gen` shipped in 2026-07 as "typed handler registration from annotated structs" and
then stopped being true. Two defects made it a ghost: the documented `cqrs-gen ./...` invocation
was rejected by the CLI's argument validator, and after that fix the default `.` scan still found
nothing, because the directory walk pruned the root under its hidden-directory rule. The
2026-11-26 fleet census (63 direct consumers of this library) found no project using it.

Meanwhile `cqrs-lint` kept teaching it. B004, B009 and B010 told every consumer to "consider
cqrs-gen" for constructors, emit functions and catalog lists, and six archived feedback rounds
answer the same way: the tool does not appear to exist, or it is not adopted. A lint suggestion
that cannot be followed is worse than no suggestion.

Three facts shaped the redesign:

- **The seam moved.** ADR-0123 makes `system.New` the only composition root for new applications.
  The old generator targeted bare `command.Dispatcher` / `query.Dispatcher` registration, the
  seam that path retires. In the census the fleet hand-writes `system.RegisterCommand` at 349
  call sites, 29 repositories clone an `Op` wrapper to do it, and every command construction is
  the same two steps (`command.New`, then a struct literal that embeds the envelope).
- **Code generation is complementary.** `docs/design/v5-consumer-api.md` (option E) rejects
  generation as the primary path, because placement depends on engines known only at startup.
  ADR-0151 parks fold code generation behind an evidence gate. Neither decision changes.
- **A marked payload could not be documented correctly.** A branded ID rendered as an object in
  AsyncAPI and OpenAPI, so the examples kept payload identifiers as plain strings.

## Decision

1. **Scope: message wiring only.** For a struct carrying a marker (`//cqrs:<kind> <wire name>`
   comment, or a `cqrs:"<kind>:<wire name>"` struct tag), `cqrs-gen` derives exactly what the
   marker implies and nothing else:
   - the wire-name constant, unless the package already states that name;
   - a typed constructor: `NewX` for a command or query that embeds the library envelope,
     `NewXEvent` for an event;
   - the composition-root registration, `RegisterXCmd` and `RegisterX`;
   - with `--handlers`, a typed event subscription; with `--catalog`, the package's catalog
     declarations as `CatalogMessages`.

   It never generates folds, projections, read models, or placement. The Evolution stays the
   declaration (ADR-0151).

2. **The default target is the composition root.** `RegisterXCmd(sys, streamType, decide)` wraps
   `system.RegisterCommand` and `system.Execute`; no hand-written `Op` wrapper remains.
   `--target=dispatcher` keeps the legacy `RegisterTyped` shape for applications that have not
   moved.

3. **A constructor is derived only where one is mechanical.** It requires the envelope embed. A
   package that already declares `NewX` keeps its own constructor, because that one may validate;
   the generator never adds a second one and never fails on the name.

4. **Options are never dropped.** Every generated constructor takes the library's options and
   forwards them (`command.Option`, `query.Option`, `event.Option`). An event constructor also
   stamps the payload's schema version through `event.WithSchemaVersion`: the marker's `vN` part,
   or version 1 when absent, so upcasting can always be added later and a payload change is one
   marker edit. A bumped version is carried into the catalog declaration as the message version.

5. **Generated output is proven, not assumed.** A test compiles the output against the workspace
   modules together with call sites for every signature. A second test keeps the README's output
   fence equal to the generator's real output. `example/goal-shaped-app` runs on the generated
   code, and its freshness is pinned twice: by a unit test and by the `cqrs-gen-output` leg of
   the pre-push sweep (`nix run .#check-generated-artifacts`).

6. **Every lint suggestion names a remedy that exists.** B004, B009 and B010 now say which marker
   to add and which declaration follows. B009 and B010 skip files carrying the generated-code
   header, because a finding inside generated output cannot be fixed in place. B032 does not judge
   a constructor that forwards an option list: tracing is decided where that list is built.

7. **Examples brand identifiers inside one context.** A payload, read model and query that stay
   inside one context carry the branded identity end to end (`id.StreamID`); on the wire it is a
   bare string and the catalog schema renders it as one. A payload that crosses a context or
   process boundary stays a plain-string contract and each side brands at its own boundary.
   `example/goal-shaped-app` and `example/metaengine-quickstart` follow the first rule,
   `example/mesh-demo` and `example/taskmanager` the second.

## Consequences

- Adopting the generator removes the registration wrapper, the two-step command construction and
  the hand-listed catalog declarations, and pairs each wire name with exactly one payload type at
  compile time.
- The generator's output is a consumed surface without a tag. Changing it means regenerating
  `example/goal-shaped-app/cqrs_gen.go` and the README fence in the same edit; both pin tests
  fail loudly otherwise.
- A `cqrs-lint` run over a package that uses the generator is clean on the generated file. The
  remaining findings point at hand-written code.
- Fold code generation stays parked. Its trigger is unchanged (ADR-0151): measured parity against
  hand-written folds, or a named consumer asking for it.

## Evidence

- Status report of the rework: `docs/status/2026-11-26_05-50_cqrs-gen-rework-composition-root-codegen-uncommitted.md`
  (fleet census, rule-by-rule audit, the three owner questions and their resolution).
- Consumer feedback on the old suggestions: `docs/feedback/archived/` (2026-07-17 through
  2026-08-05) and `docs/feedback/reviewed/archived/`.
- Proofs named in decision 5: `cmd/cqrs-gen/v4/compile_test.go`, `cmd/cqrs-gen/v4/readme_test.go`,
  `scripts/check-generated-artifacts.sh`.
