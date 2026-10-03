package analyzer

import (
	"testing"

	jsonv2 "encoding/json/v2"
)

func TestStoreSpecUnmarshalScalarAndArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		json  string
		want  []StoreKind
		fails bool
	}{
		{"scalar", `{"store":"sqlite"}`, []StoreKind{StoreSQLite}, false},
		{
			"array",
			`{"store":["postgres","sqlite"]}`,
			[]StoreKind{StorePostgres, StoreSQLite},
			false,
		},
		{"array single", `{"store":["pebble"]}`, []StoreKind{StorePebble}, false},
		{"empty array", `{"store":[]}`, nil, true},
		{"number", `{"store":42}`, nil, true},
		{"object", `{"store":{"a":1}}`, nil, true},
		{"bad kind", `{"store":"oracle"}`, nil, true}, // unknown names rejected with the valid list
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cfg struct {
				Store *StoreSpec `json:"store,omitempty"`
			}
			err := jsonv2.Unmarshal([]byte(tc.json), &cfg)
			if tc.fails {
				if err == nil {
					t.Fatalf("expected error for %s, got %+v", tc.json, cfg.Store)
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal %s: %v", tc.json, err)
			}
			if cfg.Store == nil {
				t.Fatal("store spec not populated")
			}
			if len(cfg.Store.Kinds) != len(tc.want) {
				t.Fatalf("kinds = %v, want %v", cfg.Store.Kinds, tc.want)
			}
			for i, k := range tc.want {
				if cfg.Store.Kinds[i] != k {
					t.Fatalf("kinds = %v, want %v", cfg.Store.Kinds, tc.want)
				}
			}
		})
	}
}

func TestStoreSpecMarshalScalarVsArray(t *testing.T) {
	t.Parallel()

	single, err := jsonv2.Marshal(struct {
		Store StoreSpec `json:"store"`
	}{Store: StoreSpec{Kinds: []StoreKind{StoreSQLite}}})
	if err != nil {
		t.Fatalf("marshal single: %v", err)
	}
	if got, want := string(single), `{"store":"sqlite"}`; got != want {
		t.Fatalf("single = %s, want %s", got, want)
	}

	multi, err := jsonv2.Marshal(struct {
		Store StoreSpec `json:"store"`
	}{Store: StoreSpec{Kinds: []StoreKind{StorePostgres, StoreSQLite}}})
	if err != nil {
		t.Fatalf("marshal multi: %v", err)
	}
	if got, want := string(multi), `{"store":["postgres","sqlite"]}`; got != want {
		t.Fatalf("multi = %s, want %s", got, want)
	}
}

func TestStoreSpecPrimaryAndString(t *testing.T) {
	t.Parallel()

	if got := (StoreSpec{}).Primary(); got != StoreNone {
		t.Fatalf("empty primary = %s, want none", got)
	}
	spec := StoreSpec{Kinds: []StoreKind{StorePostgres, StoreSQLite}}
	if got := spec.Primary(); got != StorePostgres {
		t.Fatalf("primary = %s, want postgres", got)
	}
	if got, want := spec.String(), "postgres+sqlite"; got != want {
		t.Fatalf("string = %s, want %s", got, want)
	}
}
