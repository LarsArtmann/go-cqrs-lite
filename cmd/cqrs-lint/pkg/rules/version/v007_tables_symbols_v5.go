package version

// V007 table continuation: ADR-0152 core/v5 copy-forward twins of the v4
// deprecated surface (identical semantics; removed with the v4 train in T26).

var deprecatedV5SymbolV5Twins = []deprecatedV5Symbol{ //nolint:gochecknoglobals // static table
	// ADR-0152: core/v5 copy-forward twins of the v4 deprecated surface
	// (identical semantics; removed together with the v4 train in T26).
	{
		fragment:    "core/command",
		symbol:      "AsRecord",
		replacement: "command.AsRecordPersisted (payload, stream type, receive stamps)",
	},
	{
		fragment:    "core/command",
		symbol:      "ParseType",
		replacement: "record.ParseType(s, ErrEmptyCommandType)",
	},
	{
		fragment:    "core/event",
		symbol:      "ParseType",
		replacement: "record.ParseType(s, ErrEmptyEventType)",
	},
	{fragment: "core/event", symbol: "EnsureCustom", replacement: "event.Metadata.WithCustom"},
	{
		fragment:    "core/event",
		symbol:      "DetectTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "core/event",
		symbol:      "MarkTombstone",
		replacement: "domain events for deletion (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "core/event",
		symbol:      "MarkRebirth",
		replacement: "domain events for restore (docs/migration/tombstone-to-domain-events.md)",
	},
	{
		fragment:    "core/event",
		symbol:      "MetadataKeyTombstone",
		replacement: "domain events for deletion",
	},
	{
		fragment:    "core/event",
		symbol:      "MetadataKeyRebirth",
		replacement: "domain events for restore",
	},
	{fragment: "core/event", symbol: "TombstoneMark", replacement: "domain events for deletion"},
	{fragment: "core/event", symbol: "TombstoneStatus", replacement: "domain events"},
	{fragment: "core/metadata", symbol: "CustomData", replacement: "metadata.Metadata[K]"},
	{
		fragment:    "core/query",
		symbol:      "ParseType",
		replacement: "record.ParseType(s, ErrEmptyQueryType)",
	},
	// v5: manual snapshot helper replaced by encoding-aware construction.
	{
		fragment:    "snapshot",
		symbol:      "SaveSnapshot",
		replacement: "NewSnapshot + SnapshotSink.Save (carries record.Encoding)",
	},
	// ADR-0123: graph projection surface absorbed by metaengine + graphadapter.
	{
		fragment:    "graph",
		symbol:      "Handler",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "ProjectionOption",
		replacement: "metaengine auto-projection + graphadapter",
	},
	{
		fragment:    "graph",
		symbol:      "WithSchema",
		replacement: "metaengine auto-projection + graphadapter",
	},
	// ADR-0114: metadata tombstone-marking middleware dies with the marks.
	{
		fragment:    "listing",
		symbol:      "StatusMiddleware",
		replacement: "event-type-driven deletion (ADR-0114)",
	},
	// v5: silent-validation helper superseded by the checked form.
	{
		fragment:    "storage/sql",
		symbol:      "KeysetPositionQuery",
		replacement: "storage/sql.KeysetPositionQueryChecked",
	},
	{
		fragment:    "storage",
		symbol:      "WithoutRelationalAutoMigrate",
		replacement: "metaengine engines with layout planning",
	},
	// ADR-0123: storage-root re-exports of the removed view tier.
	{
		fragment:    "storage",
		symbol:      "ViewColumn",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "ViewMapper",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "IndexSpec",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "SQLViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "ViewStoreOption",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewSQLiteViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewSQLViewStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewViewStoreWithDialect",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "AutoMapper",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "AutoMapperWithTombstone",
		replacement: "metaengine engines with layout planning",
	},
	// ADR-0123: storage-root re-exports of the removed relational tier.
	{
		fragment:    "storage",
		symbol:      "NewRelationalProjection",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "NewRelationalStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalProjection",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalProjectionOption",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalSchema",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalTable",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalColumn",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalStore",
		replacement: "metaengine engines with layout planning",
	},
	{
		fragment:    "storage",
		symbol:      "RelationalHandler",
		replacement: "metaengine engines with layout planning",
	},
	{fragment: "storage", symbol: "Row", replacement: "metaengine engines with layout planning"},
	{
		fragment:    "storage",
		symbol:      "ProjectionSink",
		replacement: "metaengine engines with layout planning",
	},
}
