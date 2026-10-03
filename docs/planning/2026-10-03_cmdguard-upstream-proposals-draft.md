# cmdguard upstream proposals — DRAFTS (filing is USER-GATED)

**Date:** 2026-10-03
**Provenance:** M10+M11 of the cqrs-lint CLI consistency Pareto plan
(`docs/planning/2026-10-03_05-16_SUPERB-cqrs-lint-cli-consistency-pareto-plan.md`).
**Status:** DRAFT — do not file upstream without owner review. All API claims
below were verified against cmdguard v4.0.2 source (`~/projects/cmdguard`,
`pkg/cmdguard/v4/`) on 2026-10-03.

Target repo: `github.com/larsartmann/cmdguard` (Lars owns it — filing is a
low-friction internal PR/issue, still gated for review).

---

## Proposal 1 — `WithSharedFlagSubset`: per-subcommand persistent-flag subsets

### Problem (verified motivating case)

cmdguard's persistence is all-or-nothing per field: `flag:""` on the config
struct propagates the flag to EVERY subcommand; `local:"true"` removes it
from every subcommand. There is no way to say "`--format` is shared, but
`rules` consumes only path/format/color while `version` consumes none of
them."

Consequences in cqrs-lint (all mechanically reproduced 2026-10-03):

- `cqrs-lint version --min-severity info` is ACCEPTED and silently ignored —
  the config-file parity design forces single-render commands to
  accept-and-ignore the whole shared vocabulary (documented in the README
  flag-consumption matrix as a workaround, pinned by
  `TestPersistentFlagAcceptanceMatrix`).
- Consumption honesty is documentable but not enforceable: no test can fail
  when a subcommand stops reading a flag it still accepts.

### API sketch

```go
// WithSharedFlagSubset scopes the persistent flags a command receives to a
// subset. Flags not listed are not registered on the command (unknown-flag
// error, same as local:"true") — the shared config struct is unchanged, so
// config-file values still populate cfg for commands that opt out.
func WithSharedFlagSubset(flags ...string) CommandOption
```

Usage (cqrs-lint):

```go
cmdguard.NewCommand("version", versionFlags{}, handler,
    cmdguard.WithShort("Print version"),
    cmdguard.WithNoArgs(),
    cmdguard.WithSharedFlagSubset(), // consumes nothing shared
)
cmdguard.NewCommand("rules", rulesFlags{}, handler,
    cmdguard.WithSharedFlagSubset("format", "color"),
)
```

Semantics decisions to settle upstream:

- Interaction with config-file loading: opting out of a FLAG must not opt
  out of the config KEY (the "config file must not break `version` in CI"
  ruling from cqrs-lint decision 2). Subset scopes flag REGISTRATION only.
- Empty subset = local-only for all inherited flags.

### Verify-need code sample (the motivating failing case today)

```go
cli, _ := cmdguard.NewCLI[Cfg]("tool", "d", Cfg{})
// Cfg.Format has flag:"format" (persistent)
cmd, _ := cmdguard.NewCommand[Cfg, NoFlags]("version", NoFlags{}, handler,
    cmdguard.WithShort("v"))
cmdguard.AddCommand(cli, cmd)
err := cli.ExecuteWithArgs(ctx, []string{"version", "--format", "json"})
// TODAY: accepted and ignored (handler never reads cfg.Format).
// WANTED: unknown-flag error, because version renders one format.
```

---

## Proposal 2 — validator-derived help lists

### Problem

Every command hand-writes its supported-values list in TWO places: the
validation call (`validateFormatFlag(cfg.Format, "text", "json")`) and the
help text (`WithShort("...; formats: text, json")`). cqrs-lint single-sourced
these via `formats.go` vocabulary slices + `withFormatsSuffix` — a
workaround inside ONE consumer. The framework could derive the help from the
validator registration:

### API sketch

```go
// WithFlagValidation registers an up-front validator for a flag and derives
// help text listing the accepted values, so the list exists exactly once.
func WithFlagValidation(flag string, values ...string) CommandOption
// Short/Long render "Values: a, b, c" for validated flags automatically.
```

The cqrs-lint `formats*.go` vocabulary slices + `TestCommandShortsDeriveFromVocabularies`
become unnecessary in every consumer that adopts it.

---

## Proposal 3 — unused-persistent-flag detector (lint/analyzer)

### Problem

A persistent flag that NO subcommand handler reads is a silently accepted
no-op — the exact disease the cqrs-lint audit found (`--min-severity` on
`rules`). The framework knows both the flag set and (via handler closures it
cannot introspect...) — so this ships as a static-analysis helper instead:

### Sketch

An analyzer (go/analysis) over consumer code: flag fields tagged `flag:""`
whose `cfg.<Field>` selector never appears outside the root RunE. Findings:
"persistent flag X is never read by any subcommand; scope it local or
consume it." cqrs-lint's README consumption matrix is the manual version of
this check; the analyzer makes it mechanical.

---

## Proposal 4 — document `local:"true"` scoping (docs PR)

`local:"true"` is a cmdguard struct-tag feature with no README section
(verified: repo README does not mention it as of v4.0.2; the cqrs-lint
AGENTS/README had to reverse-engineer and document the semantics —
"persistent by default, local:true = root only, cobra rejects on
subcommands"). Draft docs section:

```markdown
### Flag scoping

- Fields with a `flag:""` tag are PERSISTENT: cobra registers them on the
  root, every subcommand inherits them, and `registry.ParseFlags` syncs
  inherited values (flag OR config file) into the shared config before a
  subcommand handler runs.
- Add `local:"true"` to scope a flag to the root command only: subcommands
  reject it as an unknown flag. Use it for mode flags of the default
  (root) operation — accepting them elsewhere invites silent no-ops.
```

---

## Filing checklist (execute only after owner approval)

1. Proposal 1 as a cmdguard issue (WithSharedFlagSubset) with the
   verify-need sample.
2. Proposal 4 as a docs PR (smallest, unblock first).
3. Proposals 2+3 as issues referencing 1 (same disease family).
