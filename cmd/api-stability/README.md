# api-stability — API Surface Checker

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/api-stability/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/api-stability/v4)

CLI tool that walks core library packages, collects their exported symbols (types, funcs, methods, vars, consts) via Go AST parsing, and compares them against a golden file to detect breaking or unexpected API surface changes.

## Install

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/api-stability/v4@latest
```

## Usage

```bash
# Check API surface against golden file (fails if changed)
api-stability

# Update the golden file with current API surface
api-stability -update
```

The golden file lives at `docs/api_surface.txt`. It enumerates every exported symbol across 17 core packages: `command`, `event`, `query`, `decider`, `id`, `dispatcher`, `memory`, `catalog`, `middleware`, `signing`, `encryption`, `projection`, `listing`, `otel`, `storage`, `event/v4/eventtest`, `watermill`.

## How It Works

1. Parses all `.go` files (non-test) in the scanned packages via `go/ast`
2. Collects exported types, functions, methods, variables, and constants
3. Sorts and formats them into a canonical text representation
4. Diffs against the golden file — any addition, removal, or rename is flagged

This catches accidental breaking changes before they ship: removed exports, renamed methods, changed signatures.

## Dependencies

Zero external dependencies — uses only the Go standard library (`go/ast`, `go/parser`).

## Related Modules

- All core library modules are scanned by this tool
- Golden file: [`docs/api_surface.txt`](../../docs/api_surface.txt)
