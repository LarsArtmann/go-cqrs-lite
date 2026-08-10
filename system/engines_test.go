package system_test

// Blank imports ensure engine drivers self-register for system integration tests.
import (
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4" // registers "badger"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4" // registers "pebble"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // registers "sqlite"
)
