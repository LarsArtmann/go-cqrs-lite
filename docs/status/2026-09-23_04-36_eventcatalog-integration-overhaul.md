# EventCatalog Integration Overhaul — Status Report

**Date:** 2026-09-23 04:36
**Scope:** `catalog/` module, `catalog/eventcatalog/` exporter, `catalog/cmd/ec-fixture`, `scripts/check-eventcatalog.sh`
**Trigger:** User verdict: "catalog and its eventcatalog.dev integration is a disgrace" → READ/UNDERSTAND/RESEARCH/REFLECT, fix, verify.

---

## Executive Summary

The disgrace was real and is now **fixed and mechanically gated**. I extracted the
ground truth from the pinned `@eventcatalog/core@4.6.3` npm tarball (zod content
schemas + collection loaders) and probed the real `eventcatalog build` with
hand-crafted inputs. The exporter emitted frontmatter that either **hard-failed
the downstream build** (`InvalidContentEntryDataError`, build exit 1) or was
**silently zod-stripped** (data loss) for a wide slice of the public `catalog`
API. The old render gate never caught any of it because its fixture exercised
only 6 of 13 resource kinds and none of the risky fields.

All 16 defect classes are fixed, the fixture now covers every resource kind, the
gate builds two profiles with semantic assertions, and the whole catalog module
is green (tests, race, lint, file-size, md-go, changelog-symbols).

---

## a) FULLY DONE (verified)

### Research (ground truth, not guesswork)

- Downloaded `@eventcatalog/core@4.6.3` tarball; read `src/content.config.ts`
  zod schemas for all 13 collections (events/commands/queries/services/domains/
  channels/containers/flows/teams/users/entities/data-products/agents/customPages),
  `withExtensionProperties` (unknown top-level keys are REJECTED unless `x-`-
  prefixed), `reference()` semantics (version-qualified Astro entry IDs like
  `CreateOrder-1.0.0`), changelog/ubiquitous-language/examples sidecar-file
  collections, and shipped docs guides.
- Live build probes in an npm-installed workdir proved: unknown keys → build
  exit 1; string-typed pointer lists → build exit 1; channel `{id,version}`
  message pointers → `messages.0.collection: Required`; bare channel message
  ids → `Invalid content reference` (links never resolve).

### Fixes (all render-verified against the real build)

1. **Message `channels`**: string list → `{id, version}` channelPointer objects.
2. **Message `labels`** → `x-labels` custom property (was: unknown key, build failure).
3. **Message `responses`** → `x-responses` custom property (was: unknown key, build failure).
4. **Message `changelog`** → `changelog.mdx` sidecar file (was: unknown key);
   generated `eventcatalog.config.js` now sets `changelog: { enabled: true }`
   when changelogs exist (pages are opt-in upstream, default off).
5. **Message examples**: dead `examples.json` (never read by core) →
   `examples/example-N.json` files the example loader actually reads; verified
   rendering in the Schema Explorer ("Usage Examples" tab, `hasRenderedExamples: true`).
6. **Service/agent `flows`, service `entities`**: strings → pointer objects
   (were: `Expected type "object", received "string"` build failures).
7. **Domain `domains`/`data-products`/`entities`/`flows`**: strings → pointers.
8. **Domain `ubiquitousLanguage`**: frontmatter (unknown key) →
   `ubiquitous-language.mdx` dictionary file; verified rendering at
   `/docs/domains/<id>/language/<term>`.
9. **Channel `messages`**: bare pointers → fully qualified
   `{collection, name, id: <id>-<version>, version}`; channel pages previously
   rendered as disconnected stubs with ZERO message links — now list them
   (verified in built HTML).
10. **Team `role`/`avatarUrl`** → `x-role`/`x-avatarUrl` (unknown keys upstream).
11. **Entity `schemas`** dropped (entities have no such field; `schemaPath` kept).
12. **Badges**: `backgroundColor`/`textColor` are REQUIRED — defaulted to
    blue/white when unset (was: `backgroundColor: Required` build failure).
13. **Flow steps**: `dataStore`→`container`, `subFlow`→`flow` keys; actor `url`
    dropped (only `externalSystem` supports it); `channel` steps skipped (no
    such step kind upstream) — all documented in code.
14. **Styles**: flat `nodeColor`/`nodeLabel` (silently stripped) → nested
    `styles.node.{color,label}`; verified in rendered graph data.
15. **Custom docs**: `summary` now always emitted (REQUIRED upstream, was
    conditionally missing → entry dropped); `id` field removed (not in schema).
16. **`BaseConfig.ResourceGroups`**: now fails Export with a clear Rejection
    (EventCatalog needs typed `{id,version,type}` items the string-based
    `catalog.ResourceGroup.Items` cannot express) instead of emitting a
    guaranteed-broken build. Pinned by `TestExporter_Export_ResourceGroupsRejected`.

### Gate hardening (the regression net)

- `cmd/ec-fixture`: now exercises ALL 13 resource kinds + every risky field
  class (labels→x-, changelog, responses→x-, channels on messages + channel
  message pointers, entities/flows on services/domains/agents, ubiquitous
  language, badges with and without colors, all representable flow step kinds,
  deprecation, sidebar/styles/draft/visualiser, data-product contracts, team
  members + x- fields, custom doc). Its doc comment previously LIED ("exercises
  every resource kind") — now it's true.
- `scripts/check-eventcatalog.sh`: builds TWO fixture profiles in one installed
  workdir (default + changelog — agents omitted there due to the upstream bug
  below), fails on `InvalidContentEntryDataError` in addition to exit codes and
  content-reference errors, and asserts semantic artifacts: channel frontmatter
  carries qualified message pointers, the rendered channel page links its
  messages, the changelog page renders, the config enables changelog. Cleans
  generated resource dirs between profiles (exporter is deliberately
  non-destructive).
- Full gate run: **OK** (86 pages built, both profiles, all spot-checks pass).

### Tests/docs/process

- Updated all affected unit tests; added
  `TestExporter_ChannelMessagesFullyQualified` (pins the version-qualified
  channel pointer contract) and `TestExporter_Export_ResourceGroupsRejected`.
- Goldens regenerated module-wide; wrongly-deleted foreign snaps (asyncapi/
  openapi/d2 — known UPDATE_SNAPS trap, see d) restored.
- `catalog/AGENTS.md`: EventCatalog section rewritten with the verified format
  contract, deliberate non-exports, the upstream bug, and gate instructions.
- `catalog/README.md`: resource table corrected (flow step key mapping, team
  x- fields, enterprise custom docs) + new "Format fidelity" section.
- Root `CHANGELOG.md`: detailed Fixed entry.
- Split `exporter.go` (437 → 309 lines) via new `message_index.go` to respect
  the 350-line ratchet (baseline was 382; growth would have failed CI).
- Lint cycle to zero catalog findings (goconst constants in fixture, `new(x)`
  modernize, nlreturn, unused nolint, slices.ContainsFunc).
- Verified green: catalog module tests, `-race`, module lint, file-size gate
  (catalog files), `#check-md-go`, `#check-changelog-symbols`, `#check-eventcatalog`.
- No public API changes (all fixes internal to `eventcatalog` package) → no
  api-stability golden churn.

---

## b) PARTIALLY DONE

- **Changelog rendering under agents**: changelog pages CANNOT be enabled when
  the catalog contains agents — upstream core 4.6.3 crashes rendering
  `/docs/agents/<id>/<version>/changelog` for ANY agent (bare or with
  changelog file; `getBadgeHref` on undefined badge — reproduced and isolated
  empirically). `shouldEnableChangelog` guards it; fixture/gate have a
  changelog profile without agents to still cover the path. Removal of the
  guard is blocked on an upstream fix + pin bump.
- **Custom docs**: exported correctly to `docs/<slug>/index.mdx`, schema-clean,
  but standalone custom pages only ROUTE in EventCatalog enterprise (community
  build collects them, no pages). Documented; nothing more we can do library-side.
- **Root-level companion artifacts** (`llms.txt`, `schemas.txt`, `coeffects.md`
  written to the EC project root): verified harmless (build clean); core
  generates its own llms files under `dist/docs/llm/`. Kept for non-EC hosting.
  Their CONTENT quality was audited only lightly (llms writer covers all 13
  resource kinds).

---

## c) NOT STARTED (known, deliberately deferred)

- Upstream issue filing for the agent-changelog crash (see questions).
- `eventCatalogCoreVersion` bump to a release that fixes said crash.
- `#verify` full-suite run (exclusivity rule: never concurrent with other
  heavy work; see e/next).
- `check-release-scripts --self-test` extension for the rewritten gate script
  (needs offline-fixture design; current script inherently requires npm network).

---

## d) TOTALLY FUCKED UP (honest damage report)

- **Golden-snap deletion incident**: running `UPDATE_SNAPS=true go test ./...`
  module-wide still let the eventcatalog `snaps_clean_test.go` delete the five
  snaps owned by asyncapi/openapi/d2 (the catalog/AGENTS.md advice "always regen
  module-wide" is INSUFFICIENT — within one module-wide run, eventcatalog's
  clean test ran against other packages' snaps). Restored via `git restore`
  immediately; the AGENTS.md guidance should be corrected (see e).
- **Two gate re-runs wasted** on self-inflicted issues: (1) leftover probe
  files (`probe-ch`) from my earlier schema experiments poisoned the first
  full-fixture build; (2) the changelog profile initially crashed because the
  non-destructive exporter left the default profile's `agents/` dir behind —
  my gate design forgot cleanup between profiles. Both fixed; lesson: probes
  belong in throwaway copies, gates must control their workdir state.
- **Mangled minimal-repro configs** (sed truncation, printf confusion) cost a
  few minutes of confusing astro schema errors before I switched to editing
  files in the real workdir directly. Sloppy shell hygiene.

---

## e) WHAT WE SHOULD IMPROVE (observations from this session)

1. **The fixture/gate lied for a long time**: a render gate whose fixture
   doesn't exercise the risky surface is a false green. Rule: when an exporter
   has N input field classes, the render fixture must cover ALL of them — the
   strengthened gate now enforces this for eventcatalog; consider the same
   discipline for asyncapi/openapi/d2 golden coverage.
2. **catalog/AGENTS.md golden-snap guidance is wrong as written** — needs the
   sharper rule: regen module-wide AND verify `git status testdata/` afterwards;
   restore foreign deletions.
3. **Schema contracts should be machine-checked earlier**: the 16 defects
   existed because format knowledge lived in prose. The zod-schema facts are
   now encoded in frontmatter types + comments + the gate, but a
   "frontmatter → schema" unit-level conformance table test would harden it
   further (see next steps).
4. **Auto-commit daemon** absorbed mid-work states (expected, per AGENTS), but
   it also committed an unrelated parallel session's cqrs-lint edits in the
   same commits — attribution is muddy.
5. **Pre-existing repo violations I did NOT touch** (out of scope, will fail
   `#check-file-size` for others): `metaengine/pebbleengine/layout_planner.go`
   (391→392), `metaengine/fold.go` (496→500),
   `cmd/cqrs-lint/pkg/rules/lintutil/lintutil.go` (453→470, parallel session).

---

## f) NEXT — up to 50 candidates (roughly Pareto-ordered)

**Close out this work**

1. Re-run `nix run .#lint` (last edit removed an unused nolint — confirm zero
   catalog findings).
2. Re-run `nix run .#check-eventcatalog` once more after final lint fixes
   (changelogBody refactor + fixture constants changed output paths' inputs —
   semantics identical, but verify).
3. Run `nix run .#verify` (exclusive window) for the full-suite stamp.
4. Run `nix run .#verify-ci` (GOWORK=off per-module matrix) — mirrors CI.
5. Check `git status` for uncommitted stragglers; author a proper commit if the
   daemon hasn't (message per repo conventions, NOT "chore:").
6. `benchkit/go.mod` was dirty at session start (not mine) — decide owner.

**Harden the exporter**
7. Unit-level conformance test: table mapping every `messageFM`/`serviceFM`/...
field to its upstream zod constraint (name + shape), so field additions
without a fixture update fail locally.
8. Dangling-reference validation: warn/skip when channel `messages`,
`writesTo`/`readsFrom`, domain `services`, flow step refs point at IDs not
in the catalog (currently silent skips or broken links).
9. Emit `sidebar.position`/`order` support for custom docs (docusaurus-style
ordering) if consumers need deterministic doc order.
10. Consider exporting `message.channels[].parameters` (channelPointer
supports `parameters` record; we only emit id/version).
11. Consider `versioned/` directory output for multi-version messages
(EventCatalog's versioning story; catalog currently single-version).
12. Ubiquitous-language dictionary `icon`/`summary` terms (schema supports).
13. Data-product `hidden` etc. verified; consider `outputs[].contract`
file copying warning when the contract path doesn't exist in output.
14. `writeExamples`: honor `examples.config.yaml` metadata (title/summary/usage
per example) if catalog ever models it.
15. `coeffects.md`/`schemas.txt`: move under `docs/` (llms inclusion) or
document why root placement is intentional.

**Upstream / ecosystem**
16. File upstream issue for the agent-changelog crash (minimal repro in hand:
any agent + `changelog: {enabled: true}` → TypeError in getBadgeHref).
17. Re-check @eventcatalog/core releases for the fix; bump pin + drop the
`shouldEnableChangelog` guard + simplify the gate to one profile.
18. Upstream: ask/docs whether custom-pages routing is planned for community.
19. Verify the `eventcatalog.config.js` JSDoc type path
(`@eventcatalog/core/bin/eventcatalog.config`) actually resolves in 4.6.3
(harmless if not, but sloppy).

**Repo hygiene**
20. Fix the 3 pre-existing file-size ratchet violations (other modules).
21. Correct catalog/AGENTS.md golden-snap regeneration guidance (see e/2).
22. Add `--self-test` legs to check-eventcatalog.sh where feasible (offline
assertions on the spot-check functions via fixture files).
23. Consider a `catalog/conformance/` package sharing fixture builders between
ec-fixture and unit tests (currently duplicated literals).
24. Sweep `docserver` eventcatalog view for the same class of format drift
(it renders in-process, NOT from the MDX — different code path, unverified
this session against the new contract).
25. README quick-start example uses `reg.Build()` without error handling for
`Export` — cosmetic.

**Bigger catalog-module ideas (not started, optional)**
26. AsyncAPI exporter: same audit treatment vs AsyncAPI 3.0 spec.
27. OpenAPI exporter: same audit treatment vs OpenAPI 3.0 spec.
28. D2 exporter: same treatment vs d2lang semantics.
29. `catalog.Validate()`: add dangling-reference + EC-compat rules.
30. Typed builders for the x- custom properties (escape hatch with structure).

(31–50: reserve for follow-ups discovered during #verify / upstream replies.)

---

## Verification Evidence (this session)

| Gate                                                                           | Result                                        |
| ------------------------------------------------------------------------------ | --------------------------------------------- |
| `go test ./...` (catalog module)                                               | ok (16 pkgs)                                  |
| `go test -race ./...` (catalog module)                                         | ok                                            |
| `nix run .#lint` (catalog scope)                                               | zero findings after final cycle (last         |
| edit pending re-confirm, see f/1)                                              |                                               |
| `nix run .#check-file-size`                                                    | catalog clean (3 unrelated violations remain) |
| `nix run .#check-md-go`                                                        | 1461 blocks valid, no new errors              |
| `bash scripts/check-changelog-symbols.sh`                                      | 25 citations honest                           |
| `bash scripts/check-eventcatalog.sh`                                           | OK — both profiles, semantic checks           |
| Rendered-HTML spot checks                                                      | channel→message links, changelog page,        |
| ubiquitous language, examples tab, styles.node, defaulted badges — all present |                                               |

## g) Questions I cannot answer myself

1. **Upstream issue**: want me to file the @eventcatalog/core agent-changelog
   crash upstream (I have a minimal repro)? If yes: file it yourself, or should
   I draft it for your review first (per verify-before-filing/github-voice)?
2. **Semantics call**: `x-labels`/`x-responses`/`x-role`/`x-avatarUrl` preserve
   data EventCatalog has no field for. Alternative policy: DROP them entirely
   for a minimal schema surface. Keep the x- mapping (current) or drop?
3. **Scope**: item f/24 — should the `docserver` in-process EventCatalog view
   be audited against the same verified contract in a follow-up session, or is
   the MDX exporter the only surface you care about right now?

— Report ends. Waiting for instructions.
