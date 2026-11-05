# Status Report — 2026-11-05 18:16 CET: Fleet-Audit Follow-Up Plan, One Week Later

> Point-in-time snapshot. Scope: execution state of
> [`docs/planning/2026-10-24_19-40_SUPERB-fleet-audit-followup-pareto-plan.md`](../planning/2026-10-24_19-40_SUPERB-fleet-audit-followup-pareto-plan.md)
> (reviewed 2026-10-25 03:27) plus repo health relevant to it. Sibling report covering
> today's active session (09:46→18:10): [`2026-11-05_18-10_sse-root-cause-ci-triage-verify-interrupted.md`](2026-11-05_18-10_sse-root-cause-ci-triage-verify-interrupted.md).

**Scope line — VERIFIED here:** `git log`/`git for-each-ref` (291 commits, 30 authored since 10-25; tag creatordates); fresh `bash scripts/check-file-size.sh` (exit 0); `wc -l` on the trio (273/230/164); `jq '.modules|length' versions.json` (112); greps for every plan-task marker (T08/T09/T11/T13/T14/T15/T18/T19) against scripts, workflows, modules, TODO_LIST, CHANGELOG; CHANGELOG wave sections + the 2026-11-04 systemcatalog reconciliation note; TODO_LIST rows 180/1191/2464.
**NOT verified here:** no `#verify` run (today's attempt 13 was externally SIGINT'd at ~18:08 per the 18-10 report — zero real failures observed before the interrupt; last full green receipt is 2026-10-27 `c75d2c556`); CI state not fetched via `gh`; TODO_LIST row-truth beyond spot-greps not audited; the five auditors' repos untouched.

---

## a) FULLY DONE (since the plan)

1. **P1 — Un-red HEAD: DONE.** The file-size trio was split: `scheduling/sqlstore/store.go` 382→273, `system/adapter_event.go` 352→230, `metaengine/layout.go` 360→164. Fresh run today: `check-file-size.sh` exit 0 ("no new offenders, no growth"). Full verification pass recorded green 2026-10-27 (`c75d2c556`).
2. **P0 — Tag waves: SHIPPED (2 of 4 trains fully, see b).** Waves cut: 2026-10-26/27 `catalog/v4.11.0` + `system/v4.16.0` + `systemcatalog/v4.0.0` (tag re-cut 11-04, see d); 2026-11-03 `cmd/cqrs-lint/v4.19.0` + `catalog/v4.12.0` + `id/v4.7.4`; 2026-11-04 `cmd/cqrs-lint/v4.20.0`. **mesh-demo healed:** the pre-release replace is GONE from `example/mesh-demo/go.mod`, `ALLOWED_REPLACES` back to `taskmanager` only — the ride-until-tag pattern died at the tag exactly as designed (T01/M8/M9).
3. **T16 — SHIPPED as a module (the plan's biggest owner-gated item).** The catalog Registry-from-DomainConfig design became `systemcatalog/v4`: `FromDomainConfig`/`FromSystem`/`DeriveInto` + `catalog.AssertInventory` drift gates + `System.CommandRegistrations()/QueryRegistrations()` introspection (2026-10-26 wave). F032 `no-catalog-import` now makes catalog adoption CI-blocking for consumers (owner policy 2026-11-03), with `systemcatalog` as the system-first satisfaction path — dnsblockd's ~40-line-registry pain is answered upstream.
4. **T08 (script half) — `check-example-standalone.sh --build`** now GOWORK=off-compiles every example against published pins (Leg 2, lines 94–97) — the MarkDemo-class detector exists.
5. **T09 — mysqlengine twin test** `typed_wal_test.go` exists (mirror of pgengine's).
6. **T10 — api-stability golden** regenerated (6,440 exports) today.
7. **T12 —** art-dupl triage/consolidation session 11-04; error-taxonomy tables regenerated from erraudit v0.7.0 today (`71be8a45d`).
8. **T15 (partial) —** faq↔cqrs-upgrade cross-reference present.
9. **Beyond the plan:** 11-03 full-backlog Pareto plan (27 tasks routing 211 TODOs) + red-to-green trust plan; open-issues sweep closed 11-04/05; SSE root-cause double-bug fix + deterministic regression tests today (see 18-10 report).

## b) PARTIALLY DONE

1. **P0 wave incomplete by train:** `pgengine` (latest tag `v4.4.3`, pre-10-24 — the `AsType` compile fix is still untagged) and `systemscenario` (latest `v4.2.1`, 10-17 — the hollow-root WARNING is still untagged). Low harm (no tag ever contained the pgengine bug; the warning is additive), but T01's wave definition included both, and today's pending M07 wave (catalog, kv, id, metaengine, sqliteengine, cqrs-lint) also excludes them.
2. **T08 CI wiring:** the `--build` leg exists in the script, but `ci.yml` has no `example-standalone` leg — open as TODO row 180 ("T26 untagged-API CI gate"). The detector is built; it is not yet automatic.
3. **T11:** file-size IS now in `ci.yml` (2 legs) ✅; the nightly-matrix extension is NOT (no file-size/example legs in `nightly-gates.yml`), and the silent-manifest question (why `--check --remote` stayed quiet) remains unverified.
4. **T15:** recipes.md §2.43 hollow-root warning note still missing (faq entry done).
5. **Today's M04/M05:** verify attempt 13 externally interrupted (88 packages, zero real test failures before the SIGINT); CI green-on-master receipt still needs one clean relaunch + push.
6. **T17 half-shipped:** `scenario.WhenVersioned` shipped; the Registry-bridge + verify-remaining-half work is rowed (TODO_LIST:177, the 11-03 plan's T23).

## c) NOT STARTED (plan leftovers — T13/T14/T18/T24/T25 un-rowed; T17/T20 WERE re-derived by the 11-03 plan at TODO_LIST:177/:174)

- **T13** — AGENTS.md gotcha rows (untagged-API-in-examples; tag-vs-checkout blast-radius rule). No rows anywhere.
- **T14** — explain rule cards → RULES.md pointer (`explain_rules.go` has 0 RULES.md references).
- **T18** — empty-domain warning opt-out (`WithAllowEmptyDomain`). Owner-gated; not rowed.
- **T19** — md-go-validator rev-pin vs master-track. Owner-gated; flake input still `?ref=master` — and the gate is now on fire (d1). Rows exist (1191/2064) but the DECISION is open.
- **T24** — daemon-commit retitle-or-accept. Not rowed.
- **T25** — static empty-DomainConfig advisory sketch. NB: "E021" is now TAKEN (E021 schema-declaration findings exist in consumer runs) — the sketch must renumber.

## d) TOTALLY FUCKED UP

1. **`#check-md-go` is un-runnable** (TODO row 1191 🔥): the pinned md-go-validator derivation no longer builds (missing `pprof` zip via ginkgo chain) — "every doc change since is unvalidated by this gate." This is the 7th recurrence of the master-tracking tax the plan's T19 was written to kill; the decision is still open and the gate is red-in-effect.
2. **The `systemcatalog/v4.0.0` tag fiction (reconciled 2026-11-04):** the 10-26 wave entry claimed a tag that was never pushed; it was cut 11-04 at the `catalog/v4.11.0` release commit. Reconciliation also surfaced the open D7/D8 problem: "master's daemon line has drifted far past every release tag… no commit is yet release-line-coherent across the full train" — owner work, still open.
3. **Three external-interference events hit today's session** (/tmp wipe ×2 destroying logs+drafts, CI benchmarks job externally cancelled, SIGINT to in-flight verify) — long runs are fragile repo-side-out.

## e) WHAT WE SHOULD IMPROVE

1. **Row-or-kill discipline for plan leftovers:** T13/T14/T18/T24/T25 survived a week with zero TODO_LIST rows — later plans (11-03) re-derived their own lists (and DID catch T17/T20) but the doc-truth and owner-gated tail fell between. Harvesting at session close was the plan's own §Close-a-Wave rule; it half-happened.
2. **Stop master-tracking md-go-validator** — seven vendorHash fires is a standing tax with a one-decision fix (rev-pin).
3. **Verify-launcher as a script + repo-side logs** (today's evidence; already rowed by that session — echo, not duplicate).
4. **Check the cqrs-lint rule-number registry before sketching new E0xx rules** (T25's E021 collision).

## f) Top next tasks (impact-sorted; plan-arc scoped)

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Relaunch M04 verify clean (today's session owns it) — the M05/M06/M07 chain is blocked behind it | Critical | M |
| 2 | Fix `#check-md-go` (T19 decision + re-pin) — a dead gate validating nothing since ~10-24 | Critical | S/M |
| 3 | Add pgengine + systemscenario to the M07 wave — closes T01's definition completely | High | S |
| 4 | Wire `check-example-standalone.sh --build` into ci.yml (TODO row 180) — makes the MarkDemo class automatic | High | S |
| 5 | One-page owner decision sheet for T18/T19/T24/T25 (+ nightly-legs appetite) — prepared, one meeting | High | S |
| 6 | T14 explain→RULES.md pointer + tests; T13 AGENTS gotcha rows | Medium | XS |
| 7 | Harvest remaining plan leftovers into TODO_LIST (or explicitly kill them) so they stop depending on this file | Medium | S |
| 8 | T15 recipes §2.43 hollow-root note (+ doc-check when gate 2 is fixed) | Low | XS |
| 9 | Nightly file-size/example legs — decide if still wanted once CI wiring (f4) lands | Medium | S |

## g) Top question I cannot figure out myself

**Should pgengine and systemscenario ride the M07 wave, and can I prep the T17/T18/T19/T24 decision sheet for one sitting?** The §7 tag-vs-red-gates question resolved itself by events (un-red first, then waves — option 2), but two of the four original wave trains are still untagged and nobody has re-asked; and the four surviving owner-gated leftovers (§c) are decision-shaped, not work-shaped — one sheet, one meeting, closes the arc.
