# cqrs-lint FP sweep baseline

**Current baseline: 2026-09-28T22:46:52Z re-run (binary `1962b83fd`) — supersedes the
known-bad 2026-09-17 snapshot below.**

The 2026-09-17 capture was taken with a harness that swallowed stderr and read
empty JSON output as "0 findings": five repos (bank-sync, browser-history,
github-local-sync, go-localsync, dnsblockd) were SILENT rows, and `overview`
was an unlabeled blank. The 2026-09-28 harness refresh
(`scripts/fp-sweep.sh`: per-repo stderr capture + `NO JSON OUTPUT` skip label +
`${n:-0}` arithmetic hardening) re-ran the same 12 repos to produce this
corrected baseline.

## Corrected baseline (2026-09-28T22:46:52Z, binary `1962b83fd`)

| repo                         | findings | low-confidence (<0.5) | notes                                                               |
| ---------------------------- | -------- | --------------------- | ------------------------------------------------------------------- |
| cqrs-htmx                    | 61       | 4                     | stderr: 5 line(s)                                                   |
| bank-sync                    | 6        | 0                     |                                                                     |
| browser-history              | 4        | 0                     |                                                                     |
| github-local-sync            | 14       | 2                     | stderr: 5 line(s)                                                   |
| go-localsync                 | 15       | 3                     |                                                                     |
| crush-daily                  | 38       | 14                    | stderr: 5 line(s)                                                   |
| timesheets                   | 6        | 2                     |                                                                     |
| accountability-system        | 39       | 3                     | stderr: 5 line(s)                                                   |
| overview                     | —        | —                     | transitive-only consumer (7 indirect pins, 0 direct) — labeled skip |
| storbi                       | 32       | 1                     |                                                                     |
| standard-bug-tracking-schema | 194      | 9                     |                                                                     |
| dnsblockd                    | 17       | 3                     |                                                                     |

**Total: 426 findings, 41 low-confidence suspects.**

Outlier verdicts (M10.4, 2026-09-29): crush-daily 38/14 is STABLE vs the old
39/15 — a consistent adoption-debt signal, not a flake.
standard-bug-tracking-schema 194/9 is likewise stable.

## Superseded snapshot (2026-09-17T06:12:00Z, binary `dfc3cc492`) — KNOWN-BAD

Kept for provenance only. Five silent rows, `overview` blank, totals
undercounted.

| repo                         | findings | low-confidence (<0.5) |
| ---------------------------- | -------- | --------------------- |
| cqrs-htmx                    | 57       | 5                     |
| bank-sync                    |          |                       |
| browser-history              |          |                       |
| github-local-sync            |          |                       |
| go-localsync                 |          |                       |
| crush-daily                  | 39       | 15                    |
| timesheets                   | 6        | 2                     |
| accountability-system        | 36       | 3                     |
| overview                     |          |                       |
| storbi                       | 25       | 2                     |
| standard-bug-tracking-schema | 194      | 9                     |
| dnsblockd                    |          |                       |

Total (as reported then): 357 finding(s), 36 low-confidence suspect(s)
