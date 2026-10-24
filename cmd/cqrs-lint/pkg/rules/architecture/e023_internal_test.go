package architecture

import "testing"

// The engine-tier exemption is prefix-based on the package path (ADR-0142:
// substrate + engines own their SQL; v5-sentenced legacy tiers are known).
func TestIsEngineTierPackage(t *testing.T) {
	cases := map[string]bool{
		"github.com/larsartmann/go-cqrs-lite/metaengine/v4":            true,
		"github.com/larsartmann/go-cqrs-lite/metaengine/claimkit":      true,
		"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine":      true,
		"github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4":          true,
		"github.com/larsartmann/go-cqrs-lite/queue/postgres/v4":        true,
		"github.com/larsartmann/go-cqrs-lite/queue/mysql/v4":           true,
		"github.com/larsartmann/go-cqrs-lite/claiming/v4":              true,
		"github.com/larsartmann/go-cqrs-lite/storage/pebble":           true,
		"github.com/larsartmann/go-cqrs-lite/storage/sql":              true,
		"github.com/larsartmann/go-cqrs-lite/stack/sqlite":             true,
		"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4":    true,
		"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4":  true,
		"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer": true,
		"github.com/larsartmann/go-cqrs-lite/example/taskmanager":      true,
		"github.com/larsartmann/go-cqrs-lite/benchkit/v4":              true,
		"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench":           true,
		"github.com/larsartmann/go-cqrs-lite/system/v4":                false,
		"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4":     false,
		"github.com/larsartmann/go-cqrs-lite/queue/v4":                 false,
		"github.com/larsartmann/go-cqrs-lite/queue/v4/task":            false,
		"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance":     false,
		"github.com/larsartmann/go-cqrs-lite/middleware/v4":            false,
		"github.com/larsartmann/go-cqrs-lite/projectionhost/v4":        false,
		"github.com/example/consumer/app":                              false,
		"":                                                             false,
	}

	for pkgPath, exempt := range cases {
		if got := isEngineTierPackage(pkgPath); got != exempt {
			t.Errorf("isEngineTierPackage(%q) = %v, want %v", pkgPath, got, exempt)
		}
	}
}
