# Status Report — Zenoh-vs-Watermill Clarification Turn

> **Date:** 2026-09-15 19:57 CEST
> **Scope:** This turn only — the conceptual question _"is Zenoh a watermill adaptor, a storage, or something totally different?"_ — answered from already-verified research; plus this review pass. No new research, no artifacts, no code.
> **Format note:** `.md` per explicit user instruction (status-report skill HTML default overridden — third time; consistently flagged, never propagated).

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                 |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| 1 | Answer delivered: **Zenoh is neither — it's a network fabric one layer below watermill's plugin ecosystem** (a broker-replacement like Redis/Kafka/NATS, but brokerless-capable), with a 4-layer placement table (bridge code / broker-fabric / event store / KV cache)                                                                                  | Chat answer, this turn                                                                   |
| 2 | Every factual claim in the answer traced to prior in-chain verification: no zenoh↔watermill plugin exists (gh search, zenoh session), zenoh storages = latest-value KV without versioning/OCC/replay (storage-manager docs), adapter-would-slot-into-`WithBackend` (watermill README + `event_bus_options.go:26`), geo-query/liveliness/HLC (zenoh docs) | All verified earlier in this conversation chain — **zero new unverified claims shipped** |

## b) PARTIALLY DONE

| # | Item                                                     | What's missing                                                                                                                                                |
| - | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | The crisp 4-layer mental-model table exists only in chat | Not persisted anywhere (candidate: small addendum to the zenoh report §4 or the watermill skill's backends.md "not verified here" section) — not done unasked |

## c) NOT STARTED

Nothing was in scope for this turn. Standing not-started items from the prior reports remain: skill eval loop, NATS JetStream roundtrip leg, zenoh go/no-go, HARVEST of the three (f) lists.

## d) TOTALLY FUCKED UP

Nothing this turn — smallest possible slice, and the claims discipline finally held (all assertions pre-verified before shipping). Honest observations, not fuckups:

1. This is the **third status report in the chain**; their (f) lists overlap (NATS leg, evals appear in ≥2). Merging them into TODO_LIST via HARVEST stays blocked on the standing "WAIT".
2. **Concurrent foreign work is now unmistakable and bigger than benchkit**: working tree carries `M .agents/skills/go-cqrs-lite/SKILL.md`, `M FEATURES.md`, `M docs/api_surface.txt`, `M docs/benchmarks/README.md`; recent daemon commits show a benchkit/cqrs-bench feature wave (CHANGELOG +34, benchkit/doc.go, cqrs-bench render removal + main_test, api golden growth, a 273-line feedback doc). Inspected read-only; untouched. My `git status`-at-turn-end habit (adopted this chain) caught it immediately this time — improvement #2 from the 19:45 report is working.

## e) WHAT WE SHOULD IMPROVE

1. **Claims discipline held this turn — keep the checklist and make it permanent** (the 2-report rule has now fired; codify it, e.g. one line in AGENTS.md or docs/agents/, still pending from report #2's f8).
2. The 4-layer placement table is exactly the kind of artifact that should live in a doc, not chat — persist small crystallizations at answer time when a related doc exists (zenoh report).
3. Three overlapping (f) lists with no harvest = entropy; either harvest on next instruction or explicitly retire duplicates.

## f) Next tasks (ranked; only turn-specific + still-standing top items)

| #  | Task                                                                                                                                                | Impact   | Effort | Cat      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------- |
| 1  | Persist the 4-layer zenoh/watermill placement table as a dated addendum in the zenoh report (or watermill backends.md)                              | Low      | S      | Docs     |
| 2  | Merge + harvest the three reports' (f) lists into TODO_LIST/ROADMAP (docs-health HARVEST) — dedupe NATS leg / evals / go-no-go                      | High     | S      | Docs     |
| 3  | Run the watermill skill trigger evals (3 prompts drafted)                                                                                           | High     | M      | Quality  |
| 4  | NATS JetStream roundtrip test leg (`watermill-nats/v2` + `ephemeral-nats.sh`)                                                                       | High     | M      | Feature  |
| 5  | Zenoh go/no-go decision (gates the zenoh W1 plugin spike)                                                                                           | Critical | S      | Decision |
| 6  | Watermill-skill `references/advanced.md` (Delayed Messages, Requeuing, FanIn/FanOut, Metrics, Troubleshooting)                                      | Medium   | M      | Docs     |
| 7  | Cross-link go-cqrs-lite skill ↔ watermill skill                                                                                                     | Medium   | S      | Docs     |
| 8  | Codify claims-checklist rule (AGENTS.md/docs-agents line)                                                                                           | Medium   | S      | Process  |
| 9  | CHANGELOG decision for README NATS correction                                                                                                       | Low      | S      | Docs     |
| 10 | Coordinate on the concurrent benchkit/cqrs-bench wave before touching anything adjacent (api_surface.txt, FEATURES.md are dirty with foreign edits) | Medium   | S      | Hygiene  |

## g) Questions I cannot answer myself

1. (Standing) **Zenoh go/no-go** — pursue the W1 watermill-zenoh plugin, or file as landscape research?
2. (Standing) **HARVEST now?** Three reports' (f) lists are quarantined per your WAIT — harvest into TODO_LIST/ROADMAP, or keep holding?
3. (New) The **concurrent benchkit/cqrs-bench session** is now editing shared surfaces (`api_surface.txt`, `FEATURES.md`, go-cqrs-lite `SKILL.md`). Should I treat those files as write-locked by that session until its wave commits, or is parallel editing acceptable to you?

---

**State at close:** this turn produced no working-tree changes of mine; tree holds only the foreign benchkit-feature edits listed in (d); no gates run (nothing of mine changed). WAITING FOR INSTRUCTIONS.
