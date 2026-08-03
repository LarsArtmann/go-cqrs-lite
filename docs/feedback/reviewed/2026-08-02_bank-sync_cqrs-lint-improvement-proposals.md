# cqrs-lint — Consumer Feedback Round 2: Improvement Proposals (bank-sync)

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — CLI tool that syncs bank transactions (Wise API + Qonto CSV) into SQLite via event sourcing
**Version used:** go-cqrs-lite v4.2.0 (event, decider, storage, middleware, watermill, scenario, schema, snapshot, codec, command, query, catalog, otel, retry, encryption, id)
**lint version:** `cqrs-lint v0.2.2`

> **Routing (2026-08-02):** P0 items (B022 wrong function name, P012/P013
> cross-file blindness) are tracked in TODO_LIST.md "cqrs-lint — Open Work".
> Config-level rule disabling and `--exclude-rules` CLI flag are the top
> cqrs-lint priority. This file is the canonical consumer feedback record.
> **Date:** 2026-08-02
> **Previous feedback:** [2026-07-17_bank-sync_cqrs-lint-feedback.md](../2026-07-17_bank-sync_cqrs-lint-feedback.md) (v0.1.0)

---

## Executive Summary

First: **the v0.1.0 → v0.2.0 improvements were excellent.** The generic-call detection fix (`unwrapSelector`), closure handler tracing (`*ast.FuncLit` parameter extraction), upcaster context detection, feature-profile system, and context-aware heuristics for A012/A015/A016/S002 addressed nearly every issue from the previous feedback round. The signal-to-noise ratio improved dramatically — from 39% in v0.1.0 to roughly **70% in v0.2.2** (9 of 30 findings are actionable or correctly suppressed; the remaining 21 are false positives or not-applicable module suggestions for a CLI tool).

This document focuses on what's **still broken, missing, or could be significantly better** in v0.2.2. It is organized by impact: correctness bugs first, then missing features, then quality-of-life improvements.

| Category                                                       | Count     | Impact                                                                |
| -------------------------------------------------------------- | --------- | --------------------------------------------------------------------- |
| **Wrong suggestion text** (B022)                               | 1         | High — sends users to a non-existent function                         |
| **Cross-file detection blindness** (P012/P013)                 | 4         | High — unsuppressable false positives on every file that opens SQLite |
| **No config-level rule disabling**                             | 1 feature | Medium — forces inline comments everywhere                            |
| **F-series missing feature-profile gating** (F009, F015, F017) | 3         | Medium — fires on projects where modules are deliberately not used    |
| **Missing auto-fix for D007**                                  | 1         | Medium — mechanical migration is perfect for `--fix`                  |
| **Quality-of-life gaps**                                       | 6         | Low — documentation, presets, stale detection                         |

---

## Part 1: Correctness Bugs

### Bug 1: B022 suggests a function that does not exist

**Severity: HIGH** — this actively misleads users.

#### What happens

bank-sync uses `event.CommandCausalityEnricher` (the correct function in v4.2.0):

```go
// infrastructure.go:153
decider.WithEnricher[BalanceSyncState](event.CommandCausalityEnricher),
```

cqrs-lint reports:

```
WARNING  B022  Custom enricher (WithEnricher) passed to decider.NewRepository —
use decider.CommandCausalityEnricher for typed command causality
```

The suggestion says to use `decider.CommandCausalityEnricher`. **This function does not exist.** The enricher lives in the `event` package, not `decider`:

```go
// vendor/.../event/v4/causality.go:64
func CommandCausalityEnricher(ctx context.Context) []Option {
```

The `decider` package has no `CommandCausalityEnricher` export. A user following the suggestion would get a compile error.

#### Root cause

File: `cmd/cqrs-lint/pkg/rules/boilerplate/b022_b025.go:83-93`

```go
f, err := finding.NewBuilder(
    "B022", toolName,
    "Custom enricher ("+argName+") passed to decider.NewRepository — "+
        "use decider.CommandCausalityEnricher for typed command causality",
    // ...
).WithSuggestion("Replace the custom enricher with decider.CommandCausalityEnricher — " +
    "it stamps metadata.command.type and metadata.command.id on every event").
```

The suggestion text is hardcoded as `decider.CommandCausalityEnricher` but the actual function is `event.CommandCausalityEnricher`.

#### Additionally: B022's exemption list doesn't recognize the correct function either

The exemption check (`b022_b025.go:70-78`) exempts calls whose name contains `CommandCausalityEnricher`. bank-sync's code calls `event.CommandCausalityEnricher` — this IS exempted by the name check. But the finding still fires because the detector wraps the check in `containsEnricher(arg)`, which matches any argument containing "enrich" (case-insensitive). The selector expression `event.CommandCausalityEnricher` contains "Enricher" → matches → fires B022.

**The exemption is checking the wrong thing.** It exempts based on the argument expression text, but the enricher IS the `event.CommandCausalityEnricher` function itself — it's not a "custom enricher" at all. The detector should recognize `event.CommandCausalityEnricher` as the canonical enricher and NOT flag it.

#### Fix suggestions

1. **Fix the suggestion text** in `b022_b025.go:86,92` — change `decider.CommandCausalityEnricher` to `event.CommandCausalityEnricher`.

2. **Fix the exemption logic** — the current check exempts calls whose name IS `CommandCausalityEnricher` (line 70-78). But the detection at line 54 (`containsEnricher(arg)`) fires BEFORE the exemption check. The logic should be:
   ```go
   // Skip if this IS the canonical enricher
   if isCanonicalEnricher(arg) {
       continue
   }
   // Only flag if it's a CUSTOM enricher (not the library's own)
   if containsEnricher(arg) {
       // flag
   }
   ```

---

### Bug 2: P012/P013 — Cross-file detection blindness

**Severity: HIGH** — produces 4 unsuppressable false positives on every project that wraps SQLite in a storage package.

#### What happens

bank-sync opens SQLite and applies WAL + busy_timeout in `internal/storage/sqlite/storage.go`:

```go
// storage.go:65-77
func applyPragmas(db *sql.DB) error {
    pragmas := []string{
        "PRAGMA journal_mode=WAL",
        "PRAGMA busy_timeout=5000",
        "PRAGMA synchronous=NORMAL",
    }
    // ...
}
```

The CLI commands (`demo.go`, `helpers.go`) call `sqlite.New(cfg.Database.Path)` which internally calls `applyPragmas`. The WAL and busy_timeout ARE applied — on every connection.

cqrs-lint fires on `demo.go` and `helpers.go`:

```
WARNING  P012  demo.go:1:1  SQLite store without WAL mode
WARNING  P013  demo.go:1:1  SQLite store without busy_timeout
WARNING  P012  helpers.go:1:1  SQLite store without WAL mode
WARNING  P013  helpers.go:1:1  SQLite store without busy_timeout
```

#### Root cause

Files: `cmd/cqrs-lint/pkg/rules/performance/p012.go` and `p013.go`

Both detectors scan **per-file** using `ast.Inspect` on `*ast.CallExpr` nodes and `strings.Contains` on the call text. They check whether `SQLiteEnableWAL`, `PRAGMA journal_mode`, or `busy_timeout` appears **in the same file** as the `sqlite.New` call.

When `sqlite.New()` lives in `storage.go` and the CLI command in `demo.go` calls it, the detector sees `sqlite.New` in `demo.go` but not the PRAGMA calls in `storage.go`. There is no cross-file tracking.

#### Impact on suppression

This is the most frustrating part. These false positives are **nearly impossible to suppress cleanly**:

1. **Inline `//cqrs-lint:ignore(P012,P013)` before `package main`** — the suppression parser supports comma-separated rules (confirmed in `parser.go:263-271`), and checks the finding's line + the line above (`parser.go:136-145`). But the finding is at `line 1:1` (the `package` declaration), so a suppression comment must be ON line 1 or on a non-existent line 0. Putting the comment before `package main` shifts the package declaration to line 2, and the finding follows it to line 2 — but now the comment is on line 1 (correct: line above).

   **This should work** based on the parser code. However, `go fmt` reformats `//cqrs-lint:ignore(P012,P013)` to `// cqrs-lint:ignore(P012,P013)` (space after `//`), and the parser's regex in `ParseSuppressions` (`parser.go:245-277`) appears to handle this via `strings.TrimSpace`. The actual failure in practice was that `golangci-lint`'s `godoclint` flags a comment before `package main` as a duplicate package comment.

   **The suppression mechanism works for cqrs-lint — but conflicts with other linters.** This is an ecosystem problem, not purely a cqrs-lint bug.

2. **Config-based suppression** — `.cqrs-lint.json` has no `"disabled-rules"` or `"exclude-rules"` key (see Bug 3 below).

3. **`--exclude` flag** — excludes paths, not rules.

#### Fix suggestions

**Option A: Cross-file tracking (ideal but complex)**

Track that `sqlite.New()` is called from `demo.go`, then follow the `sqlite.New` definition to `storage.go` and check for PRAGMA/WAL there. This requires call-graph analysis or at least function-definition tracking.

**Option B: Recognize wrapper patterns (pragmatic)**

If the `sqlite.New` call passes a DSN string (not a `*sql.DB`), assume WAL/busy_timeout are applied inside the constructor. Only flag when the caller opens the DB directly with `sql.Open` and doesn't apply PRAGMAs. This matches how most well-structured Go projects work (they wrap `sql.Open` in a storage package).

**Option C: Only fire on the file that contains the PRAGMA-less open (simplest)**

If `sqlite.New` is a call to a user-defined function (not `sql.Open` directly), don't flag the caller file. Flag the file that DEFINES the wrapper if it lacks PRAGMAs. This is still per-file but at least targets the right file.

**Option D: Config-based rule disabling (see Bug 3)**

Let users disable P012/P013 globally in `.cqrs-lint.json` when they know their storage layer handles it.

---

## Part 2: Missing Features

### Feature 1: No config-level rule disabling

**Severity: MEDIUM** — forces inline comments scattered across the codebase, which conflict with other linters.

#### Current state

The `.cqrs-lint.json` config supports:

```json
{
  "min-severity": "info",
  "min-confidence": "low",
  "features": { ... },
  "preset": "",
  "rules": {
    "external-api-struct-prefixes": ["..."]
  },
  "health": { "info-cap": 20 }
}
```

There is **no way to disable specific rules globally**. The only mechanisms are:

1. `//cqrs-lint:ignore(RULE)` — inline, per-file, one finding at a time
2. `//cqrs-lint:ignore-start` / `ignore-end` — block, per-file
3. `--only C001,C002` — inclusive filter (selects only listed rules; cannot say "all except E003")
4. `--fast` — runs only critical correctness rules

There is no `"disabled-rules": ["P012", "P013", "E003"]` or `"exclude-rules": ["F009"]`.

#### Why this matters

For a project like bank-sync:

- **P012/P013** are false positives (WAL is applied in a wrapper). I need to suppress them on 2 files. But inline suppression conflicts with `godoclint`.
- **E003** (package mixing) is a deliberate architecture choice. I'd like to suppress it project-wide.
- **F009/F015/F017** are not applicable to a CLI tool. I'd like to suppress them project-wide.

Without config-level disabling, I'm forced to add inline comments to every file, or accept unsuppressed findings in the output.

#### Suggested config format

```json
{
	"disabled-rules": ["P012", "P013"],
	"disabled-rules-reason": "WAL + busy_timeout are applied in internal/storage/sqlite/storage.go via PRAGMA",
	"rule-overrides": {
		"E003": { "severity": "info" },
		"F009": { "enabled": false, "reason": "TickerScheduler is intentional (see ADR)" }
	}
}
```

Or simpler:

```json
{
	"rules": {
		"disable": ["P012", "P013", "F009", "F015", "F017"],
		"external-api-struct-prefixes": ["..."]
	}
}
```

The `rules` key already exists in the config — just needs a `disable` sub-key.

#### Complementary: `--exclude-rules` CLI flag

```bash
cqrs-lint . --exclude-rules P012,P013
```

This mirrors `--only` but is exclusive instead of inclusive. Useful for CI where you want to run "everything except these known false positives."

---

### Feature 2: F009, F015, F017 missing feature-profile gating

**Severity: MEDIUM** — fires on projects where the modules are deliberately not used.

#### Current state

The feature-profile system is excellent for F004 (Prometheus) and F013 (transport) — both check `ctx.FeatureProfile.HasServer` and return `nil, nil` when the project has no server. But three adoption rules have **no feature-profile awareness**:

| Rule              | What it checks                         | Feature gate? | Fires on CLI? |
| ----------------- | -------------------------------------- | ------------- | ------------- |
| F004 (Prometheus) | Server-mode without metrics            | `HasServer`   | No            |
| F009 (scheduling) | Timers without scheduling module       | **None**      | **Yes**       |
| F013 (transport)  | Manual HTTP without transport module   | `HasServer`   | No            |
| F015 (metaengine) | 10 queries without metaengine          | **None**      | **Yes**       |
| F017 (dedup)      | Bus subscriptions without dedup module | **None**      | **Yes**       |

#### Why these should be gated

- **F009** fires on `TickerScheduler` in bank-sync. The scheduling module (`scheduling.Scheduler` with `TimerStore`) is designed for durable, crash-safe deadline management — bank-sync is a local CLI tool where a hand-rolled timer is appropriate and documented. F009 should check whether the project uses `watermill.EventBus` with async delivery (where dedup matters) vs a synchronous bus (where events are delivered in-process, no duplicates possible).

- **F015** fires because bank-sync has 10 query registrations but no metaengine. The metaengine is a cost-based storage planner — overkill for a local SQLite tool with simple queries. F015 should check whether the project uses a storage backend that would benefit from planning (Postgres with large datasets) vs SQLite (local, small data).

- **F017** fires because bank-sync subscribes to events without the dedup module. bank-sync uses a synchronous bus (`BlockPublishUntilSubscriberAck`) — events are delivered exactly once, in-process. Duplicates are impossible. F017 should check whether the bus is synchronous or asynchronous before suggesting dedup.

#### Fix suggestion

```go
// F009 — only suggest scheduling module for async/server projects
if !ctx.FeatureProfile.HasServer && !ctx.FeatureProfile.HasAsyncBus {
    return nil, nil
}

// F015 — only suggest metaengine for non-SQLite projects or projects with many aggregate types
if ctx.FeatureProfile.Store == analyzer.StoreSQLite {
    return nil, nil  // SQLite is local; metaengine overhead is not worth it
}

// F017 — only suggest dedup for async delivery
if !ctx.FeatureProfile.HasAsyncBus {
    return nil, nil  // synchronous bus = exactly-once delivery, no dedup needed
}
```

This would require adding `HasAsyncBus` to the feature profile (detected by checking for `BlockPublishUntilSubscriberAck: false` or the absence of a message queue like NATS/Kafka).

---

### Feature 3: D007 (event.NewEvent → event.New) should support `--fix`

**Severity: MEDIUM** — the migration is purely mechanical and perfect for auto-fix.

#### What happened

D007 correctly identified that the project uses both `event.New` and `event.NewEvent`. The migration is:

| Old                                                            | New                                                                                           |
| -------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `event.NewEvent(type, streamID, streamType, version, payload)` | `event.New(type, streamID, streamType, version, payload, event.WithCodec(codec.JSONCodec{}))` |

This is a 1:1 text replacement (adding the `WithCodec` option for raw-byte payloads). It took me ~30 minutes to migrate 10 call sites across 7 files manually. `--fix` could do this in seconds.

#### Current auto-fix coverage

Only 3 rules support `--fix`: C001, C003, C006. The fix provider (`fix/provider.go`) uses `BeforeCode`/`AfterCode` substring matching.

#### Suggested implementation

D007 is a perfect candidate because:

1. The transformation is deterministic: `event.NewEvent(` → `event.New(`
2. For raw-byte payloads, append `event.WithCodec(codec.JSONCodec{})` as the last argument
3. For typed payloads (test helpers), remove the `json.Marshal` wrapper and pass the payload directly

The tricky part is (2) — detecting whether the payload is raw bytes or a typed value. The fix provider could use a heuristic: if the argument before the closing `)` is a `[]byte` variable (like `payload`, `data`, `raw`), add the codec option.

---

## Part 3: Quality-of-Life Improvements

### Improvement 1: Document suppression syntax in `--help`

The `--help` output lists flags but doesn't explain the inline suppression syntax. I discovered `//cqrs-lint:ignore(RULE)` by grepping my codebase for existing suppressions. The comma-separated syntax (`ignore(A001,E005)`) and block syntax (`ignore-start`/`ignore-end`) are even less discoverable.

**Suggestion:** Add a "Suppressions" section to `--help` output:

```
SUPPRESSIONS

  Inline (single rule):
    //cqrs-lint:ignore(C007) reason text

  Inline (multiple rules):
    //cqrs-lint:ignore(C007,A001) reason text

  Block:
    //cqrs-lint:ignore-start
    ...code...
    //cqrs-lint:ignore-end

  Block (specific rules):
    //cqrs-lint:ignore-start(C007,A001)
    ...code...
    //cqrs-lint:ignore-end
```

### Improvement 2: `cqrs-lint init --preset local-cli`

The `init` command creates a default `.cqrs-lint.json`. A `--preset` flag would generate a config tailored to the project type:

- **`local-cli`**: `server: false`, suppress F004/F009/F013/F015/F017, lower `min-severity` to `warning`
- **`library`**: `server: false`, `command-flow: read-only`, suppress E003/E016
- **`server`**: `server: true`, enable all adoption rules
- **`full-stack`**: everything on

This would eliminate the most common configuration churn for new consumers.

### Improvement 3: Stale suppression detection should also detect wrong rule IDs

The stale-suppression detector (`warning: stale suppression at infrastructure.go:275 — rule A016 does not fire here`) is excellent. But it only detects suppressions where the rule **no longer fires**. It should also detect:

- **Typo in rule ID**: `//cqrs-lint:ignore(PO12)` (letter O instead of zero) — this will never match any rule
- **Non-existent rule ID**: `//cqrs-lint:ignore(Z999)` — this rule doesn't exist
- **Suppressed rule that was renamed**: if P012 was renamed to P014 in a new version, old suppressions for P012 are stale

**Suggestion:** During analysis, collect all `//cqrs-lint:ignore(XYZ)` comments and verify that `XYZ` is a registered rule ID. If not, emit: `warning: suppression at file.go:N references unknown rule XYZ — possible typo or stale rule ID`.

### Improvement 4: `doctor` should suggest presets

The `doctor` command detects the feature profile and prints it. But it doesn't suggest which preset to use. For bank-sync, `doctor` detects:

```
store:         custom
command-flow:  commands
server:        true
```

But `server: true` is misleading — bank-sync has an HTMX dashboard (`http.Server`) but it's a local read-only tool, not a production server. The `server` detection triggers on `ListenAndServe` even for local dev servers.

**Suggestion:** `doctor` should print:

```
Suggested preset: local-cli
Reason: project uses ListenAndServe but has no net.Listener config, no TLS,
        and no health endpoint — consistent with a local dev server, not
        production. Use preset "local-cli" to suppress server-only rules.
```

### Improvement 5: `--health-score` should show the breakdown

The `--health-score` flag prints only the number. It would be more useful with a breakdown:

```
Health Score: 82/100 (Good)

Deductions:
  -12  WARNING ×3  P012/P013 SQLite WAL/busy_timeout (false positive — see storage.go)
  -4   WARNING ×2  E003 package mixing (intentional)
  -2   INFO ×4    F-series module suggestions (not applicable)

Run with --verbose for per-finding details.
```

### Improvement 6: `server` feature detection is too aggressive

The feature detector (`feature_detect.go:80-134`) sets `Server = true` when it sees `ListenAndServe`, `Serve`, or `NewServer`. This fires on local dev servers, dashboards, and tools that embed an HTTP server for convenience.

bank-sync has a local HTMX dashboard — it's not a production server. But because it calls `http.ListenAndServe`, `server: true` is detected, and F004 (Prometheus) + F013 (transport) fire (though they're then gated by `HasServer` and DO fire because `HasServer` is true).

**Suggestion:** Add a confidence heuristic: if the project has `ListenAndServe` but no `net.Listen`, no TLS config (`tls.Config`), no graceful shutdown pattern (`Shutdown` on `*http.Server`), and no health endpoint, downgrade `Server` to `ServerLocal` (a new value). Then `HasServer` returns false for `ServerLocal`, suppressing F004/F013.

Alternatively, let the user override in config:

```json
{
	"features": {
		"server": "local" // "local" = local dev server, "production" = real server
	}
}
```

---

## Part 4: What's Working Great (Acknowledgment)

To balance the criticism, these v0.2.x improvements were genuinely valuable:

| Improvement                                           | Impact                                                                                      |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **Generic call detection** (`unwrapSelector`)         | Fixed A017 and all detectors that scan generic API calls — was the #1 false positive source |
| **Closure handler tracing** (`*ast.FuncLit` params)   | Fixed E005/E007 — was the #2 false positive source (19 findings eliminated)                 |
| **Feature-profile system**                            | Eliminated F004/F013 for non-server projects — major noise reduction                        |
| **Context-aware heuristics** (A012, A015, A016, S002) | Each now checks domain context before firing — much smarter                                 |
| **Upcaster context detection**                        | A014/C005 no longer fire inside `schema.NewUpcaster` closures                               |
| **Stale suppression detection**                       | Caught a dead A016 suppression comment — very useful                                        |
| **Health score**                                      | Gives a single-number quality signal — great for CI gates                                   |
| **`doctor` command**                                  | Auto-detects feature profile and emits copy-pasteable JSON — excellent DX                   |
| **Block suppression** (`ignore-start`/`ignore-end`)   | Clean alternative to per-line comments for multi-line findings                              |
| **D002 external-API prefixes**                        | Lets consumers suppress false positives from external API structs                           |

The linter has gone from "39% signal-to-noise" (v0.1.0) to roughly "70% signal-to-noise" (v0.2.2). With the fixes proposed here (especially config-level rule disabling and cross-file SQLite detection), it would reach ~90%.

---

## Part 5: Prioritized Fix List

| Priority | Issue                                                         | Type                     | Effort                                                  | Impact                                               |
| -------- | ------------------------------------------------------------- | ------------------------ | ------------------------------------------------------- | ---------------------------------------------------- |
| **P0**   | B022 suggests non-existent `decider.CommandCausalityEnricher` | Bug (wrong text)         | Trivial (text fix)                                      | High — stops misleading users                        |
| **P0**   | P012/P013 cross-file blindness                                | Bug (per-file detection) | Medium (cross-file tracking or wrapper-aware heuristic) | High — 4 unsuppressable false positives per project  |
| **P1**   | Config-level rule disabling (`"disable": [...]`)              | Missing feature          | Low (config parsing + filter in run.go)                 | High — eliminates need for scattered inline comments |
| **P1**   | `--exclude-rules` CLI flag                                    | Missing feature          | Low (mirror `--only` logic)                             | Medium — CI-friendly                                 |
| **P1**   | F009/F015/F017 feature-profile gating                         | Enhancement              | Medium (add `HasAsyncBus` to profile, gate each rule)   | Medium — 3 fewer findings on CLI projects            |
| **P2**   | D007 auto-fix support                                         | Enhancement              | Medium (fix provider for event.NewEvent → event.New)    | Medium — saves 30 min per migration                  |
| **P2**   | Stale suppression: detect unknown rule IDs                    | Enhancement              | Low (collect ignore IDs, check against registry)        | Low — catches typos                                  |
| **P2**   | `--help` suppression syntax docs                              | Docs                     | Trivial                                                 | Low — discoverability                                |
| **P3**   | `cqrs-lint init --preset`                                     | Enhancement              | Low (template configs)                                  | Low — convenience                                    |
| **P3**   | `--health-score` breakdown                                    | Enhancement              | Low (format already-computed data)                      | Low — UX polish                                      |
| **P3**   | `server` detection too aggressive                             | Enhancement              | Medium (confidence heuristic or `ServerLocal` value)    | Medium — better feature profile accuracy             |

---

## Appendix: Full v0.2.2 Finding List (bank-sync)

```
SUPPRESSED (7 inline comments):
  B022   infrastructure.go:153  Custom enricher (false positive — event.CommandCausalityEnricher IS the canonical enricher)
  C015   infrastructure.go:345  MemoryStore.Close() returns void
  E016   infrastructure.go:92   No HealthCheck (local CLI, no k8s)
  E017   main.go:79             No GracefulClose (per-command shutdown instead)
  F017   infrastructure.go:306  No dedup module (synchronous bus, idempotent domain)
  A032   events.go:91           RelatedTransactionID is string (cross-provider bank API ref)
  C016   main.go:153            Detached context (intentional for diagnostics)

REMAINING UNSUPPRESSED (9 — all accepted):
  WARNING  E003   go.mod:1             Package mixes 3 CQRS concerns (architecture choice)
  WARNING  P012   demo.go:1            SQLite without WAL (FP — applied in storage.go)
  WARNING  P013   demo.go:1            SQLite without busy_timeout (FP — applied in storage.go)
  WARNING  P012   helpers.go:1         SQLite without WAL (FP — applied in storage.go)
  WARNING  P013   helpers.go:1         SQLite without busy_timeout (FP — applied in storage.go)
  INFO     E009   adapter.go:2         No HTTP transport (local CLI)
  INFO     F004   adapter.go:2         No Prometheus (local CLI)
  INFO     F009   scheduler.go:101     No scheduling module (intentional hand-rolled timer)
  INFO     F013   server.go:182        No transport module (uses cqrs-htmx)
  INFO     F015   adapter.go:2         No metaengine (SQLite, 10 queries)
  INFO     B004   commands.go:40,105,157  Commands with 7-8 fields (cqrs-gen not adopted)
  INFO     B010   catalog.go:56        6 catalog.Event calls (cqrs-gen not adopted)
  INFO     B018   infrastructure.go:308  3+ bus.Subscribe calls (stylistic)

FIXED THIS SESSION (no longer appear):
  D007    event.NewEvent → event.New         Migrated 10 call sites
  C023    Ignored Shutdown/Close/Stop errors  Fixed 5 call sites
  A016    Stale suppression comment           Removed
```

**FP = False Positive** — detector fires but the condition is already satisfied

---

## Resolution (2026-08-03)

9 of 12 items implemented per round-2 review (B022 text fixed, suppression parser fixed, P012/P013 fixed, config disabling, `--exclude-rules`, C036 mitigated, S006 fixed). 3 items explicitly deferred:

- **D007 auto-fix** — `--fix` flag shipped for D007 but the event.NewEvent→event.New transformation needs more work
- **F009/F015/F017 feature-profile gating** — DONE in `07-00` (HasAsyncBus detection)
- **ServerLocal heuristic** — DONE (TLS detection with `tls.Listen` gating)
