package sql

import (
	"regexp"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// identifierPattern matches bare SQL identifiers: an ASCII letter or
// underscore, followed by letters, digits, or underscores. Dotted paths,
// quoted fragments, whitespace, and punctuation are all rejected so an
// identifier can never smuggle SQL syntax into a query.
var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// isSupportedOperator reports whether op belongs to the closed set of
// operators the WHERE-clause builder can render. Any other [kv.Operator]
// value is rejected: operator strings are interpolated into SQL verbatim,
// so an open set would be an injection surface.
func isSupportedOperator(op kv.Operator) bool {
	switch op {
	case kv.OpEq, kv.OpNeq, kv.OpLt, kv.OpLte, kv.OpGt, kv.OpGte,
		kv.OpLike, kv.OpIn, kv.OpIsNull, kv.OpIsNotNull:
		return true
	default:
		return false
	}
}

// ValidateIdentifier reports whether name is a bare SQL identifier
// (letters, digits, underscore; must not start with a digit). Column names
// that fail this check can inject SQL when interpolated into a query.
func ValidateIdentifier(name string) bool {
	return identifierPattern.MatchString(name)
}

// ValidateOperator reports whether op is one of the supported comparison
// operators. Unsupported operators are rejected before any SQL is built.
func ValidateOperator(op kv.Operator) bool {
	return isSupportedOperator(op)
}

// ValidateJournalIdentifiers validates the table and timestamp-column names
// that JournalReader and the keyset helpers interpolate verbatim into SQL.
// Exported so stores can fail fast at construction instead of at query time.
func ValidateJournalIdentifiers(table, timestampColumn string) error {
	if !ValidateIdentifier(table) {
		return errorfamily.NewInfrastructure(
			"sql.invalid_identifier",
			"table name is not a bare SQL identifier: "+table,
		)
	}

	if !ValidateIdentifier(timestampColumn) {
		return errorfamily.NewInfrastructure(
			"sql.invalid_identifier",
			"timestamp column is not a bare SQL identifier: "+timestampColumn,
		)
	}

	return nil
}
