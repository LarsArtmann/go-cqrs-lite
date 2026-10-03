# Nightly gates (local, this shared host)

Wires the promoted quiet-window tooling into a scheduled nightly verification
of the committed benchmark baseline, and a nightly lint gate against the
auto-commit daemon. The bench gate refuses loud machines (noise gate + save
guard), so a nightly run is green-log-or-triage — it never re-pins on its own.

## Install (one-time; adjust the path if the repo moves)

```bash
mkdir -p ~/.config/systemd/user
cp scripts/nightly/go-cqrs-nightly-bench.{service,timer} scripts/nightly/go-cqrs-nightly-lint.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now go-cqrs-nightly-bench.timer go-cqrs-nightly-lint.timer
```

## What runs

`scripts/nightly-bench.sh` (03:00) → `scripts/quiet-window-run.sh` (waits up
to 3h for load1/load5 < 5, 2 attempts) → `scripts/benchmark-regression.sh`
(verification vs `benchmarks/benchmark-baseline.txt`; known-unstable
benchmarks advisory; noise-gate failures are retried on a quieter window).

`scripts/nightly-lint.sh` (03:30) → `nix run .#lint` — the
daemon-bypasses-lint gate (2026-10-03). The auto-commit daemon absorbs
working-tree changes without ever running a linter, so master can rot
silently between CI pushes. A pre-commit hook was considered and rejected:
it would stall or fail-loop the daemon on concurrent agents' transiently
broken trees. The nightly timer catches rot within 24h with zero daemon
interference. Self-test (fault injection via `NIGHTLY_LINT_CMD`) is wired
into `nix run .#check-release-scripts`.

## Logs

`/var/tmp/cqrs-nightly/<UTC-date>.log` — one file per night; a `REGRESSION`
line means a decision-grade quiet run saw a real median breach: triage in the
morning, re-pin only with a deliberate, noise-clean `--save`.

`/var/tmp/cqrs-nightly/<UTC-date>-lint.log` — a `LINT-ROT` line means the
daemon absorbed lint-dirty code into master: triage the golangci-lint output
above the marker.
