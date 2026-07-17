# go-cqrs-lite — Consumer Feedback: Snapshot Encryption Gap

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — bank transaction sync CLI (Wise API + Qonto CSV → SQLite)
**SDK version:** v4.0.x (encryption/v4, snapshot/v4, decider/v4, stack/v4)
**Date:** 2026-07-17
**Severity:** High — defeats the event-encryption threat model
**Status:** Gap verified against source. Undocumented workaround exists but is insufficient as a first-class story.

---

## TL;DR

`encryption.NewEncryptedStore` encrypts event payloads at rest. There is **no analogue for snapshots.** Snapshots persist folded aggregate state (`Snapshot.State []byte`) as plaintext CBOR/JSON in the `snapshots` table. An attacker who obtains the database file gets ciphertext events but **pre-computed plaintext aggregate state** from every snapshot — the very state the encrypted events fold into. Enabling event encryption while snapshots remain plaintext is **encryption-with-a-hole**: it gives a false sense of security without delivering it.

This is not a missing convenience. It is a **threat-model break.** bank-sync shipped event encryption (commit `4985fb0`) and snapshots are still plaintext because the SDK offers no snapshot encryption entry point and the composition workaround is undiscoverable.

---

## 1. The gap — verified against source

### Events have a first-class encrypted store

```go
// encryption/store.go:36
func NewEncryptedStore(inner event.Store, cipher EncrypterDecrypter, opts ...MiddlewareOption) (*encryptedStore, error)
```

`encryptedStore` implements `event.EventSink`, `event.EventSource`, `event.Journal`, `event.SeekableJournal`, `event.BackwardsSource` — every event store interface. Encrypt on write, decrypt on read, transparent passthrough for legacy plaintext events. Clean, one-liner, discoverable.

### Snapshots have no encryption entry point

The snapshot module exports exactly two constructors:

```go
// snapshot/read_pressure.go
func NewReadPressure(threshold int, opts ...ReadPressureOption) (*ReadPressure, error)

// snapshot/typed.go:73
func NewTypedStore[State any](store SnapshotStore, c codec.Codec) *TypedStore[State]
```

There is no `encryption.NewEncryptedSnapshotStore`. There is no `decider.WithSnapshotEncryption`. The `stack` package — designed to be the "wire everything in one call" layer — has **zero references to the encryption module** at all:

```
$ rg -n "encryption" stack/*.go   # exit 1, no matches
```

The `Snapshot` struct's `State` field is raw bytes with no protection:

```go
// snapshot/store.go:11
type Snapshot struct {
    AggregateID   id.AggregateID   `json:"aggregateId"`
    AggregateType id.AggregateType `json:"aggregateType"`
    Version       event.Version    `json:"version"`
    State         []byte           `json:"state"`    // <-- plaintext encoded state
    CreatedAt     time.Time        `json:"createdAt"`
}
```

The `encryption/doc.go` package documentation talks exclusively about **event payloads** — "event payload encryption and decryption," "encrypt event payloads before storage/transit." Snapshots are never mentioned. A consumer reading the encryption docs would reasonably conclude that enabling `NewEncryptedStore` makes their data store encrypted. It does not. It makes _half_ of it encrypted.

---

## 2. Why this matters — the threat model

Event-sourced snapshots are not auxiliary cache entries. They are **the folded aggregate state** — the exact end product of replaying the encrypted events. They exist specifically to skip the replay.

Consider an attacker who exfiltrates the SQLite file (laptop theft, cloud backup breach, `cp bank-sync.db ~/stolen`):

| Data location           | With `NewEncryptedStore` only | Attacker effort to read             |
| ----------------------- | ----------------------------- | ----------------------------------- |
| `events` table payloads | **Ciphertext** (AES-256-GCM)  | Infeasible without key              |
| `snapshots` table state | **Plaintext** (CBOR-encoded)  | `sqlite3` + `cbor decode` — trivial |

The snapshot **is** the aggregate state. The attacker doesn't need to decrypt a single event. They get the pre-folded result for free, every 50 events (default snapshot interval). The encryption on events provides **zero confidentiality for the aggregate state** that those events produce.

This is the security equivalent of locking the front door and leaving the back window open. The locked door (event encryption) is impressive and visible. The open window (plaintext snapshots) is where the actual breach happens.

### Concrete exposure in bank-sync

bank-sync's aggregate state (`BalanceSyncState`) is persisted in plaintext snapshots every 50 transactions:

```go
// internal/cqrs/state.go:11
type BalanceSyncState struct {
    BalanceID          domain.BalanceID
    ProfileID          domain.ProfileID
    Provider           domain.BankProvider      // "wise" or "qonto"
    LastSyncAt         time.Time
    SyncFrom           time.Time
    SyncTo             time.Time
    SeenTransactionIDs map[string]struct{}      // every transaction ID ever synced
    TotalTransactions  int64                    // lifetime transaction count
    LastError          string
    LastErrorCode      string
}
```

An attacker reading the plaintext snapshot learns:

- **Which bank provider** the victim uses (Wise / Qonto) and their internal profile/balance IDs
- **Exactly how many transactions** they've ever synced (`TotalTransactions`, `len(SeenTransactionIDs)`)
- **The complete set of transaction IDs** (`SeenTransactionIDs`) — for Qonto these are deterministic FNV-1a hashes of transaction content, leaking cardinality and identity
- **Sync cadence and time window** (`LastSyncAt`, `SyncFrom`, `SyncTo`) — usage pattern fingerprinting
- **Operational errors** (`LastError`, `LastErrorCode`) — potential leverage for social engineering

This is sensitive financial metadata. Not the full transaction amounts/merchants (those are in the encrypted events), but enough to identify, profile, and target a victim. And it's sitting in plaintext next to properly encrypted events that took real effort to wire up.

---

## 3. The undocumented workaround — and why it's insufficient

During verification I discovered that snapshot encryption **is technically possible today** via codec composition, because the decider repository uses `r.codec` (settable via `decider.WithCodec`) to serialize snapshot state:

```go
// decider/load.go:235  (load path)
err = r.codec.Decode(snap.State, &state)

// decider/decider.go:211  (save path)
encoded, encErr := r.codec.Encode(finalState)
snapshot.SaveSnapshot(ctx, r.snapshotStore, ref.Type, ref.ID, newVersion, encoded)
```

Since `encryption.NewCodec(codec.CBORCodec{}, enc)` returns a `*encryptingCodec` implementing `codec.Codec` (Encode/Decrypt), passing it to `decider.WithCodec` transitively encrypts snapshot state:

```go
// UNDOCUMENTED WORKAROUND — works today, not mentioned anywhere
enc, _ := encryption.NewAES256GCM(key)
encryptingCodec := encryption.NewCodec(codec.CBORCodec{}, enc)

repo, _ := decider.NewRepository(store, bus, d,
    decider.WithSnapshotStore[State](snapStore),
    decider.WithCodec[State](encryptingCodec),   // <-- encrypts snapshots as a side effect
    decider.WithSnapshotStrategy[State](strategy),
)
```

This works because the decider treats the codec as an opaque `Encode/Decode` pair. It has no idea the codec is encrypting. The snapshot bytes in SQLite are ciphertext.

### Why this is insufficient as the SDK's answer

1. **Zero discoverability.** The connection between "snapshot state goes through a codec" and "therefore you can encrypt it by passing an encrypting codec" is an implementation detail of the decider, not a documented capability. No consumer will find this without reading decider internals. The encryption README and doc.go never mention snapshots. The snapshot README and doc.go never mention encryption.

2. **No parity with the event API.** Events get `NewEncryptedStore(inner, cipher)` — a purpose-built, documented, one-liner with its own file (`store.go`), its own test suite (`store_test.go`), and explicit interface assertions. Snapshots get "figure out that codec composition transitively encrypts them." This asymmetry signals that snapshot encryption is an accident, not a feature.

3. **Silent footgun.** A consumer who enables `NewEncryptedStore` (the obvious, documented path) and also uses snapshots (also documented) gets **half-encrypted storage** with no warning. The SDK makes it trivial to do the right thing for events and impossible (without insider knowledge) to do the right thing for snapshots — and never tells you the two are inconsistent.

4. **No key rotation story for snapshots.** Events support key rotation via `WithMiddlewareKeyID` + `NewStaticKeyResolver` — each event carries a key ID in metadata. Snapshots have no key ID field. The codec-composition workaround uses a single cipher with no rotation path. An operator who rotates the event encryption key has no way to rotate the snapshot key without re-encrypting all snapshots out-of-band.

5. **Doesn't compose with `TypedStore` correctly.** `snapshot.NewTypedStore` calls `codec.WrapEncode`/`codec.UnwrapDecode` which add a codec envelope. Composing `encryption.NewCodec` inside `TypedStore` double-wraps. The decider path (raw `SnapshotStore` + `WithCodec`) is the only clean composition point, and it's undocumented.

---

## 4. Is event-level encryption "good enough" for now?

**No.** For bank-sync specifically, event-level encryption without snapshot encryption is worse than no encryption, because it creates a **false compliance posture.** The WARNING log says "PII event payloads are stored unencrypted — set an encryption key," and when the key is set, it says "event payload encryption enabled." Neither message mentions that snapshots remain plaintext. A security-conscious user who sets the key, sees the success log, and ships believes their data is encrypted. It is not — not all of it.

**For the SDK's next release:** snapshot encryption is not a blocker _if_ the workaround is (a) documented and (b) bank-sync adopts it. But it IS a blocker for any claim that the encryption module provides "data-at-rest protection" or "encryption-at-rest" — those claims are false while snapshots are plaintext, and they appear in the encryption README today:

> **Data-at-rest protection**: Stored events are ciphertext — database compromises leak no sensitive fields

"Stored events" is doing a lot of work in that sentence. The snapshots table is also stored data, and it leaks.

---

## 5. Proposed SDK API — three options, ranked

### Option A (recommended): `encryption.NewEncryptedSnapshotStore` — parity with events

```go
// encryption/snapshot_store.go (new file)
func NewEncryptedSnapshotStore(
    inner snapshot.SnapshotStore,
    cipher EncrypterDecrypter,
    opts ...MiddlewareOption,
) (*encryptedSnapshotStore, error)
```

Mirrors `NewEncryptedStore` exactly. Encrypts `Snapshot.State` on `Save`, decrypts on `Load`/`LoadAtVersion`. Transparent passthrough for legacy plaintext snapshots (same `ExtractCiphertext` → no metadata → return unchanged pattern from `decryptEvent`).

**Why this is the right shape:**

- Perfect parity with `NewEncryptedStore(inner, cipher, opts...)` — the API a consumer already knows.
- One-liner adoption: swap `snapStore` for `encryption.NewEncryptedSnapshotStore(snapStore, cipher)`.
- The opts can carry `WithMiddlewareKeyID` for future snapshot key rotation.
- Interface assertions (`_ snapshot.SnapshotStore = (*encryptedSnapshotStore)(nil)`) make the contract explicit.

**Addressing "snapshots have no metadata for algorithm/keyID":** Same approach as events — encode the key ID and algorithm as a length-prefixed header on the ciphertext blob (the `AttachEncryption`/`ExtractCiphertext` envelope pattern already exists for events; a simpler byte-prefix variant works for snapshots since they don't need per-snapshot structured metadata). Or: since snapshots are replaced (not appended), a single-key model with re-encryption on rotation is acceptable for v1.

### Option B: `decider.WithEncryptedSnapshots[State](cipher)` — repository-level convenience

```go
// decider/options.go
func WithEncryptedSnapshots[State any](cipher encryption.EncrypterDecrypter) RepositoryOption[State]
```

Internally wraps the snapshot store + codec. Encapsulates the composition so consumers don't need to know about codec trickery.

**Why not primary:** Mixes encryption (a cross-cutting concern) into the decider options namespace. Breaks the principle that the decider is codec/store-agnostic. Also doesn't help consumers who use snapshots outside the decider (e.g., direct `TypedStore` usage).

### Option C: Document the codec-composition workaround + add `stack.WithEncryption(cipher)` one-liner

Minimal code: just document that `decider.WithCodec(encryption.NewCodec(codec.CBORCodec{}, enc))` encrypts snapshots, and add a `stack` option that wires both event + snapshot encryption in one call.

**Why not primary:** Documentation doesn't fix the discoverability problem — consumers don't read every doc section. The `stack` one-liner is good but only helps `stack` users (bank-sync wires the decider directly, not via `stack`).

### Recommendation

**Implement Option A** (`NewEncryptedSnapshotStore`). It's ~80 lines of code (the event `encryptedStore` is the template — same encrypt/decrypt/passthrough pattern, smaller interface surface since `SnapshotStore` has only `Save`/`Delete`/`Load`/`LoadAtVersion`). Ship it alongside a test that verifies:

1. Snapshot state is ciphertext in the DB (raw SQL assertion)
2. Round-trip: Save encrypted → Load → state matches
3. Passthrough: old plaintext snapshots load correctly when key is set (backward compat)
4. Key rotation: snapshots encrypted with key-v1 decrypt when key-v1 is in the resolver

Then update the encryption README to say "events **and snapshots**" and add a warning that enabling event encryption without snapshot encryption leaves aggregate state in plaintext.

---

## 6. The deeper point: encryption should be all-or-nothing

The strongest argument for fixing this isn't the API gap — it's that **the SDK makes it possible to do half.** A well-designed encryption story is atomic: either all persisted state is encrypted, or none is, and the consumer never has to reason about which tables are protected.

Today a consumer can call `NewEncryptedStore(eventStore, cipher)` and feel done. They are not done. The SDK should either:

1. **Make snapshot encryption automatic** when event encryption is enabled (the `stack` layer could do this: `stack.WithEncryption(cipher)` wraps both stores), **or**
2. **Make the gap loud** — `NewEncryptedStore` should log a WARNING if a snapshot store is also in use without encryption, or the docs should have a prominent callout.

Option 1 (automatic) is the ideal. bank-sync would adopt `stack.WithEncryption(cipher)` and never think about snapshots again. But even Option A (explicit `NewEncryptedSnapshotStore`) is a massive improvement over the current silent gap.

---

## 7. What bank-sync will do regardless

Since the SDK fix is upstream and unscheduled, bank-sync will adopt the codec-composition workaround immediately (it works today, verified against source):

```go
// internal/cqrs/infrastructure.go — extend WithEncryption to also encrypt snapshots
if len(cfg.encryptionKey) > 0 {
    cipher, _ := encryption.NewAES256GCM(cfg.encryptionKey)
    eventStore, _ = encryption.NewEncryptedStore(rawEventStore, cipher)
    snapshotCodec = encryption.NewCodec(codec.CBORCodec{}, cipher)  // encrypts snapshot state
    // ... pass snapshotCodec via decider.WithCodec[BalanceSyncState](snapshotCodec)
}
```

This closes the gap for bank-sync. But every other consumer (DiscordSync, swettyswipper, the file renamer) faces the same undiscoverable trap. The SDK fix benefits the whole ecosystem.

---

## Appendix: source evidence

| Claim                                          | Evidence                                                                        |
| ---------------------------------------------- | ------------------------------------------------------------------------------- |
| No `NewEncryptedSnapshotStore`                 | `rg "func New" snapshot/*.go` → only `NewReadPressure`, `NewTypedStore`         |
| `stack` has no encryption                      | `rg "encryption" stack/*.go` → exit 1, no matches                               |
| Decider uses `r.codec` for snapshots           | `decider/decider.go:211` (Encode), `decider/load.go:235` (Decode)               |
| `encryption.NewCodec` implements `codec.Codec` | `encryption/codec.go:39` (Encode), `:67` (Decode)                               |
| Snapshots are plaintext in bank-sync           | `internal/cqrs/infrastructure.go:155` — `WithSnapshotStore` with no `WithCodec` |
| encryption doc.go mentions only events         | `encryption/doc.go` — "event payload encryption and decryption"                 |
| bank-sync shipped event encryption             | commit `4985fb0`, `internal/cqrs/infrastructure.go:88-104`                      |
