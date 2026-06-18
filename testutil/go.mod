module github.com/larsartmann/go-cqrs-lite/testutil/v2

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/command/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/id/v2 v2.5.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-branded-id v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 v2.5.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v2 v2.5.0 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/ro v0.3.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../codec
	github.com/larsartmann/go-cqrs-lite/command/v2 => ../command
	github.com/larsartmann/go-cqrs-lite/dispatcher/v2 => ../dispatcher
	github.com/larsartmann/go-cqrs-lite/event/v2 => ../event
	github.com/larsartmann/go-cqrs-lite/id/v2 => ../id
)
