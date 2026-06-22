# Ecosystem Boundary Fix — Comprehensive Execution Plan

> **Scope:** `go-cqrs-lite` ↔ `cqrs-htmx` ↔ `go-localsync`
> **Source analysis:** [`../ECOSYSTEM_BOUNDARIES.md`](../ECOSYSTEM_BOUNDARIES.md)
> **Date:** 2026-06-22
> **Rule:** Every task ≤ 12 minutes. No exceptions.

## Scoring Methodology

Each **epic** (parent task) is scored on four axes (1–5):

| Axis               | Meaning                                                           |
| ------------------ | ----------------------------------------------------------------- |
| **Importance**     | How critical to ecosystem health (1=nice-to-have, 5=foundational) |
| **Impact**         | Blast radius — how many consumers/modules affected                |
| **Customer Value** | Value to downstream consumers / end users                         |
| **Effort**         | Number of ≤12-min chunks (complexity proxy)                       |

**Priority Score** = `Importance × Impact × CustomerValue`, then sorted by score descending, with **Effort** as a tiebreaker (lower effort first).

---

## Epic Priority Ranking

| Rank | ID  | Epic                                           | I   | Impact | CV  | Score   | Effort (chunks) |
| ---- | --- | ---------------------------------------------- | --- | ------ | --- | ------- | --------------- |
| 1    | E01 | Make `go-cqrs-lite` public                     | 5   | 5      | 5   | **125** | 4 (+1 decision) |
| 2    | E02 | Align `usermgmt.UserID` with `id.UserID`       | 5   | 5      | 5   | **125** | 11              |
| 3    | E03 | Canonicalize `ActorID` (3 incompatible types)  | 5   | 4      | 4   | **80**  | 8               |
| 4    | E04 | Reconcile HTTP status mapping (contradiction)  | 4   | 5      | 5   | **100** | 5               |
| 5    | E05 | Remove triplicated error re-exports            | 4   | 4      | 4   | **64**  | 7               |
| 6    | E06 | Standardize error import convention            | 3   | 4      | 4   | **48**  | 4               |
| 7    | E07 | Break go-cqrs-lite module dependency cycles    | 4   | 4      | 3   | **48**  | 8               |
| 8    | E08 | Merge go-cqrs-lite catalog sub-modules         | 3   | 3      | 3   | **27**  | 9               |
| 9    | E09 | go-localsync: remove bypassed query.Dispatcher | 2   | 2      | 3   | **12**  | 3               |
| 10   | E10 | Decompose usermgmt god-package (71→8 sub-pkgs) | 2   | 3      | 2   | **12**  | 15              |
| 11   | E11 | Sync documentation across all repos            | 2   | 3      | 2   | **12**  | 6               |

**Execution order** (score desc, effort asc tiebreak):
`E01 → E04 → E02 → E03 → E05 → E06 → E07 → E08 → E09 → E11 → E10`

---

## E01 — Make `go-cqrs-lite` Public

**Why:** Private repo blocks `nix flake check` and forces committed `vendor/` in go-localsync. All siblings (`go-branded-id`, `go-error-family`) are already public. It's a released v3.0.0 library.

| Task ID | Task                                                                         | Effort           |
| ------- | ---------------------------------------------------------------------------- | ---------------- |
| E01.1   | Audit git history for secrets/credentials/tokens                             | ≤12m             |
| E01.2   | Verify LICENSE, SECURITY.md, CONTRIBUTING.md are correct                     | ≤12m             |
| E01.3   | Audit all module READMEs — badges/links point to v3, no internal-only jargon | ≤12m             |
| E01.4   | Verify no hardcoded internal paths or private sibling refs in docs           | ≤12m             |
| E01.5   | **[USER DECISION]** Flip GitHub repo setting: Private → Public               | 0m (user action) |

---

## E04 — Reconcile HTTP Status Mapping

**Why:** `cqrs-htmx/errors.go:71-86` maps `Corruption→422, Infrastructure→500`. `go-error-family` upstream maps `Corruption→500, Infrastructure→503`. They **contradict** — consumers get wrong HTTP codes.

| Task ID | Task                                                                  | Effort |
| ------- | --------------------------------------------------------------------- | ------ |
| E04.1   | Read `cqrs-htmx/errors.go:71-86` + `structured_error.go:133-164`      | ≤12m   |
| E04.2   | Read `go-error-family/family.go:66-119` upstream mapping              | ≤12m   |
| E04.3   | Decide: adopt upstream mapping OR document deliberate override in ADR | ≤12m   |
| E04.4   | Implement the decision in `errors.go` + `structured_error.go`         | ≤12m   |
| E04.5   | Update affected tests; run `go test ./...` in cqrs-htmx               | ≤12m   |

---

## E02 — Align `usermgmt.UserID` with `id.UserID`

**Why:** `cqrshtmx.UserID` (ulid-backed) and `usermgmt.UserID` (string-backed) are named identically but **type-incompatible**. Every HTTP→domain path needs manual `.String()`→`NewUserID()` conversion with zero compile-time safety.

| Task ID | Task                                                        | Effort |
| ------- | ----------------------------------------------------------- | ------ |
| E02.1   | Grep all usages of `usermgmt.UserID` in cqrs-htmx repo      | ≤12m   |
| E02.2   | Grep all usages of `cqrshtmx.UserID` (root) + bridge points | ≤12m   |
| E02.3   | Read `usermgmt/id.go:15` + `go-cqrs-lite/id/user_id.go:10`  | ≤12m   |
| E02.4   | Decide: re-export `id.UserID` vs change backing to ulid     | ≤12m   |
| E02.5   | Write ADR documenting the breaking change + migration       | ≤12m   |
| E02.6   | Update `usermgmt/id.go` — `UserID` definition + markers     | ≤12m   |
| E02.7   | Update `usermgmt` constructors (`NewUserID` signature)      | ≤12m   |
| E02.8   | Update all `usermgmt` internal usages (grep → fix)          | ≤12m   |
| E02.9   | Update `integration_test/` bridge code                      | ≤12m   |
| E02.10  | Update `usermgmt` tests + run `go test ./usermgmt/...`      | ≤12m   |
| E02.11  | Run full cqrs-htmx test suite (`go test ./...`)             | ≤12m   |

---

## E03 — Canonicalize `ActorID`

**Why:** Three incompatible `ActorID` types: bare `string` (cqrs-htmx context), kind-discriminated struct (usermgmt), branded ID (go-localsync). No type-level relationship.

| Task ID | Task                                                          | Effort |
| ------- | ------------------------------------------------------------- | ------ |
| E03.1   | Map all 3 `ActorID` definitions + their consumers (grep)      | ≤12m   |
| E03.2   | Decide canonical type (likely `usermgmt.ActorID` kind-struct) | ≤12m   |
| E03.3   | Write ADR for canonical ActorID + migration                   | ≤12m   |
| E03.4   | Update `cqrs-htmx/context.go:141` — replace bare string       | ≤12m   |
| E03.5   | Update all cqrs-htmx root consumers of ActorID                | ≤12m   |
| E03.6   | Decide go-localsync `ActorID`: adopt canonical or rename      | ≤12m   |
| E03.7   | Update go-localsync if adopting canonical                     | ≤12m   |
| E03.8   | Run tests across affected repos                               | ≤12m   |

---

## E05 — Remove Triplicated Error Re-exports

**Why:** `event/errors.go`, `command/errors.go`, `query/errors.go` each copy the identical 6-function re-export block. `command` and `query` add zero domain semantics — pure duplication.

| Task ID | Task                                                                  | Effort |
| ------- | --------------------------------------------------------------------- | ------ |
| E05.1   | Grep consumers of `command.NewRejection`, `command.NewConflict`, etc. | ≤12m   |
| E05.2   | Grep consumers of `query.NewRejection`, etc.                          | ≤12m   |
| E05.3   | Migrate command consumers → `event.*` or `errorfamily.*`              | ≤12m   |
| E05.4   | Migrate query consumers → `event.*` or `errorfamily.*`                | ≤12m   |
| E05.5   | Remove re-export block from `command/errors.go`                       | ≤12m   |
| E05.6   | Remove re-export block from `query/errors.go`                         | ≤12m   |
| E05.7   | Run `go test ./...` + `nix run .#lint` in go-cqrs-lite                | ≤12m   |

---

## E06 — Standardize Error Import Convention

**Why:** Three access patterns exist: via `event` re-export, direct `errorfamily`, or both in the same repo. Need ONE documented canonical path.

| Task ID | Task                                                     | Effort |
| ------- | -------------------------------------------------------- | ------ |
| E06.1   | Decide canonical: direct `go-error-family` (recommended) | ≤12m   |
| E06.2   | Document convention in `go-error-family/AGENTS.md`       | ≤12m   |
| E06.3   | Add deprecation notice to `event/errors.go` re-exports   | ≤12m   |
| E06.4   | Update `ECOSYSTEM_BOUNDARIES.md` with the decision       | ≤12m   |

---

## E07 — Break go-cqrs-lite Module Dependency Cycles

**Why:** `MODULE_BOUNDARY_ANALYSIS.md` found 4 cycles. Modularization is 90% correct — these 4 fixes get it to 100%.

| Task ID | Task                                                           | Effort |
| ------- | -------------------------------------------------------------- | ------ |
| E07.1   | Read `docs/modularization/MODULE_BOUNDARY_ANALYSIS.md`         | ≤12m   |
| E07.2   | Move `CatalogDispatcher` from `event/` → `command/`            | ≤12m   |
| E07.3   | Extract `eventtest/` as separate module (break event↔memory)   | ≤12m   |
| E07.4   | Fix `snapshot → memory` cycle (snapshot depends only on event) | ≤12m   |
| E07.5   | Break 3-node transitive cycle (memory→command→event→memory)    | ≤12m   |
| E07.6   | Remove `command/` re-exports of `event/` types                 | ≤12m   |
| E07.7   | Verify clean DAG with `nix run .#check-layers` / arch-lint     | ≤12m   |
| E07.8   | Run full test suite                                            | ≤12m   |

---

## E08 — Merge go-cqrs-lite Catalog Sub-modules

**Why:** 5 catalog sub-modules (`asyncapi`, `openapi`, `d2`, `eventcatalog`, `schema`) each have their own `go.mod` — zero isolation benefit, 6 go.mod files to maintain. Per `PROPOSAL_catalog_consolidation.md`.

| Task ID | Task                                                         | Effort |
| ------- | ------------------------------------------------------------ | ------ |
| E08.1   | Read `docs/modularization/PROPOSAL_catalog_consolidation.md` | ≤12m   |
| E08.2   | Plan file moves (sub-modules → packages under `catalog/`)    | ≤12m   |
| E08.3   | Move `catalog/schema/` → `catalog/schema/` (package only)    | ≤12m   |
| E08.4   | Move `catalog/asyncapi/` → package                           | ≤12m   |
| E08.5   | Move `catalog/openapi/` → package                            | ≤12m   |
| E08.6   | Move `catalog/d2/` → package                                 | ≤12m   |
| E08.7   | Move `catalog/eventcatalog/` → package                       | ≤12m   |
| E08.8   | Update import paths + delete sub-module go.mod files         | ≤12m   |
| E08.9   | Run tests + update README/docs                               | ≤12m   |

---

## E09 — go-localsync: Remove Bypassed query.Dispatcher

**Why:** `stack_adapters.go:1-23` admits `List`/`Count`/`GetTypes` bypass the dispatcher, calling ReadModel directly. The dispatcher is wired but dead weight in production.

| Task ID | Task                                                           | Effort |
| ------- | -------------------------------------------------------------- | ------ |
| E09.1   | Read `pkg/cqrs/stack_adapters.go` + `stack.go` dispatch wiring | ≤12m   |
| E09.2   | Remove bypassed dispatcher from production path (keep tests)   | ≤12m   |
| E09.3   | Run `go test ./...` in go-localsync                            | ≤12m   |

---

## E11 — Sync Documentation Across All Repos

**Why:** After all code changes, AGENTS.md / FEATURES.md / ROADMAP.md must reflect the new boundary decisions.

| Task ID | Task                                                      | Effort |
| ------- | --------------------------------------------------------- | ------ |
| E11.1   | Update `go-cqrs-lite/AGENTS.md` with boundary decisions   | ≤12m   |
| E11.2   | Update `cqrs-htmx/AGENTS.md` with boundary decisions      | ≤12m   |
| E11.3   | Update `go-localsync/AGENTS.md` with boundary decisions   | ≤12m   |
| E11.4   | Add cross-references to `ECOSYSTEM_BOUNDARIES.md` in each | ≤12m   |
| E11.5   | Update FEATURES.md / TODO_LIST.md per repo                | ≤12m   |
| E11.6   | Update CHANGELOG.md entries for each repo                 | ≤12m   |

---

## E10 — Decompose usermgmt God-package

**Why:** `usermgmt/` is 71 files in one package — a god-package. Per `2026-06-21_EXECUTION_PLAN.html`, split into 8 sub-packages. **Lowest priority** — large effort, internal hygiene, no consumer-facing breakage if done with backward-compatible facade.

| Task ID | Task                                                                | Effort |
| ------- | ------------------------------------------------------------------- | ------ |
| E10.1   | Read `cqrs-htmx/docs/modularization/2026-06-21_EXECUTION_PLAN.html` | ≤12m   |
| E10.2   | Create `usermgmt/domain/` skeleton (go.mod package dirs)            | ≤12m   |
| E10.3   | Move user events/state/decide → `domain/`                           | ≤12m   |
| E10.4   | Move membership files → `domain/`                                   | ≤12m   |
| E10.5   | Move tenant files → `domain/`                                       | ≤12m   |
| E10.6   | Move bot files → `domain/`                                          | ≤12m   |
| E10.7   | Move authz files → `authz/` sub-package                             | ≤12m   |
| E10.8   | Move webauthn files → `webauthn/` sub-package                       | ≤12m   |
| E10.9   | Move oauth2 files → `oauth2/` sub-package                           | ≤12m   |
| E10.10  | Move totp files → `totp/` sub-package                               | ≤12m   |
| E10.11  | Move store files → `store/` sub-package                             | ≤12m   |
| E10.12  | Create backward-compatible facade (type aliases at root)            | ≤12m   |
| E10.13  | Update all internal imports                                         | ≤12m   |
| E10.14  | Run full usermgmt test suite                                        | ≤12m   |
| E10.15  | Update usermgmt AGENTS.md + README                                  | ≤12m   |

---

## Summary Statistics

| Metric                  | Value               |
| ----------------------- | ------------------- |
| Total epics             | 11                  |
| Total tasks (≤12m each) | **85**              |
| Estimated total effort  | ~17 hours           |
| P0 (type safety) tasks  | 19                  |
| P1 (error handling)     | 16                  |
| P2 (consolidation)      | 17                  |
| P3 (cleanup/docs)       | 24                  |
| Breaking changes        | 2 (UserID, ActorID) |
| ADRs required           | 3                   |

---

## Execution Rules

1. **One task in_progress at a time** — no parallelism within a repo (test isolation).
2. **Run tests after every code-changing task** — no exceptions.
3. **Write ADR before any breaking change** — document the why, not just the what.
4. **Update CHANGELOG.md** for every epic completion.
5. **Never skip E01.5** — the user must flip the repo to public. Everything downstream benefits.
6. **E10 is deferrable** — it's internal hygiene with no consumer-facing impact. Ship P0–P2 first.
