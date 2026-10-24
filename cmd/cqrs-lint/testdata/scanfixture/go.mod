module example.com/scanfixture

go 1.27.2

require github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.0.0

require (
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.6 // indirect
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-codec v0.3.2 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.3.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2 // indirect
	github.com/larsartmann/go-error-family v0.12.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../../../metaengine

replace github.com/larsartmann/go-cqrs-lite/dedup/v4 => ../../../../dedup

replace github.com/larsartmann/go-cqrs-lite/record/v4 => ../../../../record
