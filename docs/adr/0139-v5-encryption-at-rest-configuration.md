# ADR-0139: v5 Encryption-at-Rest Configuration

**Status:** DRAFT (skeleton — see "Open questions")
**Date:** 2026-09-13
**Deciders:** Lars (owner ruling pending)
**Source threads:** TODO_LIST "v5 ADR: encryption-at-rest configuration"
(20-18 §f15-17/§f21, 20-57 §f9-10)

---

## Context

Encryption at rest exists in v4 as per-layer, caller-assembled opt-ins:

- **Event streams:** `encryption.EncryptSinkTransform` + `event.DecorateStore`
  (ADR-0126 canonical store wrapping) encrypt every sink write and decrypt on
  source reads.
- **Snapshots:** `encryption` snapshot-state codecs, with rotation converging
  as rows are touched — no rewrite campaign (PG proof:
  `storage/pg_integration_snapshot_encryption_test.go`).
- **Key selection:** `encryption.KeyResolver` selects a Decrypter by the key
  ID embedded in each ciphertext, which is what makes rotation possible at
  all.
- **Key material today:** the caller holds raw `[]byte` keys in memory and
  wires them into constructors by hand; nothing in `metaengine` or `system/`
  knows encryption exists.

The v5 unification (ADR-0123) collapses composition into
`system.New(...)` + engine `DriverConfig`s. That is the moment to decide
whether encryption-at-rest stays a hand-wired transform — or becomes a
declared, operator-chosen capability like durability and materialized views.

The user's north star (AGENTS.md, metaengine section) says operators own
WHERE data lives at deployment time; encryption-at-rest is a deployment-time
property of storage, so it belongs on the operator side of that boundary —
not in developer-facing code.

## Decision (proposed)

1. **`DriverConfig.Encryption` on engines** — engines DECLARE whether they
   can honor encryption-at-rest; operators TURN IT ON via driver config.
2. **`KeyProvider func(ctx) ([]byte, error)` — never raw `key []byte` in
   config structs.** The provider indirection:
   - enables rotation / hot-reload without config struct surgery,
   - keeps key material OUT of config (config is logged, marshaled, shipped
     in Doctor output — a raw key there is a leak),
   - sets THE precedent for the sibling secret class: pg/mysql passwords in
     DSN construction get the same provider shape at v5.
3. **`system/` DeploymentConfig carries a key REFERENCE slot** —
   env-var/file/secret-manager reference, never the key itself. The reference
   resolves through the KeyProvider at engine construction.
4. **Engines fail construction loudly when unable to honor the request** —
   the established refusal precedent (`RejectDurabilityTier`,
   `MaterializedViews` capability checks), not a silent downgrade to
   plaintext. A silent downgrade is a security incident, not a fallback.

## Precedents already in the tree

| Precedent                                                        | Where                                                 | What it proves                                       |
| ---------------------------------------------------------------- | ----------------------------------------------------- | ---------------------------------------------------- |
| `KeyResolver` by embedded key ID                                 | `encryption/algorithm.go`                             | rotation-compatible decryption                       |
| `EncryptSinkTransform` + `DecorateStore`                         | `encryption/store.go`, ADR-0126                       | store-level encryption without hand-written wrappers |
| Snapshot rotation convergence                                    | `encryption/snapshot_state.go` + PG integration proof | re-encrypt-on-touch works without a rewrite campaign |
| `RejectDurabilityTier` / `MaterializedViews` capability refusals | metaengine engines                                    | loud construction-time refusal as the contract shape |
| `redactDSN` contract                                             | recipes §2.33                                         | secrets never appear in errors or rendered config    |

## Consequences

- v4's hand-wired transforms remain valid (composition is unchanged); v5
  adds the operator-facing declaration layer on top.
- Every first-party engine needs a `SupportsEncryption`-class answer; engines
  that cannot (none known today — all are byte-store backed) must refuse.
- The KeyProvider signature becomes public API at v5 — its context semantics
  (per-construction call? per-open call? cached?) must be pinned BEFORE the
  v5 tag freeze.
- Secret-manager integrations stay OUT of go-cqrs-lite (dependency budget);
  the provider func is the only seam they need.

## Open questions (owner ruling requested)

1. KeyProvider call semantics: once at construction vs per-key-open vs
   cached-with-TTL? (Rotation convergence behavior depends on this.)
2. Does `system.DeploymentConfig` validate that a key REFERENCE resolves at
   deploy-check time (fail-fast) or lazily at engine construction?
3. Scope: event streams + snapshots at v5 — do read-model engine files
   (matview state) join the same config surface in v5 or later?
4. Migration: is a v4-plaintext → v5-encrypted in-place migration path
   required at v5, or is re-seed acceptable (the snapshot rotation proof
   suggests re-encrypt-on-touch is viable for streams too)?
