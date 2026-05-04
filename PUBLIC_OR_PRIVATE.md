# go-cqrs-lite: Public or Private?

> **Assessment Date:** 2026-05-04
> **Repo Visibility:** Private
> **Recommendation:** Make public, with conditions (see below)

---

## Verdict

**Make it PUBLIC — conditionally.**

The project is genuinely useful, technically impressive, and ready for eyes. But it needs a license change, a version strategy, and a cleanup pass first. Go public when the conditions below are met — not before.

---

## Project Snapshot

| Metric | Value |
|---|---|
| Total Go LOC | 33,780 |
| Test LOC | 23,211 (69% of codebase) |
| Production LOC | ~10,569 |
| Test packages | 22 (all passing) |
| Test coverage | 91–100% per module |
| Benchmarks | 43 across 12 files |
| Modules | 9 (multi-module monorepo) |
| Dependencies (prod) | 4 (ulid, go-branded-id, go-faster/yaml, go-sqlmock) |
| Commits | 678 (single author) |
| Lint issues | 0 |
| TODO/FIXME markers | 0 |
| Version tags | 15 (core v1.0.0, memory v1.0.0, etc.) |
| CI | GitHub Actions (Nix-based: build, vet, test, race, lint, coverage) |

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

- 91%+ coverage across all modules, several at 100%
- Zero lint issues (60+ linter golangci-lint config)
- Zero TODO/FIXME markers
- 43 benchmarks
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

4 production dependencies total (ulid, go-branded-id, go-faster/yaml, go-sqlmock). No transitive dependency nightmare. This is a strong selling point for teams evaluating supply-chain risk.

### 6. Branded IDs prevent real bugs

`id.Of[T]` type aliases with compile-time safety (`AggregateID` ≠ `UserID` ≠ `EventID`) solve a class of bugs that costs real money in production. The go-branded-id integration is clean and well-documented.

### 7. Multi-module architecture is the right pattern

Each module is independently importable with its own `go.mod`. Consumers pay only for what they use. This is the gold standard for Go libraries.

### 8. Portfolio and reputation value

For the author, this is a showcase project. The depth of documentation (AGENTS.md, FEATURES.md, CHANGELOG.md, ADRs, planning docs, domain glossary) demonstrates engineering maturity that few open-source projects match.

### 9. Research and planning docs are educational

The `docs/research/` and `docs/planning/` directories contain genuine thought leadership: innovative CQRS projects analysis, offline-first dimensions, error taxonomy brainstorm, architecture roadmaps. Publishing these benefits the broader community.

### 10. It's already "public-ready" in structure

CI badges in README, pkg.go.dev badge, CONTRIBUTING.md, CODE_OF_CONDUCT.md, CHANGELOG.md, example app — the project already has the trappings of a public repo. The gap is small.

---

## CONTRA: Risks and Concerns

### 1. **CRITICAL: Proprietary license blocks adoption**

The current `LICENSE` file says "PROPRIETARY LICENSE" with "Unauthorized copying, distribution, modification, or use... is strictly prohibited." No one can use this library with that license. **This must change before going public.** Recommended: MIT or Apache-2.0 (consistent with Go ecosystem norms).

### 2. **Breaking changes with no stability guarantees**

The AGENTS.md documents 13 breaking changes across sessions. The root `go.mod` has no version tag. Several modules have `v0.0.0` versions for internal dependencies. Without a clear compatibility promise, early adopters will get burned.

**Mitigation:** Before going public, tag a `v1.0.0` with a compatibility promise. Document the versioning strategy per-module.

### 3. **Single-author bus factor**

678 commits, all by one person. If the author stops maintaining, the library dies. This is normal for new OSS projects but worth acknowledging.

**Mitigation:** Accept that this is a v1.0 reality. The high test coverage and documentation make it takeover-friendly.

### 4. **README has stale dependency references**

The README mentions `cockroachdb/errors` and `go-json-experiment/json` as dependencies — both were removed in Session 54. This would confuse new users.

**Mitigation:** Update README before going public.

### 5. **No real-world production usage proof**

No case studies, no "used in production by X" testimonials. The storage module uses go-sqlmock for tests — there's no evidence of actual PostgreSQL deployment. Users will ask "Has this been battle-tested?"

**Mitigation:** Be honest. Label the storage module as "tested with go-sqlmock, awaiting production validation" (FEATURES.md already does this well).

### 6. **Example app has stale transitive deps**

`example/user/go.mod` still pulls `cockroachdb/errors` transitively. Not a blocker but looks sloppy to dependency-conscious users.

### 7. **Replace directives won't work for external consumers**

All inter-module `replace` directives point to `../sibling`. For the monorepo to work as public importable modules, you need either:
- Published module versions (remove replace directives for consumers)
- Or a `go.work` that only works locally

This is a standard Go multi-module challenge but needs a documented release strategy.

### 8. **AGENTS.md is 62KB of internal session notes**

The AGENTS.md is an incredible internal tool but contains 58 sessions of detailed history. Publishing it as-is would be overwhelming. It includes references to "Session X" notes, internal decision-making processes, and competitor analysis (ActaFlow comparison report).

**Mitigation:** Either trim AGENTS.md to project-relevant architecture notes before going public, or keep it as-is as a transparency artifact. The latter is unusual but could be a differentiator.

### 9. **No Go module proxy publishing**

The modules aren't published to proxy.golang.org. The pkg.go.dev badge in README links to a 404 until modules are published with proper versions.

### 10. **Competitive exposure**

Publishing reveals the architecture, patterns, and design decisions to potential competitors. The comparison report with ActaFlow is already in the repo. If go-cqrs-lite represents proprietary competitive advantage, going public removes that.

**Counter-argument:** For a library, network effects dominate. More users = more contributors = more value. The competitive advantage is the community, not the code.

---

## Conditions for Going Public

Before making the repo public, address these in order:

### Must-Do (Blocking)

1. **Change LICENSE to MIT or Apache-2.0.** Without this, going public is pointless — no one can legally use the code.
2. **Update README.md.** Remove stale dependency references (`cockroachdb/errors`, `go-json-experiment/json`). Add accurate current deps.
3. **Fix example/user/go.mod.** Run `go mod tidy` to clean stale transitive deps.
4. **Decide on AGENTS.md.** Either trim session-by-session history to architecture-relevant content, or embrace full transparency.
5. **Remove or .gitignore internal artifacts.** `coverage.out`, `report/`, `user` binary should not be in the public eye (they're gitignored but verify).

### Should-Do (Strongly Recommended)

6. **Tag a `v1.0.0`** for each module with a compatibility promise. Write a `VERSIONING.md` explaining per-module versioning strategy.
7. **Add a "Production Readiness" section to README.** Be honest about what's battle-tested (core, memory, middleware) vs. what needs real-world validation (storage).
8. **Publish modules to proxy.golang.org.** Ensure `go get github.com/larsartmann/go-cqrs-lite/core` works.
9. **Write a blog post or Twitter thread** announcing the library with the unique selling points (Decider pattern, auto-docs, branded IDs, zero-dep core).
10. **Set up GitHub Discussions** for community questions.

### Nice-to-Have

11. Add an issue template for bug reports and feature requests.
12. Add GitHub Actions for auto-publishing tagged releases.
13. Consider a Go.dev documentation site (pkg.go.dev auto-generates this, but a custom site could showcase the architecture).

---

## The Strategic Question

This is ultimately a question of what you want go-cqrs-lite to become:

| Path | Outcome | Tradeoff |
|---|---|---|
| **Stay private** | Personal toolkit, full control, no community maintenance burden | No network effects, no reputation building, no external validation |
| **Go public now** | First-mover advantage, start building community early | Users hit rough edges, support burden, breaking changes hurt early adopters |
| **Go public after conditions met** | Professional first impression, higher-quality adoption | Delayed network effects, more prep work |

**Recommendation:** Go public after conditions 1–5 (must-do). The should-do items can follow within 2 weeks of going public.

The library is good enough. The code quality speaks for itself. The gap between "impressive private project" and "adoptable public library" is small and entirely fixable.

---

## Competitive Landscape Summary

| Library | Language | Transport-agnostic | Auto-docs | Branded IDs | Decider Pattern | Error Taxonomy |
|---|---|---|---|---|---|---|
| **go-cqrs-lite** | Go | ✅ | ✅ AsyncAPI + EventCatalog + D2 | ✅ | ✅ | ✅ (5 families) |
| Watermill | Go | ❌ (transport-first) | ❌ | ❌ | ❌ | ❌ |
| Eventuous | Go/.NET | ⚠️ | ❌ | ❌ | ⚠️ (.NET only) | ❌ |
| go-eventstream | Go | ✅ | ❌ | ❌ | ❌ | ❌ |
| commanded | Elixir | ✅ | ❌ | ❌ | ❌ | ❌ |

**go-cqrs-lite is the only Go library offering all six features simultaneously.** This is a defensible niche.

---

*Assessment by Crush (AI architecture review), commissioned by the project author.*
