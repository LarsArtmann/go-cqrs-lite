package tursoengine

import (
	"errors"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestStrictDSNParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dsn     string
		wantErr bool
		wantMsg string
	}{
		{
			name: "no query passes",
			dsn:  "/data/app.db",
		},
		{
			name: "memory without params passes",
			dsn:  ":memory:",
		},
		{
			name: "known experimental token passes",
			dsn:  "/data/app.db?experimental=encryption,views",
		},
		{
			name: "non-encryption unknown params stay lenient",
			dsn:  "file:/data/app.db?mode=memory",
		},
		{
			name: "authtoken is not a key param",
			dsn:  ":memory:?authtoken=x",
		},
		{
			name: "hand-written known encryption params pass",
			dsn:  "/data/app.db?encryption_cipher=aegis256&encryption_hexkey=ab",
		},
		{
			name: "known encryption params case-insensitive pass",
			dsn:  "/data/app.db?Encryption_HexKey=ab",
		},
		{
			name:    "mistyped hexkey rejected with suggestion",
			dsn:     "/data/app.db?encryption_hexkkey=ab",
			wantErr: true,
			wantMsg: `did you mean "encryption_hexkey"`,
		},
		{
			name:    "mistyped cipher rejected with suggestion",
			dsn:     "/data/app.db?encryption_chiper=aegis256",
			wantErr: true,
			wantMsg: `did you mean "encryption_cipher"`,
		},
		{
			name:    "bare encrypt param rejected",
			dsn:     "/data/app.db?encrypt=true",
			wantErr: true,
			wantMsg: `parameter "encrypt"`,
		},
		{
			name:    "bare key param rejected",
			dsn:     "/data/app.db?key=ab",
			wantErr: true,
			wantMsg: `parameter "key"`,
		},
		{
			name:    "remote DSN with unknown key param rejected too",
			dsn:     "libsql://db.example.turso.io?encryption_hexkkey=ab",
			wantErr: true,
			wantMsg: `did you mean "encryption_hexkey"`,
		},
		{
			name: "remote DSN authToken passes",
			dsn:  "libsql://db.example.turso.io?authToken=secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := strictDSNParams(tt.dsn)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("strictDSNParams(%q) = %v, want nil", tt.dsn, err)
				}

				return
			}

			if err == nil {
				t.Fatalf("strictDSNParams(%q) = nil, want rejection", tt.dsn)
			}

			if !errors.Is(err, ErrUnknownEncryptionParam) {
				t.Fatalf("strictDSNParams(%q) error does not match ErrUnknownEncryptionParam: %v", tt.dsn, err)
			}

			if got := errorfamily.Classify(err); got != errorfamily.Rejection {
				t.Fatalf("strictDSNParams(%q) family = %v, want Rejection", tt.dsn, got)
			}

			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("strictDSNParams(%q) error = %q, want it to contain %q", tt.dsn, err, tt.wantMsg)
			}
		})
	}
}

func TestNewRejectsMistypedEncryptionParam(t *testing.T) {
	t.Parallel()

	_, err := New(":memory:?encryption_hexkkey=" + strings.Repeat("ab", 32))
	if err == nil {
		t.Fatal("New must reject a mistyped encryption parameter, got nil error")
	}

	if !errors.Is(err, ErrUnknownEncryptionParam) {
		t.Fatalf("New error does not match ErrUnknownEncryptionParam: %v", err)
	}

	if got := errorfamily.Classify(err); got != errorfamily.Rejection {
		t.Fatalf("New error family = %v, want Rejection", got)
	}

	if !strings.Contains(err.Error(), "WithEncryption") {
		t.Fatalf("New error should point at WithEncryption, got: %v", err)
	}
}

func TestSuggestDSNParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "double-k typo suggests hexkey",
			in:   "encryption_hexkkey",
			want: []string{"encryption_hexkey"},
		},
		{
			name: "missing-ion cipher typo suggests cipher",
			in:   "encrypt_cipher",
			want: []string{"encryption_cipher"},
		},
		{
			name: "hexkey prefix suggests hexkey",
			in:   "encryption_hex",
			want: []string{"encryption_hexkey"},
		},
		{
			name: "unrelated key name suggests nothing",
			in:   "apikey",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := suggestDSNParams(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("suggestDSNParams(%q) = %v, want %v", tt.in, got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("suggestDSNParams(%q) = %v, want %v", tt.in, got, tt.want)
				}
			}
		})
	}
}

func TestEditDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"hexkkey", "hexkey", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
	}

	for _, tt := range tests {
		if got := editDistance(tt.a, tt.b); got != tt.want {
			t.Fatalf("editDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
