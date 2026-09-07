package tursoengine

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestApplyEncryption_BuildsDSN(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("ab", 32) // 32 bytes as hex

	dsn, err := applyEncryption(":memory:", &encryptionConfig{cipher: CipherAEGIS256, hexKey: key})
	if err != nil {
		t.Fatalf("applyEncryption: %v", err)
	}

	for _, want := range []string{
		":memory:?experimental=encryption",
		"encryption_cipher=aegis256",
		"encryption_hexkey=" + key,
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN %q missing %q", dsn, want)
		}
	}
}

func TestApplyEncryption_CoexistsWithViewsFlag(t *testing.T) {
	t.Parallel()

	key := strings.Repeat("ab", 16)

	dsn, err := applyEncryption("/data/app.db?experimental=views",
		&encryptionConfig{cipher: CipherAES128GCM, hexKey: key})
	if err != nil {
		t.Fatalf("applyEncryption: %v", err)
	}

	mergedViews := strings.Contains(dsn, "experimental=views%2Cencryption") ||
		strings.Contains(dsn, "experimental=views,encryption")
	if !mergedViews {
		t.Errorf("DSN %q did not merge views+encryption flags", dsn)
	}
}

func TestApplyEncryption_Rejections(t *testing.T) {
	t.Parallel()

	valid32 := strings.Repeat("ab", 32)

	tests := []struct {
		name        string
		dsn         string
		cipher      Cipher
		hexKey      string
		wantErrText string
	}{
		{
			name:        "remote DSN rejected",
			dsn:         "libsql://my-db.turso.io",
			cipher:      CipherAEGIS256,
			hexKey:      valid32,
			wantErrText: "embedded databases only",
		},
		{
			name:        "unknown cipher lists valid options",
			dsn:         ":memory:",
			cipher:      Cipher("chacha20poly1305"),
			hexKey:      valid32,
			wantErrText: "unknown encryption cipher",
		},
		{
			name:        "non-hex key rejected without leaking it",
			dsn:         ":memory:",
			cipher:      CipherAEGIS256,
			hexKey:      "not-hex-at-all",
			wantErrText: "hex-encoded",
		},
		{
			name:        "wrong key length rejected",
			dsn:         ":memory:",
			cipher:      CipherAEGIS256,
			hexKey:      strings.Repeat("ab", 16),
			wantErrText: "want 32 for aegis256",
		},
		{
			name:        "DSN already carrying hexkey conflicts",
			dsn:         ":memory:?encryption_hexkey=ff",
			cipher:      CipherAEGIS256,
			hexKey:      valid32,
			wantErrText: "exactly one key source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := applyEncryption(tt.dsn, &encryptionConfig{cipher: tt.cipher, hexKey: tt.hexKey})
			if err == nil {
				t.Fatalf("applyEncryption(%q) succeeded, want error containing %q", tt.dsn, tt.wantErrText)
			}

			if !strings.Contains(err.Error(), tt.wantErrText) {
				t.Errorf("error %q missing %q", err.Error(), tt.wantErrText)
			}
		})
	}
}

func TestApplyEncryption_NeverQuotesKeyMaterial(t *testing.T) {
	t.Parallel()

	secret := strings.Repeat("de", 32)

	_, err := applyEncryption(":memory:", &encryptionConfig{cipher: Cipher("bogus"), hexKey: secret})
	if err == nil {
		t.Fatal("expected error for bogus cipher")
	}

	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaked key material: %q", err.Error())
	}

	badLenKey := hex.EncodeToString([]byte("short"))
	_, err = applyEncryption(":memory:", &encryptionConfig{cipher: CipherAEGIS256, hexKey: badLenKey})
	if err == nil {
		t.Fatal("expected error for short key")
	}

	if strings.Contains(err.Error(), badLenKey) {
		t.Errorf("error leaked key material: %q", err.Error())
	}
}
