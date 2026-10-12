package version

import "slices"

// Data tables for V007 split: the curated v5-removal surface is large, so the
// symbol tables live in v007_tables_symbols.go (pre-v5 shells) and
// v007_tables_symbols_v5.go (ADR-0152 core/v5 twins), concatenated here.

// deprecatedV5Symbol describes one deprecated symbol inside a module that
// otherwise survives v5.
type deprecatedV5Symbol struct {
	fragment    string // module path relative to go-cqrs-lite/
	symbol      string // exported identifier
	replacement string
}

var deprecatedV5Symbols = slices.Concat(deprecatedV5SymbolShells, deprecatedV5SymbolV5Twins) //nolint:gochecknoglobals // static table

var deprecatedV5SymbolShells = []deprecatedV5Symbol{ //nolint:gochecknoglobals // static table
	// ADR-0123: stack root — Bundle + Materialize + projection runner.
	{fragment: "stack", symbol: "Bundle", replacement: "system.New composition"},
	{
		fragment:    "stack",
		symbol:      "New",
		replacement: "system.New with DomainConfig + DeploymentConfig",
	},
	{fragment: "stack", symbol: "Materialize", replacement: "metaengine auto-projection"},
	{fragment: "stack", symbol: "NewMaterialize", replacement: "metaengine auto-projection"},
	{
		fragment:    "stack",
		symbol:      "TombstonePolicy",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "IncludeTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "ExcludeTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	{
		fragment:    "stack",
		symbol:      "OnlyTombstoned",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	// 2026-10-08: issue #36 seam — WithMetaEngine marked Deprecated in-source
	// (v5 removes it with the stack module); the typed companion is canonical.
	{
		fragment:    "stack",
		symbol:      "WithMetaEngine",
		replacement: "stackmeta.WithStore (typed companion in stack/metaengine)",
	},
	// 2026-09-16: marked Deprecated in-source (removal at v5); keep the
	// curated table and the markers in lockstep — V007 is table-driven.
	{fragment: "metaengine", symbol: "On", replacement: "metaengine fold DSL OnRecord"},
	{fragment: "metaengine", symbol: "OnTyped", replacement: "metaengine fold DSL OnRecordTyped"},
	{
		fragment:    "metaengine",
		symbol:      "Infer",
		replacement: "OnRecord/AutoInsert production fold paths",
	},
	{
		fragment:    "metaengine",
		symbol:      "InferFromNamedEvents",
		replacement: "OnRecord/AutoInsert production fold paths",
	},
	// ADR-0123: graph projection tier (GraphDriver/GraphSink survive via graphadapter).
	{
		fragment:    "graph",
		symbol:      "GraphProjection",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "NewGraphProjection",
		replacement: "metaengine auto-projection + graphadapter",
	},
	// ADR-0126: deprecated store/journal shells.
	{
		fragment:    "schema",
		symbol:      "VersionedStore",
		replacement: "schema.UpcastSourceTransform + event.DecorateStore",
	},
	{
		fragment:    "schema",
		symbol:      "NewVersionedStore",
		replacement: "schema.UpcastSourceTransform + event.DecorateStore",
	},
	{
		fragment:    "schema",
		symbol:      "VersionedSeekableJournal",
		replacement: "schema.UpcastSourceTransform + event.DecorateJournal",
	},
	{
		fragment:    "schema",
		symbol:      "NewVersionedSeekableJournal",
		replacement: "schema.UpcastSourceTransform + event.DecorateJournal",
	},
	// ADR-0126: signing middleware shells.
	{
		fragment:    "signing",
		symbol:      "RejectingPublishMiddleware",
		replacement: "event.RejectingPublishMiddleware",
	},
	{
		fragment:    "signing",
		symbol:      "RejectingHandlerMiddleware",
		replacement: "event.RejectingHandlerMiddleware",
	},
	// ADR-0126: encryption error shells.
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotJournal",
		replacement: "event.ErrInnerStoreNotJournal",
	},
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotSeekable",
		replacement: "event.ErrInnerStoreNotSeekable",
	},
	{
		fragment:    "encryption",
		symbol:      "ErrInnerStoreNotBackwards",
		replacement: "event.ErrInnerStoreNotBackwards",
	},
	// ADR-0126: metadata CustomData alias.
	{fragment: "metadata", symbol: "CustomData", replacement: "metadata.Metadata[K]"},
	// ADR-0114: tombstones replaced by domain events.
	{
		fragment:    "event",
		symbol:      "DetectTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "event",
		symbol:      "MarkTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "event",
		symbol:      "MarkRebirth",
		replacement: "domain events for restore (docs/migration/tombstone-to-domain-events.md)",
	},
	{fragment: "event", symbol: "MetadataKeyTombstone", replacement: "domain events for deletion"},
	{fragment: "event", symbol: "MetadataKeyRebirth", replacement: "domain events for restore"},
	{fragment: "event", symbol: "TombstoneMark", replacement: "domain events for deletion"},
	{
		fragment:    "event",
		symbol:      "TombstoneStatus",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneActive",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneTombstoned",
		replacement: "listing.StreamStatus or domain events",
	},
	{
		fragment:    "event",
		symbol:      "TombstoneUndetermined",
		replacement: "listing.StreamStatus or domain events",
	},
	{fragment: "event", symbol: "EnsureCustom", replacement: "event.Metadata.WithCustom"},
	// ADR-0126: metadata EnsureCustom is a method (undetectable via
	// qualifier matching) — tracked in the drift method allowlist instead.
	// ADR-0111: per-module ParseType shims replaced by record.ParseType.
	{
		fragment:    "command",
		symbol:      "ParseType",
		replacement: "record.ParseType(s, ErrEmptyCommandType)",
	},
	{fragment: "event", symbol: "ParseType", replacement: "record.ParseType(s, ErrEmptyEventType)"},
	{fragment: "query", symbol: "ParseType", replacement: "record.ParseType(s, ErrEmptyQueryType)"},
	// 2026-09-13: thin BasicCommand bridge superseded by the full-fidelity
	// persisted-command bridge (command-side depth plan, D2).
	{
		fragment:    "command",
		symbol:      "AsRecord",
		replacement: "command.AsRecordPersisted (payload, stream type, receive stamps)",
	},
}
