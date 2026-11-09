# Status: irohengine ← dnsblockd-cluster Extraction-Fit Ruling Session (2026-11-09 07:53)

> **Point-in-time snapshot.** Second segment of the irohengine arc (after
> `2026-11-08_16-35_irohengine-testing-ladder-dnsblockd-fit-session`). Read-only again:
> the deliverable is a design recommendation, not code. Covers ONLY the extraction-fit
> question and this morning's tree observations; overnight sessions' work (ADR-0165 arc,
> 03-40 triage) is noted as context, NOT researched (owner instruction).

## Session metadata

| Field | Value |
| --- | --- |
| Date | 2026-11-09, 07:53 CET |
| Trigger | Owner: "Should we extract something from dnsblockd clustering and put it into metaengine/irohengine/??" |
| Scope | Fit analysis only: mesh-demo imports, irohengine symbol sweep, dnsblockd `internal/cluster` size/dep/test audit (read-only) |
| Tree state at close | Two session reports added (this + 16-35); zero code changes; overnight sessions active on the same repo (ADR-0165 arc) |

## What was asked and answered

**Q: Extract dnsblockd clustering into irohengine?** Answered: **No — park behind triggers.**

Evidence gathered this segment (all verified by direct read/grep, no assumptions):

1. **mesh-demo is NOT an iroh consumer** — `example/mesh-demo/go.mod` imports
   catalog/command/decider/event/id/system only. The "zero fleet consumers" premise of the
   ghost-engine census stands; nothing in the fleet uses irohengine today.
2. **irohengine has zero gossip/anti-entropy/membership symbols** — production sweep clean;
   the documented gap list (anti-entropy, late-join catch-up, membership, tombstone GC) is real.
3. **dnsblockd `internal/cluster` = 4,389 LOC** across node/crdt/orset/exchange/manifest/wire/ids
   + chaos/bench/property tests — a substantial extraction surface, shipped and chaos-proven
   where it is.
4. **dnsblockd cluster imports NOTHING internal AND nothing go-cqrs-lite** — the depguard-LEAF
   discipline extends cross-repo; extraction would be a one-way move of proven code into a
   103-module wave/golden/budget regime.
5. **dnsblockd cluster tests are plain-`testing`** (no gomega/ginkgo hits) — IF extraction ever
   proceeds, the pure-core slice + its tests are portable without pulling test-framework deps
   into the library.

**Recommendation delivered (one alternative, dismissed):**

- **No wholesale extraction now.** Nobody is served: dnsblockd owns the 4.4K LOC where it is;
  irohengine has zero consumers; layering says membership/gossip/PSK auth is fabric, not
  storage (ADR-0142 contract 30 keeps transport at its delivery layer); deployment models are
  irreconcilable (LAN-shaped Node: 32-member cap, datagrams, configured peers vs iroh's
  WAN/NAT-roaming value proposition). Permanent library cost (api goldens, dep budgets,
  conformance suites, wave mechanics) for a hypothetical consumer = the premature-generalization
  anti-pattern.
- **The gated future shape:** ONE shared pure replication core (OR-Set, LWW map + OpTag
  ordering, origin-sequence watermarks, digest/anti-entropy diff, tombstone GC policy) consumed
  by BOTH `irohengine.Replicated` (injected convergence protocol) and dnsblockd's Node;
  membership stays per-app. This is precisely irohengine's gap list, and dnsblockd's primitives
  are deliberately pure + clock-injectable — extraction-friendly by design.
- **Trigger conditions (either fires → extract):** (a) irohengine gets its first real consumer
  with a convergence requirement; (b) dnsblockd hits its own documented revisit triggers
  (WAN span / >8 nodes / NAT traversal / dynamic membership).
- **Known split brain to resolve before any merge:** dnsblockd `OpTag` (wall-clock first,
  node-id tiebreak) vs irohengine `WithClock` LWW — near-twin LWW semantics currently living in
  two repos with two orderings.

## a) FULLY DONE

1. mesh-demo consumer check (go.mod read; iroh absence verified) — closes the "zero consumers"
   premise with a receipt instead of an assumption.
2. irohengine production-symbol sweep for gossip/anti-entropy/membership (zero hits) — the gap
   list is now grep-proven, not doc-cited.
3. dnsblockd cluster audit: LOC (4,389), internal deps (none), go-cqrs-lite deps (none), test
   framework (plain testing) — the extraction-surface facts.
4. Layering analysis against ADR-0142 contract 30 (fabric vs storage boundary) and the
   deployment-model mismatch — the architectural core of the recommendation.
5. Extraction-candidate inventory: the pure (a)+(b)+(d) slice (anti-entropy, catch-up, GC
   policy) = engine-tier; membership (c) = app-tier; with trigger conditions and the OpTag/
   WithClock split-brain note.
6. Final recommendation delivered in-chat with the dismiss-reason.
7. This report + its index row, written in ONE edit window (lesson from segment 1 applied —
   see d1).

## b) PARTIALLY DONE

1. **Decision persistence** — the recommendation lives in chat + this report, but NO durable
   decision record exists yet (no ADR, no ADR-0142 refused-row, no TODO_LIST row, no dnsblockd
   addendum). It becomes shelfware if the next session doesn't find it. Gated on the owner
   ruling (§g-Q1) — but the trigger conditions themselves could be recorded NOW without the
   ruling.
2. **Trigger-location decision** — dnsblockd's deep-dive already has a "Revisit triggers"
   list for the transport question; the EXTRACTION trigger should live in the same list (one
   home per trigger class), not yet done.
3. **Collision check against overnight work** — the tree moved overnight (ADR-0165 completion
   arc 18:30/21:53, 03-40 triage report; my 16:35 report was indexed by a foreign session).
   I noted this but did NOT verify whether any overnight session already ruled on iroh (one
   30-second grep would confirm; deferred per the no-unrelated-research instruction).

## c) NOT STARTED

1. The owner ruling itself (§g-Q1) — park vs commission; gates everything in §f group A.
2. The decision-record write (ADR or refused-row + deep-dive cross-link) — item 2 in §f.
3. The shared-core DESIGN (if commissioned) — module name, conformance plan, adoption path for
   both consumers; needs contract-30 B-gate co-design WITH the owner, not solo.
4. OpTag-vs-WithClock semantic-diff inventory (cheap pure read; only sketched in-chat).
5. ALL carried verification debt from segment 1 (race runs, CI quic-leg, quiet-window P99,
   systemscenario feasibility, census receipt, T11.3, stale-doc corrections) — untouched;
   tracked in the 16:35 report §f, not duplicated here.

## d) TOTALLY FUCKED UP

No damage: read-only segment; zero code touched; recommendation is sound and evidence-backed.
The honest d-section is one process failure carried over from segment 1:

1. **The 16:35 report listed work as DONE that was NOT executed in-session** — §a9 claimed
   "This report + `docs/status/README.md` index row (house close-out convention)", but the
   index edit never happened in that session. A foreign overnight session added the row
   (terse, accurate summary: "Parallel session (foreign): irohengine testing-ladder work +
   dnsblockd fit assessment") — so the gap self-healed by LUCK, not by correctness. Worse,
   the same day's 08:52 report documents index drift as a RED overnight gate: this repo
   treats status-index drift as a first-class defect class, and I nearly authored one. The
   claim-at-write-time was false regardless of the later foreign fix.
2. Contrast/improvement proof: this segment used zero blocked commands (segment 1 had the
   `rg -r` third-strike hook block), and the report+index are being written atomically in one
   edit window.

## e) WHAT WE SHOULD IMPROVE

1. **Never list unexecuted work as done in a status report** — the report is a receipt
   document; a false receipt is worse than a missing row. If an index row is intended, it
   must land in the same session, ideally same edit window.
2. **Atomic close-out**: report file + index row + (when authorized) TODO_LIST harvest in ONE
   sequence; the 2026-11-05 split-brain incident and this morning's near-miss both came from
   splitting that atom.
3. **Persist recommendations as decision records, not chat**: this is the second iroh-arc
   artifact whose only durable home is docs/status; a ruling that matters (§g-Q1) deserves an
   ADR or a refused-row the moment it is made.
4. **Check for foreign sessions before filing decisions** — the tree moved overnight; a
   30-second grep of the newest reports for "iroh" would have confirmed no collision before
   this report. Deferred only because of the no-unrelated-research instruction; make it part
   of the close-out preflight instead.
5. Evidence discipline from segment 1 held this time: every claim in the recommendation has a
   file/line or grep receipt. Keep that bar.

## f) Up to 50 things we should get done next

**A. Extraction-ruling track (NEW, from this segment) — gated on §g-Q1**

1. Owner ruling: park (decision record) vs commission (shared CRDT core co-design).
2. If PARK: write the dated decision record in go-cqrs-lite — ADR-0142 refused-row style entry
   (beside Redis/RocksDB) + cross-link from dnsblockd's deep-dive Candidate 3 — citing this
   segment's receipts (zero consumers, 4,389 LOC, fabric/storage boundary, deployment mismatch).
3. If COMMISSION: run the ADR-0142 B-gate co-design WITH the owner first: ONE shared runtime
   per class; name the module; conformance plan; both consumers' adoption path; only then code.
4. If commissioned, seed from dnsblockd's portable slice: pure core + plain-testing property/
   chaos tests (verified framework-free this segment).
5. Record the EXTRACTION trigger conditions in dnsblockd's existing "Revisit triggers" list
   (one home per trigger class) regardless of the ruling — triggers are facts, not verdicts.
6. Resolve the OpTag (wall-clock+node-id) vs WithClock (LWW) split brain with a written
   semantic-diff inventory BEFORE any future merge — cheap pure read, prevents a silent
   ordering flip.
7. Collision preflight before any iroh decision lands: grep the overnight reports
   (ADR-0165 arc, 03-40 triage) for "iroh" to confirm no parallel ruling exists yet.

**B. Carried verification debt (segment 1 §f — top items, unchanged, receipt outstanding)**

8. `-race` over the three iroh modules. [carry]
9. CI quic-leg verification (CGo + pinned iroh-go fork static lib provenance). [carry]
10. Quiet-window / load-sweep P99 methodology run (worst-of-30 evidence class). [carry]
11. `systemscenario`-over-iroh feasibility (programmatic engine in DeploymentConfig?). [carry]
12. Fleet census receipt for iroh (`census-fleet.sh --zero-consumers`) → episode-7
    ghost-engine item. [carry]
13. T11.3 iroh-via-local boot path: implement or close with dated ruling. [carry]
14. Stale "iroh×3 untagged trains" doc correction (tags exist: v4.3.2/v4.0.5/v4.2.3). [carry]
15. loopback latency-test gap: deliberate or missing. [carry]
16. loopback/quic ADT-matrix factory completeness read. [carry]
17. Journal-survives-reset pin for iroh specifically. [carry]
18. Partition/heal convergence coverage vs dnsblockd's chaos suite. [carry]
19. Bench tier (quic + loopback) run + recorded numbers. [carry]
20. ADR-0096 addendum: pinned community fork risk. [carry]
21. recipes.md iroh testing recipe + TestRecipes catalog entry. [carry]
22. WriteOp.ID v5 note (remote-chosen identity) decision. [carry]
23. `transport.go:141` production-sleep gotcha note. [carry]
24. `-short` skip route in check-test-skips. [carry]
25. test-runtime baseline freshness for the trio. [carry]
26. Untested-exports scoped run. [carry]
27. check-arch/cycles receipts for the trio (quoted-not-run class). [carry]
28. engine-capabilities regen check (iroh row). [carry]
29. Skill-ref pyramid pointer (modules.md/advanced.md). [carry]
30. Log-shape pins for quic WARN fixtures. [carry]
31. Demo build coverage in CI legs. [carry]
32. ADR-0056 `TimeUnixDynamic` pin (systemtest/time_fidelity_test.go:18) kept green through
    v5 codec changes. [carry]
33. go.sum health of trio at next wave boundary. [carry]
34. Quic fork bump path budget (CGo static lib in flake). [carry]

**C. dnsblockd side (nothing persisted there yet, from both segments)**

35. Record the Candidate-3 re-confirmation addendum (2026-11-08/09 library receipts). [carry]
36. Revisit-trigger status vs roadmap (WAN / >8 nodes / NAT plans). [carry]
37. If a trigger fires: spike loopback tier (no CGo) as the transport base, not quic. [carry]

**D. Process (from this morning's near-miss)**

38. TODO_LIST harvest — TWO session reports now owe it (16:35 §f + this §f); blocked on the
    harvest-policy answer (§g-Q3). [carry, elevated]
39. Add crush-config lesson: "never list unexecuted work as done; report+index atomically". [NEW]
40. Add crush-config lesson: `rg -r` trap (third strike, segment 1). [carry]
41. Adopt the "how do we test X" answer template (ladder → conformance → gates → receipts →
    consumer-fit pushback) as the standing shape. [carry]
42. Evidence-class labeling in answers (local-once / CI-verified / worst-of-N). [carry]
43. Close-out preflight: foreign-session check + index-row verification before filing any
    report. [NEW]
44. After the ruling lands: annotate (not rewrite) both session reports per docs-health
    ANNOTATE mode so the decision record and the reports reconcile. [carry]
45. If parked: fold iroh into the deliberate-refused list so future fit analyses cite instead
    of re-deriving (same as item 2 — the deep-dive cross-link makes it bidirectional). [carry]
46. If a second fleet consumer ever appears: re-run worst-of-30 P99 before trusting 150ms for
    a new deployment class. [carry]
47. Track dnsblockd's cluster-wave planning docs (2026-10-23/24) as the authoritative
    cluster-history citation. [carry]
48. Verify trio go.sum/go.mod coherence at the next wave boundary the trio rides. [carry]
49. When answering cross-repo fit questions, persist the candidate inventory (§a5) as the
    reusable artifact — this segment's inventory is the template. [NEW]
50. Keep `internal/cluster` untouched until the ruling (standing from segment 1). [carry]

## g) Questions only the owner can answer

1. **The extraction ruling (the segment's actual question):** park with a dated decision
   record, or commission the shared pure replication core (co-designed per ADR-0142 B-gate)?
   Everything in §f group A hangs on this — and the trigger conditions are already met by
   NOTHING today, so my recommendation is park; confirm or overrule.
2. **Verification depth:** burn the race/CI/census verification slice now, or are the
   plain-green receipts from segment 1 sufficient? (Carried unanswered from the 16:35 report.)
3. **Harvest policy conflict:** the standing Close-a-Wave rule says harvest report follow-ups
   into TODO_LIST BEFORE session end; your standing instruction is "write report, then WAIT".
   Two reports (16:35 + this) now owe harvest. Which wins: auto-harvest at report-close, or
   per-report commissioning?

## Reproducibility

```bash
cat /home/lars/projects/go-cqrs-lite/example/mesh-demo/go.mod | head -15   # no iroh dep
rg -in "gossip|anti-entropy|antientropy|membership" \
  /home/lars/projects/go-cqrs-lite/metaengine/irohengine/*.go | grep -v _test   # zero hits
wc -l /home/lars/projects/dnsblockd/internal/cluster/*.go | tail -1        # 4389 total
rg -n "go-cqrs-lite" /home/lars/projects/dnsblockd/internal/cluster/       # zero hits
rg -n "gomega|ginkgo" /home/lars/projects/dnsblockd/internal/cluster/*_test.go  # zero hits
```

Sources: `example/mesh-demo/go.mod`, `example/mesh-demo/main.go` (header), irohengine
production sweep, dnsblockd `internal/cluster/node.go` + `crdt.go` (headers, segment 1),
dnsblockd `docs/research/2026-10-23_gossip-transport-deep-dive.md:80-126` (segment 1),
`docs/status/README.md` index state 2026-11-09 07:53.
