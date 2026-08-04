module github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4

go 1.26.5

require (
	git.coopcloud.tech/decentral1se/iroh-go v0.0.0-20260731195909-77011195d3d6
	github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4 v4.0.0
)

require (
	github.com/larsartmann/go-branded-id v0.5.1 // indirect
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.0.0 // indirect
	github.com/larsartmann/go-error-family v0.10.0 // indirect
	github.com/larsartmann/go-sse v0.4.0 // indirect
)

replace (
	github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4 => ../
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../../
)
