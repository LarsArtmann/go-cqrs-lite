# cqrs-lint — Consumer Feedback (cqrs-htmx)

**Consumer:** [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) — Go library (SDK) that makes go-cqrs-lite usable with HTMX, templ, and Casbin. Multi-module Go workspace (12 modules under one `go.work`).
**Version used:** go-cqrs-lite v4.0.x (command/event/id/query/transport/stack modules)
**lint version:** `cqrs-lint v0.2.0` (`/home/lars/go/bin/cqrs-lint`, built 2026-07-17)
**Date:** 2026-07-17

---

## Executive Summary

`cqrs-lint v0.2.0` silently fails on projects that do not compile, and reports
the failure as "no CQRS usage found" instead of surfacing the real error. Worse,
the two diagnostic commands **disagree with each other** on the same broken
project, and both exit 0:

| Command               | Output                                           | Exit |
| --------------------- | ------------------------------------------------ | ---- |
| `cqrs-lint`           | `No Go files importing go-cqrs-lite found...`    | 0    |
| `cqrs-lint doctor`    | Prints a full feature profile (`postgres`, etc.) | 0    |
| `cqrs-lint --verbose` | Same "No Go files..." message, no extra detail   | 0    |

The project **does** import go-cqrs-lite (50+ source files across 12 modules).
The real problem is that the project's `go build ./...` fails with a module
resolution error (`invalid version: unknown revision 000000000000` — the known
go-cqrs-lite v4.0.0 publishing bug). `cqrs-lint` loads packages via
`go/packages`, some modules fail to resolve, and the loader **discards the
errors silently**, producing an empty analysis context.

The net effect: a user runs the linter on a broken build, gets a confident
"nothing to lint," and wastes time chasing phantom "missing import" problems
instead of being told "your project doesn't compile — here's why."

**Severity: HIGH.** Silent failure of a diagnostic tool erodes trust. The whole
point of `cqrs-lint` and `cqrs-lint doctor` is to tell the user the truth about
their project. When the project is broken, that is the **most important** thing
to report — and currently it is the one thing guaranteed never to be reported.

---

## Part 1: Reproduction

### Environment

```
cqrs-htmx (master, 2026-07-17)
go 1.26.4, GOEXPERIMENT=jsonv2
12 modules in go.work; root + usermgmt + adminui + ...
```

The project does not currently compile due to the upstream go-cqrs-lite
publishing bug (documented in cqrs-htmx's own `AGENTS.md`):

```
$ go build ./...
command/v4@v4.0.0 requires
  dispatcher/v4@v4.0.0-00010101000000-000000000000: invalid version: unknown revision 000000000000
(exit 1)
```

### Symptom 1 — `cqrs-lint` lies

```
$ cqrs-lint
No Go files importing go-cqrs-lite found. Nothing to lint.
(exit 0)
```

Reality: `grep -rn "github.com/larsartmann/go-cqrs-lite" --include="*.go"` returns
50+ matches across `usermgmt/`, root, `adminui/`, etc.

### Symptom 2 — `cqrs-lint doctor` lies differently

```
$ cqrs-lint doctor
Detected go-cqrs-lite feature profile:

store:         postgres
command-flow:  read-only
server:        false
soft-delete:   false
tracing:       off
snapshot:      off

Suggested .cqrs-lint.json features section:
{
  "features": {
    "store": "postgres",
    ...
  }
}
(exit 0)
```

`doctor` confidently reports a profile — including a real `postgres` store
detection — as if the project loaded cleanly. It did not.

### Symptom 3 — The two commands disagree

On the **same project, same invocation directory**:

- `cqrs-lint` says: "no CQRS usage here at all"
- `cqrs-lint doctor` says: "I see postgres + read-only command flow"

Both exit 0. A user running both has no way to know the project is broken. The
disagreement is not even internally consistent.

### Symptom 4 — No flag surfaces the truth

`--verbose`, `--quiet=false`, `doctor` — none of them mention that package
loading failed. There is no `--debug-loader` or equivalent. The only way to
discover the real error is to run `go build ./...` independently, which the
linter is supposed to make unnecessary.

---

## Part 2: Root Cause Analysis

All file references are against `go-cqrs-lite` at HEAD (commit `00f678a0`,
which is `cqrs-lint 0.2.1` in source; the installed binary is `0.2.0` but the
loader logic is identical in both).

### Bug 1: `BuildContext` silently swallows per-module load errors

**File:** `cmd/cqrs-lint/pkg/analyzer/loader.go:86-90`

```go
for _, dir := range modDirs {
    pkgs, err := loadFromDir(dir, fset)
    if err != nil {
        continue   // <-- silent skip on hard load failure
    }
    ...
}
```

When `loadFromDir` returns an error (e.g. `packages.Load` itself fails), the
entire module is dropped without logging. The error never reaches `main.go`,
never reaches the user. `BuildContext` returns a partial context with
`err == nil`.

**The deeper problem:** even when `loadFromDir` returns `err == nil`, individual
packages can still carry errors. `go/packages` attaches module-resolution
errors (like `invalid version: unknown revision`) to `pkg.Errors`, not to the
top-level error. Those are filtered here:

**File:** `cmd/cqrs-lint/pkg/analyzer/loader.go:94-101`

```go
for _, pkg := range pkgs {
    if len(pkg.Errors) > 0 {
        continue   // <-- silent skip on per-package errors
    }
    if !IsCQRSImport(pkg) {
        continue
    }
    ...
}
```

So a package that imports go-cqrs-lite but failed to resolve its dependencies is
silently dropped from `ctx.GoFiles`. The user is never told "package X failed to
load because of error Y." Combined with the `len(actx.GoFiles) == 0` check in
`main.go:171-177`, this produces the misleading "No Go files importing
go-cqrs-lite found" message.

### Bug 2: `doctor` reads a different field than `lint`, so they disagree

**File:** `cmd/cqrs-lint/pkg/analyzer/feature_detect.go:28-64` (Pass 1)

`DetectFeatures` iterates `ctx.Packages` — **all** packages returned by
`go/packages`, including ones with errors. A package can have `len(pkg.Errors)

> 0`and still have a partially-populated`pkg.Imports`. So `DetectFeatures`finds the`go-cqrs-lite/stack/postgres/v4`import (real — it's in`usermgmt/postgres_setup.go:10`) and sets `Store = StorePostgres`.

Meanwhile, `main.go` and the rule detectors iterate `ctx.GoFiles`, which is
populated only from error-free CQRS-importing packages (loader.go:107-124). On
this project that set is empty, so lint reports "nothing found."

**Result:** `doctor` reports a profile built from the broken packages' import
metadata, while `lint` reports nothing because the same broken packages were
filtered out of `GoFiles`. The two commands read different subsets of the same
load result and neither mentions the load failure.

### Bug 3: The "nothing to lint" exit path returns nil

**File:** `cmd/cqrs-lint/main.go:171-177`

```go
if len(actx.GoFiles) == 0 {
    if !cfg.Quiet {
        fmt.Fprintln(os.Stderr, "No Go files importing go-cqrs-lite found. Nothing to lint.")
    }
    return nil   // <-- success exit code
}
```

Returning `nil` means cmdguard exits 0. CI wiring that runs `cqrs-lint` as a
gate will treat a silently-empty analysis as **passing**. A broken build that
happens to zero out `GoFiles` is indistinguishable from a clean, pattern-free
project. This is the worst-case failure mode for a lint gate: green CI on a red
build.

### Bug 4: `doctor` has the same blind spot

**File:** `cmd/cqrs-lint/doctor.go:18-22`

```go
actx, err := analyzer.BuildContext(cfg.Path)
if err != nil {
    return fmt.Errorf("load packages: %w", err)
}
```

Because `BuildContext` never returns an error for per-package load failures
(see Bug 1), `doctor`'s error check here is dead code for the common case. It
only fires if `findGoModDirs` itself fails. `doctor` then proceeds to print a
profile from whatever partial data made it into `ctx.Packages`, exits 0, and
the user is told everything is fine.

---

## Part 3: Why This Matters

1. **Trust failure.** A linter that reports "all clear" on a broken build is
   worse than one that crashes. The user assumes the project is fine and looks
   for the bug everywhere except the build.

2. **The disagreement compounds the confusion.** A user who runs both `lint`
   and `doctor` gets two contradictory confident answers. There is no signal
   that either is unreliable. This is the kind of thing that costs hours of
   debugging.

3. **CI gates become green-on-broken.** Any project using `cqrs-lint` as a
   required check will pass CI whenever package loading fails, regardless of
   the actual code quality. The gate is silently disabled by any transient
   module-resolution issue.

4. **The recovery path is undocumented.** The only way to discover the real
   error is to run `go build ./...` by hand — exactly the manual step a linter
   is supposed to eliminate. `cqrs-lint doctor` is advertised as the
   "something's wrong, diagnose it" command, and it is the one most actively
   misleading here.

5. **Multi-module workspaces are the common case for this ecosystem.** The
   loader already supports monorepos (`findGoModDirs`), but the
   partial-success semantics mean the failure mode scales with module count:
   more modules = more chances for one to fail silently = higher probability
   of a misleading "nothing to lint" on a real project.

---

## Part 4: Fix Recommendations

All recommendations are concrete and ordered by impact. Each is independently
shippable.

### Fix 1 (critical): Surface load errors instead of swallowing them

**File:** `cmd/cqrs-lint/pkg/analyzer/loader.go`

Collect per-package errors during loading and attach them to the
`AnalysisContext` so the caller can report them. Do not `continue` silently.

```go
// Proposed: add to AnalysisContext (types.go)
type AnalysisContext struct {
    ...
    // LoadErrors holds per-package errors encountered during BuildContext.
    // Non-empty means the analysis is partial; callers should warn the user.
    LoadErrors []PackageLoadError
}

type PackageLoadError struct {
    Module  string   // go.mod directory
    PkgPath string   // offending package, if known
    Errors  []string // from pkg.Errors, or the top-level load error
}
```

Then in `BuildContext`:

```go
for _, dir := range modDirs {
    pkgs, err := loadFromDir(dir, fset)
    if err != nil {
        ctx.LoadErrors = append(ctx.LoadErrors, PackageLoadError{
            Module: dir,
            Errors: []string{err.Error()},
        })
        continue
    }
    for _, pkg := range pkgs {
        if len(pkg.Errors) > 0 {
            msgs := make([]string, 0, len(pkg.Errors))
            for _, e := range pkg.Errors {
                msgs = append(msgs, e.Error())
            }
            ctx.LoadErrors = append(ctx.LoadErrors, PackageLoadError{
                Module:  dir,
                PkgPath: pkg.PkgPath,
                Errors:  msgs,
            })
            continue
        }
        ...
    }
}
```

This preserves the "best effort, keep going" behavior for partial loads while
making the failure visible.

### Fix 2 (critical): Warn loudly when `GoFiles` is empty but load errors exist

**File:** `cmd/cqrs-lint/main.go:171-177`

The current message is only correct when the project genuinely has no CQRS
imports. When `LoadErrors` is non-empty, the message is actively wrong. Replace
the silent-success path:

```go
if len(actx.GoFiles) == 0 {
    if len(actx.LoadErrors) > 0 {
        fmt.Fprintln(os.Stderr, "No analyzable Go files found, but package loading reported errors.")
        fmt.Fprintln(os.Stderr, "The project may not compile. Fix these and re-run:")
        fmt.Fprintln(os.Stderr)
        for _, le := range actx.LoadErrors {
            fmt.Fprintf(os.Stderr, "  module %s (%s):\n", le.Module, le.PkgPath)
            for _, m := range le.Errors {
                fmt.Fprintf(os.Stderr, "    %s\n", m)
            }
        }
        return errFindingsWithErrors // non-zero exit
    }
    if !cfg.Quiet {
        fmt.Fprintln(os.Stderr, "No Go files importing go-cqrs-lite found. Nothing to lint.")
    }
    return nil
}
```

Reusing the existing `errFindingsWithErrors` sentinel keeps the exit-code
contract consistent with the "errors found" path.

### Fix 3 (critical): Make `doctor` refuse to report on a broken load

**File:** `cmd/cqrs-lint/doctor.go:18-22`

`doctor` should not print a confident profile when the load was partial. After
`BuildContext`, check `LoadErrors`:

```go
actx, err := analyzer.BuildContext(cfg.Path)
if err != nil {
    return fmt.Errorf("load packages: %w", err)
}
if len(actx.LoadErrors) > 0 {
    fmt.Fprintln(os.Stderr, "WARNING: package loading was partial; the profile below may be incomplete or misleading.")
    for _, le := range actx.LoadErrors {
        fmt.Fprintf(os.Stderr, "  %s: %v\n", le.Module, le.Errors)
    }
    fmt.Fprintln(os.Stderr)
}
```

At minimum, warn. Consider returning a non-zero exit when **all** modules
failed (the profile is then fiction).

### Fix 4 (important): Make `doctor` and `lint` agree

The deeper fix for Bug 2: `DetectFeatures` should not trust import data from
packages that failed to load. Either:

- Filter `ctx.Packages` to error-free packages before Pass 1 (consistent with
  how `GoFiles` is built), **or**
- Track which packages contributed to each feature flag so `doctor` can annotate
  "this detection came from a package with load errors."

The first option is simpler and makes the two commands read the same data. The
profile may become sparser on broken builds, but sparser-and-honest beats
confident-and-wrong.

### Fix 5 (nice to have): Add a loader diagnostic flag

A `--debug-loader` (or reuse `--verbose`) that prints, per module: packages
loaded, packages skipped, skip reason. This is invaluable for monorepo
debugging and would have made this entire class of issue self-diagnosing.

---

## Part 5: Suggested Message Redesign

The current "No Go files importing go-cqrs-lite found. Nothing to lint." is a
single message covering two very different situations:

1. **Project genuinely has no CQRS imports** → message is correct.
2. **Project has CQRS imports but failed to load** → message is a lie.

These need to be distinguished. Proposed user-facing copy for case 2:

```
cqrs-lint: could not analyze any packages.

This usually means the project does not compile. Package loading reported
errors in 1 module(s):

  /path/to/cqrs-htmx (github.com/larsartmann/cqrs-htmx/v4):
    command/v4@v4.0.0 requires
      dispatcher/v4@v4.0.0-00010101000000-000000000000: invalid version: unknown revision 000000000000

Fix the build errors above (try `go build ./...`), then re-run cqrs-lint.
Nothing was analyzed; this is not a clean bill of health.
```

Key properties of this message:

- Names the failure mode ("does not compile") up front.
- Shows the real error, not a paraphrase.
- Points at the recovery command (`go build ./...`).
- Explicitly states the analysis did not happen — closing the loophole where a
  user might interpret exit 0 as "lint passed."

---

## Part 6: Secondary Observations

These are not the focus of this feedback but surfaced during reproduction.

### The installed binary lags source

`cmd/cqrs-lint/main.go:27` declares `version = "0.2.1"`, but the installed
binary at `/home/lars/go/bin/cqrs-lint` reports `0.2.0` (built 2026-07-17
00:33, before the version bump commit `0ff1d621`). None of the loader bugs
described here are fixed in 0.2.1 — the version drift just made reproduction
slightly harder to pin down. Consider a release-and-install step in the flake
so the installed binary always matches the tagged source.

### The monorepo loader is the right design

`findGoModDirs` + per-module `loadFromDir` is a good approach for
workspace-based projects. The bug is purely in error reporting, not in the
loading strategy. The fix is additive (collect + surface errors), not
architectural.

---

## Summary Table

| #   | Bug                                                      | Severity | Fix   | Effort |
| --- | -------------------------------------------------------- | -------- | ----- | ------ |
| 1   | `BuildContext` swallows per-module load errors           | HIGH     | Fix 1 | S      |
| 2   | "No Go files found" returned as success on broken builds | HIGH     | Fix 2 | S      |
| 3   | `doctor` reports a profile from partial/broken data      | HIGH     | Fix 3 | S      |
| 4   | `lint` and `doctor` read different package subsets       | HIGH     | Fix 4 | M      |
| 5   | No way to inspect loader decisions                       | MED      | Fix 5 | S      |

All five are in the same file cluster (`loader.go`, `main.go`, `doctor.go`,
`feature_detect.go`) and can be addressed in one PR. Fixes 1-3 are the minimum
to stop the tool from actively misleading users on broken builds.
