# GOWORK Mode Decision Table

> One lookup for the recurring foot-gun: every `go` command in this repo has a
> correct resolution mode and a correct working directory. Running a command in
> the wrong mode either false-fails ("directory prefix . does not contain
> modules listed in go.work") or false-greens (workspace resolves local
> siblings and masks a broken published pin).
>
> Extracted from AGENTS.md gotchas (2026-09-08 indexed-split) and made a table.

| Command class                                            | Run from            | GOWORK            | Why this mode                                                                                            |
| -------------------------------------------------------- | ------------------- | ----------------- | -------------------------------------------------------------------------------------------------------- |
| Per-module build / test / vet / lint                     | the module dir      | `off`             | Proves the module against its PUBLISHED pins — workspace mode masks pin breaks (command/v4.7.0 class).   |
| `nix run .#build` / `#test` / `#verify` / `#verify-fast` | repo root           | on                | The composed gates run workspace-wide by design; siblings resolve from local checkouts.                  |
| `nix run .#verify-ci`                                    | repo root           | `off`             | The CI-mirror app: per-module GOWORK=off build+test matrix.                                              |
| Protocol/benchmark baselines (docs/benchmarks/)          | repo root           | on                | Baseline numbers are only comparable with other workspace-mode runs; per-module runs skew ±.             |
| Alloc-count assertions touching dependency code          | module dir          | both              | Workspace sees sibling trees AHEAD of tags; pin upper-bound budgets, diagnose both ways before assuming. |
| `go get` / `go mod tidy` sweeps                          | the module dir      | `off`             | Workspace intercepts and can write go.work.sum instead of go.mod (silent no-op).                         |
| Tag waves (`scripts/tag-release.sh`)                     | repo root           | `off` (in-script) | Script strips local replaces and tidy/builds each module standalone at cut time.                         |
| `cmd/api-stability` golden regen                         | `cmd/api-stability` | `off`             | Golden must reflect the published module graph, not workspace replacements.                              |
| `cmd/doc-check`                                          | `cmd/doc-check`     | `off`             | Import-path verification resolves modules via the published graph.                                       |
| cqrs-lint consumer probes                                | the probe dir       | off               | Probe needs its own go.mod + a single `--path`; the linter skips non-consumer projects silently.         |
| Anything from inside a package dir with doubts           | repo root first     | on                | "directory prefix . does not contain modules" = you are in the wrong cwd/mode combination.               |

## Environment chain (mandatory for every go/nix command)

```
export GOCACHE=/home/lars/projects/.gocache-disk GOMODCACHE=/tmp/gomod-verify GOPATH=/tmp/gopath-verify GOTOOLCHAIN=auto GOLANGCI_LINT_CACHE=/home/lars/projects/.golangci-disk GOTMPDIR=/home/lars/projects/.gotmp TMPDIR=/home/lars/projects/.gotmp
```

Build tag everywhere: `-tags "goexperiment.jsonv2"` (env `GOEXPERIMENT=jsonv2`
covers `go test`/`go vet`; CI and `nix run .#build` apply it automatically).
