package tursoengine

import (
	"strings"
	"testing"
)

func TestRedactDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "local file path passes through",
			dsn:  "/data/app.db",
			want: "/data/app.db",
		},
		{
			name: "in-memory passes through",
			dsn:  ":memory:",
			want: ":memory:",
		},
		{
			name: "remote URL without credentials keeps host",
			dsn:  "libsql://my-db.turso.io",
			want: "libsql://my-db.turso.io",
		},
		{
			name: "userinfo token redacted",
			dsn:  "libsql://secret-token@my-db.turso.io",
			want: "libsql://redacted@my-db.turso.io",
		},
		{
			name: "authToken query parameter redacted",
			dsn:  "libsql://my-db.turso.io?authToken=super-secret",
			want: "libsql://my-db.turso.io?authToken=%5Bredacted%5D",
		},
		{
			name: "unrelated query parameters preserved",
			dsn:  "https://db.example.com/dbname?jwt=abc",
			want: "https://db.example.com/dbname?jwt=abc",
		},
		{
			name: "unparseable remote URL replaced wholesale",
			dsn:  "libsql://%",
			want: "libsql://[redacted]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := redactDSN(tt.dsn); got != tt.want {
				t.Errorf("redactDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestRedactDSN_NeverLeaksSecrets(t *testing.T) {
	t.Parallel()

	secret := "super-secret-token-value"

	for _, dsn := range []string{
		"libsql://" + secret + "@my-db.turso.io",
		"libsql://my-db.turso.io?authToken=" + secret,
		"libsql://my-db.turso.io?token=" + secret,
		"libsql://my-db.turso.io?apikey=" + secret,
	} {
		if got := redactDSN(dsn); strings.Contains(got, secret) {
			t.Errorf("redactDSN(%q) leaked secret: %q", dsn, got)
		}
	}
}
