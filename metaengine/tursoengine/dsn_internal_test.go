package tursoengine

import "testing"

func TestWithExperimentalToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "no query appends flag",
			dsn:  ":memory:",
			want: ":memory:?experimental=views",
		},
		{
			name: "existing param merges comma list",
			dsn:  "/data/app.db?experimental=encryption",
			want: "/data/app.db?experimental=encryption%2Cviews",
		},
		{
			name: "token already present passes through",
			dsn:  "/data/app.db?experimental=views",
			want: "/data/app.db?experimental=views",
		},
		{
			name: "token present in longer list passes through",
			dsn:  "/data/app.db?experimental=encryption,views",
			want: "/data/app.db?experimental=encryption,views",
		},
		{
			name: "other params preserved in original order",
			dsn:  ":memory:?authtoken=x&encryption_cipher=aegis256",
			want: ":memory:?authtoken=x&encryption_cipher=aegis256&experimental=views",
		},
		{
			name: "empty experimental value replaced",
			dsn:  ":memory:?experimental=",
			want: ":memory:?experimental=views",
		},
		{
			name: "unparseable escape passes through untouched",
			dsn:  ":memory:?experimental=%ZZ",
			want: ":memory:?experimental=%ZZ",
		},
		{
			name: "remote DSN passes through",
			dsn:  "libsql://my-db.turso.io",
			want: "libsql://my-db.turso.io",
		},
		{
			name: "substring match does not false-positive",
			dsn:  "/data/app.db?notexperimental=1",
			want: "/data/app.db?notexperimental=1&experimental=views",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := withExperimentalToken(tt.dsn, "views"); got != tt.want {
				t.Errorf("withExperimentalToken(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestNormalizeEmbeddedDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{name: "plain path untouched", dsn: "/data/app.db", want: "/data/app.db"},
		{name: "file scheme stripped", dsn: "file:/data/app.db", want: "/data/app.db"},
		{name: "file// scheme stripped", dsn: "file:///data/app.db", want: "/data/app.db"},
		{name: "file with query params passes through", dsn: "file:/data/app.db?mode=memory", want: "file:/data/app.db?mode=memory"},
		{name: "remote untouched", dsn: "libsql://db.turso.io", want: "libsql://db.turso.io"},
		{name: "memory untouched", dsn: ":memory:", want: ":memory:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeEmbeddedDSN(tt.dsn); got != tt.want {
				t.Errorf("normalizeEmbeddedDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
