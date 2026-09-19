# Status Report: Queue Green Recovery + Fifth Config Incident — 2026-09-16 09:35

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (queue M4 — ADR-0134 tokens, T14 dep validation, FactTx/Watermarks, queue/mysql live-green; CHANGELOG 2026-09-19). Open remainder tracked in TODO_LIST "Durable Work Queue module": family tag wave (0/90 cut), README quickstart + conformance doc polish, PG `-race -count=2` symmetric leg. ARCHIVED.

> **STATUS (2026-09-16 docs-health pass):** §f3 HARVEST executed by the pass (queue/benchkit/CI sections of TODO_LIST). §f rows 1–2 (verify green, queue/postgres live conformance) remain open in TODO_LIST. The §g config questions (wrapcheck scoping, sixth-incident culprit) stay user-gated — root-causing the corruption loop is now a 🔥 TODO_LIST CI row.

**Scope of this report:** the 09:00–09:35 session ONLY (resumed from the
`2026-09-16_02-09_benchmark-statistical-rigor.md` handoff). Per instruction,
no new research beyond what this session observed.

**One-line state:** the queue module family landed red on 2026-09-15 and is
now fully green (duplication + lint + tests); a fifth `.golangci.yml`
config-corruption incident was found and recovered; repo-wide `#verify` is
blocked only by OTHER live sessions' in-flight work (`system/go.mod` was
mid-tidy at close).

---

## a) FULLY DONE (each verified green this session)

1. **Duplication gate RED → GREEN.** The 19 new clone groups in
   `queue/{postgres,sqlite,conformance}` (from the 09-15 auto-committed
   landing) are annotated with 42 `//art-dupl:accept` directives — both sides
   of every pair. Judgment: all are intentional dialect twins (pgx vs
   database/sql, `$N` vs `?`, ILIKE vs `LIKE ESCAPE`, string vs []byte
   detail), lockstep semantics pinned by the shared `queue/conformance`
   suite — same repo precedent as `command/asrecord.go` ↔ `query/asrecord.go`
   and the metaengine engines. Baseline untouched (no re-pin needed).
   Insertion was anchor-verified (script asserted each target line's content
   before inserting, bottom-up). `nix run .#check-duplication`: **0 clone
   groups, baseline 54 unchanged.**
2. **`queue/sqlite` go.mod fixed.** It had never been built `GOWORK=off`
   (go-error-family pinned v0.10.0 vs queue/v4's v0.10.1 requirement);
   `go mod tidy` applied, module now builds, vets, lints standalone.
3. **`benchkit/soak_test.go` gocyclo 24 → pass.** The 125-line
   `TestWriteSoakJSON_RoundTrip` split into `assertSoakResultRoundTrip`,
   `assertSoakSampleRoundTrip`, `assertSoakSamplePhasesRoundTrip` — identical
   failure messages, identical order. benchkit lint: **0 issues**; targeted
   tests pass (24.6s).
4. **Fifth `.golangci.yml` incident recovered (both halves).** Auto-commit
   `d54cd38a7` (09-16 08:11, 113 files) re-added `gci` to
   `formatters.enable` AND deleted the entire depguard allow-list block — an
   exact repeat of the fourth incident's double damage. Removed gci;
   spliced the 87-line depguard block back from `284d78ebe` (first splice
   attempt wrongly carried the old errcheck block — see §d2).
   `nix run .#check-lint-config`: **GREEN** (including the exhaustruct
   behavioral canaries).
5. **Queue family lint RED → GREEN: 241 findings → 0**
   (`queue` 98, `queue/sqlite` 73, `queue/postgres` 70 — the landing session
   never linted against the repo config).
   - Hand-fixes: exhaustive `task.Terminal` (all five cases — `Status` is
     string-backed, unknown values are runtime-real); wrapped the
     `queue.JSONCodec` decode error (`%w`); `errNilPool` package sentinel in
     postgres (err113); modernize ×3 in conformance (`slices.Contains`,
     `WaitGroup.Go`, `new(v)` — all compile on Go 1.26 and pass the suite);
     `tk`→`subject` suite-wide (94 sites; `tk` was an abbreviation,
     violating the no-abbreviations rule anyway); `st`→`status`/`current`,
     `by`→`dismissedBy` (param rename mirrored interface + both engines +
     helper); wsl_v5 blank-line fixes; `defer func() { _ = rows.Close() }()`
     (storage/sql idiom, 4 sites); inamedparam on `taskQuerier`; two
     documented `//nolint:nilerr` (best-effort reason decode, by design);
     one `//nolint:gosec` (G202 builder-generated WHERE, values
     parameterized).
   - Config (each with rationale in-line): exhaustruct_v5 ignore-patterns +=
     `task.New`, `task.Task`, `Filter`, `facts.Fact` (partial literals are
     the design: `Normalize()` defaults, store-assigned Seq/Time, zero-value
     match-all filters — same class as the existing `stack.Capabilities`
     entry); wrapcheck ignore-sigs += the database/sql + pgx driver
     surfaces (thin adapters pass driver errors raw by design; context is
     added by the calling layer — 76 per-site nolints avoided); mnd excluded
     for `queue/conformance/` (scenario literals ARE the scenario's meaning;
     `internal/cattest/` precedent).
6. **Post-fix ripple handled.** `nix fmt` (treefmt/golines) reflowed 13
   files; one casualty: my postgres mnd `//nolint` sat on a multiline
   statement's tail and detached. Replaced with a wrap-proof loop-built
   `$N` bind list (no magic numbers at all). Re-linted all four modules
   green post-format.
7. **Gates re-verified:** `#build` exit 0; api-stability "7092 exports OK"
   (param renames don't hit the golden); `check-changelog-symbols` — 86
   citations honest; ADR doc-assertion fixed by indexing ADR-0139 (v5
   encryption-at-rest, another session's artifact) in `docs/README.md`.
8. **Docs:** FEATURES.md coverage line 88+12 → **151+43** actual test
   functions (+ repeat/variation/P100 mentions); fifth-incident entry with
   symptom fingerprint appended to `docs/agents/gotchas-tooling-build.md`;
   CHANGELOG `[Unreleased]` "Fixed — lint-green queue family + fifth
   gci/depguard config recovery — 2026-09-16" section written.

## b) PARTIALLY DONE

1. **Full `nix run .#verify` to a green close: NOT achieved.** Every leg my
   changes touch is individually green (build, the five module lints,
   queue/sqlite conformance ×2, benchkit targeted tests, duplication,
   lint-config, api-stability, changelog-symbols, doc-assertions, treefmt).
   The full pass failed only on concurrent sessions' in-flight states:
   `cmd/doc-check/arity.go` (their broken WIP — they fixed it mid-session),
   DuckDB DDL parse (they fixed it mid-session), and `system/` (replay test
   - go.mod mid-tidy at 09:34 — still theirs, still open). I did not race
     them; a verify I ran would keep failing on their state through no fault
     of the tree.
2. **Previous session's three questions:** Q1 (queue clones) answered by
   action; **Q2 (tag wave timing) and Q3 (A/B benchstat vs per-metric CI
   gating) remain unanswered** (see §g).
3. **Full-repo lint leg after my config changes:** verified for the modules
   I touched; the wrapcheck/exhaustruct additions are suppression-only
   (cannot create findings elsewhere), but the standalone all-module lint
   pass only runs inside `#verify` — currently blocked per §b1.

## c) NOT STARTED

- **Q3 work in either direction:** A/B-by-revision benchstat workflow, or
  per-metric CI gating. Untouched (session was consumed by the red queue
  landing — the higher-urgency debt).
- **Tag wave** (needs user push approval; would strip the cqrs-bench→benchkit
  and queue/{sqlite,postgres}→queue sibling replaces).
- **The remaining ~39 items** of the 02-09 report's 44-task backlog (§f
  there): compare-command variation, per-run JSON, CI wiring for repeats,
  and the rest. This session consumed 5 of the top items (queue clones,
  soak gocyclo, FEATURES line; verify partially).
- **TODO_LIST.md harvest** of this report's §f (docs-health HARVEST — not
  run; user forbade unrelated research this session).
- **Root cause of the config corruption** (what keeps re-adding gci and
  deleting the depguard block inside auto-commit waves) — recovered twice
  now, hunted zero times.

## d) TOTALLY FUCKED UP (brutal honesty — process errors, all recovered)

1. **multiedit partial-application blindness (×2).** "Applied 1 of 2/3"
   results slipped past me: the claims.go `types/want` blank-line edit
   silently didn't apply (caught ~10 tool-calls later only because lint
   re-flagged it); the reads.go `a/b/c` rename's second edit failed on a
   comma-vs-semicolon guess and left a brief compile-broken intermediate
   (declarations renamed, usages not). Lesson: partial multiedit success is
   a failure state — verify every edit applied.
2. **Depguard splice attempt #1 was sloppy.** I extracted a sed line-range
   (131-235) without validating the block's internal structure — it carried
   18 lines of the OLD errcheck block, producing a YAML duplicate-key error
   and a failed gate run. I had the boundary evidence in hand and didn't
   check it. Attempt #2 (assert-bounded, depguard-only 87 lines) was
   correct.
3. **Non-canonical tooling first.** I ran golangci with `--default=none -E
   gocyclo` before the canonical invocation and burned a cycle being confused
   by gci findings that the real gate config wouldn't produce the same way;
   the gotchas file explicitly says the flake-pinned invocation is the only
   evidence-grade one.
4. **nolint placed where the formatter would break it.** My first mnd
   suppression sat on the tail of a multiline Sprintf; golines reflowed the
   args onto their own lines and detached the directive (nolintlint "unused"
   - 4 fresh mnd findings). I should anticipate treefmt in a repo whose
     verify formats on every run. Fixed wrap-proof; cost one extra cycle.
5. **Learned nolint same-line-only semantics by trial** (two failed
   placements) instead of reading the docs once. Same class as §d3.
6. **Mass rename via sed outside the edit tool** invalidated the edit
   tool's file tracking → mod-time rejections → a re-view/retry cycle.
   Collision-checked first, but the churn was self-inflicted.
7. **Formatted late.** First lint pass ran before `nix fmt`; 13 files
   needed normalization afterward. Format-first would have avoided the
   reflow surprise entirely.
8. **Attribution-based closure instead of a green verify.** Defensible
   (racing a live session editing `system/go.mod` is worse than waiting),
   but the honest claim is "my footprint green", not "repo green".
9. **Lucky, not rigorous, on the api golden:** predicted param renames might
   require a golden regen; the check passed, and I never confirmed what the
   golden actually records. Right outcome, unverified reasoning.

## e) WHAT WE SHOULD IMPROVE (systemic, observed this session)

1. **The config-corruption class has detection but not repair.**
   `check-formatters.sh` self-heals a re-added `gci`, but nothing self-heals
   a DELETED depguard block — two manual splices in two days (09-15, 09-16).
   The gate failing loudly is not the same as the gate fixing it. Add an
   auto-restore (pin a known-good block, or hash-pin the config sections) to
   `check-lint-config`.
2. **The root cause is still at large.** Something inside auto-commit waves
   (an agent running `golangci-lint fmt`? a config regenerator?) keeps
   re-adding gci and deleting depguard. Until identified, a SIXTH incident
   is a certainty — the incident log now has a symptom fingerprint
   (`gci: File is not properly formatted` on treefmt-clean files +
   "could not extract depguard allow list"), which is the early-warning.
3. **Broken intermediate states land on master via the daemon.** This
   session saw a repo-build-breaking `arity.go` and a 20+-minute untidy
   `system/go.mod` absorbed as `chore: auto-commit`. A daemon pre-commit
   `go build ./...` smoke (or per-file staleness marker) would stop
   publishing red states.
4. **errcheck exclude-functions short forms look dead.** `(*sql.Rows).Close`
   etc. did not match this session's findings (evidence: sqlite errcheck
   fired despite the entry) — golangci v2 likely wants fully-qualified
   `(*database/sql.Rows).Close`. My `_ = rows.Close()` idiom sidesteps it,
   but the config list is probably lying to future readers.
5. **Config exception lists grow one rationale at a time.** exhaustruct
   patterns, wrapcheck sigs, path exclusions — each defensible today, no
   pruning ritual exists. The gotchas file's probe method (delete pattern →
   re-lint → compare) should be run against SETTINGS entries periodically,
   not just paths.
6. **Session-loop discipline (self):** canonical command first; validate
   splice boundaries programmatically; `nix fmt` before lint; read linter
   docs before placing suppressions; treat partial multiedit as failure.

## f) Next tasks (up to 50; P1 = do first — §f is docs-health HARVEST fuel)

**P1 — close out this session's loose ends**

1. Re-run `nix run .#verify` to a green close once the concurrent `system/`
   session settles (go.mod tidy + replay test green) — the only blocker to
   "repo verify green".
~~2. Run `queue/postgres` conformance integration leg (needs a PG instance:~~
~~   `nix run .#integration-pg`) — the postgres engine has NEVER had its~~
~~   conformance suite executed in-repo (only build+lint+vet this session).~~ done 2026-09-16 — 15-02; TODO_LIST [x]
3. ~~HARVEST this §f into TODO_LIST.md (docs-health) — including retiring
   items done this session that may still be listed (queue clones, soak
   gocyclo, FEATURES line).~~ done (docs-health pass 2026-09-16) — P1 loose ends → TODO_LIST CI/Queue sections; P2 queue docs tail → TODO_LIST queue section; P3 benchkit → TODO_LIST benchkit section
~~4. Root-cause the config corruption: find what re-adds gci / deletes the~~
~~   depguard block inside auto-commit waves (daemon logs? an agent's fmt~~
~~   flow?). Then kill it.~~ done 2026-09-18 — TODO_LIST [x] CLOSED
~~5. Add depguard auto-restore to `check-lint-config` (mirror the gci~~
~~   self-heal; pin the known-good block).~~ done 2026-09-18 — restore-depguard.sh + golden

**P2 — queue family follow-through**
~~6. Fix the errcheck exclude-functions short forms (fully-qualify~~
~~`(*database/sql.Rows).Close` et al.) or delete dead entries; re-lint to~~
~~confirm which entries are load-bearing.~~ done 2026-09-16 — TODO_LIST queue docs tail (c)
~~7. Document `task.New` / `queue.Filter` / `facts.Fact` partial-literal~~
~~semantics in the queue package docs (the exhaustruct exemptions are~~
~~justified by design — say so where users read it).~~ done 2026-09-16 — queue/README
~~8. Consider a tiny queue/README or SKILL.md reference section for the~~
~~queue family (consumers currently discover it only via CHANGELOG).~~ done 2026-09-16 — queue/README.md shipped
~~9. `queue/mysql` is named in the Store doc comment as a future engine —~~
~~either implement behind the conformance suite or strike the mention.~~ done 2026-09-19 — engine SHIPPED (M4)
~~10. Add `queue` family to `references/modules.md` lookup (module map in~~
~~docs/agents/module-map.md likely lacks the 3 new modules — 91 go.mods~~
~~now vs "88" in older docs).~~ done 2026-09-16

**P3 — benchkit/cqrs-bench backlog (carried from 02-09 report §f)**
~~11. A/B-by-revision benchstat workflow (Q3 candidate A).~~ **Won't implement — 2026-09-19: per-metric CI gating chosen.**
~~12. Per-metric CI gating (Q3 candidate B) — gate on `MetricVariation`~~ done 2026-09-19 — TODO_LIST [x]
~~CoV thresholds in `benchmark-regression.sh`.~~ done 2026-09-19
~~13. `compare` command with cross-run variation reporting.~~ done 2026-09-19
~~14. Per-run JSON artifacts (one file per repeat run, not just the median).~~ done 2026-09-19 — benchmarks.yml noise gate
15. CI wiring for `--repeat` runs (benchstat multi-sample in the nightly).
16. Benchmark baseline refresh after this week's queue/benchkit churn.
17. `nix run .#load-sweep` before the next `#verify` (benchkit timing paths
were touched last session, sweep never run).

**P4 — repo hygiene**
18. Tag wave (AFTER user approval): strip sibling replaces (cqrs-bench→
benchkit, queue/sqlite+postgres→queue), pre-bump pins, GOWORK=off build
matrix, `scripts/tag-release.sh`.
19. Run the gotchas probe method against `linters.settings` exception
entries (exhaustruct patterns, wrapcheck sigs) — prune dead ones.
20. Daemon pre-commit build smoke (stop publishing red intermediates).
21. TODO_LIST.md full staleness audit (multiple sessions have landed since
its last refresh).
~~22. `docs/status/` older reports: annotate the 02-09 report with this~~
~~session's resolution of its top items (docs-health ANNOTATE).~~ done 2026-09-16 — 7th pass, same day
~~23. Check whether `system/` replay test failure (seen 09:18, foreign) got~~
~~fixed and captured by its owning session — if not, it belongs on this~~
~~list properly.~~ done 2026-09-19 — ADR-0143 root cause + instrumentation
24. ADR-0139 is a DRAFT skeleton from another session — either its owner
progresses it or it gets marked parked (I only indexed it).
25. CHANGELOG "Fixed — 2026-09-16" section will need its citations re-run
against the golden at tag time (check-changelog-symbols covers it).

**P5 — larger, still unscoped (from prior reports; not researched this
session, so verify before acting)**
26. cmd/doc-check arity checker (the other session's WIP): once landed,
add it to the recipe-gate docs so recipe fences stay arity-honest.
~~27. Vector-search contract tests (contract #26 says engine-native functions~~
~~verified 2026-09-15 — confirm the DuckDB DDL fix didn't regress the~~
~~fixed-`FLOAT[n]` casting).~~ done 2026-09-16 — cgo suite green
28. `#verify` exclusivity rule vs multi-session reality: document a
coordination protocol (who owns verify when two sessions are live).
29. FEATURES.md: other coverage lines may be stale the same way 88+12 was
(spot-check the biggest modules' counts).
~~30. Incident-log automation: a tiny script that diffs `.golangci.yml`~~
~~against `HEAD~` after each daemon wave and reports semantic changes~~
~~(would have caught today's incident at 08:11 instead of 09:05).~~ done 2026-09-18 — superseded: self-heal + nightly diff visibility

## g) Questions I can NOT figure out myself (max 3)

1. **Tag wave timing:** run it now (I would need your explicit approval to
   tag AND push — never without it), or wait for the system/ + doc-check
   sessions to land their WIP first? The wave strips three sibling replaces
   (cqrs-bench→benchkit, queue/sqlite→queue, queue/postgres→queue).
2. **Q3 priority:** A/B-by-revision benchstat workflow (answer "did this
   change help?" per revision) or per-metric CI gating (fail CI on noisy
   metrics via `MetricVariation`)? Both are ~day-sized; which first?
3. **Config policy ratification:** I widened wrapcheck `ignore-sigs`
   repo-wide for database/sql + pgx driver surfaces (16 signatures) rather
   than 76 per-site nolints. Keep repo-wide, or scope it to
   `queue/(sqlite|postgres)/` via path rules (more precise, more config
   surface)? Same question in principle for the four new exhaustruct
   ignore-patterns. And: do you know WHAT runs `golangci-lint fmt` or
   regenerates the config inside daemon waves — only you can identify the
   sixth-incident culprit.

---

_Point-in-time snapshot; goes stale immediately. §f is TODO_LIST HARVEST
fuel, not a commitment list. Written 2026-09-16 09:35 CEST._
