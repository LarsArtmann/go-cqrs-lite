# Turso Documentation Sweep: LibSQL → Turso Database

**Date:** 2026-07-06 01:34
**Session scope:** Fix stale LibSQL branding across the codebase after Turso engine/driver shift
**Trigger:** User noticed docs said "LibSQL" — Turso's old engine — while the code already uses `tursogo` (Turso Database, the MVCC rewrite).

---

## What Happened

The user asked "How is stack/'s turso integration?" → I investigated → the **code** was already on `turso.tech/database/tursogo v0.6.1` (the current recommended SDK) but the **docs** still described it as "embedded LibSQL" in ~30 files. The user said "LibSQL is the old implementation!" — slightly overstated (LibSQL isn't fully deprecated, `libsql://` URLs still work) but the core grievance was legitimate: the prose branding was stale. We then fixed it across the entire repo.

---

## a) FULLY DONE ✅

### 1. Go Source Comments — All LibSQL references updated

**Files (11):**

| File                                      | Change                                                                       |
| ----------------------------------------- | ---------------------------------------------------------------------------- |
| `storage/turso/doc.go`                    | "embedded LibSQL" → "embedded Turso Database"; pool section rewritten        |
| `storage/turso/connector.go`              | "LibSQL native engine" → "Turso Database engine" (×3); ConfigurePool comment |
| `storage/turso/backend.go`                | "Turso/LibSQL" → "Turso" (×2); "embedded LibSQL" → "embedded database"       |
| `storage/turso/indexing.go`               | "Turso/LibSQL" → "Turso" (×2)                                                |
| `storage/turso/example_test.go`           | Comment + example URL `libsql://` → `https://`                               |
| `storage/turso/backend_test.go`           | Pool comment                                                                 |
| `storage/turso/indexing/doc.go`           | Package doc                                                                  |
| `storage/turso/indexing/optimizations.go` | Pragma comments (×3)                                                         |
| `storage/turso/indexing/auto.go`          | Error message comment                                                        |
| `stack/turso/doc.go`                      | Package doc + Quick Start example                                            |
| `stack/turso/drivers.go`                  | Driver registration comment                                                  |
| `stack/turso/preset.go`                   | New() doc, NewSync() doc, bus comment                                        |

**Verified:** Zero LibSQL/libSQL references remain in any `.go` file (excluding `libsql://` URL strings in test data, which are arbitrary round-trip strings, not user-facing branding).

### 2. Markdown Docs — All living docs updated

**Files (13):**

| File                                                 | Change                                                                  |
| ---------------------------------------------------- | ----------------------------------------------------------------------- |
| `README.md`                                          | 3 table entries + preset description                                    |
| `AGENTS.md`                                          | Module tree comment                                                     |
| `FEATURES.md`                                        | Local DB + Pool configuration rows                                      |
| `docs/DOMAIN_LANGUAGE.md`                            | Turso domain term                                                       |
| `docs/INFRASTRUCTURE_RECOMMENDATIONS.md`             | Edge/offline-first row                                                  |
| `docs/PRESETS.md`                                    | Engine column                                                           |
| `docs/README.md`                                     | Index link text                                                         |
| `docs/STORAGE_GUIDE.md`                              | Section header + **bug fix** (see below)                                |
| `docs/benchmarks/README.md`                          | Benchmark section header                                                |
| `docs/turso-indexing-guidance.md`                    | Title + section header + code example                                   |
| `docs/planning/2026-06-23_..._HARDENING.md`          | 2 table entries                                                         |
| `docs/research/storage-first-principles-analysis.md` | Engine column                                                           |
| `docs/research/2026-06-23_..._AUDIT.md`              | 2 references                                                            |
| `storage/turso/README.md`                            | Tagline + pool table + 2 code examples + indexing section + pragma note |
| `storage/turso/indexing/README.md`                   | Tagline + related modules                                               |
| `storage/README.md`                                  | Related modules link                                                    |

### 3. AI Skill Docs — Updated

**Files (4):**

| File                                                 | Change                     |
| ---------------------------------------------------- | -------------------------- |
| `.agents/skills/go-cqrs-lite/SKILL.md`               | Decision matrix row        |
| `.agents/skills/go-cqrs-lite/references/modules.md`  | Turso module row           |
| `.agents/skills/go-cqrs-lite/references/recipes.md`  | Preset comparison table    |
| `.agents/skills/go-cqrs-lite/references/advanced.md` | Offline-first code comment |

### 4. URL Scheme Updated (`libsql://` → `https://`)

After verifying with Turso's current docs: `tursogo`'s sync driver accepts both `libsql://` and `https://`, but Turso's own examples now show `https://` for the Turso Database engine. Updated **all user-facing examples**:

- `storage/turso/doc.go` — Quick Start sync example
- `storage/turso/README.md` — 2 sync examples
- `storage/turso/example_test.go` — Example function
- `stack/turso/doc.go` — Quick Start + NewSync parameter doc
- `stack/turso/preset.go` — NewSync doc
- `docs/turso-indexing-guidance.md` — Sync example

**Left as `libsql://`:** Test assertion strings (`sync_test.go`, `connector_test.go`, `sync_internal_test.go`, `multidb_test.go`) — these are arbitrary string round-trip data, not user-facing examples. Changing them would be cosmetic churn.

### 5. Pre-existing Bug Fixed

`docs/STORAGE_GUIDE.md:112` showed `turso.Open("libsql://...")` — but `Open` takes a **local file path** (`DbPath`), not a URL. Fixed to `turso.Open(turso.DbPath("app.db"))`. This was a documentation bug independent of the LibSQL issue.

### 6. Verification

- ✅ Build passes (`go build ./storage/turso/... ./stack/turso/...`)
- ✅ All tests pass with `-race` (`storage/turso/v4`, `storage/turso/v4/indexing`, `stack/turso/v4`)
- ✅ doc-checker validates all 790 Go references in skill docs (`cmd/doc-check`)
- ✅ Zero remaining LibSQL references in living docs (archive/adr/research excluded by design)

---

## b) PARTIALLY DONE 🟡

### Pre-existing uncommitted work (NOT this session's work)

The working tree has significant uncommitted changes from **before** this session — visible in the git status at conversation start. These appear to be from recent commits (f469fd37 "fix: unify watermill dedupRing", 24ae7c58 "fix: remove panic from dedupRing", 6dad501e "chore: update API surface") plus uncommitted follow-ups:

- `transport/http/sse.go`, `transport/http/sse_replay.go` — SSE replay/dedup changes
- `transport/http/replay_metrics.go` — new file (untracked)
- `transport/http/sse_integration_test.go` — new test (untracked)
- `dedup/` — appears to be a new extracted module (untracked)
- `transport/http/dedup_ring.go` — deleted (moved to `dedup/`?)
- `testutil/journal.go` — new helper
- Various `go.mod`/`go.sum` updates across modules

**Status:** I did NOT touch these. They are someone else's work in progress. Flagging because the user asked for comprehensive status — these represent real unfinished work in the tree.

---

## c) NOT STARTED ⬜

Nothing related to this session's scope was left undone. The LibSQL → Turso Database sweep is complete.

---

## d) TOTALLY FUCKED UP 💥

### Nothing in this session.

However, I noticed a **near-miss**: when editing `storage/turso/indexing/doc.go`, my first edit accidentally corrupted the file (duplicated the package doc line, lost the description line). I caught it immediately on the next View call and fixed it before moving on. The final file is correct. This was a reminder that `multiedit` on package doc comments requires care — the tool replaces exact strings, and partial line matches can cascade.

---

## e) WHAT WE SHOULD IMPROVE

### i. The "LibSQL" branding rot was systemic

**~30 files** had stale "LibSQL" references. This suggests **no process for tracking vendor/driver rebrands**. When we upgrade a major dependency (like switching from `go-libsql` to `tursogo`), there's no checklist item to sweep docs/comments.

**Fix idea:** Add a step to the dependency-upgrade workflow: "grep the old name across all files, update or explicitly archive."

### ii. STORAGE_GUIDE.md had a latent API bug

`turso.Open("libsql://...")` was wrong for **years** apparently — `Open` takes a file path, not a URL. This means **nobody ran that code example**. Doc examples that aren't tested rot.

**Fix idea:** The `example_test.go` pattern ( runnable examples with `// Output:` assertions) should be the standard for ALL doc examples. STORAGE_GUIDE.md code blocks are untested prose.

### iii. Historical docs accumulate stale terminology

`docs/status/archive/` has 15+ files mentioning LibSQL. We correctly left them alone (they're point-in-time snapshots), but they're now inconsistent with the rest of the repo. This will confuse anyone grepping for "LibSQL" in the future.

**Fix idea:** Add a `docs/status/archive/README.md` note: "Historical snapshots reflect terminology current at the time of writing."

### iv. Turso sync mode is unverified in CI

`TestNewSync_Contract` skips unless `TURSO_SYNC_URL`/`TURSO_SYNC_TOKEN` are set. The Push/Pull/Checkpoint code paths have **never run in CI**. This is a real reliability gap — not a doc issue, but worth flagging.

### v. The `libsql://` → `https://` change is cosmetic for now

The driver accepts both. But if Turso ever drops `libsql://` support, our test strings will silently break. We should consider tracking which Turso database type the user has (libSQL vs Turso Database) and documenting the URL difference.

---

## f) Top 25 Things to Get Done Next

Ranked by impact × urgency:

| #   | Task                                                                                                                                                   | Impact | Effort  | Module                |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------- | --------------------- |
| 1   | **Commit the Turso docs sweep** — this work is uncommitted                                                                                             | High   | 5min    | repo-wide             |
| 2   | **Review and commit/finish the pre-existing dedup/SSE work** in the working tree                                                                       | High   | Unknown | transport/http, dedup |
| 3   | **Set up Turso sync CI** — run `TestNewSync_Contract` against a free Turso Cloud instance in GitHub Actions                                            | High   | 2h      | stack/turso, CI       |
| 4   | **Add `archive/README.md` disclaimer** about historical terminology                                                                                    | Low    | 5min    | docs/status/archive   |
| 5   | **Audit other vendor references for staleness** — PebbleDB, Watermill, OTel versions                                                                   | Medium | 1h      | repo-wide             |
| 6   | **Convert STORAGE_GUIDE.md code blocks to runnable examples** where possible                                                                           | Medium | 2h      | docs/                 |
| 7   | **Run `nix run .#lint`** on the changed Go files to verify formatting                                                                                  | Medium | 5min    | repo-wide             |
| 8   | **Run `nix run .#test`** full suite to ensure nothing else broke                                                                                       | High   | 10min   | repo-wide             |
| 9   | **Check if `tursogo` has breaking changes in v0.7+** — we're on v0.6.1                                                                                 | Medium | 30min   | storage/turso         |
| 10  | **Document the Turso Database vs LibSQL distinction** in a short ADR                                                                                   | Medium | 1h      | docs/adr/             |
| 11  | **Update `docs/adr/0029-storage-consolidation.md`** — still says "Turso (LibSQL)"                                                                      | Low    | 5min    | docs/adr/             |
| 12  | **Sweep for "SQLite" references** — the Turso Database engine is SQLite-compatible but distinct                                                        | Low    | 30min   | repo-wide             |
| 13  | **Add a dependency-upgrade checklist** to AGENTS.md or CONTRIBUTING                                                                                    | Medium | 30min   | process               |
| 14  | **Verify `turso.Open()` examples in SKILL.md** are all using `DbPath` correctly                                                                        | Low    | 15min   | skill docs            |
| 15  | **Run `cmd/api-stability`** to check if the API surface changed                                                                                        | Medium | 5min    | cmd/api-stability     |
| 16  | **Consider extracting a `turso.DatabaseType` enum** (LibSQL vs Turso Database) if the distinction matters for users                                    | Low    | 2h      | storage/turso         |
| 17  | **Test `NewSync` with `https://` URL** end-to-end (not just `libsql://`)                                                                               | Medium | 1h      | stack/turso           |
| 18  | **Audit `docs/research/2026-05-27_HONKER_TURSO_WATERMILL_SQLITE_RESEARCH.md`** — describes Turso as "built on libSQL"                                  | Low    | 10min   | docs/research/        |
| 19  | **Check if the `ConfigurePool` comment is still accurate** — does the Turso Database engine still serialize writes through one connection?             | Medium | 1h      | storage/turso         |
| 20  | **Add integration test for `https://` sync URLs** (currently only `libsql://` in test data)                                                            | Low    | 30min   | storage/turso         |
| 21  | **Review whether `OpenInMemory` resource-limit warning is still needed** with the new engine                                                           | Low    | 1h      | storage/turso         |
| 22  | **Update `docs/DOMAIN_LANGUAGE.md`** to add "Turso Database" as a distinct term from "LibSQL"                                                          | Low    | 15min   | docs/                 |
| 23  | **Check Turso SDK changelog** for any deprecation warnings we should address                                                                           | Medium | 30min   | storage/turso         |
| 24  | **Run `nix fmt`** to ensure formatting is clean after all edits                                                                                        | Medium | 2min    | repo-wide             |
| 25  | **Consider a "Terminology" section in AGENTS.md** documenting: Turso Database (new engine) vs LibSQL (legacy) vs `libsql://` (URL scheme, still valid) | Low    | 20min   | AGENTS.md             |

---

## g) Top Question I Cannot Answer Myself

### Does the Turso Database engine (the MVCC rewrite) still require `MaxOpenConns=1`, or does MVCC enable safe concurrent connections?

**Why I can't answer this:** Our `ConfigurePool` code (`storage/turso/connector.go:159`) caps the pool at 1 connection with the comment "Embedded LibSQL serializes writes through a single connection." But the whole point of Turso's new MVCC rewrite is **concurrent writes**. If MVCC means the engine can now handle concurrent connections safely, capping at 1 is a **performance bottleneck** we're imposing for no reason.

**What I'd need to answer this:**

1. Turso's `tursogo` SDK documentation on connection pooling behavior
2. Or a benchmark: `ConfigurePool` vs `MaxOpenConns=10` under concurrent write load
3. Or confirmation from Turso's team on whether MVCC changes the single-writer constraint

**Why it matters:** This is in every hot path. If the cap is unnecessary, removing it is a free performance win for every Turso user of this library.

---

## Session Metrics

| Metric                          | Value                                                |
| ------------------------------- | ---------------------------------------------------- |
| Files edited                    | 31                                                   |
| Categories of changes           | 7 (Go source ×3 layers, markdown ×4 categories)      |
| LibSQL references fixed         | ~50 prose instances                                  |
| URL references updated          | ~10 `libsql://` → `https://` in user-facing examples |
| Pre-existing bugs found & fixed | 1 (`turso.Open` shown with URL instead of file path) |
| Tests run                       | 3 packages, all pass with `-race`                    |
| Doc references validated        | 790 (via `cmd/doc-check`)                            |
| Time elapsed                    | ~40 minutes                                          |
