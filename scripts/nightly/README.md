# Nightly benchmark gate (local, this shared host)

Wires the promoted quiet-window tooling into a scheduled nightly verification
of the committed benchmark baseline. The gate itself refuses loud machines
(noise gate + save guard), so a nightly run is green-log-or-triage — it never
re-pins on its own.

## Install (one-time; adjust the path if the repo moves)

```bash
mkdir -p ~/.config/systemd/user
cp scripts/nightly/go-cqrs-nightly-bench.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now go-cqrs-nightly-bench.timer
```

## What runs

`scripts/nightly-bench.sh` → `scripts/quiet-window-run.sh` (waits up to 3h for
load1/load5 < 5, 2 attempts) → `scripts/benchmark-regression.sh` (verification
vs `benchmarks/benchmark-baseline.txt`; known-unstable benchmarks advisory;
noise-gate failures are retried on a quieter window).

## Logs

`/var/tmp/cqrs-nightly/<UTC-date>.log` — one file per night; a `REGRESSION`
line means a decision-grade quiet run saw a real median breach: triage in the
morning, re-pin only with a deliberate, noise-clean `--save`.
