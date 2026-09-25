# ADR-0148: Benchmark Gate Semantics — Noise, Provenance, and Refusal

- Status: Accepted
- Date: 2026-09-25
- Deciders: owner directive via the T18b campaign (2026-09-20/21), codified here
- Canonical evidence record: [`docs/benchmarks/2026-09-20-21_t18b-record.md`](../benchmarks/2026-09-20-21_t18b-record.md)
- Supersedes: none (codifies shipped behavior that lacked an ADR)

## Context

The benchmark regression gate (`scripts/benchmark-regression.sh`) guards
committed performance baselines. Through 2026-09-20 it trusted three
assumptions that failed in production: that a quiet loadavg implies honest
samples, that a saved baseline is honest by construction, and that a failing
tool reports itself. The T18b campaign (six armed verify-and-save cycles
across two days, including a load-863 storm, an overnight reboot, and a
GOTOOLCHAIN poisoning incident) produced the semantics below. This ADR
records them as contract, so reverting any leg needs a decision instead of
a quiet edit.

## Decision

The gate enforces four semantics, in this order:

1. **Machine certification precedes run certification.**
   `calibration-gate.sh` (1-min AND 5-min load < 5) certifies the HOST;
   the per-metric noise gate certifies the RUN. Neither certifies a bimodal
   sampler — median-of-5 over a bimodal metric lands on either mode
   (observed: `MIN/matview` 7.9–15µs on a quiet machine). Bimodal-suspect
   metrics go to the `KNOWN_UNSTABLE` table with an expiry, not a verdict.

2. **A quiet loadavg is necessary, not sufficient.** Five load-≤5 runs
   produced four noise-gate rejections (chronic `write_p99_ns` CoV
   11.5–54%). Tail quantiles at ~100-iteration samples measure estimator
   variance, not machine loudness — `write_p99_ns` is demoted from the
   headline set and may not be re-added without new evidence.

3. **`--save` refuses on a noise-gate failure** (`--force-save` is the
   explicit override, and its use is an incident to note). A baseline saved
   from a loud run becomes the lie every future run is compared against.

4. **Provenance is part of the artifact.** `--save` writes a titled
   re-pin header (UTC time, uptime, GOVERSION, store paths). A baseline
   without provenance is not decision-grade. Secondhand version citations
   are banned (protocol §7, calibration-2026-08-30.md).

Supporting mechanics (already shipped, kept as-is): the two-tier load
ceilings (verify 10 vs calibration 5 — see
[`docs/agents/gowork-modes.md`](../agents/gowork-modes.md)), the
quiet-window wrapper for load-sensitive commands, per-suite
`benchtime::count` with CI parity, and the drift-tripwire pinning the
gate's `NOISE_HEADLINE` to `benchkit.HeadlineMetricNames()`.

## Consequences

- A "regression" verdict requires: gate-passed window + noise-passed run +
  median beyond threshold. Any missing leg makes the verdict "unknown", and
  unknown is reported as unknown.
- Baseline re-pins are audit events: the titled header diff is reviewable,
  and a re-pin without its provenance lines fails review.
- The `KNOWN_UNSTABLE` table is the only suppression mechanism; entries
  carry expiries, and an expired entry re-arms.
- Local (non-CI) saves additionally require the calibration gate; CI is
  exempt (shared-runner load is not the calibration host's load).

## Alternatives considered

- **Trust margins**: widen thresholds until storms pass — rejected; the
  storms were correctly identified as storms, and widening would have
  laundered them into baselines.
- **Nightly-only gating**: rejected; the load-sweep leg proved timing
  regressions surface under deliberate soak, which nightly cadence alone
  misses.
