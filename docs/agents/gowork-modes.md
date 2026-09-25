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

**One sourced file since 2026-09-22 (T05):**

```
source scripts/go-env.sh
```

It force-overrides exactly these (silent when already correct) — spell the
raw exports only where sourcing is impossible:

```
export GOCACHE=/home/lars/projects/.gocache-disk GOMODCACHE=/tmp/gomod-verify GOPATH=/tmp/gopath-verify GOTOOLCHAIN=auto GOLANGCI_LINT_CACHE=/home/lars/projects/.golangci-disk GOTMPDIR=/home/lars/projects/.gotmp TMPDIR=/home/lars/projects/.gotmp
```

Adopted by: the pre-commit hook, `check-coverage.sh`,
`benchmark-regression.sh`, `preflight-composed.sh`. The force-override
matters: the ambient session env can carry `GOTOOLCHAIN=local` (the nixpkgs
Go wrapper default) and caches pointed at `/mnt/buildcache` (100% full on
2026-09-22) — set-before-use forms let that poison through.

Build tag: NONE since Go 1.27 graduated `encoding/json/v2` (2026-09-19 sweep).
The former `-tags "goexperiment.jsonv2"` / `GOEXPERIMENT=jsonv2` are no-ops and
were removed from scripts, flake.nix, CI, and docs. Old commands that still
carry the tag still work (unknown tags are ignored).

**Contract enforcement (W3 Q4 ruling, 2026-09-20):** this chain — specifically
`GOTOOLCHAIN=auto` against the go.work directive — is the toolchain contract,
not a convention. `scripts/check-go-version.sh` (flake app `#check-go-version`,
also in the `#verify` head + nightly) fails loud when the selected toolchain is
older than the go.work contract or a `go` binary cannot answer `go env
GOVERSION`. The explicit nixpkgs pin lands when nixpkgs ships the contract
version.

**Two-tier load ceilings (intent, documented 2026-09-25):** the composed-`#verify`
load guard refuses at load 10 (`verify-load-guard.sh`, CalibGate-independent)
while calibration/baseline runs refuse at load 5 (`calibration-gate.sh`) —
deliberate, not drift. `#verify` is CPU-heavy itself and only needs "no other
heavy session"; baseline-producing benches need "the host quiet" because their
NUMBERS become committed constants (a load-9 verify window would poison a
SearchQuery median but is harmless to a test-pass verdict). Do not "unify" the
ceilings; widen either one only with a recorded reason.
