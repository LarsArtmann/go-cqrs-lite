# Feature: checked-in findings baseline for mechanical drift detection

## Why

Adopting consumers accumulate adjudicated findings — deliberate non-adoptions, ratified architecture choices, known detector gaps. Today the only tools are per-site (`//cqrs-lint:ignore`, ignore-start blocks, field-level directives) and per-rule (`rules.disable`, `--exclude-rules`). Neither answers the recurring operational question:

> "The count moved from 116 to 120 — which findings are NEW?"

The CRM answered exactly that this morning and it took a full investigation (rule-by-rule count diff against a prose disposition table) to classify 4 findings as pre-existing noise vs signal. Every cqrs-lint build that shifts detection width re-runs that manual triage. Per-site suppressions are the wrong tool for it: project-level advisory findings (F-series adoption coaching, E003 on deliberately-colocated packages) would need ignore comments sprayed across files that will never act on them, and rule-disable loses the drift signal entirely — you can't see a NEW F019 appear if F019 is off.

## Proposal

A baseline file, e.g. `.cqrs-lint-baseline.json`, plus one flag:

```
cqrs-lint . --baseline .cqrs-lint-baseline.json   # exit 0: only baseline-known findings remain
cqrs-lint . --baseline … --update-baseline        # regenerate (explicit, never implicit)
```

Semantics:

- Baseline entries keyed by finding ID (rule + file + line) — which is exactly why the E003 ID-collision issue needs fixing first; with unique IDs this is mechanical.
- Findings present in the run but not the baseline → reported normally + nonzero exit (or a dedicated exit marker); findings in the baseline but gone → reported as "baseline shrunk" so the file can be pruned (stale entries should be as visible as new ones — same spirit as `--fail-on-stale-suppressions`).
- Baseline entries optionally carry a reason field, so the file doubles as the machine-readable disposition log projects currently keep in prose.

Precedent in the fleet: art-dupl's `.art-dupl-baseline.json` (`art-dupl check --baseline-path …`, exit 1 on NEW groups only) — same workflow, proven over months on the same repos.

## To verify

- Repo with 3 known findings: `--baseline` + no changes → exit 0, quiet; introduce one new finding → exit 1 naming only the new one; fix a baseline finding → "baseline shrunk" nudge; `--update-baseline` rewrites deterministically (sorted).

💘 Generated with Crush
