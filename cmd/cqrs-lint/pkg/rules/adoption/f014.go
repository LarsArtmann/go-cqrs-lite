package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F014 detects projects using kv.NewTypedStore without kv.NewCache. The cache
// layer provides LRU-bounded, process-local caching for hot read models.
//
//nolint:ireturn // factory returns public interface
func NewF014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F014-no-kv-cache",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectHasCall(ctx, "kv", "NewTypedStore") {
				return nil, nil
			}

			if projectHasCall(ctx, "kv", "NewCache") {
				return nil, nil
			}

			pos, ok := firstCallPos(ctx, "kv", "NewTypedStore")
			if !ok {
				pos, ok = firstFilePos(ctx)
				if !ok {
					return nil, nil
				}
			}

			return singleInfoFinding(ctx,
				"F014",
				"kv.NewTypedStore is used but kv.NewCache is not — read model "+
					"access hits the backing store on every read",
				"Wrap your kv.TypedStore with kv.NewCache(store, "+
					"kv.WithCacheCapacity(500)) for LRU-bounded process-local "+
					"caching. Cache is generic: kv.NewCache[V, K](store).",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}
