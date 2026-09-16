# doc-check — Documentation Verifier

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v4)

CI tool that verifies Go import paths, qualified symbol references, call arity, and navigation (TOC anchors, § cross-refs) in Markdown docs actually exist in the codebase. Catches stale documentation before it ships.

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

1. Scans Markdown files for `` ```go `` fenced code blocks.
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
| Call arity       | `system.New(ctx, cfg)` vs 3-param signature    | Yes — go/ast signature match |
| TOC anchor       | `[§2.1](#21-minimal-es)`                       | Yes — GitHub-exact slug      |
| § cross-ref      | `recipes §2.13`, bare `§3.8`                   | Yes — section number exists  |
| Stdlib symbol    | `fmt.Println`                                  | No — skipped                 |
| External symbol  | `otel.Tracer`                                  | No — skipped                 |

### Arity spot-check details

Every parseable `` ```go `` fence is compared call-shape-by-call-shape against the real
signatures (package-level exported functions only; methods and ambiguous package names
are skipped, so wrong-package hits are impossible by construction). Precision filters
keep intentional doc shapes green:

- `// Wrong` / `// Deprecated` / `// never` / `// don't` markers on or above the call —
  deliberately wrong examples are documentation, not lies.
- Comment-only argument lists (`f(/* ctx, cfg, opts */)`) — the comment stands for the args.
- Doc ellipsis abbreviations (`f(ctx, ...)`, `Cfg{...}`, `…`) count as ONE placeholder arg.
- Per-block opt-out: put `// doc-check:ignore-arity` anywhere in the fence.

### Navigation (anchor + §) scope

TOC anchors are checked in every scanned file with a GitHub-exact slugger (underscores
kept, punctuation stripped, inline-code content kept, heading links contribute their
text, `#slug-1` dedup suffixes honored). § cross-refs are a skill-docs convention and are
validated for the default scan set only (`SKILL.md`, `AGENTS.md`, skill `references/`,
the two DOMAIN_LANGUAGE docs). Known-good bare shapes: `ADR-NNNN §N` (ADR-relative),
`the former X §N` and `→ moved to` TOC bullets (historical pointers), and bare `§N` when
exactly one scanned doc has that section (ambiguous bare refs are flagged — name the doc).

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
