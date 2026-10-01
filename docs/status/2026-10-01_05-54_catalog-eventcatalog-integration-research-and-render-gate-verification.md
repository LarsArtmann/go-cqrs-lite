# Status — catalog/ ⇄ eventcatalog.dev integration: research + render-gate verification

**Timestamp:** 2026-10-01 05:54 CEST
**Session scope:** answer "How is the `catalog/` integration with eventcatalog.dev?"
and _verify_ it end-to-end rather than assert it.
**Production code touched:** none. This was a read + verify session.

---

## 0. One-paragraph truth

The session fully answered the question and **independently reproduced the two
strongest verification gates** the repo already ships: the `catalog/eventcatalog`
package test suite (offline, green) and the full `check-eventcatalog` render gate
(generate fixture → `npm install` → `npx eventcatalog build` → `@eventcatalog/linter`),
which passed clean including the plain-refs lint profile. No code was changed, so
nothing regressed. The main weaknesses were: the answer mostly _read the docs_ and
only spot-checked code; several sub-claims (manifest determinism, the linter pin,
`GenerateEventCatalog` wiring) were left unverified; and the verification was done
by invoking `scripts/check-eventcatalog.sh` directly rather than through `nix run .#check-eventcatalog`.

---

## 1. FULLY DONE

| #  | Item                                                  | Evidence                                                                                                                                       |
| -- | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Answered the architecture question coherently         | final answer mapped pipeline, contract, ref-split, hub role, known bugs                                                                        |
| 2  | Located the integration surface                       | `catalog/eventcatalog/` (28 non-test `.go` files), `catalog/cmd/ec-fixture`, `catalog/cmd/catalog-export`, `catalog/docserver/eventcatalog.go` |
| 3  | Read the canonical maintainer contract                | `catalog/AGENTS.md:57-113` (EventCatalog exporter section)                                                                                     |
| 4  | Read the consumer docs                                | `catalog/README.md:328-502`, `catalog/eventcatalog/doc.go`, `CATALOG_ARCHITECTURE.md`                                                          |
| 5  | Confirmed the core version pin                        | `eventCatalogCoreVersion = "^4.6.3"` at `catalog/eventcatalog/exporter.go:19`; generated `package.json` at `writer.go:263`                     |
| 6  | Confirmed the stable `cId` mechanism                  | `stableCatalogID` at `catalogid.go:44` (UUIDv5 under frozen namespace)                                                                         |
| 7  | Confirmed the two-consumer ref split                  | `options.go:23` (`WithPlainRefIDs`), `options.go:7` (`WithSkipBootstrapFiles`)                                                                 |
| 8  | Confirmed changelog guard                             | `shouldEnableChangelog` at `writer.go:238` (skips agents, upstream 4.6.3 crash)                                                                |
| 9  | Ran the package test suite                            | `GOWORK=off go test ./eventcatalog/...` → `ok ... 0.043s`                                                                                      |
| 10 | Ran the **full render gate**                          | `CHECK_EVENTCATALOG_DIR=/tmp/ec-verify bash scripts/check-eventcatalog.sh` → `OK: eventcatalog build clean + linter clean`                     |
| 11 | Verified default profile build                        | 88 pages, schema-clean logs, no `InvalidContentEntryDataError`                                                                                 |
| 12 | Verified changelog profile build                      | 94 pages, `changelog.mdx` rendered; only the documented cosmetic 4-link WARN                                                                   |
| 13 | Verified `@eventcatalog/linter` on plain-refs profile | `✔ No problems found! (15 files checked)` at full severity                                                                                     |
| 14 | Confirmed network + toolchain availability            | `node`/`npm`/`npx` present; `registry.npmjs.org:443` reachable                                                                                 |

---

## 2. PARTIALLY DONE

| # | Item                                    | What's missing                                                                                                                                                                                                                             |
| - | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | The explanation itself                  | Mostly derived from `README.md` + `AGENTS.md`; only ~10 code sites were opened to spot-check. The full resource-coverage table was NOT independently re-derived from `exporter_*.go`, so a doc-vs-code drift there would have gone unseen. |
| 2 | Verification breadth                    | 2 gates run (package tests + render gate). The recipes compile harness (`TestRecipes*`), `check-md-go`, `check-eventcatalog` via **nix** (uses the flake's node/bash wiring), and `docserver.GenerateEventCatalog` were not exercised.     |
| 3 | `catalog.index.json` claim              | Repeated the README's determinism claim (canonical order, written last) but never opened a generated manifest to confirm field shape / byte-stability.                                                                                     |
| 4 | "Every resource kind" claim             | Backed by the fixture's existence, not by enumerating `exporter_resources*.go` against the EventCatalog 4.6.3 collection list.                                                                                                             |
| 5 | Linter pin                              | Noted `scripts/testdata/eventcatalog-linter/` is lockfile-pinned; did not read the `package.json` version or check it against upstream.                                                                                                    |
| 6 | Duplication state of the report sources | Did not check whether README/AGENTS/skill rows for EventCatalog have drifted from each other (three copies of similar tables).                                                                                                             |

---

## 3. NOT STARTED

1. Did not run `nix run .#check-eventcatalog` (the sanctioned path; the script was invoked directly under the ambient env).
2. Did not run `cd cmd/doc-check && GOWORK=off go run .` over the skill refs / AGENTS to confirm the EventCatalog claims are still gate-clean.
3. Did not inspect the `ec-fixture` **default** profile output tree on disk (only its build result).
4. Did not open `catalog/index_manifest.go` / `message_index.go` to confirm the manifest + channel-pointer mechanics first-hand.
5. Did not verify `catalog/cmd/catalog-export/main.go` template actually compiles/exports standalone.
6. Did not verify the hub's `sources.json` contract against a real hub checkout (external repo, out of scope but reachable).
7. Did not check whether the upstream core changelog crash is fixed in a release newer than 4.6.3 (would let the guard be dropped).
8. Did not run any lint/format (correct: no Go files changed).
9. Did not capture the render-gate log into the repo (`/tmp/ec-verify/build.log` is ephemeral).
10. Did not add a regression test pinning any newly-noticed behavior (none was changed).

---

## 4. TOTALLY FUCKED UP

Nothing materially broken. Honest ledger of the sloppy bits:

| # | Sloppiness                                                                                                                                              | Impact                                                                                                |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| 1 | **Answered from docs first, verified second.** I read `README.md` before opening code, then spot-checked. Risk: doc drift presented as fact.            | Medium — mitigation: the two gates passed, so the _behavior_ is real even if a doc row is stale.      |
| 2 | **`lsp_symbols` timed out twice** (`context deadline exceeded`) and I fell back to `grep`. Worked, but suggests an LSP that is unhealthy on this repo.  | Low — no correctness impact, but worth a `lsp_restart`.                                               |
| 3 | **Ran the gate via `bash scripts/...` not `nix run .#check-eventcatalog`.** The nix app injects the pinned node/bash; the ambient env happened to work. | Low — result is valid this run, but not the _canonical_ invocation; not reproducible-by-construction. |
| 4 | **Produced no artifact until this report.** The research answer lived only in chat.                                                                     | Low now (this file exists), was medium mid-session.                                                   |
| 5 | **Did not diff the three doc copies** of the EventCatalog surface (README vs AGENTS vs skill) against the code.                                         | Medium — this is exactly the "split brain" class the repo polices.                                    |

---

## 5. WHAT WE SHOULD IMPROVE

1. **Verify before you narrate.** For an integration question, open the code that
   proves each table row _before_ writing the row. The repo has a whole
   `verify-before-filing` / `verify-external-claims` culture; apply it inward too.
2. **Use the sanctioned gate command.** `nix run .#check-eventcatalog`, not the raw
   script, so the toolchain is pinned.
3. **Persist gate evidence.** A render gate that takes ~3 min and needs network
   should leave a durable receipt (log tail + page counts) — chat-only evidence
   evaporates.
4. **Cross-check the doc triplet.** Any EventCatalog claim exists in ≥3 files; a
   `check-*` leg that diffs the resource-coverage tables would kill the drift class.
5. **Health-check the LSP.** Two `document symbol` timeouts is a signal
   (`lsp_restart` / model provider), not noise.

---

## 6. UP TO 50 NEXT THINGS

Ordered by leverage for the `catalog/eventcatalog` integration specifically.

### Verification (close this session's gaps)

1. Re-run via `nix run .#check-eventcatalog` and capture the tail into a status receipt.
2. Export the default fixture to a dir and inspect `catalog.index.json` shape + byte-determinism (export twice, `diff`).
3. Confirm `catalog.index.json` is written last (code path in `exporter.go`).
4. Enumerate `exporter_resources*.go` against EventCatalog 4.6.3's collection list; fix any missing/extra row.
5. Read `index_manifest.go` + `message_index.go`; document the channel-pointer resolution.
6. Run `catalog/cmd/catalog-export` template standalone; confirm exit 0 + tree.
7. Read `scripts/testdata/eventcatalog-linter/package.json`; record the pinned linter version.
8. Check whether core > 4.6.3 fixes the agent-changelog crash; if yes, schedule guard removal.
9. Diff README/AGENTS/skills EventCatalog tables for drift.
10. Run `cd cmd/doc-check && GOWORK=off go run .` over AGENTS + skill refs.
11. Run `cd cmd/doc-check && GOWORK=off go test -run TestRecipes .` (EC recipe fences).
12. Confirm the known cosmetic 4-link WARN is still 4 (not growing) on the changelog profile.
13. Verify `docserver.GenerateEventCatalog` writes the same tree as `NewExporter` (no divergence).
14. Check `check-eventcatalog.sh` handles `CHECK_EVENTCATALOG_DIR` re-runs idempotently.

### Contract hardening

15. Add a golden pin for `eventcatalog.config.js` (cId + conditential changelog line).
16. Add a golden pin for the generated `package.json` (core pin stability).
17. Add a test asserting `WithSkipBootstrapFiles` omits EXACTLY two files and nothing else.
18. Add a test asserting `WithPlainRefIDs` changes refs but not resource paths.
19. Add a determinism test: two exports of the same catalog → identical `catalog.index.json`.
20. Add a schemaVersion-rejection test (consumer rejects higher versions).
21. Pin the `x-*` custom-property names in a test (labels/responses/team role/avatar).
22. Add a test for the changelog guard (agents present → `changelog` block absent).
23. Add a test for flow step node kinds (`container`, `flow`; no channel step).
24. Add a round-trip test: exported tree passes `@eventcatalog/linter` in CI (already in gate, isolate it).
25. Document the exact error string emitted for `BaseConfig.ResourceGroups` (Rejection family).

### Robustness / DX

26. Make the render gate fail loudly if the network is unavailable (distinct exit code), vs silent skip.
27. Add a `--offline` mode that validates against cached zod schemas only.
28. Cache `node_modules` between gate runs (currently reinstalled).
29. Emit page-count deltas when the gate re-runs (spot silent loss).
30. Add a `catalog.index.json` diff helper for hub PRs (tool, not just doc).
31. Add a `cmd/catalog-export` `-plain`/`-skip-bootstrap` flag so hub CI is one binary.
32. Add `eventcatalog.NewExporter` doc examples that compile (doc-check).
33. Add a faq.md entry for "InvalidContentEntryDataError" debugging steps.
34. Add a faq.md entry for the cosmetic changelog WARN (stop re-triaging it).
35. Record the hub `sources.json` schema in-repo (contract for onboarders).

### Governance / process

36. Add a `check-eventcatalog-doctables` leg diffing README vs AGENTS vs skill.
37. Add CI job for the render gate (currently local/nightly-ish) if not already.
38. Record the render-gate runtime budget (~3 min) so quiet-window tooling can schedule it.
39. Add a `--no-changelog-profile` fast path for smoke.
40. File the upstream agent-changelog crash issue if still unfiled (draft exists — see KEEP-LIVE status doc).

### Cross-cutting noticed during this session

41. `lsp_restart` and confirm document symbols respond on `catalog/eventcatalog`.
42. Confirm the ambient env vs `scripts/go-env.sh` for the raw-script invocation.
43. Add a status-report receipt convention for network-dependent gates.
44. Verify `catalog.index.json` is gitignored or committed deliberately in examples.
45. Check `catalog/ec-fixture` dir (non-cmd) purpose vs `cmd/ec-fixture`.
46. Audit `eventcatalog` test files for `t.Parallel()` opportunities (0.043s, low value).
47. Confirm `exporter_error_test.go` covers every `errorfamily` code emitted.
48. Confirm `catalog/eventcatalog/benchmark_test.go` thresholds are honest.
49. Verify the `catalog` module's dep budget (5, FULL) is unchanged.
50. Add the EventCatalog surface to `docs/agents/module-map.md` notes if absent.

---

## 7. THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Do you want me to close this session's verification gaps now** (items 1–14),
   i.e. re-run via `nix run`, inspect the manifest, and diff the doc triplet — or
   was the render-gate pass sufficient proof for you?
2. **Should the render-gate evidence become a committed artifact** (a
   `docs/status/…` receipt or a machine-readable `catalog.index.json` diff in PRs),
   or stay ephemeral because it needs network + ~3 min?
3. **Is the upstream `@eventcatalog/core` agent-changelog crash issue already filed**
   upstream (the repo holds a KEEP-LIVE draft), and do you want it filed now?

---

## 8. VERIFICATION RECEIPTS (this session)

```text
$ date
Thu Oct  1 05:54:05 AM CEST 2026

$ cd catalog && GOWORK=off go test ./eventcatalog/... -count=1
ok   github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog   0.043s

$ CHECK_EVENTCATALOG_DIR=/tmp/ec-verify bash scripts/check-eventcatalog.sh
  default   profile: build Complete! 88 page(s) built in 17.45s
  changelog profile: build Complete! 94 page(s) built in 14.50s
                     WARN: 4 broken sidebar changelog links (documented cosmetic)
  default   profile: build Complete! 88 page(s) built in 15.53s
  plain-refs linter: ✔ No problems found! (15 files checked)
  OK: eventcatalog build clean + linter clean (log: /tmp/ec-verify/build.log)
```

No Go file changed → no lint/format/fmt obligation triggered.
