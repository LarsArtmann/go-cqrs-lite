# Idempotency Placement & metaengine Dedup — Session Status Review

**Date:** 2026-10-11 07:28 (Sunday CEST)
**Scope:** A Q&A / investigation thread about why `idempotency/{kvstore,sqlstore}` exist as orphan modules, what they actually do, and whether they should be deleted in favor of `system` + `metaengine`. **Zero code changes. Zero committed artifacts.** Purely analysis that lived in chat.

**Headline:** I produced a correct end-verdict (idempotency's backends are pre-ADR-0142 fossils; the storage question is metaengine's; the right end-state is a `system`-exposed dedup facade over `metaengine.DedupStore`), but only *after* (a) shipping one factual error into a user-facing answer, (b) repeating an unverified sub-agent claim, and (c) defending the existing wart twice before the user pushed back. The session's biggest failure is that it produced no artifact: an ADR was proposed in the last message and never written.

---

## What I forgot / could have done better / can still improve

- **Forgot:** to run the governing gate before asserting `versions.json` was stale. The gate (`check-versions-manifest.sh`) *documents* that the manifest is tag truth including retired dirs — one command would have killed the wrong claim before it shipped.
- **Could have done better:** verify the sub-agent's load-bearing claims (two of its claims were false: "idempotency not in flake `testModules`", and the retired-versions.json framing). And I should have gone straight to diagnosis when the user pushed back, instead of defending the modules as "the seam."
- **Can still improve:** convert this analysis into the ADR + migration scope I already outlined, so the session ends with a durable decision instead of a chat transcript.

---

## a) FULLY DONE (verified this session)

1. **Answered "why is idempotency special / not v5-ready"** — `idempotency/` has **no parent module** (`ls idempotency/` → only `kvstore/`, `sqlstore/`); the parent `go.mod` was deleted (ADR-0128). `.agents/skills`/AGENTS.md confirm.
2. **Verified ADR-0065 status** — `docs/adr/0065-extract-idempotency-module.md` status = *"Accepted (partially executed — see Addendum)"*; core extracted to external `go-idempotency` (interface-only), backends deliberately kept in-repo "permanently."
3. **Verified ADR-0152 has ZERO mention of idempotency** (`grep -in idempoten docs/adr/0152*.md` → NONE). The v5 core-merge list omits it.
4. **Found the actual v5 target:** `core/v5` — a single consolidated module (`module github.com/larsartmann/go-cqrs-lite/core/v5`) containing `command, dedup, dispatcher, event, id, kv, metadata, query, record`. Idempotency is **not** in it; `core/v5/kv/README.md:105` points *back out* to `idempotency/kvstore`.
5. **Read both modules' source in full** — `kvstore/store.go` (KVBackend adapter, expiry-as-string-value, `SetIfAbsent` CAS, race retry) and `sqlstore/store.go` + `engine.go` (dialect SQL, `idempotency_keys` table, `NewFromEngine` ADR-0142 facade, MySQL `IF()`).
6. **Mapped the metaengine primitive:** `DedupStore` (`DedupCheckAndRecord(ctx, collection, key, ttl, now) (bool,error)`, `DedupSeen`, `DedupSweep`), `MapDedupStore` degraded runtime, per-engine conformance suites (`adttest.AssertDedupStore`), ADR-0142 docs.
7. **Established the interface mismatch:** `middleware.{Command,Event,Query}Idempotency` consume external `go-idempotency.Store` (`Seen/Record/CheckAndRecord/Close`, `ErrDuplicate`); metaengine's `DedupStore` is a different shape; **metaengine does not import `go-idempotency`** (not in `metaengine/go.mod`; ADR-0062 boundary).
8. **Verified `system` has no dedup wiring** — the only "dedup" hits are comment/idiom ("SupportsDedup pattern" at `system/checkpoint_engine.go:132`); no dedup surface.
9. **Found real fleet consumers** of `go-cqrs-lite/idempotency` (go.mod + import sites): InboxClean, DiscordSync, Zlota44, accountability-system, browser-history, github-local-sync, standard-bug-tracking-schema, games/KeyCountdown, games/SEC, SwettySwipperWeb.
10. **Differentiated the two trains' status:** `idempotency/kvstore` IS in `scripts/untagged-trains.txt` (KEEP-DRIVER, ghost verdict pending T02 owner sign-off); **`idempotency/sqlstore` is NOT** — it's a live tag train (`idempotency/sqlstore/v4.4.3`).
11. **Verified both are tested in CI** — `flake.nix:283-284` lists `idempotency/kvstore` + `idempotency/sqlstore` in `testModules`.
12. **Corrected my own false claim** re `versions.json` (see d1) in-chat during the verify pass.

---

## b) PARTIALLY DONE

1. **`versions.json` investigation** — I first called `"idempotency": "idempotency/v4.4.0"` (versions.json:36) "stale bookkeeping". On verify: the manifest is **tag truth, not tree truth** (script header + ADR-0152 untag marking); the parent tag `idempotency/v4.4.0` exists locally; `check-versions-manifest.sh --check` **passes (exit 0)**. The claim was wrong. Corrected in chat; the wrong version still shipped in my earlier answer.
2. **ADR index status drift** — found `docs/adr/README.md:169` and `docs/README.md:110` label ADR-0065 "Proposed" while the file says "Accepted (partially executed)". A repo-wide mismatch scan followed but used a broken extractor (see d4) → output unusable; abandoned.
3. **The ADR I proposed** (retire `idempotency/{kvstore,sqlstore}` → `system`-exposed dedup facade over `metaengine.DedupStore` + fleet migration) — scoped verbally in the final message, **not written**.
4. **"What do they do" answer** — complete re: key-dedup semantics and the no-payload caveat, but I asserted the response-replay overlap from the PapDashboard report *and* the sqlstore source; the source reading (no payload columns) is solid, the cross-repo framing was non-local.

---

## c) NOT STARTED (deliberately — no code was requested)

- Writing the ADR (`docs/adr/0NNN-...`).
- Any code, test, or doc edit — **no file was modified this session** (this report is the first write).
- Fleet migration scoping / codemod (`go-ecosystem-upgrade` / `cmd/cqrs-upgrade`).
- A `system` dedup-surface design (constructor, DeploymentConfig wiring, health).
- A `DEFAULT` decision on whether the metaengine `DedupStore` *shape* itself should change (bool vs sentinel; drop `collection`/`now`).
- `idempotency/kvstore` ghost-verdict owner sign-off (still pending per ADR-0152 addendum / ghost-verdict memo).
- Recording anything in `TODO_LIST.md` / `DECISION-QUEUE` / `FEATURES.md`.

---

## d) TOTALLY FUCKED UP

1. **Shipped a factual error into a user-facing answer.** I called the `versions.json` parent entry "stale bookkeeping" and cited it as one of "two open items." It is correct, intentional tag truth. I never ran the governing gate before asserting — the script *documents* the exact behavior I misread.
2. **Repeated unverified sub-agent claims.** The sub-agent said (i) versions.json had a retired entry and (ii) idempotency is "not in flake `testModules`". (ii) is **false** (`flake.nix:283-284`). I caught both only in the "execute and verify" pass — the earlier answers rested on them.
3. **Defended the wart instead of diagnosing it.** I framed `{kvstore,sqlstore}` as "the interface seam, not redundant" — technically true but it dodged the real point (they're pre-ADR-0142 fossils, orphaned in the wrong place). The user had to push back twice before I reframed to the correct target.
4. **Ran a repo-wide scan with a broken parser.** My ADR status-mismatch loop extracted the file `Status` field only for the `| Status |` table format, missing the `**Status:**` prose format used by most ADRs → ~130 bogus "MISMATCH … file=[]" lines. Noise that could have misled a reader.
5. **An entire analysis session with zero artifacts.** No ADR, no decision, no TODO entry, no doc fix. The user's "Is that all?" was, effectively, answered "yes — analysis only." That is the session's defining failure.

---

## e) WHAT WE SHOULD IMPROVE (structural)

1. **Verify sub-agent claims on load-bearing facts** before repeating them; spot-check the specific assertions, not the narrative.
2. **Prefer source over summary** for any "does X ship / contain Y" claim (READMEs and agent briefs both overreached this session).
3. **Run the governing gate before asserting drift** — `check-versions-manifest.sh` would have caught d1 in one command.
4. **Switch to diagnosis mode the moment a user pushes back on an architectural defense** — stop rationalizing, name the smell.
5. **Validate scan scripts against a known-good sample** before trusting their (repo-wide) output.
6. **End every analysis session with an artifact** — ADR, `DECISION-QUEUE` entry, or `TODO_LIST` item. A chat verdict is not a decision.
7. **Note that the special-casing exists at TWO layers** — the orphan modules AND metaengine's bolt-on `DedupStore` capability interface. A fix that only moves modules doesn't fix the shape.

---

## f) NEXT — UP TO 50

**Artifact / decision capture (do first — kills the d5 failure)**
1. Write the ADR: "Retire `idempotency/{kvstore,sqlstore}` in favor of a `system`-exposed dedup facade over `metaengine.DedupStore`." Run `check-adr-numbering` before AND after.
2. Add a `DECISION-QUEUE` entry summarizing the verdict + this report.
3. Harvest this report's §f into `TODO_LIST.md` (docs-health HARVEST mode) before the session ends.
4. Add an ADR-0152 addendum row for `idempotency/kvstore` (ghost-verdict KEEP-DRIVER still pending owner sign-off).

**Design: the `system` + `metaengine` dedup surface**
5. Design `system`'s dedup surface: how a deployment engine's `DedupStore` is exposed (constructor arg? `DeploymentConfig` field? accessor?).
6. Decide the canonical server-side shape: keep external `idempotency.Store` (`Seen/Record/CheckAndRecord`) or align on `DedupStore`?
7. Resolve the `collection`/`now` extra args — should the `system` facade default them, or should `DedupStore` shrink?
8. Decide `DedupCheckAndRecord` return: keep `bool`, or add an `ErrAlreadySeen`/`ErrDuplicate` sentinel for middleware symmetry?
9. Define the non-engine fallback for system users (in-memory `MapDedupStore` semantics).
10. Decide where the single `idempotency.Store` ↔ `DedupStore` adapter lives (system, metaengine, or the external lib).
11. Specify health/quarantine interaction (ADR-0137) for a dedup facade over a quarantined engine.
12. Specify reset/catch-up interaction (ADR-0136/0143): does a dedup window survive engine reset? (Claim rows are data, not journal facts.)

**Migration (only after the ADR is accepted)**
13. Enumerate exact fleet consumers per module (kvstore vs sqlstore vs old `idempotency/v4`).
14. Audit each consumer: do they have a `*sql.DB`, a `kv.Store`, or a metaengine `Engine`? Determines migration path.
15. Prototype the codemod (`cmd/cqrs-upgrade`) replacing `sqlstore.NewSQLiteStore(db)` with the `system`/engine path.
16. Decide the deprecation ladder (deprecation notice → v5 removal → deleted) for the two modules.
17. Keep `versions.json` tags intact during deprecation (tag truth) — confirm no edit needed.
18. Update `scripts/untagged-trains.txt` if `sqlstore` gets untagged.
19. Verify `idempotency/kvstore` KEEP-DRIVER verdict is reconfirmed or reversed by the ADR.
20. Define the "last tag" story for python-proxy consumers of the retired modules.

**Docs / skill**
21. Fix the go-cqrs-lite skill's "Dedup: Which store?" table (`SKILL.md:147-153`) to lead with the `system`/metaengine path once it exists.
22. Update `references/modules.md` / `core.md` idempotency rows.
23. Update `FEATURES.md` (`:162-178`, `:1493-1494`) once the direction is fixed.
24. Reconcile `ROADMAP.md:269-271` idempotency-extraction entry (already shipped/partial).
25. Reconcile the stale ADR-0065 index status rows (`docs/adr/README.md:169`, `docs/README.md:110`) — **only if** the index is meant to mirror file status (see g-question).
26. Cross-link the new ADR from AGENTS.md "External extracted modules" bullet.

**Verification of this session's findings (pin them)**
27. Add/run a claim that `idempotency/{kvstore,sqlstore}` are in `testModules` (so the sub-agent's false claim can't recur).
28. Note the `versions.json` "tag truth, not tree truth" rule in the gotchas index so it stops being re-flagged.
29. Fix or delete the broken ADR status-mismatch one-liner; if kept, add it to a script with a `--self-test`.
30. Record the `core/v5` module-content census (idempotency absent) as a checkable fact.

**metaengine `DedupStore` shape (the deeper special case)**
31. Evaluate whether `DedupStore` should join `Engine` as a first-class surface rather than a capability interface.
32. Evaluate `Store`-level `Claim(key, ttl)` as the ergonomic entry (planner/`Store` owning it).
33. Check `adttest.AssertDedupStore` coverage across all engines (memory/sqlite/turso/pebble/bbolt/badger/duckdb/pg/mysql/dgraph/iroh).
34. Document the degraded-vs-native dedup matrix in `docs/engine-capabilities.md` (generated).

**Supply chain**
35. Confirm `go-idempotency` v0.4.0 status (still unshipped — latest tag v0.3.1) gates the `Forever`/MaxInt64 TODO item.
36. Keep the `Forever` adapter mapping item parked until upstream lands (TODO_LIST:1438).

**Process / hygiene**
37. Adopt a rule: analysis sessions end with an artifact.
38. Add "run the governing gate before asserting drift" to the session checklist.
39. Add "verify agent claims" to the sub-agent context template.

**Optional / nice-to-have**
40. Sketch the ideal end-state diagram (`system.New` → dedup facade → `metaengine.DedupStore` → engine picked at deployment).
41. Decide whether `middleware.*Idempotency` should accept the metaengine shape directly.
42. Consider collapsing `idempotency/kvstore` into `core/v5/kv` (KV is already there).
43. Consider whether `core/v5` should host an idempotency façade in the v5 cut.
44. Re-evaluate ADR-0065's "cannot be deprecated" text once ADR-0142 makes engines the substrate.
45. Add the PapDashboard cross-repo note (sqlstore ≠ response replay) to the ADR.

---

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **Is the ADR index `Status` column meant to mirror each ADR file's `Status` field, or is it an independent annotation?** The gate (`check-adr-numbering.sh`) enforces only title==H1, not status. If it should mirror, ADR-0065's two index rows are drift worth fixing (and likely others). If it's independent/coarse, I should leave it. I cannot infer the convention from the code.

2. **Do you want the ADR written now as the session artifact** (retire `idempotency/{kvstore,sqlstore}` → `system` dedup facade over `metaengine.DedupStore`, with fleet-migration scope), or do you want to rule on the direction first before anything is committed?

3. **Should metaengine's `DedupStore` interface itself be reshaped as part of this** (drop `collection`/`now`, sentinel return), or is the capability-interface shape frozen and only the *placement* of the backends is in scope? That determines whether this is a module-move or an API change.

---

**Status: analysis complete, NO code or docs changed (this report is the sole write). Awaiting instruction.**
