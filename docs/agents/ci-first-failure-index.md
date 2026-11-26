# CI First-Failure Index

> Failed leg → known classes → local repro command. Every class below cost a
> 25–100 min CI run to rediscover at least once (2026-11-21/22 CI-heal round 3);
> this page exists so the next occurrence costs one lookup instead.
> Companion to `gotchas-tooling-build.md` (which carries the full narratives).

## Pulling the evidence before diagnosing

```bash
gh run list --limit 10                                   # find the run id
gh api repos/LarsArtmann/go-cqrs-lite/actions/runs/<id>/jobs?per_page=100 \
  --jq '.jobs[] | "\(.name)\t\(.conclusion)"'            # which legs
gh run view <id> --log-failed --job <job-id>             # failing step log
# raw job logs (ANSI included): --allow-escape-sequences | sed 's/\x1b\[[0-9;]*m//g'
```

## The index

| Failed leg / symptom                                           | Known classes (newest first)                                                                    | Local repro                                                    |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| Whole run dies in <1s, "workflow file issue"                   | duplicate YAML key; bad expression                                                              | `nix run .#check-lint-config` (actionlint step)                |
| Verify-fast: docs-ui.css stale                                 | `build-docserver-css.sh` /v4 path class                                                         | `nix run .#check-docserver-css`                                |
| Verify-fast / flake-check: file-size                           | NEW >350-line file or growth of a baselined one                                                 | `nix run .#check-file-size`                                    |
| Flake-check step N (a `check-*` gate)                          | every gate has a local twin — match the step name to `nix run .#<same-name>`                    | `nix run .#check-<gate>`                                       |
| Flake-check: error-taxonomy ratchet                            | new documented code without a test; baseline row whose code left the doc                        | `nix run .#check-error-taxonomy-tests`                         |
| Flake-check: coverage floor                                    | module/package regressed past tolerance                                                         | `nix run .#check-coverage`                                     |
| Flake-check: test-runtime budget                               | environment-mode flip (Docker-present runner, cold proxy) — name the cause, raise with a reason | `nix run .#check-test-runtime`                                 |
| readme-links / doc gates                                       | row targets an UNCOMMITTED file (local pass ≠ CI)                                               | `bash scripts/check-readme-links.sh` (flags untracked targets) |
| Any script leg: path not found                                 | `/v4` path-move class in `$ROOT` constructions                                                  | `rg 'REPO_ROOT/<module>' scripts/ .github/`                    |
| Job compiles with "cannot read export data" / toolchain errors | runner's PATH go older than the module floor                                                    | fix: `setup-go` with `go-version-file`                         |
| Shell Format Drift (shfmt)                                     | local shfmt 3.13.1 vs CI 3.14.1 heredoc/case disagreement                                       | `nix shell nixpkgs#shfmt -c shfmt -d scripts/`                 |
| NATS/broker leg hangs then passes with 0 tests                 | script's interactive wait branch (no-args must RUN the suite)                                   | run the script with no args locally, watch for a prompt        |
| MySQL leg: 1213 deadlock under claimers                        | normal InnoDB gap-lock behavior; retry ladder (queue/mysql claim.go)                            | `nix run .#integration-mysql-vm`                               |
| SSE drop-count flake                                           | global counter delta attributing OTHER tests' deliberate evictions                              | scope the assertion to the unit under test                     |
| Job "succeeds" suspiciously fast                               | the work never ran (silent-skip class — NATS was the precedent)                                 | check the step output shows real test names                    |

## Standing pre-push candidate checklist (~2 min, from the round-3 scars)

Before any receipt-candidate push, run: `#check-file-size` · actionlint on
workflow diffs (inside `#check-lint-config` now) · `check-readme-links` ·
`#check-generated-artifacts` · api golden regen (`cd cmd/api-stability/v4 &&
GOWORK=off go run . --update`) · `#check-docserver-css`. Full narrative and the
8-candidate chain that motivated it:
`docs/status/2026-11-22_14-53_ci-heal-round3-honest-closeout-status.md`.
