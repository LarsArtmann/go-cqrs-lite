module github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4

go 1.26.5

require (
	github.com/cockroachdb/pebble v1.1.5
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.2.0
)

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
