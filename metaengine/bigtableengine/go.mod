module github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4

go 1.26.7

require (
	cloud.google.com/go/bigtable v1.57.0
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.13.0
	github.com/onsi/gomega v1.43.0
	google.golang.org/api v0.250.0
	google.golang.org/grpc v1.76.0
)

// Sibling replace for unpublished metaengine symbols; stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
