package event

import (
	"encoding/base64"
	"testing"
)

func TestDecodeBase64String(t *testing.T) {
	t.Parallel()

	raw := []byte("hello world")

	urlSafe := base64.URLEncoding.EncodeToString(raw)
	standard := base64.StdEncoding.EncodeToString(raw)

	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"url-safe encoding", urlSafe, raw, false},
		{"standard encoding", standard, raw, false},
		{"empty string", "", []byte{}, false},
		{"invalid base64", "!!!not-base64!!!", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecodeBase64String(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeBase64String(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if !tt.wantErr && string(got) != string(tt.want) {
				t.Errorf("DecodeBase64String(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
