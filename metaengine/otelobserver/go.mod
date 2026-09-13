module github.com/larsartmann/go-cqrs-lite/metaengine/otelobserver/v4

go 1.26.7

require (
	github.com/larsartmann/go-cqrs-lite/metaengine/v4 v4.13.0
	github.com/larsartmann/go-cqrs-lite/otel/v4 v4.4.0
)

// Sibling replace for unpublished metaengine symbols (health hooks:
// OnQuarantined/OnReactivated/OnProbe/OnCatchUp, Hooks.Merge, CurrentHooks);
// stripped by scripts/tag-release.sh at cut time.
replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../
