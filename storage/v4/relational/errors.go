package relational

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Validation errors for RelationalSchema and ProjectionSink.
//
// All classified as Rejection: invalid schema declaration or a sink contract
// violation are non-retryable programmer/config errors. errorfamily.Classify and
// errorfamily.IsRetryable therefore report the correct family for consumers, instead
// of the Transient default that plain errors.New would trigger.
var (
	errSchemaNoTables error = errorfamily.NewRejection(
		"relational.schema.no_tables",
		"relational schema: at least one table is required",
	)
	errSchemaDuplicateTable error = errorfamily.NewRejection(
		"relational.schema.duplicate_table",
		"relational schema: duplicate table name",
	)
	errSchemaTableNoName error = errorfamily.NewRejection(
		"relational.schema.table_name_required",
		"table Name is required",
	)
	errSchemaTableNoColumns error = errorfamily.NewRejection(
		"relational.schema.columns_required",
		"at least one Column is required",
	)
	errSchemaColumnNoName error = errorfamily.NewRejection(
		"relational.schema.column_name_required",
		"column Name is required",
	)
	errSchemaColumnNoType error = errorfamily.NewRejection(
		"relational.schema.column_type_required",
		"column Type is required",
	)
	errSchemaDuplicateColumn error = errorfamily.NewRejection(
		"relational.schema.duplicate_column",
		"duplicate column name",
	)
	errSchemaUnknownPKColumn error = errorfamily.NewRejection(
		"relational.schema.unknown_pk_column",
		"primary key column not declared in Columns",
	)
	errSchemaIndexNoName error = errorfamily.NewRejection(
		"relational.schema.index_no_name",
		"index Name is required",
	)
	errSchemaUnknownIndexColumn error = errorfamily.NewRejection(
		"relational.schema.unknown_index_column",
		"index column not declared in Columns",
	)
	errSchemaUniqueNoName error = errorfamily.NewRejection(
		"relational.schema.unique_no_name",
		"unique constraint Name is required",
	)
	errSchemaUnknownUniqueColumn error = errorfamily.NewRejection(
		"relational.schema.unknown_unique_column",
		"unique constraint column not declared in Columns",
	)

	errSinkEmptyRow error = errorfamily.NewRejection(
		"relational.sink.empty_row",
		"sink: row has no columns",
	)
	errSinkUnknownTable error = errorfamily.NewRejection(
		"relational.sink.unknown_table",
		"sink: table not declared in schema",
	)
	errSinkUnknownColumn error = errorfamily.NewRejection(
		"relational.sink.unknown_column",
		"sink: column not declared in schema",
	)
	errSinkNoRows error = errorfamily.NewRejection(
		"relational.sink.no_rows",
		"sink: QueryOne matched no rows",
	)
	errSinkCounterInKey error = errorfamily.NewRejection(
		"relational.sink.counter_in_key",
		"sink: counter column must not appear in the key Row",
	)
	errSinkKeyMissingPK error = errorfamily.NewRejection(
		"relational.sink.key_missing_pk",
		"sink: key Row must include all primary key columns",
	)
)
