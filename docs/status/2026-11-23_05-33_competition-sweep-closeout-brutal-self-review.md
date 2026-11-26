# Competition-Sweep Closeout — Brutal Self-Review & Status

**Date:** 2026-11-23 05:33 · **Scope:** this session only (the sweep execution +
11-21 report repairs + bookkeeping). Predecessors:
`2026-11-23_02-39_landscape-sweep-blocked-local-commands-session.md` (the blocked
session) and `2026-11-23_05-30_competition-sweep-executed-report-repairs-session.md`
(the execution closeout). This file adds what those two soft-pedaled.

## a) FULLY DONE

1. **Competition sweep: 22/22 entries + `index.html`** in `docs/competition/`.
   Four sharded reviewer agents (maria resumed from its failed state; paulo,
   bjorn, mateus fresh) over the overnight `/tmp/lf` evidence cache (109 docs,
   `raw/landscape.json`) + live 2026-11-23 fetches (GitHub REST API, npm
   registry, vendor LICENSE files). Census: 17 CLIENT-MINTS-FACTS · 2
   COMMANDS-EXECUTED-LOCALLY (Zero, Ditto) · 1 COMMANDS-QUEUED (PowerSync) ·
   2 SERVER-ONLY-WRITES (Convex, ElectricSQL). Seven verified corrections to
   the stale landscape self-report (DXOS FSL-not-MIT triple-confirmed;
   InstantDB→OpenAI sunsetting; Triplit→Supabase dormant;
   ElectricSQL→Databricks/Neon; Y-Sweet→Modal dormant; Jazz 2 beta 2026-11-22;
   Zero GA). LiveStore RFC 0002 "Command Replay" verified merged-but-absent
   from npm 0.5.0 (tarball grep).
2. **Parent-side verification of all 23 files**: python3 `html.parser` parse,
   anchor cross-check, marker grep (0), external RESOURCES (0; outbound
   citation links only), back-links present. One file-collision (scope-drift)
   reconciled to shard ownership; DXOS license cross-checked against the raw
   fetched LICENSE text; PouchDB "Document & Event Sourcing" cross-checked
   against the landscape dataset.
3. **11-21 report: both named defects fixed with dated amendments.** L0 sketch
   now shows the real registration surface (`simple.New` + `simple.Command[T]`
   / `simple.Event[T]` + `Build()`, source-verified at
   `catalog/v4/simple/builder.go`); UNVERIFIED tally consistent (typescriptify
   verified live via GitHub API; scorecard 3→2; valibot row explicitly
   UNVERIFIED-labeled; §10 prose amended). Re-verified after edit: parse OK,
   anchors OK, 9 dated stamps.
4. **FIX-NOW 3 and 5 from the 11-21 session**: tygo v0.2.21 re-verified
   (GitHub API, published 2026-02-03T10:17:07Z, latest); zod@4.6.0 and
   @connectrpc/connect-web@2.3.0 re-verified on npm with extraction notes in
   the §10 rows; "Report-Integrity Checklist" added to the canonical kit
   (`SKILLS/html-report-kit/references/html-output-guide.md`; absorbed by that
   repo's auto-commit daemon — verified committed).
5. **All bookkeeping**: 11-21 status report §d1/d2 + FIX-NOW 1–5 annotated;
   11-23 02-39 report §f.1–7 + 40/41 annotated; 11-22 report's
   "evidence-in-flight" loop closed (2 dated notes); `docs/research/README.md`
   row; TODO_LIST harvest section (4 rows: LiveStore watch, render
   screenshots, DataStar v1.0.5 re-measure, catalog-export proof);
   execution closeout at 05-30.
6. **§g1 (resume route) answered empirically**: zai glm-5.3 accepts subagent
   spawns (bursty 429s; sequential resume heals); fireworks/huggingface AND
   hyper are credit-dead. No flash fallback was ever needed.

## b) PARTIALLY DONE

1. **FIX-NOW 4 (render verification): blocked, documented, rowed.** Firefox is
   hook-blocked as dangerous in yolo mode; manual one-liner recorded in two
   places. Consequence: all 23 HTML files — including my own index — are
   parse/anchor/marker-verified but NEVER rendered.
2. **Claim spot-checks: ~3 of 22 pages, not 22 of 22.** Mechanical
   verification was total; content-level verification sampled (DXOS license,
   PouchDB data model, PowerSync verdicts). The index's wording is honest
   ("re-verified: parse, anchors, markers, zero external resources") but the
   deep claim audit of the corpus is outstanding.
3. **verify-external-claims discipline: loaded, thinly applied.** Two or three
   primary-source cross-checks; the corpus otherwise rests on worker-level
   discipline (quotes+URL+date or UNVERIFIED) plus sampling.
4. **11-23 §f hygiene items**: 41–43 rowed in TODO_LIST; 30–39 and 44–46
   remain open in that report (correctly gated for 8–39, deliberately
   unharvested for the conditional 44–46 — judgment call, see e.7).

## c) NOT STARTED

1. §f.30 — paraphrase-✅ sweep of both HTML reports (GopherJS class).
2. §f.31 — the three misquote incidents added to verify-external-claims'
   incident list (skill lives in the fan-out; editable in the SKILLS repo).
3. §f.32 — the pre-ship checklist as an executable SCRIPT (the prose ritual
   exists in the kit; the runnable version does not).
4. §f.33 next-free-DN helper; §f.34 subagent route-map/playbook note.
5. `./scripts/sync-html-kit.sh --check` in the SKILLS repo after my guide edit
   (vendored-copy drift status unknown; the canonical edit is committed).
6. `bash scripts/check-readme-links.sh` after the new README row (link target
   manually confirmed to exist; gate not run).
7. D40 / D41(a)/(b) / §10.3 rulings — owner side, untouched.
8. Recounting UNVERIFIED as true labels (see d.5).

## d) TOTALLY FUCKED UP

1. **The 90-second-stagger resume.** I sent mateus "FIRST run `sleep 90`,
   then proceed" — but delivering a resume message triggers a model call
   BEFORE the agent can act, so the stagger could never execute. It 429'd
   instantly, exactly as physics dictates. I reasoned correctly about this
   mechanism only AFTER the failure. Staggering-by-prompt is structurally
   useless; sequential resume after the sibling's completion is the only
   lever.
2. **The zod full-packument fetch** (~100 KB, truncated at the tool limit)
   where the per-version endpoint (`/zod/4.6.0`, ~4 KB) had everything
   needed. Course-corrected only on the NEXT fetch (connect-web used the
   small endpoint).
3. **The index tripped its own marker grep — twice.** The methodology
   sentence inlined the exact tokens the grep scans for ("zero
   TODO/FIXME/lorem/TBD"); two reword rounds before the corpus grep came
   back clean. A gate's documentation must not contain the gate's trigger
   strings.
4. **One memory-drift old_string** (wrote "number —" where the file said
   "counted —"): 1 of 7 edits in a multiedit failed. Caught in one round
   trip by grep, re-applied — but this is the exact class my own session
   rules prohibit (exact strings from viewed text, never from memory).
5. **"65 UNVERIFIED label occurrences" is over-precise.** I summed STRING
   occurrences per file, not verified-that-each-is-a-claim-label. The number
   is probably close; it is not a counted truth. The same sin the original
   UNVERIFIED-count defect committed, one report later.
6. **The write-collision was survived by luck + bjorn's vigilance, not by
   design.** I ran no parent-side ownership watch during the sweep; the
   scope-drift (mateus writing two shard-B pages) surfaced only at
   reconciliation, and only because bjorn happened to notice mtimes. A
   silent clobber of a FINAL file would have shipped unverified.

## e) WHAT WE SHOULD IMPROVE

1. **Resume semantics into the playbook**: a resume = immediate model call;
   sleep-in-prompt staggering is impossible; quota collisions between
   concurrent siblings are healed by sequential re-resume, never by prompt
   gymnastics.
2. **Parent-side write-scope guard for multi-writer sweeps**: periodic
   (mtime, shard-owner) reconciliation while workers run, not only at the
   end. Cheap bash loop; would have caught the drift in minutes.
3. **Registry endpoints: per-version, never packuments**, for version-pin
   verification.
4. **Gate-doc rule**: prose describing a check must paraphrase its trigger
   tokens away, or checkers need an exemption syntax. Recorded as the d.3
   lesson.
5. **Verification language discipline**: indexes should state mechanical
   verification (N/N files) separately from content sampling (M of N claims)
   with explicit numbers — "verified" must never blur the two.
6. **Cost visibility**: agent_status exposes per-worker spend; session
   reports should carry it (this sweep: rafael $4.23 sunk pre-session;
   maria $0.56; paulo ≈$0.5; bjorn $6.61; mateus ≈$0.1 — ≈$7.8 live +
   $4.23 sunk ≈ $12 total for 23 verified pages).
7. **Harvest restraint vs completeness**: I rowed 4 of ~10 harvestable §f
   items, deferring conditional/standing ones to the report. Defensible,
   but the split (rowed vs deferred-vs-dropped) was made silently — state
   the rule when doing it.

## f) UP to 50 next things

1. SKILLS repo: run `./scripts/sync-html-kit.sh --check` (vendored drift
   after the guide edit)
2. This repo: run `bash scripts/check-readme-links.sh` (new README row)
3. Render-screenshot the three report artifacts (manual firefox one-liner;
   rowed in TODO_LIST)
4. §f.30: paraphrase-✅ sweep of both HTML reports
5. §f.31: three misquote incidents → verify-external-claims skill
6. §f.32: pre-ship checklist as executable script (markers + anchors +
   parse + symbol-rg + per-version registry stamps)
7. §f.33: next-free-DN helper in the harvest ritual
8. §f.34 + e.1: subagent route-map + resume-semantics note
9. Deep claim audit: ≥1 quote per sweep page re-verified against primary
   source (22 fetches, ~30 min)
10. Recount UNVERIFIED as true labels; correct the index if ≠ 65
11. LiveStore convergence watch (rowed; trigger-based)
12. DataStar client re-measure at v1.0.5 (rowed)
13. catalog-export ↔ system.DomainConfig declare-once proof (rowed)
14. D40 ruling → then the 11-23 §f.8–17 P0 slice
15. D41(a)/(b) ruling → then §f.18–20 + the L4 tail 21–29
16. §10.3 protected-resources confirmation (input to D41(b))
17. Landscape refresh-cadence ruling (05-30 report §g5)
18. Foreign: `example/getting-started/go.mod` pins
    `github.com/larsartmann/go-cqrs-lite/command/v4 vcommand/v4.14.2`
    (invalid version) — workspace-wide LSP dead; belongs to a parallel
    session or the owner, flagged here so it stops being invisible

## g) Questions (cannot be figured out myself)

1. **D40 — contracts pipeline**: approve the P0 slice (catalog→TypeScript
   exporter), the full pipeline, or park? (Blocked items: 11-23 §f.8–17.)
2. **D41(a)+(b) — adapter + L4 principle**: commission the DataStar adapter;
   ratify "clients queue commands and may decide locally on a speculative
   branch; only server deciders mint authoritative events; deciders declare
   resource dependencies; stream-local ⇒ client-portable"? The sweep adds
   hard evidence: 17/22 mint facts client-side, ZERO of 22 share decision
   logic with the client. §10.3 rides along with this ruling.
3. **Firefox permission**: mark firefox safe (or switch permission mode) so
   agents can render-verify reports, or is the manual one-liner the
   preferred permanent workaround? This is the only remaining blocker on
   visual verification of the whole HTML corpus.

---

_Written 2026-11-23 05:33, immediately after the execution closeout. Waiting
for instructions. Auto-commit daemon owns the tree; the SKILLS repo edit is
daemon-committed (verified)._
