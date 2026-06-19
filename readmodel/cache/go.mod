module github.com/larsartmann/go-cqrs-lite/readmodel/cache/v2

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/kv/v2 v2.5.0
	github.com/larsartmann/go-cqrs-lite/readmodel/v2 v2.0.0-00010101000000-000000000000
	github.com/maypok86/otter/v2 v2.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/larsartmann/go-error-family v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/codec/v2 => ../../codec
	github.com/larsartmann/go-cqrs-lite/kv/v2 => ../../kv
	github.com/larsartmann/go-cqrs-lite/readmodel/v2 => ..
)
