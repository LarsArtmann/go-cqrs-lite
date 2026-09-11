# Runbook: Flipping the turso-go grouped-matview caveat when upstream fixes it

> THE one procedure for "a new turso-go release is out — what now?" and "the
> defects are fixed — how do we lift the caveat?". It consolidates what used
> to live in a test comment, the research draft, and prose. The verified
> version range itself is single-sourced in
> `metaengine/materialized_view_versions.go` (`metaengine.TursoGoIVMVerifiedThrough`)
> and enforced by `nix run .#check-turso-version` — never edit citations by hand
> without following step 4 here.

Background: grouped materialized views (`MaterializedViewSpec` with `GroupBy`)
silently lose cross-transaction deltas on turso-go (defect A), collapse past
~27k rows (defect B), and hit a COMMIT-abort wall (defect C, PR #8257).
Scalar views are exact. Full characterization:
`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`.

## 1. Check a new turso-go release (recurring, ~2 min)

```bash
cd metaengine/tursoengine
go mod edit -require=turso.tech/database/tursogo@vNEW && go mod tidy
GOWORK=off go test -tags "goexperiment.jsonv2 ivmrepro" -run TestIVMRepro -count=1 -timeout 30m .
```

- All three `TestIVMRepro*` tests PASS → the defects are still live on
  `vNEW`. Decide whether to adopt `vNEW` as the pin; either way update the
  canonical constants (step 4) with `TursoGoIVMLastVerified` = today.
- Any `TestIVMRepro*` FAILS with "did not reproduce" → upstream fixed that
  defect. Do NOT adopt the pin yet — finish the full flip (steps 2-6) in the
  same change as the pin bump.
- Revert an exploratory pin you are not adopting:
  `go mod edit -require=turso.tech/database/tursogo@vOLD && go mod tidy`
  (or `git restore metaengine/tursoengine/go.mod metaengine/tursoengine/go.sum`).

## 2. Enforce the fix through the guard

The defect-A guard currently SKIPS while the defect is live. Run it in
enforce mode — it must FAIL with "defect A no longer reproduces" (that
failure IS the flip signal):

```bash
cd metaengine/tursoengine
TURSO_IVM_ENFORCE_FIX=1 GOWORK=off go test -tags "goexperiment.jsonv2" \
  -run TestTursoMatView_GroupedSumDefectAEnvelopeGuard -count=1 .
```

Then remove the `TURSO_IVM_ENFORCE_FIX` skip gate from
`TestTursoMatView_GroupedSumDefectAEnvelopeGuard`
(`metaengine/tursoengine/matview_property_test.go`) so the exactness assertion
runs unconditionally and any upstream regression flips CI loudly.

## 3. Remove the Doctor WARN

`metaengine/materialized_view_doctor.go` emits the grouped-view WARN.
Delete the WARN block and its pin assertions in
`metaengine/materialized_view_doctor_test.go`
(`TestMaterializedViewsDoctorSection_GroupedWarnPin`) — keep the
`NoGroupNoWarn` test. If grouped views then need their own guidance, say so
positively (e.g. "grouped views exact as of turso-go vX"), don't leave a
stale warning.

## 4. Single-source the citation bump

One edit site, one gate:

1. Update `metaengine/materialized_view_versions.go`:
   - `TursoGoIVMVerifiedThrough` = the first version where the fix is
     verified,
   - `TursoGoIVMLastVerified` = today,
   - extend `TursoGoIVMVerifiedFrom` only if the fixed range needs restating.
2. Run `nix run .#check-turso-version` — it fails on every live citation
   (gotchas, readmodels, recipes, FEATURES, TODO_LIST, ADR-0135, bench doc,
   test comments) that still names the old "through" version. Fix each listed
   site in the same change. Historical records (CHANGELOG entries, research
   drafts, archived status reports) are point-in-time and intentionally not
   gated — leave them.

## 5. Un-skip the grouped benches

`metaengine/tursoengine/matview_bench_test.go` benches grouped matviews only
at scale=1k and `b.Skip`s the larger scales ("grouped matview seeding is
unreliable above ~1k rows"). Remove those skips so grouped read acceleration
is benched at every scale again (the seed guards in `seedInTxE` can relax to
the normal chunking once the COMMIT wall is gone — verify with a full
`nix run .#bench` sweep on the turso backend before shrinking them).

## 6. Land it

- `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`
  (the constants are exported API).
- CHANGELOG `[Unreleased]` entry citing `metaengine.TursoGoIVMVerifiedThrough`
  (`scripts/check-changelog-symbols.sh` gates the symbol).
- Update the upstream artifacts: comment on PR #8257 / the standalone issue
  (if filed) that the fix is verified, close the TODO_LIST items
  ("Code guard follow-up", "Sharpen the defect-A characterization") that this
  flip resolves.
- `nix run .#verify` (the chain includes `#check-turso-version`).
- Only after the tree is clean and green: bump the pin(s) per the AGENTS.md
  tag-wave procedure if you are shipping the new driver version to consumers.
