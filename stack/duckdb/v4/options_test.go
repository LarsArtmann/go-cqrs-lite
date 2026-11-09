package duckdb

import (
	"testing"
)

func TestAppendDuckDBOptions(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		cfg  config
		want string
	}{
		{
			name: "empty DSN with both options",
			dsn:  "",
			cfg:  config{duckdbConfig: duckdbConfig{Threads: 4, MemoryLimit: "1GB"}},
			want: "?threads=4&memory_limit=1GB",
		},
		{
			name: "empty DSN no options",
			dsn:  "",
			cfg:  config{},
			want: "",
		},
		{
			name: "DSN with existing question mark",
			dsn:  "test.db?cache=shared",
			cfg:  config{duckdbConfig: duckdbConfig{Threads: 2}},
			want: "test.db?cache=shared&threads=2",
		},
		{
			name: "DSN with existing ampersand",
			dsn:  "test.db?threads=1&cache=shared",
			cfg:  config{duckdbConfig: duckdbConfig{MemoryLimit: "512MB"}},
			want: "test.db?threads=1&cache=shared&memory_limit=512MB",
		},
		{
			name: "only memory_limit set",
			dsn:  ":memory:",
			cfg:  config{duckdbConfig: duckdbConfig{MemoryLimit: "2GB"}},
			want: ":memory:?memory_limit=2GB",
		},
		{
			name: "only threads set",
			dsn:  ":memory:",
			cfg:  config{duckdbConfig: duckdbConfig{Threads: 8}},
			want: ":memory:?threads=8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendDuckDBOptions(tt.dsn, tt.cfg)
			if got != tt.want {
				t.Errorf("appendDuckDBOptions(%q, %+v) = %q, want %q", tt.dsn, tt.cfg, got, tt.want)
			}
		})
	}
}
