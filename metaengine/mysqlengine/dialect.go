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
