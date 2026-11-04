# Open-issues sweep verdict comments (2026-11-04)

Draft source for the per-issue triage comments of the 2026-11-04 open-issues sweep.
One `## <number>` section per issue; body below it is posted verbatim via
`gh issue comment`. Comment-only by default (owner decides closes).

## 84

Already resolved in a52cb26fa — `ReadAllFn`/`ReadFromFn` setters exist with tests (`event/v4/eventtest/fake_store.go`). Can be closed.

## 85

Already resolved in 79c8b1cbf — `scenario.WhenVersioned` + `VersionedDecideFunc` exist with tests (`scenario/dsl.go`). Can be closed.

## 86

Fixed on master: `catalog.Channels(ids ...ChannelID)` MessageOption — both association directions (message→channel and channel→message) now feed the asyncapi export. Tests in `catalog/message_config_features_test.go`.

## 87

Fixed on master: `camelCaseToHuman` keeps initialism runs whole — "External ID Set Payload" instead of "External I D Set Payload" (`catalog/message_config.go`, table tests included).

## 88

Fixed on master: `simple.Builder.AddChannel`, plus `AddDataStore` while at it — the facade resource set is complete now (`catalog/simple/builder.go`).

## 89

Fixed on master: the asyncapi exporter honors catalog-declared channels (both link directions) — address/name/guarantee carried, synthetic fallback only when nothing is declared (`catalog/asyncapi/channels.go`, tests in `channels_test.go`).

## 90

Fixed on master: `applyDisplayNameFallbacks` at `Registry.Build()` derives titles from IDs (`crm.count` → "Crm Count") so no message renders a blank title (`catalog/display_name.go`).

## 91

Fixed on master: `catalog.WithResponseHeader` + `ResponseSpec.Headers` render into the OpenAPI `headers` map — `Retry-After` can be a real header now, not description prose (`catalog/openapi/exporter_rest.go`).

## 92

Fixed on master: 204/304 are exempt from the 2xx body-schema rule (RFC 9110 prohibits bodies there), so a truthful nil schema validates (`catalog/validate.go` + `TestValidate_Response204WithoutSchema_NoViolation`).

## 93

Fixed on master: `catalog.WithTypedParam` takes a full `*Property` (type/format/enum) and the OpenAPI param renders the `enum` too (`catalog/message_config_rest.go`, `catalog/openapi/exporter_rest.go`).

## 94

Fixed on master: `asyncapi.NewExporter` fabricates no kafka/localhost/production server anymore — `servers` is omitted entirely unless `WithServer` says otherwise. Breaking default change, as proposed; docserver output unaffected (it pins its own server).

## 95

Fixed on master: `catalog.WithRequestBodyContent(mediaType, schema)` + `Message.RequestBodyMediaType`/`RequestBodySchema`; JSON stays the default when unset, so multipart bodies are now exportable.

## 96

Fixed on master: `caseutil.ToPascal` splits on `.` too — `commands.wire.hooks.fax` → `postCommandsWireHooksFax` (`catalog/internal/caseutil/convert.go`).

## 97

Fixed on master: `openapi.WithOpenAPIVersion("3.1.0")` swaps the document version string; the default stays 3.0.3 (`catalog/openapi/exporter.go`, test in `features_test.go`).

## 101

Fixed on master: `scorecard.modules` pins in `.cqrs-lint.json` (project-wins merge over detected modules) via `analyzer.ApplyModulePins`; a `false` pin is rejected with "use a waiver". Composition-root projection-host usage scores honestly now.

## 102

Fixed on master: `docserver.Config.SkipIndexRoute` skips both the `GET <prefix>` route and the `{$}` redirect; `Index()` stays mountable for consumers that own their router (`catalog/docserver/docserver.go`).

## 103

Investigated — intended behavior. `StreamMarker.Name()` (3713809cd, 2026-10-03) activates go-branded-id's display prefix in `String()`, and the law test `id/stream_id_laws_test.go:25` pins it; `ParseStreamID`/`MarshalText` stay bare. Persistence must key on `Get()`, never `String()` — documented in `id/doc.go` via #104. Consumers parsing `String()` output are the bug; no change warranted here.

## 104

Fixed on master: `id/doc.go` gained a "Keys and persistence: Get(), never String()" section and `StreamID` carries a KEYING note (`id/stream_id.go`). The keying contract that bit downstream for 26 days is now written down.

## 109

Fixed on master: `--update-baseline` rematches orphaned reasons (identical-reason group, or an unambiguous 1:1 pair) and warns on stderr for every reason it still drops — never silent again (`cmd/cqrs-lint/baseline.go` + 4 new tests).

## 110

Fixed on master: scorecard waivers now suppress their module's adoption findings (`filterWaivedAdoption`, 7 waiver→rule mappings otel/F003 … listing/F016); suppressed findings still surface under `--show-suppressed` with reason+trigger (`cmd/cqrs-lint/filters.go`).

## 111

Fixed on master: `PRAGMA cache_size=-65536` (64 MiB) in both DSN constructors, applied before caller pragmas so last-write-wins overrides still hold (`metaengine/sqliteengine/dsl.go` + `pragma_test.go`).

## 112

Landed as docs: `StreamLogBackend.JournalReadFrom` is now marked Deprecated (advisory — O(afterSeq) OFFSET re-walk, and afterSeq is a position, not a seq) pointing at `SeqSeekableStreamLog.JournalReadFromSeq`; the fallback call sites carry reasoned nolints and `MaxOpenConns(1)` is documented as deliberate with the returned handle as the escape hatch. A typed `WithMaxOpenConns` EngineOption stays open as a follow-up if a second handle-owning consumer shows up.

## 113

Fixed on master: `kv.NewCacheForViewStore` wraps any `ViewStore` (shallow-copy default, `kv.WithCacheCopyFunc` for deep copies), `Has` falls back to Get when the store lacks the capability, and `NewCache` behavior is unchanged (`kv/view_cache.go`). cqrs-htmx can drop its hand-rolled adapter on the next train.

## 114

Verified on master — real gap: `metaengine/cap.go` `EnforceCap` is collection-global (the key extractor addresses rows for deletion, nothing groups), and no per-group variant exists anywhere. Proposal shape looks right. One build note: SQL engines get this via `ROW_NUMBER() OVER (PARTITION BY ...)` (sqlite/turso/duckdb/pg fine, MySQL 8+ only), memory needs a Go-side grouping path — so it lands as an engine-capability split, not one shared core loop. Leaving open as next-wave feature work.
