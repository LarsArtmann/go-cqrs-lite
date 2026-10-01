# Owner Decision Pack — quick rulings, ADR-level rulings, user actions

> Compiled 2026-10-01 (M15.1 + M15.2 + M15.4 of the SUPERB post-wave plan).
> Every item is ALREADY researched with a recommendation; a one-word answer per
> row unblocks the linked TODO_LIST row. Nothing here is executed on the
> owner's behalf — filings, pushes, tags, and legal files stay gated.

## Part A — Quick rulings (one-liners)

| #   | Question                                                                                                                                      | Recommendation                                                                                                          | Unblocks           |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------ |
| A1  | **M22/Q3 report-artifact policy**: narrow skill trigger (1–2 modules) — chat answer + "report on request", or always write the full artifact? | Sanction the deviation for narrow triggers; codify in the skill                                                         | M22/Q3 row         |
| A2  | **Push cadence**: push `master` after daemon-absorbed sessions, or only at release tags?                                                      | Push at each session close (tags are already public; master lag is pure drift risk)                                     | this session Q1    |
| A3  | **claiming V006 advisory**: content-identical `claiming/v4.0.1` re-tag, or teach V006 to skip pins at a module's newest tag?                  | Linter-semantics fix (skip at newest existing tag) — no re-tag of a content-identical module                            | claiming V006 row  |
| A4  | **iroh P99 bound 50→150ms** (worst-of-30 under gate load) — ratify?                                                                           | Ratify (documented provenance; sample methodology inflates tail)                                                        | iroh P99 row       |
| A5  | **T18b (a) deadline-lapse policy**: auto-re-arm vs one-shot                                                                                   | Auto-re-arm (the armed-pipeline mechanics are retired; policy only affects future re-arms)                              | T18b tail          |
| A6  | **T18b (b) benchmark-ceiling**: keep strict load1<5?                                                                                          | Keep strict (M11's re-runs exist to be honest, not cheap)                                                               | T18b tail          |
| A7  | **Turso DSN strict-vs-lenient**: reject unknown `*encrypt*`/`*key*` params at construction?                                                   | STRICT — silent-unencrypted DBs are the unacceptable state; behavior change is v4.x-warn/v5-hard                        | Turso DSN row      |
| A8  | **Turso sync/embedded-replica first-class support**: real consumer need or out of scope?                                                      | Defer until a consumer asks (matview-v2-style demand gate)                                                              | sync/embedded row  |
| A9  | **dgraph one-RPC scope (Q1)**: flip Set/Multimap/Log/StreamLog to O1 in one wave, or incremental?                                             | Authorize one wave AFTER per-ADT reassessment benches pass (ADTMap precedent)                                           | dgraph Q1 row      |
| A10 | **CapabilityGaps → Doctor (Q2)**: should documented gaps also silence Doctor's `--- Capability ---` violation lines?                          | Yes — one gap source, two silencers is a split brain                                                                    | CapabilityGaps row |
| A11 | **`#test-examples` joins blocking `#verify`?**                                                                                                | Yes for CI, keep local `#verify` without it (fast loops); the projectionhost bug class lived in the build-vs-tested gap | #test-examples row |
| A12 | **docs-health cadence**: weekly?                                                                                                              | Weekly, Sundays, alongside the existing nightly-gates Sunday legs                                                       | docs-health (c)    |
| A13 | **benchkit tag wave**: cut under the blanket "GET SHIT DONE" sanction (M09 precedent) or hold for explicit sign-off?                          | Cut under sanction (mechanics-only, same class as M09)                                                                  | this session Q2    |
| A14 | **Quiet-window strategy** while load1 stays ≥14: (a) auto-poll opportunistically, (b) owner names a window, (c) raise the ceiling?            | (a) auto-poll as today + (b) name a Sunday window for the composed verify if polling keeps missing                      | this session Q3    |

## Part B — ADR-level rulings (paragraph each)

### B1. SingleWriter → ADR-0146 (ratify one-pager)

One-pager: `docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md`.
Recommendation: `EngineConfig.SingleWriter` advisory lease — `<dsn>.cqrs-lease`
flock, fail-loud, default-off, one shared Tier-0-style helper. Today lease
semantics exist only in `queue/`+`claiming/` task claims; the Context-Variable
Phase-0 ADR conditions every library-store cutover on exactly this marker.
**Ruling needed:** ratify as ADR-0146 (+ authorize implementation before v5
freezes engine construction surfaces) or reject with the alternative.

### B2. Direction ruling → ADR-0147 (pick an option)

Evidence + decision memo (G-T01, delivered 2026-09-21):
`docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md`
— R01 Infer coverage inventory, R02 sanctioned-surface coverage matrix, R03
consumer shapes, 3 options + hybrid recommendation. The Goal sentence is
undefinable at 100% until this is ruled; the ruling lands as ADR-0147.
**Ruling needed:** option (or hybrid) + one-line Goal amendment.

### B3. v5 encryption-at-rest — the 4 open questions (ADR-0139)

Skeleton ADR shipped: `docs/adr/0139-v5-encryption-at-rest-configuration.md`
(`DriverConfig.Encryption` + `KeyProvider func(ctx) ([]byte, error)` +
DeploymentConfig key-reference slot; engines fail construction loudly).
**Rulings needed:**

1. Provider call semantics — per-operation, per-connection, or construction-time-only?
2. Reference validation timing — construction (fail-loud) or first use?
3. Read-model scope — encrypt projections/checkpoints too, or journals only?
4. Plaintext→encrypted migration — online (dual-read window) or offline re-import?

### B4. ADR-0138 demand check (command sourcing draft)

Design-doc-only today, building on W2's bridge, reconciling ADR-0112's planned
`CommandAwareFold`. **Ruling needed:** is there consumer demand NOW (name the
consumer) or does the draft stay parked until asked? Recommendation: park
(demand-gated like matview v2).

### B5. This session's design rulings

1. **M16.1 clock seam**: Option B (internal `nowFn` seam; `queue.WithClock` at v5) vs C (interface clock). Recommend **B** — smallest surface, ADR-0122-compatible. Doc: `docs/planning/2026-09-29_07-20_queue-conformance-clock-seam-design.md`.
2. **M18.4 pebble/bbolt versioned cells**: DEFER with demand trigger (recommendation in `docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md`; rulings A–D pending).
3. **M19 NATS CI-leg**: add to ci.yml now (needs only a nix-provisioned nats-server) or keep billing-gated with the other paid legs? Recommend **now** — the `#integration-nats` app is nix-based like the free legs.

## Part C — User actions (cannot be done by the agent)

- [ ] **GH Actions billing** — every paid CI job fails in 3–7s since ~2026-07-17 (billing/payment method). Until fixed, local `nix run .#verify` is the authoritative gate.
- [ ] **Set the `ERRAUDIT_PAT` secret** — the `error-audit` CI job arms the moment it exists (findings verified zero as of 2026-09-15).
- [ ] **Run the skill evals via the `claude` CLI** — `evals/trigger-eval-set.json` / `evals/evals.json` are UNVALIDATED since the description change.
- [ ] **benchkit/LICENSE "Unknown Author"** — correct to the real name or leave; a legal notice is not agent-editable.

## Answer routing

Reply inline (A1–A14, B1–B5, C checked-off) — each answer is routed to its
TODO_LIST row by the next session (M15.3).
