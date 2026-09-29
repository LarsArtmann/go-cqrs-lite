package system_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Rapid properties over the LoadConfig koanf/YAML/env surfaces (Feedback #6
// (a) / M17.2). Generators are restricted to YAML-safe alphabets: the
// properties under test are koanf's mapping and merge order, not YAML quoting
// of hostile strings.

var cfgNameGen = rapid.StringMatching(`[a-z][a-z0-9]{2,7}`)

var cfgDSNGen = rapid.StringMatching(`[a-zA-Z0-9:/._-]{1,24}`)

var cfgDriverGen = rapid.SampledFrom([]string{
	"sqlite", "memory", "pebble", "duckdb", "postgres", "turso", "bigtable",
})

var cfgPriorityGen = rapid.SampledFrom([]metaengine.Priority{
	metaengine.PriorityWriteSpeed,
	metaengine.PriorityReadSpeed,
	metaengine.PriorityStorageSpace,
	metaengine.PriorityBalanced,
})

// TestProperty_LoadConfig_YAMLRoundTrip: an arbitrary single-engine YAML
// document loads back field-identically through LoadConfig.
func TestProperty_LoadConfig_YAMLRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		name := cfgNameGen.Draw(rt, "name")
		driver := cfgDriverGen.Draw(rt, "driver")
		dsn := cfgDSNGen.Draw(rt, "dsn")
		priority := cfgPriorityGen.Draw(rt, "priority")
		pragmaCount := rapid.IntRange(0, 3).Draw(rt, "pragmaCount")
		pragmas := make([]string, pragmaCount)
		for i := range pragmas {
			pragmas[i] = fmt.Sprintf("pragma_%d=on", i)
		}

		var b strings.Builder
		fmt.Fprintf(&b, "engines:\n  %s:\n    driver: %s\n    dsn: %q\n", name, driver, dsn)
		fmt.Fprintf(&b, "    priority: %s\n", priority)

		if len(pragmas) > 0 {
			b.WriteString("    pragmas:\n")
			for _, p := range pragmas {
				fmt.Fprintf(&b, "      - %q\n", p)
			}
		}

		cfg, err := system.LoadConfig(writeConfig(t, b.String()))
		if err != nil {
			rt.Fatalf("LoadConfig: %v", err)
		}

		eng, ok := cfg.Engines[name]
		if !ok {
			rt.Fatalf("engine %q missing after load", name)
		}

		if eng.Driver != driver || eng.DSN != dsn || eng.Priority != priority {
			rt.Fatalf("round-trip mismatch: %+v (want driver=%s dsn=%s priority=%s)",
				eng, driver, dsn, priority)
		}

		if len(eng.Pragmas) != len(pragmas) {
			rt.Fatalf("pragmas = %v, want %v", eng.Pragmas, pragmas)
		}
	})
}

// TestProperty_LoadConfig_EnvBeatsYAML: a structured env override always wins
// over the YAML value for the same field, independent of YAML content.
func TestProperty_LoadConfig_EnvBeatsYAML(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dsn := cfgDSNGen.Draw(rt, "dsn")
		override := cfgDSNGen.Draw(rt, "override")

		t.Setenv("CQRS_ENGINES__PRIMARY__DSN", override)
		t.Cleanup(func() { os.Unsetenv("CQRS_ENGINES__PRIMARY__DSN") })

		yml := fmt.Sprintf("engines:\n  primary:\n    driver: sqlite\n    dsn: %q\n", dsn)

		cfg, err := system.LoadConfig(writeConfig(t, yml))
		if err != nil {
			rt.Fatalf("LoadConfig: %v", err)
		}

		if got := cfg.Engines["primary"].DSN; got != override {
			rt.Fatalf("env override lost: dsn = %q, want %q", got, override)
		}
	})
}

// TestProperty_LoadConfig_IndexedInstanceOverride: for n instances and an
// in-range index, the env override lands on exactly that instance; every
// out-of-range index is a loud error, never a silent drop.
func TestProperty_LoadConfig_IndexedInstanceOverride(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 4).Draw(rt, "n")
		idx := rapid.IntRange(0, n).Draw(rt, "idx")
		durability := rapid.SampledFrom([]string{"strict", "normal", "relaxed"}).Draw(rt, "durability")

		var b strings.Builder
		b.WriteString("instances:\n")
		for range n {
			b.WriteString("  - role: events\n")
		}

		key := fmt.Sprintf("CQRS_INSTANCES__%d__DURABILITY", idx)
		//nolint:usetesting // per-iteration lifetime: t.Setenv cleans up at
		// test end, leaking each iteration's key into the next draw.
		os.Setenv(key, durability)
		defer os.Unsetenv(key)

		cfg, err := system.LoadConfig(writeConfig(t, b.String()))

		if idx >= n {
			if err == nil || !strings.Contains(err.Error(), "no instances") {
				rt.Fatalf("out-of-range override: err = %v, want loud no-instances error", err)
			}

			return
		}

		if err != nil {
			rt.Fatalf("LoadConfig: %v", err)
		}

		for i, inst := range cfg.Instances {
			want := system.DurabilityTier("")
			if i == idx {
				want = system.DurabilityTier(durability)
			}

			if inst.Durability != want {
				rt.Fatalf("instances[%d].durability = %q, want %q", i, inst.Durability, want)
			}
		}
	})
}
