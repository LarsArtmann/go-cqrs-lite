# cqrs-lint — Consumer Feedback Round 2 (cqrs-htmx)

**Consumer:** [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) — Go library (SDK) that makes go-cqrs-lite usable with HTMX, templ, and Casbin authorization. Multi-module Go workspace (18 modules under one `go.work`).
**Module path:** `github.com/larsartmann/cqrs-htmx/v4`
**Version used:** go-cqrs-lite v4.2.0 (command/event/id/query/stack/decider/projectionhost modules)
**Installed cqrs-lint:** `v0.2.2` (Nix binary, hash `d6be91cad926f919f34d48f63ddf2592cad91590`)
**Source HEAD at time of writing:** `38e7d11dcc27d55f45da8ce0861838d38a3e671e` (2026-08-02 22:27)
**Date:** 2026-08-02

---

## Executive Summary

Two full audit sessions were spent fighting two issues that are **already fixed in source** but **not in the installed Nix binary**. The installed `cqrs-lint v0.2.2` (Nix hash `d6be91ca`) was built from a commit just before the round-2 fixes landed. Specifically:

1. **gofmt/space conflict** — Source commit `b4554cdc` (2026-08-02 17:41) added `normalizeCommentPrefix()` so both `//cqrs-lint:ignore` and `// cqrs-lint:ignore` (Go-idiomatic, what gofmt produces) are accepted. The installed binary does NOT have this fix.

2. **One-suppression-per-line limitation** — Source commit `589b07d6` added comma-separated rule support (`//cqrs-lint:ignore(E004,E006)`). The installed binary does NOT have this fix, despite AGENTS.md noting the Nix binary "does not support it despite the source implementing it."

**16 non-suppressed findings remain** that would all be fixable if the Nix binary matched source HEAD. The gofmt-dirty files in the codebase are an unnecessary workaround that exists solely because the installed binary is stale.

**Severity: HIGH.** The stale binary caused hours of wasted work, produced incorrect documentation (AGENTS.md now documents a "gofmt conflict" that doesn't exist in current source), and forced gofmt-dirty code into the repository as a workaround.

---

## Part 1: The Stale Binary Problem

### What happened

cqrs-lint is distributed via Nix. The installed binary was built from commit `d6be91ca` ("sync go.mod and update vendorHash after lockfile refresh"). This is the commit immediately BEFORE `b4554cdc` ("correct false positives and accept Go-idiomatic comment style"), which landed 19 cqrs-lint commits ago.

| Feature | Source (HEAD) | Installed binary (v0.2.2) |
|---------|---------------|---------------------------|
| `// cqrs-lint:ignore` (space after `//`) | ✅ Accepted via `normalizeCommentPrefix()` | ❌ Silently ignored |
| `//cqrs-lint:ignore(E004,E006)` (comma-separated) | ✅ Parsed via `strings.SplitSeq(rawIDs, ",")` | ❌ Silently ignored |
| `--exclude-rules` CLI flag | ✅ Available | ❌ Not available |
| Config `rules.disable` | ✅ Available | ❌ Not available |
| `cqrs-lint init --preset` | ✅ Available | ❌ Not available |
| Unknown-rule-ID stale detection | ✅ Available | ❌ Not available |

### Impact

1. **Hours wasted on a non-existent problem.** The gofmt vs cqrs-lint conflict (documented extensively in cqrs-htmx's AGENTS.md) does not exist in source. `normalizeCommentPrefix()` was added specifically to handle this. But because the installed binary predates the fix, the conflict is real for consumers.

2. **Gofmt-dirty code committed as a workaround.** 7 files in cqrs-htmx have `//cqrs-lint:ignore` comments (no space after `//`) that gofmt wants to reformat. These files are intentionally gofmt-dirty. This workaround would be unnecessary with the source fix.

3. **16 non-suppressable findings that ARE suppressable in source.** Every remaining finding is caused by multiple rules firing on one line (e.g., E004 + E006 on an `event.New()` call). Comma-separated suppressions fix this. The installed binary doesn't support them.

4. **Incorrect documentation propagated.** cqrs-htmx's AGENTS.md now contains a detailed entry about the "gofmt conflict" including workarounds, root cause analysis, and instructions to never run `gofmt -w` on certain files. All of this is obsolete with the source fix but remains necessary documentation for the stale binary.

### Reproduction

```bash
# Installed binary — space after // is ignored:
$ echo '// cqrs-lint:ignore(C035) test' | cqrs-lint  # (runs against a real file)
# C035 finding is NOT suppressed

# Source HEAD — space after // works:
$ go run ./cmd/cqrs-lint --strict ./...
# C035 finding IS suppressed

# Installed binary — comma-separated is ignored:
$ echo '//cqrs-lint:ignore(E004,E006) demo'  # on an event.New line
# Both E004 and E006 findings fire

# Source HEAD — comma-separated works:
$ go run ./cmd/cqrs-lint --strict ./...
# Both findings suppressed
```

### Why this is hard to discover

cqrs-lint reports version `0.2.2` for both the stale binary and the source. The `const version = "0.2.2"` in `main.go:18` was not bumped when the round-2 fixes landed. A consumer checking `cqrs-lint --version` has no signal that their binary is behind source.

---

## Part 2: Remaining Findings (All Caused by the Stale Binary)

### 2a. gofmt Conflict (would not exist with source fix)

**Affected files (7):** `usermgmt/es_readmodel.go`, `usermgmt/es_bot_readmodel.go`, `usermgmt/es_membership_readmodel.go`, `usermgmt/es_tenant_readmodel.go`, `dashboardui/config.go`, `examples/catalog-demo/main.go`, `identity-model/events.go`

Each of these files has `//cqrs-lint:ignore(RULE)` at column 1, immediately before a Go declaration. gofmt (Go 1.19+) normalizes this to `// cqrs-lint:ignore(RULE)` (adding a space), which the installed binary silently ignores.

**With source fix:** gofmt would produce `// cqrs-lint:ignore(...)` and cqrs-lint would accept it. No conflict. No gofmt-dirty files. No AGENTS.md documentation needed.

### 2b. One-Suppression-Per-Line (would not exist with source fix)

**16 remaining non-suppressed findings**, all caused by multiple rules firing on the same code line:

| Location | Rules firing | Already suppressed | Can't suppress |
|----------|-------------|-------------------|----------------|
| `dashboardui/config.go:2` | E009, F002, F011, F015, E014 | E014 | E009, F002, F011, F015 |
| `examples/dashboard-demo/main.go` (4×) | E004, E006, B027, D013 | E006 (or B027 via constants) | E004, D013 |
| `usermgmt/stack_repositories.go` (4×) | B025, A017, E008 | A017 | B025, E008 |
| `examples/dashboard-demo/main.go:237` | S003, C028 | C028 | S003 |

**With source fix:** Each line would use `//cqrs-lint:ignore(E004,E006,D013)` or `//cqrs-lint:ignore(B025,A017,E008)`. All 16 findings would be suppressed. Zero remaining.

---

## Part 3: Genuinely Useful Findings (Not Stale-Binary Related)

These findings from the audit are independent of the stale binary — they represent real code quality improvements that were made:

### 3a. closeBus error handling (real fix, not suppression)

**Before:** `_ = c.Close()` — silently discarded Close() errors, triggering C023 and C015.

**After:** `if err := c.Close(); err != nil { slog.Debug("closeBus: best-effort close failed", "error", err) }` — properly handles the error by logging it.

This is a genuine improvement: the error is now visible at debug level instead of silently swallowed.

### 3b. Stream-type constants in dashboard-demo (real fix, not suppression)

**Before:** Hardcoded `"User"` and `"Order"` string literals in `event.New()` and `id.NewStreamRef()` calls.

**After:** Extracted `streamTypeUser` and `streamTypeOrder` as `id.StreamType` constants, used consistently across all calls.

This matches Go's typed-ID philosophy and makes the demo code self-documenting.

### 3c. Stale suppressions from session 1 (18 → 0)

Session 1 left 18 stale-suppression warnings. All were caused by incorrect placement (tab-indented comments that cqrs-lint ignores, comments on wrong lines, swapped rule IDs). This session systematically fixed every one:

- C035/P011 on read models: moved from tab-indented lines to correct positions (above struct for C035, above each map field for P011)
- D009/C023 in closeBus: eliminated C023/C015 entirely by fixing the code
- F009/P008 in projection setup: swapped — each was on the other's line
- A032 in SQL view DTOs: moved from struct-level to field-level
- S007 in store.go: moved from struct declaration to constructor return
- S006 in events.go: replaced with correct F006 rule
- E010 in catalog-demo: removed (rule no longer fires)

---

## Part 4: Suppression Placement Semantics

### The line-above-only rule

cqrs-lint v0.2.2 checks exactly two lines for suppressions: the finding's own line and the line immediately above. This is documented but has subtle implications that caused significant confusion:

**Tab-indented comments inside struct bodies do work** for field-level findings:
```go
type UserReadModel struct {
    readModelCore[*UserReadModel]
    //cqrs-lint:ignore(P011) ← this works (line above the field)
    users map[id.StreamID]*User
}
```

**Tab-indented comments before the struct declaration do NOT work:**
```go
    //cqrs-lint:ignore(P011) ← this does NOT work (indented, before a different line)
type UserReadModel struct {
```

The difference is that the first case has the comment on the line immediately above the finding's line. The second case has it on a line that cqrs-lint doesn't check (it's above the struct declaration, but the P011 finding is on the map field line inside the struct).

**Recommendation:** Consider documenting this pattern explicitly in cqrs-lint's help output or suppression docs. The "line + line-above" rule is correct but non-obvious when findings are on struct fields rather than the struct declaration.

### The blank-line gap

Adding a blank line between the suppression and the finding breaks suppression:
```go
//cqrs-lint:ignore(C035) reason

type UserReadModel struct {   ← C035 fires here, NOT suppressed
```

cqrs-lint v0.2.2 does not skip blank lines when looking for the "line above." This was discovered empirically. **Source HEAD behavior is the same** — `checkSuppressionInFile` checks `line` and `line-1` only.

**Recommendation:** Consider skipping blank lines when scanning upward. A blank line is never meaningful content, and the suppression intent is clearly directed at the next declaration. Alternatively, document this explicitly.

---

## Part 5: Fix Recommendations

### Fix 1 (critical): Publish the round-2 fixes to Nix

The fixes for `normalizeCommentPrefix()` and comma-separated suppressions exist in source since commit `b4554cdc` (2026-08-02 17:41). They need to be published as a Nix binary so consumers get them. This single action would:
- Eliminate the gofmt conflict for all consumers
- Eliminate the one-suppression-per-line limitation
- Make the 16 remaining cqrs-htmx findings suppressable
- Make the AGENTS.md gofmt-conflict documentation obsolete

### Fix 2 (important): Bump the version constant

`const version = "0.2.2"` in `main.go:18` was not bumped when the round-2 fixes landed. Consumers have no way to detect whether their binary includes the fixes. Bump to `0.3.0` (comma-separated support and gofmt compatibility are user-facing features) or at minimum `0.2.3`.

### Fix 3 (nice to have): Skip blank lines in suppression scanning

In `checkSuppressionInFile` (`parser.go:129-154`), when checking `line-1`, skip blank lines:

```go
// Proposed: scan upward past blank lines
checkLine := line - 1
for checkLine >= 1 && checkLine <= len(lines) {
    if strings.TrimSpace(lines[checkLine-1]) == "" {
        checkLine--
        continue
    }
    suppressedRules := ParseSuppressions(lines[checkLine-1])
    if _, ok := suppressedRules[ruleID]; ok {
        return true
    }
    break
}
```

This makes the suppression placement more forgiving without changing the fundamental line-above semantics.

### Fix 4 (nice to have): Document field-level suppression pattern

Add to `--help` suppression section or docs:

```
Field-level suppression:
  Suppressions on the line immediately above a struct field suppress
  findings that fire on that field. This is useful for map fields in
  read models (P011) or string-typed ID fields (A032).

  type UserReadModel struct {
      //cqrs-lint:ignore(P011) bounded by finite user count
      users map[id.StreamID]*User
  }
```

### Fix 5 (nice to have): Version stamp the Nix binary with the source commit

Instead of relying on a manually-bumped `const version`, embed the git commit hash at build time. This makes stale binaries self-identifying:

```bash
$ cqrs-lint --version
cqrs-lint 0.2.2 (commit: b4554cdc, built: 2026-08-02T17:41:00Z)
```

Consumers can then compare against `git log` to determine if they have the latest fixes.

---

## Part 6: Impact Metrics

### Time spent on stale-binary-caused issues

| Activity | Time wasted | Cause |
|----------|------------|-------|
| Discovering gofmt adds spaces to suppression comments | ~15 min | Stale binary (source already fixes this) |
| Attempting blank-line workaround | ~10 min | Stale binary (blank lines don't help in either version) |
| Reverting blank-line workaround | ~5 min | Same |
| Documenting the "gofmt conflict" in AGENTS.md | ~10 min | Stale binary (conflict doesn't exist in source) |
| Trying to suppress E004 alongside E006 | ~10 min | Stale binary (source supports comma-separated) |
| Writing this feedback | ~10 min | Stale binary |
| **Total** | **~60 min** | |

### Findings that would be eliminated by updating the binary

| Category | Count | Eliminated by |
|----------|-------|---------------|
| gofmt-dirty files (gofmt conflict) | 7 files | `normalizeCommentPrefix()` |
| E004 + D013 in dashboard-demo | 5 | Comma-separated `//cqrs-lint:ignore(E004,E006,D013)` |
| B025 + E008 in stack_repositories | 5 | Comma-separated `//cqrs-lint:ignore(B025,A017,E008)` |
| S003 in dashboard-demo | 1 | Comma-separated `//cqrs-lint:ignore(S003,C028)` |
| E009/F002/F011/F015 in dashboardui | 4 | Comma-separated at `package` line |
| **Total** | **16 findings + 7 gofmt-dirty files** | |

---

## Part 7: Positive Observations

### The fixes in source are excellent

The round-2 fixes in source are exactly right:
- `normalizeCommentPrefix()` is simple, targeted, and solves the real problem (gofmt compatibility)
- Comma-separated rule support is the correct UX for multi-rule lines
- `--exclude-rules` and config `rules.disable` give consumers proper control
- `cqrs-lint init --preset` is a thoughtful UX addition for different project types
- Unknown-rule-ID stale detection catches typos

The implementation quality is high. The only problem is distribution.

### The stale detector is valuable

Even with the stale binary, the stale-suppression detector caught 18 misplaced suppressions from session 1. This feature alone improved the codebase quality significantly. The warnings are clear and actionable ("rule X does not fire here; safe to remove").

### The detector coverage is impressive

179 detectors across 10 modules found genuine issues: C021 (mutex-held-during-decode), C035 (missing mutex on map fields), B027 (hardcoded stream types), D014 (missing JSON tags). These are real code quality findings, not noise.

---

## Summary Table

| # | Issue | Severity | Status in source | Status in installed binary | Fix |
|---|-------|----------|-----------------|---------------------------|-----|
| 1 | Stale Nix binary missing round-2 fixes | HIGH | ✅ Fixed (b4554cdc) | ❌ Not fixed | Publish new Nix binary |
| 2 | Version constant not bumped | MED | ❌ Still 0.2.2 | ❌ 0.2.2 | Bump to 0.3.0 |
| 3 | gofmt/space conflict | HIGH | ✅ Fixed (normalizeCommentPrefix) | ❌ Active problem | Resolved by Fix 1 |
| 4 | One-suppression-per-line | HIGH | ✅ Fixed (comma-separated) | ❌ Active problem | Resolved by Fix 1 |
| 5 | Blank line breaks suppression | LOW | ❌ Same behavior | ❌ Same behavior | Skip blanks in upward scan |
| 6 | Field-level suppression undocumented | LOW | ❌ Undocumented | ❌ Undocumented | Add to --help / docs |

Fixes 1 and 2 are the priority. They would eliminate all 16 remaining findings and make the gofmt-dirty workaround unnecessary.
