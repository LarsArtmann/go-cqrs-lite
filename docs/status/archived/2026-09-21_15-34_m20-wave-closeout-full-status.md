> **RESOLVED-BY-ROUTING — docs-health 10th pass (2026-09-21):** the M20 one-pagers + M16/M13/M08-local wave shipped (§a). Every open tail is now tracked: M10 real VM run + M13 stamps → [TODO_LIST.md](../../TODO_LIST.md) (CI / Docs sections); M08 remote → TODO CI tail row; M26 NATS → TODO watermill row; M14/M15 → TODO README-review row; M17/M18/M19/M25/M27 → the owner-unblock plan; the §f28-32 design-ratification follow-ups (ADR-0146 SingleWriter, AggregateOn first cut, routing v1) → new TODO Metaengine row. Archived.

# Status Report — M20 One-Pagers Wave + M16/M13/M08-local/M26-tail (session close-out)

> Point-in-time snapshot: **2026-09-21 15:34 CEST**, master `5808e54bf`, clean of
> session-authored files (foreign benchkit/turso session still live on the tree).
> Session window: ~14:36–15:34 CEST. Continues
> [`2026-09-21_14-12_w0-burn-class-guards-w3-rulings-execution.md`](2026-09-21_14-12_w0-burn-class-guards-w3-rulings-execution.md)
> under the [owner-unblock-trust plan](../planning/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md)
> ("execute the WHOLE list"; owner flagged **M20 as important**).
> **Supersedes the 15:05 interim report from this same session** (that one is
> annotated below — two live reports from one session was sprawl).

## a) FULLY DONE

All items verified with the repo's gates, committed (mostly via the auto-commit
daemon — see d1), and pushed through `5808e54bf`.

| #  | Item                                                           | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| -- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1  | **M20 F109 — lease one-pager** (owner-flagged)                 | [`docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md`](../planning/2026-09-21_engine-single-writer-lease-one-pager.md). Verified current reality: lease semantics exist ONLY in `queue/` (ClaimDue/heartbeat/`ErrLeaseNotHeld`, `queue/store.go:24,52,99-101`, ADR-0134) and `claiming/` — both task claims, neither arbitrates engine open; `system.EngineConfig` has no open-mode field. Recommendation: `EngineConfig.SingleWriter` advisory `<dsn>.cqrs-lease` flock, fail-loud default-off, one shared Tier-0-style helper. Routing: ADR-0146 candidate on ratification; G-T01 adjacency noted (operator-config axis).                                                                                                                                                                                                                                |
| 2  | **M20 F110 — `AggregateOn(fn, column, group)` seam one-pager** | [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](../planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md). Problem verified: planner prices per `ReadPattern` (`EngineProfile.ReadCosts`, `engine.go:125`) but aggregate SHAPE lives in runtime `TypedReader.Count/Sum/...` calls — plan-time invisible; matview execution-side coverage EXISTS (`sqliteengine/aggregations.go:26` `serveScalarMatView`, unfiltered scalars only) yet can't influence routing/pricing. Proposal: QueryOption stamped on `QueryDecl`, `MatViewSpecReporter` capability, construction-time validation, hint-not-requirement. Scope guard: scalar-covered ONLY — grouped routing unsafe until upstream turso-go defect A (IVM cross-tx delta loss, flip-runbook gate) is fixed. Rejected: Infer-style reflection (deprecated surface).                           |
| 3  | **M20 F111 — Scan-default v5 survey**                          | [`docs/planning/2026-09-21_scan-default-v5-survey.md`](../planning/2026-09-21_scan-default-v5-survey.md). Default 100 verified at `typed_reader_scan.go:19,270`; doc warnings already loud. Census: `goal-shaped-app/app.go:209` scans capped silently; benchkit explicit `WithLimit(100)`; **`system.Find` inherits the cap** (`system/runtime.go:131` appends only when >0); CV (external) hit the truncation. Recommendation: flip to unbounded at the v5 cut + cqrs-lint nudge + optional `WithDefaultLimit(n)` operator ceiling. Feeds G-T14 (owner-gated decision).                                                                                                                                                                                                                                                                                                |
| 4  | **M20 F112–F114 — filing + routing + TODO strikes**            | One-pagers under `docs/planning/` with Routing sections (ADR candidates + G-T01/G-T14 adjacency); TODO lease row struck-with-pointer (ratification + implementation remain), routing row's S28 design step marked DONE with link (routing v1 implementation remains open — honest partial), G-T14 row updated survey-delivered.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| 5  | **M07 dispositions + CHANGELOG wave entry**                    | `[Unreleased]` Added: corruption-tripwire + verify-ergonomics entry (hash-golden, load guard, verify-lock, preflight, wait-loop, go-version gate, CI coverage rebuild, vm-mysql hardening); `check-changelog-symbols.sh` verified 11 citations honest.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| 6  | **M08 local — autoretry workflow**                             | `.github/workflows/autoretry.yml`: on CI/Nightly-Gates failure, rerun **failed jobs only**, bounded to `run_attempt == 1` (one extra partial run max; real breakage stands; Benchmarks deliberately excluded). actionlint clean.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| 7  | **M08 local — leg-set investigation + trim**                   | Diff: CI dynamic matrix = 96 entries vs flake `testModules` = 87; drift = root, typedfixture, 6 examples, eventtest. Trimmed root (proven no-op: root module `go test ./...` → "no test files", rc 0) and `cmd/cqrs-lint/testdata/typedfixture` (0 test files) from the matrix — each saved leg was a full nix-installing runner. Examples KEPT (their tests run nowhere else).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| 8  | **M16 — alias bundle to true zero-warning**                    | Advisory count grew 5 → 12 fences via iterative unmasking (one report per alias, next instance unmasked per fix) across advanced/core/faq/readmodels/recipes — every affected fence now imports the exact package (stack/{sqlite,mysql,postgres,memory}, queue/mysql, schema, storage/memory). **doc-check: 1206 refs, 54 packages, zero warnings.** `TestRecipes` harness green (21s cold, 4.8s warm).                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| 9  | **M16 — recipes §2.11 live-latency sync**                      | Every claim re-verified against source, section accurate as-written: default probe 1s + timeout 5s + jitter 0.2 (`probe.go:89-115`), `DefaultRoutingHysteresis = 0.20` + `DefaultRoutingMinDelta` (`store_routing.go:17-24`), `StartAutoReplan(ctx, interval) (stop func())`, `Replan`, `GetEngineStats`, `FormatLiveLatency`, SELECT-1 probes. No edits needed.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| 10 | **M16 — status navigation**                                    | Per-file live-reports table (17 entries, one-line summaries) in `docs/status/README.md`; per-DAY wave table (132 day-rows over 1222 archived files, incl. the odd `release-fix-*` name normalized) in `docs/status/archived/README.md`. Docs-truth TODO row struck fully resolved (Scan docs 09-17, aliases today, overflow probe embedded 09-18).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| 11 | **M13 — FEATURES maturity census**                             | Scripted bidirectional diff: header claimed 82 go.mods, disk has **96**. Six missing rows added (goal-shaped-app, scheduler-otel-status, storage/backuptest, system/integration, testutil/pgtestcontainer, testutil/mysqltestcontainer) with source-verified descriptions; sub-package row naming explained (4 no-go.mod rows + the `event/eventtest` ↔ `event/v4/eventtest` variant). Post-edit bidirectional diff clean: every rowable module rowed, no stale rows.                                                                                                                                                                                                                                                                                                                                                                                                    |
| 12 | **M13 — guarantee stamps**                                     | Architecture-Guarantees section now carries dated last-verified stamps separating this session's doc-gate evidence (doc-check/doc-links/changelog-symbols/release-scripts 86/86/actionlint) from the 2026-09-20 16:39 composed `#verify`. TODO docs-censuses row struck (all three 7th-pass items).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| 13 | **M26 skill tail (F148/F149/F150 partial)**                    | Ran the 3 drafted trigger evals: P1 (Router/Kafka ordering) fires; **P2 (atomic commit with MySQL writes) and P3 (browser live-updates) were trigger MISSES** even though the skill answers both (Forwarder=outbox at SKILL.md:55-59; browser push table at internals.md:88-89) → description extended with those phrases. New [`references/advanced.md`](../../.agents/skills/watermill/references/advanced.md): NotBefore delay, evidence-carrying Requeue (`facts.Requeued`+`RequeueEvidence`, `store.go:86-91`), ClaimMetrics/WithClaimMetrics/Metrics(), Router fan-in patterns, 7-row troubleshooting table — all source-cited. Cross-links BOTH directions (sibling SKILL.md → advanced.md; go-cqrs-lite skill §6.4 → sibling skill). Claims-checklist rule codified in `docs/agents/gotchas-tooling-build.md`. TODO watermill row updated with the eval outcome. |
| 14 | **Wave-close report**                                          | [`2026-09-21_15-05_m20-onepagers-m16-m13-m08local-wave.md`](2026-09-21_15-05_m20-onepagers-m16-m13-m08local-wave.md) — now superseded by THIS report (annotated).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |

Gate discipline this session: after every wave — `check-doc-links` (0 broken of
824 targets), doc-check (zero warnings), `TestRecipes` (ok), `check-changelog-symbols`
(11 honest), actionlint (clean). 30+ commits authored/absorbed, all pushed.

## b) PARTIALLY DONE

1. **M10 — the real `#integration-mysql-vm` run is the ONLY remainder.**
   Hardening (stale-port pre-flight, process-group trap) landed last wave and is
   self-tested; the live proof run never happened. System load oscillated
   **8 ↔ 72** all session (concurrent benchkit sessions' heavy phases); the
   hand-rolled load-watch never saw a stable load1 < 5 window. Running it under
   load is precisely the attempt-burn class the W0 guards were built to kill, so
   deferring was the disciplined call — but it means M10 stays open.
2. **M08 remote half** — nightly 04:12 triage + TagContent clean-run confirm need
   remote Actions evidence (owner question Q2 unanswered).
3. **M26 NATS leg (F146–F147)** — `ephemeral-nats.sh` exists; the
   watermill-nats roundtrip test leg + optional `#integration-nats` flake app are
   unbuilt (CPU-bound). Upstream-latest verification for the four plugins also
   open.
4. **M13 per-module stamps** — guarantee-section stamps done; per-module
   fresh-run verification stamps need quiet CPU to be honest.
5. **Wave-close report freshness** — its "will run the VM leg in the first quiet
   window" plan-of-record aged within minutes (see d2); THIS report is the
   corrected record.

## c) NOT STARTED

- **M11/M12 upstream filings** — owner-APPROVED ("go file" on record) but the
  verify-before-filing repros (exhaustruct_v5 `skippedNamed` panic, go/types+
  x/tools race, turso-go pin-bump probe) are CPU-bound and were load-blocked all
  session.
- **M14** (READMEs into doc-check gate), **M15** (quickstart drift guards) —
  author+test units, deferred to avoid half-landed states under load.
- **M17/M18** goal-shaped-app adoption demos — untouched.
- **M19 FilterContains/FilterPrefix** — gated on owner Q1 (pin-tag wave timing).
- **M22–M27** — M22 appears OWNED by the currently-live benchkit session (their
  uncommitted `benchkit/benchtest.go`, `sweep.go`, `cmd/cqrs-bench/flags.go`,
  `benchmark-regression.sh` edits + new `docs/benchmarks/baselines/` are on the
  tree right now); M23/M25/M27 untouched.
- Improvement idea filed, not built: doc-check `--list-all-ambiguous` mode
  (emit ALL instances per alias — would have collapsed today's 5-round iterate).

## d) TOTALLY FUCKED UP

1. **Lost the commit race to the auto-commit daemon FOUR times.** The #1 house
   rule is "commit immediately after verification — the daemon absorbs work in
   minutes." I still lost: (a) the M20 one-pagers themselves (my commit
   `a655034fe` landed as only the final cite-fix — 3+/2− — wearing the full M20
   message because the daemon's `59acc1842` had already taken the three files);
   (b) an `exit 1` no-op commit when `git add` found nothing left to stage;
   (c) `5808e54bf` carried 1 of the 4 files I staged (the other 3 rode daemon
   commits). Content was never at risk — everything reached `origin/master` —
   but authored history is again shredded into `chore:` waves. The fix I
   adopted mid-session (stage+commit as ONE immediate bash call) worked when I
   used it; I didn't use it every time.
2. **M16 estimate was wrong and cost 5 doc-check rounds.** "3 known advisories"
   counted ALIASES, not INSTANCES — doc-check reports the first instance per
   alias, and each fix unmasked the next (12 fences total). AGENTS contract #14
   documents this exact iterative-unmasking behavior for art-dupl; I should have
   swept all instances per alias with one grep BEFORE the first gate run. ~4
   avoidable gate cycles.
3. **Two invented details nearly shipped in the FEATURES census rows.** I wrote
   a wrong env name (`PGTEST_DSN`; actual `DATABASE_URL` > `POSTGRES_TEST_DSN`)
   and a false driver-registration claim for testutil — caught only because I
   re-verified my own citations before committing (both fixes pre-commit). This
   is the exact doc-lie class the repo gates against, and it came from ME while
   censusing OTHERS' honesty.
4. **Report sprawl from my own session** — I added a second live status report
   (15:05) that went partially stale on arrival (its VM-run plan-of-record), and
   the live-report count kept climbing against the >10 advisory I myself was
   indexing. This report supersedes 15:05 and annotates it; the count problem
   remains for the next docs-health pass.
5. **Minor mechanical friction**: two failed edits on readmodels.md
   (not-viewed guard, then non-unique match) — both avoidable by reading first,
   which I did for every other file; inconsistent discipline, wasted round
   trips.

## e) WHAT WE SHOULD IMPROVE

1. **Atomic commit protocol, always**: `git add <files> && git commit` in ONE
   bash call the second a file is final — never batch across a verification gap.
   (Adopted mid-session; must become reflex.)
2. **Sweep before iterate**: for any gate that reports "first hit per key"
   (doc-check aliases, art-dupl groups), run the own-sweep grep first — one
   pass, not N rounds.
3. **Claims-checklist on my OWN prose, every commit**: the rule codified today
   (verify env names/defaults/numbers inline, not just file:line cites) exists
   because I nearly shipped two fabrications. It should run before commit, not
   as a post-hoc rule.
4. **Use the built load tooling**: I hand-rolled a `/proc/loadavg` poll loop;
   `can-run-composed-gate.sh --wait-loop` already encapsulates wait+retry with
   the extra safety checks. Use the product.
5. **doc-check `--list-all-ambiguous`**: one mode emitting every instance per
   alias would have saved today's rounds — candidate for the M23
   md-go-validator wave.
6. **Foreign-artifact protocol**: the stray `metaengine/tursoengine/P\x11B`
   SQLite test artifact (untracked, daemon won't commit it) should be trashed
   once the owning session's run ends — it currently survives only because the
   daemon ignores untracked files; one careless `git add -A` away from
   committed garbage.
7. **Live-report hygiene**: 19 live files vs the >10 advisory; next docs-health
   pass should harvest + archive the 09-20/21 cluster (many are same-day
   chain-reports from concurrent sessions).

## f) NEXT (up to 50, owner-independent unless noted)

**Immediate (unblocked, quiet-window items marked ⏳):**

1. ⏳ M10: run real `#integration-mysql-vm` through the hardened script; F52 AGENTS rows + TODO evidence + strike.
2. M11: exhaustruct_v5 `skippedNamed` panic — fresh repro on latest → minimal repro → github-voice → file → link TODO. ⏳ (CPU)
3. M11: go/types+x/tools race repro → file. ⏳
4. M12: turso-go pin-bump probe (`TestBackend_LazyInit_Concurrent`) → file with recommendation. ⏳
5. M13: per-module fresh-run last-verified stamps. ⏳
6. M14: READMEs into doc-check gate (flake app + CI leg).
7. M15: quickstart drift-guard tests (stack/sqlite, storage/memory, decider, scheduling, projectionhost).
8. M16-tail: doc-check `--list-all-ambiguous` mode.
9. M17: goal-shaped-app pin bump → system v4.8.0; adopt `DomainConfig.Events` + `.On`.
10. M17: coeffect-loud Doctor demo.
11. M18: scenario Given/When/Then test.
12. M18: snapshot story demo.
13. M18: AsyncAPI export + cqrs-lint consumer probe.
14. M22: benchkit polish tail — CHECK OWNERSHIP first (live session's files on tree).
15. M23: md-go-validator commit baseline + flake app + CI leg; then P2 (9 consumer-facing blocks).
16. M25: rapid property tests (memory version chains). ⏳
17. M25: sqlite restart soak. ⏳
18. M25: bigtable `MapUpdateAt`/MaxAge decisions (GCP smoke cred-gated).
19. M26: NATS JetStream roundtrip leg + `ephemeral-nats.sh` wiring. ⏳
20. M26: optional `#integration-nats` flake app if leg lands clean.
21. M26: upstream-latest verification (redisstream/kafka/amqp/sql plugins).
22. Trash `metaengine/tursoengine/P\x11B` once the owning session's run ends (verify nothing references it).
23. Consolidate live status reports below the >10 advisory (harvest 09-20/21 cluster).
24. M27 long tail (existing TODO bullets, 1:1): cqrs-lint FP-sweep refresh; audit follow-ups; strict-gate holes; smoke-all resume/timing; `--from-manifest`; templ leg summary; TagContent train threshold; exclusion-map unification; new-module scaffold; verification-ladder doc; LSP env; scheduler row; `check-example-standalone.sh`.

**Ratification-gated (owner answers unlock):**
25. M19: `FilterContains`/`FilterPrefix` FilterOp (native LIKE + closure fallback, enum validation, parity vs CV's 2.3–28ms measurement) — **Q1**.
26. M08 remote: nightly 04:12 triage + TagContent clean-run confirm — **Q2**.
27. M11/M12 actual filing (process answer if "go file" is superseded — **Q3**).

**Design-ratification follow-ups from TODAY's M20 memos:**
28. ADR-0146: ratify → implement `EngineConfig.SingleWriter` + shared lease helper (verify-lock mechanics productized) + Doctor line.
29. AggregateOn first cut: `AggregateSpec` on QueryDecl + validation + `MatViewSpecReporter` + planner O(1) pricing for scalar-covered + Doctor INFO for uncovered.
30. G-T14 execution: v5 unbounded Scan default + `WithDefaultLimit(n)` operator ceiling + cqrs-lint Scan-without-Limit rule.
31. Routing integration v1 (scalar-covered shapes price O(1), route to matview engine) — after 29.
32. Watch turso-go defect A: when fixed upstream + flip-runbook gate passes, unblock grouped-shape AggregateOn routing.

**Older plan carry-forwards (unchanged):**
33. go-idempotency `Forever` adapter mapping (gated on upstream v0.4.0; write MaxInt64 directly, dedup `expiryFromTTL`, overflow boundary test).
34. Tag `system` so the coeffect gate reaches consumers (rides next tag wave).
35. benchkit cross-tier PARITY gate.
36. Tuned-tier metaengine benchmark (tuned-vs-tuned, no defaults asymmetry).
37. go-cqrs-lite READMEs: claimkit entries re-anchored after M22 lands.
38. CV consumer bump coordination (operator-side, excluded but tracked).
39. CI billing fix (owner/infra).
40. F040 branch protection (owner/infra).
41. Zenoh go/no-go (owner ruling pending).
42. G-T01 direction ruling (evidence pack → 3-option memo → owner).
43. ADR-0139 open questions.
44. Doctor-JSON semantics ruling.
45. Turso DSN/sync policies ruling.
46. dgraph one-RPC scope ruling.
47. Severity tightening Q3 release-policy ruling.
48. v5 train: all ADR-0123 deletion rows (gated, never in v4.x) — incl. eventual recipes.md stack-preset fence rewrite.
49. iroh P99 ruling (in M01 ratification set, unanswered).
50. 350-line policy ratification (unanswered).

## g) QUESTIONS (3 — cannot self-answer)

1. **M19 timing (carried Q1):** implement `FilterContains`/`FilterPrefix` now,
   or hold until the in-flight metaengine v4.14.0 / record v4.5.1 pin-tag wave
   from the queue session lands? Both touch the FilterOp surface and I can't see
   their branch plan.
2. **Actions access (carried Q2):** is `gh` permitted to read Actions runs/logs
   for M08's remote evidence (nightly 04:12 triage, TagContent clean-run
   confirm), or is that billing-gated? Local-only M08 is done; the remote half
   is blocked on this.
3. **The stray `metaengine/tursoengine/P\x11B` SQLite artifact:** a 4 KiB
   SQLite file with a control-character name appeared untracked at 14:42 during
   the concurrent turso session's work. Is it theirs and expected (leave it), or
   may I trash it once their run ends? I can't tell whether a live process
   still expects the file.

**THEN WAIT FOR INSTRUCTIONS.**
