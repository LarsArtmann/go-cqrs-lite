module github.com/larsartmann/go-cqrs-lite/kv/v3

go 1.26.3

require (
	github.com/larsartmann/go-cqrs-lite/codec/v3 v3.1.0
	github.com/larsartmann/go-error-family v0.5.1
	github.com/maypok86/otter/v2 v2.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/codec/v3 => ../codec
