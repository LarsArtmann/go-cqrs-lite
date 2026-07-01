# Ecosystem Boundaries: Where to Put What

> **Status:** Authoritative | **Scope:** `go-cqrs-lite` ↔ `cqrs-htmx` ↔ `go-localsync` (+ `github-local-sync`)
> **Date:** 2026-06-22 | **Updated:** 2026-07-01 (module counts, usermgmt constraint, file counts)

## TL;DR

The three-repo layering is **architecturally correct**. The problems are not "modules
in the wrong repo" — they are **seam violations**: duplicated types, contradictory
mappings, and inconsistent import conventions _at the boundaries between repos_.

**Do not move modules between repos. Fix the seams.**

---

## The Layering (Correct As-Is)

```
                    go-branded-id          go-error-family
                    (public, foundation)   (public, foundation)
                            │                     │
                            └──────────┬──────────┘
                                       │
                              go-cqrs-lite (v3)
                          ┌─────────────┴─────────────┐
                          │ CQRS + Event Sourcing     │
                          │ 42 modules, library-only  │
                          │ id/, event/, command/,    │
                          │ query/, decider/, codec/, │
                          │ storage/, catalog/, ...   │
                          └─────────────┬─────────────┘
                                        │
                    ┌───────────────────┴───────────────────┐
                    │                                       │
              cqrs-htmx (v3)                        go-localsync (v0.2)
          ┌─────────┴─────────┐                  (contract SDK, no provider)
          │ HTTP + HTMX +     │                        │
          │ templ + Casbin    │                        │
          │ + CSRF + SSE/WS   │                        │
          │ + usermgmt (IAM)  │                        │
          │ + catalog (HTTP)  │                        │
          └───────────────────┘                        │
                                                        │
                                               github-local-sync
                                            (consumer app: GitHub provider + CLI)
```

| Repo           | Role               | Depends on                         | Consumers depend on it           |
| -------------- | ------------------ | ---------------------------------- | -------------------------------- |
| `go-cqrs-lite` | Foundation library | `go-branded-id`, `go-error-family` | `cqrs-htmx`, `go-localsync`      |
| `cqrs-htmx`    | Web integration    | `go-cqrs-lite`, `casbin`, `nosurf` | Apps needing HTMX + CQRS + auth  |
| `go-localsync` | Sync engine SDK    | `go-cqrs-lite`                     | `github-local-sync` (and others) |

**This is a clean dependency DAG with no cycles.** The repos are correctly separated.

---

## What Belongs Where — The Decision Matrix

Use this when you have a piece of code and don't know which repo it goes in.

### go-cqrs-lite — "Infrastructure primitives, no opinions about your app"

| Belongs HERE                                      | Does NOT belong here                         |
| ------------------------------------------------- | -------------------------------------------- |
| Event/Command/Query core types and interfaces     | HTTP handlers (`net/http`)                   |
| Decider pattern, Repository, Store/Bus interfaces | HTML rendering, HTMX, templ                  |
| Storage backends (SQL, Pebble, memory, KV)        | Authorization (Casbin, RBAC)                 |
| Codec (JSON, CBOR)                                | Sync engines, providers, CRDT conflict logic |
| Catalog data model + exporters (no HTTP)          | User management, auth flows, sessions        |
| Middleware (logging, retry, tracing) — generic    | CSRF, rate limiting (HTTP-specific)          |
| Branded ID primitives (`id.Of[T]`, markers)       | Domain-specific IDs (ItemID, TenantID, etc.) |
| Error family re-export hub                        | HTTP status code mapping                     |
| Signing, encryption, OTel, Prometheus bridge      | SSE/WS transport (that's HTTP-layer)         |

**Litmus test:** _"Would ANY Go app — not just web apps — potentially use this?"_
If yes → go-cqrs-lite. If it only makes sense with HTTP/HTMX → cqrs-htmx.

### cqrs-htmx — "Making CQRS easy on the web"

| Belongs HERE (root module)                         | Does NOT belong here                         |
| -------------------------------------------------- | -------------------------------------------- |
| HTTP handlers wrapping CQRS dispatch               | Event store implementations                  |
| HTMX response builder, notifications, swap         | Branded ID definitions (re-export from lite) |
| CSRF middleware (nosurf)                           | Error family constructors (use from lite)    |
| Rate limiting (HTTP-specific)                      | Provider/sync abstractions                   |
| Security headers, request logging                  | CRDT, conflict resolution                    |
| SSE/WS broadcasters and CQRS bridges               |                                              |
| Context enrichment (UserID, CorrelationID → ctx)   |                                              |
| Catalog HTTP handlers (wraps go-cqrs-lite/catalog) |                                              |

**Litmus test:** _"Does this only make sense in an HTTP handler?"_
If yes → cqrs-htmx. If it's transport-agnostic → go-cqrs-lite.

### cqrs-htmx/usermgmt — "Identity & Access Management platform"

| Belongs HERE                                     | Does NOT belong here                   |
| ------------------------------------------------ | -------------------------------------- |
| User aggregate (event-sourced)                   | HTTP dispatch wiring (that's root)     |
| WebAuthn, TOTP, OAuth2/OIDC flows                | HTMX response formatting (that's root) |
| Sessions, account lockout, audit log             | SSE/WS (that's root)                   |
| RBAC via Casbin (projection + policy management) | CSRF (that's root)                     |
| Tenants, memberships, bots, impersonation        | Event store SQL (that's go-cqrs-lite)  |
| SQL session store                                |                                        |

**Key constraint (updated 2026-07-01):** `usermgmt` imports `cqrs-htmx/v3` root for
`RateLimiter` (since 2026-06-28, one-way dependency). It also directly depends on
go-webauthn, golang.org/x/oauth2, coreos/go-oidc, pquerna/otp, modernc/sqlite, and
casbin/v3 — forcing these as transitive deps on all consumers. The original constraint
("zero imports from root, depends only on go-cqrs-lite") no longer holds. The auth dep
bloat is documented as known debt (`cqrs-htmx/docs/modularization/2026-07-01_PROPOSAL.html`),
with sub-package extraction planned for v4. The bridging still happens in consumer apps
via structural interfaces (`Enforcer`) and string conversion (`UserID.String()`).

**Litmus test:** _"Is this identity/auth domain logic?"_ → usermgmt.
_"Is this HTTP wiring?"_ → cqrs-htmx root.

### go-localsync — "Sync provider data into local event-sourced storage"

| Belongs HERE                                       | Does NOT belong here                      |
| -------------------------------------------------- | ----------------------------------------- |
| `Provider` interface (the sync contract)           | Concrete providers (GitHub, GitLab, etc.) |
| Sync engine (`Syncer`, `ConflictAwareSyncer`)      | HTTP API serving (use cqrs-htmx for that) |
| CRDT conflict resolution (`ConflictResolver[T]`)   | HTML/HTMX rendering                       |
| Domain model (`model.Item`, events, decider)       | User auth (use usermgmt for that)         |
| Sync-specific branded IDs (`ItemID`, `ProviderID`) | Generic CQRS types (use go-cqrs-lite)     |
| Sync-specific errors (wrapping go-error-family)    | Event store implementations               |

**Litmus test:** _"Is this about syncing external data into local storage?"_
If yes → go-localsync. If it's generic CQRS → go-cqrs-lite.

---

## Seam Violations — What's Actually Wrong

These are the real problems. Ordered by severity.

### 🔴 P0 — Type Safety Broken: Incompatible `UserID` types

**The single most damaging issue across the ecosystem.**

| Location                        | Definition                            | Backing type         |
| ------------------------------- | ------------------------------------- | -------------------- |
| `go-cqrs-lite/id/user_id.go:10` | `id.UserID = Of[UserMarker]`          | `ulid.ULID` (binary) |
| `cqrs-htmx/context.go:13`       | `type UserID = id.UserID` (re-export) | `ulid.ULID` ✅       |
| `cqrs-htmx/usermgmt/id.go:15`   | `brandid.ID[userBrand, string]`       | `string` ⚠️          |

`cqrshtmx.UserID` and `usermgmt.UserID` are **named identically but are
type-incompatible**: different backing type (`ulid.ULID` vs `string`) AND different
phantom brand (`id.UserMarker` vs `usermgmt.userBrand`). Every HTTP→domain path
requires manual `.String()` → `NewUserID()` conversion with **zero compile-time safety**.

**Fix:** `usermgmt.UserID` should re-export `id.UserID` (or at minimum use the same
`ulid.ULID` backing + exported `id.UserMarker` brand). This is a **breaking change**
to usermgmt's v3 API, so it must be coordinated.

### 🟢 P0 — `ActorID` canonicalized (RESOLVED)

| Location                     | Shape                                              | Status                                                                                                                 |
| ---------------------------- | -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `cqrs-htmx/context.go:141`   | bare `string` (no branding)                        | ✅ Intentional — transport boundary, no type needed                                                                    |
| `cqrs-htmx/usermgmt/id.go`   | `struct{kind ActorKind; raw string}` (not branded) | ✅ Canonical domain actor type (user/bot discrimination)                                                               |
| `go-localsync/pkg/id/ids.go` | `brandid.ID[ActorLoginBrand, string]`              | ✅ Renamed from `ActorID` to `ActorLogin` — represents external provider actor (GitHub username), not the same concept |

The three types are now clearly distinct concepts, not a duplication:

- Root cqrs-htmx: `string` for context propagation (transport concern)
- usermgmt: `ActorID` struct for domain-level actor kind discrimination
- go-localsync: `ActorLogin` branded string for external provider usernames

### 🟢 P1 — Error handling: standardized (RESOLVED)

**Canonical convention (established):**

- `event/errors.go` is the single re-export hub for `go-error-family` in go-cqrs-lite (v0.5.0 — `Compose` removed, use stdlib `errors.Join`)
- `command/` and `query/` no longer re-export error family functions (E05 — triplication eliminated)
- Modules that depend on `event/v3` use `event.NewRejection()` etc.
- Modules that don't depend on `event` (codec, catalog, cmd) import `errorfamily` directly
- External consumers (cqrs-htmx, go-localsync) use `event.` re-export or `errorfamily` directly — consistent within each repo

**HTTP status mapping reconciled** (E04): cqrs-htmx now delegates to upstream `Family.HTTPStatus()` (go-error-family v0.5.0) — the hand-rolled `familyStatus()` switch is deleted:

- Corruption → 500, Infrastructure → 503 (was 422/500 — contradiction resolved)
- See ADR-0017 in cqrs-htmx

### 🟡 P1 — `catalog` HTTP handlers in cqrs-htmx: correct placement, thin wrapper

`cqrs-htmx/catalog/` is a **3-file facade** over `go-cqrs-lite/catalog/` that adds
`net/http` handlers (`OpenAPIHandler`, `AsyncAPIHandler`, `D2Handler`).

**This is correct.** The data model + exporters belong in go-cqrs-lite (transport-
agnostic). The HTTP serving belongs in cqrs-htmx (HTTP-specific). No action needed —
documented here to preempt "should we merge them?" questions.

The one smell: `cataloghtmx.Command[T]`/`Query[T]`/`Event[T]` are one-line delegates
to `catalog.Command[T]` etc. This is acceptable facade ergonomics, not harmful
duplication.

### 🟢 P2 — go-localsync reimplements projection replay (forced, not gratuitous)

`pkg/cqrs/runner.go:22-26` reimplements ~50 lines of journal replay because
go-cqrs-lite v3 deleted the `projection/` module (ADR-0030). The comment documents
this. This is **not duplication** — it's a forced consequence of an upstream decision.

**Long-term fix:** go-cqrs-lite should provide a lightweight replay helper (the
"Runner wiring helpers" suggested in `go-localsync/docs/planning/2026-05-25_UPSTREAM-
SUGGESTIONS.md`). Until then, the 50-line reimplementation is correct.

### 🟢 P2 — go-localsync's `query.Dispatcher` wired but bypassed

`stack_adapters.go:1-23` admits that `List`/`Count`/`GetTypes` call the ReadModel
directly, not through the dispatcher. The dispatcher exists only for test verification.
This is test-only dead weight in the production path — a local cleanup, not a cross-
repo issue.

### 🟢 P3 — go-cqrs-lite is PRIVATE (blocking downstream nix builds)

`go-localsync` cannot use `nix build` / `nix flake check` because go-cqrs-lite is a
private GitHub repo (nix sandbox can't fetch it). Workaround: committed `vendor/` dir

- `vendorHash = null`. The public siblings (`go-branded-id`, `go-error-family`) work
  fine.

**Fix:** Make `go-cqrs-lite` public. It's a v3.0.0 released library — there's no
competitive reason to keep it private, and it blocks the entire downstream toolchain.

---

## What NOT to Do

Based on existing analysis docs, these have been explicitly evaluated and rejected:

| Proposal                                             | Verdict                                                               | Source                                                               |
| ---------------------------------------------------- | --------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Extract `usermgmt` into its own repo                 | **No** — module split already gives 80% of the benefit; scored 3.2/10 | `cqrs-htmx/docs/brainstorming/extract-usermgmt-pro-contra.html`      |
| Split cqrs-htmx root into sub-modules                | **No** — 46 files form a cohesive unit; errors↔response↔csrf cycle    | `cqrs-htmx/docs/modularization/2026-07-01_PROPOSAL.html`             |
| Merge go-cqrs-lite catalog sub-modules               | **✅ DONE** — consolidated into 1 go.mod with sub-packages            | `go-cqrs-lite/docs/modularization/PROPOSAL_catalog_consolidation.md` |
| Move CRDT/conflict from go-localsync to go-cqrs-lite | **No** — sync-specific, not generic CQRS                              | (architectural reasoning)                                            |
| Move catalog HTTP handlers to go-cqrs-lite           | **No** — HTTP is cqrs-htmx's domain                                   | (this document)                                                      |

---

## Action Plan (Pareto-ordered)

### Phase 1: Unblock (1 decision, 0 code)

1. **Make `go-cqrs-lite` public.** Eliminates the `vendor/` workaround in go-localsync
   and unblocks `nix flake check` across the ecosystem. This is a GitHub repo setting,
   not a code change.

### Phase 2: Fix the type-safety seams (high impact, coordinated)

2. **Align `usermgmt.UserID` with `id.UserID`.** Either re-export `id.UserID` or
   switch the backing to `ulid.ULID` with `id.UserMarker`. Breaking change → coordinate
   with cqrs-htmx consumer code. File: `cqrs-htmx/usermgmt/id.go:15`.

3. **Canonicalize `ActorID`.** Adopt `usermgmt.ActorID` (kind-discriminated) as the
   canonical type, or rename the others. At minimum, `cqrs-htmx/context.go:141` should
   use a branded type, not bare `string`.

### Phase 3: Fix the error-handling inconsistency (medium impact)

4. **Decide the canonical error import path.** Recommendation: consumers import
   `go-error-family` directly (as go-localsync does). Remove the triplicated re-export
   blocks from `command/errors.go` and `query/errors.go` in go-cqrs-lite. Keep
   `event/errors.go` re-exports only for domain sentinels.

5. **Reconcile the HTTP status mapping.** `cqrs-htmx/errors.go:71-86` must either match
   `go-error-family` upstream or document a deliberate override with an ADR.

### Phase 4: Internal cleanup (low impact, repo-local) — PARTIALLY COMPLETE

6. **go-cqrs-lite:** ✅ Catalog sub-modules already merged into one. ✅ eventtest extracted as separate module (breaks partial cycle). ⏳ Remaining cycle: event → storage/memory → command → event (via test dependencies). Breaking requires moving event test files to a separate integration test module.

7. **cqrs-htmx:** ⏳ Decompose `usermgmt` god-package (84 files, ~11K LOC) — sub-package extraction (webauthn/, oauth2/, totp/, sql/) planned for v4. See `docs/modularization/2026-07-01_PROPOSAL.html`. CI layer enforcement scripts should be added first to prevent further degradation. (E10 — deferred)

8. **go-localsync:** ✅ The bypassed `query.Dispatcher` is an intentional performance optimization for hot read paths (documented in `stack_adapters.go`). The dispatcher is still wired for tests. No change needed.

---

## How to Decide "Where Does This Go?" — The Algorithm

```
1. Is it a CQRS/ES primitive (event, command, store, bus, codec, ID)?
   → go-cqrs-lite

2. Is it an HTTP handler, middleware, or response builder?
   → cqrs-htmx (root)

3. Is it identity/auth/user-management domain logic?
   → cqrs-htmx/usermgmt

4. Is it about syncing external data into local storage?
   → go-localsync

5. Is it a concrete provider (GitHub, GitLab) or a CLI?
   → consumer app (github-local-sync)

6. Is it a foundational type used by ALL of the above (branded ID, error family)?
   → go-branded-id / go-error-family (standalone public repos)
```
