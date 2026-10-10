# Status Report — Graph-Native Wave Execution (V2 plan, Waves 0–3)

**Date:** 2026-10-10 07:37
**Session:** Execution of [`docs/planning/2026-10-10_06-11_SUPERB-graph-native-adoption-closure-and-operator-story-V2.md`](../planning/2026-10-10_06-11_SUPERB-graph-native-adoption-closure-and-operator-story-V2.md) (the "execute" command: run Waves 0→5)
**Scope of this report:** ONLY this session's run (Waves 0–3) and what was noticed along the way. Waves 4–5 untouched.

---

## a) FULLY DONE

| Item                                                              | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Wave 0 — guards & facts (A0–A13)**                              | TODO_LIST in-flight marker; GraphDriver verdict (`graph.MemoryDriver` is the ONLY impl — Cypher/Gremlin was a portability target stated as shipped); modules.md `graph` row honesty fix + NEW `metaengine/graphadapter` row; bigtableengine dep weight (HEAVY: 3 direct GCP deps, ~50 indirect → allengines-excluded); dgraph DSN = plain `host:port` gRPC addr; facts recorded in plan §7. Authored commit `561c36073` (amended over a daemon race).                                                                                                                                                                                                                              |
| **Wave 1 — `example/graph-native/` (C1–C12), THE keystone**       | Complete follow-network app: Followed/Unfollowed events, decider with all 3 guards (self-follow Rejection, duplicate Rejection, unfollow-missing Conflict), `Edge` + `EdgeRemoval` folds → `Query[ReachabilityQuery, []string]`, sqlite default deployment, flag-gated dgraph projections (`-dgraph`), `LoadConfig` boot (`-config`). Runnable main verifies depth-1/2 traversals + retraction (exits non-zero on mismatch). systemscenario BDD suite (2 tests: traversal+retraction, guards) green WITH `-race`. Registered in go.work, flake `examplePaths`, api-stability exclusion maps (×4), cqrs-lint module catalog, check-module-layers (L7, budget 12 = exact dep count). |
| **Wave 2 — publish truth (A1–A7, B1–B5)**                         | recipes.md **§2.44** (3 fences + TOC drift fix for §2.41–2.43), compile-classified, `TestRecipes` green. advanced.md **§6.13 rewritten modern-first**: Graph ADT leads, legacy `GraphProjection` demoted to migration note, Cypher/Gremlin honesty fix, graphadapter flat-identity limitation + `Driver()` escape hatch. graphadapter `doc.go` limitation paragraph (forward-refs ADR-0156). Gates: doc-check 1229 refs valid, md-go parse gate clean.                                                                                                                                                                                                                             |
| **Wave 3 — D1–D10**                                               | D1 COOKBOOK "Graph Patterns" chapter (follow-network pattern + engine×graph matrix + node-identity note); D2 SKILL.md modern read-model table gains the graph row; D3+D4 system/README "Graph-Native Projections" section (sqlite + dgraph YAML, matrix summary); D5 root README example-tour mention; D7 readmodels.md tier table → modern-first with v5 banner; D8 FAQ relations entry; D9 FAQ `ExecuteCtx` `[]any` → `ExecuteTyped`/`ExecuteTypedByName` entry; D10 recipes §2.44 **4th fence** (systemscenario graph-BDD) + catalog entry — compile-verified green after 2 fixes (see d).                                                                                      |
| **Two product truths discovered & encoded in docs**               | (1) `id.StreamID.String()` is brand-prefixed (`"StreamMarker:alice"`) — node names must come from payload fields (FAQ + recipe + COOKBOOK). (2) The coeffect gate is BLIND to `system.RawQuery` — declaring `Events` with RawQuery projections only emits unconsumed advisories (example comment + recipe note).                                                                                                                                                                                                                                                                                                                                                                   |
| **Engine-support claims verified against source before standing** | sqlite (recursive CTE w/ iterative fallback), pg/duckdb (WITH RECURSIVE), mysql (8.0+ CTE, probed), dgraph (`n(depth:)`), badger (prefix-scan BFS), memory/iroh (BFS/passthrough) — all with undirected + edge removal; pebble + bbolt lack ADTGraph.                                                                                                                                                                                                                                                                                                                                                                                                                              |

## b) PARTIALLY DONE

| Item                                                    | State                                                                                                                                                        | What remains                                                                                                                                                                                       |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Wave 3 closure**                                      | D1–D10 done and individually gated                                                                                                                           | **D6** (`check-readme-links.sh` + deprecated-symbol gate over the touched READMEs) and **D11** (full doc-gate sweep over everything touched this wave) not yet run; wave-boundary commit not made. |
| **Authored wave commits**                               | Wave 0 landed authored (`561c36073`)                                                                                                                         | Waves 1–3 content is committed but absorbed into `chore: auto-commit` blobs (daemon races; see d) — messages lost, content verified present.                                                       |
| **`ExecuteTyped` typed-traversal story (plan task D9)** | Verified `ExecuteTyped[Q,[]string]` reconstructs typed collections (execute.go:623 `isCollectionResult` → `reconstructTyped`); FAQ + recipe + example use it | `ExecuteTypedByName` signature verified (execute.go:632) but only documented, not exercised in the example (single query there).                                                                   |

## c) NOT STARTED

- **Wave 4 (E1–E9):** ADR-0155 operator story (two-level: nix `-tags` + `cqrs.yaml`/`LoadConfig`, allengines membership table, Proposed), all-pure-Go-engines measurements (binary size + dep counts), GraphAddEdge signature survey, ADR-0156 edge labels at v5 (Option A extend vs B limitation), cross-links.
- **Wave 5 (G1–G9):** T18 ADT→recipe coverage ratchet (doc-check meta-test, waive-or-fix spatial/search/streamlog gaps), T19 consumer dry-run (fresh sub-agent answers "graph-native how?" from docs only; ≤3 iterations), HARVEST into TODO_LIST (incl. adopt-if-needed markers per A-R1, remove the A0 in-flight marker), annotate the 05-48 status report, final commit + push.
- **Not started, discovered this session (candidates, not yet queued):** coeffect-gate RawQuery blindness as a PRODUCT fix (make `buildProjections` feed raw-query fold event types into `consumed`); `example/graph-native/README.md`; graph-native row in root README's version manifest table (blocked: module untagged per ADR-0152).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Authored commits keep losing to the daemon.** Waves 1, 2, and the Wave-2 retry all hit `cannot lock ref 'HEAD'` — the pre-commit hook chain (nix fmt + BuildFlow, ~60–90s) exceeds the auto-commit daemon's cadence. Content was never lost (daemon blobs carry it), but the plan's "authored history survives the daemon" goal FAILED for Waves 1–3. Wave 0 survived only via a post-hoc `--amend`.
2. **Reproduced my own API mistake.** I hit `sc.Phase().When undefined` in the example (fixed with sed) — then wrote the SAME invalid call into the recipe fence #4 and its catalog preamble had a `*testing.T` vs `testing.TB` mismatch. Caught by `TestRecipes`, fixed both. Root cause: wrote the fence from memory instead of copying the corrected example.
3. **Published §6.13 claims before verifying them.** Wrote the engine matrix, THEN verified (order should be reversed — I even corrected two claims post-hoc: "fail at Plan time" → apply time; bare `metaengine.QueryDecl` → generic, catalog used `any`). Nothing false shipped, but the order was sloppy.
4. **Stray compiled binary.** `go build ./...` in the module dir produced `example/graph-native/graph-native`; trashed before commit.
5. **Minor write-order slips caught by build/lint:** placeholder type `*systemSystem` in main.go; missing `slices` import; `Events` universe initially declared then dropped (gate-blind discovery) — all fixed in-session, none shipped.
6. **`Then("user.followed")` baseline misunderstanding.** Took 2 attempts to learn the harness diffs against the FIRST-act baseline, not the latest act (test fix + comment). Not documented in §2.43 — future doc gap (see f).

## e) WHAT WE SHOULD IMPROVE

1. **Commit strategy under the daemon.** Either (a) pause/serialize the daemon during wave boundaries, (b) commit with hooks that don't take 60–90s for doc-only changes, or (c) accept `chore:` absorption and stop burning cycles on authored attempts. The current middle ground is worst-of-both.
2. **Verify-then-publish discipline.** Engine matrices and API claims must be grepped from source BEFORE the doc paragraph is written (this session inverted it twice).
3. **Copy fences from proven code.** Recipe/BDD fences should be pasted from the passing example, never retyped.
4. **Coeffect gate RawQuery blindness** is now a documented workaround (`no Events` + comment); the product fix (raw-query folds feeding `consumedEventTypes`) would delete the workaround class — worth an ADR/TODO row.
5. **BuildFlow pre-commit latency.** The 7,600-line report-only output makes every commit slow and noisy; a quiet mode for doc-only commits would fix the race pressure at the root.
6. **`Then*` baseline semantics** deserve one sentence in recipes §2.43 (journal diff = since FIRST act, not latest) — I learned it the hard way; the next consumer will too.

## f) NEXT — up to 50 things (ordered: finish the plan first, then discovered follow-ups)

**Finish Wave 3 (immediate):**

1. D6: `bash scripts/check-readme-links.sh` + `bash scripts/check-readme-deprecated.sh` over touched READMEs
2. D11: full doc-gate sweep (doc-check, md-go, TestRecipes) over everything Wave 3 touched
3. Wave 3 boundary commit (see d-1 for the race caveat)

**Wave 4 — decision docs:**
4. E1: ADR-0155 skeleton — two-level operator story (compile-time nix `-tags`, runtime `cqrs.yaml` via `LoadConfig`), link ADR-0123 §3
5. E2: allengines membership table (memory, sqlite, turso, pg, mysql, badger, iroh, bbolt pure-Go; duckdb behind `//go:build cgo`; bigtable EXCLUDED per A11 facts; pebble include minus graph)
6. E3: nix `-tags engines.X` per-deployment pattern (flake profiles appendix)
7. E4: consequences + open questions; status Proposed
8. E5: scratch binary importing all pure-Go engines; record build size
9. E6: `go mod graph` dep count vs single-engine; numbers into ADR-0155
10. E7: GraphAddEdge signature survey (badger/dgraph/iroh/graphadapter/sqlite)
11. E8: ADR-0156 — edge labels at v5, Option A (extend `Edge`) vs B (documented limitation + `Driver()` hatch); recommend B-for-v4.x; graphadapter doc.go already forward-refs this number
12. E9: cross-link ADR-0156 from advanced.md §6.13 + graphadapter doc.go
13. Wave 4 boundary commit

**Wave 5 — prevention, acceptance, closure:**
14. G1: enumerate `AllADTs()` × recipes.md coverage; report today's gaps (graph now covered; expect spatial/search/streamlog)
15. G2: doc-check meta-test — ADT without a recipe = fail or `// waived: <reason>`
16. G3: waive-or-fix the gaps G1 finds (waive-with-reason or TODO_LIST row)
17. G4: doc-check suite green with the ratchet in
18. G5: T19 dry-run — fresh sub-agent answers "graph-native how?" from the updated docs ONLY
19. G6: fix wherever the dry-run stalls; iterate ≤3
20. G7: second clean dry-run = acceptance GREEN
21. G8: HARVEST into TODO_LIST (remove A0 in-flight marker; add adopt-if-needed markers per A-R1)
22. G9: annotate the 05-48 status report (non-destructive appendix) + verify V1 supersede banner
23. Wave 5 commit + PUSH (user pre-authorized push for the plan's final commit)

**Discovered this session — product/doc follow-ups:**
24. Coeffect gate: teach `buildProjections` to extract event types from `rawQuerySpec` folds → kills the RawQuery workaround (small system/ change, needs its own green-light; violates D1 "zero API changes" if done now)
25. recipes §2.43: document the first-act baseline semantics of `Then*`
26. `example/graph-native/README.md` (other examples have one)
27. Root README version-manifest row for graph-native once tagging is decided (ADR-0152 untagged)
28. Verify COOKBOOK.md fences get any automated checking (they are outside doc-check's scan set; md-go parse gate only?)
29. `id.StreamID.String()` brand-prefix gotcha: consider a core.md conventions note (FAQ entry exists since this session)
30. Integration candidate: example `-dgraph` path against ephemeral Dgraph (`nix run .#integration-dgraph` harness exists)

**Pre-existing red NOT touched this session (noticed, needs owner decision):**
31. `cmd/cqrs-bench` go.sum untidy (`TestEveryModuleGoSumIsTidy` fails)
32. `core/v5` — 14 production packages, no `.go-arch-lint.yml` (meta-test fails; parallel-session territory)
33. BuildFlow preflight FAIL: pseudo-version hygiene — `metaengine/go.mod` sqliteengine `v4.5.2` drifted off zero pseudo-version
34. BuildFlow warn: 34 modules need `go mod tidy`
35. golangci-lint red in `system/`, `scheduling/sqlstore`, `stack/sqlite` (seen in pre-commit run)
36. go-licenses FAIL for `metaengine/bigtableengine` (license detection)
37. BuildFlow's govulncheck runs with go1.26 against go1.27 sources (toolchain mismatch warnings)

**Small polish:**
38. Drop the redundant `errUnexpectedResult` duplication between example main.go and a potential shared helper (fine as-is; note only)
39. Consider `WithAwaitTimeout` tuning in the BDD suite if CI load makes the 5s default tight
40. Recipes §2.44: add an undirected-traversal sentence once an undirected-needing consumer exists (speculative today)
41. system/README: link the COOKBOOK matrix too (currently links advanced.md only)
42. CHANGELOG `[Unreleased]` entries for the example + docs wave (check-changelog-symbols gate will demand exact symbols at commit time)
43. `example/graph-native` in `examplePaths` → run `nix run .#test-examples` leg once before push
44. Confirm no `.cqrs-lint.json` scorecard impact from the new example module (run cqrs-lint over it)
45. Re-run `cmd/api-stability` meta-tests after Wave 5 (they currently fail ONLY on pre-existing items 31–32)

_(46–50 intentionally unused: the list above is 45 real, current items; padding would dilute.)_

## g) Questions I CANNOT answer myself

1. **A-R1 (still open, now decision-shaping):** Does any fleet consumer (cqrs-htmx, go-appkit, or a planned app) actually NEED graph-native reads? If NO, Waves 4–5 should mark the docs "adopt-if-needed" and skip the full-polish pass (plan §2 offers the ~3h shrink). If YES, full treatment as planned.
2. **Daemon vs authored commits:** Waves 1–3 lost their authored messages to auto-commit races (hook chain ~60–90s > daemon cadence). For Waves 4–5: keep attempting authored commits, or accept daemon absorption and let the final PUSH be the only authored boundary? (Or is there a sanctioned way to pause the daemon I shouldn't know about?)
3. **Pre-existing red gates (f-31…37):** cqrs-bench go.sum, core/v5 arch-lint, pseudo-version hygiene, 34 modules needing tidy, golangci red in system/ — all untouched by me and some look like parallel-session territory. Fix any of them in this session, or leave strictly alone?

---

**Verdict:** Waves 0–3 substantively DONE and gated (one behavior-verified example, one compile-verified recipe set, one honesty-fixed doc cluster); Wave 3 needs its final two gate runs; Waves 4–5 untouched. The plan's biggest self-inflicted wound is the commit-daemon race; the biggest discovered product gap is the coeffect gate's RawQuery blindness. Session is PAUSED here per instruction — awaiting directions on questions g-1…g-3 and the go/no-go for Waves 4–5.

---

## Appendix — resolution record (added 2026-10-10, same session, continuing)

Non-destructive; snapshot above untouched. The three §g questions resolved by
user default (continue = full waves, authored commits with amend-on-race,
pre-existing red left alone), then executed:

1. **A-R1 (graph demand):** full Waves 4–5 executed; adopt-if-needed framing
   recorded in TODO_LIST (IN FLIGHT marker → DONE).
2. **Commits:** authored commits attempted each wave; two survived cleanly
   (Wave 3 status report, ADR collision resolution), the rest were
   message-only losses to daemon races on mixed-content commits (content
   always landed; amending shared daemon commits was ruled out).
3. **Pre-existing red:** untouched, still owned by parallel sessions —
   EXCEPT the ADR-0155 numbering collision, resolved: the parallel session's
   ACCEPTED declarative-schema-evolution ADR kept 0155; this wave's engine
   fleet ADR renumbered to **0157** (0156 edge labels unchanged —
   graphadapter doc.go forward-reference preserved).

Wave-level outcomes: Wave 3 D6/D11 green (readme-deprecated clean; the 23
broken README links are ALL pre-existing core/v5, 2026-10-09). Wave 4 =
ADR-0157 + ADR-0156 + measurements (pure-Go fleet: 68,185,216 B / 221
modules / 871 edges vs sqlite-only 12,747,630 B / 64 / 168). Wave 5 = T18
ADT-recipe ratchet (mutation-tested, 5 visible waivers → TODO_LIST row) +
T19 dry-run GREEN after 2 iterations (12 stalls round 1 → 9 fixed, rest
by-design example-pointers) + harvest + this appendix. Final state: push
authorized and executed at Wave 5 close.
