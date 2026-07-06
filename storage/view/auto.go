package view

import (
	"fmt"
	"reflect"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	sqlTypeText    = "TEXT"
	sqlTypeInteger = "INTEGER"
	keyColumnName  = "key"
)

// AutoMapper generates a [ViewMapper] from struct tags on V. Fields tagged
// with `view:"column_name"` become SQL columns; the SQL type is inferred from
// the Go type. Fields without the tag are skipped (not stored as columns).
//
// A field tagged `view:"-"` is explicitly skipped.
//
// The key column (always TEXT PRIMARY KEY) is added automatically — do NOT
// tag the key field.
//
// SQL type mapping:
//
//	string, *string  → TEXT
//	int, int32, int64 → INTEGER
//	uint, uint32     → INTEGER
//	float32, float64 → REAL
//	bool             → INTEGER
//	time.Time        → TEXT
//
// Example:
//
//	type UserView struct {
//	    Name       string `view:"name"`
//	    Email      string `view:"email"`
//	    Age        int    `view:"age"`
//	    Tombstoned bool   `view:"tombstoned"`
//	}
//
//	mapper := storage.AutoMapper[UserView]("users_view")
//	// Equivalent to manually defining ViewMapper with 4 columns.
//
// For tombstone support, name the boolean column "tombstoned" or pass the
// column name via [AutoMapperWithTombstone].
func AutoMapper[V any](table string) ViewMapper[V] {
	return AutoMapperWithTombstone[V](table, "")
}

// AutoMapperWithTombstone is like [AutoMapper] but also sets the
// TombstoneColumn on the generated mapper. The tombstone column must be a
// tagged boolean field. Pass an empty string to disable tombstone pushdown.
//
// Panics if V is not a struct or a pointer to a struct — this is a programmer
// error detected at startup, not a runtime condition.
func AutoMapperWithTombstone[V any](table, tombstoneCol string) ViewMapper[V] {
	var zero V

	rt := reflect.TypeOf(zero)

	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("storage: AutoMapper: %s is not a struct", rt))
	}

	type fieldInfo struct {
		index   int
		colName string
		colType string
		isBool  bool
	}

	fields := make([]fieldInfo, 0, rt.NumField())

	for i := range rt.NumField() {
		f := rt.Field(i)
		tag := f.Tag.Get("view")
		if tag == "" || tag == "-" {
			continue
		}

		sqlType, isBool := goTypeToSQL(f.Type)
		fields = append(fields, fieldInfo{
			index:   i,
			colName: tag,
			colType: sqlType,
			isBool:  isBool,
		})
	}

	columns := make([]ViewColumn[V], 0, len(fields))
	scanOrder := make([]fieldInfo, 0, len(fields))

	for _, fi := range fields {
		col := ViewColumn[V]{
			Name: fi.colName,
			Type: fi.colType,
			Extract: func(v *V) any {
				rv := reflect.ValueOf(v)
				if rv.Kind() == reflect.Pointer {
					rv = rv.Elem()
				}

				fv := rv.Field(fi.index)
				if fi.isBool {
					return fv.Bool()
				}

				return fv.Interface()
			},
		}
		columns = append(columns, col)
		scanOrder = append(scanOrder, fi)
	}

	mapper := ViewMapper[V]{
		Table:   table,
		Columns: columns,
		ScanRow: func(scan func(dest ...any) error) (*V, error) {
			var v V

			rv := reflect.ValueOf(&v).Elem()

			dest := make([]any, len(scanOrder))
			boolTargets := make([]*bool, len(scanOrder))

			for i, fi := range scanOrder {
				if fi.isBool {
					var b bool
					boolTargets[i] = &b
					dest[i] = &b
				} else {
					fv := rv.Field(fi.index)
					dest[i] = fv.Addr().Interface()
				}
			}

			if err := scan(dest...); err != nil {
				return nil, errorfamily.WrapCorruption(err, "storage.view.auto.scan_row",
					"scan row via reflection")
			}

			for i, fi := range scanOrder {
				if fi.isBool && boolTargets[i] != nil {
					rv.Field(fi.index).SetBool(*boolTargets[i])
				}
			}

			return &v, nil
		},
	}

	if tombstoneCol != "" {
		mapper.TombstoneColumn = tombstoneCol
	}

	return mapper
}

func goTypeToSQL(rt reflect.Type) (string, bool) {
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	switch rt.Kind() { //nolint:exhaustive // unknown Kinds intentionally fall through to TEXT
	case reflect.String:
		return sqlTypeText, false
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
		return sqlTypeInteger, false
	case reflect.Float32, reflect.Float64:
		return "REAL", false
	case reflect.Bool:
		return sqlTypeInteger, true
	default:
		if rt == reflect.TypeFor[time.Time]() {
			return sqlTypeText, false
		}

		return sqlTypeText, false
	}
}
