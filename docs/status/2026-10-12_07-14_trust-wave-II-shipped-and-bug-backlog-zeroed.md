# SUPERB Session Status — Trust Wave II Shipped, Bug Backlog Zeroed (2026-10-12 07:14 CEST)

> Executing the SUPERB Pareto Plan II mandate
> ([plan](../planning/2026-10-12_03-30_SUPERB-m06-closed-to-v5-cut-pareto-plan.md)).
> Continuation session after the 02:56 report. **Everything below is pushed to
> origin/master; tree clean at `e346a08a3`; ALL SIX open issues are now
> CLOSED; 9 new tags live on the proxy.**

## a) Fully done this session

- **N01 — bug trio closed with receipts.** #62 (FilterBindValue coerce-at-bind,
  5-assert mutation proof), #63 (json-tag-first resolution, cross-engine parity
  leg), #64 (C009 bare-`must` exemption + E005 system.RegisterCommand scanner
  tracking, verified at `scanner_calls.go:125`) — evidence-dump closure comments
  (voice-checker clean), all closed `completed`. Fix commits pinned: `1663fa940`,
  `f55c9e631`, `46187d98a`. Tag ETA stated honestly in each receipt (wave II).
- **N02 — post-P015 settle battery.** cqrs-lint (19 pkgs), metaengine,
  sqliteengine suites green; api golden regenerated (+2: `NewP015Detector`,
  `FilterBindValue`); `TestEvery` + `TestTagContentMatchesChangelog` green;
  **file-size gate repaired** (the P015 session's 400-line `v007_tables.go` →
  3-file split, no baseline growth); **duplication gate repaired**
  (`//art-dupl:accept` on the intentional core/v5 dedup mirror fork, both
  regions); F132 stale-gopls diagnostic verified no-op (`// indirect` already
  present).
- **M07 — V006 stopped-train ruling shipped.** `newestExistingTags` table
  (claiming → v4.0.2; refresh-at-release discipline like V003's
  `latestKnownMinor`); skip is module-scoped; 2 fixture tests; TODO row closed
  with receipt; catalog-counts test verified no-op (42/35/7 stable).
- **M08 — RawQuery coeffect-gate fix shipped.** `buildProjections` now extracts
  fold wire types from raw `QueryDecl`s (`rawQueryConsumedTypes`, value-interface
  assertion on `QueryFolds()`); 2 gate tests **mutation-verified** (pre-fix
  revision swap → 2 red; fixed → green); stale "RawQuery is opaque to the gate"
  doc lie fixed in `config_types.go`; core.md §3.9 updated; doc-check 1284
  refs valid.
- **N03 — TRUST WAVE II: 9 trains cut, pushed, proxy-smoked attempt-1.**
  schema/v4.7.0 → snapshot/v4.7.0 → decider/v4.8.0 → catalog/v4.8.0 →
  systemscenario/v4.1.0 → system/v4.13.0 → metaengine/v4.19.0 →
  metaengine/sqliteengine/v4.5.3 → cmd/cqrs-lint/v4.16.0. Every module ran its
  changed-set `GOWORK=off` suite before the cut (10-06 lesson honored); every
  tag in-tag verified via `git grep <tag>` symbol probes; `pin-sweep --check
  --remote` fully green; versions manifest fresh (112 trains); dated CHANGELOG
  wave section landed (tagged rows moved, untagged rows stay in `[Unreleased]`);
  changelog-symbols 12 citations green; master pushed.
  **Scope correction recorded:** the plan's bbolt/badger/pebble/mysql-engine and
  cmd/api-stability trains were REFUSED by `untagged-trains.txt` (ADR-0152) —
  manifest documents the refusal; `scheduling` classified pin-churn (deferred).
- **N04 — wave-C delta audit.** Submodule-aware audit (three broken script
  attempts fixed before trusting output): 3-class classification doc
  ([wave-C audit](../planning/2026-10-12_06-45_wave-c-delta-audit.md)) —
  wave-III seed (middleware 28, command 14, query 13, …), untagged roster
  (policy), pin-only verdicts (all wave-II tags clean). F129:
  `tag-release.sh --audit --baseline` → 1446 tags, **0 NEW violations**.
- **N06 — remaining bug trio zeroed.** #54 **fixed** (nil-receiver guard both
  providers; panic reproduced pre-fix with the issue's own one-liner; 2
  regression tests; receipt posted, closed); #56 **fixed** (documented the
  `Subscriber` io.Closer cleanup convention, mirroring `Bus`; receipt posted,
  closed); #58 **triaged** (the separable CBOR-through-SQLite round-trip pin
  already exists and is green; seam-vs-procedure decision → TODO_LIST
  owner-verdict row; closed as tracked).
- **M13 — idempotency retirement ADR + facade.** ADR-0158 Accepted
  ([0158](../adr/0158-idempotency-retirement-engine-dedupstore.md)): engine
  `DedupStore` capability is the primitive; both `idempotency/*` modules
  retire at v5 (deprecation markers already present). **`System.Dedup()`
  implemented** — named-"dedup"-engine-else-first resolution with
  capability assertion, errors name inspected engines; 3 tests (the
  no-capability fixture required the dedupless engine as a named secondary —
  primary must be a StreamLogBackend); ghost-verdict addendum in
  untagged-trains.txt; ADR index row (title==H1 lockstep fixed); api golden
  +1 (`system/method Dedup`); `check-adr-numbering` green (156 ADRs).
- **Incidental:** 16 modules' go.sum re-tidied after the wave's pin-settle
  automation (caught by `TestEveryModuleGoSumIsTidy`).

## b) Partially done

- **M14 (BDD harness depth 1) — scoping complete, zero code.** Read
  `equivalence.go`/`golden.go`/scenario surface. Design settled: F73 payload
  hashes = `TrailWithPayloads` masking the scenario's tracked stream IDs before
  sha256; F74 = pairwise per-stream diff in `AssertJournalEquivalence` failure
  rendering + first-appearance-ordinal order-caveat note; F75 = snapshot-
  configured decider fixture scenario. Not started because the session's
  remaining depth budget went to the wave.

## c) Not started

M05 (fleet adoption — needs sibling checkouts), N05 (P015 fleet roll), M20
(quiet composed `#verify` — load rebounded 13→52 when the window was evaluated;
`can-run-composed-gate --wait-loop` is the sanctioned form), M09–M11 (v5
deletion slices + breakage pair), M14 code (above), M15–M16, M17–M19 (graph
recipes/ADT waivers/docs-truth), M21–M22 (quiet MySQL + calibration legs), M23
(CI/infra: core/v5 eventtest fork, lint-config key-set pin, gofumpt exit-2,
pre-commit split), M24 (14th docs-health pass), N07 (stats-fold design), N08
(linter craft: ruletest --json counts, `--show-suppressed` parseability,
UPDATE_GOLDEN write-mode, README example), M25 (v5 encryption), M26 (long
tail), M27 (v5.0.0 cut — boss), N09 (bench/load hygiene).

## d) Totally fucked up this session

1. **The snapshot/v4.7.0 tag vanished between creation and push.** The
   auto-commit daemon raced `tag-release.sh`'s temp-commit window (reflog:
   daemon committed 32 files between temp-commit and `undo_temp_commit`), and
   the unpushed local tag was deleted by a mechanism I could not identify
   (suspect `fetch --prune-tags` in the daemon's sync; `systemctl` is blocked
   so I could not inspect the unit). Recovery: re-cut + push + smoke in ONE
   command. **Discipline codified: never let a created tag sit unpushed.**
2. **Three daemon commit races** (`cannot lock ref 'HEAD'`): two authored
   commits (art-dupl annotations; V006 ruling + tests) were absorbed as
   `chore: auto-commit` — content landed, attribution didn't. M08's commit
   won its race. Lesson re-confirmed: authored commits go in IMMEDIATELY at
   phase boundaries, targeted `git add` only.
3. **First CHANGELOG assembly produced a corrupt file** (duplicated header +
   wrong rows: `head -36` overlapped the rebuilt Unreleased block). Caught by
   post-edit seam verification; redone from committed HEAD with disjoint line
   ranges. No damage shipped.
4. **One invalid mutation-verification**: `git stash push -- <path>` from the
   wrong cwd silently no-oped ("No local changes to save" was swallowed), so
   the test "passed" with the fix still in place. Caught by checking
   `git stash list`; redone with the parent-revision swap
   (`git show <fix>^:<file>` → mutate → restore via `cp`). Lesson: stash from
   repo root only, verify the mutation took before reading the result.
5. **The audit script took four attempts** (inline-quoting hell; submodule
   conflation inflating root deltas; ownership test matching the module
   itself). Only the fourth produced trustworthy numbers — the earlier
   outputs would have misclassified 46-file pin-churn as content.

## e) Improvements / lessons

- **Wave-cut rhythm (works):** commit settle dirt → settle the module's own
  pins → changed-set tests → `tag-release.sh + push + --smoke` in ONE command →
  commit the artifacts + fresh settle dirt. Cut+push separation is what lost
  the first snapshot tag.
- **Tag-verification ritual:** `git grep <symbol> <tag>` for every wave row —
  catches "CHANGELOG says it, tag doesn't carry it" before publication.
- **Count maps live in test files** (`taskmanagerGoldenProfile`), not goldens —
  `CQRS_LINT_UPDATE_GOLDEN=1` prints "paste this" values; the golden FILE and
  the count map are two separate edits.
- **Settle automation creates the next cut's dirty tree** — expected and
  committable, not a failure.
- **Counted map drift after waves is EXPECTED** (settled pins reduce V003 lag
  findings): regenerate goldens AND hand-update the profile map, then full
  suite.

## f) Next 50 items (priority-ordered seed for the next session)

1. M14 code: F73 payload-hash trail, F74 per-stream equivalence diffs, F75
   snapshot-interplay scenario (design above).
2. M20 quiet `#verify` via `can-run-composed-gate --wait-loop` (dedup tail (a)
   + S03 re-record + verify-chain v4).
3. M05 fleet adoption: bank-sync state-shape patch (wave II shipped
   snapshot/decider), DiscordSync CBOR-workaround drop (schema/v4.7.0),
   cqrs-htmx UpcasterRegistry collapse.
4. N05 P015 fleet roll onto cqrs-lint/v4.16.0 (audit-or-suppress hits).
5. M09 v5 deletion slice 1 (listing type-driven status + taskmanager
   OffOnTombstone pre-reqs, then Materialize/compat-shells/BuildWhere/tombstone
   metadata API).
6. M10 slice 2 (RelationalProjection/view/GraphProjection/Bundle+presets/
   transport removal from lists).
7. M11 breakage pair (`record.NewStreamRef` validation; core/v5
   `TombstoneFilter` + sort-directive type).
8. M23 CI/infra batch (core/v5 eventtest fork; `#check-lint-config` key-set
   pin; gofumpt exit-2 root-cause; pre-commit split).
9. M17 graph recipes + ADTSet recipe + Then* docs (ADR-0157 trigger fired).
10. M18 graph-native `--dgraph` ephemeral leg + go-appkit GracefulClose.
11. M19 docs-truth XS batch (COOKBOOK into md-go walk; `scenario/v4` TB
    refactor; docs-health remainder; bench-gate single-mention).
12. M24 14th docs-health pass (~28 live 10-01..12 reports to
    annotate/archive).
13. N07 stats-fold design (capture RM vs day-buckets; suppression-expiry
    linkage for the 5 P015 sites).
14. N08 linter craft batch (F112–F115).
15. M15/M16 harness depth 2 + cqrs-htmx sqliteDeployment variant.
16. Wave III (from N04 seed): inspect middleware/command/query/storage/otel/
    id/scheduling deltas → minor/patch sweep.
17. M21/M22 quiet MySQL + calibration legs (after M20).
18. #58 verdict (owner): store-override seam vs migration helper.
19. M25 v5 encryption completion (owner 4-question pack first).
20. M26 long tail; M27 v5.0.0 cut assembly; N09 load hygiene; V006-table
    refresh at every cqrs-lint release (standing note).

## g) Questions I could not answer myself (max 3)

1. **Tag-deletion mechanism.** Something removed the unpushed
   `snapshot/v4.7.0` local tag mid-cut (daemon race confirmed; deleter
   unknown — `fetch --prune-tags`-shaped). May I read the daemon's systemd
   user unit / script (tell me the path, or grant `systemctl --user` read), or
   should I just codify the one-command cut as policy and stop investigating?
2. **M20 launch shape.** May I start `can-run-composed-gate --wait-loop` in a
   background shell now (it grabs the exclusive gate whenever load5 drops
   under the ceiling, possibly hours from now), or do you want the composed
   `#verify` to own a dedicated quiet block with nothing else running?
3. **Permanently-untagged CHANGELOG rows.** The api-stability mirror-gate and
   engine-fix rows now sit in `[Unreleased]` forever (their trains never tag
   per ADR-0152). Keep them there (honest, gate-verified), or introduce a
   dated "repo-only (untagged trains)" section so `[Unreleased]` converges to
   empty before each wave?

--
💘 Generated with Crush
