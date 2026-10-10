# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, 2026-09-08, 2026-09-11, 2026-09-16, 2026-09-19,
2026-09-20, 2026-09-21 ×2, 2026-09-22 — the 12th pass (2026-09-28) archived
the 09-20..28 accumulation: the T18b arc ×6, the unblock/verify/data-mesh/t4/
dedup/publish-integrity clusters, and five executed SUPERB plans, all
RESOLVED-BY-ROUTING-bannered).
The Declined section at the bottom is a do-not-re-litigate guard, not a backlog.

> **Prioritized execution plan (2026-09-08):**
> [`docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`](docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md)
> ranked the then-list into Pareto waves (W0 release train → W1 trust →
> W2 efficiency → W3 v5 train) and was EXECUTED through 2026-09-11 (W3's v5
> items live in the v5 section below; the user-gated P22 halves remain in the
> Turso section). **Unblock·Prove·Deliver plan (2026-09-22 01:25, EXECUTED;
> archived 2026-09-28):**
> [`docs/planning/archived/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`](docs/planning/archived/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md)
> (T01–T27; the T04 composed-verify remainder lives in the CI section below,
> the T08/T09 tag waves in the Release section; predecessor:
> [2026-09-20 17:40 owner-unblock plan](docs/planning/archived/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md),
> archived — M-items were folded into the T-numbering). **Data-mesh plan (2026-09-23 22:12,
> EXECUTED; archived 2026-09-28):**
> [`docs/planning/archived/2026-09-23_22-12_SUPERB-data-mesh-federation-pareto-plan.md`](docs/planning/archived/2026-09-23_22-12_SUPERB-data-mesh-federation-pareto-plan.md)
> (T01–T27 + 72 fine tasks) — shipped 2026-09-23/24 (reports:
> [12-26](docs/status/archived/2026-09-24_12-26_data-mesh-pareto-execution-session.md),
> [13-32](docs/status/archived/2026-09-24_13-32_data-mesh-completion-session.md));
> the open tail is the section below. **Current plan (2026-09-28 01:26):**
> [`docs/planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md`](docs/planning/2026-09-28_01-26_SUPERB-publish-integrity-pareto-plan.md)
> (M-tasks; M1–M4, M6, M14–M16, M19, M23–M25 done with receipts in this file;
> open: M5+M20 quiet-window campaign, M22 owner Q3). **Current plan (2026-10-03 17:20):**
> [`docs/planning/2026-10-03_17-20_SUPERB-full-todo-pareto-plan.html`](docs/planning/2026-10-03_17-20_SUPERB-full-todo-pareto-plan.html)
> (full-list Pareto master plan: 27 medium tasks / 118 fine tasks over all 112
> open rows + 9 open issues; truth-strike + #49/#50/#51 triage executed same
> day; §07 = the 28-ruling owner decision bundle). **Current plan (2026-10-08
> 14:59):**
> [`docs/planning/2026-10-08_14-59_SUPERB-v5-goal-pareto-plan.html`](docs/planning/2026-10-08_14-59_SUPERB-v5-goal-pareto-plan.html)
> (v5-GOAL Pareto plan: 27 medium tasks / 150 fine tasks ≤12 min over ALL 109
> open rows, sequenced onto the v5-cut-readiness Layers 0–13 — W0 ruling pack +
> publish waves → W1 quiet-window legs → W2 pre-cut migrations → W3 branch +
> deletion cascade → W4 universal fold/encryption/FilterOp → W5 docs + THE CUT
> → W6 closure; graph:
> [`.d2`](docs/planning/2026-10-08_14-59_SUPERB-v5-goal-pareto-plan.d2)). This
> file remains the living source of truth.

> **DONE (2026-10-10):** Graph-native adoption closure + operator-story wave
> (plan
> [`…V2.md`](docs/planning/2026-10-10_06-11_SUPERB-graph-native-adoption-closure-and-operator-story-V2.md))
> executed to completion: `example/graph-native` (BDD-green, all gates),
> recipes §2.44 + advanced §6.13 modern-first, COOKBOOK graph chapter,
> ADR-0157 (engine fleet two-level story, measured) + ADR-0156 (edge labels
> at v5), T18 ADT-recipe ratchet (5 visible waivers below), T19 consumer
> dry-run GREEN after 2 iterations. Graph-native is now
> **adopt-if-needed**: no fleet consumer currently requires it; when one
> does, the recipe + example are the entry points.
>
> **AMENDED 2026-10-10 (same day, harvest session):** the "no fleet consumer"
> framing is stale — the Kith CRM relations feature consumed the Graph ADT
> the same morning (first real consumer; see the CRM debut section at the
> bottom). The adopt-if-needed posture stands, but the trigger HAS fired for
> the recipe-polish row below it.

## Section index

[Legend](#legend) ·
[GitHub issue backlog](#github-issue-backlog-2026-09-30-plan) ·
[New consumer issues](#new-consumer-issues-2026-10-03-triage) ·
[Data-mesh & federation tail](#data-mesh--federation-tail-2026-09-24) ·
[Metaengine Universal Storage Substrate](#metaengine-universal-storage-substrate-proposed-2026-09-18) ·
[V5 schema evolution](#v5-declarative-schema-evolution-proposed-2026-10-09) ·
[Durable Work Queue](#durable-work-queue-module-proposed-2026-09-13) ·
[Command-side depth](#command-side-domain-depth-2026-09-13-plan) ·
[Reset-projection stall](#investigate-testsystem_resetprojection_restartandreplay-contention-stall-found-2026-09-13) ·
[Turso matviews](#turso-materialized-views-adr-0135--upstream-handoffs) ·
[Cordis follow-ups](#cordis-spatiotemporal-composability-follow-ups-2026-09-10) ·
[cqrs-lint](#cqrs-lint) ·
[Release / Tagging](#release--tagging) ·
[Metaengine follow-ups](#metaengine--follow-ups) ·
[CI / Infrastructure](#ci--infrastructure) ·
[Code Quality](#code-quality) ·
[v5 Unification](#v5-unification-phase-8-deletion--cut) ·
[Docs truth](#docs--consumer-surface-truth) ·
[benchkit tail](#benchkit-statistical-rigor-tail-2026-09-16) ·
[CV verdicts](#cv-consumer-verdict-follow-ups-2026-09-16) ·
[Goal-closure](#metaengine-goal-closure-follow-ups-2026-09-17) ·
[Watermill skill](#watermill-sibling-skill-follow-through-2026-09-15) ·
[md-go-validator](#md-go-validator-ci-integration-from-2026-09-13-audit) ·
[Temporal cells](#temporal-versioned-cells--adr-0141-follow-ups-harvested-2026-09-18) ·
[go-graph-rag](#go-graph-rag-feedback-follow-ups-2026-09-15-triaged-2026-09-19) ·
[92-tag tail](#92-tag-release-train-tail-2026-09-20-harvest) ·
[Upstream asks (cqrs-htmx)](#upstream-asks-from-cqrs-htmx-harvested-2026-09-22-docs-health-d1) ·
[Skill hard-block refocus](#go-cqrs-lite-skill-hard-block-refocus-2026-09-24-harvest) ·
[Declined](#declined--rejected-do-not-re-litigate)

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `~~Struck~~` = done sub-part of an otherwise-open row (kept for context;
  fully-completed rows are deleted outright per the header policy)
- _(Effort: XS/S/M/L/XL)_ = rough size
- **Receipt convention (2026-09-28, publish-integrity plan §6.9):** every
  "verify" task closes its row with a DATED receipt sub-bullet stating what
  was observed (numbers, commands, links) — verdicts like "confirmed / stale
  claim / already fixed" never live only in chat or session reports. Four
  stale claims were killed this way in one session; keep the discipline.

---

## GitHub issue backlog (2026-09-30 plan)

> All 10 open issues reviewed and verified against master HEAD (`004298c1a`)
> on 2026-09-30. Execution plan (27 tasks / 88 fine tasks, Pareto waves):
> [`docs/planning/2026-09-30_14-28_SUPERB-github-issue-backlog-pareto-plan.html`](docs/planning/2026-09-30_14-28_SUPERB-github-issue-backlog-pareto-plan.html)
> (graph: [`.d2`](docs/planning/2026-09-30_14-28_github-issue-plan.d2)).
> Rows below are the living source; the plan is the point-in-time snapshot.

- [x] 🔥 ~~**W1 — consumer unblock (the 1%):** #26 retract `stack/postgres/v4
      v4.2.0` + #21 finish watermill wire fix + one release wave (watermill
      v4.7.0, stack/postgres v4.4.2)~~ — **DONE 2026-10-03 (receipt):** #21 was
      already fixed AND released — `watermill/v4.6.3` exists on remote + proxy
      (tagged 2026-10-03 05:49 in the 11-module batch; `eventToMessage` writes
      scalar `correlation_id`/`causation_id` via shared `writeTracing`,
      `TestEventToMessage_TypedCausationRoundtrip` green; issue closed with
      receipt). `stack/postgres/v4.4.2` also already tagged; the missing half —
      the `retract v4.2.0` directive — added to `stack/postgres/go.mod` this
      session (tidy+build green; reaches the proxy with the next tag; #26
      commented, stays open until published). No new wave needed. — issues #21 #26
      ~~_(Effort: M)_~~ actual: XS
- [x] 🔥 ~~**W0 — close #25:** requested tag `metaengine/projectionadapter/v4.5.0`
      ALREADY EXISTS on remote (`8c87c48a6`, verified 2026-09-30) — verify
      `OccurredAt` in tag, comment receipt, close. — issue #25~~ — **DONE 2026-10-01
      (stale-row receipt 2026-10-03): issue #25 CLOSED 2026-10-01T15:13:53Z**
      (`gh issue view 25`: state CLOSED — the 2026-10-01 session tagged, verified,
      and closed). Delete at next docs-health pass.
- [ ] **W2 — linter trust (the 4%):** ~~#42 port CLI's none-import guard into
      `toolspec.detect` (guard lives only at `cmd/cqrs-lint/run.go:326`;
      provider path lints non-consumers, verified)~~ DONE 2026-10-03 (issue #42
      CLOSED 01:34:38Z; shipped as the cqrs-lint v4.13.1 clean-verdict wave,
      CHANGELOG `[Unreleased]` receipt: toolspec detect/repair share the CLI's
      `loadVerdict`, `TestBuildContext_SilentEmptyBrokenGraphFailsLoudly`
      regression) + ~~#43 D005 stops treating the first version token on a
      go-cqrs-lite line as the doc's version claim~~ DONE 2026-10-03:
      positional attachment shipped by the same-day CV-feedback session
      (CHANGELOG `[Unreleased]` entry — connector words + historical lead-ins,
      both issue repros pinned as unit tests); D005 tests re-run green this
      session; issue #43 closed with receipt.
- [ ] **W3 — consumer features (the 20%):** ~~#32 `EventAdapter.LoadByEventID`
      via `EventByIDBackend` capability (storage side exists; lost in wrapper
      layer)~~ DONE 2026-10-01 (issue #32 CLOSED 16:16:39Z; CHANGELOG
      `[Unreleased]` receipt: `metaengine.EventByIDBackend` capability + sqlite
      partial expression index + `systemtest` E2E) · ~~#35 request-context
      enricher~~ DONE 2026-10-01 (M24, shipped as `event.RequestScope` +
      `WithRequestScope` + `RequestScopeEnricher`, recipes §2.42 — cqrs-htmx
      carries the drop-local-copy TODO for its next bump) · ~~#27 committed
      module → latest-tag manifest (versions.json) + tag-push CI + README
      matrix~~ DONE 2026-10-04 (T09: `scripts/check-versions-manifest.sh`
      regenerates BOTH artifacts — versions.json (108 trains, key `.` = root
      train, semver-aware latest pick) + a collapsed README section between
      sentinels; nightly `--check --remote` leg fails on BOTH drift directions
      (missing published tag, or manifest citing an unpushed tag);
      `tag-release.sh` refreshes the manifest at tag time — the "updated on
      tag push" equivalent; hermetic `--self-test` with stale-manifest +
      unmanifested-tag mutation legs runs in the nightly leg; local vs
      origin tags verified identical at landing; issue #27 CLOSED 2026-10-04
      with receipt, ask-3 stays owner-gated below) · ~~#28 document
      the consumer upgrade sweep pattern + bless `cmd/cqrs-upgrade` (exists,
      tagged v4.0.0 2026-09-07 — ask reduces to docs)~~ DONE 2026-10-01 (issue
      #28 CLOSED 16:24:01Z; the cqrs-upgrade `--json --strict` holes (a)(b)(c)
      + NoPins scan shipped in the same CHANGELOG `[Unreleased]` wave).
      — issue #27 _(Effort: M)_
- [x] ~~**W4 — structural tail:** #36 move `WithMetaEngine` + Bundle
      registration into a `stack/metaengine` module (4 root files import
      metaengine: options/bundle/accessors/materialize); deprecated root
      forwarders until v5; full new-module gate sweep + hermetic
      metaengine-free-graph probe.~~ — **DONE 2026-10-08 (T06, pulled
      forward as a W1 blocker):** root `stack` keeps only the deprecated
      `MetaEngineStore` Close() seam (`WithMetaEngine`/`Bundle.MetaEngine`
      compile unchanged for concrete-store callers); the new
      `stack/metaengine/v4@v4.0.0` module carries `stackmeta.WithStore`/
      `stackmeta.Store` registration + concrete recovery; root require
      graph verified metaengine-FREE (`GOWORK=off go mod graph`, zero
      edges). Shipped as the dependency-first mini-wave stack/v4.5.0 →
      stack/metaengine/v4.0.0 → stack/sqlite/v4.3.5 (drops its temp
      replace, rides stack v4.5.0 via MVS), all three pushed +
      proxy-smoked attempt 1; standalone builds + `-short` suites green;
      new-module gate sweep done in the prior session; CHANGELOG dated
      section cut + versions manifest fresh; issue #36 CLOSED 2026-10-08
      with receipt. — issue #36 _(Effort: L)_
- [ ] [RULED 2026-10-08] **#27 ask-3 (owner): CI annotation of modules whose
      master HEAD is ahead of their latest tag** — **RULED: non-blocking
      workflow-summary annotation** (a `$GITHUB_STEP_SUMMARY` table — zero
      check-run noise budget impact; helps `replace`-to-master pin judgments).
      Implementation rides the versions-manifest nightly leg (S). Manifest
      landed 2026-10-04 (row above). — issue #27 item 3 _(Effort: S)_

---

## New consumer issues (2026-10-03 triage)

> Filed 2026-10-03 by consumer sessions (nsfw-classifier / go-aichat chatstore);
> not previously tracked here. Execution slots live in the 2026-10-03 plan
> (T06/T08/T10, fine tasks f28–f58).

- [x] ~~**#49 `middleware.CommandRetry` never retries Conflict — undocumented**~~ —
      **DONE 2026-10-03 (same session as triage):** documented on
      `middleware.CommandRetry` + the `RetryConfig.IsRetryable` field (default is
      Transient-only; explicit override recipe shown for journal-reloading
      pipelines), FAQ entry added (command-side pitfalls), dated correction note
      on the archived 2026-05-01 roadmap family table; middleware module tests
      green, doc-check 1,169 refs green, CHANGELOG entry gated. Issue closed with
      receipt. The classification change stays a tracked design question.
- [x] ~~**#50 `time.Time` payload fields drift sub-µs through system+sqliteengine
      journals**~~ — **DONE 2026-10-04 (T10):** traced NOT to the journal —
      `encodeStreamValue` stores exact JSON strings; the loss is the DEFAULT
      CBOR event codec (go-codec `canonicalEncMode` sets `TimeUnixDynamic` →
      fractional times become float64 unix seconds → ~22 fraction bits at
      2026 epochs = ≤256ns/hop; measured 165ns single-hop, the consumer's
      611ns is multi-hop). Pinned by `systemtest/time_fidelity_test.go`
      (JSON-codec leg nano-EXACT 0ns end-to-end through system+sqlite; CBOR
      default leg bounded sub-µs with measured drift logged; codec-level
      subtests pin 165ns/0ns). ADR-0056 carries a dated amendment (wrong
      parenthetical + consumer guidance: JSON codec / `Instant` /
      ±256ns-per-hop tolerance); upstream default-flip filed as
      LarsArtmann/go-codec#4 (`TimeRFC3339Nano`: 0ns, 36B, decode-compatible;
      `TimeRFC3339` rejected — truncates to seconds); iroh `opEncMode` keeps
      `TimeUnixDynamic` deliberately (LWW ordering survives quantization).
      FAQ entry added; issue #50 CLOSED 2026-10-04 with receipt. — issue #50 _(Effort: M)_
- [x] ~~**#51 A013 suggests value-embedding `BasicCommand`, which cannot satisfy
      `command.Command`**~~ — **DONE 2026-10-03 (same session as triage):** rule
      inverted (fires on VALUE embeds at warning severity with the
      compile-failure rationale + pointer-form suggestion; pointer embeds
      silent), RULES.md/README/catalog re-pinned, taskmanager golden regenerated
      (10 → 0 A013 findings), both directions unit-pinned, full cqrs-lint suite
      green (19 packages). Issue closed with receipt.
- [x] ~~**Owner decision bundle (28 rulings)** — consolidated table at~~
      **ANSWERED 2026-10-08 (owner blanket execution authorization "GET SHIT DONE —
      the whole list")**: every ruling adopted its documented recommendation and
      landed on its home row with dated receipts below; ADR-0150 (SingleWriter
      lease) + ADR-0151 (Goal direction HYBRID) written; ROADMAP OQ1–OQ17 answered
      inline. The genuinely-external user actions remain open: GitHub billing,
      ERRAUDIT_PAT, F153 license choice, evals CLI access.
      [`docs/planning/2026-10-03_17-20_SUPERB-full-todo-pareto-plan.html`](docs/planning/2026-10-03_17-20_SUPERB-full-todo-pareto-plan.html) §07
      _(Effort: XS per reply)_

---

## Data-mesh & federation tail (2026-09-24)

The 2026-09-23 plan (T01–T27, f01–f72) is COMPLETE — CHANGELOG `[Unreleased]`
receipts for the exporter wave (manifest, `WithSkipBootstrapFiles`, `WithPlainRefIDs`,
`Flow.Owners`, docserver DataProducts, E019, mesh-demo, cookbook recipes, gRPC v5
guide); hub-side work (merge gate, per-source governance lint, stale-source probe,
mesh-demo onboarding) lives in the eventcatalog-hub repo. Execution evidence:
[12-26 report](docs/status/archived/2026-09-24_12-26_data-mesh-pareto-execution-session.md) ·
[13-32 report](docs/status/archived/2026-09-24_13-32_data-mesh-completion-session.md) ·
[18-14 hub-phase report](docs/status/archived/2026-09-24_18-14_hub-phase-and-final-gate-session.md)
(all archived 2026-09-28, RESOLVED-BY-ROUTING). Open tail:

- [x] ~~**mesh-demo: system.New-backed variant** — runtime coeffect-gate demo (current
      demo is pure deciders).~~ DONE 2026-10-03 (M25): `mesh-demo gate` composes
      the orders context through `system.New` — a typo'd import is rejected at
      construction with `ErrDanglingEventSubscription` naming the dangling
      subscription, and the contract universe composes cleanly over a memory
      engine (`TestGate_TypoCaught`/`TestGate_ContractUniverseComposes`; CHANGELOG
      `[Unreleased]` receipt). — source: 12-26 §f22, 13-32 §f20

---

## Metaengine Universal Storage Substrate (proposed 2026-09-18)

> Owner directive 2026-09-18: metaengine becomes the ONE way data is stored/retrieved from disk.
> Full Pareto plan (23 tasks / 82 micro-tasks): [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

- [ ] **T19–T21 (v5-gated): fold capabilities into universal `Engine`, delete the duplicate SQL stacks, release train** — blocked on the v5 train per ADR-0142 §decision; DO NOT execute in v4.x (growing core interfaces is breaking, contract 21g discipline). The tag waves for claiming + the queue family are the separately-tracked item below. _(Effort: L; v5-gated)_
  - **2026-09-28 (M23 receipt):** the v5-REMOVAL CENSUS already exists as gate machinery — `cmd/cqrs-lint/pkg/rules/version/v007_tables.go` (curated removal surface: 10 whole modules incl. the 8 stack presets + storage/relational + storage/view; ~50 symbols incl. `On`/`OnTyped`, `Infer`, `InferFromNamedEvents`, `Bundle`, the ADR-0126 shells) held bidirectionally against the source by `v007_drift_test.go` + `v007_drift_scan_test.go` (every in-source v5 `Deprecated:` marker must have a table/allowlist entry and vice versa; ≥90-marker scanner floor). Drift tests re-run green 2026-09-28. No separate census doc is needed — do not hand-maintain a list that the gate already derives.
  - **2026-09-28 (M24 receipt):** row currency CONFIRMED — ADR-0142 remains Accepted, its §Decision 2 ("capability-interface path in v4.x; universal fold at v5; growing core interfaces is breaking, contract-21g discipline") matches this row's citation verbatim in substance; no drift. (Its "folds the existing 12 backends" figure is a dated-record count — bigtable arrived the same day — and ADRs are history, not living docs.)

- [x] ~~[BLOCKED] **T18b tail: two owner questions**~~ — **RULED 2026-10-08
      (blanket authorization):** (a) deadline-lapse policy = **auto-re-arm**
      (self-healing, matches calibration-gate philosophy; capped so a dead
      quiet-window never spins); (b) host benchmark-ceiling policy = **strict
      <5 stands**. The chain-hardening ask itself is MOOT: the armed
      pipeline landed green 2026-09-21 18:14 UTC and was retired
      (`/var/tmp/t18b` trashed); the storm/reboot survival record and re-arm
      mechanics live in the canonical record
      [`docs/benchmarks/2026-09-20-21_t18b-record.md`](docs/benchmarks/2026-09-20-21_t18b-record.md)
      (the campaign queue continues via the calibration row in Metaengine
      follow-ups; the gate-semantics ADR + case-study appendix via the T17 row
      in Docs truth). — source: 16-37 §d2/§e2/§f4/§f20, 14-18 §d3

- [ ] **Compound-cursor issuance (M06 tail, 2026-10-06):** engines now ACCEPT
      tie-safe `SortKeyCursor{Sort, Key}` cursors (shared `SortPaginate`
      core; all nine engine `MapScan`s delegate — memory, sqlite, pg, mysql,
      duckdb, dgraph, bbolt, pebble, badger), but issuance still mints
      value-only cursors: `TypedReader.ScanPage` reflects the last item's
      sort field. Ship the protocol: engines fill a `ScanResult.NextCursor`
      (they hold the pair keys), `ScanPage` prefers it over reflection, and
      `ParseCursor` normalizes the round-tripped `{"Sort":…,"Key":…}` JSON
      form back to the struct (`compareValue` already tolerates the float64
      drift). Until then the fix is opt-in — callers construct compound
      cursors by hand. _(Effort: M; wire-format change — golden pins
      required)_

## V5 declarative schema evolution (proposed 2026-10-09)

> Research-informed proposal (Axon/AxonIQ upcasting + LiveStore declared-schema/rematerialization model + Equinox/Akka cross-check), primary-source-verified, ruling resolved 2026-10-09 (T2-first sequence adopted): [`docs/planning/2026-10-09_v5-declarative-schema-evolution.md`](docs/planning/2026-10-09_v5-declarative-schema-evolution.md). One declared schema unifying the four disconnected half-mechanisms; both axes ride existing machinery (SourceTransform chains; journal-survives-reset + ConfirmRebuild). **Fleet evidence:** bank-sync wrote a ~45-line upcaster closure + hand-rolled `NewFieldRenameUpcaster` + a `migrate-journal` runbook; DiscordSync maintains a 211-line upcasters package (two type renames + derived-field upcasts); cqrs-htmx surfaces SchemaVersion with zero upcasters. Recommendations for all three open questions are IN the doc §12 (schema/ home via layering; metadata stamp first; promote after named-consumer burn-in).

- [x] **Owner ruling on the proposal** — RESOLVED 2026-10-09 (delegated via standing execution loop; recommendations adopted): declaration home = `schema/` (catalog renders; layering forces it); T3 stamp = metadata first, `Record` stamp only at v5 if burn-in favors; warn-first promotion = after DiscordSync + bank-sync each run one clean minor cycle with the ledger on; sequence T2→T1→T4→T3→T5→T6/T7; T2 v4.x-additive; fleet migration opportunistic (bank-sync pilot first). Implementation ADRs split out per increment.
- [x] **T2 named upcast ops** — DONE 2026-10-09: `schema/ops.go` + `schema/chain.go` + `schema/chain_engine.go` (7 ops `RenameType`/`RenameField`/`AddField`/`RemoveField`/`Transform`/`Split`/`Drop` → `Compile` → `Chain.SourceTransform()` batch-level + `Chain.Upcasters()` 1:1 bridge; order-independent most-specific matching; duplicate/rename-cycle/param rejection at Compile; per-op `WithDecodePolicy` Fail|Passthrough|Drop; encoding-aware via `codec.ForEncoding(evt.Encoding())`; identity-preserving rebuilds). Table-driven tests incl. bank-sync's money reshape as golden; CHANGELOG + README + faq/modules skill refs updated; api golden regenerated (8421 exports); schema module lint 0 findings. **Pilot PROVEN via temporary sibling replace** (bank-sync collapse: 116-line hand-rolled upcasting.go + 2 wiring sites + 3 test files onto `UpcastChain()` — build ✓, internal/cqrs tests ✓ incl. -race; pre-existing failures in internal/storage+server tests are ANOTHER session's in-flight work; scaffolding reverted, patch artifact: `docs/planning/2026-10-09_t2-pilot-bank-sync.patch`). Adoption rides the next schema/v4 tag.
- [x] **T1 the declared schema, extending `system.DomainConfig` (ADR-0123)** — CORE SHIPPED 2026-10-09: `schema/declaration.go` (`EventSchema`, `Event`, `Declare` — per-declaration validation: non-empty/unique types, op-target-match, op source < current version; compiles ONE cross-declaration chain) + `system.DomainConfig.Schema` field + `system/schema.go` (`applySchemaDeclaration` decorates `sys.eventStore` after ALL assignment paths → decider loads AND projection-host journal see current payloads; `declaredEventTypes` merges Schema types into the coeffect universe) + `system/schema_test.go` (read-path upcast, SeekableJournal survives, gate joins, invalid-declaration rejection). `#check-arch` green (system budget 20→21), api golden 8426, system module lint 0 findings, my clone group suppressed via `//art-dupl:accept`. **REMAINDER (follow-up increments):** projectionadapter `TypeDecoder` registration derivable from Schema; catalog render from Schema; cqrs-lint undeclared-event rule; `system/go.sum` pins ride the next tag wave (workspace-mode green now, GOWORK=off per-module needs schema v4.6.0 tagged).
- [ ] **T4 snapshot state-shape stamp** — mismatch discards the snapshot (counted stat) and rebuilds from the journal (ADR-0136 replayable rung, ADR-0143 journal guarantee). Fixes today's silent stale-snapshot load (bank-sync uses snapshots on BalanceSyncState — real consumer). _(Effort: S; independently correct, non-breaking)_
- [ ] **T3 payload fingerprint ledger + T5 persisted layout fingerprints** — LiveStore-style drift detection: per-event payload hash stamp (warn-first) and per-collection engine fingerprints checked at boot → `LayoutDiff` → existing `RebuildThreshold`/`ConfirmRebuild` gate. _(Effort: L; v5-gated where stamps touch `Record`)_
- [ ] **T6 compat policy + T7 docs/ADR split** — additive-change rules as a cqrs-lint rule; recipes + core.md conventions; proposal → implementation ADRs. _(Effort: M; follows the code)_

**Execution plan (2026-10-09 18:02, Pareto-sequenced):** [`docs/planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md`](docs/planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md) — 26 medium tasks (30–100 min) / 134 fine tasks (≤12 min), phases: 1% tag wave + bank-sync adoption → 4% T1 completion + DiscordSync/cqrs-htmx → 20% T4 + ADR + trust surface → tail T3/T5/T6/housekeeping. New rows surfaced by the plan:

- [ ] **Release wave: schema v4.6.0 + system minor, co-released; then apply the bank-sync pilot patch** — the 1%→51% (all shipped value is dead code until consumers import it; also unblocks system's GOWORK=off CI). Blocked by concurrent-session red gates; changed-set path per the 2026-10-06 lesson. _(Effort: M; plan M1–M5)_
- [ ] **T1 remainder: TypeDecoder derivation + catalog render/semver bridge + cqrs-lint undeclared-event rule** — makes the declaration THE single source (one list, four consumers). _(Effort: L; plan M6–M8)_
- [ ] **Fleet adoption wave 2: DiscordSync (211-line upcasters → RenameType/Transform) + cqrs-htmx (SchemaVersion surfaced, zero upcasters → declared Schema)**. _(Effort: M; plan M9–M10)_
- [ ] **Trust surface: chain benchmarks + godoc Examples + fuzz/rapid properties** — upcasting runs on every event load; costs and hostile-input behavior must be measured, not guessed. _(Effort: M; plan M14–M16)_
- [ ] **Housekeeping tail: kv-alias sweep, RevisionSnapshotFilter lead, proposal-fence convention doc, parallel-declaration ritual, runbook note, TODO prune**. _(Effort: S–M; plan M24–M25)_

## BDD harness adoption wave (executing 2026-10-09)

> The harness shipped + verified 2026-10-09 (27/27 tasks of the
> [04-04 plan](docs/planning/2026-10-09_04-04_SUPERB-bdd-testing-harness-pareto-plan.md);
> execution record: [14:40 status report](docs/status/2026-10-09_14-40_systemscenario-bdd-harness-full-execution-status.md)).
> The adoption wave ([14-49 plan](docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md),
> T01–T27, Full Execution Mode) is executing now — its addendum is the execution record.
> This section holds ONLY what survives the wave (residue the plan does not cover).

- [x] ~~**FINDING (2026-10-09, harness saga test): synchronous deriver on the system event bus deadlocks**~~
      **RULED 2026-10-09 (adoption-wave G1): (b) `deriver.WithAsyncDispatch` NOW + (c) journal-tailed
      deriver host as the v5 direction** — [ADR-0154](docs/adr/0154-deriver-async-dispatch-and-journal-tailed-host.md);
      evidence: [docs/evidence/2026-10-09_deriver-bus-deadlock.md](docs/evidence/2026-10-09_deriver-bus-deadlock.md)
      (live stack capture). The wave executes the fix (T08 option, T09 fixture flip, T26 loud-fail).
      **SHIPPED 2026-10-10**: `deriver.WithAsyncDispatch` + T26 loud-fail landed
      (`watermill.ErrReentrantPublish`, `event.MarkInDelivery`/`WithoutDeliveryMark` — see CHANGELOG
      [Unreleased] and the ADR-0154 addendum); the journal-tailed deriver host stays the v5 direction.
- [ ] **cqrs-htmx SQLite-lifecycle tests through the harness** (`sqliteDeployment` variant) — the wave
      migrates the memory-deployment trains; the sqlite-backed lifecycle tests remain on raw wiring.
      — source: 14:40 §f7 _(Effort: M)_
- [ ] **Harness depth assertions**: `Scenario.Event` version-hint invalidation for out-of-band appends
      (documented limitation), golden-trail payload hashes (modulo stream IDs), `AssertJournalEquivalence`
      per-stream diff diagnostics + stream-order contract note, snapshot interplay (`WithSnapshotStrategy`
      in-scenario round-trip), DLQ in-scenario (`projectionhost.WithDeadLetterStore`), upcaster in
      given-phase (`schema.UpcastSourceTransform`), idempotency scenario (same command twice → one event),
      optimistic-concurrency scenario (stale version → Conflict family). — source: 14:40 §f13/15/16/20,
      02-46 §f21-25 _(Effort: M each, incremental)_
- [ ] **`TimeAdvancesTo(t)` sugar** (Axon 4 `whenTimeAdvancesTo` analog; harness has `TimeAdvances(d)`
      only) + `system` `Bus()` delivery-mode knobs for test deployments (block-until-ack today).
      — source: 14:40 §f23/41 _(Effort: S each)_
- [ ] **Observational-equivalence port to system level + store-conformance suite reuse inside the
      harness** (both scenario/v4 concepts with no systemscenario analog yet). — source: 02-46 §f28/30
      _(Effort: M; demand-gated on fleet adoption)_
- [ ] **`scenario/v4` `testing.TB` refactor** (parity with systemscenario's TB-based constructors;
      enables benchmarks there too). — source: 14:40 §f48 _(Effort: S; ride any scenario/v4 touch)_
- [ ] **Companion commit-history hygiene** — pilot commits in cqrs-htmx/go-appkit are daemon `chore:`
      only; author proper messages if history matters before the next companion release.
      — source: 14:40 §f35 _(Effort: XS, optional)_
- [ ] **Externals (not this wave's work, tracked here so they are not lost)**: `core/v5` missing
      `.go-arch-lint.yml` + V007 marker-table drift (14 symbols) + doc-check alias-ambiguity warnings —
      core/v5 agent's in-flight scaffolding; metaengine file-size offenders (engine.go 712, reflect.go 359,
      typed_reader_scan.go 367, adttest/pagination_conformance.go 353) — metaengine agent's growth.
      All three keep `#verify` tree-red while MY modules are green. — source: 14:40 §f45-47, plan §7
      _(Effort: theirs; recheck at next verify)_
- [ ] **go-appkit GracefulClose upstream ask re-check** — their tripwire flipped 2026-10-09; verify
      whether `2026-10-06_upstream-ask-gocqrslite-gracefulclose.md` is closable. — source: 14:40 §f24 _(Effort: XS)_
- [ ] **cqrs-lint catalog counts test** — the systemscenario catalog entry may need an adoption-
      suggestions coverage bump (`TestCatalogHasExpectedCounts`). — source: 14:40 §f31 _(Effort: XS;
      fold into T23's rule work)_
- [ ] **Document `Then*` first-act baseline semantics in recipes §2.43** — the
      systemscenario `Then*` assertions compare against a FIRST-ACT snapshot
      (the state at the first `When`), a semantics consumers regularly
      misread as "latest state"; one clarifying paragraph + example in the
      §2.43 recipe (verify wording against systemscenario source before
      shipping — that module is parallel-session owned).
      — source: 2026-10-10 07-37 §f25, 09-05 §f14 _(Effort: XS)_
- [ ] **[BLOCKED] Tag `systemscenario` so `example/graph-native` builds
      standalone** — `scripts/check-example-standalone.sh --build` fails for
      the example until `systemscenario` cuts a tag including `Memory()`
      (pinned v4.0.0 predates it; workspace mode is green). Owner: the
      systemscenario parallel session — this row exists so the dependency is
      visible to them; unblock = next systemscenario tag wave.
      — source: 2026-10-10 09-05 §b3/§f5 _(Effort: theirs; XS verify after tag)_

## Durable Work Queue module (proposed 2026-09-13)

> ~~T20 PapDashboard adoption evaluation~~ and ~~M4 polish tail~~ done
> 2026-09-20 — verdict + receipts in
> [`docs/reviews/archived/2026-09-20_papdashboard-queue-adoption-evaluation.md`](docs/reviews/archived/2026-09-20_papdashboard-queue-adoption-evaluation.md)
> (ADOPT for the notify pipeline, gated on the tag wave; no PapDashboard-side
> blocker; archived 2026-09-28) and the CHANGELOG 2026-09-20 entries (pool options, deadlock
> backoff+jitter, `testutil/mysqltestcontainer`, PG engine-test DB isolation,
> PG `-race -count=2` + MySQL `-race -count=2` legs green). Items the 09-19
> harvest listed but later sessions already landed: conformance/doc.go
> 3-engine list, README MySQL quickstart, `MYSQL_TEST_DSN` in the nix legs.

- [x] ~~[BLOCKED] **Owner ratification: queue dep-validation semantics (M4 §f1)**~~ —
      **RULED 2026-10-08 (blanket authorization): Reply A** — `ErrDanglingDep`
      at-enqueue validation ratified as the contract (declare deps bottom-up;
      the validation teaches at enqueue time with a precise error). Decision memo:
      [`docs/reviews/2026-09-20_queue-dep-validation-ratification-memo.md`](docs/reviews/2026-09-20_queue-dep-validation-ratification-memo.md).
      Freezes with the queue-family tag wave (T02 Wave B).

- [x] **Queue M4 verification tail (harvested 2026-09-21)** — DONE 2026-09-29:
      ~~(a) unit-pin `deadlockBackoff`~~ DONE 2026-09-25 (sqlmock replay), ~~(b) mysqltestcontainer skip paths~~ DONE,
      ~~(d) vestigial warts~~ DONE, ~~(f) CI legs~~ LOCAL-half DONE (remote billing-gated),
      ~~(c) clock seam~~ DONE 2026-09-29 as DESIGN (owner-gated implementation):
      [`docs/planning/2026-09-29_07-20_queue-conformance-clock-seam-design.md`](docs/planning/2026-09-29_07-20_queue-conformance-clock-seam-design.md)
      — inventory of all 10 sleeps, options A/B/C, recommendation B (internal
      `nowFn` seam, `queue.WithClock` at v5); open question posed to owner.
      ~~(e) shared-DB parallel-migrate sweep~~ DONE 2026-09-29: exposure
      matrix drawn (pgtestcontainer per-test-DB suites are safe incl. under
      explicit DSN; RAW-DSN claimkit suite was the one live race), root cause
      fixed at all three PG DDL sites (advisory-lock serialization:
      `storage.PostgresInitSchema`, `pgengine` init + planned layouts,
      `claimkit` pg dialect — also protects multi-process rolling deploys),
      8-way concurrent-construction regression tests ×3 modules green against
      ephemeral PG; MySQL leg has zero parallel+DSN exposure (verified). —
      source: archived 22-01 §b/§e3-5/§f3-17

## Command-side domain depth (2026-09-13 plan)

> Prioritized execution plan:
> [`docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md`](docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md)
> — Pareto waves (W1 decider causation → W2 first-class command records → W3 lifecycle
> upcasting + docs parity → W4 gates/filing), derived from the three 2026-09-13 reviews
> (`docs/reviews/2026-09-13_*`). Additive-only v4.x; `decider` gains ZERO new deps
> (local `CausedCommand` capability interface). Guardrails: decider.go is 377/350
> baselined (new file required), api golden regen in same edit, CHANGELOG symbols gated.
>
> **EXECUTED IN FULL — W1–W3 shipped 2026-09-13; W4 closed 2026-09-20 (composed
> `#verify` S03 GREEN, 16-39 report); the 92-tag train published
> `decider/v4.7.0` + `command/v4.11.0` + commandlifecycle (2026-09-19). Plan
> archived.** The section keeps only the demand-gated remainder:

- [ ] [BLOCKED] **ADR-0138: command sourcing draft (consumer demand)** — design doc only, builds on W2's bridge, reconciles ADR-0112's planned `CommandAwareFold`. _(Effort: M)_

## Investigate: `TestSystem_ResetProjection_RestartAndReplay` contention stall (found 2026-09-13)

- [ ] **TestEngineHealth_CatchUpUnderConcurrentApplies load-sensitivity** — failed once under the full metaengine package suite ("primary ticks = 2001, want exactly 2000"); passes 5/5 isolated — genuine timing sensitivity to full-suite contention, no correctness signal observed since. Revisit only if it recurs. **Receipt 2026-09-28: still no recurrence — 15/15 green under `-race` (isolated), full metaengine package suite green ×2 this session (including once with concurrent edits in flight). Keep as observe-only.** _(Effort: S — observe)_

## Turso materialized views (ADR-0135) — upstream handoffs

> Created 2026-09-07 (matview operator option shipped; three upstream
> turso-go defects verified and documented — see
> `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`
> and the archived report
> `docs/status/archived/2026-09-07_19-25_turso-materialized-views-operator-option.md`).

- [ ] [BLOCKED] 🔥 **File the standalone upstream issue for the silent wrong-results bugs (defects A+B)** — grouped views diverge from the second transaction on and collapse at ~27k rows; draft is ready and fully verified in `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (everything below its first `---`). Blocked on user approval (external action). The COMMIT-abort half (defect C) is already reported: PR #8257 comment https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646. _(Effort: XS once approved)_
  - **2026-09-28 (M19 receipt): the zombie-tx readback follow-on is now FILED standalone — [tursodatabase/turso#9391](https://github.com/tursodatabase/turso/issues/9391)** (standalone `database/sql` repro, verified same-day on v0.7.2 `@latest` AND v0.8.0-pre.13: aborted chunk readable through the surviving connection on both; on pre.13 it also PERSISTS after fresh reopen and a commit-error write still lands; on v0.7.2 fresh reopen dies with `API misuse: unknown database open flags`). NEW findings for the A+B body when it files: wall now observed at 29k (not 27k) in the standalone shape; onset-matrix doc [`docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md`](docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md) sharpens defect A's envelope (single-tx always exact; ≤64 groups exact; loss needs 2nd tx × 10²–10³ group band). This row stays blocked for the A+B filing only.
  - **2026-09-28 SAME DAY: upstream FIX in flight — [tursodatabase/turso#9392](https://github.com/tursodatabase/turso/pull/9392)** (`roboturso`, +78/−2: `core/vdbe/execute.rs` resume-COMMIT fix + integration test `issue_9391_aborted_commit_views.rs`, "Fixes #9391", all CI green ~4h after filing). Mechanism per commit message: view-delta merge yielding on I/O left `commit_state` Ready → `op_auto_commit` restart saw autocommit on → spurious "cannot commit" with the tx left open — EXACTLY the zombie readback we characterized. Covers defect C + the zombie state (and by extension its downstream artifacts: post-abort persistence on pre.13, commit-error-but-persists). Does NOT cover defects A+B (wrong results on SUCCESSFUL commits — different path). **When the fix ships in a tagged tursogo release:** the `-tags ivmrepro` defect-C suite will fail its "did not reproduce" guard by design — flip per [`docs/turso-go-ivm-fix-flip-runbook.md`](docs/turso-go-ivm-fix-flip-runbook.md), re-verify the v0.7.2 reopen-crash (`API misuse: unknown database open flags`) is also gone, and re-check the A+B onset matrix on the fixed build before filing the owner-gated A+B issue.
- [ ] **Matview v2 feature surface** — planned-table matviews (ordered with `ApplyLayout` + backfill), filtered-view spec variants, multi-aggregate/DISTINCT serving, `DropMaterializedView` off-boarding, per-view IVM write-amp otel counter, `system.Introspection()` surface, cqrs-lint rules (matview-on-unsupported-driver; matview-plus-planned-table staleness trap), `example/materialized-views/`. Route individually when a consumer asks. — source: archived 19-25 §f23-35, 05-33 §f29-35
      _(Effort: M/L each)_
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). DESIGN STEP DONE 2026-09-21 (SUPERB S28/F110):
      one-pager at [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md)
      — `AggregateOn(fn, column, group)` as a QueryOption stamped on `QueryDecl`,
      `MatViewSpecReporter` capability, scalar-covered-first scope. REMAINING:
      ratification + implementation — routing v1: scalar-covered shapes price O(1) (matview-served), grouped shapes stay O(N) with a Doctor note (upstream defect A makes grouped routing unsafe). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A — scope the first cut to scalar-covered shapes only. Tracked jointly with M20 (b)/(c) in Metaengine follow-ups. — source: archived 19-25 §f29, 05-33 §f32, SUPERB S28/05-51 §f16-17
      _(Effort: M)_
- [ ] **Sharpen the defect-A characterization before filing upstream** — bisect the actual onset boundary (rows × groups × tx) for a principled property envelope and investigate the anomaly cluster (collapse at 26k vs draft's ~27k; wall onset through tursoengine observed at 24k-25k — the "deterministic at 27000" claim is scan-activity-sensitive, confirmed by the `-tags ivmrepro` suite logs 2026-09-11; post-abort views absorb the aborted tx's deltas). The scalar-at-scale exactness pin and the three-defect repro suite now exist (`metaengine/tursoengine/ivm_repro_test.go`); what remains is the principled onset-boundary characterization for the upstream issue. — source: 02-48 §d4/§f2/§f9/§f10
  - **2026-09-28 (M14 receipt): DONE — onset matrix measured and recorded in [`docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md`](docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md)** (15-config sweep via `metaengine/tursoengine/ivm_bisect_test.go`, env `TURSO_IVM_BISECT=1`). Headlines: single-tx loads ALWAYS exact; ≤64 groups exact at any tx size; 2000 single-member groups exact — divergence needs (≥2nd tx) × (groups in the 10²–10³ band), onset at tx#2 for ≥500-row txs, delayed to tx#4/tx#33 for 100/10-row txs; loss is PARTIAL (≈1–5% of the offending tx's delta); the draft's canonical 430.50 delta reproduces byte-for-byte at 2k/316/2×1000. Defect-C wall recorded as data (chunk=10 wall at tx#93 carries the per-tx-scan asterisk). The row stays open only for the remaining upstream-filing step.
    _(Effort: M)_

---

## Cordis spatiotemporal-composability follow-ups (2026-09-10)

> Source: [`docs/planning/archived/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](docs/planning/archived/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md) (executed in full 2026-09-10; archived by the 2026-09-11 docs-health pass)
> (mapping report: `docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`).
> Operationalizes the Cordis learnings — revertible effects, reactive coeffects,
> observational equivalence — as correctness + trust wins **without breaking a
> v4 consumer**: every behavior change warns-first in v4.x, hard-errors at v5
> (rides the ADR-0123 wave).
>
> **ALL 27 tasks (M-01..M-27) DONE 2026-09-10** — shipped surface lives in
> CHANGELOG `[Unreleased]`: Reset warn-guard + projectionadapter `Resettable`
>
> - ADR-0136, the coeffect gate (`DomainConfig.Events` /
>   `ErrDanglingEventSubscription`), cqrs-lint E018, `ValidateCoeffects` +
>   `coeffects.md`, the arXiv grounding + figure recounts, ADR-0137 engine
>   deactivation (quarantine/reroute/reprobe + Doctor/Stats health), and the
>   equivalence tooling (`scenario.Interleaved` /
>   `AssertObservationalEquivalence` + rapid property). Vocabulary stays
>   internal-only until v5 (M-26 default held, verified leak-free). The
>   2026-09-11 follow-up wave closed the rest in-tree (CHANGELOG
>   `[Unreleased]`): `EngineResetter` on every engine (sqlite 🔥 + pg, mysql,
>   duckdb, pebble, bbolt, badger, dgraph, iroh; turso by delegation), reset
>   capability surfaced in Doctor/`GetEngineStats` (`CanReset`), C040 fold-case
>   coverage with E018 provider parity, goleak for `metaengine` +
>   `projectionhost`, the `[Unreleased]`-position tripwire in `verify-docs.sh`,
>   and fold-write failover with `CatchUpEngine` (ADR-0137 completion —
>   writes reroute like reads; reprobe rebuilds before reactivating).

`metaengine/projectionadapter`/`irohengine` sibling replaces and repinned
every consumer (`pin-sweep --check --remote` green; tags verified
replace-free — 10-25 §a2/§a3, now archived).

- [ ] **Coeffect gate is blind to `RawQuery`-declared folds — product fix** —
      `system`'s coeffect validation (`DomainConfig.Events` /
      `ErrDanglingEventSubscription`, ADR context: the gate's three-tier
      contract) cannot see event types consumed by folds inside
      `system.RawQuery(query)` declarations, forcing consumers into the
      documented RawQuery workaround. Fix: `buildProjections` extracts event
      types from `rawQuerySpec` folds and feeds them into the coeffect
      universe. Deferred by the graph wave's D1 zero-API-change rule; needs
      its own green-light + `system` module tests when taken.
      — source: 2026-10-10 07-37 §f24, 09-05 §c1/§f2 _(Effort: S)_

---

## cqrs-lint

> CLI subcommand-consistency follow-ups (2026-10-03 harvest): full Pareto plan
> at [`docs/planning/2026-10-03_05-16_SUPERB-cqrs-lint-cli-consistency-pareto-plan.md`](docs/planning/2026-10-03_05-16_SUPERB-cqrs-lint-cli-consistency-pareto-plan.md)
> (M01–M14 / 58 micro-tasks). EXECUTED-THIS-SESSION (in CHANGELOG `[Unreleased]`,
> not tasks): doctor `--fix` → `--prune-suppressions` rename; single-render
> commands' documented-ignore of `--format` (both regression-pinned in
> `cmd/cqrs-lint/subcommand_consistency_test.go`). NOTE:
> `pkg/analyzer/store_spec.go` lint debt belongs to the concurrent multi-store
> session — deliberately not harvested here.

- [x] **M01 contract-enforcement test pack** — DONE 2026-10-03 (`contract_enforcement_test.go`: config-parity e2e ×2, help-drift golden via extracted `rootLongHelp`, completion/help surfaces; report §`2026-10-03_05-58`).
- [x] **M02 single-source format vocabularies** — DONE 2026-10-03 (`formats.go` slices feed validate+WithShort+explain+init; killed 2 csv/tsv doc omissions; subset test).
- [x] **M03 binary smoke probes + exit-code docs** — DONE 2026-10-03 (`scripts/check-cqrs-lint-cli.sh` 8 probes + fault-injection self-test wired into `#check-release-scripts`; README exit-code table).
- [x] **M04 flag-consumption audit matrix** — DONE 2026-10-03 (`flag_contract_test.go` acceptance 7×5 + rejection 7×14 + rules precedence; README consumption matrix).
- [x] **M05 doctor JSON schema + explain tri-state row** — DONE 2026-10-03 (advanced.md 16-field table; tri-state teaching note in explain).
- [x] **M06/M07 sibling CLI audits** — cqrs-gen DONE 2026-10-03 (real bug: positional paths dead code → `ArbitraryArgs` + `cli_contract_test.go`); doc-check CLEAN; cqrs-upgrade = stdlib-flag family (no cmdguard patterns, noted); **cqrs-bench DONE 2026-10-03 post-train**: `--format` validated up front per subcommand (run/compare/sweep/layout + soak subset — `formats.go` vocab + `validateFormat`), `run --format markdown` + `layout --format table` render for real, 8 e2e/unit tests in `formats_test.go` green.
- [x] **M08 daemon-bypasses-lint gate** — DONE 2026-10-03 (`scripts/nightly-lint.sh` + systemd units 03:30 + LINT-ROT marker; self-test wired into `#check-release-scripts`; pre-commit option rejected in memo). INSTALL PENDING (owner): `systemctl --user enable --now go-cqrs-nightly-lint.timer`.
- [x] **M09 preset e2e + precedence tests** — DONE 2026-10-03 (`TestInitPresetE2e` ×6 via real JSONCLoader; `TestFormatFlagBeatsConfigFile`).
- [ ] **M10–M11 cmdguard upstream proposals** — DRAFTS DONE 2026-10-03 (`docs/planning/2026-10-03_cmdguard-upstream-proposals-draft.md`, 4 proposals, claims verified vs v4.0.2 source). FILING REMAINS USER-GATED.
- [x] **M12–M14 polish** — DONE 2026-10-03 (M12 dedocumented: README states doctor doesn't colorize; M13 `computeChangelog` + missing-tag stderr notice + test; M14 three CONTRIBUTING doc lies fixed (`explain c008`, `disabled` key, 186-rule count) + rc-safe probe snippet + CHANGELOG entries).

> nsfw-classifier feedback harvest (2026-10-03): the fixed items live in
> CHANGELOG `[Unreleased]` + the reviews under `docs/feedback/archived/2026-10-03_*`.
> Deliberately deferred:

- [ ] **F024/F025 pushdown UTILIZATION variants** — pagination-window and manual-count detection over Query-R collections needs reader→slice dataflow to avoid FPs; adoption-time behavior unchanged for importers (feedback file 2, Fix 1 remainder).
- [ ] **Pushdown cookbook recipe** — "legacy in-memory grid → read model + pushdown + pagination" as a skill-references recipe, giving F022/F023 findings a landing doc (feedback file 2, Fix 5).
- [ ] **Analyzer load-scope widening** — non-importer packages (HTTP servers, slog setup, metrics) are invisible to feature detection; F028/F004 can fire on scope-blind absence. Doctor no longer pins absence-valued booleans (fixed), but honest detection needs either a wider load or an explicit "scope-limited" confidence tier (companion feedback, FP item 2).
- [ ] **F005 stable anchor** — alphabetically-first-package anchor moves when that package drops `WithSchemaVersion`; stale-suppression gate catches the drift loudly. Re-anchor (go.mod, A009 precedent) only if more consumers hit it.
- [ ] **B005 StrictApplyFolds disambiguation** — name-matched registry; same-named folds across packages could confuse it. Test-suite hardening item, not a rule change.

> Scorecard-ratchet harvest (2026-10-09, from
> `docs/status/2026-10-09_02-41_cqrs-lint-purpose-recalibration-session-review.md`):
> composition credit + waivers + Modernity headline are DONE (CHANGELOG
> `[Unreleased]`; verified live against `~/projects/journal`: 2/28 → 5/29,
> false-MISSING persistence rows credited, `Modernity: Modern`). Deliberately
> open:

- [x] **Scorecard waiver e2e in the binary probe set** — DONE 2026-10-09 (`scripts/check-cqrs-lint-cli.sh`: `probe_scorecard_waiver` config→WAIVED→exit 0, `probe_scorecard_bogus_waiver` nonzero, `probe_scorecard_path_preset` relevance delta via `--format json`; fault-injected `--self-test` stubs with positive control).
- [x] **Waiver trigger expiry surfacing** — DONE 2026-10-09 (`flagFiredWaiverTriggers`: server/async-bus/transport signals, word-exact matching, `TRIGGER LIKELY FIRED` suffix + re-litigation recommendation; advisory wording by design).
- [x] **Scorecard preset resolution is cwd-based** — DONE 2026-10-09 (`resolveScorecardPreset` in both the subcommand and root `--scorecard` path: `<path>/.cqrs-lint.json` preset wins, cwd fills; unknown presets rejected via `LoadProjectConfig`; `TestResolveScorecardPreset` ×3).
- [x] **Legacy-persistence signal in modernity** — DONE 2026-10-09 (`ScorecardDeprecated.StackPresetUses` from `FeatureProfile.StackPresets` detection; any stack surface ⇒ Legacy; counts project-wide across per-module profiles — live-verified on cqrs-htmx: Legacy with stack+removed-API dual signal).
- [x] **Doctor composition cross-render** — DONE 2026-10-09 (`renderDoctorFeatureProfile` prints the composition hint when `HasSystemComposition`; presence+absence pinned in `doctor_render_test.go`).
- [ ] **Journal-side adoption (owner-gated)** — once this cqrs-lint version is tagged: run the new scorecard in `~/projects/journal`, replace the AGENTS.md "do NOT chase the grade" moat with per-row waivers (reason + revisit triggers incl. F007's "daemon lands / multi-writer / first schema break"), and route the C033 false-positive class per its ROADMAP entry. (2026-10-09 note: journal probes clean at 5/30, Modernity Modern; denominator grew 29→30 via the foreign `systemscenario` catalog entry.)

> Point-in-time execution plan (T01–T24 / F001–F096) with per-row resolution
> markers: `docs/planning/archived/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`.
> T01–T12, T20–T24, F089, F090(a+b), F091 Tiers 1–3 (incl. P014 ApplyLayout)
> and the 2026-09-08 hardening batch are DONE (CHANGELOG `[Unreleased]`); this
> section carries the living remainder.

- [ ] **cqrs-lint audit follow-ups: loose heuristic gates (deliberately
      deferred 2026-09-11).** Documented, low-severity FP/FN vectors that
      each need their own false-positive analysis + golden churn; confidence
      levels already mitigate. Candidates: import-scope substring gates
      (V001 `/v3` `/v4`, V004/V005 `eventtest`, T001 `/decider`, T002/T005
      `/projection`, T003/T004 `catalog`/`snaps`, T007 `/event`, E016
      `Bundle`, A008 `/event/` exclusion); B018 `containsBus` lowercase-only
      and its "identical error-handling structure" claim; A015 name-collision
      write-matching at error severity; A016/A013 project-wide suppressions;
      A017 unqualified `NewRepository` matching + `NewTypedRepository`
      asymmetry; A019 vendor-path heuristic; F006 payload-class wiring under
      the strong/weak split; F009/F010 pattern tokens; V002/V003/V006
      root-go.mod-only scope; b022_b025.go (495) and
      a020_a021_a022_a023.go (~357) over the 350-line convention — bundle
      with the file-size-gate policy decision.
- [ ] [RULED 2026-10-08] **Doctor-JSON pre-merge semantics ruling** —
      **RULED: EFFECTIVE post-override values** (matches the text path; honest
      for scripting), with a `--raw` escape flag for consumers pinning today's
      shape. Golden re-pin + CHANGELOG "Changed" entry ship with the
      implementation (XS, rides the cqrs-lint typed-info wave T02 Wave A).
      — source: 05-31 §g2
- [x] ~~[BLOCKED] **Release-policy Q3: severity tightening in a minor.**~~ —
      **RULED 2026-10-08: acceptable in a minor** when documented prominently
      in the CHANGELOG "Changed" section (the S011 precedent already shipped
      that way; S008/S009 consumers get the same contract). The envelope v2
      wire-format-in-minor question inherits this ruling. — source: 02-40 §g3
- [x] ~~[BLOCKED] **Daemon Q2: `.golangci.yml` exclusion from the auto-commit
      formatter.**~~ — **RULED 2026-10-08: accept self-heal permanently**
      (`scripts/check-formatters.sh` repaired every occurrence, 4+ incidents);
      the upstream BuildFlow fix rides the T25 filings pack. ROOT-CAUSED
      2026-09-06: BuildFlow's built-in golangci defaults regenerate config at
      pre-commit; no user-facing knob found in `~/.config/buildflow`.
      — source: 02-40 §d1/§g2
- [ ] [RULED 2026-10-08] **F040 — required status checks / branch protection.**
      **RULED: enable once the Actions billing fix lands** (protection with
      never-running required checks would wedge every push incl. the daemon's):
      require `verify-fast` + the per-module isolation matrix; exceptions =
      admin-merge path for the auto-commit daemon's chore blobs. Master
      has no branch protection today. — source: 06-58 §g1
- [ ] [RULED 2026-10-08] 🔥 **350-line policy: RATIFIED — ratchet is POLICY**
      (blanket authorization), plus **harness-dir exemptions** (adttest/
      enginetest are exported test harnesses — exempted from the convention,
      not the gate), and **split waves ride file-touch moments** (no dedicated
      multi-session split programs; split a baselined file when you are
      already editing it, shrink-only ratchet does the rest). IMMEDIATE
      OBLIGATION: the three catalog NEW offenders are gate-RED until split —
      `catalog/docserver/docserver.go` (351),
      `catalog/eventcatalog/frontmatter_convert.go` (364),
      `catalog/cmd/ec-fixture/main.go` (356) — split them in T26 (W6).
      STATE: the baseline+ratchet gate SHIPPED and is GREEN
      (`scripts/check-file-size.sh` + `scripts/file-size-baseline.txt`, 58
      historical offenders baselined; fails on NEW offenders and on
      baselined-file GROWTH, allows shrinking; mutation-proven ×2; wired
      into `nix run .#check-file-size` + the CI `file-size-gate` job).
      — source: 06-56 §a9/§d1, 05-51 §a (ratchet shipped)
- [ ] [SURFACED 2026-10-09] 🔥 **File-size gate RED again — 10 sibling-grown
      offenders (W1 parallel sessions, none mine).** `nix run .#check-file-size`
      fails on: cqrs-lint `pkg/rules/consistency/d005_version.go` (406 NEW),
      `pkg/rules/resilience/helpers.go` (507 NEW), `pkg/suppression/stale.go`
      (489→533), `doctor.go` (400→433); system `system.go` (358 NEW),
      `constructor.go` (357→413); metaengine `engine.go` (700→712),
      `reflect.go` (352→359), `typed_reader_scan.go` (367 NEW),
      `adttest/pagination_conformance.go` (353 NEW — likely a baseline
      candidate under the harness-dir exemption ruling, owners' call).
      Owners: split at your next file-touch moment (shrink-only ratchet does
      the rest) or pin the baseline on a structural shift. NOT part of
      `#verify` — does not block the T03 verify-green receipt.
      — source: 00-32 W1 status §b/§f

---

## Release / Tagging

> The full 39-tag v4 wave (B1–B7) was cut, pushed, and verify-ci-green on
> 2026-08-29. The SUPERB wave tagged `otel/v4.4.0` + `cmd/cqrs-upgrade/v4.0.0`
> (2026-09-07) and a coordinated release re-tagged 15 modules (2026-09-08).
> Zero local `=> ../` replaces remain EXCEPT `storage/go.mod` (`=> ../encryption`,
> `=> ../snapshot` — the documented unpublished-sibling pattern).

- [x] ~~**Release-train tail (post-v4.9.0 waves, queued in [Unreleased])**~~ —
      **PUBLISHED 2026-10-08 (42-tag stalled-waves train, receipts below):**
      per-module `GOWORK=off go test -short` ran GREEN over all 49 candidate
      modules FIRST (the 2026-10-06 verify≠tests lesson); then
      `batch-release.sh` cut metaengine v4.17.0 (solo, dependency-first) →
      push → proxy smoke ✓ → `pin-sweep.sh` (36 modules bumped to latest) →
      38-tag batch (system v4.11.0, cmd/cqrs-lint v4.15.0, signing v4.4.0,
      storage v4.10.5, queue family v4.0.3, scheduling v4.6.2 + sqlstore
      v4.1.4, watermill v4.6.5, id v4.7.2, benchkit v4.7.0, engines, tails)
      → push (one tag initially missed by a bad push-list parse — caught and
      pushed by the manifest `--check --remote` gate: cqrs-lint v4.15.0) →
      dependents wave (cmd/cqrs-bench v4.3.4, systemtest v4.0.0 FIRST TAG,
      testutil/mysqltestcontainer v4.0.0 FIRST TAG) → smoke ✓ ×3 →
      taskmanager/systemtest/cqrs-bench sibling replaces dropped (0 remaining)
      → `check-example-standalone --build` 0 findings → versions.json 110
      trains + README matrix fresh vs origin → CHANGELOG cut into the dated
      wave section + `[Unreleased]` restored to first position (verify-docs
      green, `TestTagContentMatchesChangelog` green, changelog-symbols 55
      citations green). Run logs: `build/release-logs/batch-20261008-*.log`.
      Stale items found already-tagged during curation (scheduling/engine
      ErrEngineNotDueClaimer ∈ v4.0.2, cqrs-upgrade --strict ∈ v4.1.2,
      catalog StaticServer ∈ v4.7.1) — their wave rows were dead. — source:
      closeout §f17-24
- [ ] [RULED 2026-10-08] **claiming V006 advisory decision** — **RULED:
      linter-semantics fix** (teach V006 to skip pins at a module's newest
      existing tag — content-identical re-tags are history lies). XS cqrs-lint
      change + golden, rides Wave A (T02). Original: claiming has no
      content since v4.0.0, so examples' V006 "same release" advisory is
      structural. — source: closeout §f10/§g2
- [x] ~~**Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms~~ — **RULED 2026-10-08: KEEP 150ms** (worst-of-30 sample
      rationale stands; shipped + gated green).
- [x] ~~[BLOCKED] **`benchkit/LICENSE` says "Unknown Author"**~~ — **FIXED
      2026-10-08 (receipt):** template artifact corrected to
      "Copyright (c) 2026 Lars Artmann. All rights reserved." — verbatim the
      repo-root LICENSE form (the canonical legal file); not a licensing
      decision, an artifact correction. — source: 2026-09-28 publish-integrity
      session M1 §a + session-2 §g

- [ ] **[BLOCKED] Root README version-manifest row for `example/graph-native`**
      — the versions manifest (`scripts/check-versions-manifest.sh`) rows only
      tagged trains; graph-native rides ADR-0152's untagged-examples ruling.
      Add the row when/if a graph-native tag is ever cut (or record the
      path-vs-tag exclusion there). — source: 2026-10-10 07-37 §f27
      _(Effort: XS, blocked on tagging decision)_

---

## Metaengine — follow-ups

- [x] **Feedback #6: system test-mass gap** — DONE 2026-09-29 (M17):
      (a) config-loader table tests + 3 rapid properties — which FOUND and
      FIXED two live `system.LoadConfig` bugs (documented
      `CQRS_INSTANCES__<i>__<field>` env overrides were silently dropped AND
      corrupted the YAML instances list); (b) real-sqlite lifecycle/shutdown
      stress in `systemtest` (6 rounds × 8 concurrent creates, Count
      double-apply sentinel, racing GracefulClose/Close); (c) wiring
      determinism (two identical constructs → byte-identical
      `system.Explain`). The 2026-09-21 projectionhost double-apply find is
      evidence for its priority. — source: 23-24 followups §f19-21, feedback doc
      §4.6
- [ ] **Post-v4.9.0 metaengine tag wave** — publish the [Unreleased]
      metaengine surface: G-T13 ADTSet parity (pg/mysql, mysql VM leg still
      pending a quiet window), G-T12 `BackfillPlannedTables`,
      `ScanScoredVector`/`RowScanner`, adttest helpers (`AssertTxIsolationFromForeignContext`).
      **Receipt 2026-09-28 (M1, SUPERSEDED same day):** the wave STALLED at 1/7
      (only `dispatcher/v4.5.0` existed at receipt time; no logs kept by
      batch-release.sh; cause of the stall unknown).
      **Receipt 2026-09-28 (later session):** WAVE COMPLETED — all 7 tags exist
      local+origin and the 6 remaining resolve on the module proxy (`go list -m`
      green ×6; proxy Time 2026-09-28T05:04:20Z). Post-wave hygiene executed
      same day: full `pin-sweep.sh` (7 consumers bumped to the new tags,
      cqrs-lint goldens refreshed), `system/integration`'s dead-`storage/v4.10.0`
      sibling replace stripped (its documented obsolescence condition — published
      system/v4.10.0 carries `storage/v4 v4.10.1` — is met),
      `check-example-standalone.sh --build` green at 0 findings (taskmanager's
      dead `projectionhost→storage/v4.10.0` edge healed via MVS through
      system/v4.10.0), `pin-sweep --check` fully green (sibling + external; the
      `go-finding` family bump for cqrs-lint cleared the external leg — 19/19
      packages green). **Receipt 2026-10-08 (T02): CHANGELOG wave-section cut
      DONE — metaengine v4.17.0 + engine patch wave published (42-tag train;
      dated wave section in CHANGELOG; run logs `build/release-logs/batch-20261008-*.log`).
      REMAINING: the G-T13 mysql-VM quiet-window leg only (tracked in T04).**
      — source: closeout §f17/§c2 _(Effort: M — tag-wave mechanics)_
- [ ] **Calibration provenance protocol + quiet-window re-runs** — protocol HALF DONE 2026-09-11 (later session), re-runs remain gated on a quiet window: (a) DONE — `scripts/calibration-gate.sh` asserts 1-min load < 5 (overridable `--max-load`/`CALIB_MAX_LOAD`; CI exempt) and aborts loudly — verified against a live compile storm (load 207 → hard abort); `calibration-drift.sh` runs it before benching; (b) DONE — protocol items 6-8 in `docs/benchmarks/calibration-2026-08-30.md` define the per-entry PROVENANCE line (store path + binary version output + uptime samples) and ban secondhand version citations; the 2026-09-11 SearchQuery entry now carries an explicit provenance-gap note; (c) MECHANISM DONE, RUN PARTIAL — `benchmark-regression.sh --save` writes a titled provenance header (fixture-tested, parser-safe); the titled re-pin of `benchmarks/benchmark-baseline.txt` **DID run 2026-09-20 17:12 UTC** (receipt: the T18b canonical record `docs/benchmarks/2026-09-20-21_t18b-record.md` — noise-clean save, go1.27.1 provenance, claimkit/SQLite entries, 0 regressions vs the 2026-09-11 baseline); the quiet-window count=5 SearchQuery re-run remains pending (a 493-load storm held the 2026-09-11 session; gate correctly refuses); (d) PENDING — re-anchor ALL dgraph constants in one gate-passing window. Run when `scripts/calibration-gate.sh` passes: SearchQuery count=5 (supersede today's table if medians move >5%), then the benchmark-baseline re-pin, then the dgraph constant campaign. — source: 03-50 §b2/§b3/§f7/§f8/§f15/§f16, 02-48 §d3/§f8
      _(Effort: M)_

- [ ] [RULED 2026-10-08] **M20 design-ratification follow-ups — ALL RATIFIED
      (blanket authorization); implementation slots = plan task T11 (W2,
      BEFORE the v5 branch freezes engine construction surfaces)** —
      (a) **ADR-0150 ACCEPTED**: `EngineConfig.SingleWriter` advisory lease
      (`<dsn>.cqrs-lease` flock, fail-loud default-off, one shared helper;
      written to
      [`docs/adr/0150-engineconfig-singlewriter-advisory-lease.md`](docs/adr/0150-engineconfig-singlewriter-advisory-lease.md)).
      (b) **AggregateOn first cut APPROVED**: `AggregateSpec` QueryOption on
      `QueryDecl` + construction-time validation + `MatViewSpecReporter`
      capability + planner O(1) pricing for scalar-covered shapes (one-pager:
      [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md)).
      (c) **Routing integration v1 APPROVED** after (b): scalar-covered shapes
      price O(1) and route to the matview engine; Doctor INFO for uncovered
      shapes (tracked jointly with the Turso-section routing row).
      (d) ~~Scan-default v5 survey~~ RULED 2026-09-21 (Option C) — tracked in
      the Goal-closure G-T14 row.
      — source: archived 15-34 §a1-3/§f28-32

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

- [ ] [RULED 2026-10-08] **Turso strict-vs-lenient DSN param policy** —
      **RULED: STRICT** ("make impossible states unrepresentable"; blanket
      authorization): reject unknown `*encrypt*`/`*key*` params at tursoengine
      construction with a typed Rejection-family error naming the param and
      the close matches. Existing valid DSNs are unaffected (only MISTYPED
      params change behavior — from silently-unencrypted to fail-loud).
      Implementation + tests ride W2 (T08 tail). Original: the driver
      silently ignores mistyped encryption params (`encryption_hexkkey=`
      opens the DB UNENCRYPTED). — source: 20-57 §g3
- [x] ~~[BLOCKED] **Turso sync/embedded-replica first-class support decision**~~ —
      **RULED 2026-10-08: OUT OF SCOPE for v5** (no consumer demand on record;
      the L-effort design stays demand-gated — reopen when a real Cloud BYOK
      consumer asks; recorded in ROADMAP on-demand). Original: the ONLY Go
      path to Cloud BYOK (the `database/sql` driver has no remote client).
      — source: 20-18 §g1, 20-57 §f11-12
- [ ] [RULED 2026-10-08] **Upstream turso-go issues** — **FILING APPROVED**
      (blanket authorization) via the T25 filings pack, each claim re-verified
      against latest main first (verify-before-filing):
      (a) missing `DriverContext`/`OpenConnector` (struct-level config without
      DSN stringification); (b) pure-remote connections cannot present a BYOK
      key; (c) mistyped DSN params silently ignored → silently-unencrypted
      DBs (in-repo strict-posture fix rides W2 regardless). — source: 20-18 §c4,
      20-57 §f13-16
- [ ] [RULED 2026-10-08] **dgraph one-RPC scope (Q1)** — **RULED:
      INCREMENTAL** (per-ADT reassessment benches before each individual flip;
      no one-wave authorization — O(1) claims must be earned per shape). The
      per-ADT benches ride the T05 calibration campaign (dgraph constants
      re-anchor there anyway). ADTMap is already O1 (2026-09-07);
      Set/Multimap/Log/StreamLog still OLogN. — source:
      archived 22-33 §g1, 04-35 §f9/§f15
- [ ] [RULED 2026-10-08] **CapabilityGaps reach into Doctor (Q2)** —
      **RULED: YES** — documented gaps must silence Doctor's `--- Capability ---`
      violation lines too (consistency: a documented gap suppressing PLAN
      diagnostics but not Doctor noise is a split brain). XS impl: thread the
      gaps into `CapabilityAudit` (receives nil today); rides W2.
      Original: documented gaps silence PLAN diagnostics today.
      — source: archived 22-33 §g2, 04-35 §f17
- [ ] **Conformance-sweep + hot-path tail — live-server runs remain** — (a) dedup
      no-op case shipped (`TestApplyIdempotent_DuplicateIsNoOp`); (b) micro-bench
      DONE 2026-09-16 (struct hot path 1.9 ns / 0 allocs through the shared
      funnel; `metaengine/encoded_bench_test.go` +
      `docs/benchmarks/2026-09-15_applyfold-raw-payload-funnel.md`); (c) PG half
      DONE 2026-09-16: `PG_MODULES="scheduling/sqlstore storage" nix run
      .#integration-pg` full suite PASS incl. `TestClaimingPostgres_MetricsSnapshot`
      (+ TwoClaimersNoDoubleFire, RenewLease, RenewVsClaimRace) on the repo's own
      ephemeral PG; the `#integration-mysql-nspawn` half remains (quiet-window).
      — source: 05-38 §b2; execution status 2026-09-16 08-04 report §a6
      _(Effort: M)_
- [ ] **scheduling/sqlstore hardening tail — `RenewLease` ownership/claim
      tokens remain** (race-stress test, counter-scope pin, counter property
      test, `FuzzDecodeDueTimer`, and the `example/scheduler-otel-status`
      worked `Metrics()` + OTel example all shipped 2026-09-13..16);
      `RenewLease` ownership/claim tokens are design-gated (code comment
      defers today).
      — source: 03-50 §f18-27; execution status 2026-09-16 08-04 report §a3-a4
      _(Effort: M, one rule per slice)_

---

## CI / Infrastructure

`scripts/wait-for-quiet.sh` (1-min AND 5-min ceilings, self-tested),
`scripts/can-run-composed-gate.sh` (no-release-procs + tree-stability +
load assert), and the `#verify` `-p` parallelism cap (`VERIFY_TEST_P`, set
to 4) all shipped in T26 (09-40 §a4, now archived); composed-`#verify`
went GREEN the same day (S03). Remaining launcher ergonomics live in the
release-train tail row below. — source: archived 06-47 §f6-8, 12-02 §f9/11/17, 18-11 §f11

- [x] ~~[BLOCKED] **Push-cadence ruling (owner)**~~ — **RULED 2026-10-08:
      PHASE-BOUNDARY** — push after each execution-plan wave completes green
      (reviewability + remote CI evidence per wave; no batch hoarding, no
      per-commit pushes). The 2026-10-08 v5-GOAL plan's waves each end with a
      push. Original blocker cleared 2026-09-22 (0 unpushed); remote CI now
      gates on the billing fix row below. — source: delta §f1,
      18-19 §f38, closeout §f2/§g1
- [ ] [BLOCKED:owner] **F153: pkg.go.dev license — RESOLVED AS DESIGNED 2026-09-25:** the repo-root LICENSE is deliberately PROPRIETARY ("All rights reserved"), and pkg.go.dev hides docs for non-OSS licenses BY DESIGN — propagation was never the issue. Remaining decision is the owner's: relicense OSS (unblocks pkg.go.dev docs) or accept hidden docs (godoc remains local). Verified empirically against pkg.go.dev 2026-09-25. ~~hides all
      module docs (license-redistribution gate) for system/v4@v4.9.0 — and
      possibly every module: no LICENSE file at module subdirectory roots?
      Verify whether the repo-root LICENSE propagates to submodules on
      pkg.go.dev; if not, decide per-module LICENSE files or accept hidden
      docs. Consumer-trust blocker for the public surface. — source: T03
      post-wave verification 2026-09-22 _(Effort: S verify, M if per-module
      LICENSE files needed)_
- [ ] **Watch the first real CI runs (push-gated)** — `Examples Test` job
      (nix eval, 10m timeout, DB-skip env), the md-go-validator ci.yml leg
      (cold build ~1-2 min), the nightly `Go version contract` step, and the
      README push-leg timeout. — source: followups §f2, 18-19 §f3, W0 §f28/§f31
      _(Effort: S, observe)_
- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s; broken since ~2026-07-17. Local `nix run .#verify` remains the
      authoritative gate. _(Effort: S, user action)_
- [ ] **cqrs-lint Self-Lint credentials** — BLOCK LIKELY STALE (re-verified
      2026-09-17): go-finding resolves via the public module proxy under
      GOWORK=off (verified again: proxy serves v1.11.0, published and fetched
      by cqrs-lint's go.mod bump today — no replace directive, no SSH/git
      remote needed; the old `git ls-remote` exit 128 hit the SSH path). The
      remaining blocker is purely the Actions billing entry above. Re-run the
      self-lint CI leg once billing works; close if green. _(Effort: S,
      re-run required, gated on billing)_
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_

- [ ] **Pre-commit hook still owns three TREE-WIDE gates** (workspace
      `go build ./...`, api-surface freshness, fmt.Printf grep) that can
      block an honest commit on a CONCURRENT session's in-flight files —
      the benign version observed 2026-09-18 (go.mod/go.work 1.27 stamp
      war), the hostile version is a sibling's mid-edit file. All three
      verified green 2026-09-18 ~18:50 (build OK, 7209 exports, zero
      Printf hits). Design staged-scoped or per-module variants where
      cheap; the workspace-build gate is the one worth keeping tree-wide
      (it is the daemon-commit safety net). — source: 2026-09-18 18-12
      report §e/9 _(Effort: M)_

- [x] ~~**asyncapi-react bundle requires `unsafe-eval` — interactive AsyncAPI
      UI is DEAD under strict CSP**~~ — RESOLVED 2026-09-18 via the page-scoped
      option: `serveAsyncAPIHTML` applies `applyCSPAllowingEval`, adding
      `'unsafe-eval'` to `script-src` for `/docs/asyncapi` ONLY (never global;
      verified scoped to that one handler). The vendored bundle's ajv compiler
      needs `new Function`, which the eval-free policy blocked. The browser
      gate no longer classifies eval refusal as a known degradation —
      `TestCSPBrowser_NoViolations` requires the `aui-root` DOM to render and
      fails on ANY fatal CSP refusal; `#check-csp` re-verified green 2026-10-04.
      Nav scripts are nonce-gated (`docsNavProps` threads the nonce into
      `ThemeToggle` + `SimpleNav` `BaseProps`). A future eval-free bundle swap
      would let the relaxation be dropped entirely (nice-to-have, not tracked).

- [ ] **AsyncAPI exporter: response schemas for query replies.** The 2026-10-04
      request/reply implementation emits an opaque reply message (the catalog
      does not model query response types) — consumers get reply addressing and
      channel topology but no response schema. Requires a `catalog.Message`
      response-schema field (`ResponseSchema *Schema` or generics-backed
      `SchemaFromResponse[R]`), which is an API surface addition — pair with
      the openapi exporter's response side for one design pass. — source:
      2026-10-04 asyncapi request/reply work _(Effort: M)_

- [ ] **AsyncAPI exporter: protocol bindings + security schemes.** Watermill
      backends (kafka/nats/amqp/redis) are only reflected as a bare
      `WithServer(name, host, protocol)` server object today; AsyncAPI 3.0
      channel/message `bindings` (topic names, delivery guarantees) and
      `components.securitySchemes` (catalog.Message.Security is parsed by the
      openapi exporter but ignored by asyncapi) would make the document honest
      about HOW messages move. Keep bindings dep-free (string-typed options),
      no watermill import. — source: 2026-10-04 asyncapi review _(Effort: M-L)_

- [ ] 🔥 **CI triage: master red across ~15+ jobs, no green run in the last
      30.** Classified 2026-09-11 (run 34548534824), RE-CLASSIFIED
      2026-09-13 (run 34747274058, full log triage): (a) FIXED same-day —
      the Module-matrix go.sum class (missing `/go.mod` hashes after the
      v4.5/v4.6 pin wave). (b) **ROOT-CAUSED 2026-09-13 — one dominant
      infra cause:** the deprecated `magic-nix-cache-action` was THROTTLED
      by the GitHub Actions Cache API ("ResourceExhausted: rate limit
      exceeded" / "GitHub Actions Cache throttled Magic Nix Cache"), its
      local substituter (127.0.0.1:37515) then returned HTTP 418, nix
      disabled the substituter mid-job, and every nix-based job starved on
      closure downloads: verify-fast, Dgraph Integration, CGo Build,
      Security Scan (gosec via nix-shell), Minimum Coverage. NOT FlakeHub
      auth — the Twirp rate-limit is the GH cache backend. **Fix needs an
      owner/infra decision**: migrate to a maintained cache backend
      (`DeterminateSystems/flakehub-cache-action` — needs the parked
      FlakeHub account decision) or drop the action and accept cold builds
      (timeout-minutes must rise). (c) FIXED LOCALLY 2026-09-13, next run
      should clear — File Size Check (store.go 953→940: EventInput moved
      out), Shell Format Drift + Nix Flake Check formatting leg
      (nix fmt clean), api-stability (golden updated), cmd/cqrs-lint module
      (taskmanager golden re-pinned — rule-output drift) + verify-fast's
      TestTagContentMatchesChangelog (green against current CHANGELOG).
      (d) FIXED LOCALLY 2026-09-16: the go.work sync check job now uses plain
      `actions/setup-go@v5` with `go-version-file: go.mod` (it previously had
      NO Go toolchain at all, and rode the throttled nix cache); the
      benchmarks.yml matview-gate is restructured into per-backend subshells
      with root-relative `tee current.txt` (the old single-`cd` form hopped
      `../metaengine/tursoengine` from `stack/bench` — a nonexistent dir — and
      teed into `stack/current.txt` while the compare step reads the root
      copy; BOTH legs verified live with real bench runs). **CACHE MIGRATION
      EXECUTED LOCALLY 2026-09-18:** the owner/infra decision resolved to
      option (b) — `magic-nix-cache-action` REMOVED from all 24 ci.yml jobs
      (flakehub-cache-action was rejected: it needs the parked FlakeHub
      account + auth token) with the policy documented in a ci.yml header
      (start with nightly-gates.yml when the FlakeHub decision lands), and
      timeout-minutes raised on the starved jobs (verify-fast 45, per-module
      matrix 25, CGo 25, Dgraph 30). Remote confirmation still gated on the
      billing fix. The test-tag-release.sh
      SC2086 item was already stale — the script uses the array form
      `git "${notag[@]}"` and shellcheck is clean (verified 2026-09-13). — source: run
      34548534824, run 34747274058, `gh run list`
      _(Effort: M-L, multi-session; the cache-backend migration is the
      single highest-leverage repair)_
- [ ] [BLOCKED] **Set the `ERRAUDIT_PAT` secret** (user action) — erraudit
      findings verified zero across all modules 2026-09-15; the `error-audit`
      CI job arms the moment the secret exists. — source: 08-05 §f12/§f19
      _(Effort: S, user)_
- [ ] **Green MySQL-VM shuffled suite in the quiet window** — two attempts
      died ~15s in with transport-level resets on DIFFERENT modules/seeds
      (zero assertion failures; the documented semi-dead-VM/host-contention
      class). Replay both logged seeds (`build/shuffle-seeds.log`) when the
      box is quiet; closes the dgraph `-shuffle=on` rollout item's caveat fully.
      STATE 2026-10-09 22:05: seed list lives IN-TREE — replay from
      `build/shuffle-seeds.log` directly (`grep 'label=mysql'` for the 7
      mysql-manual seeds; the lost /tmp extraction of 21 was derived data).
      — source: 08-05 §b1/§f5 _(Effort: M)_
- [ ] **Run the real `#integration-mysql-vm` leg through the hardened
      `vm-mysql.sh` in a quiet window (load1<5)**, then update the F52 AGENTS
      integration rows + strike with evidence. The hardening is self-tested;
      the live proof run never happened (load 8–72 all day 2026-09-21).
      Also consider the same stale-port pre-flight for `vm-mysql-nspawn.sh`
      (cheap insurance). — source: archived 14-12 §b1/§f2/§f25, 15-34 §b1 _(Effort: M, quiet-window)_
- [ ] **Composed `#verify` re-record (W1 sibling)** — the S03 green
      (2026-09-20 15:04) now predates the guard chain (hash-golden,
      wait-loop, load guard, go-version gate, md-go-validator insertions),
      the v4.9.0 4-tag wave + example pin bumps, the go.work 1.27.1
      restoration, and the benchkit/cqrs-bench polish wave; one clean composed
      run re-proves the chain end-to-end (also closes the `#verify-fast`
      execution-unverified insertions). Recipe:
      `bash scripts/preflight-composed.sh && nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify`.
      STATE 2026-09-28: the E15 pin-gap blocker is dead (`dispatcher/v4.5.0`
      published 09-27) and the SC1091 shellcheck wall is fixed (09-22); the
      dedup-campaign row in Code Quality is the new pre-run obligation.
      STATE 2026-09-29: pre-run obligations are now GREEN (`#lint` 88/88 on
      the post-config-war-repair tree; config tripwires mutation-verified via
      `#check-lint-config` inside `#verify-fast`) — only the quiet window is
      missing: load1 hit 163 at 2026-09-28 22:00 (shared-host storm; still
      ~10 at 00:30). Never force under storm.
      STATE 2026-10-09 22:00: T03/f045 owns this row; BLOCKED on the sibling
      BDD-harness wave (docs/planning/2026-10-09_14-49) — systemscenario is
      GOWORK=off compile-red (uses the untagged `system.Clock`/`WithClock`
      seam while its go.mod pins `system/v4 v4.11.0`); their plan owns tagging
      `system v4.12.0` + `systemscenario v4.0.0`. Verify chain v4 armed
      detached (`~/.local/state/crush-t03/verify-chain.sh`): compile probe →
      fixture tidy probe → load gate → preflight 9/9 → `#verify`. Their
      15:49–18:40 tag wave re-rotted cqrs-lint fixtures (gjson bump) — tidied
      + converged 21:53.
      STATE 2026-10-10 13:00: the ~33-module lint-red contradiction from the
      10-09 session is RESOLVED — the ambient-env findings were REAL, with
      TWO stacked causes. (1) CONFIG REGRESSION, now fixed: the settings
      rewrite that added the depguard allow-list (daemon commits 10-08..10-10)
      silently DROPPED `cyclop.max-complexity: 25` (default 10 fired on every
      complexity-11..25 function) and the whole `errcheck.exclude-functions`
      block; both restored verbatim from the 09-29 tree with a comment in
      `.golangci.yml`. `#check-lint-config` is BLIND to settings loss (its
      canaries cover exhaustruct + depguard only) — extend it to pin the
      settings key-set if this class recurs. (2) GENUINE new-code lint debt
      from the 10-03..10-10 waves: modules shipping code without a full-lint
      pass (systemscenario wrapcheck/err113/thelper/varnamelen; systemtest
      gocognit; watermill; core/v5; pebble; catalog; …) — the module list
      under the restored config is in the 10-10 session log; per-module
      triage still owed by the owning sessions.
      — source: archived 16-37 §f10, 15-34 §f20, 15-57 §b1; 04-04 §f1
      _(Effort: M, quiet-window)_
- [ ] **Mirror-lag burn-down: fork `eventtest` + the delivery-mark trio into
      core/v5/event** — the lockstep register's only pending-mirror rows
      (`v4/eventtest/...` subtree, 80 symbols + `MarkInDelivery`/
      `WithoutDeliveryMark`/`ContextInDelivery`); every other pair is already
      shape-identical. Watch `TestMirrorLockstep`'s lag count drop to 0 and
      prune the register rows (stale-row check enforces it). Long-term: add
      signature-level lockstep (compile-time cross-import, the `event.Type`
      pattern) once core/v5 stabilizes — the golden gate is kind+name only.
      — source: 13-50-10-10 report §f3/f9 _(Effort: M)_
- [ ] **`#check-lint-config` blindness to settings loss** — the whole-file
      hash tripwire BLESSED the 2026-10-08..10 cyclop/errcheck drop (its own
      `--update` instructions re-pinned the regressed config). Extend the gate
      with settings-canaries: pin the settings KEY-SET and assert
      `cyclop.max-complexity` + the `errcheck.exclude-functions` presence, so
      an accidental drop fails even after a re-pin.
      — source: 13-50-10-10 report §f5 _(Effort: S)_
- [ ] **dupe-signal: `--keep-v5` flag + one-time suppressed-group audit by the
      v5 owner** — the wrapper suppresses ALL intra-core/v5 groups by default;
      twins like `MetadataCarrier` (core/v5/command vs core/v5/query) may be
      real consolidation signal for the v5 core, not noise. Policy decision
      pending (see report §g1).
      — source: 13-50-10-10 report §f7 _(Effort: S + decision)_
- [ ] **Root-cause the `nix fmt` gofumpt exit-2** ("failed to finalise
      formatting" persists while the tree comes up clean; hypothesis = sibling
      mid-write file + fail-on-change racing the daemon — unproven). CI's fmt
      gate (`nix fmt --fail-on-change`) depends on this being benign.
      — source: 13-50-10-10 report §f4 _(Effort: S)_
- [ ] **Verify the nightly weekly load-sweep leg fires** (Sundays-only, first
      real run) and logs cleanly. — source: 14-52 addendum 2, 16-37 §f31 _(Effort: XS, observe)_

- [ ] **Pre-existing red needs ONE owning decision (triage row)** — noticed
      2026-10-09/10, untouched by the graph waves (user default: leave
      parallel-session work alone). Six items: (1) `cmd/cqrs-bench` go.sum
      untidy (`TestEveryModuleGoSumIsTidy` fails); (2) BuildFlow pseudo-version
      hygiene — `metaengine/go.mod` sqliteengine `v4.5.2` off the zero
      pseudo-version rule; (3) 34 modules need `go mod tidy`; (4) golangci red
      in `system/`, `scheduling/sqlstore`, `stack/sqlite`; (5) go-licenses
      FAIL for `metaengine/bigtableengine`; (6) BuildFlow govulncheck runs
      go1.26 against go1.27 sources. core/v5 arch-lint + doc-check
      alias-ambiguity are tracked separately (BDD-harness Externals row + Code
      Quality row). Decide: one sweep session, or per-owner distribution.
      — source: 2026-10-10 07-37 §f31–37 (minus the separately-tracked), 09-05
      §f19–26 _(Effort: M triage + S-M fixes)_

---

## Code Quality

- [x] **Branching-flow duplicate-type triage follow-ups (2026-10-09)** — the
      465-row duplicate-type analysis was fully triaged
      ([plan](docs/planning/2026-10-09_18-16_SUPERB-branching-flow-triage.md)):
      ~95% already-governed (ADR-0152 v4↔core/v5 mirrors, annotated queue/engine
      dialect twins, sanctioned per-engine pair types, intentional DTO/markers).
      Shipped same day: `metaengine.SortColumn` = `SortSpec` alias (same-module
      split brain) + projectionhost shutdown flush via `context.WithoutCancel`.
      **Closed 2026-10-10**: (a) mirror-drift lockstep is now a hard gate —
      `TestMirrorLockstep` in `cmd/api-stability/mirror_lockstep_test.go`
      compares the 9 v4↔core/v5 mirror pairs against the api golden with a
      classified drift register
      (`cmd/api-stability/testdata/mirror_lockstep_allowlist.txt`; stale rows
      must be pruned; new `core/v5/<pkg>` must register). Same commit deduped
      the golden itself — the generator double-emitted nested-module symbols
      (2204 phantom lines; the entire apparent id/command/query mirror lag was
      duplicate-line fiction). Real lag today: event only (eventtest subtree
      + 3 delivery-mark funcs). (b) `scripts/dupe-signal.sh` filters mirror
      groups from `branching-flow dupe` output (104 groups suppressed, 77
      signal groups kept, suppressed list stays visible) — the tool's
      `//nolint:branching-flow` is ignored by the dupe analyzer (verified
      against installed 0.6.4 and the tool repo's local master).
- [ ] **`core/v5` needs a `.go-arch-lint.yml`** — 14 production packages and
      growing, `TestMultiPackageModulesHaveArchLintConfig`
      (cmd/api-stability) is red since the package count crossed the
      threshold (visible 2026-10-10; sibling v5 sessions actively adding
      packages — coordinate before writing the dependency rules, they encode
      the v5 layering design).
- [ ] **branching-flow upstream: `dupe` analyzer ignores
      `//nolint:branching-flow`** — the flag `--include-suppressed` exists but
      duplicate-type findings never suppress, in installed 0.6.4 AND local
      master (verified 2026-10-10, three directive placements tested). Fix
      belongs in `~/projects/branching-flow` (active session there —
      coordinate); `scripts/dupe-signal.sh` is the interim wrapper and becomes
      obsolete once dupe honors directives.
- [ ] **Watch the 3 baselined `.templ` clone groups (specview noscript pair,
      eventcatalogview catalogSection pair + Breadcrumbs ×4)** — the 2026-09-28
      dedup campaign (archived
      [`2026-09-28_02-21_deduplication-52-clone-groups-session.md`](docs/status/archived/2026-09-28_02-21_deduplication-52-clone-groups-session.md))
      worked the red set 52 → 3 Go-side zero (32 extractions + ~40 accept
      directives) and closed with the sanctioned structural re-pin (baseline
      60 → 187 groups; mutation-tested: novel-shape clones flag red, gate
      green). The 3 templ groups have NO suppression lever — `//art-dupl:accept`
      does not parse in `.templ` (empirically verified, HTML comments included) —
      so they live in the baseline until art-dupl learns templ directives or
      the docserver pages gain extracted shared components. ⚠ Semantics caveat
      discovered 2026-09-28: `check` matches groups by normalized-shape HASH,
      not location — a verbatim copy of already-baselined code does NOT flag;
      the gate detects novel duplication shapes, not instances. _(Effort: S to
      watch, M to extract the templ components)_
- [ ] 🔥 **Dedup-campaign verification tail — remaining: (a) composed `#verify`
      + (d)-mysql live legs only** — every other leg closed green: (b) `#lint`
      88/88 modules 2026-09-29 (G703 nolints moved to the MkdirAll/WriteFile
      SINK lines in `catalog/cmd/ec-fixture`, dupl accept on the two
      declarative rule-table files in cqrs-lint); (c) `#load-sweep` ✅ 2026-09-28;
      (d) `#integration-pg` planned-scan ✅, `#integration-dgraph` SanitizeIdent ✅,
      `#integration-redis` watermill ✅ (green after the `MessageToEvent` payload
      fix); (e) `-race` metaengine/watermill/projectionhost/system/queue ✅
      zero failure lines; (f) compile-verify systemtest + mesh-demo ✅.
      (a) is quiet-window-gated (`can-run-composed-gate`, load storm 163 at
      2026-09-28 22:00); (d)-mysql = `#integration-mysql-vm` cowLookup leg.
      — source: archived 2026-09-28 04-04 §b/§f1-14 _(Effort: S remainder)_
- [x] **Unify `metaengine.graphNeighborsFallback` onto `metaengine.GraphBFS`**
      — DONE 2026-10-01 (M21): `metaengine.GraphBFSNodes[N any]` is the typed-key
      generic core (never-nil result); `GraphBFS` is a thin string-specialized
      delegate (signature unchanged) and `graphNeighborsFallback` maps through
      the core. Nil-vs-empty unified to never-nil; covered by the existing
      fallback tests.
      Original: core's degraded-path BFS (graph_fallback.go) still carries its own
      copy of the loop with different semantics: `[]any` frontier,
      `typedNodeKey` dedup, `nil` result for depth<=0 (engines normalize to
      `[]any{}`). Merging needs a deliberate nil-vs-empty behavior decision
      and a typed-key encode param. Not a clone-gate violation today (shape
      differs enough that art-dupl does not group it). _(Effort: M)_
- [ ] **>350-line production files (~54, 2026-09-06 count)** — see the
      cqrs-lint section for the verified picture, gate-policy options, and the
      already-split offenders; the code-file split waves are a standalone
      multi-session program pending the policy decision. Decide
      harness-dir exemptions (adttest/enginetest are exported test harnesses)
      first. NEW offenders as of 2026-09-28 (daemon-era waves, gate-red until
      split): `catalog/docserver/docserver.go` (351),
      `catalog/eventcatalog/frontmatter_convert.go` (364),
      `catalog/cmd/ec-fixture/main.go` (356). _(Effort: XL, multi-session)_
- [ ] [BLOCKED] **macOS verification of ephemeral PG** —
      `scripts/ephemeral-pg.sh` claims cross-platform but was only
      static-review-tested; a GitHub Actions macOS runner leg is the
      verification route. _(Effort: M)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) —
      userspace MariaDB coverage exists but not the full nspawn env. Now also
      covers the MySQL claiming integration tests. _(Effort: M)_
- [ ] **Shuffle eval + adoption for `scripts/test-integration.sh` /
      `test-all-backends.sh`** — the two composite runners execute the same
      suites UNshuffled (documented parity gap in gotchas-testing.md).
      Gated on ROADMAP OQ #9 (are the composite scripts staying?). — source:
      02-16 §b3/§f4/§f5
      _(Effort: S)_
- [ ] **Backport contention-retry review to turso/badger engines** —
      dgraph got the execution-layer retry; check whether turso/badger have
      an analogous transient-abort class worth the same treatment. — source:
      02-16 §f19
      _(Effort: M)_
- [x] **Unify ephemeral-script passthrough conventions** — DONE 2026-10-01
      (M21): ephemeral-pg.sh + ephemeral-dgraph.sh now exec verbatim `"$@"`
      when args present (pg's go-prefix special case and EXTRA_ARGS append
      mode removed); redis/nats raw passthrough unchanged — one convention.
      Documented in gotchas-testing.
      Original: ephemeral-pg.sh
      uses positional EXTRA_ARGS, ephemeral-dgraph.sh uses
      TEST_ARGS/TEST_ARGS2, redis/nats use raw passthrough; three
      conventions for the same job complicate evaluations. — source: 02-16
      §e5/§f25
      _(Effort: M)_
- [ ] **Watch dgraph + redis CI jobs (~10 shuffled runs)** — record any
      seed that fails; rare orderings WILL eventually appear in CI (that is
      the point of shuffling). — source: 02-16 §e7/§f10
      _(Effort: XS)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1–7 done. Pre-cut deprecation markers shipped 2026-08-17; migration
> guide at `docs/V5-MIGRATION-GUIDE.md`. Snapshot wire tags (T18) DONE
> 2026-09-06 — dual-read fallbacks in snapshot/pebble, SQL columns migrated by
> `MigrateSnapshotColumnsToStream` (auto-run by every InitSchema). Error-family
> codes renamed to stream vocabulary 2026-09-08 (17 codes, 9 modules; E4 of the
> extended review thereby RESOLVED).
>
> **Execution order + dependencies (M26, 2026-10-01):** every row below is
> sequenced with its blockers in
> [`docs/planning/2026-10-01_v5-cut-readiness-checklist.md`](docs/planning/2026-10-01_v5-cut-readiness-checklist.md)
> (Layer 0 rulings → quiet-window legs → pre-cut migrations → branch →
> deletions in cascade order → flips → docs → tag). The scan-default flip
> (G-T14) executes ON the v5 branch there.

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it. _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)** —
      multi-collection batch atomicity + auto-projection replaces them. Also
      removes the remaining `aggregate_*` SQL surfaces wholesale. _(Effort: M)_
- [ ] **v5 cut: typed tombstone filter + sort-type eval in core/v5** —
      `kv.TombstoneQuerier.QueryByTombstone(ctx, excludeTombstoned, onlyTombstoned
      bool)` encodes a 3-state filter as two bools (interface-bound in v4; the
      `storage/view` implementation is deleted at v5 anyway) — core/v5/kv should
      take a typed `TombstoneFilter` enum instead. While forking, evaluate the
      `metaengine.SyncWritesTier(volatile, syncWrites bool)` signature and whether
      the v5 module family wants one shared sort-directive type
      (`kv.OrderClause` / `metaengine.SortSpec` are shape-identical today).
      _(Effort: S–M)_
- [ ] **Delete `graph.GraphProjection`** — auto-projection + graphadapter
      replaces it. _(Effort: S)_
- [ ] **Delete `stack.Bundle` + all 8 stack presets** — `system.System` is the
      only composition root; `stack/` module deleted entirely (incl.
      `stack/bench` and `stack.RunProjections` → `projectionhost.Host`). _(Effort: M)_
- [ ] **Delete deprecated compat shells from ADR-0126** — `schema.VersionedStore`
      + `NewVersionedStore`, `signing.Rejecting*` forwarders,
      `encryption.ErrInnerStoreNot*` aliases, `metadata.CustomData`. _(Effort: S)_
- [ ] **Delete `storage/sql.BuildWhereClause`** — `BuildWhereClauseChecked` is
      the validated replacement. _(Effort: XS)_
- [ ] **Breaking `record.NewStreamRef` validation** —
      `NewStreamRef(streamType, entityID string) (StreamRef, error)` rejecting
      an empty entityID; migrate call sites. Owner-confirmed 2026-08-22
      (decision memo:
      `docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md`
      Appendix B). _(Effort: M)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) — final
      v4.x tags exist; drop from go.work/flake testModules/api-stability list,
      then delete at the cut. Confirm they die BEFORE anyone renames their
      proto fields (sweep §4). _(Effort: M)_
- [ ] **Delete deprecated tombstone metadata API (ADR-0114 completion)** —
      remove `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/
      `TombstoneStatus`/`Metadata.Tombstone`; pre-reqs: type-driven status in
      `listing` (replaces the DetectTombstone call at listing/in_memory.go:155),
      migrate `example/taskmanager` off `OnTombstone`, regen golden. _(Effort: M)_
- [ ] **Rest of sweep §4 (wire-vocabulary renames):** REMAINING (2026-09-22
      pass): (a) SQL `events`/`commands` column renames + migrations —
      recommendation on the table: v5.x expand-contract, NOT the v5.0 cut
      (assessment in `docs/WIRE-FORMAT-KEYS.md`; owner ruling pending);
      (b) consumer grep for old code strings in sibling alert/dashboard
      configs at the cut; (c) `listing.aggregate_projection` collection-name
      rename (TBD). DONE in this pass + earlier waves: pebble event rows
      (`aggregate_*` → `stream_*` with decode-only legacy fallback — the
      last binary surface; census gap closed in WIRE-FORMAT-KEYS), watermill
      dual-read/dual-write, bbolt/pebble command + snapshot rows, benchkit
      keys, pebble slog keys, error-family codes, E1 encoding stamps typed
      as `codec.Encoding`, and the central wire-key table doc itself
      (`docs/WIRE-FORMAT-KEYS.md`). — source: 08-41 §b1/§f1–11, 07-48 §f5-13
      _(Effort: S remaining)_
- [ ] **v5 ADR: encryption-at-rest configuration** — SKELETON SHIPPED
      2026-09-13 as [ADR-0139](docs/adr/0139-v5-encryption-at-rest-configuration.md):
      `DriverConfig.Encryption` + `KeyProvider func(ctx) ([]byte, error)` (vs raw
      `key []byte` — the KeyProvider lean enables rotation/hot-reload and keeps keys out of
      config structs; sets THE precedent for pg/mysql passwords too) +
      `system/` DeploymentConfig key-reference slot (env/file/secret-manager
      ref, never the key). Engines fail construction loudly when unable to
      honor (precedent: `RejectDurabilityTier`, `MaterializedViews`).
      REMAINING for the ADR: owner ruling on the 4 open questions
      (provider call semantics, reference validation timing, read-model
      scope, plaintext→encrypted migration), then implementation.
      — source: 20-18 §f15-17/§f21, 20-57 §f9-10
      _(Effort: L)_
- [ ] **Migration-verification tail for T18:** 2026-09-22 pass: mixed-state
      corruption test, mid-migration (half-renamed) failure-path test,
      concurrent-init idempotency test, legacy-subset test, and a LIVE
      DuckDB run (`integration/snapshot_migration_duckdb_integration_test.go`,
      green against the embedded engine) are ALL landed. REMAINING: the live
      MySQL/MariaDB run — `snapshot_migration_mysql_integration_test.go` is
      ready, gated on the userspace MariaDB / `#integration-mysql-nspawn`
      leg in a quiet window. — source: 08-41 §f13–23 _(Effort: S — infra leg)_
- [ ] **v5 items from extended review — EXECUTED 2026-09-22** — E1 (event
      envelope Encoding typed as `codec.Encoding`; `record.Encoding`
      rejected — its closed enum would drop custom codec stamps), E7
      (`HandlerRetryConfig` rename + deprecated aliases), E8 (typed
      `middleware.Kind`), E11 (`AdapterCore.Encode` error return), E13
      (phantom param documented: Go has no generic methods — the honest
      resolution), E15 (`dispatcher.Middleware[H]` alias unification).
      E3/E9/E10/E14 verified already-done in earlier waves (bbolt
      errorfamily, turso Policy write guards, ShutdownDependency
      validation, OwnedDBHandle type split). Remaining: golden/meta-tests
      pass + cut. _(Effort: — )_
- [ ] **Post-landing sweep for the data-model series** — api-stability
      meta-tests, doc-check over skill refs, consumer-pin sweep for `record/v4`
      consumers under GOWORK=off (MarshalBinary lesson). _(Effort: M)_
- [ ] **Expand V5-MIGRATION-GUIDE** with before/after examples per v1 tier
      (incl. `relational → metaengine`) at the cut; add the envelope v2
      consumer note ("old readers compatible; no action needed") and
      operator verification snippets for the snapshots migration; sweep the
      asrecord/MIGRATION_TO_STACK/PRESETS guides once v5 nears. — source:
      08-26 §c6, 08-41 §f25–27
      _(Effort: M)_
- [x] ~~**systemtest tag-wave tail: strip the module's sibling replaces**~~ —
      **DONE 2026-10-08 (T02 Wave F):** `systemtest/v4.0.0` FIRST TAG cut,
      pushed, proxy-smoked ✓; working-tree sibling replaces dropped post-tag
      (0 `=> ../` remain) alongside taskmanager's four and cqrs-bench's
      benchkit pin; `check-example-standalone --build` green at 0 findings.
      The Feedback #4 split itself was DONE 2026-09-22 (Tier-7 `systemtest/`
      module owns system's real-engine suites; receipt: CHANGELOG).
      — source: 23-24 followups §f22, feedback doc §4.4
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate. _(Effort: M)_

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [x] ~~[BLOCKED] **M22 / Q3: report-artifact policy for narrow skill triggers**~~ —
      **RULED 2026-10-08: sanctioned deviation** — a narrow trigger (one or two
      modules) may answer in chat + "full report on request"; the full artifact
      is mandatory only for whole-project triggers. Codify in the crush-config
      status-report skill at its next edit (source repo, not the fan-out).
      — source: archived 2026-09-27 23-43 §g3, 2026-09-28 02-22 §g3,
      05-40 §b
- [ ] **README review deep-read tail (2026-09-13 cluster, 8th-pass harvest)** —
      ~~(b) add READMEs to the doc-check gate (flake app/CI)~~ and ~~(d) quick-start
      drift-guard tests~~ BOTH DONE 2026-09-29: all 97 workspace READMEs are gated
      surface (doc-check auto-discovery, 2,420 refs / 86 pkgs green, CI inherits),
      and the quick-start drift class is covered by the corpus + arity checker
      (the 6 arity lies it caught WERE the drift; per-module duplicate tests ruled
      redundant). [(a) doc-check repoRoot fix, (c) T37 deep-reads, (e)
      deprecated-symbol gate, (f) link checker — all done 2026-09-20.]
      — source: archived 12-16 §f (150-154, 158-161, 174, 179) _(Effort: M total,
      sliceable)_
- [ ] **M13 tail: per-module fresh-run last-verified stamps + script-derived
      counts** — FEATURES guarantee rows carry 09-21 doc-gate stamps, but
      per-module fresh-run verification stamps need quiet CPU to be honest;
      extend `check-canonical-facts.sh` to derive the go.mod count into
      FEATURES too (F69 overlap, kills the last hand-maintained count).
      ALSO derive (2026-09-28 addition, M7 found the drift class): engine
      implementations (12), registered drivers (11, register.go census), and
      ADTs (12) — the hand-maintained counts rotted 10-vs-11-vs-12 across
      FEATURES/skill/ROADMAP before the 2026-09-28 fix sweep. —
      source: archived 15-34 §b4/§f5, 14-12 §f23 _(Effort: M, quiet-CPU)_
- [ ] **docs-health pass hygiene — ~~(a) index-vs-disk gate~~ DONE 2026-09-25 (canonical-facts status leg + 18-row rot fix), ~~(b) harvest-ledger convention~~ DONE (crush-config harvest-guide), ~~(d) pass-checklist rows~~ DONE (verify-checklist + md-go/docs conventions) — ~~(c) weekly docs-health cadence~~ RULED 2026-10-08: **weekly standing pass (agent-side until billing fixed)** + foreign-lint rule **comment-only handoff** (no foreign fix-forward during a live concurrent session; see ROADMAP OQ17). Original: (a) index-vs-disk
      gate: extend `check-canonical-facts.sh` (or a sibling) to derive
      live-report count vs the README table, archived count vs the day-table
      sum, and day-row presence per archived day (the 9th AND 10th passes
      almost shipped index rot); (b) mechanical harvest ledger (per-report
      item → new-row/existing/declined table) as a pass artifact; (c) weekly
      docs-health cadence decision (owner); (d) pass checklist (index update
      step + banner vocabulary pointer) — suggest upstream to the crush-config
      skill repo. — source: archived 23-31 §e1-3/§f15-20 _(Effort: S gate + XS conventions)_
- [ ] **Bench-gate/tooling single-mention tail (16-37 §f, never harvested)** —
      `--explain <bench>` per-sample diagnostics; `--json` evidence output for
      quiet-window-run; deep-quiet window probe/logger; variance-aware
      stability probe (two-axes verdicts); CI baseline-artifact 5-new-entries
      confirm; fragile p99/max threshold sweep across gates;
      `SUM_VIA_GROUPED/baseline` +34.5% one-off investigation; auto-embed
      noise verdict + CoV + GOVERSION into `--save` headers; DirectSQL A/B
      gate-set decision (dep-budget review first); README ops section for
      `quiet-window-run`/`nightly-bench`. — source: archived 16-37 §f14-26/§f35
      _(Effort: S-M each, sliceable)_
- [ ] **ADT recipe gaps (T18 ratchet waivers, 2026-10-10)** — 5 of 11
      metaengine ADTs have no recipes.md heading; each is waived with reason
      in `cmd/doc-check/adt_coverage_test.go` and stays visible in test
      output until closed: **ADTSet** (FoldSet mirrors the Map fold — add
      alongside the next Set consumer), **ADTLog** (log-tail reads documented
      only in readmodels.md tier notes), **ADTStreamLog** (single prose
      mention), **ADTSortedMap** + **ADTMultimap** (no fleet consumer).
      Writing a recipe for any of them removes the waiver — the gate refuses
      stale waivers. _(Effort: XS each)_
- [ ] **`metaengine/COOKBOOK.md` fences have ZERO automated checking** —
      verified 2026-10-10: md-go-validator walks `docs/` only and doc-check
      scans READMEs + skill references, so COOKBOOK go fences are unchecked
      at BOTH parse and compile level (recipes.md, the other major cookbook,
      is compile-gated). Cheapest honest fix: add `metaengine/COOKBOOK.md` to
      the md-go walk (parse-level, XS); compile-level (recipes-catalog
      harness) only if COOKBOOK grows hand-written snippets that rot.
      — source: 2026-10-10 07-37 §f28, verified this session _(Effort: XS
      parse-level)_

---

## benchkit statistical-rigor tail (2026-09-16)

> The 02-09 session shipped P100 exactness, `RunRepeated`/`RepeatedResult`,
> per-metric `MetricVariation`, real benchstat samples, and load provenance —
> all verified green. The 09-35 session closed the two blockers (queue clones,
> soak gocyclo) and refreshed the FEATURES coverage line (151+43). What follows
> is the consolidated remainder. — source:
> [`docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md`](docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md)
> §b/§f, 09-35 §f P3

- [ ] [BLOCKED] **Supersede-note on the oversubscribed 2026-09-19 capture** —
      annotate `docs/benchmarks/2026-09-19_backend-comparison-variation.md` as
      superseded (keeping it as the what-noisy-looks-like example) once a
      quiet-window capture exists. Blocked on machine quietness: load was
      35.6/32 CPUs at the 2026-09-21 attempt. Protocol: wait for
      `scripts/calibration-gate.sh` PASS, re-run the compare command from the
      capture header, then annotate. — source: archived 15-37 §f3/§f30

- [x] ~~[BLOCKED] **Benchkit tag wave (owner go-ahead)**~~ — **DONE 2026-10-08
      (T02 Wave E, go-ahead granted by blanket authorization):** `benchkit/v4.7.0`
      cut+pushed+proxy-smoked ✓ with the statistical-rigor + polish-tail APIs
      (~+17 exports); `cmd/cqrs-bench/v4.3.4` pin-bumped + its sibling replace
      stripped; LICENSE corrected to the root form (Lars Artmann). Run log:
      `build/release-logs/batch-20261008-154909.log`. — source:
      archived 15-57-benchkit §f15/§g1

---

## CV consumer-verdict follow-ups (2026-09-16)

> CV (real consumer, evented-funnel-core Phase 0 GO) ran a parity-gated
> four-tier read-model benchmark against metaengine/storage/sqlstore and
> Phase-0 seam spike over a real 6.2k-event store. All their API-fit claims
> were re-verified against source: [`docs/reviews/2026-09-16_cv-verdicts-reflection.md`](docs/reviews/2026-09-16_cv-verdicts-reflection.md).
> The `Scan` 100-row-default doc lie is FIXED in the same change (doc comment +
> FAQ entry); `ApplyBatch` atomicity stays tracked under v5 Unification
> (ADR-0123 §10) — CV's 3.4 s → 109 ms pragma measurement is the perf
> argument for it.
>
> **Execution sequencing for this section + the metaengine/system reliability
> items lives in the Pareto plan
> [`docs/planning/archived/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md`](docs/planning/archived/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md)**
> (P0: tag wave + replay-starvation fix + ApplyBatch atomicity; P1: lease +
> FilterContains + Forever + E9/E10 + matview guard; P2: v5 deletions + E-items +
> AggregateOn seam; P3: proof + docs + v5.0.0 cut).

- [ ] [BLOCKED] **CV single-writer lease ask (tracked jointly with M20 (a) /
      ADR-0146 in Metaengine follow-ups)** — CV's Phase-0 ADR conditions every
      library-store cutover on a CV-owned `metaengine.RegisterDriver`
      decorator wrapping their `<dsn>.lease` single-writer marker, because the
      library has NO engine/store-level lock. One-pager delivered 2026-09-21
      (verified current reality: lease semantics live only in `queue/` +
      `claiming/` task claims; recommendation = `EngineConfig.SingleWriter`
      advisory `<dsn>.cqrs-lease` flock, fail-loud default-off; becomes
      ADR-0146 on ratification). REMAINS OPEN: owner ratification +
      implementation before v5 freezes engine construction surfaces.
      — source: reflection doc §4.2 _(Effort: gated on owner ruling)_

- [ ] **`FilterContains`/`FilterPrefix` FilterOp extension** — metaengine
      FilterOp today is exactly eq/ne/lt/le/gt/ge/in (`enum_validation.go:71`);
      substring search degrades to a client-side full scan (CV measured
      2.3–28 ms vs 326 µs hand-rolled LIKE). Native engines map to
      LIKE/prefix; closure fallback evaluates in Go. Fits the v5 FilterOp
      window. — source: reflection doc §4.5 _(Effort: M)_
- [ ] **go-idempotency `Forever` → adapter mapping (gated on upstream v0.4.0)** —
      when go-idempotency ships the `Forever` sentinel, `idempotency/sqlstore`
      + `idempotency/kvstore` must write `expires_at = math.MaxInt64` DIRECTLY,
      never via `expiryFromTTL` (verified by execution: `now.Add(ttl).UnixNano()`
      wraps negative past year 2262 → key dead on arrival — the unrecoverable
      direction). Dedup the verbatim-copied `expiryFromTTL` (kvstore:46,
      sqlstore:173) while touching both; add the overflow boundary test; pin
      bump middleware/sqlstore/kvstore (all v0.3.0 today) via the
      go-ecosystem-upgrade skill. — source: reflection doc §3.1/§4.7 _(Effort: M once upstream lands)_
- [ ] **benchkit cross-tier PARITY gate** — before any tier-vs-tier benchmark
      number is trusted, assert cross-tier result identity (full snapshot,
      stat counts, ranked IDs) — the template CV's four-tier benchmark
      proved out. — source: reflection doc §5; SUPERB plan T27/M102 _(Effort: M)_
- [ ] **Tuned-tier metaengine benchmark** — `BuildLayoutPlanFromType`
      composite layouts vs hand-indexed SQL, BOTH sides tuned, so future
      latency claims carry no defaults-vs-tuned asymmetry (CV's fairness
      finding). — source: reflection doc §5; SUPERB plan T27/M101; Goal-closure plan G-T16 extends this to the fold-tier goal-parity shape _(Effort: M)_

---

## metaengine Goal-closure follow-ups (2026-09-17)

> Source plan: [`docs/planning/2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md`](docs/planning/2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md)
> (distance analysis: ~55–60% consumer-experienced). Executes AFTER/ALONGSIDE
> the 2026-09-16 excellence plan (shared items cross-referenced there, never
> duplicated). Success definition in plan §7.

- [ ] [RULED 2026-10-08] 🔥 ~~**DIRECTION RULING: what does the Goal's "declare
      ONLY" mean after the `Infer` deprecation?"~~ — **RULED: HYBRID (option c,
      the memo's recommendation)** per
      [ADR-0151](docs/adr/0151-goal-direction-evolutions-are-the-declaration.md)
      (blanket authorization): the Evolution IS the declaration — reframe
      executed (AGENTS.md Goal sentence amended, ADR-0116 Layer-1 addendum,
      G-T03 `cqrs-gen` one-pager
      [PARKED](docs/planning/2026-10-08_gt03-cqrs-gen-fold-codegen-one-pager-PARKED.md)
      behind the evidence gate: G-T16 parity numbers or a named consumer ask).
      Evidence pack:
      [`docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md`](docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md).
      Gate A unblocked (ADR + routing integration tests ≥2 engines remain,
      T24). — G-T01/G-T02/G-T03
- [ ] **Scan default v5 decision — RULED 2026-09-21 (owner): Option C**
      (unbounded at the v5 cut + cqrs-lint nudge + operator ceiling; survey:
      [`docs/planning/archived/2026-09-21_scan-default-v5-survey.md`](docs/planning/archived/2026-09-21_scan-default-v5-survey.md)
      — consumer census incl. `system.Find` inheriting the cap; documented-100
      stays the loud v4 status quo in godoc+FAQ). The v4-safe add-ons LANDED:
      `metaengine.WithDefaultLimit(n)` plan option (operator ceiling for
      un-limited scans; explicit `WithLimit` wins; survives Replan) with
      tests, and cqrs-lint **F031** `scan-without-limit` (warn/low, suppressed
      by `WithDefaultLimit`, disabled in library presets). REMAINING: flip the
      built-in 100 to unbounded — executes ON THE V5 BRANCH per the ADR-0123
      cut plan. — G-T14 _(Effort: S impl)_
- [ ] **FEATURES maturity flip for the closed surface (🧪→✅)** — earned by
      the plan's gates (not declared): evidence links per row, CHANGELOG
      Goal-story entry, release notes. Final stamp of Goal closure. — G-T25
      _(Effort: S, gated on gates A–D)_
      **GATE STATUS (verified 2026-09-21, still BLOCKED — do not flip):**
      Gate A needs the owner Direction Ruling recorded as an ADR
      (G-T01/G-T02 — the 🔥 row above, unanswered; ADR-0141 slot was taken
      by temporal cells) plus routing integration tests across ≥2 engines.
      Gate B needs the G-T16 parity benchmark + new-surface conformance
      sweep. Gate C/D need the release wave (W0 ~90-tag publish) and the
      example-green-under-two-engines proof (sqlite ✓; postgres leg landed
      2026-09-21, runs under `#integration-pg`). This row executes only
      after ALL of those are earned.

---

## Watermill sibling skill follow-through (2026-09-15)

> `.agents/skills/watermill/` shipped (broker powers/tradeoffs/limits). Report:
> [`docs/status/archived/2026-09-15_19-45_watermill-skill-session.md`](docs/status/archived/2026-09-15_19-45_watermill-skill-session.md)

- [x] ~~**NATS JetStream roundtrip test leg** — `watermill-nats/v2` +
      `scripts/ephemeral-nats.sh`, mirroring `TestRedisStreamRoundtrip`; if it
      lands, an `.#integration-nats`-style flake app + CI leg analog to
      `#integration-redis`.~~ DONE 2026-10-01 (M19): `TestNatsJetStreamRoundtrip`
      + edge suite (Nack redelivery, consumer-group exactly-once, 2 MiB payloads)
      over `watermill-nats/v2`, runnable via `scripts/ephemeral-nats.sh` or the
      `nix run .#integration-nats` flake app (CHANGELOG `[Unreleased]` receipt;
      upstream plugin gaps documented in the watermill skill). — source: 19-45
      §c1/§f2/§f10

## Temporal versioned cells — ADR-0141 follow-ups (harvested 2026-09-18)

> From the temporal deep-dive reports
> ([14:07](docs/status/archived/2026-09-18_14-07_temporal-versioned-cells-deep-dive.md) §f items 19–44,
> [16:03](docs/status/archived/2026-09-18_16-03_temporal-versioned-cells-completion-gates.md)).
> Items 1–18 of the 14:07 list + docs/CHANGELOG/FEATURES/golden/lint work are DONE
> (see those reports); the core API (`VersionedStorage`, `MapSetAt`/`MapGetAsOf`,
> `temporal-asof` rule, memory/sqlite/bigtable engines) is green and documented
> (recipes §2.37, advanced §6.20, readmodels versioned-engine note). Engine
> enumeration in tooling (api-stability, cqrs-lint `StoreBigTable` +
> `metaengineEngineFromImport`) landed 2026-09-18 evening session.

- [ ] [RULED 2026-10-08] 🔥 **Real-GCP validation + prior calibration for
      `bigtableengine`** — **RULED: NOT tag-blocking** (the 🧪 experimental
      status already communicates the validation gap; real-GCP validation and
      prior calibration stay gated on GCP access/credentials — none on this
      machine). Run the suite against a real BigTable instance when access
      exists, then calibrate `NsPerOp`/RTT priors via `CALIB_DUMP=1` +
      `scripts/calibration-drift.sh`. — source: 14:07 §f20/§g2,
      `metaengine/bigtableengine/README.md` _(Effort: S each, gated on GCP access)_
- [x] **Property-based temporal tests for memory version chains** — DONE
      2026-09-29 (M18): `metaengine/temporal_property_test.go` — three rapid
      properties vs a reference model (out-of-order + same-ts LWW collapse
      with shuffled application order; retention-never-prunes-newest under
      adversarial MaxVersions=1 + MaxAge; set→tombstone→rebirth as-of
      boundaries at ±1ns and midpoints). All green ×100 draws. — source: 14:07 §f22,
      `metaengine/version_chain.go`
- [x] ~~**Pebble/bbolt versioned cells — scope decision for the next wave**~~ —
      **RULED 2026-10-08: (A) DEFER with demand trigger** (blanket
      authorization adopting the decision note's recommendation). Both keep
      natural prefix-range machinery if a consumer ever asks; the 3-engine
      versioned set stands. Decision note:
      [`docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md`](docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md).
      — source: 14:07 §f31
- [ ] **Soak env-var run for bigtableengine** — per
      `docs/agents/gotchas-testing.md` soak conventions (`-race` covered by
      `#verify`). — source: 14:07 §f33-34 _(Effort: S)_

## md-go-validator CI integration (from 2026-09-13 audit)

> The gate SHIPPED 2026-09-21 (M23): `#check-md-go` green on the 1,642-block
> corpus (1461 valid / 78 skipped / 103 baselined archived), wired into
> `#verify`, `#verify-fast`, and ci.yml; P2+P3 swept (67 `// skip-validate`
> insertions across 34 files); P4 policy = archived-only baseline with
> inert-shrink ratchet. Build narrative: the archived 18-19 + 23-24 delta
> reports. The harvested open tail: — source: 18-19 §f, 23-24-delta §f

- [ ] [BLOCKED] **Upstream md-go-validator (owner repo; verify-before-filing)** —
      (a) relative-path baseline mode (deletes every consumer's sed
      re-absolutization layer); (b) `--save-baseline` exits 0 when the save
      succeeds (wrappers should not need `|| true`). _(Effort: M + XS)_

## go-graph-rag feedback follow-ups (2026-09-15, triaged 2026-09-19)

> Consumer evaluation of `metaengine/v4.13` + `system/v4.7` (adopted neither —
> "system is a category error for a library, metaengine wrong-shaped for
> GraphRAG"). Feedback #3 (fail-closed Save) and #5 (doc.go stamps) are FIXED
> and released in the 2026-09-21 v4.9.0 wave; #2 → Turso §, #4 → v5 §,
> #6 → metaengine rows, #1/#7/#8 → ROADMAP. Source:
> [`docs/feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md`](docs/feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md)

- [ ] [RULED 2026-10-08] **Update the go-graph-rag consumer** — **COMMS
      APPROVED** (blanket authorization; github-voice + the shipped-fix receipts):
      tell the consumer feedback #3 + #5 are fixed and released
      (`system/v4.9.0` `ErrRacySaveRefused`/`WithRacySave`/
      `ErrEventSaveNotAtomic` + the 17 `# Experimental` doc.go stamps); invite a
      re-test on v4.9.0. Executes in the T25 filings/comms pack. — source:
      23-24 followups §f5/§f40 _(Effort: S)_
- [ ] [RULED 2026-10-08] **goal-shaped-app postgres e2e CI leg** — **RULED:
      ephemeral PG in the examples CI job** (the `#integration-pg`
      ephemeral-server pattern, no services container needed; billing-gated
      like all remote legs until R20 clears). The test itself already exists
      (DSN-gated, green under `#integration-pg`). — source: closeout §f11/§g3
      _(Effort: S)_
- [ ] [RULED 2026-10-08] **Does `#test-examples` join blocking `#verify`?** —
      **RULED: split ownership** — CHANGED examples join blocking `#verify`
      (diff-driven module set; the projectionhost double-apply class dies
      there); the FULL examples suite stays in the weekly/nightly rotation
      (cost honesty — it is too slow for every run). Implementation: verify
      app learns the changed-example detection (S, rides T03's composed
      re-record). — source: 23-24 followups §f11/§g2 _(Effort: S)_

---

## 92-tag release-train tail (2026-09-20 harvest)

> Harvested from the train execution report's §f (`docs/status/archived/2026-09-20_10-25_tag-wave-execution-92-tags-ci-templ-triage.md`)
> and the verify-arc reports; items since done are struck inline there. The
> owner questions ride the W3 bundle section below.

- [ ] **CI tail from the train (six legs, all root-caused)** — ~~(a) benchkit
      load-gate fixture env propagation~~ DONE-BY-VERIFICATION 2026-10-04
      (T12: the fix already landed — `flake.nix` `#check-release-scripts` and
      `calibration-gate.sh` self_run both force `CI=false` into the fixture
      invocations; full harness re-run under `CI=true nix run
      .#check-release-scripts` = EXIT 0, zero failing checks, runner env
      simulated locally) · ~~(b) coverage-gate pinned/setup Go toolchain~~
      DONE-BY-VERIFICATION 2026-10-04 (T12: the `coverage-gate` job already
      carries `actions/setup-go` v7 with `go-version-file: go.mod`, comment
      cites the exact `go: downloading go1.27.1` triple-death trap) ·
      ~~(c) auto-retry-once for cancelled/failed infra legs~~ DECLINED
      2026-10-04 (T12 ruling): the failure half ALREADY exists (autoretry.yml
      reruns failed jobs once, attempt==1 bound — infra evictions that mark
      jobs failed are covered); extending to `cancelled` runs would fight
      `cancel-in-progress: true` on BOTH CI and Nightly (every superseded
      push/dispatch would get an unwanted retry); the motivating
      toolchain-download class died with (b)'s setup-go. Overturnable in one
      edit if a genuinely-infra cancelled class is ever observed. ·
      (d) triage the 04:12 nightly-gates failure; (e) Module Isolation Build
      leg-set instability; (f) one clean post-fetch-depth-fix run confirming
      `TestTagContentMatchesChangelog`. Remote confirmation of the whole set
      is billing-gated (row above). —
      source: archived 10-25 §b1/§f5-10 _(Effort: M total)_
- [ ] [RULED 2026-10-08] **Upstream filings — APPROVED** (blanket
      authorization) via the T25 filings pack; every claim re-verified against
      latest upstream first (verify-before-filing), drafted in Lars's voice
      (github-voice):
      (a) turso-go native-lib hash-mismatch + lazy-init failure
      family (`TestBackend_LazyInit_Concurrent` /
      `TestVectorSearch_LibSQLPushdown` red-ing 1–3 legs intermittently);
      (b) exhaustruct_v5 `skippedNamed` panic (repro ready; isolate first —
      synthetic shapes do NOT trigger; run the analyzer over the historical
      pre-fix tree `eea1c3c66^`);
      (c) go/types + x/tools parallel-check race.
      — source: archived 10-25 §f3-4/§f20, 15-37 M14 _(Effort: S each)_
      2026-09-22 repro-prep session findings on (b): the upstream module is
      `dev.gaijin.team/go/exhaustruct/v5` (the gaijin fork golangci wraps as
      exhaustruct_v5) — `github.com/4meepo/exhaustruct` is 404-GONE, file at
      the Gaijin forge. A singlechecker harness built + ran v5.0.3 (kept at
      `/tmp/exhaustruct-repro`, rebuild: go get
      dev.gaijin.team/go/exhaustruct/v5/analyzer@v5.0.3); minimal synthetic
      shapes (same-pkg + cross-pkg embedded, unkeyed-element and keyed-field
      forms) do NOT trigger the panic — the trigger is subtler than the 16-51
      report's one-liner; the definitive repro is running the analyzer over
      the historical pre-fix tree (refs around `eea1c3c66^`, storage/pebble +
      stack presets still carried the old literal forms). Isolate before
      filing; check the fork's latest version for a fix first (Gate 5).
- [ ] **Release-tooling polish tail (sliceable)** — ~~batch-release.sh --from-manifest~~ DONE 2026-09-25, ~~pin-sweep --dry-run lists standalone-verifies~~ DONE, ~~TestTagContentMatchesChangelog train threshold as error~~ DONE (calibrated <5 hard, 5-9 note), ~~verification-ladder doc~~ DONE (gotchas-testing), ~~templ-generate canonical-cwd contract~~ DONE (catalog README), ~~scheduler-otel-status row in core.md §9~~ DONE (+ mesh-demo row), ~~check-example-standalone.sh~~ DONE (baselined taskmanager/mesh-demo pending replaces), ~~SOAK_SKIP_BOLT doc for the loop~~ DONE 2026-10-03 (gotchas-testing "Soak test env vars" bullet carries SOAK_SKIP_BOLT=1 with the #verify 8m-timeout rationale). Remaining: smoke-all timing/resume/cache, check-templ leg-first summary, cqrs-upgrade dogfood sentinel, exclusion-map unification, 6-place registration consolidation, LSP GOTOOLCHAIN config (2026-10-04 finding: no crush.json at ~/.config/crush and .crush/init is empty, so the GOTOOLCHAIN=auto injection point is missing — needs a crush-config skill pass, not a repo edit). Original: smoke-all: per-module
      timing, `--resume` checkpoint, artifacts under `~/.cache/` never /tmp;
      `check-templ` leg-first error summary; `batch-release.sh --from-manifest`;
      `pin-sweep --dry-run` lists its standalone-verifies;
      `TestTagContentMatchesChangelog` train-section threshold (>=10 tags) as
      an error; document `SOAK_SKIP_BOLT=1` for the per-module loop;
      cqrs-upgrade dogfood sentinel on goal-shaped-app; api-stability's four
      exclusion maps unified into one table; the 6-place module registration
      consolidated (go.work, flake testModules/examplePaths, layer script, api
      exclusions, cqrs-lint catalog) or a `new-module` scaffold;
      verification-ladder doc (meta-tests → touched-lint → verify-fast →
      verify) in AGENTS testing gotchas; LSP/gopls `GOTOOLCHAIN=auto` config
      (kills ~95 noise diagnostics per session); templ-generate canonical-cwd
      contract in the docserver README/gate echo; scheduler-otel-status row in
      core.md §9; `check-example-standalone.sh` (example require audit — would
      have caught the B6f forward-pin abort). — source: archived 10-25 §f13-14/§f31-38/§f41-50 _(Effort: M total)_
- [x] ~~[BLOCKED] **Daemon pre-commit sanity gate** (three sessions asked)~~ —
      **RESOLVED-BY-EXISTING-GATE 2026-10-08 (receipt):** the staged-`.go`
      syntax gate in `.git/hooks/pre-commit` (`scripts/check-staged-go.sh`,
      added after the 2026-08-16 staged-corruption class) already blocks the
      mid-write corruption class BEFORE any commit, daemon included. Full
      test-gating of heuristic daemon commits is DECLINED: concurrent sessions
      guarantee in-flight red tests; gating would stall the daemon for minutes
      per commit. The workspace-build pre-commit leg + weekly `#verify` remain
      the nets. — source: archived 10-25 §f17/§d6, 09-40 §e1
- [ ] [RULED 2026-10-08] **BuildFlow templ-generate cwd fix** (external repo) —
      **FILING APPROVED** (T25 pack, verify-before-filing): the pre-commit
      templ step runs from the repo root and re-corrupted a correct
      regeneration (FileName paths baked in); until fixed upstream the
      documented regenerate-from-`catalog/docserver/` + `--no-verify` pattern
      stands. — source: archived 10-25 §d3/§f16 _(Effort: S, external)_

---

## Upstream asks from cqrs-htmx (harvested 2026-09-22, docs-health D1)

- [x] ~~**Upstream `requestContextEnricher` into `event/`**~~ — DONE 2026-10-01
      (M24, GitHub #35): shipped as `event.RequestScope` + `event.WithRequestScope`
      + `event.RequestScopeEnricher` (+ `RequestScopeFromContext`), recipe
      recipes §2.42 (CHANGELOG `[Unreleased]` receipt). The cqrs-htmx local copy
      (`audit_context.go`) can drop at its next family bump — reminder drafted in
      the 2026-10-03 plan T26. Source: cqrs-htmx TODO_LIST P3 ask (3); verify pass
      there 2026-09-22.
  - ~~`system.New` checkpoint/DLQ store options~~ CLOSED 2026-09-28 as ALREADY SHIPPED (ADR-0149: `NewEngineCheckpointStore` durable checkpoints; DLQ-by-events per ADR-0117; receipts in CHANGELOG `[Unreleased]` + ADR-0149) — deleted as a row; residual = publishing (`system/v4.10.0` uncut in the stalled wave).
  - ~~`System.Explain` per-query Volume/placement~~ DONE 2026-09-28 (`Store.QueryPlacements()` + Explain rendering; CHANGELOG `[Unreleased]` receipt) — deleted as a row.

## go-cqrs-lite skill hard-block refocus (2026-09-24 harvest)

Source report: `~/projects/crush-config/docs/status/2026-09-24_13-59_go-cqrs-lite-skill-repoint-and-refocus.md` §f.
The skill now hard-requires `system` + `metaengine` for new apps (ADR-0123) and
refuses to guide manual composition. All nine content/tooling follow-ups from
that refocus shipped same-day (routing matrices, quickstart snippet, modules.md
engine rows, v5-tier sweeps, cookbook link, experimental-status notice, ADR
link, preset deprecation visibility, accessor re-verification — CHANGELOG
`[Unreleased]` "Skill: hard-requires" entry); the late-harvest batch (banner
propagation, canonical-surface decision, eval negatives, changelog entry,
dprint, ADR-0123 addendum) shipped the same day — completed rows deleted
2026-09-28 per the no-completed-rows policy. Surviving open item:

- [ ] 🔥 [BLOCKED:tooling] **Execute the evals (currently UNVALIDATED)** (`evals/trigger-eval-set.json`, `evals/evals.json`)
      after the description change — needs the `claude` CLI, currently blocked. — source:
      report §f7 _(Effort: M, tooling-blocked)_

---

## Declined / Rejected (do not re-litigate)

> Guard list, not a backlog: these were investigated and closed with rationale.
> Re-open only with new evidence or an explicit owner decision.

- **command.Bus / MemoryBus removal (v5 candidate)** — DECLINED 2026-08-29:
  the in-process bus is 47 lines, delivers the documented saga/example flows,
  and complements `watermill/`. Re-evaluate only if the saga pattern moves to
  a dedicated orchestration module.
- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
- **Rewrite `check-module-layers.sh` as Go NOW** — deferred. The script is
  stable (348 lines). Revisit when complexity grows significantly.
- **Fix LogBackend same-nanosecond collision** — atomic-counter cost not
  worth the theoretical correctness gain; a counter may be off by 1.
- **Migrate F001/F002/F005/F014 to per-module coaching** — workspace-global
  by design; low leakage risk.
- **`System.WaitReady(ctx)` API** — declined in the file-renamer TOCTOU
  review: the catch-up drain closes the race; a readiness API would be a
  false guarantee. See
  `docs/feedback/archived/2026-08-13_file-renamer_drain-live-toctou-race-review.md`.
- **file-renamer circuitbreaker/dlq modules** — rejected; failsafe-go + the
  FAQ pointer cover it. See
  `docs/feedback/archived/2026-08-13_file-renamer_extract-circuitbreaker-and-dlq-review.md`.
- **Deep per-item annotation of archived reports (V3 T42)** — declined per
  the 2026-08-29 audit precedent ("So what?" test): archived location +
  claim-level corrections carry the historical signal; mass-striking
  wishlist tails is noise. Revisit only if a specific archived report
  misleads.
- **Benchkit "Phases 6/7 remain" flag** — stale: production replay +
  `benchtest.RunSuite` shipped (ROADMAP Theme 2). Do not harvest.
- **`metaengine.PlanFromSQLite(dsn, ...)` convenience API** — declined
  2026-09-06: comment rot fixed to reference real helpers; add the API only
  if a consumer asks. — source: 07-42 §f30
- **Ack-window pipelining for CatchUpSubscriber** — deferred by design:
  ~160-280K ev/s ceiling with a documented 10× degradation trigger. — source:
  07-42 §c5
- **Per-engine high-water marks for CatchUpSubscriber tail-only re-catch-up** —
  DECIDED AGAINST 2026-09-13: full rebuild is idempotent by construction (engine
  is Reset first), while tail-only replay must assume engine state matches a
  persisted watermark — the exact stale-state class the catch-up race fix
  closed; watermark bookkeeping also needs ADR-0136 reset semantics. Revisit
  only if `Replayed`/`CompletedAt` observability shows rebuild latency hurting
  failover SLOs.
- **Additive `CatchUpEngineWithResult` API** — DECIDED AGAINST 2026-09-13: the
  result is already observable via `CatchUpSnapshot`; the breaking `ResetResult`
  return waits for v5 where signature changes are free.
- **KeyProvider tier (env/file composite provider)** — deferred to ROADMAP;
  the bank-sync ask is closed by the shipped helpers. — source: 08-26 §f14

## From the CRM's Graph-ADT debut (2026-10-10, first real consumer)

- [ ] **Recipes §graph: the Graph ADT end-to-end pattern** — the Kith CRM
      (github.com/LarsArtmann/crm, relations feature 2026-10-10) is the
      FIRST real consumer of `metaengine.Edge`/`EdgeRemoval` folds +
      `NetworkInput{Undirected}` undirected traversal over sqlite's
      recursive CTE. Document the working recipe: dual collections (record
      view for lists + `relation-graph` for edges), stateless EdgeRemoval
      folds carrying BOTH endpoints in the payload, the `Undirected`
      field-name planner contract, and the `[]any` ExecuteCtx cast. Source:
      crm `internal/domain/relation` + `internal/app/projections.go`
      (relationGraphProjection) + the crm audit report
      crm `docs/status/2026-10-10_06-59_relations-network-graph-feature.md` (research verdict inside).
- [ ] **Integration candidate: `example/graph-native --dgraph` against the
      ephemeral Dgraph harness** — the example carries a `--dgraph` flag
      (projections route to Dgraph) but nothing exercises it against a real
      server; `nix run .#integration-dgraph` already provides ephemeral
      Dgraph. Extend the example's integration story or add a systemtest leg.
      — source: 2026-10-10 07-37 §f30 _(Effort: S-M; rides the CRM-driven
      graph polish above)_
