package relational

import (
	"fmt"
	"strings"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// mysqlUnlimited is MySQL's documented way to skip rows without a row cap:
// its LIMIT must be a non-negative literal, and OFFSET requires a LIMIT.
const mysqlUnlimited = "18446744073709551615"

// appendOffsetWithoutLimit writes the page window for an Offset with no
// Limit. Only SQLite accepts a negative LIMIT; PostgreSQL rejects it
// ("LIMIT must not be negative"), so the "no cap" spelling is per dialect.
// Dialects without a known spelling keep the historical LIMIT -1.
func appendOffsetWithoutLimit(
	b *strings.Builder,
	d sqlpkg.Dialect,
	offset, paramIdx int,
	args []any,
) []any {
	ph := d.Placeholder

	switch d.(type) {
	case sqlpkg.PostgresDialect:
		fmt.Fprintf(b, " OFFSET %s", ph(paramIdx))
		args = append(args, offset)
	case sqlpkg.MySQLDialect:
		fmt.Fprintf(b, " LIMIT %s OFFSET %s", mysqlUnlimited, ph(paramIdx))
		args = append(args, offset)
	default:
		fmt.Fprintf(b, " LIMIT %s OFFSET %s", ph(paramIdx), ph(paramIdx+1))
		args = append(args, -1, offset)
	}

	return args
}
