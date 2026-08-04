# cqrs-lint — Consumer Feedback (cqrs-htmx, Round 2)

**Consumer:** [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) v4.6.1 — CQRS/HTMX Go library
**cqrs-lint version:** 4.3.0 (commit: e350355, built: 20260802224206)
**go-cqrs-lite version:** v4.2.0 (command/event/query/id)
**Date:** 2026-08-04
**Previous feedback:** [2026-07-17_cqrs-htmx_cqrs-lint-feedback.md](../2026-07-17_cqrs-htmx_cqrs-lint-feedback.md)
**Final state this session:** `No findings. Clean!` — 130 suppressed, 0 unsuppressed, 0 stale.

---

## TL;DR

Three issues, in priority order:

1. **End-of-line suppressions silently don't work** (BUG, source-confirmed). The
   parser uses `strings.HasPrefix` on the full line, so trailing comments after
   code are never recognized. Consumers waste hours discovering this
   empirically.

2. **Feature profile is workspace-wide, not per-module** (DESIGN GAP). A
   multi-module workspace with both library code and example apps gets a single
   merged profile that's wrong for every individual module.

3. **No `.cqrs-lint.json` was created** (OUR FAULT). The `library` preset and
   config system exist but cqrs-htmx never adopted them. This inflated the
   suppression count from ~80 to ~130. Half the suppressions are
   configurable-away false positives for library code.

---

## Issue 1: End-of-line suppressions silently don't work (BUG)

### Severity: HIGH (DX-breaking, silent failure)

### What happens

A consumer writes:

```go
EventType = sdk.EventType //cqrs-lint:ignore(A008) re-export of SDK type
```

cqrs-lint reports the finding anyway, showing the comment in its output but
not acting on it. There is no warning or error — the suppression is silently
ignored.

### Root cause (source-confirmed)

In `pkg/suppression/parser.go`, function `ParseSuppressions` (line 267):

```go
func ParseSuppressions(commentText string) map[string]string {
    result := make(map[string]string)
    lines := strings.SplitSeq(commentText, "\n")
    for line := range lines {
        line = strings.TrimSpace(line)
        line = normalizeCommentPrefix(line)
        if !strings.HasPrefix(line, commentPrefix) {  // <-- HERE
            continue
        }
        // ... extract rule IDs ...
    }
    return result
}
```

The check `strings.HasPrefix(line, commentPrefix)` requires the line to START
with `//cqrs-lint:ignore`. For an end-of-line comment, the trimmed line is:

```
EventType = sdk.EventType //cqrs-lint:ignore(A008) re-export
```

This does NOT start with `//cqrs-lint:ignore`, so the entire suppression is
skipped.

`checkSuppressionInFile` (line 144) does check the finding's own line:

```go
// Check the finding's own line.
if line >= 1 && line <= len(lines) {
    suppressedRules := ParseSuppressions(lines[line-1])
    if _, ok := suppressedRules[ruleID]; ok {
        return true
    }
}
```

But since `ParseSuppressions` can't see past the code prefix, end-of-line
comments are never recognized.

### The fix

Option A — search for the prefix anywhere in the line (simplest):

```go
// Replace HasPrefix with Contains:
idx := strings.Index(line, commentPrefix)
if idx < 0 {
    continue
}
// Extract from the comment prefix onward:
line = line[idx:]
```

Option B — split the line on `//` and check each segment:

```go
for _, segment := range strings.SplitSeq(line, "//") {
    segment = "//" + strings.TrimSpace(segment)
    segment = normalizeCommentPrefix(segment)
    if strings.HasPrefix(segment, commentPrefix) {
        // process this segment
    }
}
```

Option A is simpler and handles the common case. Option B is more correct
(handles multiple comments on one line) but more complex.

### Impact

Any consumer who reads the help text ("at end of line") and writes an
end-of-line suppression will have it silently ignored. This is the #1 DX
issue I encountered this session — I wasted ~14 tool calls applying 9 inline
suppressions, running cqrs-lint, discovering none worked, reverting, and
re-applying as line-above comments.

### Documentation says it should work

The `cqrs-lint --help` output states:

> Place inline suppressions on the line above the code or at end of line.

This is incorrect for the current implementation. Either fix the parser or
fix the help text.

---

## Issue 2: Feature profile is workspace-wide, not per-module

### Severity: MEDIUM (significant for multi-module workspaces)

### What happens

cqrs-htmx is a Go workspace (`go.work`) with 19 independent Go modules:

- **Library modules** (root, usermgmt, identity-model, datastar, etc.) —
  imported by consumers, never run as applications
- **Application modules** (`examples/basic`, `examples/dashboard-demo`, etc.) —
  runnable demo apps with `main()` and `ListenAndServe`

cqrs-lint detects features across the ENTIRE workspace and applies a single
merged `FeatureProfile` to all modules. This means:

| Detected value | Source | Correct for library? | Correct for examples? |
|---------------|--------|---------------------|----------------------|
| `store: sqlite` | `examples/*` import `stack/sqlite` | WRONG — library supports all backends | Correct |
| `server: true` | `examples/*` call `ListenAndServe` | WRONG — library has no `main()` | Correct |
| `command-flow: commands` | `usermgmt` calls `Dispatch` | Misleading — library provides dispatch infrastructure | N/A |
| `tracing: on` | Library imports `go.opentelemetry.io` | Misleading — library doesn't force tracing | N/A |
| `snapshot: on` | Library imports `snapshot` module | Misleading — library makes snapshot opt-in | N/A |
| `transport: true` | Library IS `cqrs-htmx` | Correct | Correct |

### Root cause

In `pkg/analyzer/feature_detect.go`, `DetectFeatures()` iterates over
`ctx.Packages` and `ctx.GoFiles` — all packages in the workspace — without
grouping by Go module. A `ListenAndServe` call in `examples/basic/main.go`
sets `fp.HasServer = true` for the entire workspace.

### Impact

The wrong profile causes rules to fire that shouldn't:

- `server: true` enables server-only rules (E009, E014, F011, F012, F013)
  that fire on library modules where they're false positives
- `store: sqlite` triggers SQLite-specific rules (P012, P013) on modules
  that don't even use SQLite
- `command-flow: commands` suppresses read-only rules that WOULD fire
  correctly on the dashboardui module (which is read-only)

### Suggested fix

**Per-module feature profiles.** When analyzing a workspace with multiple
`go.mod` files, detect features per-module and apply each module's profile
only to its own packages. The output already separates by module
(`=== /home/lars/projects/cqrs-htmx/dashboardui (12) ===`) — extend this to
the feature profile.

### Workaround (consumer-side)

Create per-module `.cqrs-lint.json` files:

```json
// Library modules: .cqrs-lint.json
{"preset": "library"}

// Example modules: .cqrs-lint.json
{"preset": "production"}
```

The parent-config inheritance (`loadParentRulesConfig`) already merges
rule disables from parent directories. Extending this to feature profiles
would partially solve the problem without code changes.

---

## Issue 3: The `library` preset doesn't go far enough

### Severity: LOW (design improvement)

### What happens

The `library` preset sets:

```go
PresetLibrary: {
    Features: ConfigFeatures{
        Server:      new(false),
        CommandFlow: new(CommandFlowReadOnly),
        Tracing:     new(TracingOff),
        Snapshot:    new(SnapshotOff),
    },
    Rules: RulesConfig{
        Disable: []string{"E003", "E016"},
    },
}
```

This is a good start but misses several rules that are false positives for
library code:

| Rule | Why it fires on libraries | Why it shouldn't |
|------|--------------------------|-----------------|
| **F002** (no event catalog) | Library defines event types but doesn't own the catalog | Consumer registers events in their catalog |
| **F006** (PII without encryption) | Library defines PII-bearing event payloads | Consumer configures encryption middleware |
| **F009** (no scheduling module) | Library has time-based features (session expiry) | Consumer chooses scheduling strategy |
| **F010** (no graph module) | Library has hierarchical queries | Consumer chooses graph traversal strategy |
| **F011** (no relational projection) | Library does multi-table writes in read models | Consumer chooses projection strategy |
| **S002** (PII without encryption middleware) | Same as F006 | Same |
| **S003** (no event signing) | Library creates events without signing | Consumer configures signing middleware |
| **S007** (in-memory session store) | Library provides in-memory store as default | Consumer configures persistent store |
| **B025** (no state cache) | Library creates repositories without WithStateCache | Library DOES wire state cache via a helper function |

### Suggestion

Add a `library-framework` preset (or extend `library`) that disables
adoption-coaching rules (F-series) and consumer-responsibility rules
(S002, S003, S007) for modules that are libraries, not applications.

---

## Issue 4: Missing `.cqrs-lint.json` — our fault

### Severity: N/A (consumer error, now understood)

cqrs-htmx never created a `.cqrs-lint.json`. The auto-detection ran with
no overrides, producing the wrong profile for a multi-module library
workspace. This inflated the suppression count significantly.

**What we should have done** (and will do next):

```json
// Root .cqrs-lint.json — applies to library modules
{
  "preset": "library",
  "rules": {
    "disable": ["F002", "F006", "F009", "F010", "F011", "S002", "S003", "S007"]
  }
}
```

And per-module overrides for examples:

```json
// examples/*/.cqrs-lint.json
{
  "preset": "production"
}
```

This would eliminate ~30-40 of the 130 suppressions.

---

## Rule-Level Feedback

### What's great

- **Comma-separated suppressions work** — `//cqrs-lint:ignore(B025,A017,E008)`
  correctly suppresses all three rules on the next line. This is the
  single most useful feature for managing multi-rule findings on the same
  line (e.g., `decider.NewRepository` triggering B025 + A017 + E008).
  Source at `parser.go:288` splits on commas inside `(...)`. Verified
  empirically on v4.3.0.

- **Block suppressions** (`//cqrs-lint:ignore-start` / `ignore-end`) —
  excellent for suppressing ranges (entire structs, demo seed-data blocks).
  Haven't needed them yet but they're the right escape hatch.

- **Stale suppression detection** — catches suppressions whose rule no
  longer fires. Found 7 stale suppressions this session. Very useful.

- **`normalizeCommentPrefix`** — accepting both `//cqrs-lint:` and
  `// cqrs-lint:` (with space) handles the gofmt doc-comment normalization
  correctly. Consumers don't need to worry about gofmt breaking their
  suppressions.

- **`doctor` command** — shows detected feature profile and loaded rule
  overrides. Great for debugging "why is this rule firing?"

### Rules that produce false positives for library code

| Rule | Category | False positive pattern |
|------|----------|----------------------|
| **E003** | Architecture | "Package mixes command/event/fold concerns" — but identity-model IS the domain model package; separating would create artificial module boundaries |
| **E005** | Architecture | "Command type has no registered handler" — fires on internal/test command types that are never dispatched |
| **E009** | Architecture | "No HTTP/gRPC transport layer" — fires on dashboardui which IS the transport layer |
| **E014** | Architecture | "No projection drain/sync call" — fires on dashboardui which consumes projections, doesn't own them |
| **F002** | Adoption | "No event catalog" — fires on observability/dashboard modules that display data, not produce events |
| **F006** | Adoption | "PII field without encryption module" — fires on library event payload definitions; consumer configures encryption |
| **F010** | Adoption | "No graph module" — fires on modules with tree/hierarchy queries; recursive CTEs are often sufficient |
| **F011** | Adoption | "No relational projection" — fires on read-only modules that don't own projections |
| **S002** | Security | "PII without encryption middleware" — same as F006; library can't force encryption on consumers |
| **B025** | Boilerplate | "No WithStateCache" — fires even when state cache IS wired, just via a helper function the linter can't see through |

### B025 — specific detector improvement needed

**B025** fires on `decider.NewRepository(store, bus, decider, opts...)`
when `opts` doesn't literally contain `decider.WithStateCache(...)`. But
in cqrs-htmx, the options are built by a helper function:

```go
func repositoryOptions[State any](cfg SnapshotConfig) []decider.RepositoryOption[State] {
    opts := snapshotOptions[State](cfg)
    return append(opts, decider.WithStateCache[State](decider.NewStateCache[State](0)))
}
```

The linter sees `repositoryOptions[UserState](snap)...` and can't trace
through the function call to discover that `WithStateCache` IS present.

**Suggestion:** When `NewRepository` receives a `...opts` spread from a
function call, check if that function (or its callees) contains
`WithStateCache`. If function-call tracing is too expensive, at minimum
recognize the pattern `someFunction(opts...)` and lower the confidence
(rather than firing at full confidence).

---

## The "115 suppressions" question

### Is this a consumer problem or a linter problem?

**Both, roughly 60/40 consumer/linter.**

### Breakdown of the 130 suppressed findings (post-session)

| Category | Count | Root cause | Fixable by |
|----------|-------|-----------|------------|
| **Library design decisions** | ~40 | Legitimate suppressions for opt-in patterns (snapshot, tracing, signing, encryption, session store) | Neither — these ARE library design decisions |
| **Examples/demo code** | ~30 | Demo seed data discards errors, omits catalog, omits schema versions | Linter: add `examples` exclusion or `demo` preset. Consumer: `.cqrs-lint.json` per-module |
| **Consumer-responsibility rules** | ~25 | F002/F006/F009/F010/F011/S002/S003 fire on library code that can't enforce these | Linter: extend `library` preset to disable F-series + S-series |
| **Detector limitations** | ~15 | B025 can't see through helper functions; A032 flags display DTOs; C009 flags constructor guards | Linter: improve detectors. Consumer: suppress. |
| **Multi-module workspace** | ~10 | Wrong feature profile causes server/store/snapshot rules to fire on library modules | Linter: per-module profiles. Consumer: `.cqrs-lint.json` |
| **gofmt-normalized comments** | ~10 | C009 panics, C035 maps, P011 unbounded maps | Legitimate suppressions for dev/test patterns |

### What the linter could eliminate

- **~25 findings** by extending the `library` preset to disable F-series
  adoption coaching and S-series consumer-responsibility rules
- **~10 findings** by per-module feature profile detection
- **~5 findings** by improving B025 to trace through helper functions
- **~30 findings** by supporting an `examples/` exclusion or `demo` preset

Total: **~70 of 130 suppressions are linter-addressable**.

### What the consumer must own

- **~40 findings** are legitimate library design decisions that will always
  need suppressions (opt-in patterns, constructor guards, dev/test stores)
- **~20 findings** are in examples that could be excluded but aren't

---

## Feature Profile Analysis

### Detected profile (current, no config)

```
store:         sqlite      ← WRONG (from examples, library supports all)
command-flow:  commands    ← MISLEADING (library provides infrastructure)
server:        true        ← WRONG (from examples, library has no main())
soft-delete:   false       ← CORRECT
tracing:       on          ← MISLEADING (library imports but doesn't force)
snapshot:      on          ← MISLEADING (library imports but makes opt-in)
domain:        unknown     ← CORRECT (identity management, not financial)
transport:     true        ← CORRECT (cqrs-htmx IS the transport)
server-local:  false       ← WRONG (derived from wrong server=true)
```

### Correct profile for cqrs-htmx library modules

```
store:         none        ← Library doesn't mandate a store
command-flow:  read-only   ← Library provides dispatch infrastructure
server:        false       ← Library has no main()
soft-delete:   false       ← Correct
tracing:       off         ← Library imports but doesn't force
snapshot:      off         ← Library makes opt-in
domain:        unknown     ← Correct
transport:     true        ← Correct
server-local:  false       ← Correct (server=false)
```

### How to get there

```json
// .cqrs-lint.json (root — applies to library modules)
{
  "preset": "library"
}
```

The `library` preset pins server=false, command-flow=read-only, tracing=off,
snapshot=off. This is exactly right for the library modules.

For examples, a separate `.cqrs-lint.json` per example directory:

```json
// examples/dashboard-demo/.cqrs-lint.json
{
  "preset": "production"
}
```

---

## Summary of Recommendations

| # | Issue | Type | Fix effort | Impact |
|---|-------|------|-----------|--------|
| 1 | End-of-line suppressions silently ignored | BUG | Small (parser fix) | HIGH — eliminates silent DX failures |
| 2 | Per-module feature profiles | DESIGN | Medium (group by go.mod) | HIGH — eliminates ~10 false positives per workspace |
| 3 | Extend `library` preset | DESIGN | Small (add rule disables) | MEDIUM — eliminates ~25 suppressions per library |
| 4 | B025 helper-function tracing | DETECTOR | Medium (AST tracing) | LOW — 4 findings in this project |
| 5 | `examples` exclusion or `demo` preset | DESIGN | Small | MEDIUM — eliminates ~30 suppressions |

---

## Appendix: Verification Commands

```bash
# Version check
cqrs-lint version
# → cqrs-lint 4.3.0 (commit: e350355, built: 20260802224206)

# Full strict run (post-session, all fixed)
cqrs-lint --strict --verbose --show-suppressed
# → No findings. Clean! (130 suppressed)

# Doctor (feature profile + config)
cqrs-lint doctor
# → shows detected profile, loaded overrides

# Comma-separated suppression test
# Place //cqrs-lint:ignore(B025,A017,E008) above a decider.NewRepository call
# → all three rules suppressed correctly
```
