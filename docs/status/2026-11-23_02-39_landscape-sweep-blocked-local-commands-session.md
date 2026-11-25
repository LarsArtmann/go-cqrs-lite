# Status — Local-Commands Design + Competition-Sweep Blockage Session

**Session scope:** 2026-11-21 22:46 → 2026-11-23 02:39. Covers: the LiveStore +
DataStar deep dive and its report, the owner's local-Commands addendum, the
D40/D41 decision-sheet work, the GopherJS follow-up correction, and the
blocked localfirst.fm competition sweep. Not a project-wide audit.

**Verified with commands this session:** HTML parse/anchor/placeholder checks
on both new reports; `rg` symbol verification for every cited go-cqrs-lite
symbol (PersistedCommand, CommandSink, DeriveCommandID, DedupStore,
ReadFrom — all confirmed at file:line); both-resolver DNS test (local
forwarder 5/5, direct 9.9.9.9 dead for Go processes); `date`; TODO_LIST row
counts. **NOT verified:** rendered layout of any HTML report (never
screenshotted); docs/competition/ contents (never created — subagent never
survived its first model call); the zai quota-bucket theory beyond observed
behavior (my flash-session passes, every full-glm-5.3 secondary spawn 429s —
consistent across ~6h, but the why is provider-side and unprovable from here).

**Artifacts:** `docs/research/2026-11-22_commands-vs-facts-livestore-datastar.html`
(1,811 lines, series #3) · TODO_LIST **D40** (renumbered) + **D41** (new) ·
research README index rows · `docs/research/2026-11-21_go-ts-frontend-contracts.html`
GopherJS correction (prior session's report, amended this session).

---

## a) FULLY DONE

1. **Three verified research legs** (agent fetch, exact-quote method): LiveStore
   deep (NO command concept — "command" hits are CLI-only; commit-callback sees
   pre-commit state; conflicts #253 still unimplemented, blocked on unbuilt
   facts system #254), DataStar deep (client v1.0.5 2026-11-09; protocol =
   exactly two SSE events; Pro-tier boundaries; "always implement backend
   validation"), and the client-mutations survey (Replicache/Zero/Electric ALL
   queue intents with server-authoritative re-execution).
2. **The Commands-vs-Facts report** — verified the owner's suspicion as
   architectural fact; placed LiveStore as the industry outlier on the
   who-mints-facts fault line; five-layer synthesis; D41 harvest.
3. **Owner's §10 addendum** (the local-Commands Question, verbatim):
   decomposed into P1–P5; the local-branch design (speculative events minted
   by client-side deciders, push Commands on reconnect, rebase intents not
   facts); the client-safe decider seam (stream-local = portable,
   cross-stream/protected = server-only, statically checkable).
4. **Owner's storage refinement wired in (§10.5)**: dual SQLite-WASM, WAL on
   OPFS in a dedicated/Shared worker — not IndexedDB — with the two technical
   sharpenings (WAL-on-:memory:-persists-nothing; service-worker-vs-worker
   choice); R1 of the report amended with a dated note; D41 row updated with
   the refined principle.
5. **GopherJS follow-up honesty fix**: the report said "not under active
   development" (✅-marked) — that phrase was not in the verified source
   (which says "not archived, but far behind"; June 2026 release contradicts
   it). Corrected in three places with dated notes; logged as status-report
   defect d6.
6. **Decision-sheet collision handling**: parallel sessions took D34–D39
   overnight; my front-end decision renumbered D34→**D40** (all references
   across 3 files updated, collision note in the row) and the new adapter/L4
   decision rowed as **D41**. Their rows untouched.
7. Both reports pass the pre-ship checklist (placeholders 0, anchors resolve,
   parse clean, zero external deps, symbols source-verified).

## b) PARTIALLY DONE

1. **The competition sweep (localfirst.fm/landscape → docs/competition/)**:
   dispatch prompt fully prepared and iteratively hardened (7 attempts), but
   the subagent NEVER completed a single model call — see (c)/(d). Zero files
   exist. All work remains with the subagent per your instruction.
2. **Provider routing matrix**: every GLM-5.3 route now mapped — zai (bucket
   exhausted for secondary sessions), fireworks + synthetic (DNS-dead during
   the ISP outage window; untested while healthy), hyper (no credits), flash
   (works, but is not the model you asked for). The map is complete; the
   execution is not.
3. **Landscape sizing**: my one fetch of localfirst.fm/landscape returned only
   the Astro hydration shell — the entry list lives behind client rendering or
   an embedded data blob nobody has extracted yet. Entry count still unknown.
4. **The GopherJS question** exposed a third misquote-as-verified instance;
   fixed, but the pattern (transcription drift from agent extraction into
   ✅-marked rows) now has three instances across two reports.

## c) NOT STARTED

1. docs/competition/ — all of it (per-entry pages, index.html, master matrix,
   synthesis). Blocked, not abandoned.
2. D40 ruling execution (contracts pipeline P0) — awaiting owner decision.
3. D41 rulings (adapter commission; L4 principle + §10.5 storage ratification)
   — awaiting owner decision.
4. The zod/validator spike, thin-client template, freshness-gate leg — all
   D40-gated P1/P2 work.
5. Render-screenshot verification of the three HTML reports.
6. ADR-0168 (contracts emission) — deferred to P0 by design.

## d) TOTALLY FUCKED UP

1. **Seven subagent spawn failures and I chased them one at a time.** The
   routing matrix (zai-bucket / DNS / credits / flash) only became explicit
   after the 4th–5th failure; a systematic route map on failure #1 would have
   saved hours and four of the seven attempts. The final diagnosis was also
   partially wrong until you corrected it: I built a host-resolv.conf theory;
   it's the ISP.
2. **The GopherJS misquote** — I wrote "not under active development" with a
   ✅ mark; the verified source says otherwise. Third instance of the
   write-what-the-source-doesn't-say class. Only caught because you asked a
   direct question about that exact row.
3. **The §10.5 edit swallowed §10.4's body paragraph** (replace instead of
   append) — self-caught within one verification pass, but it happened because
   I edited from memory of my own text instead of viewing it first. Same
   failure mode as the R1 old_string mismatch minutes earlier.
4. **`rg -rn` hook strikes, twice more** — the replace-flag trap the guard
   hook exists for; I re-ran the identical mistake form within minutes.
5. **TODO_LIST D34 numbering collision** — not caused by me (parallel
   sessions took D34–D39 overnight; the sheet has no allocation gate), but my
   D34 sat unrenumbered for hours and my first renumber attempt used a stale
   read. The sheet needs a next-free-number helper; until then every new row
   is a collision lottery.

## e) WHAT WE SHOULD IMPROVE

1. **Route-map first, then spawn**: before any long subagent task, probe each
   candidate route with a one-word call; kill nothing mid-flight. Would have
   collapsed seven failures into two.
2. **Never edit my own HTML from memory** — view the region, then edit; the
   §10.4 swallow and the R1 mismatch were both memory-vs-file drift.
3. **ISP-aware planning**: when external fetches start timing out, stop
   spawning network-dependent subagents entirely (the outage window wasted
   fireworks/synthetic attempts that were doomed regardless of quota).
4. **Transcription discipline for agent-extracted claims**: ✅-marks must
   quote, not paraphrase — three strikes now.
5. **Decision-sheet allocation helper** (next-free-DN check) — the D34–D41
   collision lottery will recur; a 5-line check in the harvest ritual or a
   gate script prevents it.
6. Keep what worked: verified-quote research legs, dated amendments over
   rewrites, symbol rg-verification before citing (caught DedupStore
   placement), the pre-ship checklist.

## f) Up to 50 next things

**Blocked-work resumption (ISP + quota dependent):**
1. When ISP stabilizes: retry GLM-5.3 via fireworks/synthetic for the
   competition sweep (prompt is ready; nothing to redo)
   — done 2026-11-23: ISP healed; zai glm-5.3 (default route) accepted
   subagent spawns after transient 429 retries; fireworks/hf were
   credit-dead, zai worked
2. If glm-5.3 bucket still 429s while network is healthy: your call — flash
   subagent vs wait (see §g1)
   — moot 2026-11-23: full glm-5.3 route worked; no flash fallback needed
3. Extract the landscape entry list (Astro data blob) and pin the total count
   — done 2026-11-23: 22 entries (Astro island props decoded, raw/landscape.json)
4. Per-entry pages for all entries → docs/competition/
   — done 2026-11-23: 22/22 pages, four sharded reviewer agents
5. index.html: master matrix + synthesis + prior-report cross-links
   — done 2026-11-23: parent-assembled from the shard verdicts
6. Index docs/competition/ in docs/research/README.md (or its own README)
   — done 2026-11-23: newest-first row at the top of the dated-deep-dive table
7. Landscape sweep feeds §10: close the "evidence base is in-flight" loop
   — done 2026-11-23: dated closure notes added to the 11-22 report (§10.4 +
   footer)

**D40-gated (contracts pipeline, awaiting ruling):**
8–13. P0 slice: catalog/v4/typescript exporter (promote renderer), goldens,
   api-stability update, package-surface registration, coverage floor,
   FEATURES/CHANGELOG rows
14. ADR-0168 draft with the P0
15. exportName collision audit across fleet message names
16. Deprecation → generated JSDoc @deprecated mapping
17. Catalog version stamp in generated file headers

**D41-gated (adapter + L4, awaiting ruling):**
18. Commission metaengine/datastaradapter P0 (~300 LOC, sibling-report plan)
19. Ratify the L4 principle incl. §10 refinement + §10.5 storage topology
20. Adopt upstream Datastar conformance corpus as adapter goldens

**L4 design tail (§10/§10.5, all gated):**
21. Client store spike: SQLite-WASM + WAL-on-OPFS in a SharedWorker
22. Client-safe decider seam enforcement (declare resource deps; cqrs-lint
    candidate rule)
23. Speculative-flag row semantics in client projections (DB2 schema sketch)
24. Rebase semantics doc: interactive rebase of intents; Rejection-as-conflict UX
25. Decider-logic version in the command envelope (skew control)
26. Dual-implementation contract-test pattern for shared decide/fold logic
    (scenario DSL both sides)
27. TS fold/decide codegen feasibility note (extends 11-21 pipeline)
28. TinyGo fold feasibility numbers (still parked, still gated)
29. IndexedDB-vs-OPFS evidence row for the report (LiveStore/Evolu verified;
    add PowerSync/SQLite-WASM specifics when network returns)

**Report hygiene (from this session's defects):**
30. Sweep both HTML reports for any remaining paraphrase-✅ rows (GopherJS
    class); quote-or-relabel each
31. Add the three misquote instances to verify-external-claims' incident list
32. HTML report pre-ship checklist as a script (placeholder grep + anchors +
    parse + symbol rg — the checks I ran by hand)
33. Decision-sheet next-free-number helper (D34–D41 lesson)
34. Add "route-map before spawn" to the subagent playbook notes
35. Screenshot-verify the three reports (never rendered-checked)

**Owner-decision queue (already rowed):**
36. D40: approve P0 / full pipeline / park
37. D41(a): commission adapter / fold into D40 P1 / park
38. D41(b): ratify L4 principle + refined §10 shape / defer
39. D35–D39 (parallel sessions' rows) — untouched, theirs to close

**Small sticks:**
40. README of docs/competition/ conventions (once it exists)
   — resolved-by-design 2026-11-23: the index's "Method & verification"
   section carries the conventions; no separate README file needed
41. LiveStore #253/#254 watch trigger → TODO_LIST row so convergence is a
   decision, not amnesia
   — done 2026-11-23: rowed in TODO_LIST "Harvest — 2026-11-23 competition
   sweep executed" (together with §f.42/43 and the blocked render-screenshot
   item)
42. Re-measure DataStar client size at v1.0.5 (sibling measured v1.0.4)
43. Check whether catalog-export cmd pattern can consume system.DomainConfig
    directly (declare-once bridge, currently asserted not proven)
44. Wire "command-queue" shape into the existing systemscenario harness docs
    (when L4 opens, its tests already have a home)
45. Vault the "facts system" (#254) design when LiveStore publishes it
46. Monitor Go blog for official Go→TS tooling (standing trigger from 11-21)
47. AsyncAPI-as-interchange note for third-party generator consumers
48. Confirm ServeSSE JSON batch overhead number for the perf budget wire row
49. Add ISP-flakiness note to gotchas-tooling (subagent spawns + fetch agents
    fail misleadingly during outages; diagnose route map, not code)
50. D40/D41 disposition rows closed either way so the sheet stays truth

## g) Questions (cannot figure out myself)

1. **Competition sweep under ISP reality**: when the network stabilizes, do I
   auto-resume with GLM-5.3 via fireworks/synthetic (my pick — honors your
   model demand), or fall back to glm-5.3-flash (zai bucket proven alive)?
   And if the flash fallback is needed at all, is that acceptable to you?
2. **§10.3 scope check**: I read "as long as they do not access other
   protected server resources" as *decider-scoped* — a decider that reads
   only its own stream is client-portable; cross-stream/PII/external reads
   make it server-only. Is that the intended line, or did you mean something
   narrower (e.g., only auth-gated queries)?
3. **D40/D41 rulings**: approve P0 of the contracts pipeline? Commission the
   DataStar adapter? Ratify the L4 principle (clients queue Commands AND may
   decide locally on a speculative branch; only server deciders mint
   authoritative events; dual SQLite-WASM storage) — now, or stay gated?

---

*Written 2026-11-23 02:39. (f) items are D40/D41-gated or ISP-blocked as
marked; D40/D41 already sit in the decision sheet so nothing is entombed.
**Now waiting for instructions.***
