# Competition Sweep Executed + 11-21 Report Repairs — Session Report

**Date:** 2026-11-23 (session ~04:00–05:30) · **Scope:** the blocked
localfirst.fm competition sweep (unblocked and completed) + the FIX-NOW
defect list from the 2026-11-21 contracts report. Not a project-wide audit.

## a) What was attempted and completed

1. **Competition sweep — DONE, 22/22 entries.** The ISP outage that killed
   seven spawn attempts had healed; `docs/competition/` now holds one verified
   review page per localfirst.fm landscape entry plus a parent-assembled
   `index.html` (master matrix, LOCAL COMMANDS census, verified-corrections
   table, adoptable-patterns kit, method section). Four sharded reviewer
   agents (maria resumed from its failed state + three fresh) worked a shared
   evidence cache (`/tmp/lf`: 109 scraped docs, `raw/landscape.json` with the
   22-entry dataset decoded from Astro island props) plus live 2026-11-23
   fetches (GitHub REST API, npm registry, vendor LICENSE files).
   Headline census: **17 of 22 CLIENT-MINTS-FACTS** (no Command concept; a
   Rejection is inexpressible); 3 Command-shaped (Zero
   COMMANDS-EXECUTED-LOCALLY, Ditto COMMANDS-EXECUTED-LOCALLY, PowerSync
   COMMANDS-QUEUED); 2 SERVER-ONLY-WRITES (Convex, ElectricSQL). None of the
   22 shares decision logic with the client — the client-safe decider seam
   of the 11-22 report §10 is unclaimed territory.
2. **11-21 report defects — both fixed with dated amendments** (never silent
   rewrites): the fabricated L0 sketch (`catalog.RegisterEvent(EventConfig{...})`)
   replaced by the real surface (`simple.New` + `simple.Command[T]` /
   `simple.Event[T]` + `Build()`, source-verified at
   `catalog/v4/simple/builder.go`); the UNVERIFIED tally made consistent
   (typescriptify verified live via GitHub API → scorecard corrected 3→2,
   valibot vendor-sizes row now explicitly UNVERIFIED-labeled, §10 prose
   amended with the recount).
3. **FIX-NOW items 3–5 from the 11-21 status report**: tygo v0.2.21
   re-verified (GitHub API, published 2026-02-03T10:17:07Z, latest); zod@4.6.0
   and @connectrpc/connect-web@2.3.0 re-verified on npm with extraction notes
   in the §10 rows; "Report-Integrity Checklist" section added to the
   html-report-kit guide (`SKILLS/html-report-kit/references/html-output-guide.md`)
   — the gate-hole (HTML reports bypass doc-check/md-go symbol honesty) now
   has a written ritual with this incident as the observed failure mode.
4. **Bookkeeping**: 11-21 status report §d1/d2 + FIX-NOW 1–5 annotated
   resolved (item 4 render-screenshot documented-blocked: firefox is
   hook-blocked in yolo mode — manual one-liner recorded); 11-23 status report
   §f.1–7 + 40/41 annotated resolved; TODO_LIST harvest section added
   (LiveStore convergence watch, render screenshots, DataStar v1.0.5
   re-measure, catalog-export DomainConfig proof); `docs/research/README.md`
   row added; the 11-22 report's "evidence base is in-flight" loop closed
   with dated notes (§10.4 + footer).

## b) What worked

- **Route empiricism over speculation.** §g1's resume-route question answered
  by trying: zai glm-5.3 (parent route) accepted subagent spawns after
  transient 429 retries; fireworks/huggingface were credit-dead (maria's
  resume surfaced "purchase pre-paid credits"); hyper credit-dead. No flash
  fallback was ever needed — the user's GLM-5.3 demand was satisfiable.
- **The shared evidence cache.** The failed overnight agents' `/tmp/lf`
  scratch (109 docs, entry lists, `show.py`, `landscape.json`) turned four
  fresh reviewers into cache-consumers, not re-scrapers. Zero re-derivation.
- **Shard ownership + parent verification.** Disjoint file scopes let four
  writers share one directory safely; the one collision (a scope-drifting
  worker wrote two out-of-shard pages) was resolved by the owning shard's
  rewrite, and the parent re-verified every file independently (parse,
  anchors, markers, external-resources=0, backlinks).
- **Dated-amendment discipline held** across both amended shipped reports.

## c) What didn't work (honest)

- **Rate-limit roulette.** Two workers died mid-run on zai 429s; one resume
  also 429'd instantly (a resume triggers a model call before any sleep
  instruction can run — staggering via prompt is useless). Both were healed
  by plain re-resume once the sibling's burst passed. Sequential-resume, not
  sleep-staggering, is the cure.
- **Scope drift.** The shard-C worker wrote two shard-B pages (zero, triplit)
  before dying; harmless here (the owner's rewrite won) but the same class as
  the file-collision warning — write-scope enforcement is prompt-level only.
- **Wasteful fetch.** The full zod packument (~100 KB, truncated) was fetched
  where the per-version endpoint (`/zod/4.6.0`, ~4 KB) had everything needed.
- **Render verification still missing.** No browser available to agents
  (firefox hook-blocked; no chromium on host). All 25 HTML files are
  parse/anchor/marker-verified but never rendered.

## d) Totally fucked up

1. **The index's method section tripped its own placeholder grep — twice.**
   Describing the check ("zero TODO/FIXME/lorem/TBD") inlined the exact
   tokens the check scans for; reworded twice before the corpus grep came
   back clean. A gate's documentation must not contain the gate's trigger
   strings.
2. **One edit-tool miss on a from-memory old_string** (11-21 status §d2:
   "number —" vs the file's "counted —"). View-first discipline caught it in
   one round trip, but it was a memory-drift mistake this session's own rules
   prohibit.
3. **The first status-annotation multiedit applied 6/7** with a
   whitespace-normalization notice — acceptable, but it means one annotation
   landed in re-indented form rather than verbatim; verified by grep after.

## e) What we should improve

- Per-version registry endpoints (`/pkg/<version>`) over full packuments for
  version-pin verification.
- Subagent route-map: when parallel workers share one provider, expect burst
  429s; resume sequentially rather than re-spawning, and never bother with
  sleep-in-prompt staggering (the resume call itself consumes quota).
- Write-scope assertion beyond prose: name the exact filenames in the shard
  brief AND in the resume message (this session did; the drift happened in a
  worker whose context predated its re-scope).
- The Report-Integrity Checklist now lives in the kit — apply it to THIS
  corpus's next refresh.

## f) Up to 50 next things

1. **D40 ruling** (contracts pipeline P0) — unchanged, owner decision
2. **D41(a) ruling** (DataStar adapter commissioning) — unchanged
3. **D41(b) ruling** (L4 principle incl. §10 refinement + §10.5 topology) —
   the sweep's census (17/22 client-mints-facts, 0 logic-sharing) is now
   direct evidence for the ruling
4. **§10.3 confirmation** (protected server resources = decider-scoped
   reading) — unchanged
5. Standing triggers, rowed 2026-11-23 in TODO_LIST: LiveStore convergence
   watch; DataStar v1.0.5 re-measure; catalog-export DomainConfig proof;
   manual render screenshots (firefox one-liner)
6. The 11-23 02-39 report's D40/D41-gated items 8–39 and conditional
   44–46 remain open there — nothing in this session changed their gating
7. Landscape refresh cadence question — see §g5

## g) Questions for the owner

1. ~~Resume route for the sweep~~ — answered empirically 2026-11-23: zai
   glm-5.3 works for subagents (bursty 429s, sequential resume heals);
   fireworks/hf and hyper are credit-dead. No action needed.
2. **D40**: approve / park the contracts pipeline P0?
3. **D41(a)/(b)**: commission the DataStar adapter; ratify the L4 principle
   (the sweep's census is new evidence — nobody in the landscape shares
   decision logic with the client).
4. **§10.3**: confirm "protected server resources" reads as
   decider-resource-declarations (stream-local ⇒ client-portable).
5. **New**: should the competition corpus carry a refresh trigger (e.g.
   re-verify any entry before citing it if older than a quarter, mirroring
   the research README's snapshot rule), or stay a one-shot 2026-11-23
   snapshot? Default if unanswered: the index already carries the
   freshness-horizon callout — treat older-than-a-quarter pages as
   snapshots.

---

*Written 2026-11-23 ~05:30. The sweep artifacts: `docs/competition/`
(22 pages + index, all parent-verified). The report repairs: dated amendments
inside the two shipped reports. The kit checklist: SKILLS repo,
`references/html-output-guide.md`. Auto-commit daemon owns the tree.*
