package relational

import "github.com/larsartmann/go-cqrs-lite/event/v3"

// Validation errors for RelationalSchema and ProjectionSink.
//
// All classified as Rejection: invalid schema declaration or a sink contract
// violation are non-retryable programmer/config errors. event.Classify and
// event.IsRetryable therefore report the correct family for consumers, instead
// of the Transient default that plain errors.New would trigger.
var (
	errSchemaNoTables = event.NewRejection(
		"relational.schema.no_tables",
		"relational schema: at least one table is required",
	)
	errSchemaDuplicateTable = event.NewRejection(
		"relational.schema.duplicate_table",
		"relational schema: duplicate table name",
	)
	errSchemaTableNoName = event.NewRejection(
		"relational.schema.table_name_required",
		"table Name is required",
	)
	errSchemaTableNoColumns = event.NewRejection(
		"relational.schema.columns_required",
		"at least one Column is required",
	)
	errSchemaColumnNoName = event.NewRejection(
		"relational.schema.column_name_required",
		"column Name is required",
	)
	errSchemaColumnNoType = event.NewRejection(
		"relational.schema.column_type_required",
		"column Type is required",
	)
	errSchemaDuplicateColumn = event.NewRejection(
		"relational.schema.duplicate_column",
		"duplicate column name",
	)
	errSchemaUnknownPKColumn = event.NewRejection(
		"relational.schema.unknown_pk_column",
		"primary key column not declared in Columns",
	)

	errSinkEmptyRow = event.NewRejection(
		"relational.sink.empty_row",
		"sink: row has no columns",
	)
	errSinkUnknownTable = event.NewRejection(
		"relational.sink.unknown_table",
		"sink: table not declared in schema",
	)
	errSinkUnknownColumn = event.NewRejection(
		"relational.sink.unknown_column",
		"sink: column not declared in schema",
	)
	errSinkNoRows = event.NewRejection(
		"relational.sink.no_rows",
		"sink: QueryOne matched no rows",
	)
)
