# Status: go-codec v0.2.0 follow-ups, upstream helper dedupe, and the verify-fast gate

> Session: 2026-08-16 ~20:45–21:59. Continuation of
> `2026-08-16_20-40_correctness-sweep-leftovers-shipped.md` — executing the
> user-approved answers to its three questions (Q1 go-codec release, Q2
> kv.Cache rework, Q3 ErrBinaryNotFound semver) plus the self-identified docs
> truth-restoration backlog.

## a) FULLY DONE ✅

### Docs truth-restoration (the prior session's owned miss)

- `docs/migration/MIGRATION-GUIDE.md`, `docs/adr/0051-…md`, `docs/adr/0053-…md`
  — all three "fallback uses JSONCodec" claims rewritten to the shipped
  configured-codec + JSON↔CBOR cross-retry behavior (ADR-0050 addendum).
  Repo-wide grep confirms zero stale claims remain in living docs (archives
  and ADR-0050's intentionally-superseded bullet excepted).
- `.agents/skills/go-cqrs-lite/references/recipes.md` — added the kv.Cache
  copy-isolation contract note (core.md needed nothing: lookup row only).
- `cmd/doc-check`: **910 refs green** after all edits.
- `docs/reviews/2026-08-14_14-25_brutal-self-review.md` lines 133–136 — all
  three sweep defects annotated FIXED; Q3 decision (v4.x patch) recorded on the
  ErrBinaryNotFound line.
- `docs/planning/…SUPERB-PARETO-EXECUTION-PLAN.md` — **T10 → 100%** (verified
  all three security-hygiene sub-items exist: SECURITY.md, release.yml
  govulncheck step, iroh exact pseudo-version pin in
  `metaengine/irohengine/quic/go.mod`).
- `FEATURES.md` — "Legacy row rescue" row added next to Envelope wrapping.
- `CHANGELOG.md` — follow-up hardening entry + Q3 decision + upstream-helper
  consolidation entry.
- Prior status report annotated (docs sweep + c.4 resolved).

### Test completeness

- Garbage-still-errors tests mirrored into command
  (`TestTypedCommandStore_GarbagePayloadStillErrors`) and snapshot
  (`TestTypedStore_GarbageStateStillErrors`) — all four blind stores now
  symmetric. Verified PASS with `-v`, full suites green standalone **and**
  with `-race`, lint 0 issues.

### art-dupl hygiene

- All four `//art-dupl:accept` comments shortened to 103 chars (< golines 120
  budget). `nix run .#check-duplication`: **zero clone groups in my modules**;
  the 11 flagged groups are the documented pre-existing metaengine ones —
  unchanged by this session.

### Q1: go-codec release (approved by user, incl. push)

- Reviewed the uncommitted-tree premise: **the perf work was already committed**
  (f90c52b et al.), merely never tagged. Only v0.1.0 existed.
- Added upstream `DecodeEnvelopeOrLegacy[T]` + `otherStandardCodec` to
  `go-codec/envelope.go` (exported one-call envelope/raw decode; envelope via
  stamped codec, raw via configured codec, exactly one JSON↔CBOR cross-retry,
  custom codecs get configured attempt only).
- 5 new tests in `go-codec/envelope_legacy_test.go` (envelope-stamped, raw
  JSON↔CBOR both directions, garbage errors, custom-codec no-cross-retry —
  covering the "only built-ins covered" gap from the prior report).
- go-codec gates: `go test -race` green, jsonv2-tag mode green, golangci-lint
  **0 issues in both build modes** (fixed my own wrapcheck/wsl/nlreturn/perfsprint
  findings first).
- FEATURES.md + CHANGELOG.md (cut v0.2.0 section) updated in go-codec.
- **Committed `0a091ee`, annotated tag `v0.2.0`, pushed master+tag** (push
  explicitly approved).
- Consumer side: `event`, `kv`, `snapshot`, `command`, `query` go.mod bumped to
  v0.2.0 (resolved from the pushed tag); all five standalone builds+tests green.
- **Deleted all four local helper copies** — call sites now use
  `codec.DecodeEnvelopeOrLegacy`. The `art-dupl:accept` annotations went with
  them: the dep-isolation rationale is void once the helper lives in the shared
  dependency. Leftover-reference grep: clean.

### Q3: ErrBinaryNotFound semver

- Decision (v4.x patch, not v5 batch) recorded in CHANGELOG Removed entry and
  the brutal-review annotation.

### api-stability

- Golden regenerated (**4188 exports** — includes the concurrent session's 3
  new pgengine `Vector*` methods, which had landed without golden regen);
  `TestEvery…` meta-tests green. Run 1 of the gate had failed on exactly this;
  fixed mechanically per the documented procedure.

### event allocs tests

- Both `TestAllocs_NewEvent_*` pass **3× in both graphs** (workspace AND
  GOWORK=off standalone). Bounds live at ≤3 with an honest comment (see d).

## b) PARTIALLY DONE ⚠️

| Item | State | Blocker |
|------|-------|---------|
| `nix run .#verify-fast` GREEN | Run 1: benchkit timing failures (load avg 27–60 — concurrent session's pgengine work landed mid-gate) + api-stability golden (fixed above). Run 2 (load ~20, exclusive): **119 packages ok**; ONLY `cmd/cqrs-bench` fails (11 tests, one root cause: DuckDB CGo static link → `No space left on device`; /tmp tmpfs at 93%/3.8G free). Every module I touched is green in the gate. | Disk headroom for the DuckDB link + one more exclusive retry. |
| Q2: kv.Cache encoded-bytes rework | **Design complete, zero code written.** Cache stores raw envelope bytes: miss = new same-package `getRaw` + 1 decode; hit = 1 decode only; Set = 1 encode via same-package `setEncoded` (avoids double-encode; `kv/cache.go` already lives in package kv so internals are accessible). Isolation becomes structural (bytes are immutable) instead of copy-on-read. | Deliberately sequenced behind benchmark-first discipline + a green gate; machine was under heavy load all session. |

## c) NOT STARTED ⏳

1. `BenchmarkCache_Get_Hit/_Miss/_Set` baseline + rework + benchmark-regression
   gate (Q2 execution).
2. Post-deletion `nix run .#check-duplication` confirmation run (the four
   accepted groups are now deleted outright — expect the suppression story to
   simplify; baseline untouched).
3. AGENTS.md gotcha rewording: the workspace-allocs bullet I added earlier
   blames uncommitted `../go-codec`; the tree is now committed+tagged, but the
   workspace/standalone allocs delta persists via **unreleased id/metadata/
   record perf commits** (id/v4.5.0..HEAD, metadata/v4.5.0..HEAD,
   record/v4.3.0..HEAD). The bullet's fix instruction is now misleading.
4. `nix fmt` (treefmt) pass over the touched modules (gofumpt/goimports applied
   per-file already).
5. Corrective follow-up for daemon commit `ac5655b17` (see d) — code already
   fixed in working tree, needs the usual daemon sweep.

## d) TOTALLY FUCKED UP 💥 (owned)

1. **Misdiagnosed the allocs delta, then the daemon cemented it.** I attributed
   the workspace-vs-standalone allocation difference entirely to go-codec,
   tightened both bounds to ≤2, and the auto-commit daemon shipped
   `ac5655b17` whose message asserts "the published and workspace graphs now
   agree on 2" — **false**. Standalone still measures 3: the residual delta
   comes from unreleased id/metadata/record sibling commits (notably ULID
   formatting, fe7c1d2e5), which I only discovered after profiling + tag-range
   diffing. Code reverted to ≤3 with a corrected comment; the wrong claim is
   permanent commit-message history. Lesson: tighten a perf bound only after
   measuring BOTH graphs, never from a single-graph inference.
2. **Half-fixed the bounds**: first revert only touched `NoOptions`;
   `WithCorrelationID` still said ≤2 and failed the very next standalone run.
   Caught by my own `-count=3` discipline, but it should have been one edit.
3. **Ran gate run 1 into a loaded machine**: uptime showed 27–60 load avg
   (concurrent session) and benchkit's Duration-abort tests are documented
   load-sensitive. I checked uptime AFTER the failure instead of before.
   ~15 min wasted, 11 phantom failures.
4. **Run 2 omitted `GOTMPDIR`/`TMPDIR`**: the DuckDB CGo link then ate /tmp
   (93% full) and died. Both the tmpfs-fills gotcha AND the
   never-run-gates-on-loaded-boxes rule are in AGENTS.md — I knew both.
5. First draft of the go-codec test file used invented helpers (`str()`,
   `errUppercaseUnsupported`) — never compiled; full rewrite. Should have read
   the existing `envelope_test.go` conventions (testName/testEmail) first —
   which the corrected version happily uses.

## e) WHAT WE SHOULD IMPROVE 🛠️

- **Pre-gate liturgy** (make it mechanical): `uptime` < ~5, `df /tmp` > 10G
  free, `GOTMPDIR`/`TMPDIR`/`GOCACHE`… set, nothing else running — THEN launch.
  Both gate failures this session were environment, not code; both were
  documented anti-patterns.
- **Daemon-commit hygiene**: when a daemon commit encodes a wrong diagnosis,
  push the corrective commit in the same breath and note the message drift in
  the status report (done here).
- **Perf-bound changes**: measure both resolution graphs (workspace + GOWORK=off)
  before writing the number; the workspace silently overlays EVERY sibling's
  unreleased commits, not just the one you're thinking about.
- **The dependency-graph trap is now twice-burned** (go-codec last session,
  id/record this session): a tiny script `for m in deps: git log latest-tag..HEAD
  -- $m` next to any alloc-count failure would name the culprit in seconds.

## f) NEXT (prioritized, not exhaustive)

1. Free /tmp headroom (or point GOTMPDIR at a healthy location) → exclusive
   `#verify-fast` retry → expect GREEN (only cqrs-bench link env failing).
2. Implement Q2: `kv` benchmarks → encoded-bytes Cache rework → isolation tests
   stay green → `scripts/benchmark-regression.sh`.
3. Re-run `nix run .#check-duplication`; confirm the four deleted groups.
4. Reword the AGENTS.md workspace-allocs gotcha (id/metadata/record, not
   go-codec); add the dep-diff one-liner next to it.
5. `nix fmt` + `nix run .#verify` full gate before any next tag wave.
6. Tag wave for id/metadata/record (+ consumers) to collapse the allocs split
   → then tighten `TestAllocs_NewEvent_*` to ≤2 for real.
7. Sweep remaining 46 go.mod files still on go-codec v0.1.0 (examples, cmd/*,
   system, integration, testutil…) → v0.2.0 for consistency.
8. Consider mirroring `DecodeEnvelopeOrLegacy` mention into
   `.agents/skills/go-cqrs-lite/references/recipes.md` codec section.
9. Upstream idea: go-codec `BencharkDecodeEnvelopeOrLegacy` for the helper.
10. Status-report hygiene: fold this file's open items into TODO_LIST routing
    (Q2 item, tag-wave item, gotcha reword item).
11. Full next-50 backlog: see prior report §f — still authoritative.

## g) QUESTIONS (cannot resolve myself)

1. **Disk strategy for the gate retry**: /tmp is 93% full; `/mnt/buildcache`
   was marked CORRUPTED (I/O errors) in AGENTS.md on 2026-08-16, but `df` now
   shows it 50% used / 105G free and this session's writes
   (`/mnt/buildcache/tmp` linker dir in gate run 1) appeared to succeed. May I
   point `GOTMPDIR`/`TMPDIR` at `/mnt/buildcache/tmp` for the retry, or do you
   want to verify/repair it first (or name another location)?
2. **Q2 sequencing**: implement the kv.Cache encoded-bytes rework now
   (benchmarks first), or hold until `#verify-fast` is green so we chase one
   red thing at a time?
3. **History hygiene for `ac5655b17`**: the daemon commit's message contains
   the false "graphs now agree on 2" claim (my misdiagnosis, not the daemon's).
   Leave history untouched (corrective code+comment already in tree) or
   `git commit --amend` the message before anything pushes this branch?
