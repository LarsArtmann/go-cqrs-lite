# go-cqrs-lite: Public or Private?

> **Assessment Date:** 2026-05-04 | **Last Updated:** 2026-05-23
> **Repo Visibility:** Private
> **Recommendation:** Make public, with conditions (see below)

---

## Verdict

**Make it PUBLIC — conditionally.**

The project is genuinely useful, technically impressive, and ready for eyes. **4 of 5 must-do conditions are resolved.** The only remaining blocker is the LICENSE change (still PROPRIETARY). Once that single file is changed, the project is ready to go public.

---

## Project Snapshot

| Metric              | Value                                                              |
| ------------------- | ------------------------------------------------------------------ |
| Total Go LOC        | ~49,300                                                            |
| Test LOC            | 33,121 (67% of codebase)                                           |
| Production LOC      | ~16,100                                                            |
| Test packages       | 34 (all passing)                                                   |
| Test coverage       | 80–100% per module (most >90%)                                     |
| Benchmarks          | 56 across 13 files                                                 |
| Modules             | 12 (multi-module monorepo)                                         |
| Dependencies (prod) | 9 unique across all modules (core: 3, catalog: 1, middleware: 2, storage: 3) |
| Commits             | 998 (single author)                                                |
| Lint issues         | 0                                                                  |
| TODO/FIXME markers  | 0                                                                  |
| Version tags        | 42 across 10 modules (core v1.5.0, memory v1.3.0, etc.)            |
| CI                  | GitHub Actions (Nix-based: build, vet, test, race, lint, coverage) |

---

## Progress Since Initial Assessment (2026-05-04 → 2026-05-23)

| What changed | Detail |
| --- | --- |
| +320 commits | 678 → 998 (47% growth) |
| +15,500 LOC | 33,780 → ~49,300 total |
| +3 modules | 9 → 12 (added `sync`, `example/todo`, `example/user` as separate go.mod) |
| +13 benchmarks | 43 → 56 |
| +27 version tags | 15 → 42 across 10 modules |
| README stale deps fixed | `cockroachdb/errors` and `go-json-experiment/json` references removed |
| example/user/go.mod cleaned | No more stale transitive deps |
| AGENTS.md trimmed | 62KB → 39KB; session history extracted to `docs/sessions/` |
| Storage module evolved | Now has 3 real database backends (Pebble, SQLite, Turso), not just mocks |
| OpenTelemetry integration | Middleware now uses OpenTelemetry for metrics and tracing |
| New module: `sync` | Distributed sync primitives (vector clocks, conflict resolution) |
| LICENSE | Still PROPRIETARY — the only remaining blocker |

---

## PRO: Why This Should Be Public

### 1. Fills a real gap in the Go ecosystem

Go has few opinionated-but-unopinionated CQRS/ES libraries. The landscape is:

- **Watermill** — heavy, Kafka/transport-first, not focused on domain modeling
- **Eventuous** — .NET primarily, Go is secondary
- **Commanded** — Elixir only
- **go-eventstream** — minimal, no aggregate support, no documentation generation

go-cqrs-lite occupies a specific niche: **a library (not framework) that gives you CQRS building blocks without dictating your transport, broker, or database**. This is genuinely valuable and under-served.

### 2. Exceptional engineering quality

- 80%+ coverage across all modules, many at 90–100%
- Zero lint issues (60+ linter golangci-lint config)
- Zero TODO/FIXME markers
- 56 benchmarks
- File size limit (250 lines), function size limit (30 lines)
- Compile-time interface checks everywhere
- Sentinel errors with `errors.Is`-compatible classification taxonomy
- No `any` types (enforced by lint)
- Race detector passing

This is not a weekend project. It's a disciplined, professional-grade codebase that would stand up to scrutiny from any senior Go engineer.

### 3. The Decider pattern is a differentiator

The `core/decider` package offers a pure-function alternative to the traditional OO aggregate (stateful `Root` interface with 9 methods). This aligns with cutting-edge thinking from the functional CQRS community (EventStoreDB's Decider pattern, DCB-style functional event sourcing). Having this as a first-class, tested option in Go is a genuine contribution.

### 4. Auto-documentation generation is unique

The `catalog` module generating AsyncAPI 3.0 YAML, EventCatalog MDX, and D2 diagrams from Go struct tags and reflection is a standout feature. No other Go CQRS library does this. This alone could attract users.

### 5. Minimal dependency footprint

9 unique production dependencies across all modules, but each module stays lean: core has 3 (ulid, go-branded-id, go-error-family), catalog has 1 (go-faster/yaml), middleware has 2 (opentelemetry), storage has 3 (pebble, sqlite, turso). Memory, testhelpers, projection, integration, and sync have zero external production deps. No transitive dependency nightmare. This is a strong selling point for teams evaluating supply-chain risk.

### 6. Branded IDs prevent real bugs

`id.Of[T]` type aliases with compile-time safety (`AggregateID` ≠ `UserID` ≠ `EventID`) solve a class of bugs that costs real money in production. The go-branded-id integration is clean and well-documented.

### 7. Multi-module architecture is the right pattern

Each module is independently importable with its own `go.mod`. Consumers pay only for what they use. This is the gold standard for Go libraries. 12 modules now (up from 9 at initial assessment), including `sync` (distributed sync primitives with vector clocks) and two example apps.

### 8. Multi-backend storage module

The storage module now supports three real database backends (Pebble, SQLite, Turso) — not just mocks. This demonstrates the library's transport-agnostic design works with real persistence engines, even if production validation is still pending.

### 9. Portfolio and reputation value

For the author, this is a showcase project. The depth of documentation (AGENTS.md, FEATURES.md, CHANGELOG.md, ADRs, planning docs, domain glossary) demonstrates engineering maturity that few open-source projects match.

### 10. Research and planning docs are educational

The `docs/research/` and `docs/planning/` directories contain genuine thought leadership: innovative CQRS projects analysis, offline-first dimensions, error taxonomy brainstorm, architecture roadmaps. Publishing these benefits the broader community.

### 11. It's already "public-ready" in structure

CI badges in README, pkg.go.dev badge, CONTRIBUTING.md, CODE_OF_CONDUCT.md, CHANGELOG.md, two example apps (user + todo) — the project already has the trappings of a public repo. The gap is small.

---

## CONTRA: Risks and Concerns

### 1. **CRITICAL: Proprietary license blocks adoption**

The current `LICENSE` file says "PROPRIETARY LICENSE" with "Unauthorized copying, distribution, modification, or use... is strictly prohibited." No one can use this library with that license. **This must change before going public.** Recommended: MIT or Apache-2.0 (consistent with Go ecosystem norms).

### 2. **Breaking changes with no stability guarantees**

The AGENTS.md documents 13 breaking changes across sessions. The root `go.mod` has no version tag. Several modules have `v0.0.0` versions for internal dependencies. Without a clear compatibility promise, early adopters will get burned.

**Mitigation:** Before going public, tag a `v1.0.0` with a compatibility promise. Document the versioning strategy per-module.

### 3. **Single-author bus factor**

998 commits, all by one person. If the author stops maintaining, the library dies. This is normal for new OSS projects but worth acknowledging.

**Mitigation:** Accept that this is a v1.0 reality. The high test coverage and documentation make it takeover-friendly.

### 4. ~~**README has stale dependency references**~~ ✅ RESOLVED

The README previously mentioned `cockroachdb/errors` and `go-json-experiment/json` as dependencies — both were removed in Session 54. **This has been fixed.** The README now accurately lists current dependencies (ulid, go-branded-id, go-error-family, go-faster/yaml).

### 5. **No real-world production usage proof**

No case studies, no "used in production by X" testimonials. The storage module now has real database backends (Pebble, SQLite, Turso) but no evidence of production deployment yet. Users will ask "Has this been battle-tested?"

**Mitigation:** Be honest. Label the storage module as "tested with real database drivers, awaiting production validation" (FEATURES.md already does this well).

### 6. **Example apps have transitive deps from storage backends**

`example/user/go.mod` is now clean (cockroachdb/errors removed). `example/todo/go.mod` pulls `cockroachdb/errors` as an indirect dep via `cockroachdb/pebble` (storage backend). Not a blocker — it's a transitive dep, not a direct one.

### 7. **Replace directives won't work for external consumers**

All inter-module `replace` directives point to `../sibling`. For the monorepo to work as public importable modules, you need either:

- Published module versions (remove replace directives for consumers)
- Or a `go.work` that only works locally

This is a standard Go multi-module challenge but needs a documented release strategy.

### 8. ~~**AGENTS.md is 62KB of internal session notes**~~ ✅ MOSTLY RESOLVED

The AGENTS.md is an incredible internal tool (now ~39KB, down from ~62KB). Session-by-session history has been extracted to `docs/sessions/SESSION_HISTORY.md`, leaving AGENTS.md focused on architecture and reference material. Still contains detailed internal decision-making and competitive analysis, but significantly more digestible than before. 95+ sessions of development history are now organized separately.

**Mitigation:** The extraction to `SESSION_HISTORY.md` is a good middle ground. AGENTS.md can be published as-is — it's now a strong architecture doc rather than a raw session dump. Keep `docs/sessions/` as a transparency artifact or .gitignore it.

### 9. **No Go module proxy publishing**

The modules aren't published to proxy.golang.org. The pkg.go.dev badge in README links to a 404 until modules are published with proper versions.

### 10. **Competitive exposure**

Publishing reveals the architecture, patterns, and design decisions to potential competitors. The comparison report with ActaFlow is already in the repo. If go-cqrs-lite represents proprietary competitive advantage, going public removes that.

**Counter-argument:** For a library, network effects dominate. More users = more contributors = more value. The competitive advantage is the community, not the code.

---

## Conditions for Going Public

Before making the repo public, address these in order:

### Must-Do (Blocking)

1. **Change LICENSE to MIT or Apache-2.0.** Without this, going public is pointless — no one can legally use the code. *(Still PROPRIETARY — unchanged.)*
2. ~~**Update README.md.** Remove stale dependency references (`cockroachdb/errors`, `go-json-experiment/json`). Add accurate current deps.~~ ✅ **RESOLVED** — README now accurately lists current deps.
3. ~~**Fix example/user/go.mod.** Run `go mod tidy` to clean stale transitive deps.~~ ✅ **RESOLVED** — `example/user/go.mod` is clean. `example/todo/go.mod` has `cockroachdb/errors` as indirect via `pebble` (transitive, acceptable).
4. ~~**Decide on AGENTS.md.** Either trim session-by-session history to architecture-relevant content, or embrace full transparency.~~ ✅ **MOSTLY RESOLVED** — Session history extracted to `docs/sessions/SESSION_HISTORY.md`. AGENTS.md is now ~39KB (down from ~62KB), focused on architecture and reference.
5. ~~**Remove or .gitignore internal artifacts.** `coverage.out`, `report/`, `user` binary should not be in the public eye~~ ✅ **VERIFIED** — `coverage.out`, `report/`, and `user` binary are all gitignored and not tracked in git.

### Should-Do (Strongly Recommended)

6. **Tag a `v1.0.0`** for each module with a compatibility promise. Write a `VERSIONING.md` explaining per-module versioning strategy. *(Partially done — 42 version tags exist across 10 modules. Core is at v1.5.0. No VERSIONING.md yet.)*
7. **Add a "Production Readiness" section to README.** Be honest about what's battle-tested (core, memory, middleware) vs. what needs real-world validation (storage).
8. **Publish modules to proxy.golang.org.** Ensure `go get github.com/larsartmann/go-cqrs-lite/core` works.
9. **Write a blog post or Twitter thread** announcing the library with the unique selling points (Decider pattern, auto-docs, branded IDs, minimal deps per module).
10. **Set up GitHub Discussions** for community questions.

### Nice-to-Have

11. Add an issue template for bug reports and feature requests.
12. Add GitHub Actions for auto-publishing tagged releases.
13. Consider a Go.dev documentation site (pkg.go.dev auto-generates this, but a custom site could showcase the architecture).

---

## The Strategic Question

This is ultimately a question of what you want go-cqrs-lite to become:

| Path                               | Outcome                                                         | Tradeoff                                                                    |
| ---------------------------------- | --------------------------------------------------------------- | --------------------------------------------------------------------------- |
| **Stay private**                   | Personal toolkit, full control, no community maintenance burden | No network effects, no reputation building, no external validation          |
| **Go public now**                  | First-mover advantage, start building community early           | Users hit rough edges, support burden, breaking changes hurt early adopters |
| **Go public after conditions met** | Professional first impression, higher-quality adoption          | Delayed network effects, more prep work                                     |

**Recommendation:** Go public once condition 1 (LICENSE change) is done — it's the only remaining blocker. Conditions 2–5 are resolved. The should-do items can follow within 2 weeks of going public.

The library is good enough. The code quality speaks for itself. 4 of 5 must-do conditions are resolved. The gap between "impressive private project" and "adoptable public library" is now essentially one file: `LICENSE`.

---

## Competitive Landscape Summary

| Library          | Language | Transport-agnostic   | Auto-docs                                    | Branded IDs | Decider Pattern | Error Taxonomy  | Multi-backend Storage |
| ---------------- | -------- | -------------------- | -------------------------------------------- | ----------- | --------------- | --------------- | -------------------- |
| **go-cqrs-lite** | Go       | ✅                   | ✅ AsyncAPI + EventCatalog + D2 + OpenAPI    | ✅          | ✅              | ✅ (5 families) | ✅ (Pebble/SQLite/Turso) |
| Watermill        | Go       | ❌ (transport-first) | ❌                                           | ❌          | ❌              | ❌              | ⚠️ (adapter-based)    |
| Eventuous        | Go/.NET  | ⚠️                   | ❌                                           | ❌          | ⚠️ (.NET only)  | ❌              | ⚠️ (EventStoreDB)     |
| go-eventstream   | Go       | ✅                   | ❌                                           | ❌          | ❌              | ❌              | ❌                    |
| commanded        | Elixir   | ✅                   | ❌                                           | ❌          | ❌              | ❌              | ⚠️ (EventStoreDB)     |

**go-cqrs-lite is the only Go library offering all seven features simultaneously.** This is a defensible niche.

---

_Assessment by Crush (AI architecture review), commissioned by the project author._
