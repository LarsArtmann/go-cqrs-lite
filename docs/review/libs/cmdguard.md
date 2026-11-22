# cmdguard — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/cmdguard/v5` @ 0bf6678 (latest local tag v5.1.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G4

**Verdict:** ADOPTED
**Score:** 8/11 capabilities fully leveraged = 73% (high)

cmdguard is the CLI framework of record for go-cqrs-lite's tooling: all five
command-bearing tools (`cqrs-gen`, `cqrs-lint`, `cqrs-bench`,
`api-stability`, `doc-check`) build on `NewCLI` + struct-tag flags +
`ExecuteAndExit`, and the deepest adopter (cqrs-bench, 28+ typed flags;
cqrs-lint, 7 subcommands with config-file loading) uses the library the way
its README preaches. What keeps this at 73% instead of the high 90s: the
battery layer (`WithBatteries`, auto `version`/`doctor`) is untouched —
cqrs-lint hand-registers a custom `version` command next to the
`WithCLIVersion` option it already passes — and the repo's sixth CLI,
`cqrs-upgrade`, still runs on bare stdlib `flag.NewFlagSet`. All five pins
sit one minor behind at v5.0.0 while other fleet repos have already moved
to v5.1.0.

## 1. What the library is

"The Go CLI framework that catches missing handlers, duplicate commands, and
invalid flags at construction — not at 2am in production" (cmdguard
README.md:11). A Cobra wrapper adding: type-safe struct-tag flags (13 tags
incl. `prompt`/`secret`), dependency injection with lifecycle (samber/do,
lazy `Provide`/`Invoke`, reverse-order teardown), zero-panic error contract
(no `Must*`, no `Run`), construction-time command-tree validation, health
checks (`DoctorCommand` + `Healthchecker` services), graceful shutdown
(SIGINT/SIGTERM, ON by default in v5), styled output (fang + lipgloss),
typo suggestions, config-file precedence (flag → env → file → default,
strict keys), `WithCLIVersion`/`WithBatteries` batteries
(`pkg/cmdguard/v5/cli_batteries.go:43-83`), `describe`/docgen, and an
auditlog middleware (cmdguard README.md:13-16, 63-83). Separate nested
modules: `spinner`, `flightrecorder`, `telemetry`, `glamour`, `prompts`,
`schema`, `lint`. v5.0.0 released 2026-11-08; **v5.1.0** (2026-11-11) adds
`WithStrictPositionalArgs`/`WithArgsPassthrough` and a concurrency-safe
`Builder` (cmdguard CHANGELOG.md:98-110). cmdguard's own go.mod carries the
entire go-output submodule family for its output-format machinery
(cmdguard/go.mod:33-43,83) — which is why all five tools see go-output
indirectly.

## 2. Usage census in go-cqrs-lite (evidence)

Pins — all five command tools, all at **v5.0.0** (one minor behind v5.1.0):

- `cmd/cqrs-gen/v4/go.mod:6`, `cmd/cqrs-bench/v4/go.mod:6`,
  `cmd/api-stability/v4/go.mod:6`, `cmd/doc-check/v4/go.mod:6`,
  `cmd/cqrs-lint/v4/go.mod:6` — `github.com/larsartmann/cmdguard/v5 v5.0.0`.

Not pinned: `cmd/cqrs-upgrade/v4/go.mod` (requires only cqrs-lint + x/mod)
— its CLI is stdlib `flag.NewFlagSet`
(`cmd/cqrs-upgrade/v4/main.go:70`).

Import sites (package `cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"`):

- cqrs-lint (deepest): `main.go:170-176` `NewCLI` + `WithCLIVersion` +
  `WithConfigFileLoader(JSONCLoader{}, ".cqrs-lint.json")` + `WithCLILong`;
  `main.go:186-263` root RunE + `ExecuteAndExit`; 7 hand-registered
  subcommands each via `NewCommand` + `WithShort`/`WithNoArgs`
  (`commands.go:42-55,111-123`, `init.go`, `doctor.go:42-50`,
  `scorecard_command.go`, `explain.go` incl. `WithMaximumArgs` +
  `ArgsFromContext`); config plumbing via `ConfigFileLoader`,
  `ParseFlagTags`, `FilterSetFields` (`config_loader.go`); typed rule
  internals reuse `TypedReader` (`cmd/cqrs-lint/v4/pkg/rules/adoption/f031.go:134`).
- cqrs-bench: `main.go:34-38` `NewCLI[AppConfig]` + `WithCLIVersion`; three
  subcommands via generic `NewCommand[AppConfig, *RunFlags]`
  (`main.go:46-83`); the flag system carries ~30 typed flags incl.
  `cmdguard.Duration` (`flags.go:32,49`).
- api-stability: `main.go:220-228,266` `NewCLI` + `WithCLILong` +
  `ExecuteAndExit`.
- doc-check: `main.go:54-60,89` same shape.
- cqrs-gen: `main.go:59-63,52` same shape.

A cross-tool bootstrap guard comment marks the shared shape as intentional
(`cmd/cqrs-lint/v4/main.go:179` "identical by design across cmd tools").

## 3. Capability inventory

| #  | Capability                                                                  | Grade    | Evidence / reason                                                                                                                                                                                                                                                             |
| -- | --------------------------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Construction + registration validation (`NewCLI`/`NewCommand`/`AddCommand`) | **Full** | all five tools; duplicate/missing-handler errors surface at build time (cmdguard README.md:15)                                                                                                                                                                                |
| 2  | Struct-tag typed flags (incl. `cmdguard.Duration`)                          | **Full** | `cmd/cqrs-bench/v4/flags.go:16-81` (~30 flags, `cmdguard.Duration` at :32/:49), lint `AppConfig` + per-command flag structs (`cmd/cqrs-lint/v4/doctor.go:24-31`)                                                                                                              |
| 3  | Error contract: printed once, exit codes (`ExecuteAndExit`)                 | **Full** | `cmd/cqrs-lint/v4/main.go:263`, `cmd/cqrs-bench/v4/main.go:105`, `cmd/doc-check/v4/main.go:89`, `cmd/cqrs-gen/v4/main.go:52`, `cmd/api-stability/v4/main.go:266` — zero hand-rolled `os.Exit(1)` ladders in the tools' mains                                                  |
| 4  | Graceful shutdown (SIGINT/SIGTERM, default-ON in v5)                        | **Full** | inherited by all five without opt-out; relevant for bench soaks (`--soak 1h`, `cmd/cqrs-bench/v4/flags.go:49`)                                                                                                                                                                |
| 5  | Arg validators (`WithNoArgs`, `WithMaximumArgs`, `ArgsFromContext`)         | **Full** | lint: `commands.go:47`, `init.go`, `doctor.go`, `explain.go` (`WithMaximumArgs` + `ArgsFromContext`)                                                                                                                                                                          |
| 6  | Typo suggestions ("did you mean?")                                          | **Full** | automatic on every tool via the cobra wrapper (cmdguard README.md:77)                                                                                                                                                                                                         |
| 7  | Config-file precedence (`WithConfigFile`/`WithConfigFileLoader`)            | **Full** | lint wires `.cqrs-lint.json` through the loader extension point (`cmd/cqrs-lint/v4/main.go:175`, `config_loader.go` with `ParseFlagTags`/`FilterSetFields`); other tools legitimately need no config                                                                          |
| 8  | `WithCLIVersion` + auto `version` battery                                   | Partial  | set by 2/5 tools (`cmd/cqrs-lint/v4/main.go:174`, `cmd/cqrs-bench/v4/main.go:38`); lint _also_ hand-registers a custom `version` command (`cmd/cqrs-lint/v4/commands.go:111-123`) — the exact row cmdguard's "Replace your hand-rolled" table targets (cmdguard README.md:72) |
| 9  | Styled output (fang/lipgloss help + errors)                                 | Partial  | inherited by all five; but lint deliberately bypasses it for findings text with raw ANSI (`cmd/cqrs-lint/v4/output.go:6-8,67-70`) and layers go-output tables on top                                                                                                          |
| 10 | Battery commands (`WithBatteries`, `DoctorCommand`, `Healthchecker`)        | Missed   | no tool calls `WithBatteries` (`pkg/cmdguard/v5/cli_batteries.go:43` registers version+doctor); lint's `doctor` is a domain command (suppression audits, `cmd/cqrs-lint/v4/doctor.go:42-50`) that only shares the name                                                        |
| 11 | DI with lifecycle (`Provide`/`Invoke`, scoped services)                     | N/A      | short-lived single-process CLIs; nothing service-shaped to inject — engines are constructed per-run in RunE. Flip trigger: a tool grows background workers (watch/lintd)                                                                                                      |
| 12 | Fleet consistency: every repo CLI on the framework                          | Missed   | `cmd/cqrs-upgrade/v4/main.go:70` uses stdlib `flag.NewFlagSet` with manual `os.Args` dispatch (`main.go:46`) — the repo's own migration tool is the odd one out                                                                                                               |

Score derivation: rows N=12, N/A row 11 → **A=11**. Full=7 (7.0),
Partial=2 (1.0), Missed=2 (0). Score = round(100 × 8.0 / 11) = **73% →
high band**.

## 4. Gap analysis

1. **The battery layer is the unclaimed half.** cmdguard's pitch is "delete
   your plumbing" (cmdguard README.md:63-83): version command, doctor,
   did-you-mean, exit ladder. The tools took the foundation and the exit
   ladder but left version/doctor batteries on the shelf — most visibly in
   cqrs-lint, which passes `WithCLIVersion(resolvedVersion())` at
   `main.go:174` and then registers its own `version` command at
   `commands.go:111` for `--verbose` output. The verbose variant is a real
   need, but it is an extension of the battery, not a replacement for it.
2. **cqrs-upgrade is the fleet-consistency hole.** Five tools demonstrate
   the cmdguard pattern (with a comment marking the bootstrap identical by
   design, `cmd/cqrs-lint/v4/main.go:179`); the sixth re-implements flag
   parsing, usage text, and exit handling on `flag.NewFlagSet`
   (`cmd/cqrs-upgrade/v4/main.go:46,70`). Every future cmdguard UX fix
   (suggestions, styled help, config precedence) reaches five of six tools.
3. **One minor behind, and the fleet is already ahead.** v5.0.0 pins vs
   v5.1.0 latest; `WithStrictPositionalArgs` is directly useful to
   cqrs-lint's path-argument handling (today hand-guaranteed via cobra
   `MaximumNArgs(1)` at `cmd/cqrs-lint/v4/main.go:189`), and the
   concurrency-safe `Builder` matters for tools that queue commands
   dynamically. Fleet evidence: go-credentials-store already bumped to
   v5.1.0 in its v0.4.3 coordination window (go-credentials-store
   CHANGELOG.md [0.4.3], go-credentials-store/go.mod:8), so go-cqrs-lite is now the laggard in
   its own dependency family.

## 5. Version currency

- Pinned: **v5.0.0** by all five tools; latest local tag **v5.1.0**
  (2026-11-11). One minor behind; repo HEAD 0bf6678 (2026-11-21) is docs/
  tooling work beyond the tag (cmdguard CHANGELOG.md:18-62).
- v5.1.0 delta for this consumer: `WithStrictPositionalArgs`/
  `WithArgsPassthrough` (maps onto lint's root-path arg), mutex-safe
  `Builder`, a doctor doc-example fix — additive, no breaking changes, so
  the bump is low-risk (cmdguard CHANGELOG.md:98-110).
- The v4 line is retired (final v4.3.1, frozen on `release/v4`) — the
  consumer never pinned v4, so no transition debt. The v4→v5 dual-support
  transition in flight in the wider fleet does not touch these tools.
- cmdguard itself pins go-output v0.38.7 across all submodules
  (cmdguard/go.mod:33-43,83), so bumping cmdguard also refreshes the
  indirect go-output graph coherently.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only
fix bugs all day long." cmdguard is the fleet's one place CLI plumbing gets
fixed, and go-cqrs-lite's tool cluster is its most concentrated customer —
five CLIs, one shared bootstrap shape, zero hand-rolled exit ladders. That
is the fleet argument working. The remaining seams are exactly where the
lens bites: cqrs-upgrade re-integrates flag parsing by hand, so every CLI
UX improvement lands 5/6; and the untouched batteries mean each tool
re-decides version/doctor behavior itself. With go-credentials-store (and
per its changelog, CV) already on v5.1.0, staying on v5.0.0 makes the
template repo the thing other fleet projects have to work around.

## 7. Ranked opportunities

1. **Migrate cqrs-upgrade to cmdguard** (high): one-file change
   (`cmd/cqrs-upgrade/v4/main.go:46-70`), immediately inherits suggestions,
   styled help, error contract; makes the repo's CLI story uniform.
2. **Bump v5.0.0 → v5.1.0 across all five go.mod pins** (high, trivial):
   additive release; restores fleet lockstep with go-credentials-store/CV;
   then adopt `WithStrictPositionalArgs` for lint's root path arg instead
   of the cobra-level `MaximumNArgs(1)` shim (`cmd/cqrs-lint/v4/main.go:189`).
3. **Collapse lint's custom version command onto the battery** (medium):
   keep `--verbose` by extending the registered command, delete the
   parallel implementation (`cmd/cqrs-lint/v4/commands.go:111-123`).
4. **Give api-stability/doc-check/cqrs-gen versions** (low): they never set
   `WithCLIVersion`; the same `debug.ReadBuildInfo` pattern lint uses
   (`cmd/cqrs-lint/v4/main.go:27-46`) ports in three lines each and lights
   up the version battery.
5. **Evaluate `WithBatteries()` as the default posture** (low): where a
   tool has no custom doctor, one option registers version + doctor for
   free (`pkg/cmdguard/v5/cli_batteries.go:43-83`).

## 8. Reconsider when

- Any tool grows long-running services or watchers — the DI row flips from
  N/A to the cheapest lifecycle answer in the fleet.
- cqrs-upgrade gains subcommands or config — then migrating it stops being
  hygiene and becomes the obvious path.
- cmdguard ships a styled-text/finding-rendering API — lint's documented
  bypass (`cmd/cqrs-lint/v4/output.go:6-8`) could finally close.
- The v5 line adds spinner/telemetry batteries a bench soak would want.

## 9. How to verify

```
rg -n 'larsartmann/cmdguard' -g go.mod /home/lars/projects/go-cqrs-lite
rg -on 'cmdguard\.[A-Z][A-Za-z]*' --type go /home/lars/projects/go-cqrs-lite/cmd | sort | uniq -c | sort -rn | head -20
git -C /home/lars/projects/cmdguard log -1 --format='%H' && git -C /home/lars/projects/cmdguard tag --list | sort -V | tail -3
sed -n '168,200p;232,263p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/main.go
sed -n '111,123p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/commands.go
sed -n '34,83p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-bench/v4/main.go
sed -n '46,80p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-upgrade/v4/main.go
sed -n '98,110p' /home/lars/projects/cmdguard/CHANGELOG.md
sed -n '63,83p' /home/lars/projects/cmdguard/README.md
```

Related reports in this directory: `go-output.md` (the output layer
cmdguard carries indirectly), `go-credentials-store.md` (fleet repo already
on v5.1.0).
