package metaengine

import (
	"encoding/base64"
	"testing"
)

// TestCursorWireFormatGolden pins the exact wire bytes of cursor payloads.
// The compound keyset cursor is a compatibility surface: cursors minted by one
// process must be consumed by another (possibly on an older patch), so the
// base64 strings below are frozen contract — change them and every issued
// cursor in the wild stops parsing. The JSON shape is {"Sort":<scalar>,
// "Key":"<base64 bytes>"} under RawURLEncoding; legacy value cursors keep
// their raw-JSON form.
func TestCursorWireFormatGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "compound integral sort",
			value: SortKeyCursor{Sort: float64(0), Key: []byte("item-012")},
			want:  "eyJTb3J0IjowLCJLZXkiOiJhWFJsYlMwd01UST0ifQ",
		},
		{
			name:  "compound int sort",
			value: SortKeyCursor{Sort: int64(7), Key: []byte("k-1")},
			want:  "eyJTb3J0Ijo3LCJLZXkiOiJheTB4In0",
		},
		{
			name:  "compound fractional sort",
			value: SortKeyCursor{Sort: 2.5, Key: []byte("zz")},
			want:  "eyJTb3J0IjoyLjUsIktleSI6ImVubz0ifQ",
		},
		{
			name:  "legacy scalar value cursor",
			value: "after-x",
			want:  "ImFmdGVyLXgi",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := Cursor{Value: tc.value}.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			if got != tc.want {
				t.Fatalf(
					"wire bytes changed:\n got %q\nwant %q\nfrozen cursor contract — see test doc",
					got,
					tc.want,
				)
			}

			parsed, err := ParseCursor(got)
			if err != nil {
				t.Fatalf("ParseCursor: %v", err)
			}

			if skc, ok := parsed.Value.(SortKeyCursor); ok {
				if base64.StdEncoding.EncodeToString(skc.Key) !=
					base64.StdEncoding.EncodeToString(tc.value.(SortKeyCursor).Key) {
					t.Fatalf("round-trip key = %q, want %q", skc.Key, tc.value.(SortKeyCursor).Key)
				}
			}
		})
	}
}
