module github.com/larsartmann/go-cqrs-lite/readmodel/v2

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.6.0
	github.com/larsartmann/go-cqrs-lite/kv/v2 v2.6.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../codec
	github.com/larsartmann/go-cqrs-lite/kv/v2 => ../kv
)
