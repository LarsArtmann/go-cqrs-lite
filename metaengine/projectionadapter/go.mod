module github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.1.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/projection/v4 v4.1.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.1.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.1.0 // indirect
	github.com/larsartmann/go-error-family v0.9.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

// Workspace-local replaces. metaengine is not yet tagged (experimental).
// event/ and projection/ are tagged but their published v4.1.0 go.mod files
// reference intermediate sibling versions that were never tagged. Resolving
// to the local workspace directories picks up the current go.mod files which
// reference correctly-tagged v4.1.0 siblings. All replaces will be removed
// once the v4.1.0 tag chain is republished or metaengine gets its first tag.
replace (
	github.com/larsartmann/go-cqrs-lite/event/v4 => ../../event
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
	github.com/larsartmann/go-cqrs-lite/projection/v4 => ../../projection
)
