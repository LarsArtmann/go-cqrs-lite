# Status Report — 2026-07-25 TODO_LIST Execution

**Session:** 2026-07-25 ~07:00–08:00
**Goal:** Break down `TODO_LIST.md`, execute, verify.
**Scope:** CI quality gate (file splits, otel flakiness, sentinels), module tagging, doc cross-links.

---

## a) FULLY DONE (19/20 tasks — verified)

| #     | Task                                                                                                                                                                            | Verification                                                                          |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| C1    | Cross-link `docs/CONSISTENCY_MODEL.md` in `docs/README.md`                                                                                                                      | link resolves ✅                                                                      |
| C2    | Add ADR-0061..0065 links in `AGENTS.md` (+ corrected wrong ADR-0047≠metaengine ref → it's COSE)                                                                                 | links resolve ✅                                                                      |
| C3    | Reference NATS + Parquet design docs in `.agents/skills/.../recipes.md` §2.13                                                                                                   | links resolve ✅ (caught + fixed wrong depth: needed `../../../../`, not `../../../`) |
| A3    | Restore truncated otel sentinels (`"shutdown"` → `"otel provider shutdown incomplete"`, `"build resource"` → `"failed to build OTel resource"`)                                 | `go test ./otel/...` ✅                                                               |
| A2    | **Fix otel test flakiness** — added `WithoutGlobalRegistration()` option; guarded 3 global Set calls; applied to 9 parallel Setup tests; made 1 global-writer test non-parallel | `go test -race -count=10 ./otel/...` ✅ + prometheus + cross-package ✅               |
| A1-1  | Split `benchkit/phases.go` 610→216 / 165 / 156 / 98 (`phases_read.go`, `phases_projection.go`, `phases_durability.go`)                                                          | build + `go test` ✅                                                                  |
| A1-2  | Split `benchkit/runner.go` 498→294 / 120 / 97 (`runner_discovery.go`, `runner_concurrent.go`)                                                                                   | build + vet ✅                                                                        |
| A1-3  | Split `benchkit/benchkit.go` 368→260 / 111 (`result.go`)                                                                                                                        | build + vet ✅                                                                        |
| A1-4  | Split `cmd/cqrs-bench/main.go` 602→286 / 62 / 169 / 106 (`flags.go`, `factory.go`, `output.go`)                                                                                 | build ✅                                                                              |
| A1-5  | Split `cmd/cqrs-lint/main.go` 452→152 / 226 / 96 (`run.go`, `diagnostics.go`)                                                                                                   | build + `go test ./cmd/cqrs-lint/` ✅                                                 |
| A1-6  | Split `cmd/cqrs-lint/pkg/analyzer/scanner_calls.go` 412→296 + `scanner_calls_helpers.go` 119                                                                                    | build + `go test ./.../analyzer/` ✅                                                  |
| A1-7  | Split `cmd/cqrs-lint/pkg/analyzer/scanner.go` 387→263 + `scanner_resolve.go` 131                                                                                                | build + analyzer test ✅                                                              |
| A1-8  | Split `projectionhost/host.go` 403→306 + `host_reset.go` 107                                                                                                                    | build + `go test ./projectionhost/...` ✅                                             |
| A1-9  | Split `storage/relational/sink.go` 378→314 (moved 3 methods into existing `sink_helpers.go` → 173)                                                                              | build + test ✅                                                                       |
| A1-10 | Split `codec/cose.go` 376→303 + `cose_helpers.go` 81                                                                                                                            | build + test ✅                                                                       |
| A1-11 | Split `graph/schema.go` 368→301 + `schema_validate.go` 71                                                                                                                       | build + test ✅                                                                       |
| A1-12 | **`check-file-size` gate now GREEN** — all 11 offenders <350 lines                                                                                                              | gate script ✅                                                                        |
| B1a   | Tagged **`metaengine/v4.0.0`** (annotated, release-clean go.mod)                                                                                                                | `git cat-file -t` = tag ✅                                                            |
| B1b   | Tagged **`idempotency/sqlstore/v4.0.0`** (annotated, release-clean go.mod)                                                                                                      | `git cat-file -t` = tag ✅                                                            |
| B2    | **Pushed `benchkit/v4.1.0` to origin**                                                                                                                                          | `git ls-remote --tags origin` shows it ✅                                             |
| —     | **Bonus:** fixed latent bug in `scripts/tag-release.sh` (grep -P no-match aborted under `pipefail` on non-cqrs replace directives → appended `\|\| true`)                       | `bash -x` trace confirmed ✅                                                          |

**Gate status:** file-size GREEN, otel flakiness FIXED, sentinels restored, doc links valid, 3 modules unblocked for release.

---

## b) PARTIALLY DONE

1. **`metaengine/projectionadapter/v4` NOT tagged** — its `go.mod` has `metaengine/v4 => ../` local replace; can't resolve metaengine from the Go proxy because **metaengine/v4.0.0 is tagged locally but not pushed**. Ordering: push metaengine → tag projectionadapter. (Only 2 of the 3 untagged modules shipped.)
2. **`nix run .#verify` not run end-to-end.** I ran per-module `go test`/`go build`/`go vet` for every changed package, and `-race -count=10` on otel. I did **not** run the full workspace gate (build+vet+test+race+lint+doc-check+module-coverage in one pass). The file-size gate passed via the standalone script; lint (`golangci-lint`) and doc-check (`cmd/doc-check`) were not exercised on my new files.
3. **No CHANGELOG / FEATURES / TODO_LIST update** — `TODO_LIST.md` still lists all the items I finished. The "move completed work to CHANGELOG" housekeeping was not done.

---

## c) NOT STARTED

- `metaengine/projectionadapter/v4` tag (blocked — see b.1)
- Full `nix run .#verify` run
- `CHANGELOG.md` entries for the 11 splits + otel fix
- `TODO_LIST.md` pruning (remove the 7 finished themes)
- Lint gate pass on the 16 new split files (gofmt ran on `cmd/cqrs-lint/main.go` only; other new files formatted by hand)

---

## d) TOTALLY FUCKED UP (mistakes, self-caught)

1. **Typo in setup.go edit** — wrote `pel.SetMeterProvider(mp)` instead of deleting the line during the multiedit. Caught immediately via View, fixed before any build. Zero impact, but sloppy.
2. **Wrong relative path in recipes.md** — first attempt used `../../../` (3 levels up); correct is `../../../../` (4 levels — recipes.md is under `.agents/skills/go-cqrs-lite/references/`). Caught by the `realpath` verification step I ran. Fixed before declaring C3 done.
3. **Wrong ADR number in AGENTS.md** — the original `(ADR-0047, ADR-0061-0063)` claimed ADR-0047 was metaengine; ADR-0047 is actually "COSE Support". Corrected during C2.
4. **Mis-stated scope in the plan** — I told the user "13 oversized files" in the plan echo; the gate excludes `*/example/*` and `*.pb.go`, so only **11** were real offenders (`example/taskmanager/decider.go` 555 and `cqrs.pb.go` 570 are exempt). The plan table had 13 rows; I executed the correct 11.
5. **Raced the auto-git daemon on commit** — my `git commit` for the tag-release.sh fix hit `cannot lock ref HEAD` because the daemon moved HEAD underneath me (`905663cd` → `27c16791`). The daemon ended up committing my fix anyway. No data lost, but I wasted a round trip and should have checked `git log` immediately before committing in a daemon-driven repo.
6. **Almost wrote a useless post-hoc HTML "plan report"** — the user correctly stopped me. A planning document after the work is done is ceremony, not value. Should have jumped straight to this status report.

---

## e) WHAT WE SHOULD IMPROVE

1. **Run `nix run .#verify` before declaring victory**, not just per-module tests. The file-size gate is one of ~7 sub-checks.
2. **gofmt all new files in one pass** (`gofmt -w <list>`) instead of one-at-a-time. I only caught `cmd/cqrs-lint/main.go` needing fmt.
3. **Update living docs (TODO_LIST, CHANGELOG, FEATURES) as part of "done"**, not as a forgotten tail. The AGENTS.md docs-health skill exists for this.
4. **Check dependency ordering before tagging a multi-module release.** projectionadapter→metaengine proxy dependency was foreseeable; I discovered it at tagging time.
5. **The new `otel.WithoutGlobalRegistration()` is public API** — it should be documented in `AGENTS.md` (the OTel section) and the Crush skill `references/core.md`, plus get a dedicated test asserting globals are _not_ mutated. I added the option and used it in tests but did not document it as a consumer-facing feature.
6. **`tag-release.sh` is fragile** — the grep bug I fixed is one of likely several. It also commits internally and races the daemon. Worth a dedicated review (nix-review or bash strict-mode audit).
7. **2 unauthored dirty files appeared** (`metaengine/sqlite_backends.go`, `stack/pebble/preset.go`) — BuildFlow auto-removed `//nolint:wrapcheck` comments. I left them (no-revert rule) but did not report them until now. Should have surfaced immediately.
8. **The "Rejected" section of TODO_LIST** was correctly excluded from work — good — but I should have explicitly said so in the plan rather than silently dropping it.

---

## f) Next up to 50 things (rough Pareto order)

**Release finish (highest value):**

1. Push `metaengine/v4.0.0` + `idempotency/sqlstore/v4.0.0` tags to origin (blocked on your push authorization)
2. After #1, tag `metaengine/projectionadapter/v4.0.0` (run tag-release.sh — now that metaengine is on the proxy)
3. Run full `nix run .#verify` and fix anything red
4. Update `CHANGELOG.md` with the 11 file splits + otel fix + tag-release.sh fix
5. Prune `TODO_LIST.md` — remove the 7 finished themes, keep only projectionadapter + whatever verify surfaces

**otel hardening:** 6. Add a dedicated test `TestSetup_WithoutGlobalRegistration_DoesNotMutateGlobals` (assert Get*Provider unchanged before/after) 7. Document `WithoutGlobalRegistration()` in `AGENTS.md` OTel section + Crush skill `references/core.md` 8. Audit `prometheus/exporter_test.go:87` — it still sets the global meter provider with only `t.Cleanup` reset; consider `WithoutGlobalRegistration` there too 9. Consider making `Setup()` default to no-global-registration and require an explicit `WithGlobalRegistration()` opt-in (breaking — needs ADR)

**Split polish / lint:** 10. `gofmt -w` all 16 new split files in one pass, confirm zero diff 11. Run `nix run .#lint` and clear any new findings on the split files 12. Run `cmd/doc-check` on AGENTS.md + SKILL.md + recipes.md (part of verify) — confirm my new doc links + code refs are valid 13. Add file-header package comments to the new split files where the original had a file-level doc comment (e.g. `phases.go` lost its file-grouping comment)

**tag-release.sh / release tooling:** 14. Audit `scripts/tag-release.sh` end-to-end with `bash -x` for other `pipefail` traps like the one I fixed 15. Add a `--dry-run` mode to tag-release.sh (it currently commits + resets) 16. Make tag-release.sh tag a single module without touching all 58 go.mod files (it's all-or-nothing) 17. Consider a `scripts/check-release-ready.sh` that reports which modules have clean go.mod + no unpushed tags

**Docs health:** 18. Run the `docs-health` skill to detect drift between TODO_LIST / FEATURES / CHANGELOG / AGENTS after this session 19. Add the 3 newly-tagged modules to `FEATURES.md` status table 20. Update `docs/README.md` ADR table — it stops at ADR-0035 then jumps to ADR-0046; ADRs 0036–0065 are missing from the index

**Test coverage:** 21. The 16 new split files have no per-file tests — they're covered by existing package tests, but coverage attribution changed. Re-run coverage report. 22. `benchkit` `GOWORK=off go test` fails on go.sum drift (pre-existing) — worth a `go mod tidy` pass or a CI note 23. The `example/taskmanager/decider.go` (555 lines) is gate-exempt but is a readability smell — consider splitting for the example's sake

**Operational:** 24. The BuildFlow daemon auto-removed 2 `//nolint:wrapcheck` directives — audit whether those nolints were actually needed (if lint passes without them, good; if not, the daemon broke lint) 25. `otel/golden_test.go` has 2 gopls `[gopls stdversion]` warnings (`json.Marshal requires go1.27`) — pre-existing, tied to the `goexperiment.jsonv2` flag; worth a note in AGENTS.md so it's not "fixed" repeatedly 26. Add `WithoutGlobalRegistration` to the `otel` module's API stability snapshot (`api_surface.txt`) if that's tracked

**Lower priority:** 27. Consolidate the 3 benchkit `phases_*.go` doc comments — they duplicate the "phase" framing 28. The `cmd/cqrs-bench/output.go` `fatalf`/`version` could move to a `util.go` for clarity 29. Graph `schema_validate.go` could grow a `Validate*Ref` bench test (hot path on large schemas) 30. `projectionhost/host_reset.go` — the `Reset` method has no direct unit test (covered via host integration); add one
31–50. _(Reserve for whatever `nix run .#verify` surfaces — likely lint nits, doc-check symbol drift, and module-coverage gaps. I won't invent 20 more speculative items.)_

---

## g) Questions I cannot figure out myself (3)

1. **Push the 2 new tags?** `metaengine/v4.0.0` and `idempotency/sqlstore/v4.0.0` exist **locally only**. They're useless to consumers until pushed, and projectionadapter can't be tagged until metaengine is on the proxy. You authorized pushing benchkit/v4.1.0 explicitly — do I have the green light to push these 2 as well (and master, which is 28+ commits ahead of origin)?

2. **Run the full `nix run .#verify` now?** It's the real CI gate (build+vet+test+race+lint+doc-check+module-coverage, several minutes). I verified each changed package in isolation but not the whole gate in one pass. Should I run it and fix whatever it surfaces, or do you want to run it yourself?

3. **Update `TODO_LIST.md` / `CHANGELOG.md` as part of "done"?** The docs-health rule says completed work moves from TODO_LIST → CHANGELOG. I did the work but left both files untouched (the daemon committed code, not docs). Should I now prune TODO_LIST and write the CHANGELOG entries, or is that your manual step?

---

## Resolution (2026-07-26)

All three open questions from this report are now resolved:

1. **metaengine/v4.1.1 + projectionadapter/v4.0.0 tagged** — both pushed to origin. projectionadapter was the last untagged module; the full 58/58 module graph is now tagged (ADR-0062 dependency boundary).

2. **`nix run .#verify` is GREEN** — build + vet + test + race + lint (0 issues across all 58 modules) + API stability + doc-check (948 references) all pass end-to-end.

3. **TODO_LIST and CHANGELOG rebuilt** — the docs-health session (2026-07-26) pruned stale items, harvested new ones, and recorded shipped work in CHANGELOG `[Unreleased]`.

The ADR index gap noted in this report (f.20: "stops at ADR-0035") is fixed: docs/README.md now indexes all 67 ADRs (0001–0069, gaps 0036/0041), and a CI check in `scripts/verify-docs.sh` prevents future drift.
