module example.com/scanfixture

go 1.27.1

require github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.0.0

require (
	github.com/larsartmann/go-branded-id v0.7.0 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.2 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.0 // indirect
	github.com/larsartmann/go-error-family v0.11.0 // indirect
	github.com/larsartmann/go-sse v0.6.2 // indirect
)

replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../../../metaengine
