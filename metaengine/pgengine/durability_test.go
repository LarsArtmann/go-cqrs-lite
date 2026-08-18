package pgengine

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own durability translation table test
func TestDurabilityDSN(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tier metaengine.DurabilityTier
		dsn  string
		want string
	}{
		{
			tier: "",
			dsn:  "postgres://h:5432/db?sslmode=disable",
			want: "postgres://h:5432/db?sslmode=disable",
		},
		{
			tier: metaengine.DurabilityStrict,
			dsn:  "postgres://h:5432/db",
			want: "postgres://h:5432/db?synchronous_commit=on",
		},
		{
			tier: metaengine.DurabilityNormal,
			dsn:  "postgres://h:5432/db?sslmode=disable",
			want: "postgres://h:5432/db?sslmode=disable&synchronous_commit=off",
		},
		{
			tier: metaengine.DurabilityRelaxed,
			dsn:  "host=localhost user=app",
			want: "host=localhost user=app synchronous_commit=off",
		},
		{
			tier: "",
			dsn:  "host=localhost user=app",
			want: "host=localhost user=app",
		},
	} {
		got, err := durabilityDSN(tc.tier, tc.dsn)
		if err != nil {
			t.Fatalf("durabilityDSN(%q, %q): %v", tc.tier, tc.dsn, err)
		}

		if got != tc.want {
			t.Fatalf("durabilityDSN(%q, %q) = %q, want %q", tc.tier, tc.dsn, got, tc.want)
		}
	}
}

func TestDurabilityDSN_ConflictingSetting(t *testing.T) {
	t.Parallel()

	_, err := durabilityDSN(metaengine.DurabilityStrict, "postgres://h/db?synchronous_commit=off")
	if err == nil {
		t.Fatal("explicit synchronous_commit + strict tier must conflict")
	}

	_, err = durabilityDSN("", "postgres://h/db?synchronous_commit=off")
	if err != nil {
		t.Fatalf("explicit setting without a tier must pass, got %v", err)
	}
}

func TestDurabilityDSN_InvalidTier(t *testing.T) {
	t.Parallel()

	_, err := durabilityDSN("bogus", "postgres://h/db")
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("error = %v, want ErrUnsupportedDurability", err)
	}
}
