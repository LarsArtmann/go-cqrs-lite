package view

import "errors"

// errNilViewValue is returned when Set is called with a nil value.
var errNilViewValue = errors.New("storage: nil view value")

// Validation errors for ViewMapper.
var (
	errMapperTableRequired   = errors.New("mapper: Table is required")
	errMapperScanRowRequired = errors.New("mapper: ScanRow is required")
	errMapperColumnsRequired = errors.New("mapper: at least one Column is required")
	errMapperColumnNameEmpty = errors.New("mapper: column Name is required")
	errMapperExtractRequired = errors.New("mapper: column Extract is required")
	errMapperKeyReserved     = errors.New("mapper: column name \"key\" is reserved")
	errMapperDuplicateColumn = errors.New("mapper: duplicate column name")
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
