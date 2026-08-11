//go:build cgo

package system_test

import (
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4" // registers "duckdb"
)
