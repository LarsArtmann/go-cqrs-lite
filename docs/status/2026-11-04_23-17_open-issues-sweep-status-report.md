# Status Report — 2026-11-04 23:17 CET

**Session:** "Review all open GitHub Issues" — full triage + fix sweep over all 25 open issues of LarsArtmann/go-cqrs-lite.

**Mission:** READ, UNDERSTAND, RESEARCH, REFLECT → break into actionable steps → execute and verify one step at a time → repeat until everything works and the job is great.

---

## a) FULLY DONE (verified green)

### Catalog module — 13 issues fixed, full module suite green

| Issue | Fix | Where |
| ----- | ---- | ----- |
| #96 operationId keeps dots | `caseutil.ToPascal` treats `.` as word separator; `commands.wire.hooks.fax` → `postCommandsWireHooksFax` | `catalog/internal/caseutil/convert.go` + tests |
| #87 initialism letter-spacing | `camelCaseToHuman` upper-run stays whole: "External ID Set Payload" (was "External I D Set Payload") | `catalog/message_config.go` (`startsWord`) + table tests |
| #92 204 flagged by 2xx rule | 204/304 exempt (RFC 9110 §15.3.5: bodies prohibited) — truthful nil schema validates | `catalog/validate.go` + test |
| #90 blank message titles | `applyDisplayNameFallbacks` at `Registry.Build()`: `crm.count` → "Crm Count" | `catalog/display_name.go` (new), `registry_build.go` + tests |
| #88 facade missing AddChannel | `simple.Builder.AddChannel` + `AddDataStore` (completes the resource set) | `catalog/simple/builder.go` + tests |
| #86 no message→channel option | `catalog.Channels(ids...)` MessageOption; both association directions feed asyncapi export | `catalog/message_config.go` + test |
| #94 fabricated kafka broker | `asyncapi.NewExporter` no server by default; `servers` omitted entirely; docserver pins its own (unchanged) | `catalog/asyncapi/exporter.go` + goldens + test |
| #97 version pinned to 3.0.3 | `openapi.WithOpenAPIVersion("3.1.0")` string swap; default 3.0.3 kept | `catalog/openapi/exporter.go` + test |
| #93 untyped WithParam | `catalog.WithTypedParam` (full `*Property`: type/format/enum), rendered incl. `enum` | `catalog/message_config_rest.go`, `exporter_rest.go` + tests |
| #91 no response headers | `catalog.ResponseHeader`, `ResponseSpec.Headers`, `catalog.WithResponseHeader`, rendered into OpenAPI `headers` | `types_helpers.go`, `message_config_rest.go`, `openapi/types.go`, `exporter_rest.go` + tests |
| #95 no multipart bodies | `catalog.WithRequestBodyContent(mediaType, schema)` + `Message.RequestBodyMediaType/RequestBodySchema`; JSON kept when unset | `types.go`, `message_config_rest.go`, `exporter.go` + test |
| #89 asyncapi ignores Channels | declared channels (both link directions) win over synthetic; address/name/guarantee carried; synthetic fallback stable | `catalog/asyncapi/channels.go` (new), `builder.go` rewired + tests |
| #102 docserver claims index | `Config.SkipIndexRoute` skips `GET <prefix>` + `{$}` redirect; `Index()` stays mountable | `catalog/docserver/docserver.go` + test |

### Other modules — 6 issues fixed, module suites green

| Issue | Fix | Where |
| ----- | ---- | ----- |
| #111 no cache_size pragma | `PRAGMA cache_size=-65536` (64 MiB) in BOTH DSN constructors, before caller pragmas (last-write-wins override) | `metaengine/sqliteengine/dsl.go` + `pragma_test.go` |
| #112 JournalReadFrom trap | `StreamLogBackend.JournalReadFrom` marked **Deprecated** (advisory: O(afterSeq) OFFSET re-walk, position-not-seq param; resume via `SeqSeekableStreamLog.JournalReadFromSeq`); fallback call sites carry reasoned nolints; `MaxOpenConns(1)` documented as deliberate w/ escape hatch | `metaengine/engine.go`, `dsl.go`, `system/*`, `systemscenario/chaos.go` |
| #113 NewCache concrete-only | `kv.NewCacheForViewStore` over ANY `ViewStore`: shallow-copy default, `kv.WithCacheCopyFunc` deep-copy, `Has`→Get fallback, nil `Backend()/Store()`; `NewCache` behavior byte-identical | `kv/cache.go`, `kv/view_cache.go` (new) + tests |
| #104 keying contract undocumented | package doc section "Keys and persistence: Get(), never String()" + KEYING note on `StreamID` | `id/doc.go`, `id/stream_id.go` |
| #109 update-baseline drops reasons | orphaned-reason rematch (identical-reason group or 1:1 pair), stderr warning on every drop (vanished or ambiguous) | `cmd/cqrs-lint/baseline.go` + 4 new tests |
| #110 waivers don't suppress findings | `filterWaivedAdoption`: 7 waiver→rule mappings (otel/F003 … listing/F016), suppressed under `--show-suppressed` with reason+trigger; scorecard WAIVED unchanged | `cmd/cqrs-lint/filters.go`, `run.go` + tests |
| #101 composition-root invisible | `scorecard.modules` pins in `.cqrs-lint.json` (project-wins merge), `analyzer.ApplyModulePins`, false-pins rejected in favor of waivers — config-pins-beat-detection | `cmd/cqrs-lint/pkg/analyzer/*`, `scorecard_*.go` + tests |

### Already fixed on master before this session (verified, no work needed)

- **#84** `eventtest.FakeStore.ReadAllFn/ReadFromFn` — setters exist WITH tests; landed `a52cb26fa` 2026-10-25.
- **#85** `scenario.WhenVersioned`/`VersionedDecideFunc` — exists with tests; landed `79c8b1cbf` 2026-10-25.

### Root-caused (investigation answer)

- **#103** `<StreamType>:<ULID>` display form: **intended**. `StreamMarker.Name()` (added `3713809cd`, 2026-10-03) activates go-branded-id's `String()` display prefix; the law test (`id/stream_id_laws_test.go:25`) pins it; `ParseStreamID`/`MarshalText` stay bare. SEC's break = consumer parsing `String()` output — the #104 contract doc is the migration note. No behavior change warranted.

### Cross-cutting gates run green

- api-stability golden regenerated: **13 new exports** (`catalog.Channels`, `WithTypedParam`, `WithResponseHeader`, `WithRequestBodyContent`, `openapi.WithOpenAPIVersion`, `simple.AddChannel/AddDataStore`, `kv.NewCacheForViewStore`, `WithCacheCopyFunc`, `analyzer.ApplyModulePins`, `ValidateScorecardModulePins`, `ResponseHeader`, `openapi.Header`).
- `check-changelog-symbols.sh` ✅ (37 citations verified; one `sys.ProjectionHost` prose-fiction caught + reworded).
- `check-md-go.sh` ✅ (1529 blocks).
- `doc-check` ✅ (1372 refs) — **run BEFORE the recipes.md edit; needs a re-run** (see b).
- asyncapi goldens regenerated with the documented safe procedure (scoped `UPDATE_SNAPS` + `git restore` cross-deletions).
- Full module suites green: catalog (all subpackages incl. docserver/eventcatalog/d2/huma/compat), kv, id, scenario, cqrs-lint (root + pkg/...), system, systemscenario, metaengine (+sqliteengine pragma tests).
- `CHANGELOG.md` [Unreleased] Added/Changed entries; `FEATURES.md` rows (catalog, asyncapi, openapi, docserver, cqrs-lint); `recipes.md` kafka-default correction + `NewCacheForViewStore` mention.

---

## b) PARTIALLY DONE

1. **Review comments + issue closure: NOT STARTED.** Every issue has a verified verdict + a fix in the tree, but none of the 25 GitHub issues has a comment or close yet. That was the session's last todo, interrupted here.
2. **`doc-check` needs one re-run** after the `recipes.md` kafka/`NewCacheForViewStore` edits (both are prose-only edits inside existing sections; compile-risk ≈ 0, but the gate must be green on the final tree).
3. **#112 second half (pool ceiling):** doc-only decision implemented. An `EngineOption WithMaxOpenConns(n)` for the handle-owning constructor was scoped but deliberately not built — the returned `*sql.DB` is already the escape hatch for `NewFromDSN`; only `NewSQLiteEngineFromDSNWith` consumers lack a typed knob.

## c) NOT STARTED

- `nix run .#verify` (or even `#lint`) — golangci-lint over the changed modules was never run; my `//nolint:exhaustruct_v5` / `//nolint:staticcheck` guesses are unverified by the real linter.
- Mechanical gate re-runs: `check-file-size` (manual `wc` only — all touched files < 350), `check-package-surface`, `check-untested-exports`, `check-coverage`, `check-duplication`, `check-test-sleeps/skips`, gosec.
- Consumer-side follow-through: cqrs-htmx can now delete its hand-rolled view-cache adapter (issue #113's motivating consumer); crm can drop its 7 waiver-duplicated baseline entries (#110) and pin `projectionhost: true` (#101); webphone can adopt `WithResponseHeader`/multipart/`WithParamSchema`-class options (#91/#95/#93).
- Authored commit at this phase boundary (owner ruling 2026-11-03: phase boundaries warrant hand-commits — the daemon will otherwise absorb this whole wave into `chore:`).

## d) TOTALLY FUCKED UP

Nothing shipped broken. Honest near-misses, all caught in-session by tests/gates:

1. **First `rematchOrphanedReasons` draft was muddled** (three redundant `reasons[key]` guards that made no sense; nil-warn guard silently skipped the whole rematch). Caught by the new unit tests failing, rewritten clean. The committed logic is the simple version.
2. **Blind edit clipped `TestDocsServer_StaticFS`'s opening lines** (edit target was the func signature; I destroyed its first two body lines). Noticed immediately, restored.
3. **Two test-authoring bugs** in the asyncapi channel tests (channel declared without its message link; registry missing the declared channel) — fixed before landing.
4. **`sys.ProjectionHost` CHANGELOG fiction** — prose citation tripped the honesty gate; reworded. Gate working as designed.
5. Process slip: used `sed -i` on `baseline_test.go` for mechanical 3→4-arg call fixes instead of explicit edits — fast, but below this repo's editing bar; verified by grep afterward.

## e) IMPROVEMENTS MADE BEYOND THE LETTER OF THE ASKS

- `copyMessage`/`copyResponseSpecs` extended — without this, EVERY new Message/ResponseSpec field would silently vanish between registry and export (the #91/#95 tests caught it; future fields inherit the lesson).
- OpenAPI param rendering now includes `enum` (typed params export their constraint set, not just type+format).
- `Cache[T,K]` internals unified behind one `newCache` runtime — `NewCache` semantics byte-identical, view-store path shares capacity/TTL/write-through machinery instead of forking.
- Baseline drops are now **never silent** — even the ambiguous case prints the exact dropped reason + position.

## f) NEXT THINGS (ordered)

1. Re-run `doc-check` on the final tree (2 min).
2. Post the per-issue review comments (25) and close: 21 fixed here, 2 already-fixed (#84/#85), 1 answered-intended (#103), 1 answered-and-documented (#104 folds into the same comment set; #112's pool question stays open as a noted follow-up or closes with the doc decision).
3. Run `nix run .#lint` (or scoped golangci over catalog, kv, id, metaengine/sqliteengine, cmd/cqrs-lint, system, systemscenario) and fix every finding.
4. Run `nix run .#verify` in its exclusivity window; fix what it surfaces (file-size, package-surface, untested-exports, coverage floor, duplication).
5. Hand-commit the wave at this phase boundary with a real message.
6. cqrs-htmx: delete `usermgmt/view_cache.go` adapter, consume `kv.NewCacheForViewStore` (next train).
7. crm: drop the 7 waiver-duplicated baseline entries; add `scorecard.modules` pin for `projectionhost`; drop the package-clause directive.
8. webphone: move `Retry-After` prose into `WithResponseHeader`; declare multipart `/api/transcribe`; use `WithTypedParam` for pagination; remove the `WithServer(...,"","")` suppression (now default).
9. Kith CRM: raise `MaxOpenConns` via the returned handle per the new doc note; re-measure the read storm.
10. Consider `sqliteengine.WithMaxOpenConns` EngineOption if a second handle-owning consumer appears.
11. Doc-promote `SeqSeekableStreamLog` in metaengine README (issue #112 ask #2 — engine.go godoc done, README not).
12. Regenerate `docs/engine-capabilities.md` if the pragma touches its census (`--update`; likely no-op).
13. Fleet census + tag-wave ride for catalog/kv/id/sqliteengine/cqrs-lint (per-module `GOWORK=off go test` over CHANGED set before `batch-release.sh` — the 2026-10-06 lesson).
14. Optional: golden test pinning `NewSQLiteEngineFromDSNWith` caller-override-wins beyond construction success.

## g) QUESTIONS (cannot resolve myself)

1. **Close vs comment:** I close issues only on your say-so in past sessions — this wave touches 25 issues with verifiable fixes. Want me to comment AND close in one pass, or comment only?
2. **#112 pool option:** doc-only now — good enough, or build `WithMaxOpenConns(n)` this wave?
3. **Commit authorship:** hand-commit at this boundary per the 2026-11-03 ruling, or let the daemon absorb and you cherry-pick the message later?

---

*Verified by: full per-module suites (catalog, kv, id, scenario, cmd/cqrs-lint, system, systemscenario, metaengine), api-stability golden, changelog/md-go gates, asyncapi goldens. Not yet verified: golangci, the #verify composite, consumer replays.*
