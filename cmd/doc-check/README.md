# doc-check — Documentation Verifier

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v4)

CI tool that verifies Go import paths and qualified symbol references in Markdown docs actually exist in the codebase. Catches stale documentation before it ships.

## Install

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/doc-check@latest
```

## Usage

### Default (auto-discover)

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../AGENTS.md
```

When no arguments are given, doc-check auto-discovers:

- `SKILL.md`
- `AGENTS.md`
- `docs/DOMAIN_LANGUAGE.md`
- `.agents/skills/*/references/*.md`

### Explicit files

```bash
doc-check path/to/doc1.md path/to/doc2.md
```

Exit code is non-zero if any broken references are found.

## How It Works

1. Scans Markdown files for ` ```go ` fenced code blocks.
2. Extracts import paths matching `github.com/larsartmann/go-cqrs-lite/...` via regex.
3. Extracts qualified references (`pkgSymbol.Symbol` — lowercase-dot-uppercase pattern).
4. Builds an export index by parsing actual `.go` files with `go/ast`.
5. Verifies each reference exists in the export index.
6. Reports broken references with file and line number.

## What It Checks

| Reference Type   | Example                                        | Verified?                    |
| ---------------- | ---------------------------------------------- | ---------------------------- |
| Import path      | `github.com/larsartmann/go-cqrs-lite/event/v4` | Yes — directory exists       |
| Qualified symbol | `event.NewEvent`                               | Yes — exported symbol exists |
| Stdlib symbol    | `fmt.Println`                                  | No — skipped                 |
| External symbol  | `otel.Tracer`                                  | No — skipped                 |

- Strips `/v4` suffix from import paths to resolve directory locations.
- Walks up to `.git` to find the repo root (handles worktrees).
- Only checks `go-cqrs-lite` internal packages; external imports are skipped.

## CI Integration

```bash
# In CI:
cd cmd/doc-check && GOWORK=off go run . \
    ../../SKILL.md \
    ../../.agents/skills/go-cqrs-lite/references/*.md \
    ../../AGENTS.md
```

The AGENTS.md documents this as part of the doc maintenance workflow.

## Related Modules

- [**api-stability**](../api-stability/README.md) — API surface checker (complementary: checks exports vs golden file)
- [**cqrs-lint**](../cqrs-lint/README.md) — Domain-aware linter for go-cqrs-lite code
