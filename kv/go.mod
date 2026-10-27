module github.com/larsartmann/go-cqrs-lite/kv/v4

go 1.27.2

require (
	github.com/larsartmann/go-codec v0.3.2
	github.com/larsartmann/go-cqrs-lite/record/v4 v4.6.2
	github.com/larsartmann/go-error-family v0.12.0
	github.com/maypok86/otter/v2 v2.4.0
	pgregory.net/rapid v1.3.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.6 // indirect
	github.com/gkampitakis/go-snaps v0.5.24 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

// Sibling replace for unpublished record symbols (DeferClose, ADR-0144);
// stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/record/v4 => ../record
