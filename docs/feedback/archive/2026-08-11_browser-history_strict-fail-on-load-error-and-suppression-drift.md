# Feedback: cqrs-lint `--strict` should hard-fail on load errors, exclude-glob docs, and suppression-drift detection

> **From**: browser-history consumer (5-module Go workspace, 99 files, 23 inline suppressions)
> **Date**: 2026-08-11
> **Priority**: High (1 functional bug, 2 UX improvements)
> **Status: ✅ ALL 4 ITEMS RESOLVED 2026-08-11** — `--strict` hard-fail
> (`isStrictMode`, `run.go:225`), exclude globs (`matchExcludePattern`,
> `filters.go`), suppression-drift audit (`AuditSuppressions`, `doctor_audit.go`),
> doctor multi-module config (`mergeMostPermissiveProfile`). Commits `1551bd396`,
> `515b50bbf`. See CHANGELOG `[Unreleased]` and
> `docs/status/2026-08-11_15-58_cqrs-lint-feedback-strict-globs-audit-suppressions.md`.

---

## 1. BUG: `--strict` silently skips broken packages and reports "Clean!"

### What happened

The `api` module (the most important module — HTTP handlers, middleware, server wiring) had a compile error (`req redeclared in this block`). Running `cqrs-lint --strict --verbose --show-suppressed` produced:

```
WARNING: 1 package(s) failed to load; analysis is partial.
Use --verbose for details or --strict to fail on any load error.

Analyzed 20 files in 1.225s
9 finding(s) suppressed by inline comments
...
No findings. Clean!
```

**"No findings. Clean!"** — while the most critical module was completely invisible. The `WARNING` line was buried between "Analyzed 20 files" and the findings count. A CI gate scanning for "Clean!" would pass.

After fixing the compile error, the same command analyzed **99 files** (not 20) and found **27 findings** (not 9). The entire discrepancy was the silently-skipped `api` module.

### Expected behavior

`--strict` should **exit non-zero** when any package fails to load. The current behavior — "analyze what you can, report the rest" — is dangerous because:

1. CI gates that check the exit code or scan for "Clean!" will pass on a partial analysis
2. The WARNING is easy to miss in CI logs (1 line among hundreds)
3. The file count ("Analyzed 20 files") gives no indication that 79 files were skipped
4. The suppression count ("9 finding(s) suppressed") is also wrong — should be 23

### Concrete data

| Metric | Broken (before fix) | Correct (after fix) |
|--------|--------------------:|--------------------:|
| Files analyzed | 20 | 99 |
| Total findings | 9 | 27 |
| Suppressed | 9 | 23 |
| Unsuppressed | 0 | 4 |
| Exit code | 0 | 0 |
| Output | "No findings. Clean!" | "No findings. Clean!" |

### Reproduction

1. Take any multi-package Go workspace using cqrs-lint
2. Introduce a compile error in the largest/most important package (e.g., duplicate `var` declaration)
3. Run `cqrs-lint --strict --verbose --show-suppressed`
4. Observe: partial analysis, buried warning, "Clean!" output, exit code 0

### Suggested fix

Two options (ideally both):

**Option A (exit code):** When `--strict` is set, any `Load errors (N > 0)` should cause `os.Exit(1)`. The current text says "Use --strict to fail on any load error" but `--strict` doesn't actually do this.

**Option B (output):** When packages fail to load, replace "No findings. Clean!" with a loud banner:
```
⚠ INCOMPLETE ANALYSIS — 1 package(s) failed to load (79 files skipped)
  /path/to/api: ./ingest.go:78:6: req redeclared in this block

Results below are from 20 files only. Do not trust "Clean!" status.
```

---

## 2. UX: `exclude` glob pattern — path globs don't work, only filename globs

### What happened

Created `.cqrs-lint.json` to exclude generated templ files:

```json
{
  "exclude": "**/*_templ.go"
}
```

**Did not work.** The `*_templ.go` files were still being analyzed (3 C014 findings appeared in `timeline_templ.go`).

Had to discover by trial and error that bare filename globs work:

```json
{
  "exclude": "*_templ.go,*_string.go"
}
```

### Expected behavior

Either:
1. **Document the glob syntax** in `cqrs-lint explain` output (currently says `"exclude": "Paths to exclude (comma-separated glob patterns)"` — doesn't clarify that these are filename patterns, not path patterns)
2. **Support both** `**/*_templ.go` (path glob) and `*_templ.go` (filename glob) like `.gitignore` does

### Impact

Any consumer using [a-h/templ](https://github.com/a-h/templ) (a major Go templating library, used by the templ-components ecosystem) will have generated `*_templ.go` files that produce false-positive findings. This is a common pattern. The config syntax should either "just work" with path globs or clearly document that only filename globs are supported.

---

## 3. FEATURE REQUEST: Suppression-drift detection

### Context

During this session's audit of 23 inline suppressions, I found that **4 suppression comments were factually inaccurate**:

```go
//cqrs-lint:ignore(A017,B025) aggregates are 1-event-per-stream (snapshot and StateCache provide zero benefit for single-event streams)
```

This claim ("1-event-per-stream") was **wrong**. The actual stream lengths:
- BrowserHistory: max 2 events per stream (not 1)
- DomainSettings/DomainOverride/GoalSettings: technically **unbounded** (tag→retag→untag cycles accumulate), but trivially small in practice (<5)

The suppression was still **correct** (snapshot provides zero value for <10-event streams), but the human-written justification had drifted from reality. This is dangerous: a future developer reading the comment would trust the reasoning and extend the pattern incorrectly.

### Suggested feature

Add a `cqrs-lint doctor --audit-suppressions` mode that:

1. Collects all inline suppressions
2. Re-evaluates whether the finding is still a false positive or has been fixed upstream
3. Flags suppressions whose **rule heuristic has changed** since the suppression was written (e.g., if A017 now excludes <10-event streams by default, the suppression is no longer needed)
4. Reports suppressions with **stale reasoning** (harder — but could flag comments that reference code patterns that no longer exist at the cited location)

### Why this matters

Suppressions are silent debt. They accumulate over time and are never re-audited unless someone manually reads every one. A periodic "suppression health check" would surface:
- Suppressions for bugs that have been fixed (remove the suppression)
- Suppressions for heuristics that have been improved (remove the suppression)
- Suppressions with wrong reasoning (fix the comment before it misleads someone)

---

## 4. Minor: `doctor` suggested config is stale

### What happened

`cqrs-lint doctor` suggests pinning the auto-detected feature profile:

```json
{
  "features": {
    "command-flow": "read-only",
    "server": false,
    "soft-delete": false,
    "tracing": "off",
    "snapshot": "off"
  }
}
```

But the **workspace-level** auto-detection sees the root module (which has no CQRS code) and reports `command-flow: read-only, server: false, tracing: off`. The actual `api` module (detected separately in "PER-MODULE PROFILES") has `server: true, tracing: on, command-flow: commands, transport: true, async-bus: true`.

Pinning the workspace-level profile would silence correct findings in the `api` module (e.g., E003 server checks, tracing checks) because the pinned values say "no server, no tracing."

### Suggested fix

Either:
1. The suggested config should use the **most permissive** profile across all modules (so nothing gets silenced)
2. Or `doctor` should warn that pinning the workspace profile may hide findings in sub-modules with richer profiles
3. Or support per-module `.cqrs-lint.json` files (monorepo inheritance)

---

## Consumer Context

- **Project**: browser-history — Go multi-module monorepo (go.work, 7 modules)
- **SDK version**: go-cqrs-lite v4.0.x (event, command, decider, query, projection, middleware, idempotency, watermill, otel, storage, id)
- **Scale**: 99 Go files, 23 inline suppressions across 12 files
- **Profile**: Multi-module workspace — `api` module has `server=true, tracing=on, command-flow=commands, async-bus=true`; root/domain/storage/projection modules are library-only
- **Generated files**: Uses a-h/templ (generates `*_templ.go` files that should be excluded from linting)
- **CI gate**: `nix run .#check-cqrs-lint` (0 unsuppressed findings enforced)
