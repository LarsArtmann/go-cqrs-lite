# Owner Ruling Memo — ADR-0139 Encryption-at-Rest + Sweep §4 SQL Columns

**Date:** 2026-09-22
**Purpose:** ONE consolidated reply settles the five open rulings that gate
the v5 train's remaining items (T25/T26 consolidation). Each question lists
the options, the tradeoffs, and this memo's recommendation with reasons.
Reply per question with the option letter (or "as recommended").

---

## Q1–Q4: ADR-0139 encryption-at-rest ([docs/adr/0139](../adr/0139-v5-encryption-at-rest-configuration.md))

### Q1. KeyProvider call semantics

`KeyProvider func(ctx) ([]byte, error)` — WHEN is it called?

| Option | Semantics | Tradeoffs |
| ------ | --------- | --------- |
| A | Once at engine construction; key cached until Close | Deterministic, trivially testable; rotation needs an explicit re-open/Rotate step |
| B | Per key-open (every file/txn open re-calls) | Rotation converges passively; a failing/slow provider turns every operation into a potential failure (context deadlines kill background writes) |
| C | Cached with TTL | Passive rotation without per-op calls; a hidden clock — non-deterministic tests, surprise mid-write key switches |

**Recommendation: A + explicit rotation seam.** The provider is called once
per engine construction; rotation is an operator action through an explicit
`Rotate`/re-open path (v5.x), and old-key DECRYPTION keeps working through
the existing `encryption.KeyResolver` embedded-key-ID lookup (already
rotation-shaped). Rationale: no hidden clocks (C is the test-flake factory
this repo keeps killing), no per-op failure surface (B makes every write a
provider call), and the snapshot re-encrypt-on-touch proof shows
convergence-on-touch works WITHOUT a passive provider — the writer just
needs the current key, which A gives it after rotation re-open.

### Q2. Key-REFERENCE validation timing

`DeploymentConfig` carries env/file/secret-manager references (never keys).

| Option | Behavior | Tradeoffs |
| ------ | -------- | --------- |
| A | Fail-fast: deploy-check resolves every reference eagerly | Catches typos before engines build; fails when a secret manager is unreachable at check time (sidecar warm-up races) |
| B | Lazy: references resolve at engine construction; construction fails loudly | Single failure point (the established refusal precedent); form errors surface late |
| C | Layered: deploy-check validates reference FORM (non-empty, known scheme); construction does the actual resolution | Catches the cheap errors early without demanding live secret-manager auth at check time |

**Recommendation: C.** Form validation is free and kills the typo class at
CI/deploy-check; eager resolution (A) couples deploy-check liveness to
external systems this library does not own (and cannot depend on — secret
manager integrations stay out per the ADR's dependency-budget note).
Construction-time loud failure stays the last line of defense, exactly like
`RejectDurabilityTier`.

### Q3. Scope at v5: do read-model engine files join?

| Option | Scope | Tradeoffs |
| ------ | ----- | --------- |
| A | Event streams + snapshots at v5; matview state later | Protects the irreplaceable data first; ships with machinery that already has proofs (transforms, rotation convergence) |
| B | Events + snapshots + matview files at v5 | One surface immediately; but matview state is DERIVED — and every first-party engine can already reset + replay it (ADR-0136/0143) |

**Recommendation: A.** The journal (events) is the crown jewel — ADR-0143
makes it the replay source that survives every reset; snapshots are the
second irreplaceable artifact. Matview/read-model files are rebuildable from
the journal by design, so their at-rest classification is lower and their
encryption can ride a later minor without weakening the security story.
Declaring `SupportsEncryption`-class capability on all engines at v5 (per the
ADR's consequence list) keeps B open as pure config addition.

### Q4. Plaintext → encrypted migration path

| Option | Path | Tradeoffs |
| ------ | ---- | --------- |
| A | Mandated in-place migration tool at v5 (batched re-encrypt of every row) | Zero-choice for operators; a large-table campaign as a CONSTRUCTION side effect — exactly what the snapshots-column assessment rejected for events |
| B | No tool at v5: re-seed OR re-encrypt-on-touch convergence (documented pattern; the PG snapshot rotation proof generalizes) | v5.0 migration surface stays equal to the snapshots migration; the campaign, if wanted, is a deliberate v5.x operator step |

**Recommendation: B**, for the same reason `docs/WIRE-FORMAT-KEYS.md`
defers the SQL column rename: a full-table rewrite belongs to an
operator-driven window, not a version cut. Document the two supported
transitions in V5-MIGRATION-GUIDE (re-seed; re-encrypt-on-touch via the
transform pair) and add an in-place campaign only on demand in a 5.x minor.

---

## Q5. Sweep §4 tail: SQL `events`/`commands` column rename — v5.0 or 5.x?

Assessment + recommendation already on file in
[`docs/WIRE-FORMAT-KEYS.md`](../WIRE-FORMAT-KEYS.md) (M25.2): expand-contract
in a **5.x minor**, NOT the v5.0 cut. Rationale recap: columns have no
decode-only fallback; on the events table (largest, hottest) `RENAME COLUMN`
is a full copy (MySQL/MariaDB) or a lock-taking metadata change (PG) —
unacceptable as a construction side effect. The expand-contract plan
(1. add + dual-write + batched backfill; 2. switch reads; 3. drop + re-index
a release later) is written there.

**Recommendation: ratify 5.x.** Consequence for v5.0.0: the SQL events/
commands tables remain the one binary surface carrying the aggregate
vocabulary through the v5 cut (documented in the wire-key table's status
column) — matching the transport/grpc precedent (module deleted at v5 with
its proto fields unchanged).

---

## What each ruling unblocks

| Ruling | Unblocks |
| ------ | -------- |
| Q1–Q4 | ADR-0139 status DRAFT → ACCEPTED; `DriverConfig.Encryption` + KeyProvider implementation (the L-effort item) |
| Q5 | v5.0.0 cut scope freeze (the cut can proceed with SQL columns documented as the deferred 5.x item) |

Reply format: five letters (or "all as recommended").
