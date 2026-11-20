# Reviews — Series Index

Point-in-time review reports (data-model, schema, depth) — snapshots, not
living documents. Each links its findings to what actually happened to them;
errata are applied in place with dated notes rather than silently.

| Report | One-line takeaway |
| ------ | ----------------- |
| [2026-11-19 core data-model review](2026-11-19_core-data-model-review.html) | The core/v5 NEW type surface audit (R1-R12) that drove the type-finalization plan; R2/R3 brands, R4 Payload tri-state, R5 StreamRef split-brain, R6-R8 causation/actor/clock lies — all fixed by the 2026-11-20 waves (M03-M09), ADR-0166 Accepted. Errata applied 2026-11-20 (M02): honest methodology (24/91 files + census greps), R1 cut is T26-gated, R10 severity corrected. Cross-refs: PC-1 (typed WAL) and PC-3 (key law) below. |
| [2026-11-09 brutal self-review](2026-11-09_07-30_brutal-self-review.html) | Session-quality critique: ghost systems, split brains, and the improvement plan that seeded the brutal-episode7 rulings (incl. ADR-0166's original B3 proposal). |
| [2026-11-08 brutal self-review](2026-11-08_13-20_brutal-self-review.html) | The earlier episode: v4/v5 dual-train drift inventory, ~20k LOC deprecated surface quantified — inputs to the 11-13 deep dive's top risk. |
| [2026-10-17 BDD depth follow-ups](2026-10-17_bdd-depth-followups-owner-decision-pack.md) | Owner decision pack for the BDD-depth follow-ups (scenario/systemscenario boundaries). |
| [2026-10-17 on-disk schema data-model review](2026-10-17_on-disk-schema-data-model-review.html) | The STORAGE-side counterpart to the 11-19 review: PC-1 typed WAL (rides the v5 core Record work), PC-3 key law (one canonical stream key — the on-disk twin of ADR-0166), PC-5 fingerprints (the drift killer); fixes replayable through reset/replay (ADR-0136/0143). |
| [2026-09-29 pebble/bbolt versioned-cells scope](2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md) | Scope decision for versioned-cell storage in the embedded engines. |
| [2026-09-22 ADR-0139 encryption + sweep memo](2026-09-22_adr0139-encryption-and-sweep-owner-memo.md) | Owner memo behind the encryption-at-rest rulings (cut Layer 11). |
| [2026-09-20 queue dep-validation ratification](2026-09-20_queue-dep-validation-ratification-memo.md) | Ratification memo for queue dependency validation. |

Older snapshots live in [archived/](archived/). The series conversation:
the 10-17 review found the on-disk half of the identity/type problems, the
11-19 review found the in-memory half, and the 2026-11-20 type-finalization
plan (`docs/planning/2026-11-19_20-52_SUPERB-v5-core-type-finalization-pareto-plan.md`)
closed the in-memory half — the on-disk half (PC-1..PC-5) remains staged for
the cut.
