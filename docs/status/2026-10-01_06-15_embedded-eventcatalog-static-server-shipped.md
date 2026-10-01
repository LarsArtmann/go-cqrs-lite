# Status — Embedded EventCatalog static server (`catalog/eventcatalog.StaticServer`)

**Timestamp:** 2026-10-01 06:15 CEST
**Session scope:** implement the requested "option to serve an embedded version of
EventCatalog" — a Go-binary-served EventCatalog with no Node at deploy time.
**Production code touched:** `catalog/eventcatalog/server.go` (new), tests,
docs, CHANGELOG, api golden. No existing behavior changed.

---

## 0. One-paragraph truth

The feature shipped end-to-end: `eventcatalog.NewStaticServer(fs.FS)` +
`Handler()` wrap a built EventCatalog `dist/` (or any `//go:embed` tree) and do
Astro-correct static routing, verified against a **real** EventCatalog build
produced by the render gate (200s, 301 canonicalization, immutable `_astro/`
caching, 404 fallback). Tests, lint, vet, gofmt/goimports, api golden,
file-size, check-arch, doc-check, and the recipe compile/coverage gates are
green. The main misses were process ones: the first test fixture was placed in a
gitignored `dist/` directory (caught late), the pinned `md-go-validator` could
not be built (`vendorHash` mismatch) so md-go is host-binary-advisory only, and
the final README note was re-checked only after the report request.

---

## 1. FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | `StaticServer` + `Handler()` implemented | `catalog/eventcatalog/server.go` (263 lines, ≤350) |
| 2 | `NewStaticServer(fsys, opts...)` with nil-FS → Rejection error | `server.go:71`, `TestNewStaticServerRejectsNilFS` |
| 3 | Directory index routing (`/docs/foo/`) | `TestStaticServerRouting/directory_index` |
| 4 | Canonical trailing-slash 301 (`/docs/foo` → `/docs/foo/`) | `resolveFile`/`redirectTarget`, routing test |
| 5 | Extensionless `.html` resolution (`/legacy` → `legacy.html`) | routing test |
| 6 | Custom `404.html` served at 404; plain-text fallback | `serveNotFound`, `TestStaticServerOptions/custom_not-found_file` |
| 7 | No directory listings | `/docs` (dir without index) → 404 test |
| 8 | Traversal rejection (`/docs/../404.html`) | `hasDotDotSegment`, routing test |
| 9 | GET/HEAD only; 405 + `Allow` otherwise | `TestStaticServerRejectsNonGET` |
| 10 | HEAD has no body | `TestStaticServerHeaders/head_has_no_body` |
| 11 | `_astro/` → `Cache-Control: immutable`; HTML → `no-cache` | headers tests |
| 12 | Content-Type by extension (HTML/JS/JSON) | headers tests |
| 13 | Options: `WithNotFoundFile`, `WithImmutableAssetPrefixes` | `options` tests |
| 14 | `//go:embed` end-to-end test | `server_embed_test.go` + `testdata/site/` |
| 15 | **Real-build verification** against the render gate's `dist/` | see §8 receipts |
| 16 | `doc.go` cross-reference to `NewStaticServer` | `catalog/eventcatalog/doc.go` |
| 17 | `catalog/README.md` "Serving an embedded catalog" section | `catalog/README.md` |
| 18 | `catalog/AGENTS.md` maintainer note (why `eventcatalog`, not `docserver`) | `catalog/AGENTS.md` |
| 19 | Skill recipe §2.9 (compile-verified fence) + catalog entry | `recipes.md`, `recipes_catalog.go` |
| 20 | CHANGELOG `[Unreleased] ### Added` entry | `CHANGELOG.md` |
| 21 | API golden regenerated (7530 exports) | `docs/api_surface.txt` |
| 22 | `golangci-lint` clean (fixed errcheck ×2, gosec G710, modernize/slicescontains) | 0 issues |
| 23 | `go vet`, `gofmt`, `goimports -local`, `gofumpt` clean | no output |
| 24 | File-size ratchet green; check-arch green (no new deps) | §8 |
| 25 | Recipe coverage + compile gates green | §8 |
| 26 | doc-check green on README/AGENTS/recipes | 801 refs valid |

---

## 2. PARTIALLY DONE

| # | Item | What's missing |
|---|------|----------------|
| 1 | Gate coverage | Did **not** run the composed `nix run .#verify` (exclusive, ~full workspace); ran targeted gates only. |
| 2 | md-go gate | Could only run the **host** `md-go-validator` (rev `c541a5e`), not the flake pin (`7b677c5`) — the pin's FOD fails on `vendorHash` mismatch. Results are advisory. |
| 3 | Real-dist coverage | Verified against **one** real build (default fixture profile). The changelog profile and the plain-refs profile were not served. |
| 4 | Base-path mounting | Documented "mount at root (Astro root-absolute URLs)"; no `StripPrefix`/base-path support or test. |
| 5 | Client-side routes | Did not probe EventCatalog's deep client routes (visualiser/client nav) for a static-hosting 404 edge case. |
| 6 | HTTP correctness breadth | No conditional-request / range / gzip / `HEAD`-with-`Content-Length` tests. |
| 7 | Example module | No self-contained `example/` app demonstrating embed; only tests + README/recipe. |
| 8 | Integration with `docserver` | No helper to mount the static site alongside `DocsServer` (deliberately decoupled; documented). |
| 9 | Fixture provenance | Fixture is hand-written minimal HTML, not a slice of a real build (kept small on purpose). |

---

## 3. NOT STARTED

1. `FEATURES.md` / `ROADMAP.md` rows for the embedded-server capability.
2. `docs/agents/module-map.md` / skill `modules.md` mention.
3. `nix run .#check-eventcatalog` re-run (exporter untouched, but as confirmation).
4. A `cmd/` demo or `example/` module wiring `//go:embed` → `StaticServer`.
5. Wiring the static server into any preset/`stack`/`system` composition root.
6. A mount helper for a sub-path with HTML URL rewriting (the honest "option" for non-root hosting).
7. `Content-Length`/`Last-Modified`/`ETag` support (files are embedded, immutable-ish).
8. `testdata/site` fixture regeneration policy (it is committed input now).
9. CSP interaction check (EventCatalog ships inline scripts; `docserver`'s CSP nonce machinery is not involved here).
10. Baseline regen for the pre-existing archived md-go error (blocked on pinned binary).
11. Filing/updating the TODO for the md-go pin build break.

---

## 4. TOTALLY FUCKED UP

No breakage; honest sloppiness ledger:

| # | Sloppiness | Impact |
|---|-----------|--------|
| 1 | **First fixture lived at `testdata/dist/`** — the global gitignore ignores `dist/`, so the embed test would have passed locally but been untracked/broken in CI. Caught only at final `git status`, renamed to `testdata/site/`. | High if shipped; caught. |
| 2 | **Ran the render gate via raw `bash scripts/…` (previous session) and this session's md-go via host binary** rather than the Nix app. | Medium — not the canonical invocation. |
| 3 | **Recipe compile first reported "(cached)"** after a doc edit; had to force `-count=1` because Go's test cache does not track the external `recipes.md`. | Low; re-ran correctly. |
| 4 | **Pinned md-go gate cannot build** (`vendorHash` mismatch in md-go-validator's own flake). Job wasted ~a minute; left the gate's true state unknown. | Medium — external blocker, but I did not triage the hash. |
| 5 | **Last README edit (gitignore note) was re-checked only after this report was requested**, not immediately. | Low — doc-check re-run green (801 refs). |
| 6 | **Unrelated md-go red on master** (archived fence unbaselined) + I annotated a live §2.42 snippet; both pre-existing, neither caused by this feature. I did not fully root-cause the archived entry. | Medium — the gate stays red and I cannot pin-verify a fix. |

---

## 5. WHAT WE SHOULD IMPROVE

1. **Never place committed test fixtures under a globally-ignored path.** Add a
   `git check-ignore` assertion to the fixture/test or a note in
   `catalog/AGENTS.md` (`dist/` is ignored by the user's global gitignore).
2. **Always use the Nix app** for gates that have one (`nix run .#check-md-go`),
   and treat the host binary as advisory — the script already warns about rev drift.
3. **Fix the `md-go-validator` flake `vendorHash`** so the pinned gate is runnable
   (this blocks honest md-go verification for everyone).
4. **Re-run doc-check immediately after any doc edit**, not at the end.
5. **Verify against more than one real build profile** when a feature consumes
   generated artifacts.

---

## 6. UP TO 50 NEXT THINGS

### This feature's tail

1. Serve the render gate's **changelog** profile dist through `StaticServer` (page-count parity).
2. Serve the **plain-refs** profile dist (prove non-default trees route too).
3. Add an `example/embedded-eventcatalog/` module (`//go:embed` + `ListenAndServe`).
4. Add a `cmd/catalog-export`-style `-serve` flag to demo embedded serving.
5. Add a `docserver` convenience: mount `StaticServer` next to `DocsServer` (opt-in).
6. Add `StripPrefix` base-path helper + HTML URL rewriting (honest sub-path option).
7. Add conditional requests (`ETag`/`If-None-Match`) for embedded files.
8. Add `Last-Modified`/`Content-Length` on `HEAD`.
9. Add gzip/br pre-compression support for text assets.
10. Add a test asserting no directory listing for every intermediate dir.
11. Add a test for `?query` preservation on the tracking-slash redirect.
12. Add a test rejecting `//` and encoded traversal (`%2e%2e`).
13. Test Windows-style separators are not a bypass.
14. Test empty FS (missing index) → 404, not panic.
15. Test `WithImmutableAssetPrefixes()` (empty) disables immutable caching.
16. Document the exact EventCatalog dist layout the handler assumes.
17. Add a screenshot/manual smoke recipe to README.
18. Add `nix run .#check-eventcatalog-embed` (serve a built dist + curl routes).
19. Consider `http.FileServerFS` parity note (why custom handler).
20. Add a benchmark for handler routing (alloc counts).

### Fixture & tests

21. Move `testdata/site` under a documented fixture-regeneration comment.
22. Add a `git check-ignore` guard test for the fixture path.
23. Add golden snapshot of served headers per route.
24. Add race test (concurrent requests) — trivial but pins sharing safety.
25. Add a test for a `404.html` present vs absent.

### Gates / process

26. Fix `md-go-validator` flake `vendorHash` (blocker for pinned gate).
27. Re-baseline the archived md-go error under the pinned binary once buildable.
28. Decide whether the §2.42 `// skip-validate` stays (pin vs host divergence).
29. Run `nix run .#verify` once the tree is quiet.
30. Add FEATURES.md row for embedded serving.
31. Add ROADMAP note (embedded docs as a deployment story).
32. Update skill `modules.md` catalog row.
33. Update `docs/agents/module-map.md`.
34. Add CHANGELOG cross-ref in `catalog/AGENTS.md` render-gate bullet.
35. Consider a `check-embedded-fixture-tracking.sh` leg.

### Product direction

36. Publish the option in the website/docs (when the site exists).
37. Consider embedding a **served** hub merge (federation) in one binary.
38. Consider a `system`/`stack` preset that serves embedded docs automatically.
39. Consider ETag-from-content-hash for `_astro` (filename already hashed).
40. Consider range support for large JSON/API files.

### Housekeeping noticed this session

41. The daemon auto-committed mid-edit; for authored history, commit at phase boundaries.
42. `.gitignore` global `dist/` rule affects fixtures — document it.
43. Re-check that no build binaries are committed under `catalog/cmd/*` (policy from CHANGELOG).
44. Add a short FAQ entry: "EventCatalog renders blank under a base path" (root-absolute URLs).
45. Add a FAQ entry: "`//go:embed` can't find `dist/`" (gitignored build output).
46. Verify `staticFilesystem`-style caching not needed here (stateless handler).
47. Confirm handler is safe when `fsys` is an `http.FS`-wrapped FS.
48. Consider `io/fs` `ReadDirFS` requirement documentation.
49. Add a test for deep nesting (`a/b/c/d/index.html`).
50. Revisit `Modernize` suggestions proactively before lint (slices.Contains class).

---

## 7. THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Do you want sub-path hosting support** (mount the embedded EventCatalog at
   e.g. `/catalog` with URL rewriting), or is "mount at root" acceptable?
   It is the one genuinely ambiguous design fork, with real complexity.
2. **Should the built `dist/` be committed** so `//go:embed` works, or is the
   consumer expected to build + copy it to a non-ignored dir in CI? (Affects the
   documented workflow and whether an example module is viable.)
3. **Do you want me to fix the `md-go-validator` flake `vendorHash` build break**
   (a repo-external tool pin issue) as part of this project's gates, or leave it?

---

## 8. VERIFICATION RECEIPTS

```text
$ cd catalog && GOWORK=off go test ./... -count=1
ok  .../catalog/eventcatalog  0.194s   (all catalog packages green)

$ GOWORK=off golangci-lint run ./eventcatalog/...
0 issues.

$ cd cmd/api-stability && GOWORK=off go run .
API surface OK: 7530 exports verified

$ bash scripts/check-file-size.sh
✓ file-size ratchet: no new offenders, no growth

$ bash scripts/check-arch.sh
━━━ All architecture checks passed ━━━

$ cd cmd/doc-check && GOWORK=off go test -count=1 -run 'TestRecipesCatalogCoversFile|TestRecipesCompile' .
ok  .../cmd/doc-check/v4  (coverage + compile green)
```

Real EventCatalog build served through `StaticServer` (temporary verifier,
since removed via `trash`):

```text
/                                          200 ct=text/html             len=69606
/docs/commands/CreateOrder/1.0.0/          200 ct=text/html             len=73465
/docs/commands/CreateOrder/1.0.0           301 loc=/docs/commands/CreateOrder/1.0.0/
/api/sidebar-data.json                     200 ct=application/json     len=14201
/_astro/abnfDiagram-…js                    200 cache=…immutable
/definitely-missing                        404 ct=text/plain
```

Known-broken (pre-existing, unrelated): `check-md-go` — archived
`docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md` fence is
unbaselined; pinned `md-go-validator` build fails on `vendorHash` mismatch.
