module github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4

go 1.27.1

require (
	git.coopcloud.tech/decentral1se/iroh-go v0.0.0-20260830120307-d6233351aba3
	github.com/fxamacker/cbor/v2 v2.9.4
	github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.2
	github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4 v4.3.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.14.0
	github.com/onsi/gomega v1.43.1
	github.com/samber/lo v1.53.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/larsartmann/go-branded-id v0.6.0 // indirect
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.0 // indirect
	github.com/larsartmann/go-error-family v0.10.1 // indirect
	github.com/larsartmann/go-sse v0.6.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

// Sibling replace for unpublished irohengine symbols (DefaultDedupCapacity);
// stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4 => ../
