# Comprehensive TODO Plan — go-cqrs-lite Ecosystem

**Created:** 2026-07-18 02:00
**Sources:** Self-review (50 items) + Corrective audit (7 items) + v4 migration cleanup (50 items)
**Deduplicated:** ~115 raw items → 72 unique tasks after removing completed/false-alarm items
**Split:** Every task ≤12 min. Large tasks broken into sub-tasks.

---

## Priority Legend

| Priority | Meaning | Count |
|----------|---------|-------|
| **P0** | Blocks shipping or other work | 5 |
| **P1** | High impact, customer-facing | 12 |
| **P2** | Technical debt, type safety | 22 |
| **P3** | Documentation, lint improvements | 16 |
| **P4** | CI, process, nice-to-have | 17 |

---

## P0 — Blocking (Must Do First)

| # | Task | Project | Effort | Impact | Why |
|---|------|---------|--------|--------|-----|
| 1 | Create root-level `sqlc.yaml` symlink to `internal/database/sqlc.yaml` | KeyCountdown | 2m | Critical | Blocks ALL pre-commit hook commits |
| 2 | Verify BuildFlow pre-commit hook passes with sqlc fix | KeyCountdown | 5m | Critical | Confirms #1 works |
| 3 | Audit go-localsync commits since v0.3.0 for tagging readiness | go-localsync | 5m | High | Prerequisite for #4 |
| 4 | Tag go-localsync v0.4.0 (v4 migration included) | go-localsync | 3m | High | Unblocks #5, #6 |
| 5 | Remove `replace go-localsync` directive + `go get github.com/larsartmann/go-localsync@v0.4.0` | github-local-sync | 5m | High | Eliminates local replace hack |
| 6 | Update go-localsync dep from pseudo-version to v0.4.0 + `go mod vendor` | standard-bug-tracking-schema | 5m | Medium | Consistency |

---

## P1 — High Impact (Customer-Facing Quality)

### Pre-Existing Test Failures (7 projects, 1 task each)

| # | Task | Project | Effort | Impact | Why |
|---|------|---------|--------|--------|-----|
| 7 | Fix Zlota44 discovery test (SQL syntax error) | Zlota44 | 10m | High | SQL bug in production code |
| 8 | Fix Standup-Killer test (missing config) | Standup-Killer | 8m | High | Config setup issue |
| 9 | Fix accountability-system test (route conflict) | accountability-system | 12m | High | Route registration bug |
| 10 | Fix standard-bug-tracking-schema test (cache + property) | standard-bug-tracking-schema | 12m | Medium | Test logic bugs |
| 11 | Fix KeyCountdown webauthn test (performance timeout) | KeyCountdown | 10m | Medium | Flaky timing test |
| 12 | Fix CV test (NLP fuzz, chat validation) | CV | 12m | Medium | Test correctness |
| 13 | Fix Kernovia test (TypeSpec, web accessibility) | Kernovia | 12m | Medium | Test correctness |

### Ecosystem-Wide Verification

| # | Task | Scope | Effort | Impact | Why |
|---|------|-------|--------|--------|-----|
| 14 | Run `golangci-lint` on KeyCountdown, DiscordSync, reports, crush-daily | 4 high-risk projects | 12m | High | Catch lint issues from --no-verify commits |
| 15 | Run `golangci-lint` on SEC, Zlota44, ChastityAPI, Standup-Killer | 4 more projects | 12m | High | Same |
| 16 | Run `golangci-lint` on remaining 17 consumer projects | All others | 12m | Medium | Full coverage |
| 17 | Run `go mod tidy` on all 25 consumer projects | Ecosystem | 10m | Medium | Clean go.sum files |
| 18 | Verify vendor dirs up-to-date in 6 vendored projects | 6 projects | 8m | Medium | Prevent build breaks |

---

## P2 — Technical Debt & Type Safety

### Instant Type Enhancements (go-cqrs-lite event/time_types.go)

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 19 | Add `Instant.Zero` constant | event/time_types.go | 3m | Medium | Represent "no timestamp" without nil |
| 20 | Add `Instant.Sub(other Instant) time.Duration` | event/time_types.go | 3m | Low | Convenience method |
| 21 | Add `Instant.Add(d Duration) Instant` | event/time_types.go | 3m | Low | Always-UTC arithmetic |
| 22 | Decide on CBOR tag 1 vs bare int64 (keep int64 — internal format) | event/time_types.go | 2m | Low | Document the decision |

### WallTime Type Enhancements

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 23 | Add `WallTime.MarshalCBOR()` / `UnmarshalCBOR()` | event/time_types.go | 8m | Medium | Explicit CBOR handling |
| 24 | Add `WallTime.PreviousOccurrence()` | event/time_types.go | 5m | Low | Inverse of NextOccurrence |
| 25 | Add `WallTime.IsValid()` method | event/time_types.go | 3m | Low | Check without constructor |

### New Types

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 26 | Design `Date` type (calendar date without time) | event/date.go (new) | 12m | High | For employment dates, sex dates, etc. |
| 27 | Add `Date` tests (construction, validation, JSON, CBOR) | event/date_test.go (new) | 12m | High | Type safety |
| 28 | Consider `event.NewFromStruct` / `NewFromBytes` aliases | event/event.go | 10m | Medium | API clarity (New vs NewEvent confusion) |

### C013 Lint Rule Improvements

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 29 | C013: Detect nested `time.Time` in embedded structs | c013.go | 12m | High | Catches deeply nested payload fields |
| 30 | C013: Add auto-fix suggestion with exact replacement code | c013.go | 8m | Medium | Helps consumers migrate |
| 31 | C013: Detect `time.Now()` without `.UTC()` near event constructors | c013.go (or new c014) | 12m | Medium | Catch encoding-time bugs |

### Consumer Code Fixes

| # | Task | Project | Effort | Impact | Why |
|---|------|---------|--------|--------|-----|
| 32 | Fix SwettySwipperWeb `EXIF.DateTaken` — convert `*time.Time` to `string` | SwettySwipperWeb | 10m | Low | Never populated in prod, but struct is wrong |
| 33 | Fix DiscordSync `PollPayload.Expiry` — add UTC normalization at adapter | DiscordSync | 8m | Low | Expiry never set in prod, but type is wrong |
| 34 | Refactor KeyCountdown `LiteToDomainEvent` to avoid CBOR→JSON round-trip | KeyCountdown | 12m | Medium | Wasteful decode/re-encode on every event read |
| 35 | Audit Standup-Killer `domain.Now()` seam returns UTC | Standup-Killer | 5m | Medium | Verify the model project is correct |
| 36 | Audit Standup-Killer `CheckinSubmittedPayload.Date` type | Standup-Killer | 5m | Low | Verify domain.Date handles TZ |

### Migration Cleanup

| # | Task | Scope | Effort | Impact | Why |
|---|------|-------|--------|--------|-----|
| 37 | Fix Kernovia `nix-fmt` pre-commit hook (invalid UTF-8 in scripts) | Kernovia | 10m | Medium | Blocks future commits |
| 38 | Fix standard-bug-tracking-schema `nix-fmt` pre-commit hook | standard-bug-tracking-schema | 10m | Medium | Same |
| 39 | Audit `go.work` files — track or gitignore | All projects | 8m | Low | Consistency |
| 40 | Review `go.work.sum` tracking in cqrs-htmx, SEC, CV, InboxClean | 4 projects | 5m | Low | Consistency |

---

## P3 — Documentation

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 41 | Update `V3_MIGRATION.md` with CBOR + `time.Time` gotcha | go-cqrs-lite docs | 8m | High | Future migrators need this |
| 42 | Write CHANGELOG.md entry for v4.0.2 changes | go-cqrs-lite | 8m | Medium | Release notes |
| 43 | Write ADR for Instant/WallTime types | go-cqrs-lite docs/adr | 12m | Medium | Design decision record |
| 44 | Add timezone handling section to `event/README.md` | go-cqrs-lite event | 5m | Low | Discoverability |
| 45 | Update FEATURES.md with timezone-safe types | go-cqrs-lite | 5m | Low | Feature inventory |
| 46 | Write dedicated consumer migration guide (standalone) | go-cqrs-lite docs | 12m | Medium | Step-by-step checklist |
| 47 | Add timezone testing guide (DST edge cases) | go-cqrs-lite docs | 10m | Low | Testing guidance |
| 48 | Document SEC pseudo-version workaround in AGENTS.md | SEC | 5m | Low | Knowledge capture |
| 49 | Annotate previous status report `2026-07-17_07-39_...` | docs/status | 5m | Low | Historical accuracy |
| 50 | Update planning doc `2026-07-18_00-18_...` with task status | go-cqrs-lite docs | 8m | Low | Plan accuracy |
| 51 | Add `GOEXPERIMENT=jsonv2` note to each consumer README | All 25 projects | 12m | Low | Onboarding |
| 52 | Document `GOEXPERIMENT=jsonv2` requirement in flake.nix devShells | All projects with flake.nix | 12m | Medium | Build reproducibility |

---

## P4 — CI, Process & Nice-to-Have

### CI Improvements

| # | Task | Scope | Effort | Impact | Why |
|---|------|-------|--------|--------|-----|
| 53 | Add `GOEXPERIMENT=jsonv2` to all CI workflows | All consumers | 12m | Medium | Build reproducibility in CI |
| 54 | Add C013 to BuildFlow pre-commit pipeline | BuildFlow config | 8m | Medium | Prevent future time.Time in payloads |
| 55 | Run C013 in CI across all consumer projects | CI configs | 12m | Medium | Continuous verification |
| 56 | Add CI check for `encoding/json/v2` compatibility | go-cqrs-lite CI | 12m | Low | Catch regressions |
| 57 | Test `tag-release.sh` with dry-run (strip + restore logic) | go-cqrs-lite | 10m | Medium | Verify script works |
| 58 | Add integration test for `tag-release.sh` strip logic | go-cqrs-lite | 12m | Low | Prevent regressions |
| 59 | Automate `replace` directive stripping in CI | go-cqrs-lite CI | 12m | Low | Not just in tag-release |

### New Lint Rules

| # | Task | File | Effort | Impact | Why |
|---|------|------|--------|--------|-----|
| 60 | C014: Detect `time.Local` usage in event-related code | c014.go (new) | 12m | Medium | Prevent local time in events |
| 61 | C015: Detect missing timezone validation at API boundaries | c015.go (new) | 12m | Medium | Reject naive timestamps |

### Tooling

| # | Task | Scope | Effort | Impact | Why |
|---|------|-------|--------|--------|-----|
| 62 | Add `go-cqrs-lite` version matrix to dependency graph tool | go-cqrs-lite | 12m | Low | Track versions |
| 63 | Add `replace` directive warnings to dep graph tool | go-cqrs-lite | 8m | Low | Surface local replaces |
| 64 | Create migration verification script (all consumers same version) | scripts | 10m | Medium | Future migrations |
| 65 | Review `who-uses` output for version mismatches | go-cqrs-lite | 5m | Low | Audit |
| 66 | Audit duplicate dependencies (v3 + v4 simultaneously) | All projects | 10m | Medium | Clean removal of v3 |
| 67 | Review leftover `json.Unmarshal` on event payloads | All projects | 10m | Medium | Migration completeness |
| 68 | Add `flake.nix` check for `GOEXPERIMENT=jsonv2` in devShells | Template | 8m | Low | Standardize |
| 69 | Create pre-flight checklist for mass migrations | docs | 10m | Low | Process improvement |
| 70 | Consider Go workspace (`go.work`) at `/home/lars/projects/` | Root | 5m | Low | Cross-project development |
| 71 | Clean up `/tmp` migration scripts | /tmp | 2m | None | Tidiness |
| 72 | Consider lockstep versioning (go-localsync, cqrs-htmx, go-cqrs-lite) | Design | 10m | Low | Reduce version drift |

---

## Summary Metrics

| Priority | Tasks | Total Est. Effort | Avg/task |
|----------|-------|-------------------|----------|
| **P0** (Blocking) | 6 | 25 min | 4.2 min |
| **P1** (High Impact) | 12 | 127 min | 10.6 min |
| **P2** (Tech Debt) | 22 | 189 min | 8.6 min |
| **P3** (Docs) | 12 | 92 min | 7.7 min |
| **P4** (CI/Process) | 20 | 176 min | 8.8 min |
| **TOTAL** | **72** | **609 min** | **8.5 min** |

## Already Completed (Not Counted Above)

- Root cause fix (CBOR TimeUnixDynamic) ✅
- Instant + WallTime types ✅
- C013 lint rule created ✅
- defaultClock returns UTC ✅
- event/doc.go time handling note ✅
- Tags created + pushed (codec/event/pebble/watermill v4.0.2) ✅
- All 25 consumer go.mod updated to v4.0.2 ✅
- go vet clean across 25 consumers ✅
- C013 run against 26 consumers ✅
- KeyCountdown struct literal + shadowing bugs fixed ✅
- DiscordSync CommunicationDisabledUntil UTC-normalized ✅
- 3 previously blocked migration commits resolved ✅
- All 25 consumer projects build ✅
- Wall-clock "mistake" verified as false alarm (all instants) ✅
- crush-daily/SEC blanket sed verified as clean ✅
