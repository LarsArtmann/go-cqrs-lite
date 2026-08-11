# Status: codec/ Extraction into go-codec — 2026-08-11

> Extraction of `codec/` into a standalone `go-codec` repo. Functional but
> with known gaps and one correctness issue.

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
1. Publish `go-codec` to GitHub (`github.com/larsartmann/go-codec`)
2. Tag `go-codec` with `v0.1.0`
3. Remove `replace` directive from `go.work`, add `../go-codec` to `use` block
4. Run `go mod tidy` in all 53 consumer modules to get proper go.sum entries
5. Verify `go mod tidy` works in consumer modules after publish
6. Delete `codec/testdata/` (dead directory)
7. Check and clean up `codec/reports/` if stale
8. Run `nix run .#verify` (or at minimum `#verify-fast`)
9. Run `nix run .#lint` to check for lint issues across all modules
10. Run `nix run .#check-arch` to verify dependency budgets still pass

### go-codec project setup
11. Add `.golangci.yml` to go-codec (copy from go-retry pattern)
12. Add `.github/workflows/ci.yml` to go-codec
13. Add `.github/dependabot.yml` to go-codec
14. Write `FEATURES.md` for go-codec
15. Write `ROADMAP.md` for go-codec
16. Write `TODO_LIST.md` for go-codec
17. Write `SECURITY.md` for go-codec
18. Verify go-codec passes `golangci-lint run` standalone
19. Add go-codec to the go-codec README: detailed usage sections (CBOR tags, streaming, etc.)

### Documentation
20. Write ADR for codec extraction (next ADR number after latest)
21. Update `docs/DOMAIN_LANGUAGE.md` — change codec import path
22. Update `flake.nix` — add comment noting codec/ is deprecated alias
23. Update `.agents/skills/go-cqrs-lite/references/modules.md` — annotate codec as deprecated
24. Update SKILL.md if it references codec module map entry
25. Scan all `.agents/skills/go-cqrs-lite/references/*.md` for codec usage examples
26. Update `cmd/cqrs-gen/main.go` template if it generates codec imports in consumer code
27. Verify `cmd/cqrs-lint` architecture rules still recognize go-codec as Tier 0

### Testing & Verification
28. Run full test suite with `-race` across all modules
29. Run `nix run .#test-integration` to verify integration tests pass
30. Run `nix run .#check-coverage` — verify coverage drift detection passes
31. Run `nix run .#check-duplication` — verify no new clones introduced
32. Run `nix run .#vulncheck` — per-module standalone build check
33. Verify `cmd/api-stability` golden file is correct (diff against expected)
34. Test that a consumer can still import `codec/v4` (backward compat verification)
35. Test that a consumer can import `go-codec` directly (new path verification)

### Cleanup & Polish
36. Update `codec/CHANGELOG.md` to note deprecation and extraction
37. Update `codec/CONTRIBUTING.md` to point to go-codec
38. Add `// Deprecated:` comments to every symbol in `codec/alias.go` (per Go convention)
39. Verify `codec/go.sum` can be generated (run `go mod tidy` in codec/ after publish)
40. Check if any `example/` modules need go.sum updates
41. Verify `stack/contracttest/` still works (imports codec)
42. Check `integration/` tests pass with new import paths
43. Verify `system/` module tests pass (composition root)
44. Run `nix fmt` to ensure formatting is correct

### Strategic
45. Consider whether `go-codec` should depend on `go-error-family` directly or
    create its own error types (currently depends on go-error-family)
46. Consider whether COSE types/functions belong in go-codec or should be
    extracted separately (they're only used by signing/encryption)
47. Evaluate if the alias module should eventually be deleted entirely (like
    retry/ will be) or maintained indefinitely
48. Add go-codec to the `website-launch` pipeline if public docs are wanted
49. Consider adding `go-codec` to the `go-cqrs-lite` README as an external dependency
50. Review whether the `depguard` allow list in `.golangci.yml` needs `go-codec` added

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
