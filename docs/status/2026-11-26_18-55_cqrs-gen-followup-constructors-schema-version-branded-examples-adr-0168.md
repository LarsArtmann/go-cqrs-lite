# Status: cqrs-gen follow-up: constructors, schema version, branded examples, ADR-0168

**Date:** 2026-11-26 18:55 CET (Thursday)
**Scope of session:** the continuation of the 05:50 report
(`2026-11-26_05-50_cqrs-gen-rework-composition-root-codegen-uncommitted.md`): every open item f1 to
f40 and every question g1 to g3 of that report.
**Head at report time:** `cf93d419f`.
**Commit state:** this continuation's edits are UNCOMMITTED in the working tree. The session does
not commit without an explicit instruction. The 05:50 work landed through another session's
commits (`0b4d58409`, `87efe5539`, `d0a91edaa`), which swept finished pieces of this session as
they went; the same session still shares the tree with in-flight work of its own (irohengine,
sqliteengine), which is why the whole-tree gates below are not green as a whole.

## Scope line: what was verified, and what was not

**Verified here** (18:00 to 18:55 CET, load average 17 to 40, working tree as described above):

| Check                                                                                                                                                                                                                        | Result                                                                                                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cd cmd/cqrs-gen/v4 && GOWORK=off go test -race -count=1 -cover ./...`                                                                                                                                                       | `ok`, coverage 94.0% (floor 90.3)                                                                                                                                                           |
| `cd cmd/cqrs-lint/v4 && GOWORK=off go test -count=1 ./...`                                                                                                                                                                   | `ok` in all 21 packages, including the taskmanager golden and the adoption package                                                                                                          |
| `example/goal-shaped-app`, `example/metaengine-quickstart`: `go vet` and `go test -race`, GOWORK on and off                                                                                                                  | `ok` in all four combinations                                                                                                                                                               |
| `PATH=<built binary> go generate ./...` in `example/goal-shaped-app`                                                                                                                                                         | `unchanged cqrs_gen.go (8 markers)`, exit 0                                                                                                                                                 |
| `GT_LINT_MODULES="cmd/cqrs-gen/v4 cmd/cqrs-lint/v4 catalog/v4 example/goal-shaped-app example/metaengine-quickstart" nix run .#lint-scope`                                                                                   | 5 of 5 modules clean                                                                                                                                                                        |
| cqrs-lint over `example/goal-shaped-app` with the current binary                                                                                                                                                             | no finding on `cqrs_gen.go`; B005, E009, F007, F031 remain on hand-written code (pre-existing coaching)                                                                                     |
| erraudit counts for the five touched modules                                                                                                                                                                                 | `cmd/cqrs-gen/v4` 25, `cmd/cqrs-lint/v4` 101, `catalog/v4` 20, `example/metaengine-quickstart` 16 (all equal to the pinned row), `example/goal-shaped-app` 13 (down from 16, row re-pinned) |
| `bash scripts/check-generated-artifacts.sh --self-test`                                                                                                                                                                      | `5 fault classes verified, all green` (new: the cqrs-gen leg against a temp copy)                                                                                                           |
| `bash scripts/check-adr-numbering.sh`, `check-changelog-symbols.sh`, `check-file-size.sh`, `check-package-surface.sh`, `check-test-sleeps.sh`, `check-test-skips.sh`, `check-module-layers.sh`, `check-readme-deprecated.sh` | all green                                                                                                                                                                                   |
| `bash scripts/check-md-go.sh`                                                                                                                                                                                                | `All 1537 code blocks valid (100 skipped)`, no new errors                                                                                                                                   |
| `cmd/doc-check` over SKILL.md, the skill references, AGENTS.md, and the touched READMEs and ADR                                                                                                                              | `All 1404 references valid` and `All 173 references valid`                                                                                                                                  |
| `cd cmd/api-stability/v4 && GOWORK=off go run .`                                                                                                                                                                             | `API surface OK: 6544 exports verified`, no golden diff                                                                                                                                     |

**Not verified here:**

- `nix run .#verify`: not run. It needs a quiet box and an exclusive window, and the tree carries
  another session's in-flight modules.
- `#check-duplication`, `#check-test-runtime`, `#check-coverage`, `#check-gosec`: not run.
- `bash scripts/check-readme-links.sh` reports two targets "on disk but not in git": the ADR-0168
  file is untracked. This resolves with the first commit that adds it.
- `bash scripts/check-canonical-facts.sh` reports two drifts that are not this session's: the live
  status index is missing two reports, and AGENTS.md cites Go 1.27.2 while every directive is 1.27
  (the other session's minor-form sweep). `check-untested-exports.sh` reports
  `metaengine/v4/func TypeName`, also not this session's.
- The full erraudit sweep still prints GREW rows for `metaengine/irohengine/*` and
  `metaengine/sqliteengine/v4`: the other session's in-flight modules.

## a) What changed since the 05:50 report

1. **Generator (`cmd/cqrs-gen/v4`).**
   - Typed constructors: `NewX(streamID, payload, opts...)` for a command and `NewX(payload, opts...)`
     for a query that embed the library envelope; a package that already declares `NewX` keeps its
     own. `resolve.go` took the symbol table and resolution out of `parse.go` (262 and 157 lines).
   - Every generated constructor forwards the library's options; nothing a caller could pass to
     `command.New`, `query.New` or `event.New` is lost.
   - An event marker carries the payload's schema version (`//cqrs:event task.updated v2`). The
     constructor stamps it through `event.WithSchemaVersion` (version 1 when the marker names
     none), and a bumped version becomes the catalog message version.
   - Constructors return the library's error unchanged: it already names the wire name and carries
     its error family, so a `fmt.Errorf` wrap only added context twice and a finding under the
     error-policy gate.
   - `readme_test.go` pins the README's output fence to the generator's real output;
     `constructor_test.go` and `schema_test.go` pin the new shapes; `compile_test.go` calls every
     generated signature, including a versioned event.
2. **cqrs-lint.** B004, B009 and B010 name the marker to add and the declaration that follows.
   B009 and B010 skip files with the generated-code header. B032 does not judge a constructor that
   forwards an option list. F005 counts only a schema version beyond the first: stamping version 1
   is what D013 asks for, and the two rules used to follow each other like a trap.
3. **Examples.** `example/goal-shaped-app` builds every command and query through the generated
   constructors and brands its task identity end to end (`id.StreamID` in payloads, read model and
   query). `example/metaengine-quickstart` does the same for its task map; its two Record-stamped
   columns carry cited `cqrs-lint:ignore(A032)` rows.
4. **Gates.** `scripts/check-generated-artifacts.sh` gained the `cqrs-gen-output` leg and a
   mutation check for it. `scripts/erraudit-baseline.txt` re-pins one row (`example/goal-shaped-app`
   16 to 13) and its header total.
5. **Docs.** ADR-0168 (accepted), CHANGELOG, FEATURES, the skill references (`advanced.md` section
   6.7, rows in `core.md` and `modules.md`), `docs/DOMAIN_LANGUAGE.md`, `docs/agents/module-map.md`,
   AGENTS.md, a gotcha section in `docs/agents/gotchas-tooling-build.md`, the 2026-11-22
   utilization review (risk 3 and recommendation 3 marked done), and TODO_LIST (decision D42 plus a
   harvest section).

## b) The three questions of the 05:50 report

1. **Commits in a shared tree.** Not answered by the owner; not needed any more. Another session
   committed the 05:50 work. This continuation stays uncommitted until the owner says "commit".
2. **Payload ID policy.** Decided as a rule with two halves (ADR-0168, decision 7): inside one
   context, identifiers are branded end to end, because on the wire they are bare strings and the
   catalog now renders them as strings; a payload that crosses a context or process boundary stays
   a plain-string contract, each side branding at its own boundary. `goal-shaped-app` and
   `metaengine-quickstart` follow the first half, `mesh-demo` and `taskmanager` the second. Whether
   that split stands is decision D42 on the owner sheet.
3. **B004 and B009.** Constructors are generated now, so B004's promise is true for a command that
   embeds the envelope. Emit functions are not generated: a publish helper's sink and policy are the
   application's (bus, store with an expected version, batch), so nothing there is derivable. B009
   points at the typed event constructor instead.

## c) The 05:50 next steps, item by item

| Item                                   | State             | Where                                                                                                                               |
| -------------------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| f1                                     | done              | `cmd/cqrs-gen/v4/resolve.go`                                                                                                        |
| f2                                     | done              | `063b10cd7`; re-verified here: `API surface OK`                                                                                     |
| f3                                     | landed otherwise  | `0b4d58409`, `87efe5539`; this continuation is uncommitted                                                                          |
| f4, f5                                 | done              | B009 and B010 skip generated files; the dogfood lints clean on `cqrs_gen.go`                                                        |
| f6, f7                                 | partly            | cheap gates green (table above); `#verify`, `#check-duplication`, `#check-test-runtime`, `#check-coverage` not run                  |
| f8, f9                                 | done              | CHANGELOG, FEATURES                                                                                                                 |
| f10                                    | partly            | `advanced.md` 6.7 and the `core.md` and `modules.md` rows are rewritten; the `core.md` quickstart does not mention generated wiring |
| f11                                    | done              | ADR-0168                                                                                                                            |
| f12                                    | partly            | two examples flipped; two stay as contracts by decision (D42)                                                                       |
| f13                                    | done              | `go generate ./...` with the built binary on PATH: unchanged                                                                        |
| f14                                    | done              | TODO_LIST harvest and D42; both reports indexed in `docs/status/README.md`                                                          |
| f15                                    | done              | `docs/DOMAIN_LANGUAGE.md`                                                                                                           |
| f16, f27                               | done              | gotcha section in `docs/agents/gotchas-tooling-build.md`, including the Go 1.27 spread-argument trap and the `nix fmt` path hazard  |
| f17                                    | done              | rule texts and the taskmanager golden; RULES.md unchanged (descriptions did not move)                                               |
| f18                                    | done              | `cqrs-gen-output` leg with `--self-test`                                                                                            |
| f20                                    | done              | both examples under `-race`, GOWORK on and off                                                                                      |
| f25                                    | done              | `TestReadme_ShowsRealOutput`                                                                                                        |
| f30                                    | done              | utilization review annotated                                                                                                        |
| f38                                    | done              | scratch directories under `/tmp` trashed                                                                                            |
| f19                                    | left to the owner | the `go-branded-id` v0.7.1 to v0.8.0 sweep across 93 modules is a dependency bump under the SEC dep-bump policy (D39)               |
| f21 to f24, f28, f36, f37, f40         | parked            | TODO_LIST harvest rows                                                                                                              |
| f22, f23, f24                          | parked            | see the harvest rows                                                                                                                |
| f26, f29, f31, f32, f33, f34, f35, f39 | parked            | no row; none is needed before the next release wave                                                                                 |

## d) Found on the way, fixed at the root

- **A lint trap between two rules.** D013 asks for `event.WithSchemaVersion(1)`, and F005 then
  reported "uses WithSchemaVersion but has no upcaster". F005 now counts only a version beyond the
  first.
- **Go 1.27 changed the AST for a spread argument.** It is no longer an `*ast.Ellipsis` in
  `CallExpr.Args`; the position lives in `CallExpr.Ellipsis`. A test written against the old shape
  passed for the wrong reason until the probe below settled it.
- **The README's output fence had drifted from the generator twice in one day** before it was
  pinned by a test.
- **erraudit is shrink-only.** Generated code that re-wraps errors would have grown every
  consumer's count by one per message.

## e) Next

1. Commit this continuation (the owner's call): 40 files, listed by `git status --short`.
2. `nix run .#verify` on a quiet box, then `#check-duplication`, `#check-test-runtime`,
   `#check-coverage`.
3. Answer D42 on the owner sheet.
4. Tag `catalog` with the branded-ID schema fix at the next release wave; `cmd/cqrs-gen/v4` is an
   untagged train and needs no tag.
