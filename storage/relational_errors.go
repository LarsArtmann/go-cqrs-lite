package storage

import "errors"

// Validation errors for RelationalSchema and ProjectionSink.
var (
	errSchemaNoTables        = errors.New("relational schema: at least one table is required")
	errSchemaDuplicateTable  = errors.New("relational schema: duplicate table name")
	errSchemaTableNoName     = errors.New("table Name is required")
	errSchemaTableNoColumns  = errors.New("at least one Column is required")
	errSchemaColumnNoName    = errors.New("column Name is required")
	errSchemaColumnNoType    = errors.New("column Type is required")
	errSchemaDuplicateColumn = errors.New("duplicate column name")
	errSchemaUnknownPKColumn = errors.New("primary key column not declared in Columns")

	errSinkEmptyRow      = errors.New("sink: row has no columns")
	errSinkUnknownTable  = errors.New("sink: table not declared in schema")
	errSinkUnknownColumn = errors.New("sink: column not declared in schema")
	errSinkNoRows        = errors.New("sink: QueryOne matched no rows")
)
