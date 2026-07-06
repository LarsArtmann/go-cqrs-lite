package view

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// errNilViewValue is returned when Set is called with a nil value.
var errNilViewValue = errorfamily.NewRejection("storage.view.nil_value", "storage: nil view value")

// Validation errors for ViewMapper. All Rejection: a misconfigured mapper is a
// non-retryable programmer error, not a Transient fault.
var (
	errMapperTableRequired = errorfamily.NewRejection(
		"storage.view.mapper.table_required",
		"mapper: Table is required",
	)
	errMapperScanRowRequired = errorfamily.NewRejection(
		"storage.view.mapper.scan_row_required",
		"mapper: ScanRow is required",
	)
	errMapperColumnsRequired = errorfamily.NewRejection(
		"storage.view.mapper.columns_required",
		"mapper: at least one Column is required",
	)
	errMapperColumnNameEmpty = errorfamily.NewRejection(
		"storage.view.mapper.column_name_required",
		"mapper: column Name is required",
	)
	errMapperExtractRequired = errorfamily.NewRejection(
		"storage.view.mapper.extract_required",
		"mapper: column Extract is required",
	)
	errMapperKeyReserved = errorfamily.NewRejection(
		"storage.view.mapper.key_reserved",
		"mapper: column name \"key\" is reserved",
	)
	errMapperDuplicateColumn = errorfamily.NewRejection(
		"storage.view.mapper.duplicate_column",
		"mapper: duplicate column name",
	)
)

// ViewStoreOption configures a [SQLViewStore] at construction time.
type ViewStoreOption func(*viewStoreConfig)

type viewStoreConfig struct {
	autoMigrate bool
}

// WithoutViewAutoMigrate skips automatic CREATE TABLE IF NOT EXISTS.
// Use this when the caller manages schema manually (e.g. external migrations).
func WithoutViewAutoMigrate() ViewStoreOption {
	return func(c *viewStoreConfig) { c.autoMigrate = false }
}
