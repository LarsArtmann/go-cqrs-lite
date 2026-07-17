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

### 1.1 `stack.WithEncryption` — One-liner encryption wiring (revised: KeyProvider-based)

> **Correction (2026-07-17):** The original proposal used `WithEncryptionFromEnv("KEY")` as the
> primary API. That was wrong — **environment variables are insecure for encryption keys.**
> They leak to every child process via `/proc/<pid>/environ`, persist in shell history, appear
> in crash dumps and `docker inspect`, and are readable by any process the user runs. Baking
> an insecure source into the API name as the recommended default is misleading.
>
> The revised proposal uses a `KeyProvider` interface (defined in
> [2026-07-17_bank-sync_encryption-key-management-standardization.md](./2026-07-17_bank-sync_encryption-key-management-standardization.md))
> as the canonical abstraction. `EnvKeyProvider` still exists as a convenience but carries a
> doc-comment warning about env-var exposure.

**Problem:** Today, wiring encryption requires 5 manual steps:

1. Generate/load a 32-byte key (where? how? OS keychain? file?)
2. `encryption.NewAES256GCM(key)`
3. `encryption.EncryptMiddleware(encrypter)` on `bus.UsePublish(...)`
4. `encryption.DecryptMiddleware(decrypter)` on `bus.Use(...)`
5. Key rotation: build a `StaticKeyResolver`, wire it into the decrypter

bank-sync doesn't do this because it's 5 steps of plumbing for a local CLI. DiscordSync doesn't do it either. **If encryption were a one-liner, both would have it.**

**Proposal:** Add `stack.WithEncryption` accepting an `encryption.KeyProvider`:

```go
// Primary API — one entry point, any key source
bundle, err := sqlite.New("bank.db",
    stack.WithEncryption(myKeyProvider),
)

// Built-in providers (in the encryption package):
//   encryption.StaticKeyProvider(keyBytes)    // raw bytes (testing, programmatic)
//   encryption.EnvKeyProvider("MY_KEY_VAR")    // dev/test convenience — NOT production
//   encryption.NoKeyProvider()                 // explicit "encryption off"
//
// Consumer-implemented providers (NOT in the SDK — platform-specific):
//   keychainKeyProvider{service: "bank-sync"}   // macOS Keychain / Linux secret-service / DPAPI
//   passphraseKeyProvider{...}                  // argon2id derivation from user passphrase
```

`KeyProvider` is a single-method interface:

```go
type KeyProvider interface {
    Key() (key []byte, err error)  // returns ErrNoKey if unconfigured
}
```

The consumer plugs in their key source; the SDK wires the cipher, `EncryptMiddleware`,
`DecryptMiddleware`, and (once available) encrypted snapshot store — all from one option.
The stack internally calls `provider.Key()` once at startup.

**Why `KeyProvider` instead of `WithEncryptionFromEnv`:**

- Env vars are insecure (see correction above). The API should not privilege an insecure source.
- `KeyProvider` forces the consumer to think about where the key comes from — which is the
  entire point of encryption.
- Extensible without SDK changes: a consumer adding OS keychain support implements one method,
  no new `With*` function needed.
- Usable at any layer (not just `stack`): bank-sync wires the decider directly, not via `stack`,
  but still benefits from `KeyProvider` as the standard interface.

**Key-source security hierarchy (for consumers choosing a provider):**

| Source                          | Disk-theft safe | Friction | Notes                                      |
| ------------------------------- | --------------- | -------- | ------------------------------------------ |
| OS keychain (Keyring/DPAPI)     | ✅ Yes          | Zero     | Ideal for local tools — bound to user session |
| Passphrase → argon2id           | ✅ Yes          | High     | Key never persists; user types each startup   |
| File with `chmod 600` (separated) | Medium        | Low      | Acceptable if on a separate volume/key drive  |
| Environment variable            | ❌ Weak         | Low      | Leaks via `/proc`, child processes, shell history |
| Plaintext file next to DB       | ❌ No           | Zero     | Security theater + data-loss risk             |

**Why this matters:** S002 (encryption) fires on every consumer with PII. The SDK should make
compliance trivial, not something each consumer must figure out from scratch. This is the "pit
of success" principle — the easiest path should be the secure path. But the secure path means
keychain, not env var.

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

> **Dropped (per Appendix A review):** `WithSecurity(SecurityConfig)` with a struct was correctly
> identified as leaky — it reinvents functional options and loses compile-time safety. The
> individual `WithEncryption(provider)`, `WithSigning(provider)`, `WithObservability(...)`
> options are the right granularity. No bundle struct needed.

The encryption + signing + observability one-liners are individual options. They share a theme:
**security and operational middleware.** But bundling them into a struct loses the composability
of functional options. Keep them separate — consumers compose what they need.

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

---

## Appendix A — Maintainer Review (PRO / CONTRA)

**Reviewer:** maintainer (Crush)
**Date:** 2026-07-17
**Verdict:** the _instincts_ are right (make the secure path easy, fix detectors, evidence-driven). The _packaging_ is wrong — it bundles three independent debates into one, and the deployment taxonomy over-generalizes from N=2 consumers.

### PRO — genuinely good ideas

1. **"Pit of success" SDK one-liners are correct and important.** Wiring encryption in 5 manual steps IS why consumers skip it. `WithEncryptionFromEnv("KEY")` is the right shape — functional option, composable, matches the existing `middleware.NewOTelBundle` precedent.
2. **The "hard floor" guardrail is well-designed.** "Declarations can only suppress, never enable" + non-suppressible S001/S002/C001-C012/E001-E002 prevents score-gaming.
3. **The encryption argument (Part 3) is well-reasoned.** Laptop theft, cloud sync, accidental commit — "local tool" really isn't an excuse.
4. **Evidence-based framing (39→8 findings).** Two consumers with concrete FP counts is the right kind of data to drive decisions.

### CONTRA — real problems

1. **Couples three independent things that should be three PRs.** P0 detector fixes ≠ profile system ≠ SDK one-liners. Bundling risks blocking the cheap unambiguous fixes behind the expensive design debate.
2. **The deployment taxonomy is premature — built from N=2.** `kind: local-cli | server | library | batch-job` misses workers, lambdas, embedded, CLIs-that-call-servers, gRPC servers. `data: internal` is hopelessly vague. Classic premature generalization.
3. **Four competing config mechanisms is too many.** `deployment.*` declarative + auto-detection heuristics + `rules.exclude` + per-rule overrides. **Pick one primary mechanism.**
4. **Auto-detection heuristics are brittle and trade one FP class for another.** PII detection via field-name string matching (`email`, `iban`) will FP on `email_queue_size` and fail every non-English domain.
5. **The profile system is largely a workaround for weak detectors.** Most "Fix:" entries in Part 4 are detector fixes. If the detectors are fixed properly, the deployment taxonomy may be unnecessary.
6. **`WithSecurity(SecurityConfig)` is leaky and reinvents functional options.** A struct with required/optional fields + runtime validation loses compile-time safety. Drop it.
7. **The closure-fix snippet is fragile.** It assumes the handler's 2nd param is always `*T`. Handlers can take value receivers and have injected deps.
8. **No schema-versioning story.** When the taxonomy expands, what breaks?

### The pivot: feature-based vocabulary, not deployment archetypes

> **Decision:** the config vocabulary should be grounded in **go-cqrs-lite's own features/modules**, not in abstract deployment concepts. "You can imagine anything otherwise and it will be a nightmare to map all these correctly in go-cqrs-lite."

Instead of `"kind": "local-cli"` (fuzzy, hard to auto-detect, drifts constantly), declare **which go-cqrs-lite features the consumer uses**. Each flag maps 1:1 to a library module, and auto-detection is a reliable import + constructor scan:

| Feature flag   | Values                                                | go-cqrs-lite module                  | Auto-detect signal                  |
| -------------- | ----------------------------------------------------- | ------------------------------------ | ----------------------------------- |
| `store`        | sqlite, postgres, pebble, memory, turso, custom, none | `stack/*`, `storage/*`               | stack preset import                 |
| `command-flow` | read-only, sync, commands                             | `command/`, `decider/`               | `command.Dispatcher` + `Dispatch()` |
| `server`       | true, false                                           | `transport/http/`, `transport/grpc/` | `net/http` / `grpc.NewServer`       |
| `soft-delete`  | true, false                                           | event tombstone                      | tombstone-like event type names     |
| `tracing`      | off, on                                               | `otel/`, `middleware` OTel           | otel import + middleware wiring     |
| `snapshot`     | off, on                                               | `snapshot/`                          | snapshot store usage                |

**Why this is better:**

- The vocabulary IS the library's vocabulary — no translation layer.
- Auto-detection is reliable (imports + constructor calls), not NLP-grade field-name guessing.
- Each flag directly answers the one rule that needs it (no 5-axis matrix that drifts).
- A consumer reading `"command-flow": "sync"` understands it refers to `command.Dispatcher` usage; `"deployment": "local-cli"` required interpretation.

### Reality check: what is ALREADY shipped

> The feedback doc's Part 2.4 (E005/E007 closure tracing) and Part 2.5 (generic `unwrapSelector`) were **already implemented** in commit `579a3438`, and the per-detector heuristics (S002 local-only, A012 tombstone, A016 dispatch) in `5a9425a6`. D005 wildcard handling (`isVersionCompatible`) also shipped.

What genuinely **remains** is smaller than this doc implies:

- ✅ Generic call detection (`SelectorFromExpr`) — DONE
- ✅ Closure handler tracing — DONE
- ✅ `Type()` method FP fix — DONE
- ✅ Upcaster context detection — DONE
- ✅ D005 wildcard + migration-arrow handling — DONE
- ✅ Per-detector heuristics — DONE (but **scattered across 3 files** — the architectural smell)
- ❌ Centralize the 3 scattered heuristics into one `FeatureProfile` (the architectural cleanup)
- ❌ `features` config section in `.cqrs-lint.json` + AppConfig
- ❌ SDK one-liners (`WithEncryptionFromEnv`, etc.) — **lower priority** (PII is not the current focus)

### Recommendation

| Do now                                                        | Do later                                         | Drop                                  |
| ------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------- |
| Centralize heuristics → `FeatureProfile` (feature vocabulary) | `WithEncryptionFromEnv` / `WithObservability`    | `WithSecurity(SecurityConfig)` struct |
| `features` config section + auto-detect                       | `encryption.GenerateKey()` first-run UX          | The 5-axis deployment taxonomy        |
| `cqrs-lint doctor` to print detected features                 | Property detectors (write-after-init) as primary | Auto-detection as "fallback"          |
| Split the proposal into 3 independent PRs                     |                                                  |                                       |

The proposal's _instincts_ are right; the _packaging_ and _vocabulary_ need this pivot. See `docs/planning/2026-07-17_01-45_feature-profile-and-detector-consolidation.md` for the execution plan.
