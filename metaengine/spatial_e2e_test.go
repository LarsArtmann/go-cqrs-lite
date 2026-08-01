package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type PlaceLocated struct {
	ID  string
	Lat float64
	Lon float64
}

type NearbySearchInput struct {
	Lat    float64
	Lon    float64
	Radius float64
	Limit  int
}

func TestSpatialFoldPipeline_EndToEnd(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[NearbySearchInput, metaengine.SpatialResult](
			"nearby_places",
			metaengine.On(PlaceLocated{}, func(e PlaceLocated) metaengine.Point {
				return metaengine.Point{ID: e.ID, X: e.Lon, Y: e.Lat}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Berlin area: Brandenburg Gate ~52.5163, 13.3777
	// Places near Brandenburg Gate
	places := []PlaceLocated{
		{ID: "bg", Lat: 52.5163, Lon: 13.3777},        // Brandenburg Gate
		{ID: "reichstag", Lat: 52.5186, Lon: 13.3762}, // ~250m away
		{ID: "alex", Lat: 52.5219, Lon: 13.4132},      // Alexanderplatz ~2.5km
		{ID: "tempelhof", Lat: 52.4734, Lon: 13.4042}, // ~5km
	}

	for _, p := range places {
		if err := store.Apply(ctx, "PlaceLocated", p); err != nil {
			t.Fatal(err)
		}
	}

	// Search within 1km of Brandenburg Gate
	results, err := metaengine.SpatialExecuteTyped[NearbySearchInput](
		ctx, store,
		NearbySearchInput{Lat: 52.5163, Lon: 13.3777, Radius: 1000, Limit: 10},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Should find "bg" and "reichstag" within 1km
	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}

	if !ids["bg"] {
		t.Errorf("expected bg in results")
	}

	if !ids["reichstag"] {
		t.Errorf("expected reichstag in results")
	}

	if ids["alex"] {
		t.Errorf("alex should be >1km away")
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results within 1km, got %d", len(results))
	}

	// Nearest should be "bg" itself (distance ~0)
	if results[0].ID != "bg" {
		t.Errorf("nearest should be bg, got %s", results[0].ID)
	}
}

func TestSpatialFoldPipeline_Classification(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[NearbySearchInput, metaengine.SpatialResult](
			"spatial_classify",
			metaengine.On(PlaceLocated{}, func(e PlaceLocated) metaengine.Point {
				return metaengine.Point{ID: e.ID, X: e.Lon, Y: e.Lat}
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].ADT != metaengine.ADTSpatial {
		t.Errorf("expected ADT %q, got %q", metaengine.ADTSpatial, collections[0].ADT)
	}

	if collections[0].ReadPattern != metaengine.ReadSpatialRange {
		t.Errorf("expected pattern %q, got %q",
			metaengine.ReadSpatialRange, collections[0].ReadPattern)
	}
}
