# Status Report — idempotency module usage review + doc reconciliation

**Date:** 2026-11-17 12:26 CET
**Session:** interactive, two prompts (module-existence question, then fleet usage review)
**Branch:** `master`

## Scope line

**Verified here (fresh commands in this session):**

- `project-dependency-graph who-uses "…/idempotency" --direct-only=false` → 7 consumers, all `v4.4.0`.
- `project-dependency-graph who-uses "…/idempotency/sqlstore" --direct-only=false` → 3 consumers, all `v4.4.3`.
- Fleet-wide `grep` over `~/projects` for `go-cqrs-lite/idempotency` in `go.mod`/`*.go` (excludes `.corpus-pins/`, `ci-siblings/`, `BuildFlow.vendor.bak/`) → zero real `kvstore` consumers; 17 files importing the parent path.
- `ls idempotency/` and `find … -name go.mod | grep idempotency` → only `idempotency/sqlstore/v4` exists in the tree; the parent module directory is gone.
- `git log --diff-filter=D -- idempotency/go.mod` → deletion in `875bb689b` (2026-08-15); `git for-each-ref` → tag `idempotency/v4.4.0` created 2026-08-12 (tag predates deletion).
- Consumer wiring read at: `InboxClean/internal/emailcqrs/setup.go`, `browser-history/api/server.go` + `server_setup.go`, `games/KeyCountdown/internal/container/packages/cqrs.go`, `DiscordSync/cmd/discordsync/{init,idempotency_sweeper}.go`, `games/SEC/internal/cqrs/app/app.go`, `github-local-sync/internal/branch/service.go`, `Zlota44/internal/app/{wire,dispatcher,idempotency_sqlite}.go`, `games/SwettySwipperWeb/services/api/{server,dispatcher,idempotency_dedup}.go`.
- Gates run: `cmd/doc-check` → "All 1391 references valid across 57 package(s)"; `go test -run TestRecipesCatalogCoversFile` → ok; `gofmt -l` on 5 edited Go files → empty; `scripts/check-adr-numbering.sh` → OK (165 ADRs); `scripts/check-versions-manifest.sh --check` → fresh (117 trains).
- `git status` clean; session edits landed in `e8dc8b8a8` (10 files) and `b42d3de72` (ADR-0167 addendum).

**NOT verified here:**

- No `nix run .#verify` / `#lint` / `#test` / `#build` was run. All Go edits were comments; verification was gofmt + doc-check + the two gates above.
- No `nix run .#check-md-go` after the ADR addendum (prose only, no code fence).
- The frozen tags' proxy/pkg.go.dev resolution was inferred from local tags, not fetched.
- Foreign repos were not built: Swetty's dangling `replace`, DiscordSync/KeyCountdown/browser-history migration posture, Zlota44's store are read-only observations.
- Test-side usage in the consumers was not audited beyond incidental grep hits.
- `TODO_LIST.md` was NOT updated (HARVEST deferred per the user's "wait for instructions").

---

## TL;DR

Two dead-or-frozen modules, not one: `idempotency/kvstore/v4` is deleted with zero users (already known), and **`idempotency/v4` itself has no source at HEAD — it is a frozen alias shim over `go-idempotency` that 7 fleet apps still import**. `idempotency/sqlstore/v4` is deprecated, untagged since 2026-10-08, and still has 3 consumers. The session cleaned the live docs that described `kvstore` as a usable module and corrected ADR-0167's fleet census with a dated addendum. The consumer review found two unbounded-table risks (no `Sweep`), one satellite-SQL reimplementation, and two foreign-repo hygiene defects left untouched.

---

## a) FULLY DONE

| # | Work | Evidence |
|---|------|----------|
| 1 | Established that `idempotency/kvstore/v4` has **no source and no consumers** (only pin-snapshot copies under `.corpus-pins/` and `ci-siblings/`, both census-excluded) | fleet grep; depgraph shows no kvstore consumer; `ls idempotency/` |
| 2 | Removed the last live-doc claims that kvstore is a usable module: `AGENTS.md:345` tier list, `AGENTS.md:361` dependency bullet, `docs/architecture-understanding/SEVEN-TIER-MODEL.md:98`, `.agents/skills/go-cqrs-lite/SKILL.md:159` + dedup table row, `.agents/skills/go-cqrs-lite/references/recipes.md:992`, `docs/agents/gotchas-testing.md:19` | commit `e8dc8b8a8`; doc-check green (1391 refs) |
| 3 | Removed stale references to the also-deleted `transport/grpc` in the race-threshold note and in `metaengine/v4/enginetest/race_{on,off}.go` | `e8dc8b8a8`; gofmt clean |
| 4 | Fixed dangling pointers in `idempotency/sqlstore/v4/race_{on,off}_test.go` to the deleted `idempotency/kvstore/race_*_test.go` | `e8dc8b8a8` |
| 5 | Corrected `integration/v4/idempotency_ttl_validation_test.go`'s contract comment to list the real implementations (memory, sqlstore), naming kvstore's 2026-10-17 deletion | `e8dc8b8a8` |
| 6 | Produced a per-consumer usage inventory for both modules (7 + 3 modules, API symbols, wiring shape, store choice) | this report §below; depgraph + file reads |
| 7 | Found that the parent `idempotency/v4` module is a **pure alias shim with no source at HEAD** (deleted 2026-08-15 in the ADR-0128 shim sweep), frozen at tag `v4.4.0` (2026-08-12) | `git show idempotency/v4.4.0:idempotency/alias.go`; `git log --diff-filter=D` |
| 8 | Confirmed every consumer of the parent already resolves `go-idempotency v0.3.2` via MVS (so the frozen shim does not pin old code) | per-repo `go.mod` greps; browser-history `api/go.mod:123` |
| 9 | Confirmed `idempotency/sqlstore/v4` is untagged since 2026-10-08 (`scripts/untagged-trains.txt:89`) and every consumer pins the final tag `v4.4.3` | tag list; depgraph |
| 10 | Added a dated addendum to ADR-0167 correcting the 2026-11-15 fleet census (5 → 3 direct modules, duplicate-checkout and indirect explanation) and recording the parent-shim deletion | commit `b42d3de72`, `docs/adr/0167-…-facade.md:72` |
| 11 | Verified the repo-level gates still pass after the doc edits (`check-adr-numbering`, `check-versions-manifest --check`, doc-check, recipes catalog) | command outputs in scope line |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker | Effort |
|---|-----------|--------------|---------|--------|
| 1 | kvstore doc sweep over live docs | Deliberate residue left in historical records (ADRs 0046/0062/0065/0128/0152/0158, `docs/planning/`, `docs/status/`, `docs/feedback/`, `docs/architecture-understanding/2026-07-23_*`) and in intent-preserving artifacts (`versions.json:38`, README tag-truth row, `scripts/untagged-trains.txt:88`) | Policy: ADRs/reports are dated records; version/README rows are tag truth per ADR-0152 | — |
| 2 | ADR-0167 corrected via addendum | The ADR body still states "5 direct consumers" and "the parent survives"; a reader who stops at the Context section gets the old picture | Addendum discipline forbids rewriting decision bodies | S |
| 3 | Fleet usage review of both modules | Test-only usage not audited; foreign repos not built; `Seen`/`Record` direct-call claim is grep-based (prod files only) | Out of scope for the prompt | M |
| 4 | Documentation of the parent-shim finding | Skill refs still route new code to `idempotency` + `middleware`: `.agents/skills/go-cqrs-lite/references/core.md:204`; `modules.md:63` also still lists `KVStore` in the parent row | None — just not edited in this session | S |
| 5 | Verification of edits | No full `#verify`/`#lint`/`#test` run; no `#check-md-go` on the ADR addendum | Comment/doc-only changes; full verify is expensive and was not requested | M |

## c) NOT STARTED

| # | Work | Why not started | Priority |
|---|------|-----------------|----------|
| 1 | Migrate the 7 parent-shim consumers (17 files) to `github.com/larsartmann/go-idempotency` | Needs the owner's call on timing (now vs bundled with engine-backed dedup) | High |
| 2 | `cmd/cqrs-upgrade` codemod rule for the parent import path | Not requested; blocked on #1 decision | High |
| 3 | Gate/lint rule that fails on imports of deleted module paths | Not started; would have caught this class earlier | High |
| 4 | Migrate the 3 sqlstore consumers to `system.Dedup()` / `metaengine.DedupStore` | ADR-0167 facade design (decision-sheet D27) is still owner-pending | High |
| 5 | Add periodic `Sweep` in browser-history and KeyCountdown | Read-only review only; foreign repos | High |
| 6 | Replace Zlota44's 172-line hand-rolled SQLite `idempotency.Store` | Foreign repo; needs owner go-ahead | High |
| 7 | Remove SwettySwipperWeb's dangling `replace` line | Foreign repo; latent, not breaking today | Medium |
| 8 | Resolve `DiscordSync/` vs `discordsync/` duplicate checkout | Foreign repo hygiene; could be intentional | Medium |
| 9 | HARVEST section (f) into `TODO_LIST.md` / `ROADMAP.md` | User said "wait for instructions" after the report | High |
| 10 | v5 deletion of `idempotency/sqlstore` + retirement of the frozen parent docs surface | Waits for v5.0.0 and D27 | Medium |

## d) TOTALLY FUCKED UP

| # | What is broken | Severity | Root cause | Mitigation |
|---|----------------|----------|------------|------------|
| 1 | **7 fleet modules import a module whose source no longer exists** (`idempotency/v4`, frozen `v4.4.0`, deleted 2026-08-15). It builds only while the proxy keeps serving the old tag. No fix, patch, or release is possible. | High (maintenance/security) | ADR-0128 deleted the re-export shim without migrating consumers; ADR-0167 then asserted the parent "survives" | Import swap to `go-idempotency` (all already resolve v0.3.2); 17 files total |
| 2 | **ADR-0167's fleet census was wrong for 2 days** (5 sqlstore consumers listed; real direct count 3 — one duplicate-checkout module counted twice, one indirect module counted as direct). Decision documents were steered by a number no gate validates. | Medium | Census artifact not re-verified; "browser-history ×2" counted an `// indirect` require; `discordsync` is the same module path as `DiscordSync` | Addendum added (`b42d3de72`); needs a gate, not another eyeball |
| 3 | **Unbounded dedup tables**: `browser-history/api` and `KeyCountdown` never call `Sweep`; sqlstore only reclaims lazily when a key is revisited. browser-history's comment even claims the SQLite store has "native TTL, no goroutine" — false. | Medium (disk growth, slowing scans) | Sweep is caller-owned and undocumented at the wiring site | Add a periodic sweeper (DiscordSync's `idempotency_sweeper.go:113` is the working pattern) or move to engine `DedupSweep` |
| 4 | **SwettySwipperWeb has a dangling `replace`** to `../../../../go-cqrs-lite/idempotency/sqlstore` — a directory with no `go.mod`. Harmless only because the require is gone; re-adding it breaks the build with a confusing error. | Medium (latent) | Migration left residue | Delete `go.mod:20` |
| 5 | **Zlota44 re-implements the dedup store in satellite SQL** (172 lines, own schema), contradicting contract 30 ("all durable I/O rides metaengine") and duplicating the upstream contract suite. | Medium | Migrated off sqlstore before `system.Dedup()` existed | Port to `metaengine.DedupStore` (or `sqlstore.NewFromEngine` if the `idempotency.Store` interface must stay) |
| 6 | **Two checkouts of the same module** (`DiscordSync/`, `discordsync/`, identical module path) inflate fleet counts and make reviews ambiguous. | Low-Medium | Unknown (two clones, remotes differ only in case/spelling) | Keep one canonical checkout or have the census dedupe by module path |
| 7 | **My first answer under-scoped the problem**: I reported kvstore correctly but did not notice the parent module was also source-less, even though `ls idempotency/` already showed only `sqlstore/`. The user had to ask the follow-up for that to surface. | Medium (process) | I verified the directory listing but not that 7 apps *depend on* the missing directory | See (e) #1/#2 and (f) #41–44 |

## e) WHAT WE SHOULD IMPROVE

1. **"Tag exists" ≠ "module exists."** Three states must be distinguished in every module review: source at HEAD, deprecated-but-present, deleted-but-proxy-served. A one-line directory check + `git tag -l` would have surfaced the parent shim in the first turn. (Impact: this exact miss cost a follow-up round-trip.)
2. **Use the canonical tool first.** The fleet census exists (`project-dependency-graph who-uses`, `scripts/census-fleet.sh`); I grepped manually in turn 1 and only used the graph tool when told. Manual greps also miss aliased imports and double-count nothing — the tool gets both right. (Impact: 2× work, one wrong consumer count in a decision doc.)
3. **Decision-doc numbers need a gate, not a ritual.** ADR-0167 carried a consumer count that no check verifies. A tiny gate that runs `who-uses` for every module an ADR cites would end this class. (Impact: decisions built on stale fleet truth.)
4. **A deprecation must state the caller-owned lifecycle.** `sqlstore.Sweep` is documented as optional, and two consumers took that literally. Deprecated APIs should carry "you must schedule Sweep or grow unboundedly" at the constructor's godoc, not only on the method.
5. **Deleted-module tombstones belong in detail tables, not just prose.** `modules.md` handles kvstore well (explicit `_(deleted)_` row); the parent `idempotency` row still describes a live module plus a `KVStore` symbol that moved with kvstore. Tombstone rows are cheap and greppable.
6. **Don't let the auto-commit daemon blend foreign edits into a session commit.** `e8dc8b8a8` contains `cmd/cqrs-lint/v4/pkg/toolspec/toolspec_test.go`, which was already dirty at session start and is **not mine**. The commit message cannot distinguish them. (Impact: attribution and review clarity.)
7. **Cross-repo findings need an explicit disposition.** The Swetty replace and Zlota44 store are real defects; leaving them as prose in a status report risks them dying there. Every cross-repo finding should become a TODO_LIST row with the owning repo named.

## f) Up to 50 things we should get done next

Ranked by impact; categories: Bug, Migration, Quality, Cleanup, Documentation, Tooling. Effort S <30m, M 30m–2h, L >2h.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Swap `go-cqrs-lite/idempotency/v4` → `github.com/larsartmann/go-idempotency` in InboxClean (1 file) | Critical | S | Migration |
| 2 | Same swap in github-local-sync (1 file) | Critical | S | Migration |
| 3 | Same swap in games/SEC (1 prod + 1 test file) | Critical | S | Migration |
| 4 | Same swap in KeyCountdown (2 files) | Critical | S | Migration |
| 5 | Same swap in browser-history (3 files) | Critical | S | Migration |
| 6 | Same swap in SwettySwipperWeb (3 files) | Critical | S | Migration |
| 7 | Same swap in Zlota44 (5 files) | Critical | S | Migration |
| 8 | Add a `cmd/cqrs-upgrade` rule rewriting the frozen parent import path | High | M | Tooling |
| 9 | Add a gate that fails on imports of module paths absent from HEAD (deleted-module registry) | High | M | Tooling |
| 10 | Add a periodic `Sweep` to browser-history's sqlstore wiring (or move to engine `DedupSweep`) | High | S | Bug |
| 11 | Add a periodic `Sweep` to KeyCountdown's sqlstore wiring | High | S | Bug |
| 12 | Migrate DiscordSync off sqlstore to `metaengine.DedupStore` + `DedupSweep` | High | L | Migration |
| 13 | Migrate KeyCountdown off sqlstore to `system.Dedup()` once D27 lands | High | M | Migration |
| 14 | Migrate browser-history off sqlstore and collapse the memory/sqlite branch | High | M | Migration |
| 15 | Port Zlota44's hand-rolled SQLite store to `metaengine.DedupStore` | High | M | Migration |
| 16 | HARVEST this report's section (f) into `TODO_LIST.md` / `ROADMAP.md` | High | S | Documentation |
| 17 | Run `nix run .#verify` on the current tree to prove the comment/doc edits are green | High | M | Quality |
| 18 | Add a gate that re-runs `who-uses` for every consumer count cited in an ADR | High | L | Tooling |
| 19 | Delete SwettySwipperWeb's dangling `replace` for `idempotency/sqlstore` | Medium | S | Cleanup |
| 20 | Replace Swetty's local dedup adapter with `sqlstore.NewFromEngine` or `system.Dedup()` | Medium | S | Cleanup |
| 21 | Fix browser-history's false "native TTL, no goroutine" comment at `api/server.go:65-69` | Medium | S | Documentation |
| 22 | Decide and execute one canonical `DiscordSync` checkout (remove or archive the twin) | Medium | S | Cleanup |
| 23 | Teach the census/depgraph to dedupe consumers by module path and to separate direct vs indirect in the report | Medium | M | Tooling |
| 24 | Update `references/core.md:204` to route dedup to `system.Dedup()` / `metaengine.DedupStore` | Medium | S | Documentation |
| 25 | Fix `references/modules.md:63` (drop `KVStore` from the parent row; mark the shim as tag-only) | Medium | S | Documentation |
| 26 | Add a "deleted / tag-only" tombstone convention to all module detail tables | Medium | S | Documentation |
| 27 | Run `nix run .#check-md-go` over the ADR addendum and edited docs | Medium | S | Quality |
| 28 | Document restart-loses-keys explicitly in each memory-store consumer's AGENTS.md | Medium | S | Documentation |
| 29 | Audit SEC's 1-minute sweep interval vs its shared `idempotencyTTL` and document intent | Low | S | Quality |
| 30 | Add a sqlstore godoc note at the constructors that callers MUST schedule `Sweep` | Medium | S | Documentation |
| 31 | Add a test or doc pin for "no Sweep ⇒ expired rows persist" so the semantics cannot be re-misread | Medium | S | Quality |
| 32 | Add a `TODO_LIST` row (with owning repo) for every cross-repo finding in this report | High | S | Documentation |
| 33 | Add a ROADMAP entry for fleet-wide engine-backed dedup (post-D27) | Medium | S | Documentation |
| 34 | Refresh the 2026-11-13 deep-dive's "ten fleet requires on removed idempotency/v4" row with the verified 7 modules / 17 files | Low | S | Documentation |
| 35 | Consider command-level dedup for DiscordSync (today it only dedups events at capture) — needs domain input | Medium | M | Quality |
| 36 | Write the v5 deletion entry for the frozen parent docs surface (`versions.json` stays tag truth) | Medium | S | Documentation |
| 37 | Add `idempotency/sqlstore` to the v5 deletion checklist with its 3-consumer precondition | Medium | S | Cleanup |
| 38 | Add a `check-canonical-facts` row for "deleted-but-required modules" count | Low | M | Tooling |
| 39 | Add a lint rule flagging `// Deprecated:` markers that point at a module with no source | Low | M | Tooling |
| 40 | Add a fleet-consumer snapshot fixture pinning ADR-0167's numbers | Low | M | Tooling |
| 41 | Add a session checklist item: "does the module's source exist at HEAD, or only a tag?" | High | S | Quality |
| 42 | Add the tag-vs-source distinction to `docs/agents/gotchas-*.md` (module management) | High | S | Documentation |
| 43 | Make `who-uses` the documented first step in the fleet-review procedure (AGENTS.md or skill) | Medium | S | Documentation |
| 44 | Write a two-paragraph postmortem of this session's first-turn miss into the project status/feedback trail | Low | S | Documentation |
| 45 | Flag foreign dirty files before the daemon commits, or ask the user to stash them | Low | M | Quality |
| 46 | Add a doc-check assertion: module paths named in SKILL.md must exist at HEAD or carry a deleted marker | High | L | Tooling |
| 47 | Add a migration cookbook entry "frozen shim → external module" with the 17-file checklist | Medium | S | Documentation |
| 48 | Verify (build) each foreign repo after its migration lands; add a per-repo smoke command to the checklist | Medium | S | Quality |
| 49 | Sweep `docs/agents/module-map.md` and `docs/SPAN_NAMING.md` for any remaining live-module wording | Low | S | Documentation |
| 50 | Re-run the full fleet census (`scripts/census-fleet.sh`) after the migrations and diff the counts against ADR-0167's addendum | Medium | S | Tooling |

> **HARVEST note:** items 1–50 are the input for `docs-health` HARVEST. Nothing above was written into `TODO_LIST.md` yet — the user asked to wait for instructions. Items 1–7, 10–11, 16–17, 32 are the ones that should land in `TODO_LIST.md` immediately rather than `ROADMAP.md`.

## g) 3 questions I cannot answer myself

1. **Timing of the parent-shim migration:** do the 17-file import swap to `go-idempotency` now as a mechanical sweep, or bundle it with the ADR-0167 engine-backed dedup migration so each app gets touched once? (Bundling is fewer total edits but keeps 7 apps on a source-less module for as long as D27 stays open.)
2. **Durability intent:** is restart-forgetting acceptable for the 5 memory-store dedup paths (InboxClean, SEC, github-local-sync, plus browser-history/Swetty fallbacks), or should dedup become durable-by-default? I can implement either, but only you can say whether re-execution after a restart is a real risk in those products.
3. **Cross-repo scope:** do you want me to repair the foreign-repo defects found here (Swetty dangling `replace`, Zlota44 satellite SQL store, `DiscordSync`/`discordsync` duplicate checkout, browser-history's false TTL comment), or keep this repo's sessions strictly inside `go-cqrs-lite` and leave those as TODO_LIST rows?

---

## Appendix — consumer usage matrix (verified read)

| Consumer | Parent `idempotency/v4` | `sqlstore/v4` | Store choice | Wiring | Notable |
|---|---|---|---|---|---|
| InboxClean | yes (1 file) | — | memory, 5m sweep / 24h entries | `EventIdempotency` | event-level dedup only |
| browser-history/api | yes (3 files) | yes (v4.4.3) | sqlite when `UseSQLiteReadModel`, else memory | `CommandIdempotency` | no `Sweep`; comment claims native TTL |
| KeyCountdown | yes (2 files) | yes (v4.4.3) | sqlite, DI via `samber/do` | `CommandIdempotency` outside retry | no `Sweep` |
| DiscordSync | no | yes (v4.4.3) | sqlite | capture-level `SetIdempotencyStore` | only consumer with a sweeper |
| games/SEC | yes (2 files) | — | memory, 1m sweep | `CommandIdempotency` | keys lost on restart |
| github-local-sync | yes (1 file) | — | memory, 30m | `CommandIdempotency` | keys lost on restart |
| Zlota44 | yes (5 files) | no | hand-rolled 172-line SQLite store | `CommandIdempotency` + key extractor | satellite SQL, contract-30 violation |
| SwettySwipperWeb | yes (3 files) | replace residue only | `metaengine.DedupStore` adapter | `CommandIdempotency` | already migrated; dangling replace |

**API surface actually used:** `Store`, `MemoryStore` (concrete, for `Close`), `NewMemoryStore`, `ErrDuplicate`, `ErrInvalidTTL`, `sqlstore.NewSQLiteStore`, `sqlstore.Store` (DiscordSync sweeper type). **Zero production users** of `Record`, `Seen`, `CheckAndRecord` directly, `NewPostgresStore`, `NewMySQLStore`, or `NewFromEngine`.
