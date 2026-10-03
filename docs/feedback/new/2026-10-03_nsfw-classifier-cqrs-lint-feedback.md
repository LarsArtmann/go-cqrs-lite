# Consumer Feedback: cqrs-lint (from nsfw-classifier adoption)

**From:** nsfw-classifier remediation sessions (2026-10-03)
**Perspective:** AI agent running cqrs-lint (dev, commit 3756eb4) against a
production Go app event-sourced on `system/v4` + `metaengine/v4` (three
domains: feedback, rooms, verdicts; SQLite engine; go-sse transport)
**Tone:** Honest, direct, grateful but critical where warranted

**Headline:** started at 1 ERROR + ~20 WARNING + ~12 INFO (exit 1); now
0 findings with 11 inline suppressions + 7 config disables. Roughly a third
of the original findings became real improvements we adopted; a third were
verified false positives; a third were deliberate-decline coaching. The
tool made the codebase measurably better — and almost every false positive
shares one root cause, which is fixable.

---

## What Works Superbly

### 1. Suppressions with mandatory reasons + stale detection

`//cqrs-lint:ignore(CODE) reason` plus `--fail-on-stale-suppressions` is the
best suppression design we have used in any linter. Reasons are forced into
the diff, and the stale check turned suppression drift from silent rot into
a gate failure. During our sessions it caught two misplaced suppressions the
moment their anchor moved.

### 2. Coaching we adopted (the wins)

- `decider.StrictApply` for known-type enforcement (C003/B005 family) —
  unknown event types now fail closed in all three domains.
- `event.WithSchemaVersion(1)` stamps (D013) — cheap now, priceless at the
  first schema migration.
- `metaengine.Volume(n)` hints on all three queries (F019).
- Value-embedded `command.BasicCommand` (A013) — removed a whole class of
  nil-deref risk at dispatch sites.
- The `must*` panic-helper naming convention (C009) with its documented
  exemption list — we renamed helpers and moved guards into `New*`
  constructors instead of suppressing.

### 3. Severity and exit-code discipline

Errors exit non-zero; WARNING/INFO coaching does not. `--adoption` keeping
F-series out of the health score is the right call for adoption coaching.
`--min-severity`/`--min-confidence`/`--fp-suspects`/`--show-suppressed` are
all things we used or wished for and found already present.

### 4. Config ergonomics

JSONC with comments survived by the parser (our entire disable rationale
lives inline in `.cqrs-lint.json`), plus `explain` and `doctor`
subcommands that made the config self-documenting.

---

## False Positives Found (all verified against rule source and our code)

Every item below was reproduced and root-caused before suppressing.

### 1. Detection is blind to helper indirection (two separate rules)

`detectSoftDeleteRegistry` reads `EventTypesEmitted`, which never sees event
types routed through a constructor helper:

```go
func newRoomEvent(t event.Type, ...) (event.Event, error) { // single emit path
    return event.New(t, ...)
}
```

We emit `room.item_deleted` (tombstone-like); raw detection said
`soft-delete: false`. A012's tombstone case-matching then also failed to
recognize `case evtRoomItemDeleted:` (constant identifier, not a string
literal like the suggested `"user.deleted"`), so the rule fired on a fold
that demonstrably handles deletion. **Fix sketch:** resolve constant
identifiers to their `event.Type` string values in the registry pass, and
match A012 case clauses on the resolved type name.

### 2. Detection only sees go-cqrs-lite importers (scope artifacts)

The analyzer reads the 17 files that import the library. Our HTTP server,
`cli.Bootstrap` (which calls `slog.SetDefault`), and our `/metrics`
registry live in non-importer packages. Consequences we hit:

- `server: false` detected — we run a three-probe `/healthz`//`readyz`//
  `/startupz` server with graceful shutdown and optional bearer auth.
- F028 fired ("never calls slog.SetDefault") — false; the call exists
  outside the scanned scope.
- F004 fired ("runtime observability is unavailable") — false; `/metrics`
  exists with its own registry (JSON shape by contract, not Prometheus).

Worse, `doctor`'s suggested config pins `Server` and `SoftDelete`
**unconditionally** (`ToConfigFeatures` always emits both), so we copy-pasted
scope-artifact values as project truth. That single design choice caused our
only real misconfiguration. **Fix sketch:** only suggest pins for
confidence-checked detections, or annotate scope-limited keys ("server not
found in analyzed packages — confirm") in the suggestion.

### 3. Single-word JSON keys counted as camelCase (A011)

`id` vs `source_dir` in the same struct = "mixed JSON key casing". A
single-word key has no case convention; counting it as camel manufactures
the mixture. We suppressed three of these. **Fix sketch:** skip keys with no
underscore AND no interior capitals when the majority style is snake.

### 4. Heuristic-scoped FPs we suppressed individually

- **F031**: flagged `bufio.Scanner.Scan` loops — we have zero metaengine
  `reader.Scan` calls (grep-verified); the reader does point `Get` only.
- **F026**: fires at `metaengine.NewReader` construction and wants
  `WithPrefetch` — pointless for a reader that never Scans (prefetch caches
  cursor pages). Consider checking whether Scan is actually called on the
  reader before coaching.
- **F010**: linear-scan coaching on a helper that linearly scans a small
  in-memory slice by design.
- **A009**: coaches `stack/` presets — deprecated pre-v5 per ADR-0123; the
  current direction is `system.New` + `metaengine`, which is what we use.
  The rule should recognize the composition root as the adopted path.

---

## Design Suggestions

### 1. Feature pins should act as explicit declines

Pinning `"tracing": "off"` did not stop F003 from coaching OTel adoption —
the rule gates only on HasServer/ServerLocal. `Monetary` already works the
decline way (its doc says "off" downgrades C008). Feature pins that state a
deliberate "off" should suppress the corresponding adoption coaching
(F003→tracing, arguably F004→a future metrics pin).

### 2. Project-wide rules need stable anchors

F005 anchors at the alphabetically-first package that calls
`WithSchemaVersion` — ours landed on `package feedback` end-of-line. If that
package ever drops the call, the anchor (and our suppression) silently moves
to another package. `--fail-on-stale-suppressions` catches the fallout, but
a stable project-level anchor (or per-package reporting) would remove the
trap.

### 3. Stale-suppression warnings could name the anchor

Our two stale suppressions were misplaced doc-comment placements. The
warning says "rule F021 does not fire here" — adding "fires at
domain_projections.go:92" would have made the fix zero-think.

### 4. No `--config` flag (verified against `--help`)

Probing config variations requires mutating the tracked `.cqrs-lint.json`
(we backed it up and restored by hand mid-session). A `--config <path>`
flag would make A/B probing and CI overrides non-destructive.

### 5. DomainKind: two of four values are undiscoverable

`internal` and `security` exist as DomainKind values, but
`detectDomainRegistry` only ever returns `financial` or `unknown`, and
doctor never mentions pinnable kinds — we learned they exist by reading the
analyzer source. A doctor hint ("domain: unknown (pinnable: internal,
security, financial)") would surface them.

### 6. Minor: same-named folds across packages

B005's `StrictApplyFolds` registry is name-matched; same-named folds in
different packages could confuse it. Not hit here — noted for the test suite.

---

## Summary Scorecard

| Area                     | Verdict                                                                                  |
| ------------------------ | ---------------------------------------------------------------------------------------- |
| Coaching value (adopted) | 5 adoptions, all genuinely good (StrictApply, schema stamps, Volume, value embed, must*) |
| False-positive rate      | ~10 of ~33 original findings, mostly one root cause (indirection + scope blindness)      |
| Suppression system       | Best-in-class; stale detection is a gate we now run always                               |
| Config ergonomics        | JSONC + doctor + explain: excellent; doctor suggestions need confidence gating           |
| Verdict                  | Keep in the dev loop; every FP above is fixable without weakening the rules              |

— Filed from the nsfw-classifier sessions; full evidence trail:
`nsfw-classifier/docs/status/2026-10-03_00-56_cqrs-lint-remediation.md`,
`..._01-47_cqrs-lint-closeout-self-review.md`,
`..._02-05_feature-profile-audit-self-review.md`.
