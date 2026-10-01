# Status Report — BDD suite for the embedded EventCatalog server + config-war #8 discovered

- **Timestamp:** 2026-10-01 06:41 (Thursday)
- **Session scope:** implement "PROPER bdd-testing (skill)" for `catalog/eventcatalog/`
  (the embedded `StaticServer`), applying the global `bdd-testing` skill; verify with
  the repo's gates.
- **Author context:** this report covers ONLY this session's run and what was noticed
  during it. No unrelated research.
- **Verdict:** 🟡 BDD task DONE and green — but the session's own `nix run .#lint`
  verification came back **RED repo-wide**, and root-causing that uncovered a
  **live `.golangci.yml` corruption (config-war recurrence #8)** that is NOT mine
  and is currently un-repaired. Details in section (d).

---

## Session at a glance

| Area | Result |
| ---- | ------ |
| BDD suite for `StaticServer` | ✅ DONE (18 specs, green) |
| Plain tests replaced by BDD suite | ✅ DONE (2 files removed) |
| `catalog/AGENTS.md` maintainer note | ✅ DONE (uncommitted in worktree) |
| `go test ./eventcatalog/...` | ✅ green (18/18) |
| `go test ./...` (catalog module) | ✅ green |
| `golangci-lint` on new files | ✅ clean |
| file-size ratchet | ✅ clean |
| api-stability golden | ✅ unchanged (7530 exports) |
| `nix run .#lint` (repo-wide) | ❌ RED — foreign cause (config-war #8) |
| `nix run .#verify` / `#verify-fast` | ⬜ NOT RUN |
| `check-duplication`, `-race`, `check-md-go` | ⬜ NOT RUN |

---

## (a) FULLY DONE

### 1. Loaded and applied the `bdd-testing` skill

Read `/home/lars/.config/crush/skills/bdd-testing/SKILL.md` before writing anything
(the skill trigger matched the request verbatim). Verified the repo already uses
Ginkgo + Gomega BDD across ~11 modules (`event`, `command`, `query`, `decider`,
`signing`, `encryption`, `middleware`, `listing`, `catalog`, `integration/*`,
`metaengine`) — so this was applying an existing convention, not inventing one.

Conventions adopted from the skill + repo:
- **Black-box** `package eventcatalog_test`.
- **One bootstrap per package** with `RegisterFailHandler(Fail)` + `RunSpecs`.
- **Fresh subject per spec** — server rebuilt in `BeforeEach`, never shared.
- **`DescribeTable`/`Entry`** for the routing matrix (each `Entry` is its own spec).
- **No committed focus** (`FIt`/`FDescribe`) — grepped, clean.
- Benchmarks left as plain `testing.B` (`benchmark_test.go` untouched), per the
  skill's "benchmarks are not specs".

### 2. `catalog/eventcatalog/eventcatalog_bdd_suite_test.go` (13 lines) — NEW

```go
// skip-validate
package eventcatalog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventCatalogBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventCatalog BDD Suite")
}
```

Matches the repo's `<pkg>_bdd_suite_test.go` naming exactly (`signing_bdd_suite_test.go`,
`listing_bdd_suite_test.go`, `catalog_bdd_suite_test.go`).

### 3. `catalog/eventcatalog/server_bdd_test.go` (192 lines) — NEW, 18 specs

Behavior inventory written from the **consumer's** vantage point ("a developer
embedding a built EventCatalog site"), grouped into scenarios:

- **Constructing the server** — rejects nil `fs.FS`.
- **Routing GET requests** (`DescribeTable`, 9 entries) — root index, explicit
  index, directory index `/docs/foo/`, extensionless `/legacy`, explicit
  `/legacy.html`, 301 canonicalization `/docs/foo` → `/docs/foo/`, custom 404,
  directory-without-index 404, `..` traversal rejected.
- **Caching & content types** — HTML `no-cache` + `text/html`; `_astro/*`
  `immutable` + `javascript`; JSON content-type; HEAD returns 200 with empty body.
- **HTTP method guard** — POST → 405 + `Allow: GET, HEAD`.
- **Configuring the server** — `WithNotFoundFile` fallback to plain-text 404;
  `WithImmutableAssetPrefixes` override.
- **Embedded deployment shape** — real `//go:embed` → `fs.Sub` → serve.

The `//go:embed testdata/site` declaration moved into this file; the fixture
(`testdata/site/`) is unchanged and still git-tracked.

### 4. Removed the superseded plain tests

- `catalog/eventcatalog/server_test.go` (deleted)
- `catalog/eventcatalog/server_embed_test.go` (deleted)

Both were authored in the prior session and fully subsumed by the BDD suite
(their behaviors are now specs). Rationale: avoid two parallel test styles for one
subject and dodge the `art-dupl` clone gate. **This was an autonomous decision and
is flagged as a question in (g).** Confirmed committed by the daemon
(`00beeed31`, `git ls-files` shows only the two BDD files).

### 5. Verification run (green legs)

| Gate | Command | Result |
| ---- | ------- | ------ |
| Package tests | `cd catalog && GOWORK=off go test ./eventcatalog/... -count=1` | ✅ ok |
| Spec count | `... -v` | ✅ **Ran 18 of 18 Specs, SUCCESS** |
| Module tests | `cd catalog && GOWORK=off go test ./... -count=1` | ✅ all packages ok |
| Lint (new files) | `golangci-lint run ./eventcatalog/...` | ✅ new files not flagged |
| Format | `gofumpt -l`, `goimports -local … -l` | ✅ no output |
| File size | `bash scripts/check-file-size.sh` | ✅ no new offenders |
| API golden | `cmd/api-stability && go run .` | ✅ 7530 exports verified |
| Dangling refs | grep for removed helpers | ✅ none |
| Focus commits | grep `FIt`/`FDescribe` | ✅ none |

### 6. Documented the suite for maintainers

`catalog/AGENTS.md:119` (the "Embedded serving" bullet) now states the behavior is
pinned by the BDD suite and points maintainers at the two files (so nobody adds a
third plain table test). **Currently uncommitted** (`git status` shows ` M`).

---

## (b) PARTIALLY DONE

1. **Scope coverage.** I built BDD specs for the **embedded `StaticServer` only**.
   The request ("PROPER bdd-testing … for this embedded catalog/eventcatalog/")
   is ambiguous: it could mean the *whole* `eventcatalog` package (exporter,
   writer, frontmatter, index-manifest — ~20 existing plain test files). I chose
   the embedded-server reading because that is the feature just shipped and the
   word "embedded" points at `StaticServer`. **Owner confirmation needed (g#2).**
2. **Behavior edge-coverage.** The suite covers the happy path + the obvious
   guards, but several real `server.go` behaviors are **untested** (see (e)/#1–9):
   query-string preservation on redirect, HEAD on the 404 path, URL-encoded
   traversal, nested immutable prefixes, no-Content-Type fallback, double-slash.
   `server.go` is at/near the 350-line limit but its logic has more branches than
   the suite exercises.
3. **Realistic fixture.** The BDD suite uses `fstest.MapFS` + a 6-file toy fixture.
   The prior session verified `server.go` against a **real** render-gate build
   (`/tmp/ec-verify/dist`, 88-page Astro output) via a throwaway `main.go` — that
   real-tree check is **not** part of the committed suite, so it does not regress-protect.
4. **Composed gate.** I ran each leg individually but did **not** run
   `nix run .#verify` / `#verify-fast`. `#check-duplication`, `-race`, and
   `#check-md-go` were not run for this change.
5. **Report index row.** The `docs/status/README.md` index row for this report was
   added as part of this deliverable (see below) — but the previous two reports'
   rows were written by the daemon-absorved commits, so cross-session row-ownership
   is fuzzy.

---

## (c) NOT STARTED

1. `nix run .#verify` (or `#verify-fast`) composed gate.
2. `nix run .#check-duplication` after adding specs.
3. `go test -race ./eventcatalog/...`.
4. Coverage measurement (`go test -cover`) for `server.go`.
5. `nix run .#check-md-go` (blocked — see (d)).
6. Concurrent-use / `Handler()`-reuse specs.
7. A realistic Astro-shaped fixture committed alongside the toy one.
8. BDD specs for the rest of the `eventcatalog` package (if that was the intent).

---

## (d) TOTALLY FUCKED UP

### d1. 🔴 **`.golangci.yml` is CORRUPTED — config-war recurrence #8 (live, un-repaired)**

This is the headline finding of the session and **it is not mine.**

While running the session's lint verification, `nix run .#lint` returned
**RED across the entire repo**:

```
10 issues:
* gci: 10
❌ Lint: findings in: command commandlifecycle … catalog middleware … systemtest
```

Root-causing that (because lint findings in *untouched* packages are a smell)
uncovered:

**Evidence chain**
- `.golangci.yml:831-832` — `formatters.enable` contains **`gci`**.
- AGENTS.md internal-contract **#18** says: *"`gci` was REMOVED from `.golangci.yml`
  formatters (2026-08-16) because two tools fighting over the same import blocks
  re-broke 95+ files once."*
- Commit **`903232e3f`** (`chore: auto-commit …`, **2026-10-01 06:16:45**, the
  auto-commit daemon) touched 29 files, and its diff of `.golangci.yml`:
  - **deleted the entire `depguard:` settings block** (93 lines removed), and
  - **added `+    - gci`** to `formatters.enable`,
  - and **rewrote `scripts/depguard-block.golden.yml`** (184 lines changed).
- `bash scripts/check-golangci-hash.sh` →
  ```
  ✗ .golangci.yml content hash MISMATCH (corruption tripwire)
      expected sha256:c556e6fc29350ff2…
      actual   sha256:ddc994ceecbc9fce…
  ```
- The same signature is documented verbatim in the **2026-09-29 06:41** report
  ("daemon damage to .golangci.yml (gci re-added, … depguard deleted)") and in
  **today's 2026-10-01 04:30** report ("config-war #7 … repaired, hash re-pinned").
  So this is **recurrence #8**, ~100 minutes after #7 was reportedly fixed.

**Why it matters**
- Two formatters (`gci` + `goimports`/`gofumpt`) fight over import blocks →
  repo-wide phantom lint failures (exactly the failure mode AGENTS #18 warns about).
- `depguard` being deleted silently disables the dependency-budget allow-list.
- `check-lint-config` / the hash tripwire is designed to catch this — and it did.
- **Every module's lint is red until repaired**, which makes per-module lint
  verification meaningless right now.

**Repair is documented** (`scripts/restore-depguard.sh`, `scripts/check-golangci-hash.sh
--update`, `scripts/depguard-block.golden.yml`) but is a **config + golden commit**
and reverts a change I did not author — per AGENTS ("NEVER revert changes you didn't
author … ASK before touching it") and the user's "then WAIT", **I did not repair it.**

### d2. ⚠️ Plain tests deleted without explicit approval

`server_test.go` and `server_embed_test.go` were removed (see (a#4)). Correct by
the skill and dedup hygiene, but the user may have wanted both. Owner confirmation
requested (g#3). Recoverable from git (`00beeed31`).

### d3. ⚠️ `check-md-go` is red on master (pre-existing, was known)

Carried over from the earlier sessions and **not** touched by this work: the pinned
`md-go-validator` flake FOD has a `vendorHash` mismatch (build break), and the host
binary reports an unbaselined archived fence
(`docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md:46`). Cannot
be fixed without the pinned binary building or a baseline regen. Not attributable
to this session.

### d4. ⚠️ `nix run .#lint` reports gci findings — even on files I never touched

Same root cause as d1; flagged separately because it means **"lint green" cannot be
claimed for this session** — only "lint clean on the new files".

---

## (e) WHAT WE SHOULD IMPROVE

1. **Close the redirect-query gap.** `server.go:119` appends `r.URL.RawQuery` to the
   redirect target; **no spec asserts it.** Add `Entry("… ?x=1 → /docs/foo/?x=1")`.
2. **HEAD-on-404 spec.** `serveNotFound` (`server.go:246`) returns headers with an
   empty body for HEAD; untested.
3. **Encoded-traversal spec.** `r.URL.Path` decodes `%2e%2e` to `..`, so
   `hasDotDotSegment` (`server.go:147`) should catch `/docs/%2e%2e/x`; untested.
4. **Nested immutable prefix.** Default `_astro/` with `_astro/sub/app.js`
   (`strings.HasPrefix` at `server.go:225`) — untested.
5. **Unknown-extension fallback.** `contentTypeFor` (`server.go:253`) returns `""`
   when `mime.TypeByExtension` is empty → no `Content-Type` header; untested.
6. **Double-slash / empty-segment** paths (`//`, `/docs//`) — untested.
7. **No-directory-listing assertion.** The doc claims listings are never generated;
   assert it structurally (a dir with no index → 404, never a listing body).
8. **Options matrix.** Use `DescribeTable` for `WithImmutableAssetPrefixes` with
   zero args, multiple args, and non-matching prefixes.
9. **Realistic fixture.** Commit an Astro-shaped tree (`docs/{type}/{id}/{version}/`,
   `api/*.json`, a hash-named bundled JS) so router changes regress-protect against
   real EventCatalog output, not a 6-file toy.
10. **Run the composed gate.** `nix run .#verify-fast` (or `#verify`) is the only
    thing that proves this change is release-safe; individual legs are not enough.
11. **Run `-race` + `check-duplication`** — cheap and the repo gates on both.
12. **Record coverage.** `go test -cover` before/after gives an honest number for the
    "did BDD improve safety" claim.
13. **Stop claiming lint-green when config is corrupt.** The session should have
    surfaced the red lint immediately as a blocker, not as a footnote.
14. **Treat "lint findings in untouched packages" as a tripwire**, not noise — it
    was the signal that led to d1.
15. **Daemon hardening.** The auto-commit daemon keeps re-planting `gci` and deleting
    `depguard`; consider excluding `.golangci.yml`/its golden from the daemon, or
    adding `check-golangci-hash` as a commit-time hook. (Recurrence #8 — pattern, not
    accident.)

---

## (f) Up to 50 things to get done next

**Config-war (highest priority)**
1. Inspect `git show 903232e3f -- .golangci.yml` fully and classify intentionally vs corruption.
2. Repair `.golangci.yml`: restore `depguard`, remove `gci` (`scripts/restore-depguard.sh`).
3. Restore `scripts/depguard-block.golden.yml` (corrupted in the same commit).
4. Re-pin `scripts/golangci-config-hash.golden.txt` (`check-golangci-hash.sh --update`).
5. Run `nix run .#check-lint-config` and confirm green.
6. Run full `nix run .#lint` and confirm repo-wide green (no phantom `gci`).
7. Audit other files in `903232e3f` for hidden semantic damage: `FEATURES.md`,
   `catalog/docserver/*_templ.go`, `metaengine/tursoengine/cte_probe_test.go`,
   `example/mesh-demo/*`, `cmd/cqrs-lint/*`, `otel/otlp/*`, `flake.lock`.
8. Determine why the daemon's heuristic wrote `.golangci.yml` at all.
9. Add a daemon exclusion (or a commit-time hash gate) so recurrence #9 cannot land.
10. Document recurrence #8 in AGENTS/status so the pattern is greppable.

**BDD suite hardening**
11. Add redirect-query-preservation spec.
12. Add HEAD-on-404 spec.
13. Add URL-encoded traversal spec.
14. Add double-slash spec.
15. Add unknown-extension (no Content-Type) spec.
16. Add nested `_astro/sub/` immutable spec.
17. Add `WithImmutableAssetPrefixes()` zero-arg spec.
18. Add `WithNotFoundFile` pointing at an *existing* file spec (custom 404 body).
19. Add explicit no-directory-listing assertion.
20. Convert the options cases to `DescribeTable`.
21. Commit a realistic Astro-shaped fixture + a spec over it.
22. Add a concurrency spec (parallel GETs, `-race`).
23. Add a `Handler()`-reuse spec.
24. Run the suite with `ginkgo --repeat` to surface flakes.
25. Measure and record `server.go` coverage.
26. Add a `helpers_test.go` split if the spec file grows past ~250 lines.
27. Decide BDD scope for the rest of the package (exporter/writer/frontmatter).
28. If in scope, port the highest-value exporter behaviors to Ginkgo.

**Verification / gates**
29. Run `nix run .#verify-fast`.
30. Run `nix run .#verify` before any release.
31. Run `nix run .#check-duplication` (specs may create new clone shapes).
32. Run `go test -race ./eventcatalog/...`.
33. Run `nix run .#check-arch` (no dep change expected — confirm).
34. Re-run `cmd/api-stability` (confirm still 7530).
35. Re-run `cmd/doc-check` on `catalog/AGENTS.md` (prose-only; expect "no Go refs").
36. Decide whether the deleted plain tests stay deleted.

**Docs / housekeeping**
37. Commit the `catalog/AGENTS.md` BDD note (daemon may already have absorbed it).
38. Add the `docs/status/README.md` index row for this report (done in this commit).
39. Consider a `catalog/README.md` "Testing" line pointing at the BDD suite.
40. Add a package `doc.go` sentence referencing the BDD suite.
41. Add CHANGELOG entry only if a *consumer-visible* behavior changed (it did not).
42. Reconcile the pre-existing md-go baseline (blocked on pinned validator build).
43. Report the pinned `md-go-validator` `vendorHash` break as a separate issue.
44. Verify `.art-dupl-baseline.json` untouched by the test consolidation.
45. Confirm `testdata/site/` fixture files remain git-tracked (they do).

**Structural / longer-term**
46. Evaluate extracting `StaticServer` routing into a table-driven `route()` fn for
    coverage of every branch.
47. Add a spec for the documented root-mount-only limitation (sub-path hosting).
48. Decide whether `StaticServer` should support `StripPrefix` sub-path hosting.
49. Add a fuzz test for `requestKey`/`hasDotDotSegment` (traversal robustness).
50. Wire a `nix run .#test`-level BDD smoke so a broken spec fails CI loudly.

---

## (g) Questions I CANNOT answer myself (max 3)

1. **Config-war #8 — intentional or repair it?**
   Commit `903232e3f` (06:16:45 today) added `gci` back and deleted `depguard` from
   `.golangci.yml`; the hash tripwire now mismatches. Is that edit *intended*, or is
   it daemon corruption I should repair (restore depguard, drop gci, re-pin hash)?
   I deliberately did not touch a file I did not author.

2. **BDD scope — `StaticServer` only, or the whole `eventcatalog` package?**
   I built specs for the embedded server. Did you want BDD for the exporter /
   writer / frontmatter too (≈20 existing plain test files), or is the embedded
   server the intended subject?

3. **Keep or drop the plain tests?**
   I deleted `server_test.go` and `server_embed_test.go` (subsumed by the BDD
   suite, to avoid duplication). Restore them alongside the suite, or is the
   replacement right?

---

*Report generated 2026-10-01 06:41. Awaiting instructions.*
