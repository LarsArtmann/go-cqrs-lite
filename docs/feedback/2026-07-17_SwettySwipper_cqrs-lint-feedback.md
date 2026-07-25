# cqrs-lint — Consumer Feedback (SwettySwipper)

**Consumer:** [SwettySwipperWeb](https://github.com/LarsArtmann/SwettySwipperWeb) — media comparison/voting platform (Go monorepo, event-sourced CQRS, server-rendered HTML, SQLite)
**Version used:** go-cqrs-lite v4.0.0 (all modules)
**Lint version:** `cqrs-lint v0.2.0` (binary at `/home/lars/go/bin/cqrs-lint`)
**Date:** 2026-07-17

---

## Executive Summary

Ran cqrs-lint v0.2.0 against the full SwettySwipperWeb codebase (130 Go files, 9
services, Go workspace monorepo). The tool reported **88 findings** and a health
score of **0/100 ("Needs Improvement")**. After source-level verification of every
finding AND inspection of the detector source code:

| Category                                  | Count | Verdict                                                        |
| ----------------------------------------- | ----- | -------------------------------------------------------------- |
| **False positives — table-driven wiring** | 47    | E005 (21) + E007 (10 real queries) — cross-package indirection |
| **False positives — HTTP DTOs**           | 16    | E007 flags `*Request` structs that are not CQRS queries        |
| **Deliberate tested design**              | 6     | C003 — forward-compatible fold, tested by name                 |
| **Acceptable Must pattern**               | 1     | C009 — `readmodel.MustNew`, programmer-error-only panic        |
| **Valid boilerplate/style (INFO)**        | 18    | B001/B004/B005/B007/B010/A009/A017/D002/B013 — all declined    |

**Signal-to-noise ratio: 0%.** Zero of the 88 findings required a code change.
The health score of 0/100 is assigned to a codebase where every aggregate has
tested fold functions, every command is registered, and every query is dispatched.

### What improved from v0.1.0 to v0.2.0

The "reduce false positives" work (commit `00f678a0`) made real progress:

| Rule      | v0.1.0  | v0.2.0 | Delta   | What changed                                     |
| --------- | ------- | ------ | ------- | ------------------------------------------------ |
| A001      | 5       | 0      | -5      | No longer flags query types as "manual command"  |
| C009      | 6       | 1      | -5      | Recognizes `mustNew`/`Must` constructor pattern  |
| A012      | 6       | 0      | -6      | Tombstone rule removed entirely (was pure noise) |
| E005      | 31      | 21     | -10     | No longer flags query types as unregistered cmds |
| B010      | 0       | 1      | +1      | New: catalog boilerplate detection (useful)      |
| **Total** | **107** | **88** | **-19** |                                                  |

These are meaningful improvements. A001 and A012 elimination removed two entire
categories of false positives. The C009 `Must`-pattern recognition is exactly
right. The remaining 88 findings, however, are still 100% noise.

---

## Part 1: Persistent False Positives — Detector Limitations

### Bug 1: E005/E007 can't trace table-driven handler registration (31 findings)

**Severity of impact: CRITICAL** — this is the single biggest source of noise,
accounting for 31 of 88 findings and -94 health-score points.

#### What happens

SwettySwipper registers all command handlers via a table-driven map pattern that
the linter cannot trace:

```go
// services/battle/commands.go:65
func RegisterHandlers(repo *decider.Repository[BattleState], d *corecmd.Dispatcher) error {
    handlers := map[corecmd.Type]corecmd.Handler{
        command.CmdCreateBattle:  newCreateBattleHandler(repo),
        command.CmdArchiveBattle: newArchiveBattleHandler(repo),
    }
    for cmdType, handler := range handlers {
        if err := d.Register(cmdType, handler); err != nil {
            return errorfamily.WrapInfrastructure(err, "battle.dispatcher.register", "register command handler")
        }
    }
    return nil
}
```

Every command IS registered — the indirection through a `map[Type]Handler` loop is
the obstacle. The linter's `IsCommandRegistered` looks for direct
`d.Register(typeConst, ...)` calls, not `for k, v := range map { d.Register(k, v) }`.

The same pattern affects queries. Real query types are registered cross-package
in `services/api/handler/query_dispatcher.go` via `cqrsquery.RegisterTyped`, which
the linter cannot see from the service package where the query is defined.

#### Root cause

E005 (`architecture/rules.go`) checks `ctx.Registry.IsCommandRegistered(cmd.Name)`.
The registry's command registration tracking is populated by `scanner_calls.go`
which matches direct `Register(typeConst, handler)` calls. The table-driven map
pattern — a common, clean Go idiom — defeats this because:

1. The map literal contains `CmdConstant: handlerFunc` entries, not `Register()` calls
2. The `for` loop calls `d.Register(cmdType, handler)` where `cmdType` is a loop variable, not a constant

E007 (`e003_e007.go`) has a parallel issue: it checks
`ctx.Registry.IsCommandRegistered(ts.Name.Name)` for each struct, but query
registration happens in a different package (`api/handler/query_dispatcher.go`)
via `cqrsquery.RegisterTyped(d, QueryGetBattle, func(ctx, q) {...})`.

#### Why this is hard to fix

The linter does single-package AST analysis. Table-driven registration and
cross-package wiring are architectural patterns that require inter-package or
whole-program analysis to trace. This is a fundamental limitation of the current
scanner design.

#### Mitigation suggestions

1. **Detect the table-driven registration idiom**: If a package contains a
   `for` loop calling `Register` on map values, treat all command types in the
   map as registered. This is a targeted heuristic that handles the most common
   pattern.

2. **Cross-package awareness**: When scanning package A, check if any other
   package in the module/workspace calls `RegisterTyped` with types from A.

3. **Confidence downgrade to zero-impact**: Until inter-package tracing exists,
   E005/E007 should not deduct ANY health-score points. They are hints for
   human review, not correctness defects.

4. **Provide a suppression mechanism** (see Bug 4 below).

#### All 31 affected findings

**E005 (21 commands — all registered via table-driven map):**

| Command                  | Service        | Registration file:line        |
| ------------------------ | -------------- | ----------------------------- |
| CreateBattleCmd          | battle         | battle/commands.go:67         |
| ArchiveBattleCmd         | battle         | battle/commands.go:68         |
| CreateMediaItemCmd       | media          | media/commands.go:113         |
| ProcessMediaItemCmd      | media          | media/commands.go:114         |
| FailMediaItemCmd         | media          | media/commands.go:115         |
| RefreshMediaItemURLCmd   | media          | media/commands.go:116         |
| ReprocessMediaItemCmd    | media          | media/commands.go:117         |
| CreateMediaGroupCmd      | media          | media/group_commands.go:80    |
| UpdateMediaGroupCmd      | media          | media/group_commands.go:81    |
| DeleteMediaGroupCmd      | media          | media/group_commands.go:82    |
| CreateTournamentCmd      | tournament     | tournament/commands.go:99     |
| ActivateTournamentCmd    | tournament     | tournament/commands.go:100    |
| CompleteTournamentCmd    | tournament     | tournament/commands.go:101    |
| CancelTournamentCmd      | tournament     | tournament/commands.go:102    |
| AddTournamentBattlesCmd  | tournament     | tournament/commands.go:103    |
| CastVoteCmd              | voting         | voting/commands.go:57         |
| DeleteVoteCmd            | voting         | voting/commands.go:58         |
| StartVotingSessionCmd    | voting-session | voting-session/commands.go:92 |
| ProgressVotingSessionCmd | voting-session | voting-session/commands.go:93 |
| CompleteVotingSessionCmd | voting-session | voting-session/commands.go:94 |
| FailVotingSessionCmd     | voting-session | voting-session/commands.go:95 |

**E007 (10 real queries — all registered cross-package):**

| Query                | Service    | Registration file                               |
| -------------------- | ---------- | ----------------------------------------------- |
| GetBattleQuery       | battle     | api/handler/query_dispatcher.go (RegisterTyped) |
| ListBattlesQuery     | battle     | api/handler/query_dispatcher.go (RegisterTyped) |
| GetMediaItemQuery    | media      | api/handler/query_dispatcher.go (RegisterTyped) |
| ListMediaItemsQuery  | media      | api/handler/query_dispatcher.go (RegisterTyped) |
| GetMediaGroupQuery   | media      | api/handler/query_dispatcher.go (RegisterTyped) |
| ListMediaGroupsQuery | media      | api/handler/query_dispatcher.go (RegisterTyped) |
| GetTournamentQuery   | tournament | api/handler/query_dispatcher.go (RegisterTyped) |
| GetBracketQuery      | tournament | api/handler/query_dispatcher.go (RegisterTyped) |
| GetVotesQuery        | voting     | api/handler/query_dispatcher.go (RegisterTyped) |
| GetLeaderboardQuery  | voting     | api/handler/query_dispatcher.go (RegisterTyped) |

---

### Bug 2: E007 flags HTTP request DTOs as queries (16 findings)

**Severity of impact: HIGH** — 16 false positives, each labeled "Query type" when
the type is a plain HTTP JSON request body.

#### What happens

```go
// services/api/handler/vote_handler.go:52
type castVoteRequest struct {
    DecisionTimeMs *uint              `json:"decisionTimeMs,omitempty"`
    VoteType       enums.VoteType     `json:"voteType"`
    BattleID       ids.BattleID       `json:"battleId"`
    // ...
}
```

This is a JSON request body for an HTTP handler. It has no `Type()` method, no
`AggregateID()`, no connection to the query dispatcher. cqrs-lint flags it as:

```
WARNING  Query type "castVoteRequest" has no registered handler — dispatching it will fail
```

#### Root cause

In v0.2.0, E007's `isLikelyQuery` heuristic matches any struct whose name ends
in `Request` OR `Query`. The `*Request` suffix matches HTTP request DTOs that
have nothing to do with CQRS queries.

**This is already fixed in HEAD** (commit `00f678a0`, post-v0.2.0 tag): the
`isLikelyQuery` function was restricted to `*Query` suffix only, with a comment
documenting the rationale. The fix just hasn't shipped in a tagged release yet.

#### Fix status

Already fixed in source at HEAD. Ship v0.2.1 to resolve.

#### All 16 affected findings

| DTO name                     | File                                     |
| ---------------------------- | ---------------------------------------- |
| createBattleRequest          | api/handler/battle_handler.go:36         |
| importAllRequest             | api/handler/discord_browse_handler.go:61 |
| syncRequest                  | api/handler/discord_sync_handler.go:92   |
| createMediaGroupRequest      | api/handler/media_group_handler.go:32    |
| updateMediaGroupRequest      | api/handler/media_group_handler.go:103   |
| deleteMediaGroupRequest      | api/handler/media_group_handler.go:130   |
| createMediaItemRequest       | api/handler/media_handler.go:45          |
| failMediaItemRequest         | api/handler/media_handler.go:142         |
| cancelRequest                | api/handler/response.go:74               |
| completeTournamentRequest    | api/handler/response.go:81               |
| createTournamentRequest      | api/handler/tournament_handler.go:36     |
| castVoteRequest              | api/handler/vote_handler.go:52           |
| deleteVoteRequest            | api/handler/vote_handler.go:318          |
| createVotingSessionRequest   | api/handler/voting_session_handler.go:34 |
| progressVotingSessionRequest | api/handler/voting_session_handler.go:67 |
| failVotingSessionRequest     | api/handler/voting_session_handler.go:93 |

---

## Part 2: Deliberate Design Choices (Not Bugs)

### C003: Forward-compatible fold — tested contract (6 findings)

**Verdict: Deliberate design, not a defect.**

cqrs-lint flags all 6 fold functions for silently ignoring unknown event types
in the `default` case. The suggested fix is to return an error for unknown events.

SwettySwipper deliberately uses lenient folds for **forward compatibility**: when
a new event type is added in a new version, old fold functions skip it gracefully
instead of crashing during replay. This is a valid event-sourcing pattern.

**This is a tested contract**, not an accident:

```go
// services/battle/battle_test.go:104
t.Run("ignores unknown event types", func(t *testing.T) {
    evt := mustCreateEvent(t, event.Type("UnknownEvent"), aggID, 1, map[string]string{"foo": "bar"})
    state, err := foldBattle(BattleState{}, evt)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)  // ← error is NOT expected
    }
    if state.Status != "" {
        t.Errorf("expected empty status for unknown event, got %s", state.Status)
    }
})
```

Identical tests exist for `voting_session_test.go:138`. The test name itself —
`"ignores unknown event types"` — is a specification.

**Recommendation for C003:** The rule should detect this pattern and downgrade
to INFO, or the suggestion should acknowledge forward-compatible folds as a valid
alternative: "If this is a deliberate forward-compatible fold, annotate with
`//cqrs-lint:forward-compatible-fold` to suppress."

### C009: MustNew in readmodel — acceptable (1 finding)

```go
// services/readmodel/readmodel.go:44
func MustNew[T any](fold FoldFunc[T]) *ReadModel[T] {
    rm, err := New(fold)
    if err != nil {
        panic(err)
    }
    return rm
}
```

This is the standard `Must` constructor pattern (cf. `template.Must`,
`regexp.MustCompile`). The panic fires only when `fold` is nil — a programmer
error at initialization time, not a runtime condition. v0.2.0 correctly
suppressed the 5 `mustNew` helpers in command constructors but still catches this
one. Consistent treatment would suppress it too.

---

## Part 3: Critical Tooling Issues

### Bug 3: Health score 0/100 on a clean, tested codebase

**Severity of impact: CRITICAL** — the health score is the headline metric. A
0/100 score on a well-architected codebase makes the tool unusable for CI gating
and destroys trust in the metric.

#### Breakdown

The health score formula (from `health.go`):

| Severity | Base deduction | Confidence weight         |
| -------- | -------------- | ------------------------- |
| Critical | 10             | High/Full → 1.0           |
| Error    | 5              | Medium (0.5) → 0.75       |
| Warning  | 2              | Low (0.25) → 0.5          |
| Info     | 1              | No confidence (0.0) → 1.0 |
| Info cap | max -20        |                           |

The actual deductions from the v0.2.0 run:

| Rule | Severity | Findings | Expected (with conf.) | Actual | Health-score impact |
| ---- | -------- | -------- | --------------------- | ------ | ------------------- |
| E007 | warning  | 26       | 26 × 2 × 0.75 = 39    | **52** | -52                 |
| E005 | warning  | 21       | 21 × 2 × 0.75 = 32    | **42** | -42                 |
| C003 | error    | 6        | 6 × 5 × 1.0 = 30      | **30** | -30                 |

E005 and E007 both declare `ConfidenceMedium` (0.5, verified via JSON output),
which should map to weight 0.75. But the health score shows full-weight
deductions (-52 = 26 × 2, -42 = 21 × 2), as if confidence were not applied.

#### Suspected root cause

The `confidenceWeight` function in `health.go` treats the zero value (0.0) as
full weight (1.0):

```go
func confidenceWeight(c finding.Confidence) float64 {
    switch {
    case c >= finding.ConfidenceHigh:   // 0.75, 1.0 → 1.0
        return 1.0
    case c >= finding.ConfidenceMedium: // 0.5 → 0.75
        return 0.75
    case c >= finding.ConfidenceLow:    // 0.25 → 0.5
        return 0.5
    default:                            // 0.0 → 1.0 (!!!)
        return 1.0
    }
}
```

If the confidence field is not propagated from the finding to the health score
computation (e.g., a serialization gap, or the wrong struct being passed), the
default case kicks in and applies full weight. The JSON output confirms
`"confidence": 0.5` on the findings, so the value IS set — but it may not reach
`ComputeHealthScoreWithCap`.

**Even if confidence weighting worked correctly**, the score would still be
0/100: E007 (-39) + E005 (-32) + C003 (-30) = -101. The fundamental problem is
that false positives dominate.

#### Recommendations

1. **Fix the confidence propagation bug** so Medium/Low confidence findings cost less.
2. **Cap Warning-severity deductions** the same way Info is capped. A flood of
   false-positive warnings from a single broken detector (E005/E007) should not
   be able to zero the score.
3. **Consider confidence-gated scoring**: findings below Medium confidence should
   not deduct from the health score at all — they appear in the report but don't
   affect the number. This prevents speculative heuristics from dominating.
4. **Report the health score formula in the output** so users understand WHY the
   score is what it is. Currently the breakdown shows per-rule deductions but not
   the confidence weighting or caps applied.

---

### Bug 4: No general per-rule suppression mechanism

**Severity of impact: HIGH** — without suppression, every CI run fails on known
false positives, making the tool unusable in CI.

#### What exists

- `.cqrs-lint.json` supports `rules.external-api-struct-prefixes` for D002 only
- In-source marker `//cqrs-lint:external-api` for D002 only
- The `--only` flag filters to specific rules (inverse of what's needed)
- The `--exclude` flag excludes paths, not rules

#### What's missing

There is no way to:

1. **Disable a specific rule** (`"disable": ["E005", "E007"]`)
2. **Suppress a specific finding** (`//cqrs-lint:ignore E005`)
3. **Set per-rule severity** (`"severity": {"C003": "info"}`)
4. **Mark a pattern as intentional** (`//cqrs-lint:forward-compatible-fold`)

Every other production linter (golangci-lint, eslint, clippy) provides these.
Without them, the consumer's only options are:

- Fork the linter
- Don't use it in CI
- Accept 0/100 scores forever

#### Recommendation

Add a `"disable"` or `"ignore-rules"` array to `.cqrs-lint.json`:

```json
{
	"disable": ["E005", "E007"],
	"rules": {
		"external-api-struct-prefixes": ["Discord"]
	}
}
```

And an in-source marker for per-finding suppression:

```go
//cqrs-lint:ignore E005
func RegisterHandlers(repo *decider.Repository[BattleState], d *corecmd.Dispatcher) error {
```

---

### Bug 5: --fix crashes with path-doubling bug

**Severity of impact: HIGH** — auto-fix is completely non-functional.

#### What happens

```
$ cqrs-lint --fix --dry-run --only C003 .
Pipeline run: iteration 1: apply fixes: [io] backup
/home/lars/projects/SwettySwipperWeb/home/lars/projects/SwettySwipperWeb/services/battle/state.go:
[io] open file for backup: open
/home/lars/projects/SwettySwipperWeb/home/lars/projects/SwettySwipperWeb/services/battle/state.go:
no such file or directory.
```

The project root path is doubled: `<cwd>/<cwd>/services/battle/state.go`.

#### Root cause

`resolveSafePath` in `go-finding/pipeline/path_safety.go:23`:

```go
func resolveSafePath(rootDir, relPath string) (string, bool) {
    // ...
    fullPath := filepath.Join(rootDir, relPath)  // ← BUG: doubles if relPath is absolute
    // ...
}
```

`filepath.Join("/home/lars/projects/SwettySwipperWeb", "/home/lars/projects/SwettySwipperWeb/services/battle/state.go")`
produces `/home/lars/projects/SwettySwipperWeb/home/lars/projects/SwettySwipperWeb/services/battle/state.go`.

The cqrs-lint scanner stores **absolute paths** in `finding.Position.File`, but
`resolveSafePath` treats the second argument as a relative path and joins it with
`rootDir`.

#### Fix

```go
func resolveSafePath(rootDir, relPath string) (string, bool) {
    cleanRoot := filepath.Clean(rootDir)
    if resolved, err := filepath.EvalSymlinks(cleanRoot); err == nil {
        cleanRoot = resolved
    }

    var fullPath string
    if filepath.IsAbs(relPath) {
        fullPath = relPath  // Already absolute — don't join
    } else {
        fullPath = filepath.Join(rootDir, relPath)
    }
    cleanPath := filepath.Clean(fullPath)
    // ... rest of safety check unchanged
}
```

Or fix the caller (cqrs-lint scanner) to store relative paths in findings.

---

### Bug 6: Version mismatch

**Severity of impact: LOW** — but confusing for bug reports.

```
$ cqrs-lint version
cqrs-lint 0.2.0

$ cqrs-lint --format json . | head -5
{
  "tool": {
    "name": "cqrs-lint",
    "version": "0.2.1"     ← different!
  },
```

The `version` command and the JSON metadata disagree. This makes it impossible to
know which version actually produced a given report.

---

## Part 4: INFO Findings — Reviewed and Declined (18 findings)

All INFO-severity findings were reviewed. None required action.

| Rule | Count | Assessment                                                                                                 |
| ---- | ----- | ---------------------------------------------------------------------------------------------------------- |
| B004 | 10    | Command constructor boilerplate — `cqrs-gen` not adopted; manual constructors are clearer for this project |
| B005 | 6     | Fold switch → `decider.StrictApply` — conflicts with forward-compatible fold design (see C003)             |
| A017 | 6     | Missing snapshot strategy — premature optimization; aggregates are small                                   |
| B013 | 1     | Missing correlation enricher — valid enhancement, ticketed                                                 |
| D002 | 1     | Mixed JSON casing — Discord SSE types mirror Discord API; `//cqrs-lint:external-api` marker appropriate    |
| B010 | 1     | Catalog boilerplate — 12 events; `cqrs-gen` adoption is a future decision                                  |
| B007 | 1     | Handler registration loop — 3 entries; table-driven is already the pattern used                            |
| B001 | 1     | Single-event helper — `event.Single()` not available in v4.0.0                                             |
| A009 | 1     | Missing stack preset — custom wiring is intentional for this monorepo                                      |

---

## Summary: What Would Make cqrs-lint Usable for SwettySwipper

| Priority | Issue                              | Effort  | Impact                                      |
| -------- | ---------------------------------- | ------- | ------------------------------------------- |
| P0       | Ship E007 `*Request` fix (v0.2.1)  | Done    | Eliminates 16 FP                            |
| P0       | Add rule suppression mechanism     | Medium  | Makes tool usable in CI                     |
| P0       | Fix --fix path-doubling bug        | Small   | Makes auto-fix work at all                  |
| P1       | Fix health score confidence bug    | Small   | Medium/Low confidence findings cost less    |
| P1       | Cap Warning deductions in health   | Small   | One broken detector can't zero the score    |
| P1       | Detect table-driven registration   | Medium  | Eliminates 21 E005 FP                       |
| P2       | Cross-package query tracing        | Large   | Eliminates 10 E007 FP                       |
| P2       | C003 forward-compatible fold aware | Small   | Downgrade or suppress for deliberate design |
| P3       | Version mismatch fix               | Trivial | Consistent version reporting                |

The tool has excellent potential. The rule catalog is comprehensive, the AST
analysis is solid, and the v0.1.0 to v0.2.0 improvements show the right
trajectory. The blocking issue is that **without a suppression mechanism, the
tool cannot be adopted in CI** — every run produces 88 findings and a 0/100 score
on a codebase with zero actual defects.
