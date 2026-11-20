# cqrs-lint stale-suppression gate is CLI-only — BuildFlow's embedded toolsdk run can never fail on stale directives

TL;DR: `--fail-on-stale-suppressions` and `doctor --audit-suppressions` exist only as CLI flags, but BuildFlow's embedded run goes through the toolsdk `Spec` (`Detect`/`Repair`, no options surface), so consumers whose ONLY invocation is `buildflow -s cqrs-lint` can never gate on stale `//cqrs-lint:ignore` directives — dead directives accumulate as blind spots. Ask: **a `.cqrs-lint.json` key (e.g. `{"suppressions": {"fail-on-stale": true}}`) that makes `DetectStaleSuppressions` emit a warning-severity finding per stale directive at the Spec boundary**, so every toolsdk host inherits the gate through its normal findings flow, no flag plumbing.

## Reproduced

cqrs-lint v4.22.2 embedded in the BuildFlow binary (root `go.mod` pin, toolspec blank import). The Spec carries no options:

```go
// skip-validate: pseudo-code sketch of the upstream Spec shape (elided fields as ...)
// pkg/toolspec/toolspec.go
func Spec() toolsdk.Spec {
	return toolsdk.Spec{
		Name: ..., Trigger: ..., Inputs: ...,
		Detect: finding.DetectorFunc(detect),
		Repair: toolsdk.RepairerFunc(...),
	}
}
```

No path exists to set `FailOnStaleSuppressions` (`main.go:128` flag) from a host. quotaboard's cqrs-lint invocation is buildflow-only, so its six suppression sites (five inline directives plus one config disable) can only be audited by running a SEPARATE standalone cqrs-lint binary the repo otherwise never uses.

## Root cause (source)

- The stale detector is pipeline-level and flag-gated: `pkg/suppression/stale_test.go` covers `DetectStaleSuppressions`; `main.go:128` binds it to `--fail-on-stale-suppressions` (CLI exit code only).
- The Spec boundary (`pkg/toolspec/toolspec.go:detect`) runs the suppression FILTER (`suppression.NewSuppressionFilter().Transform`) but never the stale audit, and reads only rule config (`loadEffectiveRules`), which has no suppressions key.

## Design

Add a suppressions section to `.cqrs-lint.json`; when enabled, `detect` (the toolsdk entry) emits one warning-severity finding per stale directive, positioned at the directive's own line — BuildFlow's findings gate then fails the step exactly as it would for any other warning, and the CLI keeps its exit-code behavior. Related but distinct: #126 fixed the audit classifying profile-gated suppressions as stale; this issue is about surfacing the (now-correct) audit through the embedded path.

💘 Generated with Crush
