package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"
)

// Server dialects. Detected once at engine construction via SELECT VERSION();
// MariaDB reports "11.8.8-MariaDB-...", MySQL reports "8.4.11".
const (
	dialectMySQL   = "mysql"
	dialectMariaDB = "mariadb"
)

// detectDialect queries the server version and classifies it. MariaDB is
// wire-compatible with MySQL but lacks the -> JSON path operator and
// CAST(expr AS JSON), which it rejects with Error 1064. On any detection
// failure the conservative default is the MySQL dialect (previous behavior).
func detectDialect(db *sql.DB) string {
	var version string
	if err := db.QueryRowContext(context.Background(), "SELECT VERSION()").
		Scan(&version); err != nil {
		return dialectMySQL
	}

	return classifyVersion(version)
}

// classifyVersion maps a SELECT VERSION() string to a dialect constant.
// MariaDB reports e.g. "11.8.8-MariaDB-ubu2404"; MySQL reports "8.4.11".
func classifyVersion(version string) string {
	if strings.Contains(strings.ToUpper(version), "MARIADB") {
		return dialectMariaDB
	}

	return dialectMySQL
}

// isMariaDB reports whether the engine talks the MariaDB JSON dialect.
func (e *mysqlEngine) isMariaDB() bool { return e.dialect == dialectMariaDB }

// jsonFieldExpr renders the JSON path access expression for a field:
//
//	MySQL:   value->'$.field'         (JSON-typed, enables JSON comparison rules)
//	MariaDB: JSON_EXTRACT(value, '$.field')
//
// MariaDB does not implement the -> operator (Error 1064).
func (e *mysqlEngine) jsonFieldExpr(field string) string {
	if e.isMariaDB() {
		return fmt.Sprintf("JSON_EXTRACT(value, '$.%s')", escapeJSONPath(field))
	}

	return fmt.Sprintf("value->'$.%s'", escapeJSONPath(field))
}

// jsonCompareExpr renders the left-hand expression for a filter comparison
// against a parameter:
//
//	MySQL:   value->'$.field' compared against CAST(? AS JSON)
//	MariaDB: JSON_UNQUOTE(JSON_EXTRACT(value, '$.field')) compared against ?
//
// MariaDB's JSON_EXTRACT returns LONGTEXT, so JSON_UNQUOTE yields the scalar
// text form that binds directly against string and number parameters.
func (e *mysqlEngine) jsonCompareExpr(field string) string {
	if e.isMariaDB() {
		return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(value, '$.%s'))", escapeJSONPath(field))
	}

	return e.jsonFieldExpr(field)
}

// jsonParamPlaceholder renders the right-hand placeholder for a filter value:
// "CAST(? AS JSON)" on MySQL (exact JSON comparison semantics), plain "?" on
// MariaDB (which has no CAST target type JSON).
func (e *mysqlEngine) jsonParamPlaceholder() string {
	if e.isMariaDB() {
		return "?"
	}

	return "CAST(? AS JSON)"
}

// jsonSortExprs renders the ORDER BY expressions for a sort field.
//
//	MySQL:   value->'$.field' (JSON-typed: numbers compare numerically)
//	MariaDB: CAST(JSON_EXTRACT(value, '$.field') AS DECIMAL(65,10)),
//	         JSON_UNQUOTE(JSON_EXTRACT(value, '$.field'))
//
// MariaDB's JSON_EXTRACT returns LONGTEXT, so a bare sort text-sorts numbers
// ("10" < "2"). The dual key sorts numbers numerically while preserving
// lexical order for text fields: non-numeric text CASTs to 0 with a warning
// and the text tiebreak decides. Verified against MariaDB 11.8 and MySQL 8.4.
func (e *mysqlEngine) jsonSortExprs(field string) []string {
	if !e.isMariaDB() {
		return []string{e.jsonFieldExpr(field)}
	}

	escaped := escapeJSONPath(field)

	return []string{
		fmt.Sprintf("CAST(JSON_EXTRACT(value, '$.%s') AS DECIMAL(65,10))", escaped),
		fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(value, '$.%s'))", escaped),
	}
}

// jsonCursorExpr renders the keyset-pagination predicate expression for a
// cursor against a sort field. It must match the primary ORDER BY semantics:
// on MariaDB a numeric cursor compares through the DECIMAL cast (numeric
// order), any other cursor compares the unquoted text form (lexical order,
// which is total because the numeric cast ties at 0 for text values).
func (e *mysqlEngine) jsonCursorExpr(field string, cursor any) string {
	if !e.isMariaDB() {
		return e.jsonFieldExpr(field)
	}

	if isNativeNumber(cursor) {
		return fmt.Sprintf(
			"CAST(JSON_EXTRACT(value, '$.%s') AS DECIMAL(65,10))", escapeJSONPath(field))
	}

	return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(value, '$.%s'))", escapeJSONPath(field))
}

// isNativeNumber reports whether v is a Go numeric type (excluding bool).
func isNativeNumber(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// jsonFilterParam converts a Go filter value to the parameter bound against
// jsonCompareExpr. The MySQL dialect always binds the JSON-encoded text (the
// CAST re-parses it). MariaDB binds scalars natively so the text comparison
// sees the same scalar form JSON_UNQUOTE produces:
//
//	string "open" -> open      (matches JSON_UNQUOTE output)
//	bool   true   -> "true"    (JSON_UNQUOTE renders booleans as true/false)
//	numbers       -> passed through (driver binds a numeric parameter)
//	other (maps, slices, nil) -> JSON text (byte-identical to the stored form)
func (e *mysqlEngine) jsonFilterParam(v any) any {
	if !e.isMariaDB() {
		jb, _ := json.Marshal(v)

		return string(jb)
	}

	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}

		return "false"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return t
	default:
		jb, _ := json.Marshal(v)

		return string(jb)
	}
}
