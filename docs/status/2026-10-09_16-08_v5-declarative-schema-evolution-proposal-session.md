# Status Report — V5 Declarative Schema Evolution Proposal Session

- **Date:** 2026-10-09 16:08 CEST
- **Session scope:** the evolvable-schema-for-v5 research + design proposal (Axon / LiveStore / Equinox / Akka), its verification pass, fleet-evidence pass, and the system/metaengine integration correction. NOT a whole-project audit.
- **Canonical artifacts:**
  - Proposal: [`docs/planning/2026-10-09_v5-declarative-schema-evolution.md`](../planning/2026-10-09_v5-declarative-schema-evolution.md) (256 lines)
  - TODO section: [`TODO_LIST.md` §"V5 declarative schema evolution"](../../TODO_LIST.md)
- **Concurrent context:** another session landed the `systemscenario` BDD harness (ADR-0153) and began a `core/v5/*` module wave DURING this session — two interactions with my work, noted in (d).

---

## a) FULLY DONE

1. **External research, primary-source verified.** Axon AF4 upcasting (raw `AxonIQ/reference-guide` `event-versioning.md`), AxonIQ Framework 5.2+ `EventTransformation` (Java source via GitHub blob), LiveStore migrations internals (raw `migrations.ts` + `state-tables.ts` — incl. `__livestore_schema`, `__livestore_schema_event_defs`, `__livestore_rebuild` marker), Equinox (raw `README.md`@master), Akka `EventAdapter` (source via Sourcegraph). Every load-bearing claim is tabulated with source + status in the doc's **Verification status** block; `RevisionSnapshotFilter` explicitly labeled ⚠️ unverified-lead (Sourcegraph found no confirmation; T4 does not depend on it).
2. **Repo current-state study with file:line evidence.** Five-mechanism gap table (`record/record.go:104-107`, `schema/upcaster.go:7-11`, `catalog/message_config.go:89-90,214-245`, `metaengine/relayout.go:64,171`, `snapshot/` absence), incl. the int-vs-semver split-brain between `Record.SchemaVersion` and catalog's `"1.0.0"`.
3. **Fleet evidence gathered (the proposal's load-bearing proof).** bank-sync: ~45-line upcaster closure (`internal/cqrs/upcasting.go:29-72`) + hand-rolled `NewFieldRenameUpcaster` + `migrate-journal` runbook + snapshot usage on `BalanceSyncState`. DiscordSync: 211-line `internal/eventschema/upcasters.go` (two type renames + derived-field upcasts). cqrs-htmx surfaces `SchemaVersion` (`sync_pull.go:282`), zero upcasters. go-appkit: untouched. Conclusion: T2 demand is proven by a consumer that built the layer locally.
4. **The proposal document itself** — adopt/reject matrices for all four systems with repo-specific rationale; two-axis model (payload evolution vs read-model evolution); T1–T7 increments with blast radius, breaking-change classification, and test plans; worked before/after example (bank-sync BalanceUpdated); failure-mode analysis (encryption/signing, rolling deploys, rename aliasing, hash mechanics, rebuild atomicity, ADR-0137 reprobe interplay, snapshots-vs-upcasting); version-identity ruling (keep wire int, derive semver); recommendations for all three open questions; ADR-0123 integration rule.
5. **system/metaengine integration verified in code and folded in.** `system.DomainConfig` already declares `Events`/`Evolutions`/`Projections`/decoders (`system/config_types.go:19-114`) → T1 rewritten to EXTEND `DomainConfig.Schema` instead of adding a third registry. Verified `system/` + `metaengine/` have ZERO upcaster wiring → new integration rule: upcasters declared via DomainConfig are applied inside system's adapter layer (`system/adapter_event_journal.go:16`). T5 fingerprints hash the metaengine `LayoutPlan`, checked at `system.New` boot.
6. **Gate verification, evidence:** doc-check on the proposal + canonical docs: `✓ All 1218 references valid` (my doc contributes zero warnings); `check-md-go`: `✅ no new errors (105 baselined)`. TODO_LIST section + index row current through all three upgrades.
7. **Two on-sight gate fixes (not mine, fixed anyway):** ADR-0153's pseudo-code fence was failing the md-go gate tree-wide → added `// skip-validate` (author content preserved); readmodels.md `kv`-alias ambiguity → added the scoping import.

## b) PARTIALLY DONE

1. **Proposal acceptance** — done: recommendations ready (doc §12). Missing: owner ruling on all three (declaration home confirmed-layers answer, T3 stamp staging, warn-first promotion criterion). Effort: S (one decision sitting). Blocker: owner.
2. **Doc-check zero-warning state for the whole scan set** — my artifacts are clean, but the concurrent `core/v5/kv` module collides with every un-imported `kv` alias in skill references (recipes.md:65 + more). I fixed one instance (readmodels.md); the cascade belongs to the active v5-wave session. Effort: S-M per file. Blocker: avoid racing a live session mid-wave.
3. **Equinox depth** — README-level verification only; `DOCUMENTATION.md` (AccessStrategy internals) not read. The doc claims only what the README proves; deeper Equinox material would sharpen, not change, the verdicts. Effort: S.
4. **Companion migration inventory** — sketched that DiscordSync/bank-sync upcasters collapse to `RenameType`/`Transform` ops; a precise per-upcaster mapping table (which of their ~10 upcasters → which op) was not written. Effort: S.

## c) NOT STARTED

All implementation. Zero proposal code exists; every item below is blocked on the ruling (or is prep work that could start unblocked):

1. **T2 `schema/ops.go`** — named upcast ops + build-time chain validation + `OnDecodeError` policy. Additive, v4.x-shippable, unblocked even pre-ruling.
2. **T1 `schema/declaration.go` + `system/config_types.go` Schema field + adapter application + catalog bridge + cqrs-lint rule.**
3. **T4 snapshot state-shape stamp** (`snapshot/` + `decider/` path + conformance test).
4. **T3 payload fingerprint ledger** (`record/` helper + `schema/` check + write-path options).
5. **T5 layout fingerprints** (`metaengine/` store + boot check + sqliteengine reference impl + `system` boot hook).
6. **T6 compat-policy lint rule + docs.** **T7 ADR split + recipes + core.md conventions.**
7. **CHANGELOG entries** — nothing to log yet (docs-only session; symbol gate N/A).

## d) TOTALLY FUCKED UP

Radical honesty; all caught and fixed in-session, but they were real failures:

1. **v1 report encoded agent-summarized claims as verified fact.** The first doc cited Axon/LiveStore mechanics from a summarizer's rendering — exactly the fabrication class `verify-external-claims` exists to prevent (I even had that skill available and did not load it until challenged). Severity: would have shipped subtly-wrong API names into a design doc. Fix: full re-verification pass, verification-status table, one claim demoted to ⚠️ unverified.
2. **v1 proposed a THIRD declaration registry beside `system.DomainConfig`.** The repo kills split-brains on sight; my `schema.Declare` would have created one (catalog's versions, system's `Events`, AND mine). Caught only by the owner's direct challenge — I had read ADR-0123's rule and still missed applying it to my own design. Fix: integration rule section; T1 rewritten to extend DomainConfig.
3. **Two self-inflicted gate trips** from proposal fences using the real `schema.` qualifier for not-yet-existing symbols (doc-check false references) and an un-scoped ambiguous alias. Process fix encoded: proposal fences start with `v5schema`-style qualifier + `// skip-validate`.
4. **Two "file modified since last read" edit failures** — the auto-commit daemon plus a concurrent session were landing commits mid-edit; I should re-read before every edit batch in an active tree, not only after failures.
5. **Wasted research cycles:** agentic_fetch rate-limited twice; AxonIQ docs SPA returned nav-only twice; `eqnx.li` DNS dead locally. Should have gone to raw GitHub sources on the first failure instead of retrying the friendly path.

## e) WHAT WE SHOULD IMPROVE

1. **Load `verify-external-claims` BEFORE first encode of any external research** — this session proves the trigger fires too late by default. Cost of the miss: one full re-verification pass (~40 min). Candidate: a checklist line in the go-cqrs-lite skill's contributing section.
2. **Standing design question for this repo: "does my proposal add a parallel declaration?"** — `system.DomainConfig` (ADR-0123) + `catalog` + `errorfamily` docs already own several declaration surfaces. A one-line grep ritual (`rg "type DomainConfig|type.*Config struct" system/ catalog/`) before designing any registry would have caught (d)2 in minutes.
3. **Proposal-fence convention** (qualifier + skip-validate from the first draft) is now learned — belongs in `docs/agents/gotchas-tooling-build.md` next to the existing doc-check notes.
4. **The `kv`-alias ambiguity sweep** will hit every future doc edit until the v5-wave session either renames `core/v5/kv` or the skill fences all import it. Whoever finishes that wave should sweep `.agents/skills/go-cqrs-lite/references/`.
5. **Fleet-evidence gathering belongs in the FIRST pass** of any library design work here — the bank-sync/DiscordSync findings changed the proposal's ordering (T2 first) and three design details (Transform first-class, error policy, §11 boundary). It cost ~10 minutes; it should be step 2, not step 9.

## f) Next tasks (up to 50, ranked)

Ruling-gated (top 3 = the unblock):

1. **Rule on the proposal** — accept/reject T-sequence (T2→T1→T4→T3→T5→T6/T7 recommended). Impact: Critical. Effort: S. Decision.
2. **Rule: T1/T3 in v4.x-additive or v5-wave-only** (ADR-0152 dual-support choice). Impact: Critical. Effort: S. Decision.
3. **Rule: companions co-release?** — ship ops + declaration WITH cqrs-htmx/bank-sync/DiscordSync migration in one wave (per ADR-0152 co-release model). Impact: Critical. Effort: S. Decision.

T2 (unblocked even pre-ruling; additive):
4. `schema/ops.go`: `RenameType`, `RenameField`, `AddField`, `RemoveField`, `Transform`, `Split`, `Drop`. Impact: High. Effort: M. Feature.
5. Build-time chain validation (duplicate exact (type,version) rejected; version-gap detection; most-specific-first, order-independent matching). High. M. Feature.
6. `OnDecodeError` Fail|Passthrough|Drop policy per op (default Fail). High. S. Feature.
7. Encoding-aware field ops (decode via stamped `Record.Encoding`, transform, re-encode; CBOR + JSON paths). High. M. Feature.
8. Table-driven tests incl. golden before/after payloads per op. High. M. Quality.
9. api-stability golden regen + `schema/README.md` update. Medium. S. Documentation.
10. bank-sync pilot: collapse `NewBalanceUpdatedV1ToV2Upcaster` + `NewFieldRenameUpcaster` onto library ops (proof of demand → proof of value). High. M. Feature.

T1:
11. `schema.EventDef[T]` + `Declare(...)` builder (name + current version + codec + type + chain). High. M. Feature.
12. `system.DomainConfig.Schema` field + derive `Events` (coeffect gate behavior unchanged). High. M. Feature.
13. Apply declared upcasters inside system adapters (`EventAdapter.ReadFrom` + decider load path). High. M. Feature.
14. `projectionadapter.TypeDecoder` registration derivable from `Schema`. Medium. M. Feature.
15. `catalog` bridge: render declaration for EventCatalog export (kills the versions split-brain). Medium. M. Feature.
16. cqrs-lint undeclared-event rule (symmetric to E018). Medium. M. Feature.
17. Version-grammar mapping: `WithVersion("2.0.0")` ↔ int current-version. Medium. S. Feature.
18. System integration tests: upcasting through `system.New` end-to-end (journal read + decider + projection host). High. M. Quality.
19. cqrs-htmx pilot: replace its SchemaVersion passthrough assumptions with declaration-backed validation. Medium. M. Feature.

T4 (small, independently correct):
20. Snapshot envelope state-shape stamp field (absent = accept, present+mismatch = discard+count). High. S. Feature.
21. `decider/` snapshot path honors the stamp. High. S. Feature.
22. Conformance test: stale snapshot discarded, aggregate rebuilt from journal, counter increments. High. S. Quality.
23. bank-sync validation: `BalanceSyncState` snapshots survive fold changes correctly. Medium. S. Quality.

T3:
24. Payload-shape fingerprint helper in `record/` (hash of DECLARED shape, canonicalized). Medium. M. Feature.
25. Write-side stamp via event options (metadata staging). Medium. S. Feature.
26. Read-side drift warning vs declaration (advisory, opt-in hard mode). Medium. M. Feature.
27. Tests: rotation of encryption keys does NOT trip the ledger (shape ≠ bytes). Medium. S. Quality.
28. Decision checkpoint: promote stamp to first-class `Record` field at v5 or not (burn-in data). Medium. S. Decision.

T5:
29. LayoutPlan fingerprint function in `metaengine` (hash of `BuildLayoutPlanFromType` output). High. M. Feature.
30. Engine-side persisted fingerprint storage (cleared on reset; journal exempt — ADR-0143). High. M. Feature.
31. Boot-time declared-vs-persisted diff → `LayoutDiff` → `RebuildThreshold`/`ConfirmRebuild`. High. L. Feature.
32. Completed-replay marker row (crash mid-rebuild never serves half-state). High. M. Feature.
33. sqliteengine reference implementation. Medium. M. Feature.
34. Engine conformance suite extension (all engines). Medium. L. Quality.
35. `system.New` boot hook + Doctor/health surfacing. Medium. M. Feature.
36. Rename-alias identity tests (`RenameType` mid-flight: both names = one identity for lint + coeffects). High. S. Quality.

T6/T7 + docs:
37. Compat-policy cqrs-lint rule (add-default safe / remove safe / rename needs op / remove-def never). Medium. M. Feature.
38. ADR split on acceptance (declaration/ops; stamping; boot gate). Medium. S. Documentation.
39. recipes.md section + core.md §3 conventions entry. Medium. S. Documentation.
40. SKILL.md reference updates (schema module row gains ops/declaration). Medium. S. Documentation.
41. Worked-example expansion: DiscordSync rename pair → `RenameType` one-liners in the doc. Low. S. Documentation.

Hygiene / session fallout:
42. `kv`-alias ambiguity sweep across skill references once the v5 wave settles. Medium. S. Cleanup.
43. Add "proposal-fence convention" (illustrative qualifier + skip-validate) to `docs/agents/gotchas-tooling-build.md`. Medium. S. Documentation.
44. Add "parallel-declaration check" ritual (grep DomainConfig surfaces first) to AGENTS.md contributing notes or the skill. Medium. S. Documentation.
45. Verify `RevisionSnapshotFilter` once Axon apidocs are reachable (or drop the lead from the doc). Low. S. Documentation.
46. Optional: Equinox `DOCUMENTATION.md` AccessStrategy read to sharpen the T4 notes. Low. S. Documentation.
47. Optional: baseline md-go sweep of `docs/planning/archived/` ghosts (105 entries — shrink opportunities). Low. M. Cleanup.

(HARVEST note: items 1–3 are the TODO section's existing ruling row; 4–40 map onto the existing T-rows; 42–47 are new — TODO_LIST already carries the T-rows, so no separate harvest was run this session per the user's "report only, then wait" instruction.)

## g) Questions I cannot answer myself

1. **The ruling:** do you accept the proposal and its T2-first sequence (or do you want the declaration T1 first, accepting that ops then land into an empty registry)? I cannot pick: both orderings are defensible; only your risk appetite for additive-surface-first vs declaration-first decides.
2. **v4.x-additive vs v5-wave:** should T1's `DomainConfig.Schema` and T3's metadata stamp land NOW on v4 module paths, or ride the ADR-0152 v5 wave with the companions? This is a product/release-train decision (tag-wave cost vs. earlier fleet value), not derivable from code.
3. **Companion migration appetite:** when the ops land, do you want bank-sync + DiscordSync + cqrs-htmx migrated onto them in the same wave (co-release model), or left to migrate opportunistically? Determines whether item 10/19 are in-scope for the first cut or follow-ups.

---

_Report format: `.md` at the user's explicit path demand — overrides the status-report skill's HTML-canonical default (flagged per skill contract)._
