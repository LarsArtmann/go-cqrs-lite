# Status Report — EventCatalog Overhaul Closeout: Verification Re-Runs + Upstream Issue Draft

**Date:** 2026-09-23 05:14 (Wednesday)
**Session scope:** Continuation of the EventCatalog integration overhaul (prior report: `2026-09-23_04-36_eventcatalog-integration-overhaul.md`). This segment: final verification re-runs, two upstream bugs root-caused to source, doc corrections, upstream issue drafted + verified. Report covers THIS session's run and what was noticed in passing.

---

## a) FULLY DONE ✅

1. **Lint re-confirmation after the nolint removal** (`catalog/eventcatalog/writer.go:114`): `nix run .#lint` → `catalog` module **0 issues**. The 8 modules WITH findings (scheduling/sqlstore, middleware, storage, watermill, cmd/cqrs-lint, metaengine, metaengine/duckdbengine, systemtest) are parallel-session residue — untouched by this work, not fixed silently.
2. **End-to-end render gate re-run** (`bash scripts/check-eventcatalog.sh`): **OK** — both profiles (default 86 pages, changelog 92 pages), 4/4 semantic spot-checks, no `InvalidContentEntryDataError`. This closed the "re-run after final refactors" gap (fixture constants + `new()` + `changelogBody` refactor all landed after the previous green run).
3. **Changelog-profile WARN root-caused to upstream (new finding this session):** the build's link-validation logs 3 broken sidebar links (`/docs/{channels,data-products,entities}/<id>/<version>/changelog`). Source-verified in @eventcatalog/core 4.6.3: the changelog page route (`_index.data.ts` `itemTypes`) covers only agents/events/commands/queries/services/domains/systems/flows/containers — but `sidebar-store/state.ts:794` (channels), `builders/entity.ts`, `builders/data-product.ts:106-109` emit Changelog links for the 3 missing collections anyway. **Unfixable from exporter output** (even a `changelog.mdx` cannot make the page generate); cosmetic WARN; gate correctly does not fail on it.
4. **UPDATE_SNAPS guidance corrected** — the prior guidance in `catalog/AGENTS.md` ("Always regen module-wide: `UPDATE_SNAPS=true go test ./...`") was **exactly the destructive command**: go-snaps v0.5.23 `Clean` (`snaps/clean.go:198-227` `examineFiles`) deletes every `.snap` the current *binary* didn't touch, and `catalog/testdata/golden/` is shared by 4 packages running as 4 separate binaries — so each binary cleans the others, even module-wide. New guidance: scoped regen + `git ls-files -d -- testdata/golden | xargs -r git restore --`. Same trap documented repo-wide in `docs/agents/gotchas-testing.md` (one added sentence, links to catalog/AGENTS.md).
5. **Known-upstream-WARN note added** to `catalog/AGENTS.md` (the 3 broken links, why they're unfixable, and that the gate must not fail on them).
6. **Upstream agent-changelog crash: diagnosis completed to source level, all 5 verify-before-filing gates PASSED, issue drafted in Lars's voice** (`docs/status/2026-09-23_eventcatalog-agent-changelog-crash-issue-draft.md`, checker: 0 FAIL / 0 WARN):
   - Root cause chain, source-verified in both 4.6.3 and **4.11.2 (latest)**: route `itemTypes` includes `agents`+`systems` → changelog page `getBadge()` (`index.astro:77-101`) has **no `agents`/`systems` branch** → returns `undefined` → `badges = [undefined]` → `Badge.astro:25` → `getBadgeHref(undefined)` → `badge-styles.ts:224` `if (!badge.url)` → **TypeError**.
   - **Live reproduction on 4.11.2**: minimal project + 1 agent + `changelog:{enabled:true}` → `TypeError: Cannot read properties of undefined (reading 'url') at getBadgeHref`, exit 1. **Control**: same fixture, `enabled:false` → build Complete. No `changelog.mdx` needed.
   - **Not a regression**: identical failure on 4.6.3 and 4.11.2.
   - **No duplicate**: searched open+closed issues ("agent changelog", "changelog crash", "TypeError", "badge") — nothing matching.
   - Draft includes the related sidebar broken-links bug as a split-off offer.
7. **Docs gates for my edits:** `nix fmt` stable on my files; md-go validator green on the first run this session (see b) for the later failure — environment, not content).
8. **Working tree reconciled:** all my edits absorbed by the auto-commit daemon (verified via `git log`); nothing of mine dangling. No authored commit made (no user commit instruction — harness rule).

## b) PARTIALLY DONE ⚠️

1. **`#check-md-go` gate verification** — green at ~04:5x ("no new errors", 1461 blocks, 104 baselined) directly after my doc edits; **later re-runs (~05:1x) fail to build the validator** with nix errors (`Cannot build md-go-validator-4dd9437.drv`; go-modules FOD zip missing from store). Evidence points to a **parallel session actively editing the `md-go-validator` sibling checkout** (a new store build `md-go-validator-5b72f89` and a `prepared-source-...-dirty` derivation appeared between my two runs; store GC churn). My additions contain no `go` fences (markdown + one bash fence), so they cannot introduce md-go findings. **Needs one re-run once the sibling is committed/stable.** Two retries burned before recognizing the concurrency signal — see (d)/(e).
2. **Upstream issue** — drafted, verified, voice-checked; **NOT filed** (deliberately: prior session's open question #1 offered "draft for review OR user files it"; filing is the user's call).
3. **Render gate known-warn** — permanently present until upstream fixes the sidebar/route inconsistency; documented + tolerated by design. Not closable from here.

## c) NOT STARTED 📋 (deliberate — awaiting user decisions)

1. **docserver audit** — validating docserver's in-process EventCatalog view against the same zod contract (prior open question #3).
2. **x- mapping policy decision** — keep mapping non-representable fields (`labels`, `responses`, team `role`/`avatarUrl`) into `x-` custom properties vs dropping them (prior open question #2).
3. **Filing the upstream issue** (draft ready).
4. **Upstream pin bump** `^4.6.3` → fixed-later release + removing `shouldEnableChangelog` guard and the changelog gate profile (blocked on upstream fix).
5. **Full `#verify` in an exclusive window** (prior next-steps item 4; still pending, expected to trip on the 3 pre-existing unrelated file-size violations).

## d) TOTALLY FUCKED UP 💥

Nothing new of mine this segment, but two honest entries:

1. **Prior-session landmine found and defused:** catalog/AGENTS.md's UPDATE_SNAPS guidance recommended the *exact command* that deletes foreign snapshots. Anyone following it would corrupt three other packages' goldens. That was written by me (prior segment) — verified-then-corrected this session, but it shipped wrong once.
2. **md-go gate re-run discipline:** I retried a failing nix build twice after the FIRST failure already carried the concurrency tell (validator store hash flipped `4dd9437` → `5b72f89`, `-dirty` prepared-source appearing). A third participant is mid-edit in the sibling repo; the correct move after failure #1 was "note environment contention, defer re-run", not retry. Wasted ~2 minutes of wall time; no damage done.

## e) WHAT WE SHOULD IMPROVE 🛠️

1. **Parallel-session collision awareness:** this repo runs multiple concurrent agents (validator sibling being edited, 8 modules with lint findings, 53 files committed unformatted, TODO_LIST/CHANGELOG edits appearing mid-session). Gates that rebuild shared derivations are *contended resources* — on unexplained build failure, check for a hash flip / `-dirty` source BEFORE retrying.
2. **The auto-commit daemon commits unformatted code:** my mid-session `nix fmt` reformatted 53 files a parallel session had committed unformatted (they would fail CI's `nix fmt --fail-on-change`). The daemon absorbing code before treefmt sees it is a systemic gap — a pre-absorb fmt step (or fmt hook in the daemon) would close it.
3. **UPDATE_SNAPS needs a safe wrapper:** the correct procedure is now 2 commands with a non-obvious restore step. A `scripts/regen-snaps.sh <pkg>` encoding regen+restore would make the trap impossible to hit. (Also: `go test -p 1` for the catalog module in CI would remove the cross-binary hazard entirely.)
4. **Probe debris discipline (recurring lesson):** /tmp/ec-probe-411, /tmp/ec-core-latest, /tmp/master-*.txt left behind (~60MB, /tmp-volatile). Prior session recorded the same class of lesson; still not habitual.
5. **Gate warn-budgeting:** check-eventcatalog.sh greps for hard failures and tolerates ALL warns. Asserting the known-warn signature (exactly 3 broken changelog links) would make NEW warns fail loudly instead of relying on a human reading build.log.
6. **Report forensic precision:** the early-green md-go run wasn't timestamped in my notes — pinning gate-run timestamps makes later "was it my change or the environment?" questions one lookup instead of an archaeology dig.

## f) THINGS TO GET DONE NEXT (brainstorm, impact-sorted; not a commitment list)

1. **Decide + file the upstream issue** (draft ready at `docs/status/2026-09-23_eventcatalog-agent-changelog-crash-issue-draft.md`; 2-minute action).
2. **Re-run `#check-md-go`** once the md-go-validator sibling has a committed/clean tree (the only verification debt from this segment).
3. **Answer the x- mapping policy** (keep vs drop non-representable fields).
4. **docserver audit** against the verified EventCatalog zod contract.
5. **Harvest this section into TODO_LIST.md** (docs-health HARVEST) — not done now because the user said report-then-wait.
6. Triage the 8 parallel-session lint-failing modules (scheduling/sqlstore, middleware, storage, watermill, cqrs-lint, metaengine, duckdbengine, systemtest).
7. Add `scripts/regen-snaps.sh <pkg>` safe wrapper (regen + `git ls-files -d` restore).
8. Consider `go test -p 1` (or single-binary golden harness) for catalog in CI to kill the cross-binary snap hazard structurally.
9. Warn-budget assertion in check-eventcatalog.sh (fail on any warn ≠ the 3 known broken links).
10. Split-file the sidebar broken-links bug upstream (offered as related note in the draft; separate issue if maintainers prefer).
11. Track upstream fix; when landed: bump `^4.6.3`, drop `shouldEnableChangelog` guard, drop the changelog gate profile, re-run both.
12. ec-fixture: add a second changelog entry covering a DIFFERENT resource kind (service) to pin multi-resource changelog rendering.
13. Full `#verify` in an exclusive window; expect the 3 pre-existing file-size violations (pebbleengine layout_planner.go, metaengine fold.go, cqrs-lint lintutil.go) — decide baseline-vs-shrink (owner call, NOT mine).
14. CHANGELOG.md: the parallel session left an over-long line (metaengine separator hunk) — nix fmt/markdownlint pass.
15. Clean probe debris: /tmp/ec-probe-411, /tmp/ec-core-latest, /tmp/master-*.
16. Ask whether catalog/README "Format fidelity" section should also mention the known-warn (currently only catalog/AGENTS.md has it).
17. Consider asserting `catalog` lint cleanliness in a fast standalone gate (currently only visible inside full #lint).
18. Systemtest lint findings (gocyclo 22, nestif, prealloc) — parallel session's, but small and mechanical if they don't get to it.
19. Prior session follow-up that remains valid: the render fixture could pin the exact page-count per profile (86/92) to catch silent page loss.
20. Consider recording the verify-before-filing outcome (4.6.3+4.11.2 repro, no dupes) next to the draft so a future re-verify doesn't redo the npm-pack dance.
21. If upstream fixes get slow: document the crash workaround (no agents + changelog, or changelog off) in catalog/README for consumers hitting it via OUR exporter.
22. Re-check `benchkit/go.mod` dirtiness origin (was dirty at prior session start; confirm daemon absorbed or still floating).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (max 3)

1. **File the upstream issue?** The draft is verified (reproduced on latest 4.11.2, no duplicates found) and voice-checked. Should I file it to `event-catalog/eventcatalog` now via `gh`, or do you want to review/edit the draft first? (Splitting the sidebar-links bug into a second issue — yes/no — rides along.)
2. **x- mapping policy:** keep exporting non-representable fields as `x-` custom properties (`labels`→`x-labels`, `responses`→`x-responses`, team `role`/`avatarUrl`) to preserve data, or drop them for a leaner, schema-pure export? Both are defensible; it's a product-taste call.
3. **docserver audit now or later?** Should I validate docserver's in-process EventCatalog view against the same source-verified zod contract in a follow-up session, or is that out of scope for this overhaul?

---

*Report format: Markdown (.md) per explicit user request — overrides the status-report skill's HTML default (flagged per skill policy). Auto-commit daemon will absorb this file; no manual commit per harness rule.*
