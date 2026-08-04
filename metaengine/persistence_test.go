package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPersistence_ZeroValueIsVolatile(t *testing.T) {
	t.Parallel()

	var p metaengine.Persistence
	if p != metaengine.PersistenceVolatile {
		t.Errorf("zero value should be PersistenceVolatile, got %q", p)
	}
}

func TestPersistence_Constants(t *testing.T) {
	t.Parallel()

	if metaengine.PersistenceVolatile != "" {
		t.Errorf("PersistenceVolatile should be empty string, got %q", metaengine.PersistenceVolatile)
	}
	if metaengine.PersistencePersistent != "persistent" {
		t.Errorf("PersistencePersistent should be \"persistent\", got %q",
			metaengine.PersistencePersistent)
	}
}

func TestEngineProfile_IsVolatile_ZeroValue(t *testing.T) {
	t.Parallel()

	var p metaengine.EngineProfile
	if !p.IsVolatile() {
		t.Error("zero-value EngineProfile should be volatile")
	}
	if p.IsPersistent() {
		t.Error("zero-value EngineProfile should not be persistent")
	}
}

func TestEngineProfile_IsPersistent(t *testing.T) {
	t.Parallel()

	p := metaengine.EngineProfile{
		Persistence: metaengine.PersistencePersistent,
	}
	if !p.IsPersistent() {
		t.Error("profile with PersistencePersistent should be persistent")
	}
	if p.IsVolatile() {
		t.Error("profile with PersistencePersistent should not be volatile")
	}
}

func TestEngineProfileString_IncludesVolatileForVolatileEngine(t *testing.T) {
	t.Parallel()

	p := metaengine.NewMemoryEngine().Profile()
	s := p.String()
	if !strings.Contains(s, "volatile") {
		t.Errorf("volatile engine String() should contain \"volatile\", got: %s", s)
	}
}

func TestEngineProfileString_OmitsVolatileForPersistentEngine(t *testing.T) {
	t.Parallel()

	p := metaengine.SQLiteEngineProfile()
	s := p.String()
	if strings.Contains(s, "volatile") {
		t.Errorf("persistent engine String() should not contain \"volatile\", got: %s", s)
	}
}

func TestMemoryEngine_ProfileIsVolatile(t *testing.T) {
	t.Parallel()

	p := metaengine.NewMemoryEngine().Profile()
	if !p.IsVolatile() {
		t.Error("Memory engine should be volatile")
	}
}

func TestSQLiteEngineProfile_IsPersistent(t *testing.T) {
	t.Parallel()

	p := metaengine.SQLiteEngineProfile()
	if !p.IsPersistent() {
		t.Error("SQLite engine profile should be persistent")
	}
}

func TestStorePersistence_AccessorForVolatileEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if got := store.Persistence("find_task"); got != metaengine.PersistenceVolatile {
		t.Errorf("Memory-backed query Persistence: expected %q, got %q",
			metaengine.PersistenceVolatile, got)
	}
}

func TestStorePersistence_AccessorForUnknownQuery(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if got := store.Persistence("nonexistent"); got != metaengine.PersistenceVolatile {
		t.Errorf("unknown query Persistence: expected %q (safe default), got %q",
			metaengine.PersistenceVolatile, got)
	}
}

func TestCollections_ExposesPersistence(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	c := collections[0]
	if c.Persistence != metaengine.PersistenceVolatile {
		t.Errorf("Memory-backed collection Persistence: expected %q, got %q",
			metaengine.PersistenceVolatile, c.Persistence)
	}
}

func TestSerializableQuery_PersistenceRoundTrip(t *testing.T) {
	t.Parallel()

	volatileEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "test-volatile",
		Persistence: metaengine.PersistenceVolatile,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{volatileEngine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{volatileEngine})
	if len(sp.Queries) != 1 {
		t.Fatalf("expected 1 query in serialized plan, got %d", len(sp.Queries))
	}

	if sp.Queries[0].Persistence != metaengine.PersistenceVolatile {
		t.Errorf("serialized query Persistence: expected %q, got %q",
			metaengine.PersistenceVolatile, sp.Queries[0].Persistence)
	}
}
