# V007 `v5-removed-api-usage` — Demo

Release-notes-ready demonstration of the v5-migration detector
(`cmd/cqrs-lint/v4.9.0`). Captured 2026-09-06; updated 2026-09-08 with the
dot-import detection (F090), which ships in the next minor.

## The consumer situation

A project pins `storage/v4 v4.6.0` and uses the relational tier that
v5 removes (ADR-0123/0126). Nothing warns at `go build` — the API works
fine on v4 and dies at the major bump.

## What cqrs-lint reports

```text
$ cqrs-lint --path .

WARNING .../main.go:10:6 storage.NewRelationalStore is removed at v5 (ADR-0126) — replace with metaengine engines with layout planning
  [V007]
  Suggestion: Migrate off storage.NewRelationalStore before the v5 cut; see docs/adr (ADR-0126)
```

## Dot-imports are flagged too (new)

A dot-import hides every qualifier, so removed-API usage behind it was
invisible to V007. The F090 hardening flags the import itself:

```text
WARNING .../main.go:8:2 dot-import of go-cqrs-lite module storage/relational hides v5-removed-API usage from this linter — name the import
  [V007]
  Suggestion: Replace the dot-import of github.com/larsartmann/go-cqrs-lite/storage/relational/v4 with a named or default import so V007 can attribute removed-API usage
```

## Why it matters

- **Surfaced at v4, not at the v5 cut.** Every removed-API touchpoint is
  a named, located finding while there is still time to migrate.
- **Each finding cites its ADR** (0114 tombstones, 0123 unification,
  0126 store transforms) — the remediation doc is one click away via the
  SARIF help URI / `RULES.md#v007` anchor.
- **Coverage is contract-locked**: a drift meta-test fails CI if a
  `Deprecated:` marker in the repo has no V007 entry or vice versa, and
  the examples gate keeps every `example/` V007-silent.

## Try it

```sh
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4@latest
cqrs-lint --path .
```
