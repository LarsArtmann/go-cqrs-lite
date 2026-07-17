# cqrs-lint — Consumer Feedback (Cyberdom)

**Consumer:** [Cyberdom](https://github.com/LarsArtmann/Cyberdom) — AI-powered Discord agent with scheduled briefings, event-sourcing via go-cqrs-lite, and an extensible tool system
**Version used:** go-cqrs-lite v4.0.1 (event, id, storage, watermill)
**lint version:** `cqrs-lint v0.2.0` (installed binary); source reviewed at `cmd/cqrs-lint/` master (`v0.2.1`)
**Date:** 2026-07-17

---

## Executive Summary

Ran cqrs-lint against the Cyberdom codebase (13 Go files) across two sessions: first
with v0.1.0 (7 findings), then re-ran with v0.2.0 after upgrading. The tool is genuinely
useful — it caught a deprecated API call, a doc/code version mismatch, and a JSON casing
inconsistency, all of which were legitimate and fixed immediately.

After the v0.1.0 fixes, v0.2.0 reports **3 remaining findings** (1 warning, 2 info),
all of which are intentional design decisions documented in TODO_LIST.md. However, the
v0.2.0 run also surfaced **three bugs in the linter itself** — including a showstopper
where `cqrs-lint init` generates a config file that breaks all subsequent runs.

| Category                                     | Count | Action taken                                             |
| -------------------------------------------- | ----- | -------------------------------------------------------- |
| **Valid findings (fixed in v0.1.0)**         | 3     | All fixed — A014, D002, D005                             |
| **False positive (v0.1.0, fixed in v0.2.0)** | 1     | A016 — no command dispatcher exists; now gated correctly |
| **Intentional design decisions**             | 3     | Documented in TODO_LIST.md — S003, A009, B014            |
| **Linter bugs found**                        | 3     | Reported below with root-cause analysis and fixes        |

**Signal-to-noise ratio: 100%.** Every finding was either valid, a known false positive
(now fixed), or an intentional decision. No noise. This is a marked improvement over
the DiscordSync (39% SNR) and bank-sync (41% SNR) runs, though those codebases are
significantly larger and exercise more rules.

---

## Part 1: Valid Findings — Fixed (v0.1.0)

### A014: Deprecated API usage — `event.NewEvent`

**Verdict: Correct. Fixed.**

**File:** `internal/events/payloads.go:146`

The linter correctly identified that `event.NewEvent` is deprecated in favor of
`event.New`. Since the payload was already marshaled to `[]byte` before the call,
the migration was a drop-in rename — `event.New` accepts `[]byte` directly and
uses it as-is (JSON encoding stamped automatically).

```go
// Before
core, err := event.NewEvent(
    eventType, aggregateID, aggregateType, version, data,
    event.WithSource("cyberdom"),
)

// After
core, err := event.New(
    eventType, aggregateID, aggregateType, version, data,
    event.WithSource("cyberdom"),
)
```

**Impact:** Zero behavior change. Verified with `go build ./... && go test ./...`.

### D002: Mixed JSON key casing

**Verdict: Correct. Fixed.**

**File:** `discord/bot.go:23-29`

The `ReconnectConfig` struct used snake_case JSON tags (`max_retries`,
`initial_delay`) while every event payload struct in `internal/events/payloads.go`
uses camelCase (`channelId`, `authorId`, `toolCalls`, etc.). The struct also
carries `koanf` tags (snake_case, for YAML config loading) — those were left
untouched since YAML convention is separate from JSON convention.

```go
// Before
MaxRetries   int           `json:"max_retries"   koanf:"max_retries"`

// After
MaxRetries   int           `json:"maxRetries"   koanf:"max_retries"`
```

**Impact:** The struct is never JSON-serialized in production (only loaded via
koanf/YAML). Zero behavior change.

### D005: Documentation version mismatch

**Verdict: Correct. Fixed.**

**File:** `AGENTS.md`

The documentation referenced `go-cqrs-lite v3.7.4` and `storage/v2` while
`go.mod` had already migrated to v4 modules. Updated all references:

- Tech stack table: `go-cqrs-lite v3.7.4` → `go-cqrs-lite v4`
- Event store description: `go-cqrs-lite/storage/v2` → `go-cqrs-lite/storage/v4`
- Section 8 header and module list: all `/v3` → `/v4`
- Also fixed `go-error-family v0.6.1` → `v0.7.0` (same drift pattern, adjacent line)

---

## Part 2: False Positive — Fixed in v0.2.0

### A016: Command dispatcher lacks idempotency middleware

**v0.1.0 verdict: FALSE POSITIVE. Resolved in v0.2.0.**

Cyberdom has no command dispatcher — AGENTS.md explicitly states commands and
queries were removed; the event bus alone handles all communication. In v0.1.0,
A016 fired because it matched `eventBus.Use(...)` as a dispatcher middleware call.

In v0.2.0, the `FeatureProfile.CommandFlow` detection correctly identifies
Cyberdom as `command-flow: read-only`, and A016 no longer fires. This is the
same fix described in the DiscordSync feedback (resolution log item #5), and it
works correctly here.

---

## Part 3: Intentional Design Decisions — No Action Taken

### S003: Event store without signing middleware

**Verdict: INTENTIONAL. Deferred — documented in TODO_LIST.md #33.**

Cyberdom is a single-process Discord bot with a local SQLite database and an
in-process Watermill GoChannel event bus. Event signing (`signing.SignMiddleware`

- `signing.VerifyMiddleware`) protects against tampering in multi-process or
  networked event bus deployments. For this architecture:

1. The event store is a local SQLite file — an attacker with write access to the
   DB also has access to the signing key (co-located in the same process/config).
2. The event bus is in-process (`GoChannel`) — there is no network hop where
   events could be intercepted or injected.
3. `signing.VerifyMiddleware` passes unsigned events through by default (mixed
   stream support), so adding it without `RequireSignatureMiddleware` provides
   no enforcement.

Signing becomes meaningful when either the bus is networked or the key is managed
by an external KMS. Both are out of scope for the current architecture.

### A009: No `stack/` preset

**Verdict: INTENTIONAL. Deferred — documented in TODO_LIST.md #34.**

The current event bus wiring is ~30 lines, tested, and readable. Cyberdom's
event bus setup includes a custom persistence middleware
(`events.NewPersistenceMiddleware`) that wraps `store.Save` with event bus
middleware chaining. Migrating to a `stack/` preset would require either:

1. Replacing the custom middleware with the stack's built-in persistence (losing
   the domain-specific error wrapping and logging).
2. Combining the stack preset with custom middleware (adding complexity instead
   of removing it).

The ROI is low — the explicit wiring is clear and hasn't caused maintenance
issues. The linter's own suggestion acknowledges this: "keep custom wiring if you
need full control."

### B014: Missing OTel tracing middleware

**Verdict: INTENTIONAL. Deferred — already tracked as TODO_LIST.md #32.**

Cyberdom has no tracing infrastructure — no tracer, no trace provider, no spans.
It does use OTel for **logging** via the `otelslog` bridge (`internal/logger/`),
but that is a separate concern from distributed tracing. Adding tracing middleware
to the event bus without a tracer provider or trace exporter would be dead code.

This is already tracked as a P3 task ("Add OpenTelemetry tracing spans for event
bus operations") and will be addressed when tracing infrastructure is added
holistically.

---

## Part 4: Linter Bugs Found in v0.2.0

### Bug 1 (SHOWSTOPPER): `cqrs-lint init` generates a config that breaks all subsequent runs

**Severity: CRITICAL — makes the tool unusable after running `init`.**

#### What happens

```bash
$ cqrs-lint init
Created .cqrs-lint.json with default settings

$ cat .cqrs-lint.json
{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "exclude": [],
  "only": "",
  "features": {},
  "preset": ""
}

$ cqrs-lint
Error creating CLI: short="Domain-aware linter for go-cqrs-lite consumers",
initializing CLI "cqrs-lint": failed to load config file: loading config file:
failed to load config file: loading ".cqrs-lint.json": failed to parse config
file: json: cannot unmarshal array into Go struct field AppConfig.exclude of type string
```

Every `cqrs-lint` command fails until `.cqrs-lint.json` is manually deleted.

#### Root cause

The config template in `init.go:18` emits `"exclude": []` (JSON array), but the
`AppConfig.Exclude` field in `main.go:48` is typed `string`:

```go
// init.go:14-23
const configTemplate = `{
  ...
  "exclude": [],          // ← array literal
  ...
}`

// main.go:48
Exclude string `default:"" flag:"exclude" help:"Exclude paths (comma-separated)"`
```

The `--exclude` flag is documented as "comma-separated" (a string like
`"vendor,test"`), confirming the field type is correct and the template is wrong.

#### Fix

```go
// init.go:18 — change array to empty string
"exclude": "",
```

#### Additional issue: `init --dry-run` writes the file anyway

The `--dry-run` global flag is documented as "Show fixes without applying." The
`init` subcommand ignores it entirely — `os.WriteFile` is called unconditionally
at `init.go:34`. This is arguably a design choice (`--dry-run` applies to
auto-fixes, not to config creation), but the UX is surprising. Consider either:

1. Honoring `--dry-run` in `init` (print the template to stdout without writing).
2. Documenting that `--dry-run` only applies to `--fix` operations.

---

### Bug 2: `doctor` reports `tracing: on` when no tracing exists

**Severity: MEDIUM — misleading output, contradicts B014 findings.**

#### What happens

```bash
$ cqrs-lint doctor
Detected go-cqrs-lite feature profile:

store:         custom
command-flow:  read-only
server:        true
soft-delete:   false
tracing:       on        # ← wrong
snapshot:      off
```

Cyberdom has **no tracing** — no tracer, no trace provider, no spans, no
`otel/sdk/trace` imports. It only imports OTel for logging:

```
internal/logger/logger.go:
  "go.opentelemetry.io/contrib/bridges/otelslog"
  "go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
  "go.opentelemetry.io/otel/log/global"
  "go.opentelemetry.io/otel/sdk/log"
```

#### Root cause

`feature_detect.go:143-150` sets `TracingOn` if **any** OTel import is found,
without distinguishing tracing imports from logging/metrics imports:

```go
// feature_detect.go:143-150
if fp.Tracing == TracingUnknown {
    if hasOTelImport {        // ← matches otelslog, otel/log, etc.
        fp.Tracing = TracingOn
    } else {
        fp.Tracing = TracingOff
    }
}
```

The `hasOTelImport` flag is set during import scanning (Pass 1) based on any
import path containing `go.opentelemetry.io/otel`. This conflates:

- **Tracing:** `otel/trace`, `otel/sdk/trace`, `otel/sdk/trace/...`
- **Logging:** `otel/log`, `otel/sdk/log`, `otel/exporters/stdout/stdoutlog`,
  `otel/contrib/bridges/otelslog`
- **Metrics:** `otel/metric`, `otel/sdk/metric`, `otel/exporters/...`

#### Fix

Restrict the tracing detection to tracing-specific import paths:

```go
func isTracingImport(path string) bool {
    return strings.HasPrefix(path, "go.opentelemetry.io/otel/trace") ||
        strings.HasPrefix(path, "go.opentelemetry.io/otel/sdk/trace")
}
```

Or, more precisely, check for `otel/sdk/trace` (the SDK is always required for
actual tracing setup — bridges/exporters alone don't imply tracing).

#### Impact

The false `tracing: on` also creates a **contradiction with B014**: `doctor`
says tracing is on, but B014 fires saying the event bus lacks OTel tracing
middleware. The user sees two contradictory messages about the same concern.
B014 is correct (no tracing middleware is wired); `doctor` is wrong.

---

### Bug 3: B014 ignores `FeatureProfile.Tracing` — split-brain with `doctor`

**Severity: LOW — correct finding, but architecturally inconsistent.**

#### What happens

B014 fires on Cyberdom (correctly — no tracing middleware is wired). But
`doctor` reports `tracing: on` (Bug 2). These two subsystems disagree about
whether tracing exists in the project.

#### Root cause

B014 (`b011_b014.go:165-236`) does its own AST scan for `EventTracing` /
`CommandTracing` / `NewOTelBundle` calls, completely bypassing
`ctx.FeatureProfile.Tracing`. The detector correctly finds no tracing middleware
calls, so the finding itself is valid. The problem is the architectural split:

- `feature_detect.go`: "tracing is on" (any OTel import)
- B014: "no tracing middleware" (AST scan for specific calls)

#### Fix (after Bug 2 fix)

Once `feature_detect.go` correctly distinguishes tracing imports from
logging/metrics imports, both subsystems will agree. If `fp.Tracing == TracingOff`,
B014 should still fire (it's the one that detects the gap). If
`fp.Tracing == TracingOn` but B014 finds no middleware calls, that's a different
signal worth investigating (tracing imports without bus wiring).

No code change needed in B014 itself — fixing Bug 2 resolves the contradiction.

---

## Part 5: Inline Suppression — Available in source, not in v0.2.0 binary

The DiscordSync feedback documented the `//cqrs-lint:ignore(RULE)` suppression
mechanism. The source code at master (`v0.2.1`) fully implements this:

- `pkg/suppression/parser.go` — parses `//cqrs-lint:ignore(S003) reason` comments
- `main.go:201` — wires `NewSuppressionFilter()` into the pipeline
- `main.go:224-253` — splits suppressed/active findings, prints suppression count
- `main.go:53` — `--show-suppressed` flag for auditing suppressed findings

However, **none of this is available in the installed v0.2.0 binary**:

```bash
$ cqrs-lint --show-suppressed
Unknown flag: --show-suppressed.
```

The `//cqrs-lint:ignore(S003)` comment has no effect when running v0.2.0. This
is not a bug — it's a feature that exists in the source but hasn't been released
yet. When v0.2.1 ships, Cyberdom will adopt suppressions for the S003 finding.

**Usability note for the suppression mechanism** (from reading the source):
`checkSuppressionInFile` in `parser.go:124` checks the finding's own line and
the line above. This works well for statement-level findings but is awkward for
function-level findings like S003 (which points to a `store.Save()` call deep
inside a middleware closure). Consider also checking the enclosing function
declaration or supporting a file-level `//cqrs-lint:ignore(S003)` at the top of
the file.

---

## Part 6: What v0.2.0 Got Right

### FeatureProfile + `doctor` command

The `doctor` command is an excellent addition. It makes the linter's internal
model transparent — you can see exactly what it detected about your project
before it applies rules. The feature profile concept (`store`, `command-flow`,
`server`, `soft-delete`, `tracing`, `snapshot`) is the right abstraction for
gating rules to reduce false positives.

### A016 fix via `command-flow: read-only`

This is the single biggest improvement for Cyberdom. In v0.1.0, A016 fired
incorrectly (no command dispatcher exists). In v0.2.0, the `CommandFlowReadOnly`
profile correctly suppresses it. The fix generalizes to any event-sourcing-only
system.

### Accurate findings with actionable suggestions

Every finding that fired on Cyberdom was either valid (A014, D002, D005) or an
intentional decision (S003, A009, B014). The suggestions were specific enough to
act on immediately — the deprecated API name, the exact JSON tags to change, the
version string to update. No vague "consider refactoring" noise.

---

## Summary Table

| Rule | Severity | v0.1.0 | v0.2.0 | Verdict            | Action                                             |
| ---- | -------- | ------ | ------ | ------------------ | -------------------------------------------------- |
| A014 | WARNING  | Fired  | —      | **VALID**          | Fixed — `event.NewEvent` → `event.New`             |
| D002 | INFO     | Fired  | —      | **VALID**          | Fixed — JSON tags aligned to camelCase             |
| D005 | WARNING  | Fired  | —      | **VALID**          | Fixed — AGENTS.md updated v3 → v4                  |
| A016 | WARNING  | Fired  | Gone   | **FALSE POSITIVE** | Fixed in v0.2.0 — `command-flow: read-only` gating |
| S003 | WARNING  | Fired  | Fired  | Intentional        | Deferred — TODO #33, no value for local SQLite     |
| A009 | INFO     | Fired  | Fired  | Intentional        | Deferred — TODO #34, custom middleware wiring      |
| B014 | INFO     | Fired  | Fired  | Intentional        | Deferred — TODO #32, no tracer infrastructure      |

---

## Suggestions for cqrs-lint Improvement

Ranked by impact:

1. **(CRITICAL) Fix `init` config template** — `"exclude": []` → `"exclude": ""`.
   This is a one-line fix that prevents every `init` user from hitting a fatal
   config parse error on their next run. See Bug 1.

2. **(MEDIUM) Distinguish tracing imports from logging/metrics imports in
   `doctor`** — `hasOTelImport` is too broad; it matches `otelslog` and
   `otel/sdk/log` as tracing. Check for `otel/sdk/trace` specifically. See Bug 2.

3. **(LOW) Support file-level suppression** — `//cqrs-lint:ignore(S003)` on the
   first line of a file (or above the package declaration) should suppress all
   findings of that rule in the file. The current line + line-above check is
   too narrow for function/struct-level findings. (Already in v0.2.1 source;
   consider extending before release.)

4. **(LOW) `init --dry-run` should not write files** — Either honor the flag
   (print template to stdout) or document that `--dry-run` only applies to
   `--fix`. See Bug 1 additional issue.

5. **(NICE-TO-HAVE) `doctor` should show the health score** — Currently `doctor`
   and `--health-score` are separate invocations. Combining them (or adding
   `--with-health-score` to `doctor`) would give a single-command project
   overview.
