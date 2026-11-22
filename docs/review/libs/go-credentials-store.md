# go-credentials-store — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-credentials-store` @ 16ab83f (latest local tag v0.4.3) · Consumer go-cqrs-lite @ feec9bc74 · Group G4

**Verdict:** NOT-ADOPTED-JUSTIFIED
**Score:** 0/1 capabilities fully leveraged = 0% (low)

Zero requires across all 104 go.mod files — and this zero is structural, not
lazy: go-credentials-store is not an upstream utility but a **downstream
application of go-cqrs-lite**, built on the consumer's own `system`,
`metaengine`, and `encryption` modules (go-credentials-store/go.mod:10-19).
Importing it back would be a dependency U-turn. The library side of
go-cqrs-lite correctly carries no credentials (its signing/encryption
surfaces take key material as explicit arguments), and even the example apps
and CLI tools read no secrets from the environment. The single row that
survives the N/A filter is key *provisioning*: the repo ships key-args APIs
(`signing.NewEd25519(privateKey)`) but no canonical story for where those
bytes come from — the problem go-credentials-store solved for the fleet, at
application level. The flip trigger is therefore an app, never a module.

## 1. What the library is

An "event-sourced, encrypted-at-rest credentials store for personal
infrastructure: API keys, Playwright session blobs, and passwords — with
lifecycle audit and a single-operator loopback admin UI"
(go-credentials-store/README.md:3-6). Library-first shape:
`credentials.Build(ctx, opts)` + `Resolve(ctx, platform, kind)`
(README.md:8-9, 52-58); secrets sealed XChaCha20-Poly1305 with KeyID
rotation *before* they become events, and views/HTML carry verdicts only
(README.md:10-12); built on go-cqrs-lite's `system` composition root +
`metaengine` read models per ADR-0123 (README.md:13-14). The opt-in
`adminui.Mount` shim ships HTML UI, JSON API, SSE, one-time audited reveal,
`/status`, and `/metrics` in one handler (README.md:75-78). FEATURES.md
records the full inventory: Resolve SDK with store→env fallback and cache,
`Resolver.RecordUse`, sealed kinds, key rotation, freshness classifier,
export/import bundles, recovery drill, loopback guard + API-key auth for
non-loopback binds, and the `cvcompat` env-fallback shim
(go-credentials-store/FEATURES.md, rows 1-14, 38-41). Status:
feature-complete for single-operator use, adopted by CV in production
(README.md:18-20). Latest tag **v0.4.3**; HEAD 16ab83f (2026-11-20).

## 2. Usage census in go-cqrs-lite (evidence)

- **Zero requires**: `rg -n 'go-credentials-store' -g go.mod` over all 104
  go.mod files returns nothing — no module, direct or indirect, no example,
  no tool, no test.
- **Zero hand-rolled credential loading** to replace: the only environment
  reads in examples/tools are paths and addresses, not secrets —
  `DATABASE_PATH` (`example/taskmanager/main.go:30`),
  `ADDR` (`example/scheduler-otel-status/main.go:86`),
  `CQRS_DEPLOYMENT_CONFIG` (`example/standard-topology/main.go:26`), and
  Go-toolchain vars in cqrs-lint's version rule
  (`cmd/cqrs-lint/v4/pkg/rules/version/v003_latest.go:129-242`). The repo's
  own security rule *advises consumers* to load secrets from env or a
  secret manager (`cmd/cqrs-lint/v4/pkg/rules/security/rules.go:82`) — the
  repo itself has none to load. The one "credentials" string in the
  examples is about CI git auth, not runtime secrets
  (`example/taskmanager/must.go:6`).
- **The dependency arrow points the other way**: go-credentials-store
  requires nine go-cqrs-lite modules in production
  (go-credentials-store/go.mod:10-19: command, decider, encryption, event,
  id, metaengine, metaengine/sqliteengine, query, system, systemscenario)
  and its resolver takes the consumer's composition root directly
  (`cqsystem.System` in go-credentials-store/resolve.go:50).
- Fleet context: the store already tracks the consumer's releases
  (system/v4 v4.16.0, event/v4 v4.15.2 at go-credentials-store/go.mod:14,18)
  and moved to cmdguard v5.1.0 in its v0.4.3 window — it is a follower of
  go-cqrs-lite, not a dependency of it.

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Resolve SDK (`Build` + `Resolve`/`ResolveAPIKey`, cache, env fallback) | N/A | a library/SDK has no runtime credentials to resolve; go-cqrs-lite's modules must stay secret-agnostic (key-args doctrine below) |
| 2 | Sealed-at-rest storage + KeyID rotation | N/A | the primitives already live *inside* the consumer as `encryption/v4` (envelope, KeyID, XChaCha20/AES-GCM); the store consumes them (go-credentials-store/go.mod:12) |
| 3 | Lifecycle audit trail (created/rotated/probed/used/revealed) | N/A | no credential lifecycle exists in the consumer's scope |
| 4 | `adminui.Mount` operator UI (HTML/JSON/SSE, one-time reveal) | N/A | no operator surface for secrets; docserver documents a system, not secrets |
| 5 | Health/`/status`/`/metrics` endpoints | N/A | the consumer ships its own: docserver health (`catalog/v4/docserver/health.go`), prometheus module (`prometheus/v4`), cqrsotel |
| 6 | `cvcompat` store-first + legacy-env fallback shim | N/A | CV-specific migration seam (README.md:20-21); no legacy env in scope |
| 7 | Export/import bundles + recovery drill | N/A | nothing secret to back up in the consumer |
| 8 | Master-key provisioning/rotation story for key-args APIs | Missed | the one applicable row: `signing/v4` takes raw key args (`signing/v4/ed25519.go:23,100` — `NewEd25519(privateKey)`, `NewCOSEEd25519Signer`) and `encryption/v4` takes `map[KeyID][]byte`, but no example, recipe, or module shows where those bytes come from or how to rotate them; go-credentials-store is the fleet's only canonical answer and it is an application |

Score derivation: rows N=8, N/A rows 1–7 → **A=1**. Full=0, Partial=0,
Missed=1 (0). Score = round(100 × 0 / 1) = **0% → low band**. The low
percentage is the honest shape of a justified zero: nothing here is
fixable by adopting the library into go-cqrs-lite.

## 4. Gap analysis

1. **The relationship is inverted, and that is correct.** go-credentials-
   store requires the consumer's modules (go-credentials-store/go.mod:10-19);
   adopting it into any go-cqrs-lite module would create a module cycle and
   drag an application (SQLite state, admin UI, HTTP server) into a library
   whose per-module production dep budgets (`nix run .#check-arch`) exist to
   prevent exactly that. The zero is doctrine-compliant, same class as the
   transport-agnostic rule (ADR-0127).
2. **"Library with key args" is the right doctrine for the consumer — but
   it has a soft underbelly.** `signing/v4/ed25519.go:23` and the
   `encryption/v4` envelope API push key provisioning to callers, which is
   correct for a library; yet the repo demonstrates key use nowhere, so
   every fleet app invents provisioning (env var? file? age?). The security
   rule's own advice ("load secrets from environment variables or a secret
   manager", `cmd/cqrs-lint/v4/pkg/rules/security/rules.go:82`) has no
   worked example in the repo. This is a documentation/example gap, not an
   adoption gap.
3. **The real integration surface is app-level and already designed.** When
   a fleet app (a taskmanager deployment with real auth, a CV-style
   operator service) needs credentials, the intended path exists:
   `credentials.Build` against the app's own system config
   (go-credentials-store/README.md:52-58), or the narrower `Resolve` SDK
   with env fallback. Nothing in go-cqrs-lite should wrap or re-export it.

## 5. Version currency

- Pinned: **nothing**; latest local tag **v0.4.3** (2026-11-13), HEAD
  16ab83f (2026-11-20) — with no require, currency is structurally moot for
  this consumer.
- Fleet-relevant recency: v0.4.3 is a coordinated wave release — templ
  v0.3.1100 + templ-components v1.27.0 (matching the pins already in
  `catalog/v4/go.mod:12-14`) and cmdguard v5.1.0
  (go-credentials-store/CHANGELOG.md [0.4.3]). Any fleet app adopting the
  store's v0.4.3 lands on the same templ/security fix train
  (GHSA-94c2-xg24-pwhv sanitization) the consumer's docserver already
  rides — compatible, no conflict.
- The store tracks the consumer's released tags (event/v4 v4.15.2,
  system/v4 v4.16.0 at go-credentials-store/go.mod:14,18) rather than HEAD;
  a consumer API break therefore surfaces in the store's CI, which is the
  fleet's early-warning for this pair.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only
fix bugs all day long." Credentials are the classic case for that argument
— every fleet app that hand-rolls env-var key loading re-creates rotation,
audit, and at-rest sealing bugs, and a leaked-key incident would have to be
patched N times. That argument is *already satisfied*: the fleet's one
canonical implementation exists, is event-sourced on the consumer's own
engine, and is production-proven in CV. The failure mode to avoid is not
under-adoption by go-cqrs-lite (impossible by design) but **under-
documentation of the seam**: the library ships key-args APIs, the store
implements provisioning, and the missing middle is one recipe in
`.agents/skills/go-cqrs-lite/references/recipes.md` showing an app wiring
`signing/v4` keys via `credentials.Build`. One recipe, every fleet app
stops inventing key loading; zero new dependencies for the library.

## 7. Ranked opportunities

1. **Write the key-provisioning recipe** (medium): a docs/recipes entry
   showing a fleet app loading its `signing/v4`/`encryption/v4` key
   material through `credentials.Build`/`Resolve` — closes finding 2's
   example gap without any module change.
2. **Link the store from SECURITY.md or docs/signing-architecture.md**
   (low): today nothing in the consumer points fleet apps at the canonical
   store; a pointer makes the deliberate non-adoption discoverable instead
   of silent.
3. **Keep the store pinned as a consumer in fleet sweeps** (low): its
   go.mod pins of system/event/etc. are canaries for consumer API breaks;
   worth listing in the fleet census script's repos.

## 8. Reconsider when

- A go-cqrs-lite example app gains real third-party credentials (the
  taskmanager grows upstream API calls, or an example demonstrates signed
  webhooks with managed keys) — adopt `credentials.Build` in that app's
  go.mod, never in a library module.
- The owner stands up a deployed fleet service (not a library) that needs
  operator-grade secret handling — the `adminui.Mount` shim plus loopback
  guard is the default posture.
- go-credentials-store ever extracts a minimal "key-provider" SDK
  (interface + env/file implementations without the event store) — *that*
  would be upstream-adoptable by `encryption/v4` callers as an optional
  module, and the N/A rows 1-2 would reopen.

## 9. How to verify

```
rg -n 'go-credentials-store' -g go.mod /home/lars/projects/go-cqrs-lite   # no output
rg -n 'os.Getenv|LookupEnv' --type go /home/lars/projects/go-cqrs-lite/example /home/lars/projects/go-cqrs-lite/cmd | grep -v _test
git -C /home/lars/projects/go-credentials-store log -1 --format='%H' && git -C /home/lars/projects/go-credentials-store tag --list | sort -V | tail -3
sed -n '10,19p' /home/lars/projects/go-credentials-store/go.mod
sed -n '44,78p' /home/lars/projects/go-credentials-store/README.md
sed -n '20,30p' /home/lars/projects/go-cqrs-lite/signing/v4/ed25519.go
sed -n '75,85p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/security/rules.go
```

Related reports in this directory: `cmdguard.md` (the store is ahead of the
consumer on v5.1.0), `go-codec.md`/`go-error-family.md` (shared upstream
primitives both repos consume).
