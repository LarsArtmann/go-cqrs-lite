module github.com/larsartmann/go-cqrs-lite/idempotency/v3

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/command/v3 v3.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/event/v3 v3.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/id/v3 v3.0.0-00010101000000-000000000000
	github.com/larsartmann/go-cqrs-lite/kv/v3 v3.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 v3.0.0-00010101000000-000000000000 // indirect
	github.com/larsartmann/go-error-family v0.5.1 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v3 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v3 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v3 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v3 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v3 => ../id
	github.com/larsartmann/go-cqrs-lite/kv/v3 => ../kv
	github.com/larsartmann/go-cqrs-lite/query/v3 => ../query
	github.com/larsartmann/go-cqrs-lite/schema/v3 => ../schema
	github.com/larsartmann/go-cqrs-lite/snapshot/v3 => ../snapshot
	github.com/larsartmann/go-cqrs-lite/storage/memory/v3 => ../storage/memory
)
