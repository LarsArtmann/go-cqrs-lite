# Status: Catalog docserver templ-components overhaul — MID-IMPLEMENTATION

**Date:** 2026-08-16 19:33 (Sunday)
**Session scope:** Resumed the catalog UI/UX + EventCatalog overhaul. Prior session fixed the three EventCatalog exporter defects (all green, 14/14 packages) and produced a researched-but-unimplemented templ-components adoption plan gated on 3 open questions. This session the user ordered: execute everything, one verified step at a time, until done.
**Snapshot:** Docserver UI rewrite is code-complete, compiles, and passes existing tests. The packaging around it (flake gates, depguard, new tests, docs, pinning) is mid-flight.

---

## a) FULLY DONE (this session)

1. **Resume verification** — Full catalog module test run before touching code: 14/14 packages `ok` (baseline green confirmed, not assumed).
2. **Research loaded** — templ-components skill (consumer guide + authoring playbook), library v1.8.3 sources read directly from the Go module cache: `layout.Base`/`PageProps` (CSSPath/Favicon/HTMXSrc semantics, ThemeScript in head, children into `<main>`), `SimpleNav`/`NavLink`/`Card`/`StatCard`/`PageHeader`/`Grid`/`Badge`/`CopyButton`/`Button` prop structs (exact field names), Tailwind v4 adoption guide (`@import "tailwindcss" source(none)`, `@source` scanning, `@custom-variant dark`, theme/custom CSS roles).
3. **Toolchain verified for reproducibility** — templ CLI v0.3.1020 at `~/go/bin/templ` (exactly matches the library's pinned generator); nixpkgs `tailwindcss_4` = 4.3.3 (exactly matches the library's own CSS build header); `templ-components@v1.8.3` present in `~/go/pkg/mod`; `DEP_BUDGET[catalog]=4`; depguard allow-list structure mapped.
4. **The 3 open questions DECIDED autonomously** (user ordered full autonomous execution):
   - **Q1 — EventCatalog layout change on v4: ship as-is.** Rationale: the old output was _invalid_ for current EventCatalog (dead `data/` collection, duplicate per-service message pages, missing required `cId`). That is a correctness fix, not an API redesign; gating invalid output behind an Option doubles the test matrix for output nobody should want. CHANGELOG migration note still pending (see c/25).
   - **Q2 — templ-components as production dep: ADOPTED.** Budget now 3/4 (templ-components + a-h/templ direct; go-faster/yaml + go-error-family). Library commits `*_templ.go` so _consumers_ of catalog need no templ CLI — only catalog maintainers regenerate.
   - **Q3 — pin `@eventcatalog/core`: YES, `^4.6.3`.** Latest verified against the npm registry this session (4.6.3). "latest" makes generated projects non-reproducible and can break without any consumer change. Implementation pending (see c/16).
5. **Dependencies added** — `go get github.com/larsartmann/templ-components@v1.8.3` + `github.com/a-h/templ@v0.3.1020`; `go mod tidy` green; module builds with `-tags "goexperiment.jsonv2"`.
6. **Old hand-rolled HTML shells deleted** (`html.go` with `fmt.Sprintf` two-line shells + relative `static/...` URLs) — replaced by typed, accessible templ pages:
   - `layout.templ` — `docsPageProps` (embedded stylesheet + favicon anchored at DocsPath; htmx auto-injection suppressed — nothing here needs it), `docsNav` (SimpleNav + ThemeToggle, dark mode wired), `spaHeader` (slim back-to-index bar for full-height SPAs), `sectionHeading`.
   - `index.templ` — `IndexPage`: PageHeader + version badge, 4 StatCards (services/commands/events/queries), artifact cards (OpenAPI/AsyncAPI/D2/Catalog JSON with raw JSON/YAML/D2 links), per-service cards with count badges.
   - `specview.templ` — `ScalarPage` + `AsyncAPIPage`: vendored SPAs wrapped in the slim chrome, **absolute** DocsPath-anchored asset URLs (the old relative ones broke under nested mounts), noscript fallbacks linking the raw specs.
   - `d2view.templ` — `D2Page`: PageHeader, card with Download `.d2` button, CopyButton, `<pre><code>` source block.
   - `pages.go` — view models (`indexPageData`/`specLink`/`serviceCard`/`catalogStats`) with **deduplicated** message counting (shared messages counted once, keyed by `catalog.Key`).
   - `render.go` — `renderComponent(templ.Component)` HTTP glue with error handling.
7. **New routes + accessors** — `Index()`, `D2View()`, `D2Diagram()` exported handlers; `RegisterRoutes` now serves `GET {prefix}` (index), `{prefix}/d2` (HTML view), `{prefix}/d2.txt` (raw D2 text); `serveOpenAPIHTML`/`serveAsyncAPIHTML` render the templ pages. All additive to the public API.
8. **templ codegen done** — 4 `*_templ.go` files generated with the exact pinned generator; build green; **existing docserver tests still pass** (Scalar/AsyncAPI JS-init assertions preserved by design).
9. **CSS pipeline built AND verified end-to-end**:
   - `static/docs-ui.src.css` — committed source entry (scans docserver `.templ`/`.go`, dark variant, brand token override).
   - `scripts/build-docserver-css.sh` — build + `--check` drift modes; resolves `templ-components@<version>` from the module cache, appends `@source` lines for display/layout/navigation/utils, imports `templ-components-theme.css` + `custom.css`.
   - `static/docs-ui.css` — built artifact (59.5 KB minified, tailwind 4.3.3). Content verified: page classes (`max-w-7xl`, `100dvh`, `bg-gray-950`), library custom utilities (`tc-snap`, `tc-auto-grow`, `popover`, `dialog`, `data-tc-tooltip`), semantic theme (`--color-tc-primary`). `--check` re-run confirms deterministic.
   - `static/favicon.svg` added (embedded via existing `static/*`).
10. **Three templ footguns root-caused with minimal repros, fixed, and documented for posterity** (see e/2): multi-line component-call parser bug, `@if`/`@for` vs plain `if`/`for`, script-body non-interpolation.

## b) PARTIALLY DONE

1. **flake.nix wiring — NOT APPLIED.** The multiedit was rejected (edit tool requires a View-tool read first; I had only inspected flake.nix via bash). The five prepared edits are unapplied:
   - file-size gate: exclude `*_templ.go` (generated files routinely exceed the 350-line limit)
   - new apps `#build-docserver-css` and `#check-docserver-css` (drift gate)
   - `verify` + `verify-fast`: add "Check Docserver CSS" step
   - devShell: add `pkgs.tailwindcss_4`
2. **Dependency added to go.mod but NOT to `.golangci.yml` depguard allow list** → `#lint` will fail on `github.com/larsartmann/templ-components` + `github.com/a-h/templ`. One-line fix, known location (Main allow list).
3. **Docserver UI code done + compiles + old tests pass — but zero NEW tests** for index/D2/d2.txt routes and page content.
4. **Environment workarounds identified but not persisted**: every Go command needs `GOTOOLCHAIN=auto GOWORK=off GOCACHE=$HOME/.cache/go-build GOMODCACHE=$HOME/go/pkg/mod` (toolchain pin 1.26.5 vs go.work ≥1.26.6; dead `/mnt/buildcache` mount breaks both `GOCACHE` and `GOMODCACHE` defaults). All gopls/golangci LSP diagnostics in tool output are this environment, not code.

## c) NOT STARTED

1. New tests: index page (content, stats, links), D2 view page, d2.txt handler, route registration smoke, asset-URL anchoring assertions.
2. Pin `@eventcatalog/core` to `^4.6.3` in `eventcatalog/writer.go` + golden regen + test expectations.
3. Remove the `GET {prefix}/` catch-all route (see d/2).
4. Docs: `catalog/README.md` (layout table: top-level messages, `containers/`, `cId`; fix stale `Exporter{OutputDir}` quickstart → `NewExporter(dir)`), `FEATURES.md` rows, skill references (`modules.md` docserver routes, `recipes.md` docs-UI recipe), `catalog/AGENTS.md` (templ-components adoption table + templ gotchas), root `CHANGELOG.md`.
5. `cmd/api-stability` golden regen (docserver gained exported methods).
6. Full gates after the above: `nix run .#lint` (needs depguard fix), `#check-arch` (budget), `#check-duplication` (new code), `#check-docserver-css` (needs flake app), doc-check, then `#verify-fast` / `#verify`.
7. Module-wide test run AFTER the UI change (only `./docserver/...` ran post-change).
8. Reconcile/annotate the prior session's HTML report (`docs/status/2026-08-16_18-57_*.html`) with this session's progress.
9. TODO_LIST.md harvest from both status reports (docs-health).

## d) TOTALLY FUCKED UP!

1. **Dropped the flake edits mid-flight.** Attempted multiedit on flake.nix without a View-tool read → tool rejected → got pulled into this status report without re-applying. The drift gate I "finished" is not actually wired into verify. Mechanical fix, but it means "CSS pipeline done" is really "pipeline built, gate unwired".
2. **`RegisterRoutes` registers `GET {prefix}/` as a subtree catch-all** — any unknown `/docs/*` path renders the index instead of 404. Masks broken links and typos. Plan: register exact `GET {prefix}` only.
3. **Burned multiple generate-fix cycles on templ syntax the library's own examples already answered.** The library sources use plain `if`/`for` and single-line component calls everywhere; I wrote `@if`/`@for` and multi-line calls first. Lesson recorded: read working examples before writing new syntax.
4. **Silent script interpolation failure.** `{ specURL }` inside `<script>` does not interpolate in templ (script bodies are raw text) — it renders _literally_ with no error or warning. Caught only by reading the generated `_templ.go`. Fixed via `data-spec-url` attribute + `dataset.specUrl`. This is a dangerous failure mode worth a permanent note in catalog/AGENTS.md.
5. Minor: first `go get` was undone by `go mod tidy` (no importers yet) — expected behavior, briefly confusing.

## e) WHAT WE SHOULD IMPROVE!

1. **Apply edit-tool prerequisites immediately**: when an edit is rejected, re-read + re-apply in the same breath; do not carry prepared-but-unapplied diffs across task boundaries.
2. **Add a templ pattern cheat-sheet to `catalog/AGENTS.md`**: single-line component calls; complex props via `{{ }}` Go blocks; plain `if`/`for` in element bodies; script text is raw (use data attributes); regenerate with `GOWORK=off GOMODCACHE=$HOME/go/pkg/mod ~/go/bin/templ generate ./docserver/...`.
3. **Add a templ-sync guard** (mirror the library's `check-templ-sync.sh`): regenerate + diff `*_templ.go` in verify, so generated files can never silently drift from `.templ` sources.
4. **treefmt risk**: `nix fmt` runs gofumpt/golines over all `*.go` including `*_templ.go` — may reformat generated files and churn. Needs either a treefmt exclude or acceptance check.
5. **Persist the cache-mount workaround** (restore `/mnt/buildcache` or export overrides in devShell) instead of per-command env prefixes.
6. **Embed hygiene**: `//go:embed static/*` also embeds `docs-ui.src.css` (harmless, ~1 KB) — could scope to exact files later.
7. **CI**: wire `#check-docserver-css` into `ci.yml` explicitly, not only via `#verify`.
8. **CSP caveat to document**: CopyButton/ThemeToggle scripts are nonce-guarded, but the SPA init `<script>` tags are plain inline — consumers with strict CSP need to allow them (or we add nonce plumbing later).

## f) Next tasks (ordered by impact — top 45)

**Unblock gates (do first):**

1. View flake.nix ranges; re-apply the 5 pending edits (file-size `*_templ.go` exclusion, `#build-docserver-css`, `#check-docserver-css`, verify ×2 wiring, devShell tailwindcss_4).
2. Add `github.com/larsartmann/templ-components` + `github.com/a-h/templ` to `.golangci.yml` depguard allow.
3. Run `nix flake check` / `nix eval` sanity for the new attrs.
4. Run `nix run .#lint` (catalog) → green.
5. Run `nix run .#check-arch` → catalog budget 3/4 confirmed.

**Correct the route mistake:**
6. Drop the `GET {prefix}/` catch-all from RegisterRoutes.
7. Add 404 assertion test for unknown `/docs/*` path.

**New tests:**
8. `TestDocsServer_Index`: 200, text/html, title, version badge, stat values (1/1/1/1 for fixture), links to all four artifacts + raw links.
9. `TestDocsServer_IndexDeduplicatesMessages`: message shared by two services counted once.
10. `TestDocsServer_D2View`: 200, contains D2 source, Download link, `data-tc-copy` attributes.
11. `TestDocsServer_D2Text`: 200, text/plain, equals `D2Handler` output for same catalog.
12. `TestDocsServer_RegisterRoutes`: all routes resolve on a stdlib mux.
13. `TestDocsServer_ScalarPage` asset anchoring: absolute `/docs/static/scalar.js` (not relative).
14. `TestDocsServer_AsyncAPIPage`: `asyncapi-react.css` stylesheet link + absolute JS URL.
15. noscript fallback text present in both SPA pages (assert raw-spec link).

**Pinning:**
16. `eventcatalog/writer.go`: `"@eventcatalog/core": "^4.6.3"` (named constant) + golden regen + test updates.
17. `cId` override Option on the exporter (from prior backlog).

**Codegen/format hygiene:**
18. Check `*_templ.go` line counts vs 350 gate (confirm the flake exclusion is actually needed).
19. `templ generate` idempotency check (no diff on rerun).
20. templ-sync guard script + verify wiring (or fold into #1).
21. `nix fmt` run + git diff on `*_templ.go` — add treefmt exclude if churn.
22. gofumpt on new hand-written files (pages.go, render.go, docserver.go, d2.go).

**Docs:**
23. `catalog/README.md`: output layout table (top-level `commands|events|queries/`, `containers/`, `cId`, `llmsTxt`).
24. README quickstart fix: `NewExporter(dir)` (not `Exporter{OutputDir}`).
25. README/CHANGELOG: docserver section — new routes (`/docs`, `/docs/d2`, `/docs/d2.txt`), templ-based UI, embedded stylesheet, CHANGELOG migration note for the exporter path change (behavior change).
26. `FEATURES.md`: docserver UI + EventCatalog exporter correctness rows.
27. `catalog/AGENTS.md`: templ-components adoption table + templ gotchas + regenerate command + CSS rebuild command.
28. Skill `references/modules.md`: docserver entry (routes, pages, CSS pipeline).
29. Skill `references/recipes.md`: "Documentation UI" recipe (mount DocsServer, index, SPAs, D2, EventCatalog export at startup).
30. Annotate prior HTML status report with pointer to this file.
31. TODO_LIST.md harvest from both reports (docs-health HARVEST).

**Verification close-out:**
32. `cmd/api-stability` golden regen (new exported handlers).
33. Full catalog module test run post-change (`GOWORK=off go test ./... -tags ...`).
34. `nix run .#check-duplication` (new pages.go/render.go).
35. doc-check on updated skill markdown.
36. `nix run .#verify-fast`, then full `#verify` (exclusive, nothing heavy running).
37. Confirm example apps still build (docserver API additive; `OpenAPIUI()` signature unchanged).

**Polish / follow-on:**
38. StatCard icons (`icons.*`) for visual polish on the index.
39. Consider `/docs/llms.txt` route (EventCatalog writes llmsTxt; docserver could serve one) — ROADMAP.
40. CSP nonce plumbing for SPA scripts — ROADMAP.
41. D2 playground deep-link button (`https://play.d2lang.com`) — ROADMAP (needs URL-guess policy check).
42. Verify dark mode end-to-end (ThemeToggle + `.dark` classes present in built CSS).
43. Extend `example/rest` (or taskmanager) to mount docserver routes as living demo.
44. Env-gated real-CLI build test for generated EventCatalog output (prior backlog).
45. Restore `/mnt/buildcache` or persist GOMODCACHE/GOCACHE overrides in devShell.

## g) Questions I can NOT figure out myself

1. **v4 shipping of the EventCatalog layout change**: I proceeded with "ship as a correctness fix on v4" (invalid output → top-level messages, `containers/`, `cId`). If you want a compat Option emitting the legacy layout until v5, say so before the CHANGELOG/release step — otherwise the migration note is the only consumer-facing artifact.
2. **Catch-all route removal**: OK to make unknown `/docs/*` paths 404 (removing the `GET {prefix}/` subtree registration)? Anyone depending on `/docs/` (trailing slash) resolving to the index would shift to `/docs` — I can keep an exact `/docs/` → index redirect instead if you prefer.
3. **templ codegen enforcement strategy**: generated `*_templ.go` are committed (correct for a library). Should `#verify` also run `templ generate` + diff (requires adding a pinned `templ` package to the flake toolchain — stronger guarantee, small toolchain cost), or keep codegen a documented local-only step with the CSS-style drift gate limited to CSS for now?

---

_Point-in-time snapshot. The auto-commit daemon may fold parts of this work into ambient commits; treat `git log`/working tree, not this file, as source of truth for landed state._
