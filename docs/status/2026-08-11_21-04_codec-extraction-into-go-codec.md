# Status: codec/ Extraction into go-codec — 2026-08-11

> Extraction of `codec/` into a standalone `go-codec` repo. Functional but
> with known gaps and one correctness issue.
>
> **Update 2026-08-11:** Extraction shipped — see CHANGELOG `[Unreleased]`.
> Remaining open (publish go-codec to GitHub, delete dead dirs, write ADR) tracked
> in TODO_LIST → Codec Extraction.

---

## a) FULLY DONE

1. **Standalone go-codec project created** at `/home/lars/projects/go-codec/`
   - All 15 source files, 16 test files, testdata (fuzz + golden snapshots)
   - `go.mod`: `module github.com/larsartmann/go-codec`
   - `go mod tidy` clean, all tests pass with `-race`
   - Has git repo with 3 commits (auto-commit daemon)
   - Has AGENTS.md, README.md, LICENSE, CHANGELOG, CONTRIBUTING

2. **codec/ converted to deprecated alias module**
   - `alias.go`: all 9 types, 15 constants, 7 errors, 25+ functions re-exported
   - `doc.go`: DEPRECATED notice with migration instructions
   - `README.md`: rewritten with deprecation notice
   - `go.mod`: requires `go-codec v0.1.0`
   - All implementation and test files deleted

3. **All 114 .go files migrated** from `go-cqrs-lite/codec/v4` → `go-codec`
   across 53 consumer modules (event, signing, encryption, storage, stack,
   transport, middleware, benchkit, decider, etc.)

4. **All 53 go.mod files updated** — requires changed to `go-codec v0.1.0`,
   old replace directives removed from encryption/signing/transport-http

5. **go.work configured** with `replace github.com/larsartmann/go-codec => ../go-codec`
   (needed because repo is unpublished)

6. **go.sum cleanup** — stale `codec/v4` hash entries removed from all go.sum files

7. **Infrastructure updated**:
   - `cmd/api-stability` golden regenerated (4097 exports)
   - `cmd/doc-check` passes (747 refs valid, 44 packages)
   - `cmd/cqrs-lint/pkg/analyzer/module_catalog_data.go`: added `go-codec` hint
   - AGENTS.md module map and tier table annotated as DEPRECATED
   - CHANGELOG.md updated with extraction entry
   - `.agents/skills/go-cqrs-lite/references/recipes.md` import updated

8. **Verified**:
   - Full workspace build passes: `go build -tags "goexperiment.jsonv2" ./...`
   - Key module tests pass: event, signing, encryption, kv, snapshot, command,
     query, decider, schema, storage, stack, watermill, transport, middleware,
     benchkit, cqrs-gen, cqrs-lint/lintutil
   - `go vet` passes on codec-dependent modules
   - api-stability meta-tests pass

---

## b) PARTIALLY DONE

1. **go-codec project scaffolding incomplete** — Has README, LICENSE, CHANGELOG,
   CONTRIBUTING, AGENTS.md. **Missing**: `.golangci.yml`, `.github/workflows/ci.yml`,
   FEATURES.md, ROADMAP.md, TODO_LIST.md, SECURITY.md. The extracted go-retry
   and go-idempotency repos have all of these.

2. **Historical docs not updated** — 9 files under `docs/` still reference
   `go-cqrs-lite/codec/v4` (DOMAIN_LANGUAGE.md, design docs, feedback, planning,
   status reports). These are historical snapshots and may not need updating,
   but DOMAIN_LANGUAGE.md is a living doc.

3. **Consumer-facing skill references partially updated** — `recipes.md` was
   updated but no other reference files were checked line-by-line for codec
   usage examples or import paths (though grep shows 0 hits for the old path).

---

## c) NOT STARTED

1. **flake.nix not updated** — `codec` is still in `testModules` (line 176) and
   `wasmMods` (line 924). The codec module is still a valid module (alias), so
   this works, but a comment noting deprecation would be appropriate.

2. **go-codec repo not published** — No GitHub repo exists. The `go.work` uses
   a `replace` directive as a workaround. `go mod tidy` fails in consumer
   modules because it can't fetch go.sum hashes from the network.

3. **Per-module `go mod tidy` not run** — Because the repo is unpublished,
   `go mod tidy` fails. go.sum files have stale entries removed but may be
   missing `go-codec` hash entries. Builds work via workspace `replace`.

4. **No verification of `nix run .#verify`** or `nix run .#lint` — only `go build`
   and `go test` were run directly.

5. **No ADR written** for the extraction (retry has ADR-0064, idempotency has
   ADR-0065; codec extraction has no ADR).

---

## d) TOTALLY FUCKED UP

1. **go-codec `snaps_clean_test.go` had a pre-existing bug** — `_ = snaps.Clean(m)`
   was an assignment mismatch (snaps.Clean returns 2 values). Fixed to
   `snaps.Clean(m)`. **But the original codec/ had this same bug** — it was
   masked because the workspace resolved a different go-snaps version or the
   test wasn't being run in isolation. This means the original codec/ tests
   may not have been running standalone either.

2. **`go mod tidy` is broken in all consumer modules** — Because go-codec is
   unpublished, running `go mod tidy` in ANY consumer module fails with
   `repository not found`. This is a **blocker for CI** and for anyone running
   `go mod tidy` locally outside the Nix devShell. The workspace `replace`
   directive only helps within `go build`/`go test`, not `go mod tidy`.

3. **codec/testdata/ left behind** — The `codec/testdata/` directory still
   exists with golden snapshots, but all test files that used them were deleted.
   Dead directory.

4. **codec/reports/ left behind** — A `codec/reports/` directory exists but
   wasn't examined. Possibly stale.

---

## e) WHAT WE SHOULD IMPROVE

1. **The `replace` directive in go.work is a time bomb** — It works now but
   will confuse future sessions. It MUST be converted to a `use` directive once
   the repo is published. Add a TODO with a clear deadline.

2. **The alias module approach is correct but heavyweight** — The `alias.go`
   file manually lists every symbol. If go-codec adds new exports, the alias
   won't pick them up automatically. Consider whether this is acceptable for
   a deprecated module (probably yes — we don't want new symbols leaking).

3. **Should have checked `nix run .#verify`** — The session only ran `go build`
   and `go test` directly. The Nix-based verification (`#verify-fast` at minimum)
   would catch lint, doc-check, and architecture issues.

4. **Should have initialized the go-codec repo properly** — Missing CI, lint
   config, and project docs. The go-retry/go-idempotency repos set the pattern;
   go-codec should match.

5. **The extraction should have an ADR** — Every prior extraction (retry →
   ADR-0064, idempotency → ADR-0065) has an ADR explaining the decision.

---

## f) Up to 50 Things to Get Done Next

### Critical (blocks publish/CI)

~~1. Publish `go-codec` to GitHub (`github.com/larsartmann/go-codec`)~~ done - published (github.com/larsartmann/go-codec)
~~2. Tag `go-codec` with `v0.1.0`~~ done - v0.1.0+ resolving through the proxy
~~3. Remove `replace` directive from `go.work`, add `../go-codec` to `use` block~~ done - ../go-codec in the go.work use block
~~4. Run `go mod tidy` in all 53 consumer modules to get proper go.sum entries~~ done - mass tidy via 94261a568 (79 modules)
~~5. Verify `go mod tidy` works in consumer modules after publish~~ done - tides green; standalone builds green
6. Delete `codec/testdata/` (dead directory) <- NOT-DO - codec/ deleted entirely at 5127039da (ADR-0128); testdata gone with it
7. Check and clean up `codec/reports/` if stale <- NOT-DO - same: reports/ gone with the deletion
~~8. Run `nix run .#verify` (or at minimum `#verify-fast`)~~ done at 5f2198189
~~9. Run `nix run .#lint` to check for lint issues across all modules~~ done - 76/76 clean since 444be10a7
~~10. Run `nix run .#check-arch` to verify dependency budgets still pass~~ done - Check Arch green since 8c384f0f5

### go-codec project setup

~~11. Add `.golangci.yml` to go-codec (copy from go-retry pattern)~~ done - go-codec has .golangci.yml (pareto T18 verified)
~~12. Add `.github/workflows/ci.yml` to go-codec~~ done - .github/workflows/ci.yml present (pareto T18)
13. Add `.github/dependabot.yml` to go-codec
~~14. Write `FEATURES.md` for go-codec~~ done - FEATURES.md present (pareto T18)
15. Write `ROADMAP.md` for go-codec
16. Write `TODO_LIST.md` for go-codec
~~17. Write `SECURITY.md` for go-codec~~ done - SECURITY.md present (pareto T18)
18. Verify go-codec passes `golangci-lint run` standalone
19. Add go-codec to the go-codec README: detailed usage sections (CBOR tags, streaming, etc.)

### Documentation

~~20. Write ADR for codec extraction (next ADR number after latest)~~ done - ADR-0128 written and shipped
~~21. Update `docs/DOMAIN_LANGUAGE.md` — change codec import path~~ done - DOMAIN_LANGUAGE import paths fixed (12-40 session)
22. Update `flake.nix` — add comment noting codec/ is deprecated alias <- NOT-DO - codec/ removed from flake entirely at 5127039da
~~23. Update `.agents/skills/go-cqrs-lite/references/modules.md` — annotate codec as deprecated~~ done - references point at the external repo; doc-check green
~~24. Update SKILL.md if it references codec module map entry~~ done - SKILL.md swept; doc-check green
~~25. Scan all `.agents/skills/go-cqrs-lite/references/*.md` for codec usage examples~~ done - references swept at 2e9a2fc28-wave; doc-check 797 refs green
~~26. Update `cmd/cqrs-gen/main.go` template if it generates codec imports in consumer code~~ done - cqrs-gen builds; no stale codec imports possible (module gone)
~~27. Verify `cmd/cqrs-lint` architecture rules still recognize go-codec as Tier 0~~ done - cqrs-lint F007 + ImportHints recognize go-codec; lint green

### Testing & Verification

~~28. Run full test suite with `-race` across all modules~~ done - race phase green 3x since 5f2198189
~~29. Run `nix run .#test-integration` to verify integration tests pass~~ done - integration suites green in verify
~~30. Run `nix run .#check-coverage` — verify coverage drift detection passes~~ done - gate repaired at 875bb689b; green since
~~31. Run `nix run .#check-duplication` — verify no new clones introduced~~ done - baseline re-pinned; green since
32. Run `nix run .#vulncheck` — per-module standalone build check <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist)
~~33. Verify `cmd/api-stability` golden file is correct (diff against expected)~~ done - golden current (4133 exports)
34. Test that a consumer can still import `codec/v4` (backward compat verification) <- NOT-DO - codec/ deleted (ADR-0128); consumers advised to import go-codec directly (CHANGELOG advisory)
~~35. Test that a consumer can import `go-codec` directly (new path verification)~~ done - all internal consumers migrated to direct imports

### Cleanup & Polish

36. Update `codec/CHANGELOG.md` to note deprecation and extraction <- NOT-DO - module deleted
37. Update `codec/CONTRIBUTING.md` to point to go-codec <- NOT-DO - module deleted
38. Add `// Deprecated:` comments to every symbol in `codec/alias.go` (per Go convention) <- NOT-DO - alias.go deleted with the module
39. Verify `codec/go.sum` can be generated (run `go mod tidy` in codec/ after publish) <- NOT-DO - module deleted
    ~~40. Check if any `example/` modules need go.sum updates~~ done - example go.sum updated via the mass tidy
    ~~41. Verify `stack/contracttest/` still works (imports codec)~~ done - workspace builds/tests green
    ~~42. Check `integration/` tests pass with new import paths~~ done - green in every verify since
    ~~43. Verify `system/` module tests pass (composition root)~~ done - same
    ~~44. Run `nix fmt` to ensure formatting is correct~~ done - lint clean since 444be10a7

### Strategic

45. Consider whether `go-codec` should depend on `go-error-family` directly or
    create its own error types (currently depends on go-error-family)
46. Consider whether COSE types/functions belong in go-codec or should be
    extracted separately (they're only used by signing/encryption)
    ~~47. Evaluate if the alias module should eventually be deleted entirely (like~~ done - deleted outright at 5127039da (ADR-0128)
    retry/ will be) or maintained indefinitely
47. Add go-codec to the `website-launch` pipeline if public docs are wanted
48. Consider adding `go-codec` to the `go-cqrs-lite` README as an external dependency <- OPEN. README.md does not mention go-codec as external dep yet - minor docs gap
    ~~50. Review whether the `depguard` allow list in `.golangci.yml` needs `go-codec` added~~ done - depguard Main allow list entry landed at 6f9199f0c

---

## g) Questions I Cannot Answer Myself

1. **Should I publish `go-codec` to GitHub now?** The repo is local-only. The
   `go.work` `replace` directive is a workaround, but `go mod tidy` is broken in
   all consumer modules until the repo is published with a `v0.1.0` tag. Should
   I create the GitHub repo and push, or do you want to review first?

2. **Do you want the COSE types (COSESign1, COSEEncrypt0, MarshalCOSESign1,
   etc.) to stay in go-codec, or should they move to a separate module?** They're
   only consumed by `signing/` and `encryption/` — they feel like they belong
   there, not in a generic codec package. But moving them now would be a bigger
   refactor.

3. **Should `codec/` alias module be eventually deleted entirely (like `retry/`
   is marked for deletion) or maintained as a permanent compatibility shim?**
   The retry/ and idempotency/ alias modules are marked DEPRECATED with intent
   to eventually remove. Should codec/ follow the same lifecycle?

---

## Resolution (2026-08-15, docs-health pass)

39 of 50 items carry verdicts. The extraction completed far beyond this
plan's ask: instead of a deprecation alias, the codec/ module was deleted
outright (ADR-0128, `5127039da`), mooting the backward-compat and alias
items (6-7, 22, 34, 36-39). Publication + tidy + gates all green; go-codec
repo scaffolding partially verified (ci.yml, FEATURES, SECURITY per pareto
T18). Untouched: 13/15/16/18/19 (go-codec repo details, external lane),
45-46 (design considerations), 48 (website). Stays active for the README
gap (49) + vulncheck (32).
