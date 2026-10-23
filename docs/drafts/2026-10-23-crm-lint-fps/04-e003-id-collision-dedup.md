# E003 pins every finding at go.mod:1:1 — CLI dedup collapses them, the BuildFlow provider doesn't

## Symptom

Same repo, same `.cqrs-lint.json`, same analyzer code: the CLI reports **116 findings (E003 ×1)**, the BuildFlow toolspec provider reports **120 (E003 ×5)**. Verified end-to-end today on LarsArtmann/crm @ 34f26a6 with `cqrs-lint 4.16.1-0.20261018201801-5f9b57ba413c` and a fresh CLI build from ec36eb8d2b — every other rule count identical, only E003 diverges.

Two consequences beyond the count split:

- **The CLI's surviving finding is nondeterministic.** E003 iterates a `map[string]map[string]bool` of packages; the five findings share one ID, so `collectFindings` keeps whichever the map yields first — the count is stable (1), the *content* varies run to run.
- Any other ID-keyed consumer (SARIT dedup, result caches, `--fail-on` gates) silently collapses project-level findings the same way.

## Root cause (source)

`cmd/cqrs-lint/pkg/rules/architecture/e003_e007.go:81` (@ ec36eb8d2b) gives every package-level finding the same position:

```go
// skip-validate
f, err := findingTemplate.Builder(
	"E003",
	fmt.Sprintf("Package %s mixes %d CQRS concerns …", pkg, len(constructs), …),
	finding.SeverityWarning,
	finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
).
```

The finding ID is `tool:rule:file:line:col` — so all five are `cqrs-lint:E003:<root>/go.mod:1:1`.

The CLI then dedups by ID (`cmd/cqrs-lint/filters.go:131`, `collectFindings` → `seen[f.ID]`), collapsing 5→1. The toolspec path (`pkg/toolspec/toolspec.go:72`, `detect`) appends detector output directly and never runs `collectFindings` — so the BuildFlow provider reports all five. Same code, two answers.

I confirmed the registry itself is identical on both paths with a scratch program calling `analyzer.BuildContext` directly: commands 25, events 25, folds 5 — all five domain packages legitimately at 3 constructs. This is purely a reporting-layer divergence, not a detection difference.

## Fix

Pick one source of truth:

- Make E003 positions unique per package (the package's first contributing construct's `file:line`, or `go.mod:<n>:1` with a per-package ordinal) — then ID-dedup is harmless and CLI/provider agree at 5; **or**
- keep `go.mod:1:1` but include the package name in the finding ID, and run the same dedup in the toolspec path.

The first option is better: E003's message already names the package, and a real position makes `--show-suppressed`, SARIF inline annotations, and editor jumps work.

## To verify

- Repo with ≥2 packages mixing command+event+fold: CLI and toolspec report the same N≥2 findings, same package set, run over run.
- Real consumer: crm @ 34f26a6 — both surfaces report E003 ×5 (or a stable adjudicated count) with identical IDs.

💘 Generated with Crush
