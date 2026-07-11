package storage

import (
	"github.com/larsartmann/go-cqrs-lite/storage/v4/readmodel"
)

// Re-exports for backward compatibility. Consumers importing
// storage.SQLKVStore continue to work. New code should import
// the readmodel package directly.

type SQLKVStore = readmodel.SQLKVStore

var (
	NewSQLKVStore            = readmodel.NewSQLKVStore
	NewSQLiteKVStore         = readmodel.NewSQLiteKVStore
	NewSQLKVStoreWithDialect = readmodel.NewSQLKVStoreWithDialect
)
