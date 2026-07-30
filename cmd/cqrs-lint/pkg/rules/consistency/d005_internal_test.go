package consistency

import "testing"

func TestParseVersionParts_TrailingPunctuation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  []string
	}{
		{"v4.2.0.", []string{"4", "2", "0"}},
		{"v4.2.0,", []string{"4", "2", "0"}},
		{"v4.2.0", []string{"4", "2", "0"}},
		{"v4.0.x", []string{"4", "0", "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got := parseVersionParts(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseVersionParts(%q) = %v, want %v", tc.input, got, tc.want)
			}

			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseVersionParts(%q)[%d] = %q, want %q",
						tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestIsVersionCompatible_TrailingDot(t *testing.T) {
	t.Parallel()

	if !isVersionCompatible("v4.2.0.", "v4.2.0") {
		t.Error("isVersionCompatible should treat trailing-dot version as compatible")
	}

	if !isVersionCompatible("v4.2.0", "v4.2.0") {
		t.Error("isVersionCompatible should match identical versions")
	}

	if isVersionCompatible("v4.2.0", "v4.3.0") {
		t.Error("isVersionCompatible should reject different versions")
	}
}
