# Contributing to cqrs-lint

## Adding a New Rule

### 1. Choose a Category and ID

Rules are organized into 6 categories:

| Category     | Prefix | Example | Description                             |
| ------------ | ------ | ------- | --------------------------------------- |
| correctness  | C      | C001    | Bugs that cause incorrect behavior      |
| api          | A      | A001    | Misuse of the go-cqrs-lite API          |
| boilerplate  | B      | B001    | Repetitive code that could be generated |
| architecture | E      | E001    | Structural violations (layer deps)      |
| consistency  | D      | D001    | Inconsistent conventions                |
| security     | S      | S001    | Security vulnerabilities                |

Find the next available ID in your category by checking `rules.ListRules()`.

### 2. Register the Rule Metadata

Add a `RuleInfo` entry in the appropriate catalog file (`pkg/rules/catalog.go` or `catalog_extra.go` or `catalog_extra2.go`):

```go
{
    ID:          "C013",
    Name:        "your-rule-name",
    Category:    "correctness",
    Severity:    "error",    // info, warning, error, critical
    Confidence:  "high",     // low, medium, high
    Description: "What this rule detects",
    AutoFix:     false,
},
```

### 3. Implement the Detector

Create a file in the appropriate category package. Two patterns:

**Registry-driven** (uses pre-scanned CQRS patterns):

```go
func NewC013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
    return finding.NamedDetectorFunc(
        "C013-your-rule-name",
        func(_ context.Context) ([]finding.Finding, error) {
            var findings []finding.Finding
            for _, cmd := range ctx.Registry.Commands {
                // detection logic
                f, err := finding.NewBuilder("C013", toolName, "message",
                    finding.SeverityError,
                    finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
                ).
                    WithCategory(finding.CategoryCorrectness).
                    WithConfidence(finding.ConfidenceHigh).
                    WithSnippet(ctx.SourceLine(cmd.File, cmd.Pos.Line)).
                    WithSuggestion("How to fix").
                    Build()
                if err == nil {
                    findings = append(findings, f)
                }
            }
            return findings, nil
        },
    )
}
```

**AST-scanning** (directly inspects Go syntax):

```go
for _, gf := range ctx.GoFiles {
    if gf.IsTest { continue }
    ast.Inspect(gf.AST, func(n ast.Node) bool {
        // pattern matching logic
        return true
    })
}
```

### 4. Register the Detector

Add your detector to `RegisterAll()` in `pkg/rules/register.go`.

### 5. Write Tests

Create a test file in the same package. Always include:

- A **positive test** that asserts the finding count > 0
- A **negative test** that asserts 0 findings on clean code
- Use `BuildContextFromSource` for AST-only tests
- Use `ctx.Packages` for import-graph tests (E001, E002, D003)

```go
func TestC013_DetectsIssue(t *testing.T) {
    ctx := analyzer.BuildContextFromSource(t, map[string]string{
        "example.go": `package main
// fixture code that triggers the rule
`,
    })
    findings := runDetector(t, correctness.NewC013Detector(ctx))
    assertRule(t, findings, "C013", 1)
}
```

### 6. Verify

```bash
cd cmd/cqrs-lint
GOWORK=off go build -tags "goexperiment.jsonv2" ./...
GOWORK=off go test -tags "goexperiment.jsonv2" ./... -count=1
```

## CI Constraints

- Max 350 lines per Go file (split proactively)
- All lint issues must be resolved (`nix run .#lint`)
- No panics in detector code — always return `([]finding.Finding, error)`
- Test files are skipped by all AST-scanning detectors (`gf.IsTest`)

## Detector Conventions

### Consult FeatureProfile, not private heuristics

Rules that depend on deployment context (server vs CLI, read-only vs commands,
soft-delete domain, tracing) MUST consult `ctx.FeatureProfile` instead of
re-deriving project context with private heuristic functions. This centralizes
"what kind of system is this?" in one place.

```go
// CORRECT — consult the centralized feature profile
if !ctx.FeatureProfile.HasServer {
    return nil, nil // local-only system, suppress
}

// WRONG — private heuristic that duplicates DetectFeatures
func isLocalOnly(ctx *AnalysisContext) bool { ... }
```

### Use SelectorFromExpr for all call matching

All detectors that match function calls MUST use `analyzer.SelectorFromExpr`
instead of direct `call.Fun.(*ast.SelectorExpr)` type assertions. Direct
assertions are blind to generic calls like `decider.WithSnapshotStore[State](store)`.

```go
// CORRECT — handles generic instantiation wrappers
sel, ok := analyzer.SelectorFromExpr(call.Fun)
if !ok { return true }

// WRONG — panics or silently skips generic calls
sel := call.Fun.(*ast.SelectorExpr)
```

## Architecture

```
cmd/cqrs-lint/
├── main.go          # CLI entry point, cmdguard setup
├── filters.go       # Severity/confidence/path filtering
├── health.go        # Health score computation
├── color.go         # Colored terminal output
├── commands.go      # Rules/version subcommands
├── init.go          # Init subcommand
├── doctor.go        # Doctor subcommand (feature profile detection)
├── pkg/
│   ├── analyzer/    # CQRS pattern scanning, registry, feature profile, test helpers
│   │   ├── feature_profile.go  # FeatureProfile types, ConfigFeatures, presets
│   │   ├── feature_detect.go   # DetectFeatures (centralized context detection)
│   │   ├── ast_helpers.go      # SelectorFromExpr, unwrapSelector (generics support)
│   │   └── ...
│   ├── rules/       # Rule catalog, registration, filtering
│   │   ├── correctness/   # C001-C012
│   │   ├── api/           # A001-A019
│   │   ├── boilerplate/   # B001-B015
│   │   ├── architecture/  # E001-E007
│   │   ├── consistency/   # D001-D005
│   │   └── security/      # S001-S003
│   ├── fix/         # Auto-fix provider
│   └── suppression/ # //cqrs-lint:ignore comment parser
```

## Release Process

Releasing cqrs-lint requires coordinating the Go version constant, the Nix
vendorHash, the Go module checksums, and the git tag. Follow this checklist:

### 1. Bump the version constant

Edit `main.go` and set `const version` to the new semver (e.g., `"4.4.0"`).
The version must match the next `cmd/cqrs-lint/vX.Y.Z` tag.

### 2. Sync Go module dependencies

```bash
cd cmd/cqrs-lint
GOWORK=off go mod tidy
```

### 3. Update the Nix vendorHash

```bash
nix build .#cqrs-lint
# If hash mismatch: copy the "got:" hash from the error
# Edit flake.nix line ~362: vendorHash = "sha256-NEW_HASH"
nix build .#cqrs-lint  # verify it builds
```

### 4. Run the full verification suite

```bash
nix run .#verify  # build + vet + test + race + lint + doc-check
```

### 5. Tag and verify

```bash
git tag -a cmd/cqrs-lint/v4.4.0 -m "cqrs-lint v4.4.0"
git push origin cmd/cqrs-lint/v4.4.0
```

### 6. Verify consumers can resolve

```bash
# From a temp project:
go get github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint@v4.4.0
```

### Tag conventions

- Tags use the full module path: `cmd/cqrs-lint/vX.Y.Z` (not `cqrs-lint/vX.Y.Z`)
- Tags must be monotonically increasing in BOTH semver AND commit ancestry
- Always use annotated tags (`git tag -a`), never lightweight tags
- Verify with: `git tag -l 'cmd/cqrs-lint/v*' | sort -V | tail -1`

### SystemNix distribution

The system-installed `cqrs-lint` binary comes from the SystemNix flake
(`~/projects/SystemNix`), which pins this repo as a `flake=false` input.
After a release:

```bash
cd ~/projects/SystemNix
nix flake update go-cqrs-lite
sudo nixos-rebuild switch --flake .#l
```

The `scripts/bump-cqrs-lint.sh` helper automates the vendorHash + go mod tidy

- nix build cycle from the repo root.
