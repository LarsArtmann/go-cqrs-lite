module example.com/busfixture

go 1.27

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0

require (
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-codec v0.3.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.5.2 // indirect
	github.com/larsartmann/go-cqrs-lite/id/v4 v4.7.1 // indirect
	github.com/larsartmann/go-cqrs-lite/metadata/v4 v4.7.3 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ../../../../event
