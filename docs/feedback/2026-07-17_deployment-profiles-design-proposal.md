# Design Proposal: Deployment Profiles for go-cqrs-lite + cqrs-lint

**From:** bank-sync consumer (second consumer to provide cqrs-lint feedback)
**Date:** 2026-07-17
**Scope:** SDK-side and linter-side changes, proposed together because they reinforce each other

---

## The Problem

go-cqrs-lite is versatile by design. The same library powers:

| Consumer    | Deployment | Concurrency | Data Sensitivity | Write Pattern           |
| ----------- | ---------- | ----------- | ---------------- | ----------------------- |
| bank-sync   | Local CLI  | Single-user | PII (financial)  | Scheduled batch sync    |
| DiscordSync | Server     | Multi-user  | Messages         | Real-time event capture |
| Example TM  | Local dev  | Single-user | None             | Interactive commands    |

But cqrs-lint treats every consumer identically. The result, confirmed across **two independent consumers** (DiscordSync feedback 2026-07-16 + bank-sync feedback 2026-07-17):

- **A016** (idempotency) fires on both DiscordSync (no command dispatcher) and bank-sync (read-only dashboard). Both false positives.
- **D005** (doc version) fires on both. Both false positives (empty token extraction / historical ADR title).
- **A009** (no stack preset) fires on both. Both intentional (shared DB / custom wiring).

These repeat across consumers because the linter has no concept of **what kind of system** it's analyzing. It checks the same rules regardless of whether the deployment is a local CLI or a multi-user server.

**The encryption gap is real and separate.** Both consumers store PII without encryption. S002 fires correctly on both. The issue isn't that the linter is wrong — it's that the SDK makes encryption hard enough to skip. More on that below.

---

## Part 1: SDK Changes — Make the Right Thing Easy

### 1.1 `stack.WithEncryption` — One-liner encryption wiring

**Problem:** Today, wiring encryption requires 5 manual steps:

1. Generate/load a 32-byte key (where? how? env var? file?)
2. `encryption.NewAES256GCM(key)`
3. `encryption.EncryptMiddleware(encrypter)` on `bus.UsePublish(...)`
4. `encryption.DecryptMiddleware(decrypter)` on `bus.Use(...)`
5. Key rotation: build a `StaticKeyResolver`, wire it into the decrypter

bank-sync doesn't do this because it's 5 steps of plumbing for a local CLI. DiscordSync doesn't do it either. **If encryption were a one-liner, both would have it.**

**Proposal:** Add `stack.WithEncryption` to the stack options:

```go
// Option A: Key from environment variable (production-recommended)
bundle, err := sqlite.New("bank.db",
    stack.WithEncryptionFromEnv("BANK_SYNC_ENCRYPTION_KEY"),
)

// Option B: Key from explicit bytes (testing, local tools with generated key file)
bundle, err := sqlite.New("bank.db",
    stack.WithEncryption(keyBytes),
)

// Option C: Key rotation with a resolver (advanced)
bundle, err := sqlite.New("bank.db",
    stack.WithEncryptionResolver(resolver),
)
```

`WithEncryptionFromEnv` reads a base64-encoded 32-byte key from an environment variable. If the variable is absent, it returns a clear error:

```
encryption key not found: set BANK_SYNC_ENCRYPTION_KEY to a base64-encoded
32-byte key. Generate one with: openssl rand -base64 32
```

The stack internally wires `EncryptMiddleware` on publish and `DecryptMiddleware` on subscribe. The consumer never touches middleware directly.

**Why this matters:** S002 (encryption) fires on every consumer with PII. The SDK should make compliance trivial, not something each consumer must figure out from scratch. This is the "pit of success" principle — the easiest path should be the secure path.

### 1.2 `stack.WithSigning` — One-liner tamper detection

Same pattern for `signing.SignMiddleware`. Currently requires manual key generation + middleware wiring. A `WithSigningFromEnv("BANK_SYNC_SIGNING_KEY")` option would make event integrity verification trivial.

For local CLIs this is lower priority (single machine, no shared store). For server deployments it's essential. The linter's S003 finding is correct for servers.

### 1.3 `stack.WithObservability` — One-liner OTel wiring

`middleware.NewOTelBundle` already exists but requires manual spreading into 4 middleware chains. Bundle it into the stack:

```go
bundle, err := sqlite.New("bank.db",
    stack.WithObservability(otelTracer, otelMeter), // wires all 4 chains
)
```

This eliminates B014 (missing OTel) as a finding for any stack consumer.

### 1.4 The missing `SecurityBundle`

The encryption + signing + observability one-liners are individual options. But they share a theme: **security and operational middleware that every production deployment should have.** Consider a higher-level bundle:

```go
type SecurityConfig struct {
    EncryptionKey  []byte    // required if DataSensitivity != "none"
    SigningKey     []byte    // required for multi-user or shared stores
    Tracer         trace.Tracer
    Meter          metric.Meter
}

bundle, err := sqlite.New("bank.db",
    stack.WithSecurity(stack.SecurityConfig{
        EncryptionKey: keyFromEnv("ENCRYPTION_KEY"),
        Tracer:        otelTracer,
    }),
)
```

`WithSecurity` wires encryption, signing (if key present), and OTel (if tracer present) in the recommended order. Missing required fields produce clear errors with remediation instructions.

---

## Part 2: Linter Changes — Deployment-Aware Rules

### 2.1 The `.cqrs-lint.json` profile system

**Current state:** `.cqrs-lint.json` supports `format`, `min-severity`, `min-confidence`, `fast`. No deployment context.

**Proposal:** Add a `deployment` section:

```json
{
  "deployment": {
    "kind": "local-cli", // local-cli | server | library | batch-job
    "concurrency": "single-user", // single-user | multi-user
    "data": "pii", // none | internal | pii | financial
    "writes": "sync", // none | sync | user-commands | real-time
    "store": "sqlite" // sqlite | postgres | pebble | in-memory
  },
  "rules": {
    "exclude": ["B005"] // per-rule overrides still allowed
  }
}
```

Each field is optional. Missing fields default to "unknown" (no suppression).

### 2.2 How declarations affect rules

**Core principle: declarations can only suppress rules, never enable them.** This prevents consumers from gaming the config for a clean score. Rules with unconditional applicability (like S002) fire regardless of deployment context.

| Rule                      | Default          | Suppressed when...                                           | Rationale                                                                                                |
| ------------------------- | ---------------- | ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| **S002** (PII encryption) | **always fires** | never — cannot be suppressed via deployment                  | No excuse for unencrypted PII. Period.                                                                   |
| S003 (event signing)      | fires            | `kind: local-cli` AND `concurrency: single-user`             | Local single-user stores don't need tamper detection                                                     |
| A016 (idempotency)        | fires            | `writes: none` or `writes: sync`                             | Read-only and domain-dedup systems don't need idempotency middleware                                     |
| A015 (global mutable)     | fires            | `concurrency: single-user` AND write-after-init check passes | Single-user tools with init-only globals are safe                                                        |
| B014 (OTel middleware)    | fires            | `kind: local-cli` or `kind: library`                         | Local tools and libraries gain nothing from distributed tracing                                          |
| A012 (tombstone)          | fires            | no tombstone-like event types exist in the registry          | Don't flag if the domain has no delete concept                                                           |
| A009 (no stack preset)    | fires            | never suppressed, but suggestion adapts                      | If `kind: server`, suggest `stack/sqlite.New`. If `kind: local-cli`, acknowledge custom wiring is common |
| D005 (doc version)        | fires            | version token is wildcard (`v4.x`) or inside ADR title       | Historical doc references are not version claims                                                         |

**The non-suppressible rules** (the "hard floor"):

- S001 (hardcoded secrets) — always fires
- S002 (PII without encryption) — always fires when `data: pii` or `data: financial`
- C001-C012 (correctness rules) — always fire
- E001, E002 (layer violations, circular deps) — always fire

These are correctness and security rules where context is never an excuse.

### 2.3 Auto-detection as fallback

Consumers who don't write a `.cqrs-lint.json` get auto-detection heuristics:

| Property      | Detection heuristic                                                                                                                    |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `kind`        | `net/http` import + `ListenAndServe` call → `server`. `os/exec` or `cobra.Command` root → `local-cli`. No `main` package → `library`   |
| `concurrency` | `net/http.Server` or `http.ListenAndServe` → `multi-user`. Otherwise → `single-user`                                                   |
| `data`        | Scan event payload structs for field names matching PII patterns (`email`, `ssn`, `phone`, `address`, `iban`, `bic`). If found → `pii` |
| `writes`      | `command.Dispatcher` with `Dispatch()` calls → `user-commands`. Only `event.New` in batch loops → `sync`. No writes at all → `none`    |
| `store`       | `import modernc.org/sqlite` → `sqlite`. `import lib/pq` or `pgx` → `postgres`. `import pebble` → `pebble`                              |

Auto-detection runs first; `.cqrs-lint.json` declarations override. This means **zero-config consumers get better results immediately** without writing config, and config-declaring consumers get precise control.

### 2.4 The E005/E007 closure bug (still the #1 priority)

Profile-aware rules don't help if the detector can't trace handler registration through closures. This is the single biggest noise source across **both** consumers (19 FPs in bank-sync, multiple in DiscordSync). The fix is in `handlerTypeFromCall` (`scanner_calls.go:63`):

```go
// Current: only finds CompositeLit args
// Fix: also extract type from FuncLit parameter signatures
case *ast.FuncLit:
    if a.Type != nil && a.Type.Params != nil {
        for _, param := range a.Type.Params.List {
            // The handler closure's 2nd param is always *T where T is the command/query type
            if star, ok := param.Type.(*ast.StarExpr); ok {
                if id, ok := star.X.(*ast.Ident); ok {
                    return id.Name
                }
            }
        }
    }
```

This handles the canonical `RegisterTyped(d, type, func(ctx, c *MyCommand) error {...})` pattern that **both** consumers use. Without this fix, E005/E007 are unusable on any real codebase.

### 2.5 Generic call unwrapping (systemic fix)

`WithSnapshotStore[T](store)` is `*ast.IndexExpr(SelectorExpr)`, not `*ast.SelectorExpr`. Every detector that does `call.Fun.(*ast.SelectorExpr)` is blind to generic API calls. A shared helper:

```go
func unwrapSelector(expr ast.Expr) *ast.SelectorExpr {
    switch e := expr.(type) {
    case *ast.SelectorExpr:   return e
    case *ast.IndexExpr:      return unwrapSelector(e.X)   // X[T]
    case *ast.IndexListExpr:  return unwrapSelector(e.X)   // X[T, U]
    default:                  return nil
    }
}
```

Apply this across ALL detectors. This is a one-time fix that prevents an entire class of false positives.

---

## Part 3: The Encryption Argument

### Why "local tool" is not an excuse

I initially dismissed S002 for bank-sync as "local single-user tool." I was wrong.

| Risk              | Scenario                                                               |
| ----------------- | ---------------------------------------------------------------------- |
| Laptop theft      | SQLite file with plaintext emails, names, financial transactions       |
| Cloud backup sync | iCloud/Google Drive/Dropbox backs up the `.db` file to a shared folder |
| Screen sharing    | Demo or pair programming exposes raw data in DB viewer                 |
| Accidental commit | `.db` file accidentally committed to git (happens regularly)           |
| Shared filesystem | NFS/SMB mount on a home server                                         |

**Defense in depth** means encrypting at the application layer even when you think the OS layer protects you. The library ships `encryption.EncryptMiddleware` for exactly this reason.

### What the SDK should do

The encryption gap is not a linter problem — it's an **SDK ergonomics problem.** If wiring encryption were `stack.WithEncryptionFromEnv("KEY")` (one line), no consumer would skip it. The current 5-step manual wiring is the barrier.

**Action items for the SDK:**

1. Ship `stack.WithEncryption` / `stack.WithEncryptionFromEnv` (Part 1.1)
2. Document key generation in the stack preset quickstart (not buried in the encryption module)
3. Make `cqrs-lint init` generate a `.cqrs-lint.json` with `"data": "pii"` if PII fields are detected, so S002 fires from day one

**Action items for bank-sync (the consumer):**

1. Generate an encryption key on first run (store in `~/.config/bank-sync/key`)
2. Wire `EncryptMiddleware` + `DecryptMiddleware` on the event bus
3. Document that the key file must be backed up (data is unrecoverable without it)

---

## Part 4: Cross-Consumer Pattern Analysis

Data from two consumers (bank-sync + DiscordSync) reveals which rules provide value and which are noise:

| Rule                   | bank-sync                  | DiscordSync            | Cross-consumer verdict                     |
| ---------------------- | -------------------------- | ---------------------- | ------------------------------------------ |
| C009 (panic)           | Intentional (`must*`)      | **Valid (3 fixed)**    | Keep, but recognize `must*` pattern        |
| C001 (tx commit)       | Not fired                  | **FP (dangerous)**     | Fix: trace closures                        |
| C005 (raw json)        | Intentional (upcaster)     | Not fired              | Fix: detect upcaster context               |
| C008 (float money)     | Not fired                  | **FP** (2 sites)       | Fix: require multiple money signals        |
| A005 (manual proj)     | Not fired                  | **FP** (2 sites)       | Fix: distinguish projection from pub-sub   |
| A009 (no stack)        | Intentional                | Intentional            | Profile-aware suggestion, don't drop       |
| A012 (tombstone)       | Not applicable             | Not fired              | Auto-detect tombstone events               |
| A014 (deprecated API)  | Intentional (upcaster)     | Not fired              | Fix: detect upcaster context               |
| A015 (global mutable)  | Intentional (read-only)    | Not fired              | Fix: check write-after-init                |
| A016 (idempotency)     | Intentional (read-only)    | **FP** (no dispatcher) | Fix: verify dispatcher type; profile-aware |
| A017 (no snapshot)     | **FP** (generic blindness) | Not fired              | Fix: unwrap generic selectors              |
| B014 (OTel)            | Intentional (noop tracer)  | Not fired              | Profile-aware: skip for local-cli          |
| D005 (doc version)     | **FP** (2 sites)           | **FP** (2 sites)       | Fix: wildcard handling, skip ADR titles    |
| E005/E007 (no handler) | **FP** (19 sites)          | Not fired              | **Critical fix:** closure tracing          |
| E006 (no projection)   | Not fired                  | **FP** (1 site)        | Fix: cross-reference event registry        |
| S002 (PII encryption)  | **Valid** (skipped)        | Not fired              | **SDK fix:** make encryption trivial       |

**Pattern:** A016, D005, A009 fire on BOTH consumers and are ALWAYS false positives or intentional. These rules need either detector fixes or profile-aware suppression.

---

## Part 5: Priority Matrix

### SDK changes (go-cqrs-lite)

| Priority | Change                                             | Impact                                       | Effort |
| -------- | -------------------------------------------------- | -------------------------------------------- | ------ |
| **P0**   | `stack.WithEncryptionFromEnv`                      | Every PII consumer gets encryption trivially | Medium |
| **P1**   | `stack.WithSigning` / `WithSigningFromEnv`         | Tamper detection one-liner                   | Medium |
| **P1**   | `stack.WithObservability`                          | OTel one-liner, eliminates B014              | Low    |
| **P2**   | `stack.WithSecurity(SecurityConfig)`               | Unified security wiring                      | Medium |
| **P2**   | Key generation helper (`encryption.GenerateKey()`) | First-run UX                                 | Low    |

### Linter changes (cqrs-lint)

| Priority | Change                                                    | Impact                                           | Effort |
| -------- | --------------------------------------------------------- | ------------------------------------------------ | ------ |
| **P0**   | E005/E007 closure tracing                                 | Eliminates 19+ FPs across consumers              | Medium |
| **P0**   | Generic call unwrapping (`unwrapSelector`)                | Eliminates A017 + prevents systemic class of FPs | Low    |
| **P1**   | `.cqrs-lint.json` deployment section                      | Profile-aware rule suppression                   | Medium |
| **P1**   | Auto-detection heuristics                                 | Zero-config improvement                          | Medium |
| **P1**   | D005 wildcard + ADR title handling                        | Eliminates repeat FP across consumers            | Low    |
| **P2**   | E005 `Type()` over-match fix (require BasicCommand embed) | Eliminates pflag FP                              | Low    |
| **P2**   | C009 `must*` pattern recognition                          | Eliminates intentional-pattern FP                | Low    |
| **P2**   | C005/A014 upcaster context detection                      | Eliminates upcaster FPs                          | Medium |
| **P2**   | A015 write-after-init detection                           | Distinguishes safe globals                       | Medium |
| **P2**   | A016 dispatcher-type verification + write-pattern check   | Eliminates cross-consumer FP                     | Medium |
| **P3**   | S002 confidence boost for detected PII fields             | Makes encryption finding more precise            | Low    |

---

## Part 6: Concrete `.cqrs-lint.json` Schema

```json
{
  "$schema": "https://larsartmann.dev/go-cqrs-lite/cqrs-lint.schema.json",
  "format": "text",
  "min-severity": "info",
  "min-confidence": "low",

  "deployment": {
    "kind": "local-cli",
    "concurrency": "single-user",
    "data": "pii",
    "writes": "sync",
    "store": "sqlite"
  },

  "rules": {
    "exclude": ["B005"],
    "S002": { "confidence": "high" }
  }
}
```

- `deployment` provides context. Auto-detected if absent.
- `rules.exclude` is an explicit escape hatch (always available).
- Per-rule overrides (like `S002.confidence`) allow fine-tuning without deployment context.
- `S002` cannot be excluded via `rules.exclude` when `deployment.data` is `pii` or `financial` — the schema validator rejects it.

### What this config produces for bank-sync

Before (current, no config): **39 findings, 23 false positives, signal-to-noise 39%**

After (with deployment profile + detector fixes):

- E005/E007 (19 FPs) → eliminated by closure tracing fix
- A017 (1 FP) → eliminated by generic unwrapping fix
- D005 (2 FPs) → eliminated by wildcard/ADR handling fix
- A016 (1 intentional) → suppressed by `writes: sync`
- A015 (1 intentional) → suppressed by `concurrency: single-user` + write check
- B014 (1 intentional) → suppressed by `kind: local-cli`
- **S002 stays** → valid, drives the encryption adoption
- **C009 stays** → valid, `must*` fix downgrades to INFO with explanation
- Remaining: **~8 findings, 0 false positives, signal-to-noise ~100%**

That's the goal.
