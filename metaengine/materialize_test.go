package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestMaterializeVsReplay_HighReadLowWrite_RecommendsMaterialize(t *testing.T) {
	t.Parallel()

	q := findTaskQuery()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
		metaengine.WithWorkloadStats(map[string]metaengine.WorkloadStats{
			"find_task": {
				WriteRatePerSec: 10,
				ReadRatePerSec:  100,
				AvgStreamLength: 50,
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// replay_cost = 100 * 50 * 1 = 5000
	// materialize_cost = 10 * 1 + 100 * 0.1 = 20
	// materialize wins overwhelmingly
	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelInfo &&
			strings.Contains(d.Message, "materialize recommended") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected materialize recommendation, got diagnostics: %v", store.Plan().Diagnostics)
	}
}

func TestMaterializeVsReplay_LowReadHighWrite_RecommendsReplay(t *testing.T) {
	t.Parallel()

	q := findTaskQuery()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
		metaengine.WithWorkloadStats(map[string]metaengine.WorkloadStats{
			"find_task": {
				WriteRatePerSec: 1000,
				ReadRatePerSec:  1,
				AvgStreamLength: 5,
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// replay_cost = 1 * 5 * 1 = 5
	// materialize_cost = 1000 * 1 + 1 * 0.1 = 1000.1
	// replay wins overwhelmingly
	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelWarn &&
			strings.Contains(d.Message, "replay may be cheaper") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected replay recommendation, got diagnostics: %v", store.Plan().Diagnostics)
	}
}

func TestMaterializeVsReplay_NoStats_NoRecommendation(t *testing.T) {
	t.Parallel()

	q := findTaskQuery()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "materialize") || strings.Contains(d.Message, "replay") {
			t.Errorf("unexpected materialization diagnostic without stats: %s", d.Message)
		}
	}
}

func TestMaterializeVsReplay_ZeroRates_NoRecommendation(t *testing.T) {
	t.Parallel()

	q := findTaskQuery()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
		metaengine.WithWorkloadStats(map[string]metaengine.WorkloadStats{
			"find_task": {}, // all zeros
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "materialize") || strings.Contains(d.Message, "replay") {
			t.Errorf("unexpected materialization diagnostic with zero rates: %s", d.Message)
		}
	}
}

func TestMaterializeVsReplay_FormulaCorrectness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		stats          metaengine.WorkloadStats
		wantMaterialize bool
	}{
		{
			name: "balanced 1:1 read/write, short stream",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 10,
				ReadRatePerSec:  10,
				AvgStreamLength: 2,
			},
			// replay = 10*2*1 = 20, materialize = 10*1 + 10*0.1 = 11 → materialize
			wantMaterialize: true,
		},
		{
			name: "high write, low read, long stream",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 100,
				ReadRatePerSec:  1,
				AvgStreamLength: 100,
			},
			// replay = 1*100*1 = 100, materialize = 100*1 + 1*0.1 = 100.1 → replay wins barely
			wantMaterialize: false,
		},
		{
			name: "very high read, low write",
			stats: metaengine.WorkloadStats{
				WriteRatePerSec: 1,
				ReadRatePerSec:  1000,
				AvgStreamLength: 100,
			},
			// replay = 1000*100 = 100000, materialize = 1 + 100 = 101 → materialize
			wantMaterialize: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rc := metaengine.ReplayCost(tt.stats)
			mc := metaengine.MaterializeCost(tt.stats)
			got := metaengine.ShouldMaterialize(tt.stats)

			if got != tt.wantMaterialize {
				t.Errorf("ShouldMaterialize(%+v) = %v, want %v (replay=%.2f, materialize=%.2f)",
					tt.stats, got, tt.wantMaterialize, rc, mc)
			}
		})
	}
}
