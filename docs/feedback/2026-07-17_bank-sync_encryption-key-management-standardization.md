# go-cqrs-lite — Consumer Feedback: Encryption Key Management Standardization

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — bank transaction sync CLI (Wise API + Qonto CSV → SQLite)
**SDK version:** v4.0.x (encryption/v4)
**Date:** 2026-07-17
**Severity:** Medium-High — every consumer reinvents the same key-lifecycle boilerplate, inconsistently
**Status:** Gap verified against source. Proposal below.

---

## TL;DR

The encryption module provides excellent **primitives** (AES-256-GCM, XChaCha20-Poly1305, HKDF derivation, key-ID rotation) but has no opinion on the **key lifecycle**: generation, validation, serialization, persistence, and loading. Every consumer hand-rolls the same `crypto/rand` + `base64` + config-loading dance, and the SDK's guidance collapses to a one-liner in the doc.go (`// key must be 32 bytes`). The result is predictable: consumers make different choices, some insecure, and the SDK can't enforce or even recommend a standard.

This document proposes a tiered standardization: small focused helpers (generate, validate, serialize), a documented key-source taxonomy, and a `KeyProvider` pattern that lets consumers declare their key strategy declaratively instead of imperatively. It also resolves the "auto-generate vs explicit setup" question that bank-sync (and every other local-tool consumer) faces.

---

## 1. The current state — what the SDK provides

| Capability                       | SDK API                                                        | Status                                                                    |
| -------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Create cipher from raw key       | `NewAES256GCM(key []byte)`, `NewXChaCha20Poly1305(key []byte)` | ✅ Provided                                                               |
| Derive key from master (HKDF)    | `DeriveKey(masterKey, info, length)`                           | ✅ Provided                                                               |
| Key rotation (events)            | `WithMiddlewareKeyID`, `NewStaticKeyResolver`                  | ✅ Provided                                                               |
| Multi-key resolution             | `StaticKeyResolver`, `KeyResolver` interface                   | ✅ Provided                                                               |
| **Generate a random key**        | —                                                              | ❌ **Missing**                                                            |
| **Validate key length/format**   | —                                                              | ❌ **Missing** (only `ErrInvalidKey` on empty, inside cipher constructor) |
| **Serialize key for config/env** | —                                                              | ❌ **Missing**                                                            |
| **Load key from env/file**       | —                                                              | ❌ **Missing**                                                            |
| **Persist key (OS keychain)**    | —                                                              | ❌ **Missing** (explicitly punted in doc.go)                              |
| **Passphrase → key (argon2id)**  | —                                                              | ❌ **Missing** (explicitly punted in doc.go)                              |

The doc.go is candid about the two intentional omissions:

> This library does NOT implement argon2id because passphrase-based key derivation is a client-side concern, not a server-side library concern.

That boundary is reasonable for argon2id (opinionated, memory-hard, parameter-heavy). But the **same boundary is not drawn for key generation** — and key generation is unambiguously a library concern. Today every consumer writes:

```go
key := make([]byte, 32)
if _, err := crypto_rand.Read(key); err != nil { ... }
```

…often wrong (forgotten error check, wrong length, copied from a Stack Overflow answer). This is the cryptographic equivalent of telling consumers "implement your own hash function" — the primitive is trivial to get subtly wrong.

---

## 2. What every consumer hand-rolls — bank-sync as the concrete example

bank-sync's key lifecycle is spread across three files with no SDK help:

### Generation (told to the user, not provided by SDK)

```go
// pkg/config/config.go:68
// EncryptionKey is a base64-encoded 32-byte key for AES-256-GCM event payload
// encryption. Generate one with: openssl rand -base64 32
EncryptionKey string `koanf:"encryption_key"`
```

The user is told to run `openssl rand -base64 32` in their shell. The SDK has `crypto/rand` imported in five files but exports no `GenerateKey`. The consumer documents a shell command instead of calling a function.

### Serialization (hand-rolled base64)

```go
// cmd/bank-sync/helpers.go:151
if cc.Config.Security.EncryptionKey != "" {
    key, err := base64.StdEncoding.DecodeString(cc.Config.Security.EncryptionKey)
    if err != nil {
        return buildError("decode encryption key", err, nil)
    }
    opts = append(opts, cqrs.WithEncryption(key))
}
```

Standard library `base64`. Works, but every consumer writes the same 4 lines with the same error message. A malformed key produces a generic "decode encryption key" error instead of "encryption key must be base64-encoded 32 bytes, got N bytes."

### Validation (implicit, late, unhelpful)

There is no explicit validation. The key reaches `encryption.NewAES256GCM(key)` which calls `cipher.NewGCM(aes.NewCipher(key))`. `aes.NewCipher` panics on wrong length on some code paths and returns an error on others. The consumer sees a low-level crypto error, not "your encryption key is 24 bytes; AES-256 requires exactly 32."

### Persistence (not addressed)

The key lives wherever the user puts it — env var, config file, or nowhere (unencrypted). The SDK offers no guidance on where a local tool should store a key such that it survives restarts but isn't trivially co-located with the ciphertext.

---

## 3. The auto-generate vs explicit-setup question — analyzed

This is the question bank-sync faces: **auto-generate a key on first run (store in `~/.config/bank-sync/key`) or require explicit setup (`openssl rand -base64 32` + env var)?**

Neither answer is correct without understanding **what threat the key is supposed to mitigate.** Let's reason from first principles.

### The key's job

An encryption key protects data-at-rest against an attacker who has the storage medium (the SQLite file) but not the key. The key's value comes entirely from **separation from the ciphertext.** If the key and the ciphertext live on the same disk and the attacker has the disk, the encryption is theater.

### The key-source strategies

| Strategy                        | Key location                                          | Disk theft                             | Friction | When appropriate              |
| ------------------------------- | ----------------------------------------------------- | -------------------------------------- | -------- | ----------------------------- |
| **Machine-bound (OS keychain)** | macOS Keychain / Linux secret-service / Windows DPAPI | ✅ Protected                           | **Zero** | Local tools, CLI apps (ideal) |
| **Derived key (HKDF)**          | Derived from master at runtime                        | ✅ Protected                           | Medium   | Multi-tenant, KMS-backed      |
| **Passphrase (argon2id)**       | User's memory                                         | ✅ Protected                           | Medium   | High-security, human-operated |
| **Raw key (env var)**           | Env var                                               | ⚠️ Weak — leaks to all child processes | Low      | CI/automation only (NOT prod) |
| **Raw key (separated file)**    | Config file with `chmod 600`, separate volume         | Medium (if separated)                  | Low      | Server, headless fallback     |
| **Auto-gen on disk**            | `~/.config/app/key` (plaintext file)                  | ❌ **Useless**                         | Zero     | **Never** (theater)           |

### Why environment variables are a weak key source

Env vars are convenient but **insecure for encryption keys.** They leak through multiple attack vectors:

- **`/proc/<pid>/environ`** — any process the user runs can read ALL env vars. Malware, a malicious npm package, even a poorly-written shell script gets the key for free. No privilege escalation needed.
- **Child process inheritance** — every subprocess bank-sync spawns inherits the key automatically, widening the exposure surface.
- **Shell history** — `export BANK_SYNC_ENCRYPTION_KEY=...` persists forever in `.bash_history` / `.zshhistory`.
- **Crash dumps / `gdb attach`** — `show environ` or core dumps expose the key to any debugger.
- **Container inspection** — `docker inspect`, `kubectl describe pod` — visible to anyone with cluster read access.
- **Systemd units / `.env` files** — stored in plaintext wherever the process is launched from.

For a **financial tool** where the key protects transaction data, "any process the user runs can read the key" is a real problem. Env vars are acceptable for CI/automation where the environment is ephemeral and controlled. They are **not** acceptable as the primary key source for a local tool running on a developer's laptop.

### The trap of "auto-generate to disk"

Auto-generating a key and storing it in `~/.config/bank-sync/key` is the obvious "encryption by default" play. It is also **security theater** for the disk-theft threat model:

- The key file and the SQLite file are on the same disk.
- An attacker who steals the laptop (or copies the backup) gets both files.
- The encryption provides zero confidentiality against the most likely attacker.
- It adds real risk: **key loss = permanent data loss**, with no security benefit to offset it.

This is strictly worse than plaintext. Plaintext has no key to lose. Auto-gen-on-disk has a key to lose AND provides no protection. It is a pure downside.

The only scenario where auto-gen-on-disk helps is "the attacker has the SQLite file but not the rest of the filesystem" — e.g., a partial backup leak, a database export shared accidentally. That's a narrow threat model and not one worth the data-loss risk.

### The real "encryption by default": machine-bound keys

If the goal is zero-friction encryption that actually protects against disk theft, the answer is **OS keychain integration:**

- **macOS:** Keychain Services (`security add-generic-password`)
- **Linux:** Secret Service API (GNOME Keyring / KDE Wallet) via `dbus`
- **Windows:** Data Protection API (DPAPI) via `cryptoProtectData`

The key is generated once, stored in the OS keychain (bound to the user session, encrypted by the OS), and retrieved at startup. An attacker with the disk but not the running user session cannot decrypt it. This is how 1Password, Chrome, and every modern local app handle "encryption by default."

**This is a consumer concern, not an SDK concern** — the SDK shouldn't depend on CGO or platform-specific APIs. But the SDK CAN provide the primitives that make it trivial: `GenerateKey()`, and a documented "KeyProvider" pattern where the consumer plugs in their keychain integration.

### Resolution for bank-sync

| Option                | Threat mitigated              | Key-loss risk           | Friction | Verdict                                                 |
| --------------------- | ----------------------------- | ----------------------- | -------- | ------------------------------------------------------- |
| OS keychain           | Disk theft                    | Low (OS-managed)        | Zero     | ✅✅ **Correct default for a local tool**               |
| Passphrase (argon2id) | Disk theft                    | Low (user memorizes)    | Medium   | ✅ High-security option                                 |
| Explicit env var      | Disk theft (if env separated) | Low (user chose)        | Low      | ⚠️ CI/headless fallback only — leaks to child processes |
| Auto-gen to disk      | Partial-file leak only        | **High** (catastrophic) | Zero     | ❌ Theater + data-loss risk                             |

**The correct default for bank-sync is OS keychain integration**, not env vars.

bank-sync's **current** approach (explicit env var) was originally endorsed as "the correct default for a financial tool." That was **wrong.** Env vars leak to every child process via `/proc/<pid>/environ` — for a financial tool, "any process can read the key" is unacceptable. The env-var path should remain as a **CI/headless fallback** for automation where the environment is ephemeral.

The ideal path is OS keychain (`KeyProvider` implemented consumer-side using `security`/`secret-tool`/DPAPI): zero friction after setup, disk-theft safe, and the key never touches the filesystem in plaintext. This is how Chrome, 1Password, and every modern local app handle "encryption by default."

The SDK's job is to make the keychain path as smooth as possible by providing the `KeyProvider` interface + `GenerateKey()` primitives. The consumer implements the platform-specific keychain access — the SDK shouldn't depend on CGO/dbus/Win32.

---

## 4. Proposed standardization — tiered, additive

### Tier 1: Key generation + validation helpers (low effort, high value)

These belong in the `encryption` package. They eliminate hand-rolled `crypto/rand` across every consumer.

```go
// encryption/key.go (new file)

// KeySize is the required key length for all supported algorithms (AES-256-GCM,
// XChaCha20-Poly1305). Both use 256-bit keys.
const KeySize = 32

// GenerateKey returns a cryptographically random KeySize-byte key suitable
// for NewAES256GCM or NewXChaCha20Poly1305. Uses crypto/rand.
func GenerateKey() ([]byte, error)

// GenerateKeyBase64 returns GenerateKey() encoded as base64 standard string,
// suitable for storage in environment variables or config files.
func GenerateKeyBase64() (string, error)

// ValidateKey returns an error if key is not exactly KeySize bytes.
// The error is classified as Rejection with code "encryption.invalid_key_length"
// and a human-readable message including the actual vs expected length.
func ValidateKey(key []byte) error

// DecodeKeyBase64 decodes a base64-encoded key and validates its length.
// Returns a clear error if the input is malformed or the wrong length.
// This is the single entry point for loading a key from config/env.
func DecodeKeyBase64(s string) ([]byte, error)
```

**bank-sync adoption (before / after):**

```go
// BEFORE — cmd/bank-sync/helpers.go (4 lines, generic error)
key, err := base64.StdEncoding.DecodeString(cc.Config.Security.EncryptionKey)
if err != nil {
    return buildError("decode encryption key", err, nil)
}

// AFTER — 1 line, clear error
key, err := encryption.DecodeKeyBase64(cc.Config.Security.EncryptionKey)
```

The config comment changes from "Generate one with: `openssl rand -base64 32`" to a reference to a programmatic generator, or stays as-is for shell users (both paths now work).

### Tier 2: KeyProvider pattern (medium effort, declarative key sources)

A small interface that lets consumers declare their key strategy instead of imperatively loading it. This standardizes the "where does the key come from?" question across consumers.

```go
// encryption/keyprovider.go (new file)

// KeyProvider supplies an encryption key and its identifier. Implementations
// decide the key source: environment variable, config file, OS keychain,
// KMS, HKDF derivation, etc.
//
// KeyProvider is the single integration point between a consumer's
// configuration system and the encryption module. Consumers configure
// a KeyProvider once at startup; the encryption module never needs to
// know where the key came from.
type KeyProvider interface {
    // Key returns the encryption key. Called once at startup.
    // Returns ErrNoKey (a sentinel) if no key is configured — the
    // consumer decides whether that means "run unencrypted" or "fail".
    Key() (key []byte, err error)
}
```

Built-in providers (the SDK provides the common cases; consumers add their own):

```go
// From environment variable — dev/test/CI convenience ONLY.
// Env vars leak to all child processes via /proc/<pid>/environ and
// persist in shell history. Do NOT use as the primary key source in production.
func EnvKeyProvider(varName string) KeyProvider

// From raw bytes (already decoded — for programmatic/testing use)
func StaticKeyProvider(key []byte) KeyProvider

// No key configured (returns ErrNoKey) — explicit "encryption off"
func NoKeyProvider() KeyProvider
```

Consumers with keychain integration implement `KeyProvider` themselves (the SDK doesn't depend on platform APIs):

```go
// consumer-side: macOS Keychain provider
type keychainKeyProvider struct{ account string }
func (k keychainKeyProvider) Key() ([]byte, error) {
    // call security find-generic-password ...
}
```

**Why an interface and not just functions:** The interface lets the `stack` layer accept a single `stack.WithEncryption(provider encryption.KeyProvider)` option that handles the full lifecycle (generate-if-needed, validate, wire into event + snapshot stores). Without it, every consumer's "enable encryption" code is bespoke.

**Why not over-engineer:** The interface has one method. No `KeyID`, no rotation, no async refresh — those are event-middleware concerns (`WithMiddlewareKeyID`, `StaticKeyResolver`) that compose on top. `KeyProvider` is purely "give me the bytes."

### Tier 3: Documented key-source taxonomy (zero code, high clarity)

Update `encryption/doc.go` with a "Key Sources" section that replaces the implicit "figure it out" guidance:

```
# Key Sources

Choose a key source based on your deployment:

| Source        | Generate                      | Store                         | Disk-theft safe |
|---------------|-------------------------------|-------------------------------|-----------------|
| Environment   | encryption.GenerateKeyBase64() | $APP_ENCRYPTION_KEY         | If env not on disk |
| Config file   | encryption.GenerateKeyBase64() | YAML/TOML key field         | If config separated |
| OS keychain   | encryption.GenerateKey()      | Keychain/DPAPI/secret-service| Yes             |
| HKDF derived  | encryption.DeriveKey()        | Master key in KMS/vault      | Yes             |
| Passphrase    | (argon2id — consumer-side)    | User memory                   | Yes             |

NEVER auto-generate a key and store it as plaintext on the same disk as
the encrypted data — this provides no protection against disk theft and
introduces unrecoverable data-loss risk if the key file is lost.
```

This single paragraph would have saved bank-sync from even considering auto-generate-on-disk. It turns an implicit footgun into an explicit warning.

---

## 5. Why the SDK should own this (and not leave it to consumers)

The argument against SDK key management is "the key source is deployment-specific." That's true for **where** the key is stored (keychain vs env vs KMS). It is NOT true for:

1. **Generation** — `crypto/rand.Read(make([]byte, 32))` is identical in every consumer. There is no deployment-specific decision. Not providing `GenerateKey()` forces every consumer to write the same 3 lines and get them wrong in the same ways.

2. **Validation** — "is this key the right length for the cipher?" is a property of the cipher, not the deployment. Today validation happens implicitly inside `aes.NewCipher` and produces unhelpful errors. `ValidateKey` turns a crypto-layer panic into a clear Rejection error.

3. **Serialization** — base64 is the universal key transport format. `DecodeKeyBase64` with a clear error message is identical in every consumer.

4. **The taxonomy** — "don't store the key next to the ciphertext" is universal advice. Leaving it undocumented means every consumer rediscovers it (or doesn't).

The deployment-specific part (keychain vs env vs KMS) is exactly what `KeyProvider` abstracts — the consumer plugs in their source, the SDK handles the rest.

---

## 6. What NOT to standardize (scope boundaries)

To avoid overreach, these stay consumer-side:

| Concern                 | Why it's consumer-side                                                                                                |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------- |
| argon2id passphrase KDF | Memory-hard, parameter-heavy, opinionated. Correctly punted in current doc.go.                                        |
| OS keychain access      | Platform-specific (CGO / dbus / Win32). SDK shouldn't depend on these.                                                |
| Key backup/escrow       | Operational policy, not a library concern.                                                                            |
| Key rotation scheduling | Business logic. The SDK provides the rotation primitive (`StaticKeyResolver`); when to rotate is the consumer's call. |

The line: the SDK owns **key bytes** (generate, validate, serialize, provide). The consumer owns **key location** (where the bytes live when the app isn't running).

---

## 7. Recommendation — phased adoption

### Phase 1 (ship now, ~30 min): Tier 1 helpers

Add `GenerateKey()`, `GenerateKeyBase64()`, `ValidateKey()`, `DecodeKeyBase64()` to the encryption package. These are pure functions with no dependencies beyond `crypto/rand` and `encoding/base64` (both already imported elsewhere in the package). Immediate adoption by bank-sync replaces 4 lines with 1 and improves error messages.

### Phase 2 (next minor): Tier 2 KeyProvider + stack integration

Add the `KeyProvider` interface + built-in providers. Wire `stack.WithEncryption(provider)` to handle the full event + snapshot encryption lifecycle (pending the snapshot encryption fix from the companion feedback doc). This is the "encryption by default done right" path — consumers configure a provider, the stack wires both stores.

### Phase 3 (docs only): Tier 3 taxonomy

Update `encryption/doc.go` and `encryption/README.md` with the key-source table and the "never auto-gen to disk" warning. Zero code, maximum clarity.

---

## 8. Cross-consumer impact

This is not a bank-sync-specific concern. Every consumer in the ecosystem faces the same gap:

| Consumer               | Key handling today                                  | What they hand-roll                     |
| ---------------------- | --------------------------------------------------- | --------------------------------------- |
| bank-sync              | `openssl rand -base64 32` + env var + base64 decode | Generation guidance, decode, validation |
| DiscordSync            | (not yet adopted)                                   | Would face the same decisions           |
| swettyswipper          | (not yet adopted)                                   | Same                                    |
| file-and-image-renamer | (not yet adopted)                                   | Same                                    |

Standardizing in the SDK means the next consumer to adopt encryption gets `GenerateKeyBase64()` + `DecodeKeyBase64()` + `EnvKeyProvider("APP_KEY")` instead of reinventing the wheel. The consolidation also means a future security audit covers one implementation, not N.

---

## Appendix A: bank-sync's current key lifecycle (full trace)

```
User runs:        openssl rand -base64 32
User sets:        export BANK_SYNC_SECURITY_ENCRYPTION_KEY=<key>
                  (or config file: security.encryption_key: <key>)

Startup:          config.Load() → koanf unmarshal → Config.Security.EncryptionKey (string)
                  helpers.go:151 → base64.StdEncoding.DecodeString(key)
                  helpers.go:157 → cqrs.WithEncryption(key []byte)
                  infrastructure.go:88 → if len(key) > 0
                  infrastructure.go:89 → encryption.NewAES256GCM(key)
                  infrastructure.go:94 → encryption.NewEncryptedStore(store, cipher)

Gaps:             No validation (wrong length → late crypto error)
                  No generation helper (shell command in a comment)
                  No standard loading (hand-rolled base64)
                  No key-source guidance (auto-gen vs explicit is unresolved)
```

## Appendix B: the `openssl` dependency problem

Telling users to run `openssl rand -base64 32` introduces an external dependency for a task the SDK's own process could perform. Not every deployment has `openssl` in PATH (minimal containers, Windows without Git Bash, restricted shells). `encryption.GenerateKeyBase64()` eliminates the dependency and makes key generation testable, scriptable, and embeddable in a `bank-sync init` command.

---

_Companion document: [2026-07-17_bank-sync_snapshot-encryption-gap.md](./2026-07-17_bank-sync_snapshot-encryption-gap.md) — the other half of the encryption completeness story._
