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
			name: "encryption key query parameter redacted",
			dsn:  "/data/secret.db?experimental=encryption&encryption_cipher=aes256gcm&encryption_hexkey=aabbccdd",
			want: "/data/secret.db?encryption_cipher=aes256gcm&encryption_hexkey=%5Bredacted%5D&experimental=encryption",
		},
		{
			name: "memory DSN with encryption key drops query on parse failure",
			dsn:  ":memory:?encryption_hexkey=deadbeef",
			want: ":memory:",
		},
		{
			name: "unparseable local path with key drops query",
			dsn:  "/data/%zz.db?encryption_hexkey=deadbeef",
			want: "/data/%zz.db",
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
		"/data/secret.db?experimental=encryption&encryption_cipher=aes256gcm&encryption_hexkey=" + secret,
		":memory:?encryption_hexkey=" + secret,
		// Adversarial shapes (2026-09-08 audit): case variants, key-ish
		// substrings, userinfo+query combos, other remote schemes, and
		// URL-encoded secret payloads.
		"libsql://my-db.turso.io?AUTH_TOKEN=" + secret,
		"libsql://my-db.turso.io?authToken=" + secret + "&token=" + secret,
		"libsql://" + secret + "@my-db.turso.io?authToken=" + secret,
		"https://my-db.turso.io?authToken=" + secret,
		"http://my-db.turso.io?authToken=" + secret,
		"libsql://my-db.turso.io?hexkey=" + secret,
		"libsql://my-db.turso.io?myAPIKEY=" + secret,
		":memory:?encryption_hexkey=" + secret + "&experimental=encryption",
		"/data/secret.db?encryption_hexkey=" + secret,
	} {
		if got := redactDSN(dsn); strings.Contains(got, secret) {
			t.Errorf("redactDSN(%q) leaked secret: %q", dsn, got)
		}
	}
}

// TestRedactDSN_PreservesNonSecrets guards the other direction: redaction
// must keep operator-relevant non-credential context (host, file path,
// non-secret query params). A redactor that blanked the whole DSN would pass
// NeverLeaks vacuously.
func TestRedactDSN_PreservesNonSecrets(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct{ dsn, wantFragment string }{
		{"libsql://my-db.turso.io?authToken=abc", "my-db.turso.io"},
		{"/data/app.db?experimental=encryption&encryption_hexkey=cafe", "/data/app.db"},
		{"/data/app.db?experimental=encryption&encryption_hexkey=cafe", "experimental=encryption"},
	} {
		if got := redactDSN(tt.dsn); !strings.Contains(got, tt.wantFragment) {
			t.Errorf("redactDSN(%q) = %q, lost non-secret fragment %q", tt.dsn, got, tt.wantFragment)
		}
	}
}

// TestRedactDSN_MalformedRemoteNeverPanics: unparseable remote DSNs collapse
// to the fixed placeholder instead of panicking or echoing raw input.
func TestRedactDSN_MalformedRemoteNeverPanics(t *testing.T) {
	t.Parallel()

	secret := "ht!tp://[::1]:namedport%%"
	for _, dsn := range []string{
		"libsql://[invalid host with spaces?authToken=" + secret,
		"https://" + secret,
	} {
		got := redactDSN(dsn)
		if strings.Contains(got, secret) && got != "libsql://[redacted]" {
			t.Errorf("redactDSN(%q) echoed raw input: %q", dsn, got)
		}
	}
}
